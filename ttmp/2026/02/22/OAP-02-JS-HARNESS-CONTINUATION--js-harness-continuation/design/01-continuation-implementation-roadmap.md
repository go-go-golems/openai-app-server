---
Title: Continuation Implementation Roadmap
Ticket: OAP-02-JS-HARNESS-CONTINUATION
Status: active
Topics:
    - goja
    - glazed
    - openai-app-server
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/openai-app-server/harness_run_command.go
      Note: |-
        Existing harness execution command and wait-gate behavior
        Live run path and stop-gate execution mechanism
    - Path: pkg/codexrpc/client.go
      Note: |-
        Existing transport/handshake/router foundation to build on
        Transport and routing baseline that phases build on
    - Path: pkg/js/module_codex.go
      Note: Current JS API surface to expand in Phase 3
    - Path: pkg/js/runtime.go
      Note: Existing module-based JS runtime foundation
    - Path: ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/design/01-openai-app-server-js-harness-architecture.md
      Note: Original architecture goals and acceptance criteria baseline
    - Path: ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/design/02-codex-app-server-contract-validation-report.md
      Note: Verified approval response compatibility findings from OAP-01
ExternalSources:
    - https://developers.openai.com/codex/app-server/
Summary: Detailed execution roadmap for remaining work after OAP-01 baseline, with phased deliverables, test gates, and concrete implementation units.
LastUpdated: 2026-02-22T22:42:00-05:00
WhatFor: Convert the architecture baseline into an execution-ready implementation program with progressive validation.
WhenToUse: Use as the primary execution plan for OAP-02 implementation and task tracking.
---


# Continuation Implementation Roadmap

## 1. Objective

OAP-01 proved the runtime foundation, live transport integration, and approval-response compatibility against local `codex-cli 0.104.0`. OAP-02 completes the remaining architecture scope so the project evolves from a live probe harness into a production-usable JS harness framework and CLI.

This roadmap is intentionally implementation-first: every phase has a concrete output, a test shape, and a stop-gate for any real harness script.

## 2. Baseline at OAP-02 Start

Already implemented and validated in OAP-01:

- Glazed CLI root with `harness run`, `thread list`, and `thread read` skeleton.
- JSON-RPC client with strict handshake sequencing and request/notification routing.
- Stdio transport and live app-server integration path.
- goja runtime with native modules: `codex`, `rpc`, `ui`, `clock`, `approval`.
- Real harness execution flow with deterministic UI wait gates.
- Approval response compatibility fix confirmed by real run (`decision`-wrapped response shape).

Still missing versus original architecture and imported requirements:

- Full thread/turn/review wrapper surface in JS.
- Persistent projected store for thread/turn/item state.
- Harness framework (`defineHarness`, `composeHarnesses`, deterministic handler order, exactly-once request response guardrails).
- Built-in harness implementations (autopilot approvals, plan gate, tdd loop, review gate, compaction).
- Richer CLI command surface and help pages.
- Robust replay/contract regression test suite.

## 3. Scope and Non-Goals

In-scope for OAP-02:

- End-to-end implementation of the core harness framework and first built-ins.
- Progressive hardening of transport/runtime behavior and CLI UX.
- Documentation parity between implementation and ticket plans.

Out-of-scope for OAP-02:

- New transport backends beyond current stdio unless required by blocker.
- Large UI application work; keep UI bridge as runtime events and prompts.
- Broad refactors unrelated to harness architecture objectives.

## 4. Implementation Phases

## Phase 0: OAP-02 Handoff and Program Setup

Deliverable:

- New ticket initialized with detailed roadmap, full task breakdown, and diary/changelog cadence.

Implementation units:

- Create and relate roadmap + diary.
- Capture OAP-01 closure evidence and carry-over risks.
- Define explicit stop-gate policy for real harness script runs.

Validation:

- `docmgr doctor --ticket OAP-02-JS-HARNESS-CONTINUATION --stale-after 30`

## Phase 1: Complete Thread Read Path and Typed Thread APIs

Deliverable:

- `thread read` performs real RPC with typed decoding and structured CLI output.

Implementation units:

- Add `Client.ThreadRead(...)` helper in `pkg/codexrpc`.
- Replace `thread read` placeholder output with live call wiring.
- Add unit/integration tests using memory transport.

Validation:

- `go test ./...`
- CLI integration tests for success and malformed payload handling.

## Phase 2: State Projection Store

Deliverable:

- In-memory projected store for thread/turn/item lifecycle with diff/plan snapshots.

Implementation units:

- Add `pkg/state` models/store/projector.
- Project notifications (`thread/*`, `turn/*`, `item/*`, `turn/diff/updated`, `turn/plan/updated`).
- Add query methods used by runtime and CLI.

Validation:

- Projector unit tests using replayed notifications.
- Deterministic assertion tests for final projected state.

## Phase 3: JS Session Ergonomics (Thread/Turn Wrappers)

Deliverable:

- JS API supports ergonomic wrappers over session RPC:
  - `session.threads.start/list/read`
  - `thread.turn.start/steer/interrupt`
  - `thread.review.start`

Implementation units:

- Extend `pkg/js/module_codex.go` with object model wrappers.
- Keep `session.request` passthrough for low-level operations.
- Maintain runtimeowner invariants for callback execution.

Validation:

- Runtime contract tests for method availability and parameter passing.
- One live smoke script (stop-gated) demonstrating wrapper calls.

## Phase 4: Harness Runtime Framework Core

Deliverable:

- Framework APIs for `defineHarness` and `composeHarnesses` with deterministic dispatch.

Implementation units:

- Add `pkg/harness` runtime context/dispatch/composition.
- Handler registration by method name (`notification`, `request`).
- Exactly-once request-response guardrail with explicit diagnostics.

Validation:

- Unit tests for handler ordering and composition semantics.
- Unit tests for duplicate response prevention and timeout behavior.

## Phase 5: Built-in Harnesses I (Autopilot + Plan Gate)

Deliverable:

- First built-ins runnable and testable end-to-end:
  - autopilot approvals
  - plan gate (steer/interrupt behavior)

Implementation units:

- Add built-ins in `pkg/harness/builtin`.
- Add corresponding JS examples in ticket `scripts/`.
- Update playbooks with expected markers.

Validation:

- Replay tests against fixture transcripts.
- Live script run stop-gate for each built-in.

## Phase 6: Built-in Harnesses II (TDD/Review/Compaction)

Deliverable:

- Additional built-ins implemented and validated:
  - tdd loop
  - review gate
  - token-usage compaction trigger

Implementation units:

- Extend built-in package set and helper utilities.
- Ensure state-store integration for diff/plan/token triggers.

Validation:

- Unit tests for decision logic.
- Controlled live scripts with stop-gates.

## Phase 7: Reliability Hardening

Deliverable:

- Hardened runtime/transport behavior under load and protocol edge cases.

Implementation units:

- Add retry/backoff policy for retryable errors.
- Add bounded event buffers and configurable retention.
- Add notification opt-out capability in initialize flow.

Validation:

- Unit tests for retry policy and buffer bounds.
- Fault-injection integration tests.

## Phase 8: CLI and Docs Completion

Deliverable:

- Production-grade CLI/help/docs with examples and troubleshooting guidance.

Implementation units:

- Complete command set for harness inspect/state views.
- Add help pages and examples.
- Final documentation sync and publication artifacts.

Validation:

- `go test ./...`
- `docmgr doctor` clean
- final scripted smoke suite (stop-gated before each real harness script)

## 5. Acceptance Criteria for OAP-02 Completion

OAP-02 is complete when all are true:

1. `thread read` is live-wired and tested.
2. State projector exists and is used for diff/plan/item lifecycle views.
3. JS wrappers cover thread/turn/review operations with tests.
4. Harness composition framework is implemented with deterministic ordering tests.
5. At least two built-in harnesses are live-validated behind stop-gates.
6. Reliability features (retry/bounds/opt-out) are implemented and tested.
7. CLI/docs are synchronized with implementation and doctor-clean.

## 6. Risks and Mitigations

Risk: protocol drift between docs and local runtime.

- Mitigation: treat local codex-rs schema and live runs as contract source; keep adapter logic centralized.

Risk: deadlocks/races in goja callback paths.

- Mitigation: continue strict runtimeowner pattern and require tests for async callback behavior.

Risk: false positives from noisy live outputs.

- Mitigation: continue explicit UI completion markers and wait-gated command mode.

Risk: scope creep from too many built-ins at once.

- Mitigation: split built-ins across two phases and ship each with tests first.

## 7. Execution Policy for Real Harness Scripts

For this ticket, every real harness script run follows:

1. Explain script purpose and exact command.
2. Wait for explicit user approval.
3. Run once with wait-gated completion marker.
4. Record outcomes in diary/changelog/tasks before next implementation step.

## 8. Immediate Next Steps

1. Complete Phase 1 (`thread read` live wiring + tests).
2. Commit Phase 1 changes.
3. Update diary/changelog/tasks.
4. Begin Phase 2 store scaffolding.
