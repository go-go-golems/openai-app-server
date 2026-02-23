---
Title: Usage Examples
Ticket: OAP-02-JS-HARNESS-CONTINUATION
Status: active
Topics:
    - goja
    - glazed
    - openai-app-server
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: openai-app-server/ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/01-wrapper-api-smoke.js
      Note: Wrapper API smoke example
    - Path: openai-app-server/ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/02-builtin-autopilot-smoke.js
      Note: Built-in autopilot example
    - Path: openai-app-server/ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/04-phase6-review-gate-smoke.js
      Note: Review-gate flow example
ExternalSources: []
Summary: Practical script/CLI examples for wrapper, built-in, and projected-state workflows.
LastUpdated: 2026-02-22T23:17:00-05:00
WhatFor: Provide copy/paste examples for common implementation and validation patterns.
WhenToUse: Use when authoring new scripts, running smoke checks, or validating projected state.
---

# Usage Examples

## Goal

Show concrete command + script patterns for the main OAP-02 workflows.

## Context

Examples assume the `openai-app-server` repository root and live stdio transport.

## Quick Reference

### Example 1: Wrapper API smoke

Script:

- `ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/01-wrapper-api-smoke.js`

Run:

```bash
go run ./cmd/openai-app-server harness run \
  --script ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/01-wrapper-api-smoke.js \
  --transport stdio \
  --stdio-command codex \
  --stdio-args "app-server --listen stdio://" \
  --wait-for-ui-type wrapper-smoke-complete \
  --wait-for-ui-timeout-ms 180000 \
  --fail-on-wait-ui-ok-false
```

### Example 2: Built-in autopilot smoke

Script:

- `ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/02-builtin-autopilot-smoke.js`

Run:

```bash
go run ./cmd/openai-app-server harness run \
  --script ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/02-builtin-autopilot-smoke.js \
  --transport stdio \
  --stdio-command codex \
  --stdio-args "app-server --listen stdio://" \
  --wait-for-ui-type autopilot-smoke-complete \
  --wait-for-ui-timeout-ms 200000 \
  --fail-on-wait-ui-ok-false
```

### Example 3: Review-gate smoke

Script:

- `ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/04-phase6-review-gate-smoke.js`

Run:

```bash
go run ./cmd/openai-app-server harness run \
  --script ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/04-phase6-review-gate-smoke.js \
  --transport stdio \
  --stdio-command codex \
  --stdio-args "app-server --listen stdio://" \
  --wait-for-ui-type phase6-review-gate-smoke-complete \
  --wait-for-ui-timeout-ms 220000 \
  --fail-on-wait-ui-ok-false
```

### Example 4: Replay projected state from events

`events.json` shape:

```json
[
  {"method":"thread/started","params":{"thread":{"id":"thread-1"}}},
  {"method":"turn/started","params":{"threadId":"thread-1","turn":{"id":"turn-1"}}}
]
```

Run replay:

```bash
go run ./cmd/openai-app-server harness state-replay \
  --events-file ./events.json
```

## Usage Examples

### Reduce notification noise for live runs

```bash
go run ./cmd/openai-app-server harness run \
  --script ttmp/.../scripts/01-wrapper-api-smoke.js \
  --transport stdio \
  --stdio-command codex \
  --stdio-args "app-server --listen stdio://" \
  --opt-out-notification-methods "item/agentMessage/delta,codex/event/reasoning_content_delta" \
  --wait-for-ui-type wrapper-smoke-complete \
  --fail-on-wait-ui-ok-false
```
