---
Title: Module API Thread Turn Live Run Plan
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
    - Path: openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/04-module-api-thread-turn-live.js
      Note: Canonical live script for thread/start + turn/start module API validation
    - Path: openai-app-server/pkg/js/module_rpc.go
      Note: Request/notification module API used by script
    - Path: openai-app-server/pkg/js/module_ui.go
      Note: ui.emit bridge surface used for assertions
    - Path: openai-app-server/pkg/js/module_clock.go
      Note: settle timing used to collect notifications before completion marker
ExternalSources: []
Summary: "Live validation plan for module-API thread/start + turn/start flow against real app-server."
LastUpdated: 2026-02-23T04:36:00-05:00
WhatFor: "Validate live thread and turn lifecycle path after module-API migration."
WhenToUse: "Use before running the next real harness test that exercises thread/start and turn/start."
---

# Module API Thread/Turn Live Run Plan

## Purpose

Exercise a full minimal thread and turn lifecycle via module imports:
- `rpc.request("thread/start", ...)`
- `rpc.request("turn/start", ...)`
- notification fanout via `session.onNotification(...)`
- result markers via `ui.emit(...)`

## Preflight

```bash
cd /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server
go test ./...
test -f ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/04-module-api-thread-turn-live.js
```

## Command (STOP-GATE approval required before execution)

```bash
go run ./cmd/openai-app-server harness run \
  --script ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/04-module-api-thread-turn-live.js \
  --transport stdio \
  --stdio-command codex \
  --stdio-args "app-server --listen stdio://" \
  --timeout-ms 90000 \
  --settle-ms 6000
```

## Expected Output Checks

- `ui.emit` contains `type:"module-api-thread-start"` with `ok:true` and non-empty `threadId`.
- `ui.emit` contains `type:"module-api-turn-start"` with `ok:true` and non-empty `turnId` (or at least turn payload).
- one or more `ui.emit` lines appear with `type:"module-api-thread-turn-notification"`.
- final `ui.emit` contains `type:"module-api-thread-turn-complete"` with `ok:true`.
- process prints `harness.run completed`.

## Failure Signals

- Completion marker with `ok:false`.
- Missing `threadId` in `thread/start` response.
- transport/handshake failure or process launch failure.
- no notifications observed during settle window.
