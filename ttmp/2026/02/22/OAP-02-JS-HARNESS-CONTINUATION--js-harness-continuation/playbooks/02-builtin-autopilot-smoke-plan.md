---
Title: Built-in Autopilot Smoke Plan
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
    - Path: openai-app-server/ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/02-builtin-autopilot-smoke.js
      Note: Phase 5 autopilot behavior smoke script
ExternalSources: []
Summary: Live smoke plan for autopilot-like approval behavior in OAP-02.
LastUpdated: 2026-02-22T23:02:00-05:00
WhatFor: Validate approval decision behavior and request callback plumbing in a built-in style scenario.
WhenToUse: Use only after explicit user approval at the Phase 5 stop-gate.
---

# Built-in Autopilot Smoke Plan

## Purpose

Exercise command approval routing with a deterministic policy and verify completion markers.

## Command

```bash
go run ./cmd/openai-app-server harness run \
  --script ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/02-builtin-autopilot-smoke.js \
  --transport stdio \
  --stdio-command codex \
  --stdio-args "app-server --listen stdio://" \
  --timeout-ms 240000 \
  --settle-ms 1000 \
  --wait-for-ui-type autopilot-smoke-complete \
  --wait-for-ui-timeout-ms 200000 \
  --fail-on-wait-ui-ok-false
```

## Expected Output Checks

- `autopilot-smoke-start`
- `autopilot-smoke-thread-started`
- `autopilot-smoke-turn-started`
- one or more `autopilot-smoke-decision` events when approval requests occur
- final `autopilot-smoke-complete` with `ok:true`

## Failure Signals

- timeout waiting for completion marker
- completion marker has `ok:false`
- request callbacks occur but no decision events emitted
