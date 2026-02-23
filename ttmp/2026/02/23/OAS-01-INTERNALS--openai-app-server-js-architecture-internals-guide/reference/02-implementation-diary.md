---
Title: Implementation Diary
Ticket: OAS-01-INTERNALS
Status: active
Topics:
    - architecture
    - documentation
    - onboarding
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Step-by-step narrative of the deep-dive exploration and documentation of the openai-app-server codebase"
LastUpdated: 2026-02-23T10:02:41.532730184-05:00
WhatFor: "Recording the investigation process for future reference and reproducibility"
WhenToUse: "When reviewing how the architecture guide was produced, or when doing similar explorations"
---

# Implementation Diary: OAS-01-INTERNALS Deep Dive

## Session: 2026-02-23

### Phase 1: Orientation (10:00)

**What I did**: Started by exploring the top-level workspace structure. Discovered this is a Go workspace (`go.work`) with 4 modules: geppetto, go-go-goja, pinocchio, and openai-app-server.

**Key discovery**: The openai-app-server is relatively self-contained. It depends on go-go-goja (for JS runtime plumbing) and glazed (for CLI scaffolding), but NOT on geppetto or pinocchio. This means it can be understood in isolation.

**What worked**: Running parallel exploration agents to map the entire workspace while simultaneously reading key files. This gave me both breadth (full directory structure) and depth (actual code understanding) simultaneously.

### Phase 2: Deep Code Read (10:05)

**What I did**: Systematically read every `.go` file in the openai-app-server module:
- `pkg/config/defaults.go` (57 lines) - Simple but important: defaults for transport, model, policies
- `pkg/codexrpc/*.go` (~900 lines total) - The RPC backbone
- `pkg/js/*.go` (~1300 lines total) - The JavaScript bridge
- `pkg/state/*.go` (~540 lines total) - Event-sourced state
- `pkg/harness/**/*.go` (~750 lines total) - Composable harness framework
- `cmd/openai-app-server/*.go` (~750 lines total) - CLI commands

**Key insight**: `pkg/js/runtime.go` at 793 lines is the heart of the system. It handles:
- Promise bridging between Go goroutines and JS event loop
- Event waiter system (waitFor with timeout + predicate)
- Approval policy with epoch-based versioning
- Event metrics tracking

**What was tricky**: Understanding the threading model. The key is `runtimeowner.Runner` from go-go-goja - it provides a thread-safe way to execute code on the JS VM. All JS execution must go through `runner.Call()` (blocking) or `runner.Post()` (non-blocking). Go goroutines that perform async RPC calls then post results back to the JS thread.

### Phase 3: Understanding the Event Flow (10:15)

**What I did**: Traced the complete data flow from CLI invocation through script execution.

**The critical wiring** happens in `harness_run_command.go:249-255`:

```go
client.OnNotification("*", func(_ context.Context, msg *codexrpc.Message) {
    _ = runtime.EmitRPCNotification(msg.Method, decodeMessagePayload(msg.Params))
})
client.OnRequest("*", func(_ context.Context, msg *codexrpc.Message) {
    _ = runtime.EmitRPCRequest(msg.ID, msg.Method, decodeMessagePayload(msg.Params))
})
```

This registers wildcard handlers on the Go-level RPC client that forward EVERYTHING into the JS runtime. The JS runtime then fans out to registered JS handlers, event waiters, and approval policy handlers.

**What was tricky**: The bidirectional nature of requests. The server can send requests TO us (approval requests), and we can send requests TO the server (thread/start, turn/start). The same `Request()` mechanism handles both directions, but the approval flow adds the complexity of the JS runtime needing to respond asynchronously.

### Phase 4: Studying Example Scripts (10:20)

**What I did**: Read the example harness scripts in `ttmp/` to understand real-world usage patterns.

**Scripts reviewed**:
- `01-first-real-harness.js` - Minimal: list threads, emit UI event
- `05-final-full-smoke.js` - Full flow: start thread, start turn, handle approvals, read thread
- `01-scientific-rlm-research-db.js` - Advanced: multi-thread research pipeline with database storage

**Key observation**: The scripts range from ~12 lines (simple list) to ~256 lines (research pipeline). The API is ergonomic - `session.thread(id).turn.start(...)` reads naturally. The `waitFor()` + `waitCompleted()` primitives handle async orchestration cleanly.

**What I noticed**: The advanced sketches (OAP-04) use a `db` module that doesn't exist in the current openai-app-server codebase. These are aspirational sketches for future capabilities. The current modules are: codex, rpc, approval, ui, clock.

### Phase 5: Writing the Architecture Guide (10:25)

**What I did**: Wrote a comprehensive architecture guide covering all packages, with:
- ASCII diagrams for the mental model
- Complete API reference for JS modules
- End-to-end data flow walkthrough
- Design patterns catalog
- File map for quick navigation
- Example harness scripts at multiple complexity levels

**Decisions made**:
- Organized by conceptual layers (protocol -> runtime -> modules -> harness -> state -> CLI) rather than alphabetically
- Included the end-to-end walkthrough as the centerpiece - this is what connects all the pieces
- Added appendices with practical examples and glossary
- Kept the focus on "how it actually works" rather than "what each function does"

**What I would improve**: The guide could benefit from diagrams showing the concurrency model more explicitly - which goroutines exist, what channels connect them, and where the synchronization points are. Also, the built-in harness modules (autopilot, plan_gate, etc.) are currently Go-only and not directly usable from JS scripts - this architectural gap could be documented more explicitly.

### Phase 6: Document Storage & Upload (10:40)

**What I did**:
1. Created docmgr ticket OAS-01-INTERNALS
2. Added two documents: architecture guide and diary
3. Wrote both documents
4. Uploading to reMarkable

### Review Instructions

To validate this documentation:
1. Read `pkg/js/runtime.go` and verify the event flow description matches the code
2. Run `go test ./...` to confirm the test descriptions are accurate
3. Try running a simple harness: `go run ./cmd/openai-app-server harness run --script <path> --dry-run` to verify CLI flags
4. Compare the module API tables against the actual `module_*.go` files

### Follow-ups

- [ ] Document the `db` module when it's implemented
- [ ] Add sequence diagrams for the approval flow
- [ ] Document how to write and test new built-in harness modules
- [ ] Add troubleshooting section (common errors, debugging tips)
