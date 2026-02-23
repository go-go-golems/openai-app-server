---
Title: Architecture and Internals Guide
Ticket: OAS-01-INTERNALS
Status: active
Topics:
    - architecture
    - documentation
    - onboarding
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - openai-app-server/cmd/openai-app-server/main.go
    - openai-app-server/cmd/openai-app-server/root.go
    - openai-app-server/cmd/openai-app-server/harness_run_command.go
    - openai-app-server/pkg/codexrpc/client.go
    - openai-app-server/pkg/codexrpc/protocol.go
    - openai-app-server/pkg/js/runtime.go
    - openai-app-server/pkg/js/module_codex.go
    - openai-app-server/pkg/state/store.go
    - openai-app-server/pkg/state/projector.go
    - openai-app-server/pkg/harness/dispatch.go
    - openai-app-server/pkg/harness/builtin/autopilot.go
ExternalSources: []
Summary: "Comprehensive architecture, implementation, and internals guide for the openai-app-server JS infrastructure. Covers the Go workspace, JSON-RPC client, JavaScript runtime, harness system, state management, and all supporting packages."
LastUpdated: 2026-02-23T10:02:36.044518964-05:00
WhatFor: "Onboarding new developers to the openai-app-server codebase"
WhenToUse: "When starting work on the project, debugging issues, or understanding how the pieces fit together"
---

# OpenAI App Server JS - Architecture & Internals Guide

> **Audience**: New developers/interns joining the project.
> **Scope**: Everything you need to understand how this system works, from 10,000 feet down to the function level.

---

## Table of Contents

1. [What Is This Project?](#1-what-is-this-project)
2. [The Big Picture: Monorepo Layout](#2-the-big-picture-monorepo-layout)
3. [openai-app-server Module Overview](#3-openai-app-server-module-overview)
4. [The JSON-RPC 2.0 Protocol Layer (codexrpc)](#4-the-json-rpc-20-protocol-layer-codexrpc)
5. [The JavaScript Runtime (pkg/js)](#5-the-javascript-runtime-pkgjs)
6. [JS Modules: The Developer API](#6-js-modules-the-developer-api)
7. [The Harness System](#7-the-harness-system)
8. [State Management](#8-state-management)
9. [Configuration](#9-configuration)
10. [CLI Commands](#10-cli-commands)
11. [Data Flow: End-to-End Walkthrough](#11-data-flow-end-to-end-walkthrough)
12. [Design Patterns & Conventions](#12-design-patterns--conventions)
13. [Testing Strategy](#13-testing-strategy)
14. [Key Dependencies](#14-key-dependencies)
15. [Quick Reference: File Map](#15-quick-reference-file-map)

---

## 1. What Is This Project?

The **openai-app-server** is a Go application that:

1. **Spawns** a Codex App Server process (like `codex-app-server`) via stdio
2. **Communicates** with it using JSON-RPC 2.0 over that stdio pipe
3. **Runs JavaScript harness scripts** that orchestrate the conversation - starting threads, sending turns, handling approval requests, waiting for events
4. **Manages state** by projecting incoming RPC events into an in-memory store

Think of it as a **programmable automation layer** for Codex. Instead of manually chatting with the AI, you write JavaScript scripts that drive the interaction programmatically.

### Why JavaScript?

The JavaScript runtime (powered by [Goja](https://github.com/dop251/goja), a pure-Go ECMAScript engine) gives harness authors a familiar, expressive language to script complex workflows - multi-thread orchestration, approval policies, test-driven development loops, scientific research pipelines - without recompiling Go.

### The Core Mental Model

```
                                    JSON-RPC 2.0 (stdio)
  ┌──────────────────┐         ┌────────────────────────────┐
  │                  │  send   │                            │
  │  JS Harness      │────────>│   Codex App Server         │
  │  Script          │         │   (external process)       │
  │                  │<────────│                            │
  │  (runs in Goja)  │  recv   │                            │
  └──────────────────┘         └────────────────────────────┘
         │                                   │
         │ require('codex')                  │
         │ require('rpc')                    │
         │ require('approval')               │
         │ require('ui')                     │
         │ require('clock')                  │
         ▼                                   │
  ┌──────────────────┐              Events stream back:
  │  Go Runtime      │              - thread/started
  │  (pkg/js)        │              - turn/completed
  │                  │              - item/commandExecution/requestApproval
  │  Bridges RPC     │              - turn/diff/updated
  │  to/from JS      │              - ...
  └──────────────────┘
```

---

## 2. The Big Picture: Monorepo Layout

This project lives in a **Go workspace** (`go.work`) with 4 modules:

```
app-server-js/
├── go.work                    # Go workspace (Go 1.25.7)
├── AGENT.md                   # Development guidelines
│
├── geppetto/                  # LLM inference framework (the "engine")
│   └── pkg/inference/         # Core inference engine, middleware, tools
│
├── go-go-goja/                # JavaScript runtime sandbox library
│   ├── engine/                # Goja runtime factory
│   └── modules/               # Native Go modules for JS (fs, db, exec, timer)
│
├── pinocchio/                 # CLI LLM tool + Web chat application
│   ├── cmd/pinocchio/         # YAML prompt runner CLI
│   └── cmd/web-chat/          # WebSocket-based chat UI
│
└── openai-app-server/         # <-- THIS IS OUR FOCUS
    ├── cmd/openai-app-server/ # CLI entry point
    └── pkg/                   # Core library packages
```

### Module Relationships

```
openai-app-server
    depends on ──> go-go-goja (runtime machinery)
    depends on ──> glazed (CLI framework)

geppetto
    depends on ──> go-go-goja (JS integration)
    provides  ──> inference engine used by pinocchio

pinocchio
    depends on ──> geppetto (LLM framework)
    depends on ──> go-go-goja (JS runtime)
```

The `openai-app-server` is relatively self-contained. It uses `go-go-goja` for the runtime plumbing (event loop, thread-safe JS execution) and `glazed` for CLI scaffolding. It does NOT depend on geppetto or pinocchio.

---

## 3. openai-app-server Module Overview

```
openai-app-server/
├── cmd/openai-app-server/
│   ├── main.go                         # Entry point
│   ├── root.go                         # Cobra root + command tree
│   ├── args.go                         # CSV arg parsing utility
│   ├── harness_run_command.go          # `harness run` command
│   ├── harness_state_replay_command.go # `harness state-replay` command
│   ├── thread_list_command.go          # `thread list` command
│   └── thread_read_command.go          # `thread read` command
│
├── pkg/
│   ├── codexrpc/               # JSON-RPC 2.0 client + transport
│   │   ├── protocol.go         # Message types, constructors
│   │   ├── errors.go           # Error definitions
│   │   ├── transport.go        # Transport interface
│   │   ├── transport_stdio.go  # Stdio process transport
│   │   ├── memory_transport.go # In-memory test transport
│   │   ├── client.go           # Client with handshake FSM
│   │   ├── threads.go          # Thread list/read operations
│   │   ├── initialize.go       # Handshake parameter builder
│   │   └── retry.go            # Exponential backoff retry
│   │
│   ├── js/                     # JavaScript runtime + native modules
│   │   ├── runtime.go          # Core runtime (793 lines, the heart)
│   │   ├── module_codex.go     # require('codex') - main API
│   │   ├── module_rpc.go       # require('rpc') - low-level RPC
│   │   ├── module_approval.go  # require('approval') - approval helpers
│   │   ├── module_ui.go        # require('ui') - UI events
│   │   └── module_clock.go     # require('clock') - time utilities
│   │
│   ├── state/                  # Event-sourced state management
│   │   ├── models.go           # Data models (Thread/Turn/ItemState)
│   │   ├── store.go            # Thread-safe in-memory store
│   │   └── projector.go        # Event -> state projection
│   │
│   ├── harness/                # Harness composition framework
│   │   ├── context.go          # Request context + response tracking
│   │   ├── dispatch.go         # Event dispatcher
│   │   ├── compose.go          # Harness composition
│   │   └── builtin/            # Built-in harness modules
│   │       ├── autopilot.go    # Auto-approval for safe commands
│   │       ├── plan_gate.go    # Plan review + approval gate
│   │       ├── review_gate.go  # Code review gate
│   │       ├── auto_compact.go # Token usage compaction trigger
│   │       └── tdd_loop.go     # Test-driven dev loop
│   │
│   └── config/                 # Configuration defaults
│       └── defaults.go         # Transport, model, policy defaults
│
└── ttmp/                       # Documentation + experiment scripts
```

---

## 4. The JSON-RPC 2.0 Protocol Layer (codexrpc)

This is the communication backbone. Everything between our Go process and the Codex App Server flows through JSON-RPC 2.0 messages.

### 4.1 Message Format

Every message is a JSON object on a single line (newline-delimited):

```go
// pkg/codexrpc/protocol.go
type Message struct {
    ID     any              `json:"id,omitempty"`
    Method string           `json:"method,omitempty"`
    Params json.RawMessage  `json:"params,omitempty"`
    Result json.RawMessage  `json:"result,omitempty"`
    Error  *RPCError        `json:"error,omitempty"`
}
```

Three message types:

| Type | Has ID? | Has Method? | Example |
|------|---------|-------------|---------|
| **Request** | Yes | Yes | `{"id":1, "method":"thread/start", "params":{...}}` |
| **Notification** | No | Yes | `{"method":"turn/completed", "params":{...}}` |
| **Response** | Yes | No | `{"id":1, "result":{...}}` or `{"id":1, "error":{...}}` |

### 4.2 The Handshake State Machine

Before any real work happens, the client must complete a two-step handshake:

```
State: NEW
   │
   │  send Request("initialize", params)
   │  recv Response with result
   │
   ▼
State: INITIALIZED
   │
   │  send Notification("initialized", {})
   │
   ▼
State: READY
   │
   │  Now all methods are allowed
   │  ...
   │
   ▼
State: CLOSED (after Close())
```

This is enforced by `validateRequestMethod()` and `validateNotifyMethod()`:
- In `stateNew`, only `"initialize"` requests are allowed
- In `stateInitialized`, only `"initialized"` notifications are allowed
- In `stateReady`, everything is allowed

**Code**: `pkg/codexrpc/client.go:335-365`

### 4.3 The Client Read Loop

Once started (lazily via `ensureReadLoop()`), a background goroutine continuously reads messages from the transport:

```go
func (c *Client) readLoop() {
    for {
        msg, err := c.transport.Recv(ctx)
        // ...
        switch {
        case msg.IsResponse():
            c.dispatchResponse(msg)      // Match to pending Request
        case msg.IsNotification():
            c.dispatchNotification(msg)  // Fan out to handlers
        case msg.IsRequest():
            c.dispatchRequest(msg)       // Fan out to handlers
        }
    }
}
```

**Response dispatching** uses a `pending` map of `string -> chan *Message`. When you call `client.Request()`, it:
1. Generates a unique ID via `atomic.Int64`
2. Creates a buffered channel and stores it in `pending[idKey]`
3. Sends the request message
4. Blocks on the channel until the response arrives (or context cancels)

**Notification/Request dispatching** fans out to all registered handlers for that method, plus any wildcard (`"*"`) handlers.

### 4.4 Transport Abstraction

```go
// pkg/codexrpc/transport.go
type Transport interface {
    Send(ctx context.Context, msg *Message) error
    Recv(ctx context.Context) (*Message, error)
    Close() error
}
```

Two implementations:

**StdioTransport** (`transport_stdio.go`): Spawns a subprocess, pipes JSON-RPC messages over stdin/stdout. Each message is a single JSON line terminated by `\n`.

```go
transport, err := codexrpc.NewStdioTransport("codex-app-server", "--flag1", "--flag2")
```

**MemoryTransport** (`memory_transport.go`): For testing. `Push()` injects messages, `SentSnapshot()` retrieves what was sent.

### 4.5 Retry Support

```go
// pkg/codexrpc/retry.go
func RequestWithRetry(ctx, client, method, params, policy) (*Message, error)
```

Retries on `ResponseError` with code `-32001` (ServerOverloaded). Uses exponential backoff: `150ms * 2^attempt`, capped at `2s`. Default: 3 attempts.

### 4.6 Thread Operations

```go
// pkg/codexrpc/threads.go
func (c *Client) ThreadList(ctx, limit) ([]ThreadSummary, error)
func (c *Client) ThreadRead(ctx, threadID, includeTurns) (*Thread, error)
```

Convenience wrappers around `client.Request()` for the `thread/list` and `thread/read` RPC methods.

---

## 5. The JavaScript Runtime (pkg/js)

This is the **heart of the system**. It bridges the Go RPC client with JavaScript harness scripts.

### 5.1 Architecture

```
┌─────────────────────────────────────────────────────┐
│                  pkg/js/Runtime                      │
│                                                     │
│  ┌──────────┐    ┌──────────────┐    ┌───────────┐ │
│  │ goja.VM  │    │  EventLoop   │    │  Runner   │ │
│  │(JS exec) │    │(async/timer) │    │(thread    │ │
│  │          │    │              │    │ safety)   │ │
│  └──────────┘    └──────────────┘    └───────────┘ │
│                                                     │
│  Bridges:                                           │
│  ┌──────────┐    ┌──────────────┐                   │
│  │RPCBridge │    │  UIBridge    │                   │
│  │(Go↔RPC)  │    │(stdout emit) │                   │
│  └──────────┘    └──────────────┘                   │
│                                                     │
│  Handler Registries:                                │
│  - rpcNotificationHandlers []goja.Callable          │
│  - rpcRequestHandlers      []goja.Callable          │
│  - uiEventHandlers         []goja.Callable          │
│                                                     │
│  Event Infrastructure:                              │
│  - eventWaiters   map[int]*runtimeEventWaiter       │
│  - approvalPolicy *approvalPolicyHandlers           │
│  - eventMethodCounts map[string]int                 │
└─────────────────────────────────────────────────────┘
```

### 5.2 Thread Safety Model

JavaScript is single-threaded. Go is multi-threaded. The bridge between them is the **Runner** (from `go-go-goja`):

- **`runner.Call(ctx, name, fn)`**: Executes a function on the JS thread and returns a result. **Blocking**.
- **`runner.Post(ctx, name, fn)`**: Queues a function to run on the JS thread. **Non-blocking** (fire-and-forget).

All JS execution goes through the Runner. Go code NEVER touches the `goja.Runtime` directly from arbitrary goroutines.

### 5.3 Initialization

```go
func NewRuntime(opts Options) (*Runtime, error) {
    loop := eventloop.NewEventLoop()      // Node.js-style event loop
    go loop.Start()                        // Start in background

    vm := goja.New()                       // Create JS VM
    runner := runtimeowner.NewRunner(vm, loop, ...)

    rt := &Runtime{vm, loop, runner, ...}

    // Register native modules
    reg := require.NewRegistry()
    registerCodexModule(reg, rt)           // require('codex')
    registerRPCModule(reg, rt)             // require('rpc')
    registerUIModule(reg, rt)              // require('ui')
    registerClockModule(reg, rt)           // require('clock')
    registerApprovalModule(reg, rt)        // require('approval')

    // Enable require() in the VM
    reg.Enable(vm)
    return rt, nil
}
```

### 5.4 Event Flow: Go -> JavaScript

When the RPC client receives a notification from the Codex server:

```
codex-app-server sends: {"method":"turn/completed","params":{...}}
        │
        ▼
    client.readLoop() -> dispatchNotification()
        │
        ▼
    wildcard handler registered in harness_run_command.go:
    client.OnNotification("*", func(msg) {
        runtime.EmitRPCNotification(msg.Method, msg.Params)
    })
        │
        ▼
    Runtime.EmitRPCNotification() -> runner.Post(func() {
        1. recordEventMetrics()           // track counts
        2. dispatchEventToWaiters()       // resolve any waitFor() promises
        3. call all rpcNotificationHandlers  // invoke JS callbacks
    })
```

The same pattern applies for requests (`EmitRPCRequest`), with the addition of `applyApprovalPolicy()`.

### 5.5 Event Flow: JavaScript -> Go

When JS code calls `session.request("thread/start", params)`:

```
JS: session.request("thread/start", {...})
        │
        ▼
    module_codex.go: rpcRequestPromiseFrom(vm, method, params)
        │
        ▼
    Creates Promise, spawns goroutine:
    go func() {
        result, err := rt.rpc.Request(ctx, method, params)
        rt.runner.Post(func() {
            if err { reject(err) } else { resolve(result) }
        })
    }()
```

The goroutine calls the Go RPC bridge (which calls `client.Request()`), then posts the result back to the JS thread via `runner.Post`.

### 5.6 The waitFor() System

`waitFor()` creates a **Promise** that resolves when a matching event arrives:

```javascript
// Wait for a turn to complete
session.waitFor({
    method: "turn/completed",
    timeoutMs: 60000,
    where: (evt) => evt.params.threadId === myThreadId
}).then(evt => { ... });
```

Implementation:
1. A `runtimeEventWaiter` is created with method filter, timeout, and optional predicate
2. It's stored in `rt.eventWaiters[id]`
3. A `time.AfterFunc` timer is started for timeout rejection
4. On every incoming event, `dispatchEventToWaiters()` checks all waiters
5. If method matches and predicate passes, the promise resolves and the waiter is cleaned up

**Code**: `pkg/js/runtime.go:362-567`

### 5.7 The Approval Policy System

JavaScript harnesses can set a declarative approval policy:

```javascript
session.approvals.setPolicy({
    command: (req) => "acceptForSession",    // auto-approve commands
    fileChange: (req) => "decline",          // deny file changes
    fallback: (req) => "cancel"              // cancel everything else
});
```

When an approval request arrives (`item/commandExecution/requestApproval` or `item/fileChange/requestApproval`), `applyApprovalPolicy()`:
1. Looks up the matching handler (command, fileChange, or fallback)
2. Calls it with request context
3. If the handler returns a string decision, sends the response immediately
4. If the handler returns a **Promise** (thenable), attaches `.then()` to send the response async

**Epoch-based versioning** prevents stale policies: each `setPolicy()` increments an epoch, and `clearApprovalPolicy()` only clears if the epoch matches.

**Code**: `pkg/js/runtime.go:625-792`

---

## 6. JS Modules: The Developer API

### 6.1 `require('codex')` - The Main API

This is what harness scripts use most. `codex.connect()` returns a session object:

```javascript
const codex = require("codex");
const session = codex.connect();
```

**Session object shape**:

```javascript
session = {
    // Low-level RPC
    request(method, params)        // -> Promise
    notify(method, params)         // -> void
    respond(id, result)            // -> void
    respondError(id, code, msg)    // -> void

    // Handler registration (returns unsubscribe function)
    onNotification(callback)       // -> () => void
    onRequest(callback)            // -> () => void
    onUIEvent(callback)            // -> () => void

    // Event waiting
    waitFor(spec)                  // -> Promise

    // Thread management
    threads: {
        start(params)              // -> Promise (start new thread)
        list(params)               // -> Promise (list threads)
        read(threadId, includeTurns) // -> Promise (read thread)
        byId(startResult)          // -> threadId string
    },

    // ID extraction helpers
    ids: {
        thread(startResult)        // -> threadId string
        turn(turnResult)           // -> turnId string
    },

    // Thread handle (sugar for thread-scoped operations)
    thread(threadId) => {
        id: threadId,
        turn: {
            start(params)          // -> Promise
            steer(message)         // -> Promise
            interrupt()            // -> Promise
            waitCompleted(opts)    // -> Promise
        },
        review: {
            start()                // -> Promise
        }
    },

    // Approval management
    approvals: {
        setPolicy(policy)          // -> cleanup function
        respond(request, decision) // -> void
    },

    // Event metrics
    events: {
        metrics() => {
            countByMethod()        // -> {method: count}
            totalNotifications()   // -> number
            totalRequests()        // -> number
            reset()                // -> void
        }
    }
}
```

**Code**: `pkg/js/module_codex.go` (311 lines)

### 6.2 `require('rpc')` - Low-Level RPC

Direct access to the RPC bridge, without the session abstraction:

```javascript
const rpc = require("rpc");
rpc.request("thread/list", { limit: 5 });
rpc.notify("some/method", {});
rpc.onNotification((evt) => { ... });
rpc.onRequest((evt) => { ... });
```

**Code**: `pkg/js/module_rpc.go` (34 lines)

### 6.3 `require('approval')` - Approval Helpers

Convenience functions for responding to approval requests:

```javascript
const approval = require("approval");
approval.accept(requestId);           // approve once
approval.acceptForSession(requestId); // approve for the session
approval.decline(requestId);          // reject
approval.cancel(requestId);           // cancel
approval.acceptWithExecpolicyAmendment(requestId, amendment); // approve with policy change
```

**Code**: `pkg/js/module_approval.go` (82 lines)

### 6.4 `require('ui')` - UI Events

For emitting events that the harness runner can listen for:

```javascript
const ui = require("ui");
ui.emit({ type: "my-test-done", ok: true, data: {...} });
ui.onEvent((evt) => { console.log("UI event:", evt); });
```

UI events are the primary signaling mechanism between JS harness scripts and the Go host process. The `--wait-for-ui-type` flag on the CLI watches for matching events.

**Code**: `pkg/js/module_ui.go` (34 lines)

### 6.5 `require('clock')` - Time Utilities

```javascript
const clock = require("clock");
clock.nowMs();          // -> current timestamp in milliseconds
clock.sleep(5000);      // -> Promise that resolves after 5 seconds
```

`sleep()` uses the event loop's timer system for non-blocking async sleep.

**Code**: `pkg/js/module_clock.go` (41 lines)

---

## 7. The Harness System

The harness system provides a **composable** framework for building reusable notification/request handlers. While the JS runtime is the primary harness authoring method today, the Go-level harness framework exists for built-in behaviors.

### 7.1 Core Concepts

**Harness**: A named bundle of notification and request handlers.

```go
type Harness struct {
    Name          string
    Notifications []NotificationBinding  // {Method, Handler}
    Requests      []RequestBinding       // {Method, Handler}
}
```

**Dispatcher**: Merges multiple harnesses and routes events to handlers.

```go
dispatcher := harness.Compose(autopilotHarness, planGateHarness, tddHarness)
dispatcher.DispatchNotification(method, params)
dispatcher.DispatchRequest(requestCtx)
```

**RequestContext**: Wraps a single inbound request with one-response enforcement:

```go
reqCtx := harness.NewRequestContext(id, method, params, sender)
reqCtx.Respond(result)       // OK - first response
reqCtx.Respond(otherResult)  // ERROR - "already responded"
```

### 7.2 Built-in Harness Modules

#### Autopilot (`builtin/autopilot.go`)

Auto-approves safe commands (test runners, etc.) and applies basic file change policies.

**Safe command prefixes** (approved automatically):
- `go test`, `npm test`, `pytest`, `cargo test`, `make test`, `yarn`, `pnpm`

**File change policy**:
- Denied path prefixes: `/etc/`, `/usr/`, `/var/`, `~/.ssh/`, `~/.gnupg/`
- Max changed lines threshold: 400 (exceeding this = decline)

```go
harness := builtin.NewAutopilot(builtin.AutopilotConfig{
    SafeCommandPrefixes: []string{"go test", "npm test"},
    MaxChangedLines:     400,
})
```

#### Plan Gate (`builtin/plan_gate.go`)

Intercepts `turn/plan/updated` events to approve or reject plans. Can steer the turn with a custom message on approval.

```go
harness := builtin.NewPlanGate(builtin.PlanGateConfig{
    Controller:       controller,
    SteerInstruction: "Plan approved. Continue with implementation.",
    Approve:          func(plan any) bool { return true },
})
```

#### Review Gate (`builtin/review_gate.go`)

After a turn completes, initiates a code review. If review finds issues, starts a followup turn with feedback.

```go
harness := builtin.NewReviewGate(builtin.ReviewGateConfig{
    Controller:       controller,
    FollowupTemplate: "Please address the following review feedback:\n\n%s",
})
```

#### Auto-Compact (`builtin/auto_compact.go`)

Monitors token usage and triggers compaction when nearing the context window limit.

```go
harness := builtin.NewAutoCompact(builtin.AutoCompactConfig{
    SoftLimitRatio:  0.75,  // Trigger at 75% of context window
    ResetBelowRatio: 0.60,  // Reset flag when below 60%
})
```

#### TDD Loop (`builtin/tdd_loop.go`)

After each turn completes, runs tests. If tests fail, starts a followup turn asking the AI to fix them. Limited to N iterations per thread.

```go
harness := builtin.NewTDDLoop(builtin.TDDLoopConfig{
    MaxIterations:    3,
    FollowupTemplate: "Tests failed. Please fix:\n\n%s",
})
```

---

## 8. State Management

The state package implements **event sourcing** - immutable events are projected into a mutable in-memory store.

### 8.1 Data Models

```go
type ThreadState struct {
    ID         string
    Status     string
    Cwd        string
    Model      string
    TokenUsage map[string]any
    Turns      []TurnState
}

type TurnState struct {
    ID         string
    ThreadID   string
    Status     string
    Error      string
    LatestDiff string
    LatestPlan any
    Items      []ItemState
}

type ItemState struct {
    ID       string
    ThreadID string
    TurnID   string
    Type     string
    Status   string
    // + arbitrary key-value data via map[string]any
}
```

### 8.2 The Store

Thread-safe, hierarchical store with **LRU eviction**:

```
Store
 └── threadRecord (map, ordered by access)
      ├── ThreadState
      └── turnRecord (map, ordered by creation)
           ├── TurnState
           └── ItemState (map, ordered by creation)
```

Bounds (configurable):
- Max 100 threads (oldest evicted when exceeded)
- Max 100 turns per thread
- Max 200 items per turn

All reads return **deep-cloned snapshots** (via JSON marshal/unmarshal) to prevent data races.

**Code**: `pkg/state/store.go` (347 lines)

### 8.3 The Projector

The Projector takes RPC events and mutates the store:

```go
projector := state.NewProjector(store)
projector.Apply("thread/started", map[string]any{"thread": {"id": "t1", "status": "active"}})
projector.Apply("turn/started", map[string]any{"turn": {"id": "turn1", "threadId": "t1"}})
projector.Apply("turn/completed", map[string]any{"turn": {"id": "turn1", "threadId": "t1"}})
```

**Event method -> handler mapping**:

| Method Pattern | Handler | What It Does |
|---|---|---|
| `thread/started`, `thread/updated` | `applyThreadEvent` | Upsert thread state |
| `thread/tokenUsage/updated` | `applyTokenUsageEvent` | Update token usage map |
| `turn/started`, `turn/completed`, `turn/updated` | `applyTurnEvent` | Upsert turn, set status |
| `turn/diff/updated` | `applyTurnDiffEvent` | Update latest diff |
| `turn/plan/updated` | `applyTurnPlanEvent` | Update latest plan |
| `item/started`, `item/completed`, `item/updated` | `applyItemEvent` | Upsert item, set status |

**Flexible param extraction**: The projector handles both nested (`{thread: {id: ...}}`) and flat (`{threadId: ...}`) parameter shapes.

**Code**: `pkg/state/projector.go` (156 lines)

---

## 9. Configuration

```go
// pkg/config/defaults.go
type Defaults struct {
    Transport                  string   // "stdio"
    StdioCommand               string   // "codex-app-server"
    StdioArgs                  []string
    OptOutNotificationMethods  []string // from OAP_OPT_OUT_NOTIFICATION_METHODS env
    Model                      string   // "gpt-5"
    Cwd                        string   // os.Getwd()
    ApprovalPolicy             string   // "on-request"
    SandboxPolicy              string   // "workspace-write"
}
```

Environment variable: `OAP_OPT_OUT_NOTIFICATION_METHODS` (comma-separated list of notification methods to opt out of during initialize).

---

## 10. CLI Commands

The binary uses [Cobra](https://github.com/spf13/cobra) + [Glazed](https://github.com/go-go-golems/glazed) for CLI scaffolding.

### Command Tree

```
openai-app-server
├── harness
│   ├── run          # Execute a JS harness script
│   └── state-replay # Replay events through the state projector
└── thread
    ├── list         # List threads from the server
    └── read         # Read a specific thread
```

### `harness run`

The most important command. Executes a JavaScript harness script:

```bash
go run ./cmd/openai-app-server harness run \
  --script ./path/to/harness.js \
  --stdio-command codex-app-server \
  --model gpt-5 \
  --timeout-ms 60000 \
  --wait-for-ui-type "my-test-done" \
  --fail-on-wait-ui-ok-false
```

**Execution flow**:
1. Parse settings, apply defaults
2. If `--dry-run`, print settings and exit
3. Create `context.WithTimeout` from `--timeout-ms`
4. Spawn codex-app-server via StdioTransport
5. Create codexrpc.Client, perform handshake (`Connect()`)
6. Create JS Runtime with RPC and UI bridges
7. Register wildcard notification/request handlers to pipe events into JS
8. Read and execute the harness script via `runtime.RunString()`
9. If `--wait-for-ui-type` set, block until matching UI event (or timeout)
10. Optional settle sleep for async callbacks
11. Cleanup: close runtime, close client

**Code**: `cmd/openai-app-server/harness_run_command.go` (294 lines)

### `harness state-replay`

Replays a JSON events file through the state projector. Useful for debugging state issues offline:

```bash
go run ./cmd/openai-app-server harness state-replay \
  --events-file ./events.json \
  --thread-id "t123"
```

### `thread list` / `thread read`

Direct RPC operations against the server:

```bash
go run ./cmd/openai-app-server thread list --limit 10
go run ./cmd/openai-app-server thread read --thread-id "t123" --include-turns
```

---

## 11. Data Flow: End-to-End Walkthrough

Let's trace what happens when you run a simple harness script:

```javascript
// harness.js
const codex = require("codex");
const ui = require("ui");
const session = codex.connect();

session.threads.start({ model: "gpt-5" })
  .then(result => {
    const threadId = session.ids.thread(result);
    const handle = session.thread(threadId);
    return handle.turn.start({
      input: [{ type: "text", text: "Hello, world!" }]
    });
  })
  .then(() => ui.emit({ type: "done", ok: true }))
  .catch(err => ui.emit({ type: "done", ok: false, error: String(err) }));
```

**Step-by-step**:

```
1. CLI: `harness run --script harness.js --wait-for-ui-type done`
   ├─ Parse flags, create context with timeout
   │
2. CLI: NewStdioTransport("codex-app-server")
   ├─ exec.Command spawns codex-app-server
   ├─ Pipes stdin/stdout connected
   │
3. CLI: client.Connect(ctx, initializeParams)
   ├─ client.Request("initialize", {...})
   │   ├─ Sends: {"id":1,"method":"initialize","params":{...}}
   │   ├─ readLoop starts (once)
   │   ├─ Blocks until response received
   │   └─ State: NEW -> INITIALIZED
   ├─ client.Notify("initialized", {})
   │   ├─ Sends: {"method":"initialized","params":{}}
   │   └─ State: INITIALIZED -> READY
   │
4. CLI: Create JS Runtime
   ├─ Start event loop
   ├─ Create goja.Runtime + Runner
   ├─ Register modules (codex, rpc, ui, approval, clock)
   │
5. CLI: Register wildcard handlers
   ├─ client.OnNotification("*", -> runtime.EmitRPCNotification)
   ├─ client.OnRequest("*", -> runtime.EmitRPCRequest)
   │
6. CLI: runtime.RunString(scriptContent)
   ├─ JS executes synchronously:
   │   ├─ require("codex") -> returns codex module
   │   ├─ require("ui") -> returns ui module
   │   ├─ codex.connect() -> returns session object
   │   ├─ session.threads.start({model:"gpt-5"})
   │   │   └─ Creates Promise, spawns goroutine:
   │   │       go func() {
   │   │           result := client.Request("thread/start", params)
   │   │           runner.Post(-> resolve(result))
   │   │       }
   │   └─ .then() chain registered (but not yet executed)
   │
7. ASYNC: goroutine sends thread/start request
   ├─ client sends: {"id":2,"method":"thread/start","params":{...}}
   ├─ readLoop receives response: {"id":2,"result":{"thread":{"id":"t1"}}}
   ├─ dispatchResponse: delivers to pending channel
   ├─ goroutine receives response, posts resolve to JS thread
   │
8. JS EVENT LOOP: .then() executes
   ├─ session.ids.thread(result) -> "t1"
   ├─ session.thread("t1") -> thread handle
   ├─ handle.turn.start({input:[...]})
   │   └─ Another promise + goroutine for "turn/start"
   │
9. MEANWHILE: Notifications stream in from codex-app-server
   ├─ {"method":"thread/started","params":{"thread":{"id":"t1"}}}
   │   └─ client.dispatchNotification -> wildcard -> runtime.EmitRPCNotification
   │       └─ runner.Post: recordMetrics, dispatchToWaiters, call JS handlers
   ├─ {"method":"turn/started","params":{"turn":{"id":"turn1"}}}
   ├─ {"method":"item/started","params":{"item":{"id":"item1"}}}
   ├─ ... (AI thinking, tool calls, etc.)
   ├─ {"method":"turn/completed","params":{"turn":{"id":"turn1"}}}
   │
10. ASYNC: turn/start response received
    ├─ .then() -> ui.emit({type: "done", ok: true})
    │   └─ stdoutUIBridge.Emit: prints "ui.emit {...}"
    │   └─ onEmit callback: matches "done" type, sends to waitEventCh
    │
11. CLI: waitEventCh receives the event
    ├─ Checks ok:true (--fail-on-wait-ui-ok-false)
    ├─ Prints "wait-for-ui-type matched type=done"
    │
12. CLI: settle sleep (250ms default)
    │
13. CLI: cleanup
    ├─ runtime.Close() -> shutdown event loop + runner
    ├─ client.Close() -> setState(closed), close pending, close transport
    └─ transport.Close() -> kill codex-app-server process
```

---

## 12. Design Patterns & Conventions

### Pattern 1: Event Sourcing
State is never modified directly. Events flow through the Projector, which applies them to the Store. This makes state reproducible (see `state-replay` command).

### Pattern 2: Promise Bridging (Go <-> JS)
All async Go operations are wrapped in Promises:
```go
promise, resolve, reject := vm.NewPromise()
go func() {
    result, err := goOperation()
    runner.Post(func() {
        if err { reject(err) } else { resolve(result) }
    })
}()
return vm.ToValue(promise)
```

### Pattern 3: Handler + Unsubscribe
Every `On*()` method returns an unsubscribe function:
```go
unsub := client.OnNotification("turn/completed", handler)
defer unsub()  // cleanup when done
```

### Pattern 4: One-Response Enforcement
`RequestContext` ensures exactly one response per request, preventing double-respond bugs.

### Pattern 5: Epoch-Based Policy Versioning
Approval policies use monotonic epochs to prevent stale cleanup:
```go
epoch := rt.setApprovalPolicy(handlers)
cleanup := func() { rt.clearApprovalPolicy(epoch) }
```

### Pattern 6: Deep Clone on Read
All state reads return cloned data (via JSON round-trip), making concurrent access safe.

### Pattern 7: Wildcard Handlers
Both client handlers and harness dispatcher support `"*"` as a catch-all method.

### Pattern 8: Lazy Initialization
The client's read loop starts only when first needed (`sync.Once`).

### Pattern 9: Composable Harnesses
Multiple harness modules merge into a single dispatcher without coupling.

### Convention: Interface Enforcement
```go
var _ cmds.BareCommand = (*harnessRunCommand)(nil)
```
Compile-time check that the struct implements the interface.

### Convention: Glazed CLI Framework
Commands use Glazed's field/schema system instead of raw Cobra flags, providing typed argument parsing with defaults.

---

## 13. Testing Strategy

### Unit Tests

| File | Tests |
|------|-------|
| `pkg/state/store_test.go` | Store CRUD, LRU eviction, snapshot isolation |
| `pkg/state/projector_test.go` | Event projection, all method types |
| `pkg/harness/dispatch_test.go` | Dispatcher routing, wildcard, request context |
| `pkg/harness/builtin/builtin_test.go` | Autopilot, plan gate, review gate, auto-compact, TDD loop |
| `pkg/codexrpc/client_test.go` | Client handshake FSM, request/response, handler dispatch |
| `pkg/codexrpc/threads_test.go` | Thread list/read RPC wrappers |
| `pkg/codexrpc/reliability_test.go` | Retry logic, backoff calculation |
| `pkg/js/runtime_test.go` | JS runtime, module loading, event dispatch |
| `cmd/.../harness_run_command_test.go` | Harness run integration |
| `cmd/.../harness_state_replay_command_test.go` | State replay integration |
| `cmd/.../thread_list_command_test.go` | Thread list integration |
| `cmd/.../thread_read_command_test.go` | Thread read integration |

### Testing Patterns

- **MemoryTransport**: Used for client tests - inject messages, verify sends
- **Test harnesses**: Built-in modules tested by creating harnesses, composing them, and dispatching events
- **Testable seams**: `newHarnessRunClient` and `newHarnessRuntime` are package-level vars that can be replaced in tests

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./pkg/codexrpc/...
go test ./pkg/js/...

# Specific test
go test ./pkg/state/ -run TestProjector
```

---

## 14. Key Dependencies

| Dependency | Purpose |
|---|---|
| `github.com/dop251/goja` | Pure-Go JavaScript engine (ECMAScript 5.1+) |
| `github.com/dop251/goja_nodejs` | Node.js compatibility (require, event loop, console) |
| `github.com/go-go-golems/go-go-goja` | Runtime ownership, thread-safe Runner, native module system |
| `github.com/go-go-golems/glazed` | CLI framework (typed flags, field schemas, output formatting) |
| `github.com/spf13/cobra` | CLI command tree |

---

## 15. Quick Reference: File Map

### Entry Point
- `cmd/openai-app-server/main.go` - `func main()` creates root command
- `cmd/openai-app-server/root.go` - Registers harness + thread command groups

### Core Pipeline (in execution order)
1. `pkg/config/defaults.go` - Load defaults
2. `cmd/openai-app-server/harness_run_command.go` - Parse flags, orchestrate
3. `pkg/codexrpc/transport_stdio.go` - Spawn subprocess
4. `pkg/codexrpc/client.go` - JSON-RPC client + handshake
5. `pkg/codexrpc/protocol.go` - Message format
6. `pkg/js/runtime.go` - JS runtime initialization
7. `pkg/js/module_codex.go` - Main developer API
8. `pkg/js/module_rpc.go` - Low-level RPC access
9. `pkg/js/module_approval.go` - Approval helpers
10. `pkg/js/module_ui.go` - UI event emission
11. `pkg/js/module_clock.go` - Time utilities

### State (optional, used by harness state-replay)
12. `pkg/state/models.go` - Data structures
13. `pkg/state/store.go` - In-memory store
14. `pkg/state/projector.go` - Event -> state projection

### Harness Framework (optional, composable Go-level handlers)
15. `pkg/harness/context.go` - Request context
16. `pkg/harness/dispatch.go` - Event dispatcher
17. `pkg/harness/compose.go` - Harness composition
18. `pkg/harness/builtin/*.go` - Pre-built harness modules

---

## Appendix A: Example Harness Scripts

### Minimal: List Threads

```javascript
const codex = require("codex");
const ui = require("ui");
const session = codex.connect();

session.request("thread/list", { limit: 1 })
  .then(result => ui.emit({ type: "test", ok: true, result }))
  .catch(err => ui.emit({ type: "test", ok: false, error: String(err) }));
```

### Full Smoke Test with Approval Handling

```javascript
const codex = require("codex");
const approval = require("approval");
const ui = require("ui");

const session = codex.connect();

// Handle approval requests
session.onRequest((evt) => {
  if (evt.method === "item/commandExecution/requestApproval") {
    if (evt.params.command.startsWith("curl")) {
      approval.acceptForSession(evt.id);
    } else {
      approval.decline(evt.id);
    }
  }
});

// Start thread and turn
session.threads.start({ model: "gpt-5" })
  .then(result => {
    const handle = session.thread(session.ids.thread(result));
    return handle.turn.start({
      input: [{ type: "text", text: "Run `curl -I https://example.com`" }]
    });
  })
  .then(() => ui.emit({ type: "smoke-done", ok: true }))
  .catch(err => ui.emit({ type: "smoke-done", ok: false, error: String(err) }));
```

### Advanced: Multi-Thread Research Pipeline (sketch)

```javascript
const codex = require("codex");
const ui = require("ui");

const session = codex.connect();

// Decline all approvals (read-only research)
session.approvals.setPolicy({
  command: () => "decline",
  fileChange: () => "decline",
  fallback: () => "decline"
});

async function main() {
  // Manager thread plans subtasks
  const managerResult = await session.threads.start();
  const manager = session.thread(session.ids.thread(managerResult));

  const planTurn = await manager.turn.start({
    input: [{ type: "text", text: "Plan 3 research subtasks for: ..." }]
  });
  await manager.turn.waitCompleted({ timeoutMs: 120000 });

  // Worker threads execute subtasks in parallel
  const workers = subtasks.map(async (task) => {
    const result = await session.threads.start();
    const worker = session.thread(session.ids.thread(result));
    await worker.turn.start({ input: [{ type: "text", text: task.question }] });
    await worker.turn.waitCompleted({ timeoutMs: 180000 });
  });
  await Promise.all(workers);

  ui.emit({ type: "research-complete", ok: true });
}

main().catch(err => ui.emit({ type: "research-complete", ok: false, error: String(err) }));
```

---

## Appendix B: Glossary

| Term | Meaning |
|---|---|
| **Codex App Server** | External process that runs the AI. Communicates via JSON-RPC 2.0 |
| **Harness** | A JavaScript script (or Go composition) that orchestrates AI interactions |
| **Thread** | A conversation session with the AI |
| **Turn** | A single request-response cycle within a thread |
| **Item** | A discrete unit within a turn (tool call, file change, command execution) |
| **Approval request** | The server asking permission to execute a command or modify a file |
| **Goja** | Pure-Go JavaScript engine |
| **Runner** | Thread-safe wrapper for executing code on the Goja VM |
| **Projector** | Maps incoming events to state mutations |
| **Settle** | Post-script pause to let async callbacks complete |
