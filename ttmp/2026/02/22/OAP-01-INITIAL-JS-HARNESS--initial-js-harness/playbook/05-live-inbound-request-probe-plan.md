---
Title: Live Inbound Request Probe Plan
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
    - Path: openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/06-live-inbound-request-probe.js
      Note: Probe script that attempts to trigger and handle inbound request events
    - Path: openai-app-server/cmd/openai-app-server/harness_run_command.go
      Note: Wait-for-ui gate behavior used to deterministically capture probe completion marker
    - Path: openai-app-server/pkg/js/module_rpc.go
      Note: `rpc.respond` implementation exercised by probe
ExternalSources: []
Summary: "Live probe for inbound request handling behavior under on-request policy."
LastUpdated: 2026-02-23T05:07:00-05:00
WhatFor: "Determine whether current runtime/session path yields inbound request events and validate response plumbing against real server behavior."
WhenToUse: "Use after request/response plumbing is implemented and before committing to approval harness semantics."
---

# Live Inbound Request Probe Plan

## Purpose

Attempt to trigger at least one inbound request event (for example an approval-style request) in a real run, respond via `rpc.respond`, and record whether this event class is currently observable in this integration mode.

## Preflight

```bash
cd /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server
go test ./...
test -f ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/06-live-inbound-request-probe.js
```

## Command

```bash
go run ./cmd/openai-app-server harness run \
  --script ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/06-live-inbound-request-probe.js \
  --transport stdio \
  --stdio-command codex \
  --stdio-args "app-server --listen stdio://" \
  --timeout-ms 120000 \
  --settle-ms 1000 \
  --wait-for-ui-type inbound-request-probe-complete \
  --wait-for-ui-timeout-ms 90000 \
  --fail-on-wait-ui-ok-false
```

## Expected Output Checks

- `ui.emit` includes `type:"inbound-request-probe-thread-start"` with `ok:true`.
- `ui.emit` includes `type:"inbound-request-probe-turn-start"` with `ok:true`.
- If inbound requests are observable in this mode:
  - one or more `inbound-request-probe-request` events
  - one or more `inbound-request-probe-response` with `ok:true`
  - final `inbound-request-probe-complete` with `ok:true` and `probeStatus:"request_observed_and_responded"`.
- If inbound requests are not observed:
  - final `inbound-request-probe-complete` will still be `ok:true` with `probeStatus:"no_inbound_request_observed"`, which should be recorded as an integration observation.

## Failure Signals

- command timeout before completion marker.
- `inbound-request-probe-complete` with `ok:false` (script/runtime failure).
- `rpc.respond` throws in probe response path.
