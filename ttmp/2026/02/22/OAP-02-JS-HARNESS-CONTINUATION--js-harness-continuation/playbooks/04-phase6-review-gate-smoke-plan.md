---
Title: Phase 6 Review Gate Smoke Plan
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
    - Path: openai-app-server/ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/04-phase6-review-gate-smoke.js
      Note: Phase 6 review-gate smoke script
ExternalSources: []
Summary: Live smoke plan for review-gate style behavior in Phase 6.
LastUpdated: 2026-02-22T23:08:00-05:00
WhatFor: Validate turn-completed -> review/start -> followup-turn flow through wrapper APIs.
WhenToUse: Use after Phase 6 preflight and explicit stop-gate approval.
---

# Phase 6 Review Gate Smoke Plan

## Purpose

Validate that we can react to a completed turn, trigger `review/start`, and launch a follow-up turn from script logic.

## Command

```bash
go run ./cmd/openai-app-server harness run \
  --script ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/04-phase6-review-gate-smoke.js \
  --transport stdio \
  --stdio-command codex \
  --stdio-args "app-server --listen stdio://" \
  --timeout-ms 260000 \
  --settle-ms 1000 \
  --wait-for-ui-type phase6-review-gate-smoke-complete \
  --wait-for-ui-timeout-ms 220000 \
  --fail-on-wait-ui-ok-false
```

## Expected Output Checks

- `phase6-review-gate-smoke-start`
- `phase6-review-gate-thread-started`
- `phase6-review-gate-turn-started`
- `phase6-review-gate-turn-completed`
- `phase6-review-gate-review-started`
- final `phase6-review-gate-smoke-complete` with `ok:true`

## Failure Signals

- timeout waiting for final completion marker
- completion marker has `ok:false`
- review trigger error marker (`phase6-review-gate-review-error`)
