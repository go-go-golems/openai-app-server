# Tasks

## Phase 0: OAP-02 Handoff and Program Setup

- [x] Create OAP-02 ticket workspace
- [x] Add OAP-02 diary document
- [x] Add continuation implementation roadmap document
- [x] Prefill multi-phase tasks from roadmap
- [x] Relate key code/docs files into OAP-02 index and roadmap docs
- [x] Run `docmgr doctor --ticket OAP-02-JS-HARNESS-CONTINUATION --stale-after 30`
- [x] Commit Phase 0 documentation bootstrap

## Phase 1: Thread Read Live Wiring

- [x] Add typed `ThreadRead` API in `pkg/codexrpc`
- [x] Replace `thread read` skeleton output with live RPC wiring
- [x] Add integration tests for `thread read` command using memory transport
- [x] Add/adjust tests for malformed `thread/read` payload handling
- [x] Run `go test ./...` for Phase 1
- [x] Commit Phase 1 changes
- [x] Record Phase 1 diary/changelog updates

## Phase 2: State Projection Store

- [x] Create `pkg/state/models.go` with thread/turn/item projection models
- [x] Create `pkg/state/store.go` with bounded in-memory store + query APIs
- [x] Create `pkg/state/projector.go` for notification-to-state updates
- [x] Wire projector with `turn/diff/updated` and `turn/plan/updated` handling
- [x] Add replay-based projector unit tests
- [x] Run `go test ./...` for Phase 2
- [x] Commit Phase 2 changes
- [x] Record Phase 2 diary/changelog updates

## Phase 3: JS Session Wrapper Expansion

- [x] Extend `require("codex")` session with `threads.start/list/read`
- [x] Add thread object helpers: `turn.start`, `turn.steer`, `turn.interrupt`
- [x] Add review helper wrapper: `thread.review.start`
- [x] Add wrapper API smoke script under ticket `scripts/`
- [x] Add wrapper API smoke playbook under ticket `playbooks/`
- [x] Add runtime tests for wrapper API availability and call routing
- [x] Run `go test ./...` for Phase 3
- [x] Commit Phase 3 changes
- [x] Record Phase 3 diary/changelog updates

## Phase 4: Harness Framework Core

- [x] Add `pkg/harness/context.go` with canonical handler context
- [x] Add `pkg/harness/dispatch.go` with notification/request dispatch pipeline
- [x] Add `pkg/harness/compose.go` with deterministic composition order
- [x] Add exactly-once request response guardrails and diagnostics
- [x] Add unit tests for ordering and one-response enforcement
- [x] Run `go test ./...` for Phase 4
- [x] Commit Phase 4 changes
- [x] Record Phase 4 diary/changelog updates

## Phase 5: Built-ins I (Autopilot + Plan Gate)

- [ ] Implement autopilot approvals built-in harness
- [ ] Implement plan-gate built-in harness (`turn/plan/updated`)
- [ ] Add deterministic fixture/replay tests for both built-ins
- [ ] Add ticket scripts/playbooks for built-ins under `scripts/` and `playbooks/`
- [ ] Run `go test ./...` preflight
- [ ] **STOP-GATE:** Explain first built-ins live run command and wait for approval
- [ ] Execute first built-ins live run
- [ ] Commit Phase 5 changes
- [ ] Record Phase 5 diary/changelog updates

## Phase 6: Built-ins II (TDD/Review/Compaction)

- [ ] Implement tdd-loop built-in harness
- [ ] Implement review-gate built-in harness
- [ ] Implement token-usage compaction built-in harness
- [ ] Add deterministic tests for all Phase 6 built-ins
- [ ] Run `go test ./...` preflight
- [ ] **STOP-GATE:** Explain Phase 6 live run command and wait for approval
- [ ] Execute Phase 6 live run
- [ ] Commit Phase 6 changes
- [ ] Record Phase 6 diary/changelog updates

## Phase 7: Reliability Hardening

- [ ] Add retry/backoff helper for retryable RPC/transport failures
- [ ] Add bounded event buffer/retention controls
- [ ] Add notification opt-out capability in initialize params
- [ ] Add fault-injection tests for retry and buffering behavior
- [ ] Run `go test ./...` for Phase 7
- [ ] Commit Phase 7 changes
- [ ] Record Phase 7 diary/changelog updates

## Phase 8: CLI and Docs Completion

- [ ] Add CLI state-inspection commands that use projected state
- [ ] Add Glazed help pages and troubleshooting docs
- [ ] Add final examples and usage docs
- [ ] Run full `go test ./...`
- [ ] Run `docmgr doctor --ticket OAP-02-JS-HARNESS-CONTINUATION --stale-after 30`
- [ ] **STOP-GATE:** Explain final full live smoke run command and wait for approval
- [ ] Execute final full live smoke run
- [ ] Publish final docs/report to reMarkable
- [ ] Commit final Phase 8 changes
- [ ] Record final diary/changelog updates and close ticket
