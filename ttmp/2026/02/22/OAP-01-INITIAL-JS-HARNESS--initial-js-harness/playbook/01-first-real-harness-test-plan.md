---
Title: First Real Harness Test Plan
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
    - Path: openai-app-server/cmd/openai-app-server/harness_run_command.go
      Note: Harness run command entrypoint that will be exercised
    - Path: openai-app-server/pkg/codexrpc/client.go
      Note: Handshake and request routing logic used during first live run
    - Path: openai-app-server/pkg/js/runtime.go
      Note: JS runtime host and callback paths used by harness scripts
ExternalSources: []
Summary: "Pre-flight plan for first live harness run; explicitly prepared but not executed yet."
LastUpdated: 2026-02-23T03:20:00-05:00
WhatFor: "Provide deterministic steps for the first real harness execution after explicit user confirmation."
WhenToUse: "Use this exactly when proceeding past the stop-gate for the first live harness test."
---

# First Real Harness Test Plan

## Purpose

Run the first real harness test against a live app-server process to validate end-to-end behavior: handshake, event flow, and at least one command-level interaction path.

## Environment Assumptions

- Current branch contains commits through phase 4 (`codexrpc` + `pkg/js` runtime skeleton).
- `openai-app-server` builds and tests cleanly (`go test ./...`).
- A runnable app-server command is available in PATH or via explicit command flag.
- Network/auth prerequisites for the selected model are configured if required by the target app-server.

## Commands

1. Build and smoke check local CLI binary:

```bash
cd /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server
go test ./...
go run ./cmd/openai-app-server --help
```

2. Create a minimal harness script fixture (example path):

```bash
cat > /tmp/oap-first-harness.js <<'JS'
const codex = require("codex");
const session = codex.connect();
session.onNotification((evt) => {
  console.log("notification", JSON.stringify(evt));
});
JS
```

3. Launch first real harness run (STOP-GATE approval required before this command):

```bash
go run ./cmd/openai-app-server harness run \
  --script /tmp/oap-first-harness.js \
  --transport stdio \
  --stdio-command codex-app-server \
  --stdio-args "stdio-jsonl"
```

4. Capture and inspect output for handshake + event markers.

## Exit Criteria

- `initialize` and `initialized` handshake completes without protocol errors.
- Harness script loads with `require("codex")` successfully.
- At least one notification callback is observed in command output/logs.
- Command exits cleanly or remains running in expected interactive mode without panic.

## Failure Modes

- `unsupported transport` or process launch errors from stdio command.
- handshake errors (`ErrHandshakeRequired`, duplicate initialize, malformed response).
- JS runtime panic due to callback/threading issue.
- missing module error for `require("codex")`.

## Post-Run Follow-up

- Record exact command, outputs, and errors into `reference/01-diary.md` (new step).
- Update `tasks.md` to mark real-harness execution task status.
- Add changelog entry with run outcome and next fixes.

