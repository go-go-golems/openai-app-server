# Tasks

## Phase 0: Audit and Design (Completed)

- [x] Create OAP-04 ticket workspace and baseline docs
- [x] Audit OAP-01 scripts for recurring JS API friction patterns
- [x] Audit OAP-02 scripts for continuation-level API ergonomics gaps
- [x] Extract larger real-life harness examples from imported source doc
- [x] Author detailed improved JS API design with signatures and extensive examples
- [x] Maintain detailed diary entries covering audit and design steps
- [x] Upload OAP-04 design document to reMarkable
- [x] Expand design doc with file-level implementation details and concrete runtime/test blueprint
- [x] Add scenario sketch scripts under `scripts/sketches/` (research, incident, release, security, support)
- [x] Expand design with intern-oriented onboarding/resumability sections and extend sketch pack with additional checkpointed real-world scenarios

## Phase 1: Core Lifecycle Helpers

- [x] Implement `session.ids.thread(...)` and `session.ids.turn(...)`
- [x] Implement `session.waitFor({ method, timeoutMs, where })`
- [x] Implement `thread.turn.waitCompleted({ turnId?, timeoutMs? })`
- [x] Add runtime tests for identity helpers and lifecycle waiters

## Phase 2: Coverage and Policy Helpers

- [x] Implement `session.events.metrics()` (`countByMethod`, `totalNotifications`, `totalRequests`, `reset`)
- [x] Implement `session.approvals.setPolicy({ command, fileChange, fallback })`
- [x] Implement `session.approvals.respond(reqOrId, decision)`
- [x] Add runtime tests for metrics and policy-driven approval responses

## Phase 3: Canonical Script Migration

- [x] Migrate `scripts/05-final-full-smoke.js` to improved API as canonical v2 reference
- [x] Add diagnostic `ui.emit` instrumentation for branch-miss reasons in canonical smoke
- [x] Run `go test ./...`
- [x] Commit implementation and test updates
- [x] Update OAP-04 diary/changelog with implementation details

## Phase 4: Live Validation Gate

- [x] Explain real run command for canonical v2 smoke and wait for approval
- [x] Execute real harness run for canonical v2 smoke
- [x] Confirm approval-request and turn-completed branches were observed
- [x] Record live run outcome in OAP-04 diary/changelog
