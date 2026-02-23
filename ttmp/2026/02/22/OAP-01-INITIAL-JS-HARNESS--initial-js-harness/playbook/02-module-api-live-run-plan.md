---
Title: Module API Live Harness Run Plan
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
    - Path: openai-app-server/pkg/js/module_ui.go
      Note: `require("ui")` module used by smoke script
    - Path: openai-app-server/pkg/js/module_rpc.go
      Note: `require("rpc")` module used for live request path
    - Path: openai-app-server/pkg/js/module_clock.go
      Note: `require("clock")` module used for async settle behavior
    - Path: openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/03-module-api-live-smoke.js
      Note: Canonical post-migration live-run script
ExternalSources: []
Summary: "Post-migration live harness plan validating module API (`ui/rpc/clock`) against a real app-server process."
LastUpdated: 2026-02-23T04:20:00-05:00
WhatFor: "Validate module-based JS runtime API in a real harness run after `__host` removal."
WhenToUse: "Use before running the first post-migration real harness smoke command."
---

# Module API Live Harness Run Plan

## Purpose

Run a real harness smoke test that validates `require("ui")`, `require("rpc")`, and `require("clock")` in one script against a live app-server transport.

## Preflight

```bash
cd /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server
go test ./...
test -f ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/03-module-api-live-smoke.js
```

## Command (STOP-GATE approval required before execution)

```bash
go run ./cmd/openai-app-server harness run \
  --script ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/03-module-api-live-smoke.js \
  --transport stdio \
  --stdio-command codex \
  --stdio-args "app-server --listen stdio://" \
  --timeout-ms 60000 \
  --settle-ms 2000
```

## Expected Output Checks

- Command prints `harness.run` header lines and ends with `harness.run completed`.
- At least one `ui.emit` line includes `"type":"module-api-thread-list"` and `"ok":true`.
- A completion event appears as `ui.emit` with `"type":"module-api-smoke-complete"` and `"ok":true`.
- No runtime error line appears with `"type":"module-api-smoke-complete"` and `"ok":false`.

## Failure Signals

- `unsupported transport` or child process launch errors.
- Handshake failure (`initialize`/`initialized` path errors).
- JS module load failures for `require("ui"|"rpc"|"clock")`.
- Completion event with `ok:false`.
