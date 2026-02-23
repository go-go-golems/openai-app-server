---
Title: Wrapper API Smoke Plan
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
    - Path: openai-app-server/ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/01-wrapper-api-smoke.js
      Note: Phase 3 wrapper API smoke script
    - Path: openai-app-server/pkg/js/module_codex.go
      Note: Session/thread wrapper implementation under test
ExternalSources: []
Summary: Smoke plan for validating new session/thread wrapper APIs through harness run.
LastUpdated: 2026-02-22T22:52:00-05:00
WhatFor: Validate wrapper routing behavior against real transport once Phase 3 is complete.
WhenToUse: Use after Phase 3 test preflight and explicit user approval.
---

# Wrapper API Smoke Plan

## Purpose

Validate that `session.threads.start` + `session.thread(id).turn.start` wrappers execute correctly end-to-end in a live run.

## Preflight

```bash
cd /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server
go test ./...
test -f ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/01-wrapper-api-smoke.js
```

## Command

```bash
go run ./cmd/openai-app-server harness run \
  --script ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/01-wrapper-api-smoke.js \
  --transport stdio \
  --stdio-command codex \
  --stdio-args "app-server --listen stdio://" \
  --timeout-ms 220000 \
  --settle-ms 1000 \
  --wait-for-ui-type wrapper-smoke-complete \
  --wait-for-ui-timeout-ms 180000 \
  --fail-on-wait-ui-ok-false
```

## Expected Output Checks

- `wrapper-smoke-start` appears.
- `wrapper-smoke-thread-started` appears with non-empty `threadId`.
- `wrapper-smoke-turn-started` appears.
- final `wrapper-smoke-complete` appears with `ok:true`.

## Failure Signals

- timeout waiting for `wrapper-smoke-complete`.
- completion event with `ok:false`.
- script error indicating missing thread id or wrapper method failures.
