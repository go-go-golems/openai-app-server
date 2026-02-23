# Changelog

## 2026-02-23

- Created OAP-04 design document `design/01-improved-js-harness-api-design.md` with:
  - full OAP-01/OAP-02 script audit findings
  - proposed improved JS API signatures (session/thread/turn/events/approvals/ids)
  - detailed before/after rewrite for `scripts/05-final-full-smoke.js`
  - large adapted examples (autopilot, plan-gate, tdd-loop, review-gate, recursive decomposition, auto-compaction)
  - migration strategy, testing strategy, and acceptance criteria
- Added detailed implementation diary in `reference/01-diary.md` capturing ticket bootstrap, audit workflow, findings, and design rationale.
- Uploaded design document to reMarkable as `OAP-04 Improved JS Harness API Design` under `/ai/2026/02/23/OAP-04-IMPROVED-JS-API`.
- Expanded design document with implementation-level engineering details:
  - file-by-file change blueprint
  - runtime waiter/policy algorithms
  - contract edge cases
  - phased PR plan and detailed test case checklist
- Added scenario sketch script pack under `scripts/sketches/`:
  - scientific RLM research
  - production incident triage
  - release readiness gate
  - security vulnerability triage
  - customer support escalation
- Added Phase 1/2 helper implementations in `pkg/js`:
  - `session.ids.thread/turn`
  - `session.waitFor(...)`
  - `thread.turn.waitCompleted(...)`
  - `session.events.metrics()`
  - `session.approvals.setPolicy(...)` and `session.approvals.respond(...)`
- Added runtime tests in `pkg/js/runtime_test.go` covering helper availability, waiter behavior, metrics, and approval policy responses.
- Expanded OAP-04 `tasks.md` with phased implementation checklist and checked off Phase 1/2 helper items.
- Validation passed:
  - `go test ./pkg/js -count=1`
  - `go test ./... -count=1`

## 2026-02-22

- Initial workspace created
