---
Title: Live Escalation Request Probe Plan
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
    - Path: openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/07-live-escalation-request-probe.js
      Note: Probe script that attempts to provoke escalation/inbound request behavior via network command
    - Path: openai-app-server/pkg/js/module_approval.go
      Note: Approval decision helpers used to answer approval requests with valid enum values
ExternalSources: []
Summary: "Follow-up live probe attempting to provoke inbound request flow via network command task."
LastUpdated: 2026-02-23T05:20:00-05:00
WhatFor: "Explore whether escalation-prone prompts surface inbound request events to harness callbacks."
WhenToUse: "Use after baseline inbound probe results are recorded."
---

# Live Escalation Request Probe Plan

## Purpose

Run a live scenario that asks the model to execute a network command, which may increase the chance of approval/escalation request events; capture whether requests are surfaced to harness `onRequest` callbacks.

## Preflight

```bash
cd /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server
go test ./...
test -f ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/07-live-escalation-request-probe.js
```

## Command

```bash
go run ./cmd/openai-app-server harness run \
  --script ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/07-live-escalation-request-probe.js \
  --transport stdio \
  --stdio-command codex \
  --stdio-args "app-server --listen stdio://" \
  --timeout-ms 120000 \
  --settle-ms 1000 \
  --wait-for-ui-type escalation-probe-complete \
  --wait-for-ui-timeout-ms 90000 \
  --fail-on-wait-ui-ok-false
```

## Expected Output Checks

- `escalation-probe-thread-start` and `escalation-probe-turn-start` are `ok:true`.
- final `escalation-probe-complete` has `ok:true`.
- `probeStatus` indicates either:
  - `request_observed_and_responded`, or
  - `no_inbound_request_observed`.

## Failure Signals

- timeout before completion marker.
- completion payload with `ok:false` or `probeStatus:"script_error"`.
