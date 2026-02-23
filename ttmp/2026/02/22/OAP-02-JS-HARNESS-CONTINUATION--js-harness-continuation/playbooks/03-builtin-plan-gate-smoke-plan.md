---
Title: Built-in Plan Gate Smoke Plan
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
    - Path: openai-app-server/ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/03-builtin-plan-gate-smoke.js
      Note: Phase 5 plan-gate behavior smoke script
ExternalSources: []
Summary: Live smoke plan for plan-gating behavior using `turn/plan/updated` and `turn/steer`.
LastUpdated: 2026-02-22T23:02:00-05:00
WhatFor: Validate gating event handling and steering action in a built-in style scenario.
WhenToUse: Use only after explicit user approval at the Phase 5 stop-gate.
---

# Built-in Plan Gate Smoke Plan

## Purpose

Verify that plan update notifications are detected and converted into a single steer action.

## Command

```bash
go run ./cmd/openai-app-server harness run \
  --script ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/03-builtin-plan-gate-smoke.js \
  --transport stdio \
  --stdio-command codex \
  --stdio-args "app-server --listen stdio://" \
  --timeout-ms 240000 \
  --settle-ms 1000 \
  --wait-for-ui-type plan-gate-smoke-complete \
  --wait-for-ui-timeout-ms 200000 \
  --fail-on-wait-ui-ok-false
```

## Expected Output Checks

- `plan-gate-smoke-start`
- `plan-gate-smoke-thread-started`
- `plan-gate-smoke-turn-started`
- `plan-gate-smoke-plan` when plan notification is seen
- `plan-gate-smoke-steer-sent`
- final `plan-gate-smoke-complete` with `ok:true`

## Failure Signals

- timeout waiting for completion marker
- completion marker has `ok:false`
- plan was observed but steer send failed (`plan-gate-smoke-steer-error`)
