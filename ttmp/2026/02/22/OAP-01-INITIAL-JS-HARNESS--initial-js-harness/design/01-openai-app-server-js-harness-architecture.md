---
Title: OpenAI App Server JS Harness Architecture
Ticket: OAP-01-INITIAL-JS-HARNESS
Status: active
Topics:
    - goja
    - glazed
    - openai-app-server
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/pkg/js/modules/geppetto/api_sessions.go
      Note: Promise and streaming session lifecycle patterns
    - Path: geppetto/pkg/js/modules/geppetto/api_tool_hooks.go
      Note: Tool hook policy and retry/error handling patterns
    - Path: geppetto/pkg/js/modules/geppetto/api_tools_registry.go
      Note: JS tool registry integration and callback bridging
    - Path: geppetto/pkg/js/modules/geppetto/module.go
      Note: JS module registration and export wiring precedent
    - Path: geppetto/pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl
      Note: TypeScript contract pattern for JS SDK
    - Path: go-go-goja/modules/common.go
      Note: Native module registration architecture
    - Path: go-go-goja/pkg/runtimeowner/runner.go
      Note: Runtime owner-thread execution model and safety invariants
    - Path: go-go-goja/pkg/runtimeowner/types.go
      Note: Runtime scheduler and runner interface contracts
    - Path: openai-app-server/cmd/XXX/main.go
      Note: Current CLI scaffold gap to replace
    - Path: openai-app-server/cmd/openai-app-server/harness_run_command.go
      Note: |-
        Implemented harness command skeleton
        Implemented live harness execution path
    - Path: openai-app-server/cmd/openai-app-server/harness_run_command_test.go
      Note: Implemented harness run command integration test
    - Path: openai-app-server/cmd/openai-app-server/root.go
      Note: Implemented phase-1 Glazed root command scaffold
    - Path: openai-app-server/cmd/openai-app-server/thread_list_command.go
      Note: |-
        Implemented thread command skeleton
        Phase-3 thread list CLI wiring to codex client
    - Path: openai-app-server/cmd/openai-app-server/thread_list_command_test.go
      Note: Phase-3 fake-transport integration test
    - Path: openai-app-server/cmd/openai-app-server/thread_read_command.go
      Note: Phase-3 thread read command skeleton
    - Path: openai-app-server/go.mod
      Note: Current module placeholder requiring bootstrap changes
    - Path: openai-app-server/pkg/codexrpc/client.go
      Note: Implemented handshake state machine and message routing
    - Path: openai-app-server/pkg/codexrpc/client_test.go
      Note: Implemented handshake unit tests
    - Path: openai-app-server/pkg/codexrpc/memory_transport.go
      Note: In-memory transport for integration tests
    - Path: openai-app-server/pkg/codexrpc/protocol.go
      Note: Implemented JSON-RPC envelope types
    - Path: openai-app-server/pkg/codexrpc/threads.go
      Note: Thread list API helper
    - Path: openai-app-server/pkg/codexrpc/transport.go
      Note: Implemented transport abstraction
    - Path: openai-app-server/pkg/codexrpc/transport_stdio.go
      Note: |-
        Implemented stdio transport skeleton
        Enabled stderr passthrough for live diagnostics
    - Path: openai-app-server/pkg/config/defaults.go
      Note: Implemented transport/session defaults
    - Path: openai-app-server/pkg/js/module_codex.go
      Note: Phase-4 codex module skeleton
    - Path: openai-app-server/pkg/js/runtime.go
      Note: Phase-4 runtime bootstrap and host primitives
    - Path: openai-app-server/pkg/js/runtime_test.go
      Note: Phase-4 runtime unit tests
    - Path: openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/sources/local/app-server-js.md
      Note: Imported requirements and API baseline
ExternalSources:
    - local:app-server-js.md
Summary: Architecture and implementation design for a goja-based JS harness runtime and Glazed CLI in openai-app-server, derived from imported requirements and existing go-go-goja/geppetto integration patterns.
LastUpdated: 2026-02-23T02:05:00-05:00
WhatFor: Design the first production-ready JS harness for Codex App Server in this repository.
WhenToUse: Use as the implementation blueprint for building openai-app-server from the current scaffold.
---







# OpenAI App Server JS Harness Architecture

## 1. Objective

This document designs the first implementation of `openai-app-server` as a Go + goja host for a JavaScript harness API on top of Codex App Server JSON-RPC streams.

The design is based on three concrete inputs:

1. Imported source document `app-server-js.md` (the target API and behavior).
2. Existing native-module and runtime patterns in `go-go-goja`.
3. Existing JS-facing orchestration and async/event bridging patterns in `geppetto`.

The intended outcome is a codebase that provides:

- A stable transport/client layer for Codex App Server.
- A runtime-safe goja host with explicit owner-thread execution.
- A JS SDK (`codex`) for event-native harness authoring.
- A Glazed-based CLI for running harness scripts, inspecting state, and managing sessions.

## 2. Current State Analysis

### 2.1 Imported Requirements (`app-server-js.md`)

The imported design requests a layered architecture with these explicit capabilities:

- Mandatory app-server handshake (`initialize`, then `initialized`) before all other methods.
- Event-native API: notifications and server-initiated requests are first-class.
- Turn-centric controls: `turn/start`, `turn/steer`, `turn/interrupt`, `review/start`.
- Approval request handling (`item/commandExecution/requestApproval`, `item/fileChange/requestApproval`) with exact response semantics.
- Harness composition model (`defineHarness`, `composeHarnesses`) and policy plugins.
- UI bridge and UI-oriented event projections (diff, plan, streaming deltas, command output).
- Derived state store keyed by thread and turn.

The imported file also defines representative harness patterns we should support without runtime refactors:

- Autopilot approvals.
- Plan-gated execution.
- TDD feedback loops.
- Review gate loops.
- Recursive decomposition with worker threads.
- Auto-compaction on token thresholds.

These examples effectively become API acceptance tests for the first implementation.

### 2.2 `go-go-goja` Integration Patterns to Reuse

From `go-go-goja/modules/common.go`, module integration is centered on:

- `NativeModule` interface with `Name`, `Doc`, and `Loader(*goja.Runtime, *goja.Object)`.
- Global module registry (`modules.Register`, `modules.EnableAll`) mapped into `goja_nodejs/require`.
- Native module registration through `RegisterNativeModule`.

From `go-go-goja/pkg/runtimeowner/runner.go`, runtime safety is solved by:

- Serializing runtime access onto a scheduler loop (`Runner.Call`, `Runner.Post`).
- Owner-context detection to avoid illegal cross-goroutine goja access.
- Explicit cancellation/schedule rejection semantics and panic recovery.

This owner model is non-negotiable for `openai-app-server`: any JS callback, promise settlement, or event fanout must run through runtimeowner.

### 2.3 `geppetto` Integration Patterns to Reuse

`geppetto` is the strongest precedent for goja-in-Go orchestration in this workspace:

1. Module registration shape (`pkg/js/modules/geppetto/module.go`):
- Runtime-local module state object (`moduleRuntime`) per VM.
- Export installation in a single place.
- Hidden reference attachment for JS objects (`__geppetto_ref`) to preserve Go pointers.

2. Owner-thread bridge (`pkg/js/runtimebridge/bridge.go`, `api_owner_bridge.go`):
- `Call`/`Post` wrappers used for all JS callback invocation from Go.

3. Async/event integration (`api_sessions.go`, `api_events.go`):
- Promise creation on VM thread.
- Settlement via owner-thread post from background goroutines.
- Event collector delivering typed payloads to JS listeners.

4. Typed JS API discipline (`spec/geppetto.d.ts.tmpl`):
- Generated TypeScript contracts for JS users.
- Stable function/object model and strict option decoding.

5. Go <-> JS codec boundary (`codec.go`, `api_builder_options.go`):
- Explicit map/slice decoding and key normalization.
- Clone-to-JSON behavior to reduce mutability hazards.

`openai-app-server` should follow these patterns almost verbatim where applicable, rather than inventing a second integration style.

### 2.4 Gaps in `openai-app-server`

Current codebase status:

- `openai-app-server/cmd/XXX/main.go` is empty.
- `openai-app-server/pkg/doc.go` is empty.
- `openai-app-server/go.mod` is still placeholder module `github.com/go-go-golems/XXX`.

Conclusion: this is a greenfield implementation from scaffold, with no migration constraints inside this repo.

## 3. Target System Architecture

### 3.1 Layered Design

```
Glazed CLI (cmd/openai-app-server)
        |
        v
Application Services (pkg/app)
- run harness, render events, state snapshots
        |
        v
Harness Runtime (pkg/harness)
- plugin lifecycle, dispatch, approvals, composition
        |
        v
Codex Client (pkg/codexrpc)
- transport, handshake, RPC router, state projection
        |
        v
Transport (stdio jsonl / websocket)
```

JS-facing interfaces are exposed from a host module injected into goja and optionally via `require("codex")`.

### 3.2 Proposed Package Layout

```
openai-app-server/
  cmd/openai-app-server/
    main.go
    root.go
    harness_run.go
    harness_inspect.go
    thread_list.go
  pkg/codexrpc/
    client.go
    transport_stdio.go
    transport_ws.go
    protocol.go
    router.go
    errors.go
  pkg/state/
    store.go
    projector.go
    models.go
  pkg/js/
    runtime.go
    host_primitives.go
    module_codex.go
    module_ui.go
    module_clock.go
    module_fs.go
    module_exec.go
    typescript/
      codex.d.ts
  pkg/harness/
    runtime.go
    compose.go
    dispatch.go
    approvals.go
    context.go
    builtin/
      autopilot.go
      plan_gate.go
      tdd_loop.go
      review_gate.go
      auto_compact.go
  pkg/app/
    run_session.go
    reconnect.go
    telemetry.go
  pkg/config/
    config.go
    defaults.go
```

This separates protocol concerns from JS runtime concerns, and both from CLI concerns.

## 4. Goja Host and JS Runtime Design

### 4.1 Runtime Lifecycle

Per CLI invocation (or interactive session):

1. Build scheduler (`eventloop` or equivalent loop scheduler).
2. Create `goja.Runtime`.
3. Wrap runtime with `runtimeowner.Runner`.
4. Register native modules into `require.Registry`.
5. Evaluate user harness script.
6. Bind process signals and teardown sequence.

Critical invariant: **all goja access runs on owner thread**.

### 4.2 Host Primitives

Adopt imported proposal but implement as strongly typed host modules:

- `__host.rpc.request(method, params) -> Promise`
- `__host.rpc.notify(method, params) -> void`
- `__host.rpc.onNotification(fn)`
- `__host.rpc.onRequest(fn)`
- `__host.ui.emit(event)`
- `__host.ui.onEvent(fn)`
- `__host.clock.nowMs()`
- `__host.clock.sleep(ms)`

Optional modules gated by policy:

- `fs` (read/write/glob operations).
- `exec` (command execution).

The imported document suggests both; we should keep them optional because deployment profiles differ in risk tolerance.

### 4.3 Module Strategy

Use two styles together:

1. `require("codex")` native module (versioned API).
2. `globalThis.__host` low-level primitives.

Why both:

- Low-level primitives are stable kernel-like contracts.
- `codex` module can evolve ergonomics and wrappers while preserving host primitives.

### 4.4 Type Contract Strategy

Follow `geppetto` precedent and generate/maintain `codex.d.ts` for harness authors.

Versioning rule:

- `codex.version` semver-style string.
- Backward-incompatible API shifts only when major changes are unavoidable.

## 5. Codex RPC Client and Event Router

### 5.1 Transport

Implement transport abstraction:

```go
type Transport interface {
  Send(ctx context.Context, msg *Message) error
  Recv(ctx context.Context) (*Message, error)
  Close() error
}
```

Transport implementations:

- `stdio jsonl` default.
- `websocket` optional/experimental.

### 5.2 Handshake Enforcement

Client startup sequence must enforce:

1. Send `initialize` request and await result.
2. Send `initialized` notification.
3. Only then allow all other methods.

If the script invokes thread/turn APIs before handshake completion, return explicit client error.

### 5.3 Router and Correlation

Router responsibilities:

- Map outgoing request IDs to promise resolvers.
- Route incoming responses/errors to awaiting callers.
- Route incoming notifications to subscription pipeline.
- Route incoming server requests into request handlers requiring explicit response.

One-response rule for server requests should be enforced in code (guard double-send).

### 5.4 Event Normalization

Normalized event shape:

```ts
type CodexEvent = {
  kind: "notification" | "request" | "response" | "error";
  method?: string;
  params?: any;
  requestId?: string | number;
  threadId?: string;
  turnId?: string;
  itemId?: string;
  timestampMs: number;
}
```

Add method-specific helpers while preserving raw payload.

### 5.5 Reconnect and State Reconstruction

Use persisted thread data (`thread/list`, `thread/read includeTurns`) to rebuild state projection if reconnect/restart occurs.

This should live in `pkg/app/reconnect.go` so it can be reused by both CLI and future UI server mode.

## 6. State Projection and Store

### 6.1 Store Responsibilities

Implement a store with three scopes:

- Global state.
- Per-thread state.
- Per-turn state.

The imported harness examples require fast lookups of:

- latest diff (`turn/diff/updated`)
- latest plan (`turn/plan/updated`)
- token usage summaries
- running/completed item statuses

### 6.2 Data Model

```go
type Store struct {
  Global map[string]any
  Threads map[string]*ThreadState
}

type ThreadState struct {
  Meta ThreadMeta
  Data map[string]any
  Turns map[string]*TurnState
}

type TurnState struct {
  Meta TurnMeta
  Data map[string]any
  Items map[string]*ItemState
}
```

Need explicit lock strategy because events and JS calls are concurrent.

## 7. Harness Runtime Design

### 7.1 Core APIs (JS)

- `codex.connect(opts)`
- `codex.defineHarness(def)`
- `codex.composeHarnesses(list)`
- `ctx.respond(...)`
- `ctx.approvals.accept|acceptForSession|decline|cancel`

This aligns with imported source while keeping implementation minimal.

### 7.2 Dispatch Model

For every incoming normalized event:

1. Resolve context (`thread`, `turn`, `item`, `state`).
2. Build immutable event view for handlers.
3. Execute handlers in composition order.
4. For requests, assert exactly one response.
5. Capture handler errors and route to runtime logger + UI diagnostics.

### 7.3 Composition Semantics

- Default: ordered pipeline.
- Handlers may return `consume=true` to stop propagation.
- Non-consuming handlers continue.
- For approval requests, default fallback is `cancel` if no handler responds and timeout is reached.

### 7.4 Built-in Harnesses

Ship built-ins mirroring imported examples:

- `autopilotApprovals`
- `planGate`
- `tddLoop`
- `reviewGate`
- `autoCompact`

Decomposition harness can be added as phase 2 because it depends on more robust worker orchestration.

## 8. Approval and Security Model

### 8.1 Approval Routing

Map app-server approval requests to typed helpers:

- `item/commandExecution/requestApproval`
- `item/fileChange/requestApproval`

Response vocabulary and validation:

- `accept`
- `acceptForSession`
- `decline`
- `cancel`
- plus optional amendment payload where supported.

### 8.2 Policy Layers

Three layers:

1. Runtime-level hard-deny rules (protected paths/commands).
2. Harness-level policy logic (risk heuristics).
3. User-interactive prompts (UI bridge or CLI prompt).

### 8.3 Threat Notes

Key risk surfaces:

- Untrusted JS harness code invoking `exec`/`fs`.
- Approval bypass due to incorrect request-response wiring.
- Cross-turn data leakage if thread/turn scoping is wrong.

Mitigations:

- Disable dangerous modules by default.
- Strict response validation and request deadline handling.
- Explicit thread/turn context binding in store lookups.

## 9. Glazed CLI Design

Use Glazed command construction per repository conventions.

### 9.1 Command Tree

Root binary: `openai-app-server`.

Commands:

1. `harness run`
- Run a JS harness against app-server.
- Inputs: script path, transport settings, model/cwd defaults, approval mode.

2. `harness inspect`
- Validate harness exports and print API/schema metadata.

3. `thread list`
- List available threads with summary fields.

4. `thread read`
- Read a thread, optional `--include-turns`.

5. `turn start`
- Convenience wrapper for manual turn execution and smoke tests.

### 9.2 Glazed Settings and Sections

Each command should include:

- Glazed output schema section (`settings.NewGlazedSchema`).
- Command settings section (`cli.NewCommandSettingsSection`).
- Command-specific flags declared via `fields.New`.

Example `harness run` settings struct:

```go
type HarnessRunSettings struct {
  ScriptPath     string `glazed:"script"`
  Transport      string `glazed:"transport"`
  StdioCommand   string `glazed:"stdio-command"`
  StdioArgs      string `glazed:"stdio-args"`
  WebsocketURL   string `glazed:"ws-url"`
  Model          string `glazed:"model"`
  Cwd            string `glazed:"cwd"`
  ApprovalPolicy string `glazed:"approval-policy"`
  SandboxPolicy  string `glazed:"sandbox-policy"`
  SessionName    string `glazed:"session-name"`
}
```

Output mode defaults:

- `harness run` should default to streaming-friendly output (`yaml` + stream true) for event timelines.

### 9.3 CLI Output Modes

- `table` for summaries (`thread list`).
- `json`/`yaml` for automation.
- `text` streaming for live event display.

### 9.4 Help Integration

Wire Glazed help system early because this project is command-heavy and protocol semantics are not trivial.

Minimum docs:

- `harness run` examples with approvals and steering.
- Event names and approval response semantics.
- Troubleshooting handshake and overload retry behavior.

## 10. Testing Strategy

### 10.1 Unit Tests

- RPC router request-response correlation.
- Handshake state machine.
- Store projector updates for diff/plan/item streams.
- Approval request one-response enforcement.
- Runtimeowner bridge behavior (especially promise settlement).

### 10.2 Integration Tests

- Fake app-server transport replaying known event transcripts.
- Harness scripts for:
  - autopilot approvals,
  - plan gate,
  - tdd loop.
- Verify generated actions (`turn/steer`, `turn/interrupt`, `turn/start`).

### 10.3 Contract Tests (JS API)

Like `geppetto/module_test.go`, execute JS snippets and assert:

- API availability and types.
- event subscription and callback execution order.
- approval helper responses.
- state access semantics across thread/turn scopes.

### 10.4 End-to-End Smoke

- Launch against real app-server stdio transport.
- Run a minimal harness script that:
  - starts thread,
  - starts turn,
  - receives deltas,
  - handles at least one approval request,
  - completes.

## 11. Phased Implementation Plan

### Phase 0: Bootstrap

- Rename module path and binary from placeholders.
- Add root command and logging setup.
- Add config object and defaults.

Deliverable: compilable CLI skeleton.

### Phase 1: Transport + Handshake + Basic RPC

- Implement codexrpc client with stdio transport.
- Add strict handshake sequencing.
- Add raw request/notify methods.

Deliverable: command to initialize and call `thread/list`.

### Phase 2: goja Runtime + Host Primitives

- Integrate runtimeowner + event loop.
- Expose `__host` primitives.
- Implement `codex.connect` in JS module.

Deliverable: JS script can connect and issue one RPC call.

### Phase 3: Router + State Store + Event Subscriptions

- Add notification/request routing.
- Add store projections for thread/turn/item lifecycle.

Deliverable: live event stream with projected state.

### Phase 4: Harness Runtime + Approvals

- `defineHarness`, `composeHarnesses`, dispatch pipeline.
- request handling with single-response guarantees.

Deliverable: autopilot approvals harness works end-to-end.

### Phase 5: Glazed UX + Built-ins + Docs

- Implement full command set.
- Add built-in harnesses and examples.
- Add help pages and troubleshooting docs.

Deliverable: production-usable CLI with examples.

## 12. Risks and Mitigations

1. Runtime race bugs in JS callbacks.
- Mitigation: mandatory runtimeowner wrapper for all callback paths.

2. Approval deadlocks due to missing response.
- Mitigation: request timeout + fallback decision + explicit diagnostics.

3. Event-volume pressure causing memory growth.
- Mitigation: bounded in-memory buffers; configurable retention; optional notification opt-out.

4. Transport instability (websocket overload errors).
- Mitigation: centralized retry/backoff and idempotence guidance.

5. API drift between imported design and actual app-server schemas.
- Mitigation: schema adapters in router with feature flags; integration tests against real streams.

## 13. Concrete Design Decisions

1. Reuse runtimeowner and geppetto-style owner-thread patterns without modification in first pass.
2. Use both `__host` primitives and `require("codex")` ergonomic layer.
3. Keep CLI Glazed-first, not Cobra-only.
4. Ship stdio transport first; websocket second.
5. Implement built-in harnesses for approvals, planning, test loops, review gate, compaction.

## 14. Acceptance Criteria

The first implementation is acceptable when all are true:

1. `harness run --script examples/harness/autopilot.js` can connect, start thread, and process events.
2. At least one server-initiated approval request is handled and responded to correctly.
3. `turn/diff/updated` and `turn/plan/updated` are projected into state and visible in CLI output.
4. Harness composition order is deterministic and tested.
5. CLI commands use Glazed sections and return structured output.
6. End-to-end smoke test passes against live app-server.

## 15. Mapping Imported Harness Examples to Implementation Units

- Autopilot approvals -> `pkg/harness/builtin/autopilot.go`
- Plan gate -> `pkg/harness/builtin/plan_gate.go`
- TDD loop -> `pkg/harness/builtin/tdd_loop.go`
- Review gate -> `pkg/harness/builtin/review_gate.go`
- Auto-compaction -> `pkg/harness/builtin/auto_compact.go`
- Recursive decomposition -> future `pkg/harness/builtin/decompose.go`

## 16. Recommended Immediate Next Changes

1. Replace placeholder module path in `openai-app-server/go.mod`.
2. Replace `cmd/XXX/main.go` with real root command scaffold.
3. Create `pkg/codexrpc/client.go` with handshake state machine tests.
4. Create `pkg/js/runtime.go` using runtimeowner runner and registry wiring.
5. Add one smoke harness example under `openai-app-server/examples/harness/`.

## 17. References Used

- `openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/sources/local/app-server-js.md`
- `go-go-goja/modules/common.go`
- `go-go-goja/pkg/runtimeowner/runner.go`
- `go-go-goja/pkg/runtimeowner/types.go`
- `geppetto/pkg/js/modules/geppetto/module.go`
- `geppetto/pkg/js/modules/geppetto/api_sessions.go`
- `geppetto/pkg/js/modules/geppetto/api_events.go`
- `geppetto/pkg/js/modules/geppetto/api_tools_registry.go`
- `geppetto/pkg/js/modules/geppetto/api_tool_hooks.go`
- `geppetto/pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl`

