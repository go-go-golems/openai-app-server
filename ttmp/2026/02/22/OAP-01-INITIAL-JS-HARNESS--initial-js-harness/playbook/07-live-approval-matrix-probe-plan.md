---
Title: Live Approval Matrix Probe Plan
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
    - Path: openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/08-live-approval-matrix-probe.js
      Note: Matrix probe script for approvalPolicy/sandbox combinations
    - Path: openai-app-server/pkg/js/module_rpc.go
      Note: Request callback response path exercised when inbound requests are observed
ExternalSources: []
Summary: "Live matrix probe comparing approval/sandbox combinations for inbound request visibility."
LastUpdated: 2026-02-23T05:55:00-05:00
WhatFor: "Determine if inbound request callbacks appear under different approval policy combinations."
WhenToUse: "Use after escalation probe baseline is recorded."
---

# Live Approval Matrix Probe Plan

## Purpose

Run one live harness script that executes multiple `thread/start` + `turn/start` cases with varying `approvalPolicy` values while keeping sandbox stable, then report whether inbound request callbacks were observed in each case.

## Cases Included

- `on-request_workspace-write`
- `on-failure_workspace-write`
- `never_workspace-write`

All cases run the same turn input:

- `Run curl -I https://example.com and return one short sentence with the HTTP status line.`

## Preflight

```bash
cd /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server
go test ./...
test -f ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/08-live-approval-matrix-probe.js
```

## Command

```bash
go run ./cmd/openai-app-server harness run \
  --script ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/08-live-approval-matrix-probe.js \
  --transport stdio \
  --stdio-command codex \
  --stdio-args "app-server --listen stdio://" \
  --timeout-ms 220000 \
  --settle-ms 1000 \
  --wait-for-ui-type approval-matrix-probe-complete \
  --wait-for-ui-timeout-ms 180000 \
  --fail-on-wait-ui-ok-false
```

## Expected Output Checks

- Each case emits `approval-matrix-case-start` and `approval-matrix-case-complete`.
- Final `approval-matrix-probe-complete` appears and has `ok:true`.
- Final payload includes `results` entries for each case.
- `sawAnyRequest` may be `true` or `false`; this is a measured outcome.

## Failure Signals

- timeout waiting for completion marker.
- final completion event with `ok:false`.
- any case result with `status:"case_error"`.
