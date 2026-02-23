---
Title: Live Sandbox Variant Probe Plan
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
    - Path: openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/09-live-sandbox-variant-probe.js
      Note: Sandbox-variant probe script with reduced notification verbosity
    - Path: openai-app-server/pkg/js/module_approval.go
      Note: `approval.acceptForSession` helper used for approval request responses
ExternalSources: []
Summary: "Live probe comparing sandbox variants under fixed approval policy to inspect inbound request visibility and approval response correctness."
LastUpdated: 2026-02-23T06:20:00-05:00
WhatFor: "Measure how sandbox settings affect request callback surfacing and validate approval response payloads."
WhenToUse: "Use after approval-policy matrix baseline is recorded."
---

# Live Sandbox Variant Probe Plan

## Purpose

Run one live harness script that executes multiple `thread/start` + `turn/start` cases while keeping `approvalPolicy` fixed to `on-request` and varying sandbox values.

## Cases Included

- `on-request_workspace-write`
- `on-request_read-only`
- `on-request_danger-full-access`

All cases use the same turn input:

- `Run curl -I https://example.com and return one short sentence with the HTTP status line.`

## Output Filtering

The script suppresses high-volume reasoning-delta notifications and emits only lifecycle/command-interesting notification types plus request/response events, so run output stays reviewable.

## Preflight

```bash
cd /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server
go test ./...
test -f ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/09-live-sandbox-variant-probe.js
```

## Command

```bash
go run ./cmd/openai-app-server harness run \
  --script ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/09-live-sandbox-variant-probe.js \
  --transport stdio \
  --stdio-command codex \
  --stdio-args "app-server --listen stdio://" \
  --timeout-ms 220000 \
  --settle-ms 1000 \
  --wait-for-ui-type sandbox-variant-probe-complete \
  --wait-for-ui-timeout-ms 180000 \
  --fail-on-wait-ui-ok-false
```

## Expected Output Checks

- Each case emits `sandbox-variant-case-start` and `sandbox-variant-case-complete`.
- Final marker `sandbox-variant-probe-complete` appears and has `ok:true`.
- Final payload includes per-case `results` with request and notification deltas.
- No stderr decode error line containing `failed to deserialize CommandExecutionRequestApprovalResponse`.

## Failure Signals

- timeout before final completion marker.
- final completion event with `ok:false`.
- any case result with `status:"case_error"`.
- stderr contains `failed to deserialize CommandExecutionRequestApprovalResponse` (known compatibility issue to resolve in current phase).
