---
Title: Final Full Smoke Plan
Ticket: OAP-02-JS-HARNESS-CONTINUATION
Status: active
Topics:
    - goja
    - glazed
    - openai-app-server
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles:
    - Path: openai-app-server/ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/05-final-full-smoke.js
      Note: Final comprehensive smoke script
ExternalSources: []
Summary: Final end-to-end live smoke plan before ticket closure.
LastUpdated: 2026-02-22T23:21:00-05:00
WhatFor: Validate wrapper API, approval handling, turn completion observation, and thread-read check in one run.
WhenToUse: Use at final stop-gate with explicit user approval.
---

# Final Full Smoke Plan

## Purpose

Execute one end-to-end live validation run that covers:

- wrapper API thread/turn calls
- approval request handling path
- turn completion observation
- `threads.read(..., true)` check

## Command

```bash
go run ./cmd/openai-app-server harness run \
  --script ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/05-final-full-smoke.js \
  --transport stdio \
  --stdio-command codex \
  --stdio-args "app-server --listen stdio://" \
  --timeout-ms 300000 \
  --settle-ms 1000 \
  --wait-for-ui-type final-full-smoke-complete \
  --wait-for-ui-timeout-ms 240000 \
  --fail-on-wait-ui-ok-false
```

## Expected Output Checks

- `final-full-smoke-start`
- `final-full-smoke-thread-started`
- `final-full-smoke-turn-started`
- one or more `final-full-smoke-approval` events when approval requests surface
- `final-full-smoke-thread-read` with `ok:true`
- final `final-full-smoke-complete` with `ok:true`

## Failure Signals

- timeout waiting for completion marker
- `final-full-smoke-complete` has `ok:false`
- `summary.errors` non-empty
- `summary.threadReadWorked:false`
