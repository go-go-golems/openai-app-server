---
Title: CLI Help and Troubleshooting
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
    - Path: openai-app-server/cmd/openai-app-server/harness_run_command.go
      Note: Primary harness run command and wait-gate behavior
    - Path: openai-app-server/cmd/openai-app-server/harness_state_replay_command.go
      Note: Projected-state replay command
    - Path: openai-app-server/pkg/codexrpc/retry.go
      Note: Request retry helper for retryable RPC failures
ExternalSources: []
Summary: Copy/paste CLI help and troubleshooting guide for common harness workflows.
LastUpdated: 2026-02-22T23:17:00-05:00
WhatFor: Provide fast operator guidance for harness run, thread commands, state replay, and common failures.
WhenToUse: Use when running commands manually or diagnosing command/runtime failures.
---

# CLI Help and Troubleshooting

## Goal

Give a concise, copy/paste-friendly runbook for common CLI tasks and failure modes.

## Context

These commands assume repository root:

- `/home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server`

## Quick Reference

### Core command groups

- `openai-app-server harness run`
- `openai-app-server harness state-replay`
- `openai-app-server thread list`
- `openai-app-server thread read`

### Key flags

- `--wait-for-ui-type`: block until matching `ui.emit` event type arrives
- `--wait-for-ui-timeout-ms`: wait timeout for completion marker
- `--fail-on-wait-ui-ok-false`: fail immediately if completion marker has `ok:false`
- `--opt-out-notification-methods`: comma-separated list sent as `initialize.capabilities.optOutNotificationMethods`

### Troubleshooting table

| Symptom | Likely cause | Action |
|---|---|---|
| timeout waiting for completion marker | script never emitted expected `type` | verify `ui.emit({type:"..."})` and wait flag match |
| `--script` error | wrong path or missing file | run `test -f <script>` first |
| handshake errors (`handshake required`, `initialized out of order`) | client/connect sequence broken | verify transport setup and initialize flow |
| response decode errors | payload shape mismatch | check `pkg/js/module_approval.go` and protocol schema |
| noisy output during live run | high-volume notifications | use `--opt-out-notification-methods` and filtered script emits |
| repeated run failures after overload | server overloaded (`-32001`) | use retry helper path (`RequestWithRetry`) where applicable |

## Usage Examples

### List threads

```bash
go run ./cmd/openai-app-server thread list \
  --transport stdio \
  --stdio-command codex \
  --stdio-args "app-server --listen stdio://"
```

### Read one thread

```bash
go run ./cmd/openai-app-server thread read \
  --thread-id <THREAD_ID> \
  --include-turns \
  --transport stdio \
  --stdio-command codex \
  --stdio-args "app-server --listen stdio://"
```

### Replay state from JSON events

```bash
go run ./cmd/openai-app-server harness state-replay \
  --events-file ./events.json
```

### Run harness script with deterministic completion gate

```bash
go run ./cmd/openai-app-server harness run \
  --script ttmp/.../scripts/<script>.js \
  --transport stdio \
  --stdio-command codex \
  --stdio-args "app-server --listen stdio://" \
  --wait-for-ui-type <completion-type> \
  --wait-for-ui-timeout-ms 180000 \
  --fail-on-wait-ui-ok-false
```
