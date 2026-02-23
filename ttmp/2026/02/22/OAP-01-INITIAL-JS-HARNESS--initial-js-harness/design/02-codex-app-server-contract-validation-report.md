---
Title: Codex App Server Contract Validation Report
Ticket: OAP-01-INITIAL-JS-HARNESS
Status: active
Topics:
    - goja
    - openai-app-server
    - codex
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: openai-app-server/pkg/js/module_approval.go
      Note: Approval response helper module under validation
    - Path: openai-app-server/pkg/js/runtime_test.go
      Note: Runtime-level approval response shape tests
    - Path: openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/09-live-sandbox-variant-probe.js
      Note: Live validation script that surfaces request-approval response behavior
    - Path: openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/playbook/08-live-sandbox-variant-probe-plan.md
      Note: Validation playbook and expected output checks
    - Path: openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/design/01-openai-app-server-js-harness-architecture.md
      Note: Primary architecture/plan document updated by this validation
    - Path: openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/tasks.md
      Note: Implementation-phase plan updated with compatibility phase
ExternalSources:
    - https://developers.openai.com/codex/app-server/
    - https://github.com/openai/codex/tree/main/codex-rs/app-server
Summary: "Validation report comparing local codex app-server runtime behavior against official web documentation, with required plan/documentation updates."
LastUpdated: 2026-02-23T07:05:00-05:00
WhatFor: "Track contract drift and required implementation/doc updates for approval request response handling."
WhenToUse: "Read before changing approval response logic or running compatibility validation probes."
---

# Codex App Server Contract Validation Report

## Scope

This report validates the current ticket documentation and implementation plan against:

1. Official Codex App Server web docs.
2. Live runtime observations from the local `codex-cli 0.104.0` app-server process.
3. Current ticket playbooks/scripts/design/tasks.

## Validation Inputs

### Runtime / tooling versions (local)

- `codex --version` -> `codex-cli 0.104.0`
- `codex app-server --help` confirms app-server command surface from same binary.

### Web source reviewed

- Official docs: <https://developers.openai.com/codex/app-server/>
- Approval semantics section reviewed (`item/commandExecution/requestApproval`):
  - docs indicate decision payload can be:
    - `"accept" | "acceptForSession" | "decline" | "cancel"`
    - or `{ "acceptWithExecpolicyAmendment": string[] }`

### Local source reviewed (cloned codex repository)

- Clone path: `/home/manuel/code/others/llms/codex`
- Files inspected:
  - `codex-rs/app-server-protocol/src/protocol/v2.rs`
  - `codex-rs/app-server-protocol/schema/json/CommandExecutionRequestApprovalResponse.json`
  - `codex-rs/app-server/tests/suite/v2/turn_start.rs`
- Confirmed protocol shape:
  - command approval response is an object:
    - `{ "decision": "acceptForSession" }`
  - amendment variant is nested:
    - `{ "decision": { "acceptWithExecpolicyAmendment": { "execpolicy_amendment": [...] } } }`

### Live run evidence (ticket scripts)

- `scripts/09-live-sandbox-variant-probe.js` real runs showed:
  - request callbacks observed for some sandbox variants,
  - but approval response payload decode errors persisted.

## Key Findings

## 1) Documented approval contract and local runtime behavior diverge

### Web docs expectation

Docs show string decision payloads are allowed for command approval responses, plus the amendment object variant.

### Local runtime observation (codex-cli 0.104.0)

Observed errors in live runs:

- Earlier shape: `{"approved":true,"decision":"approve"...}` ->
  - `unknown variant 'approve', expected one of 'accept', 'acceptForSession', 'acceptWithExecpolicyAmendment', 'decline', 'cancel'`
- Current shape after hardening to string decision helper ->
  - `invalid type: string "acceptForSession", expected struct CommandExecutionRequestApprovalResponse`

### Interpretation

This is now confirmed, not inferred: the local app-server protocol expects a structured response object containing a `decision` field. Sending bare strings is rejected by the current runtime despite web docs showing string decisions.

## 2) Approval callback surfacing is sandbox-sensitive in this environment

From sandbox variant probes:

- `workspace-write`: often no request callback observed.
- `read-only` and `danger-full-access`: request callbacks observed (`requestDelta:1` in run results).

This affects reproducibility of approval-path tests and should remain encoded in playbooks/tasks.

## 3) Ticket documentation hygiene gap was fixed

`docmgr doctor` initially reported imported-source issues (missing frontmatter / non-numeric filename) for `sources/local/app-server-js.md`.

Remediation completed:

- Source document normalized to `sources/local/01-app-server-js.md` with frontmatter.
- All ticket references updated to new path.
- `docmgr doctor --ticket OAP-01-INITIAL-JS-HARNESS --stale-after 30` now passes.

## Documentation / Plan Validation Matrix

- `design/01-openai-app-server-js-harness-architecture.md`:
  - status: partially outdated on approval-response assumptions.
  - update required: add explicit compatibility note (docs vs runtime contract drift).
- `playbook/08-live-sandbox-variant-probe-plan.md`:
  - status: updated with decode-error check.
  - update required: add explicit known-failure signature for current runtime and compatibility-run expectation.
- `tasks.md`:
  - status: updated.
  - update required: keep Phase 16 as active compatibility phase and gate next real run.
- `reference/01-diary.md` and `changelog.md`:
  - status: include live-failure observations and response-shape pivots.
  - update required: append this validation pass and web-source findings.

## Required Updates (Actionable)

1. Completed: determine exact struct schema expected by local `CommandExecutionRequestApprovalResponse` from cloned `codex-rs` protocol sources.
2. Completed: update `pkg/js/module_approval.go` to emit structured `decision` objects.
3. Completed: extend runtime tests to validate structured response shape.
4. Pending: run one gated real validation probe (`scripts/09`) and confirm no deserialize errors after the schema-aligned patch.
5. In progress: keep web-doc + local-source references in ticket docs and mark version-sensitivity explicitly.

## Risk Assessment

- High: approval-path reliability remains blocked until response-shape compatibility is solved.
- Medium: docs/runtime drift can recur as Codex versions change.
- Low: overall harness runtime stability (event routing, wait-gating, test suite) remains good.

## Recommendations

1. Treat approval response shape as versioned contract, not static assumption.
2. Add explicit compatibility notes in design + playbooks whenever live behavior differs from web examples.
3. Preserve at least one real probe script (`09`) as canonical regression test for approval handling.

## References

- Official Codex App Server docs:
  - <https://developers.openai.com/codex/app-server/>
- Codex app-server source tree:
  - <https://github.com/openai/codex/tree/main/codex-rs/app-server>
- Local clone protocol definitions:
  - `/home/manuel/code/others/llms/codex/codex-rs/app-server-protocol/src/protocol/v2.rs`
  - `/home/manuel/code/others/llms/codex/codex-rs/app-server-protocol/schema/json/CommandExecutionRequestApprovalResponse.json`
  - `/home/manuel/code/others/llms/codex/codex-rs/app-server/tests/suite/v2/turn_start.rs`
- Ticket-local imported source baseline:
  - `ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/sources/local/01-app-server-js.md`
- Current probe playbook/script:
  - `ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/playbook/08-live-sandbox-variant-probe-plan.md`
  - `ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/09-live-sandbox-variant-probe.js`
