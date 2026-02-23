---
Title: Module API Turn Completed Gate Plan
Ticket: OAP-01-INITIAL-JS-HARNESS
Status: active
Topics:
    - goja
    - glazed
    - openai-app-server
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles:
    - Path: openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/05-module-api-turn-completed-gate.js
      Note: Deterministic live script waiting for matching turn/completed
    - Path: openai-app-server/pkg/js/module_rpc.go
      Note: Module API request + notification path used by script
    - Path: openai-app-server/pkg/js/module_clock.go
      Note: Poll/sleep timeout loop used for deterministic wait
ExternalSources: []
Summary: "Deterministic live run plan that only succeeds after matching turn/completed is observed."
LastUpdated: 2026-02-23T04:56:00-05:00
WhatFor: "Validate turn lifecycle completion gate using module API under live transport."
WhenToUse: "Use for the next real run after phase-8 variable settle behavior."
---

# Module API Turn Completed Gate Plan

## Purpose

Run a live script that does not declare success until a matching `turn/completed` notification is observed for the started turn.

## Preflight

```bash
cd /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server
go test ./...
test -f ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/05-module-api-turn-completed-gate.js
```

## Command (STOP-GATE approval required before execution)

```bash
go run ./cmd/openai-app-server harness run \
  --script ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/05-module-api-turn-completed-gate.js \
  --transport stdio \
  --stdio-command codex \
  --stdio-args "app-server --listen stdio://" \
  --timeout-ms 90000 \
  --settle-ms 6000
```

## Expected Output Checks

- `ui.emit` includes `type:"module-api-turn-gate-thread-start"` with `ok:true`.
- `ui.emit` includes `type:"module-api-turn-gate-turn-start"` with `ok:true`.
- at least one `ui.emit` contains `method:"turn/completed"` in `type:"module-api-turn-gate-notification"`.
- final `ui.emit` includes `type:"module-api-turn-gate-complete"` with `ok:true` and `turnCompleted:true`.
- final process line is `harness.run completed`.

## Failure Signals

- final `module-api-turn-gate-complete` has `ok:false`.
- timeout error waiting for `turn/completed`.
- missing thread or turn id extraction in start responses.
