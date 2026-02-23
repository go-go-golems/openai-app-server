---
Title: Diary
Ticket: OAP-04-IMPROVED-JS-API
Status: active
Topics:
    - goja
    - openai-app-server
    - codex
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: openai-app-server/pkg/js/module_codex.go
      Note: |-
        Existing session/thread wrappers evaluated for ergonomic gaps
        Module API baseline analyzed during work
    - Path: openai-app-server/pkg/js/runtime.go
      Note: |-
        Runtime dispatch constraints considered for waiter and policy helper design
        Runtime event callback model analyzed during work
    - Path: openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/05-module-api-turn-completed-gate.js
      Note: |-
        Poll-based turn completion logic used as redesign baseline
        Evidence source for waiter ergonomics issue
    - Path: openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/08-live-approval-matrix-probe.js
      Note: |-
        Multi-case orchestration boilerplate analyzed for scenario helper proposal
        Evidence source for scenario helper proposal
    - Path: openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/09-live-sandbox-variant-probe.js
      Note: |-
        Notification filtering/counter patterns used to define event tools API
        Evidence source for event filtering API
    - Path: openai-app-server/ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/05-final-full-smoke.js
      Note: |-
        Canonical continuation smoke script used for before/after API design
        Canonical before/after benchmark
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/design/01-improved-js-harness-api-design.md
      Note: |-
        Primary design output produced in this work session
        Primary design output tracked in diary steps
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/00-sketches-readme.md
      Note: Sketch scenario index added during step-3 follow-up
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/01-scientific-rlm-research-db.js
      Note: Scientific research sketch added in follow-up
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/02-production-incident-triage-db.js
      Note: Incident triage sketch added in follow-up
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/03-release-readiness-gate-db.js
      Note: Release readiness sketch added in follow-up
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/04-security-vulnerability-triage-db.js
      Note: Security triage sketch added in follow-up
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/05-customer-support-escalation-db.js
      Note: Support escalation sketch added in follow-up
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/06-vendor-due-diligence-risk-review-db.js
      Note: Vendor due diligence sketch added with resumable phase checkpoints
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/07-data-pipeline-quality-guardian-db.js
      Note: Data pipeline quality sketch added with resumable anomaly triage lifecycle
ExternalSources:
    - local:01-app-server-js.md
Summary: Step-by-step diary for OAP-04 script audit, API redesign analysis, and publication.
LastUpdated: 2026-02-22T23:50:40-05:00
WhatFor: Preserve exact workflow, findings, and rationale behind the improved JS API proposal.
WhenToUse: Use when continuing OAP-04 implementation or validating design assumptions.
---




# Diary

## Goal

Capture a detailed implementation diary for OAP-04, including script audit findings from OAP-01/OAP-02, rationale for API improvements, and publication of the resulting design document.

## Step 1: Ticket Bootstrap and Full Script Audit

I created the OAP-04 ticket workspace and immediately set up both the design and diary documents so findings could be recorded as the investigation progressed. I then ran a broad source audit over all script files in OAP-01 and OAP-02, plus the larger “real-life” harness examples in the imported source document.

The objective of this step was to collect concrete evidence of where script ergonomics are currently weak. Rather than relying on general preferences, I extracted repeated patterns directly from working scripts: ID parsing redundancy, sleep-based waiting, manual request branching, and repetitive probe orchestration code.

### Prompt Context

**User prompt (verbatim):** "Go over the other scripts in OAP-01 and OAP-02, and then write a detailed design document on improved JS API as a new ticket OAP-04-... be very detailed, lots of exampleps, api signatures, but really just a lot of examples, including some of the more real life larger ones from openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/sources/local/01-app-server-js.md . Store the design document in the ticket, keep a detailed diary as you work, and upload the document to remarkable."

**Assistant interpretation:** Create a new OAP-04 ticket, audit existing scripts and large source examples, produce a detailed improved JS API design document with many examples and signatures, keep a detailed diary during the process, and publish to reMarkable.

**Inferred user intent:** Establish a strong, implementation-ready API proposal grounded in real script pain points, with enough documentation detail that work can continue without ambiguity.

**Commit (code):** N/A (documentation-only session)

### What I did

- Created ticket workspace:
  - `docmgr ticket create-ticket --ticket OAP-04-IMPROVED-JS-API --title "Improved JS API" --topics goja,openai-app-server,codex`
- Created docs:
  - `docmgr doc add --ticket OAP-04-IMPROVED-JS-API --doc-type design --title "Improved JS Harness API Design"`
  - `docmgr doc add --ticket OAP-04-IMPROVED-JS-API --doc-type reference --title "Diary"`
- Audited script sets with line-numbered dumps:
  - OAP-01 scripts `01` through `09`
  - OAP-02 scripts `01` through `05`
- Pulled large source examples from:
  - `.../sources/local/01-app-server-js.md` (autopilot, plan-gate, tdd-loop, review-gate, rlm, auto-compact sections)
- Reviewed current runtime/module surface:
  - `pkg/js/module_codex.go`
  - `pkg/js/module_rpc.go`
  - `pkg/js/module_approval.go`
  - `pkg/js/runtime.go`

### Why

- The user explicitly requested script-driven analysis over OAP-01/OAP-02 and the larger source examples.
- A concrete API redesign requires direct mapping from pain points to new primitives.

### What worked

- Script audit quickly revealed consistent and repeated ergonomics issues.
- Existing OAP-02 `05-final-full-smoke.js` worked well as the canonical “before” benchmark.
- Source document examples gave strong guidance for “after” target ergonomics.

### What didn't work

- N/A. No command failures or blockers occurred in this step.

### What I learned

- Most complexity is not protocol complexity; it is repeated client-side orchestration boilerplate.
- A small set of wrappers (`ids`, `waitFor`, `approvals.setPolicy`) would eliminate a large percentage of current script noise.

### What was tricky to build

- The tricky part was balancing two constraints: preserve low-level power for advanced scripts and still define an opinionated high-level API.
- The underlying cause is that current scripts intentionally exercise raw surfaces; over-abstracting would risk hiding protocol details that some scripts still need.
- I addressed this by using a layered model (escape-hatch raw RPC + additive high-level wrappers).

### What warrants a second pair of eyes

- Whether the proposed helper names and namespaces (`events`, `approvals`, `ids`, `scenarios`) fit long-term style conventions.
- Whether object-only parameter signatures should be enforced now or phased in with deprecation.

### What should be done in the future

- Implement the helper APIs in phases and migrate one reference script (`05-final-full-smoke.js`) first.

### Code review instructions

- Where to start:
  - `openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/design/01-improved-js-harness-api-design.md`
- How to validate analysis basis:
  - Compare findings in design Section 3 to the audited scripts listed in `RelatedFiles`.

### Technical details

- Key script evidence sources:
  - OAP-01: `scripts/05-module-api-turn-completed-gate.js`, `scripts/08-live-approval-matrix-probe.js`, `scripts/09-live-sandbox-variant-probe.js`
  - OAP-02: `scripts/05-final-full-smoke.js`
- Key target examples source:
  - `sources/local/01-app-server-js.md` lines covering Harness 1–6.

## Step 2: Design Authoring and Publication Preparation

I authored the full OAP-04 design document with a concrete API surface, TypeScript signatures, migration strategy, and multiple rewritten script examples. The document includes both short smoke-level rewrites and large, real-life harness examples adapted from the imported source document.

The design explicitly maps old script patterns to improved helpers, with special emphasis on making `05-final-full-smoke.js` cleaner and less timing-sensitive. This gives the continuation effort a practical implementation blueprint instead of an abstract wishlist.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Deliver a highly detailed design doc with many examples and signatures, then store and publish it.

**Inferred user intent:** Produce an implementation-ready API evolution plan that can be used as the central specification for next coding phase.

**Commit (code):** N/A (documentation-only session)

### What I did

- Wrote:
  - `design/01-improved-js-harness-api-design.md`
- Included in design:
  - script-audit findings summary
  - design principles
  - proposed API signatures
  - “before/after” rewrite of `05-final-full-smoke.js`
  - adapted large examples: autopilot, plan-gate, tdd-loop, review-gate, recursive decomposition, auto-compaction
  - migration strategy and acceptance criteria
- Updated this diary with full step-format details.

### Why

- The user asked for a very detailed design with many practical examples.
- The implementation team needs a clear migration path, not only new API concepts.

### What worked

- The improved API structure naturally absorbed the biggest script friction points.
- Large source examples translated cleanly into the new wrapper model.

### What didn't work

- N/A in writing phase.

### What I learned

- Introducing `waitFor` and `turn.waitCompleted` is likely the single highest value change for reliability and readability.
- Approval policy centralization (`approvals.setPolicy`) also has strong leverage.

### What was tricky to build

- The main challenge was avoiding “too much framework.”
- I kept the design anchored by making every high-level helper an additive wrapper over existing protocol semantics and keeping `rpc` as the escape hatch.

### What warrants a second pair of eyes

- Ensure no proposed helper unintentionally conflicts with future official app-server naming.
- Verify edge-case semantics for waiter cancellation/timeouts in runtime implementation plan.

### What should be done in the future

- Implement phase-by-phase helpers and immediately convert one canonical script as a proving ground.

### Code review instructions

- Where to start:
  - `openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/design/01-improved-js-harness-api-design.md`
- How to validate:
  - Confirm each proposed helper corresponds to a repeated pattern in audited scripts.
  - Check that large examples remain protocol-faithful to source document behavior.

### Technical details

- Proposed high-impact helper groups:
  - `ids.*`
  - `waitFor` / `turn.waitCompleted`
  - `approvals.setPolicy`
  - `events.*` and optional scenario helpers.

## Step 3: Implementation-Level Design Expansion and Sketch Script Pack

After the baseline OAP-04 design was completed, I expanded it into a handoff-grade implementation blueprint with file-level engineering details, runtime algorithms, edge-case handling, and explicit test case names. In parallel, I started building a practical sketch script pack under `scripts/sketches/` so the next implementation engineer has concrete real-world targets, not only API prose.

This step focused on two concrete outcomes: making the design doc implementation-ready and capturing scenario-driven examples that stress the API in real workflows (research, incidents, releases, security, support escalation). The sketches are intentionally design-time scripts against the proposed APIs.

### Prompt Context

**User prompt (verbatim):** "save as example in the scripts/sketches/ folder of OAP-04...\n\nCreate a few other example scripts of real world scenarios (not just rlm)"

**Assistant interpretation:** Add the RLM example as a script sketch in OAP-04 and create additional non-RLM real-world scenario scripts in the same sketches folder.

**Inferred user intent:** Make the proposal tangible and implementation-oriented by providing reusable scenario templates.

**Commit (code):** N/A (documentation + sketch scripts only)

### What I did

- Added `scripts/sketches/00-sketches-readme.md` describing intent and script index.
- Added/updated sketch scripts:
  - `scripts/sketches/01-scientific-rlm-research-db.js`
  - `scripts/sketches/02-production-incident-triage-db.js`
  - `scripts/sketches/03-release-readiness-gate-db.js`
  - `scripts/sketches/04-security-vulnerability-triage-db.js`
  - `scripts/sketches/05-customer-support-escalation-db.js`
- Expanded `design/01-improved-js-harness-api-design.md` with:
  - implementation blueprint by file,
  - runtime waiter/policy algorithms,
  - contract edge cases,
  - phased PR sequence,
  - regression risks/mitigations,
  - detailed test case checklist,
  - companion sketch script section.

### Why

- The user requested concrete examples for real-world scenarios and deeper implementation detail for handoff readiness.
- The next engineer benefits more from executable-style target scripts and low-ambiguity implementation guidance than from conceptual API design alone.

### What worked

- The same API primitives (`ids`, `waitCompleted`, approval policies, `db.*`) generalized well across very different domains.
- The design doc became substantially more actionable after adding file-by-file implementation guidance and explicit tests.

### What didn't work

- A tool turn was interrupted during script creation workflow; state had to be re-verified before continuing.
- Resolution: verified current file list and resumed in requested order (design additions first, then sketches).

### What I learned

- Scenario-driven sketches expose API gaps faster than abstract type signatures.
- Incident/release/security/support scripts validate that non-RLM workflows need the same core ergonomic helpers as RLM-style orchestration.

### What was tricky to build

- The hardest part was keeping sketches realistic while clearly signaling that they target proposed APIs not fully implemented yet.
- If sketches look fully production-ready, they can mislead implementation expectations.
- I addressed this by labeling each file as a sketch and adding a folder README that states design-time intent.

### What warrants a second pair of eyes

- Ensure proposed runtime policy routing for approvals does not conflict with existing manual `onRequest` flows.
- Ensure `waitFor`/waiter implementation is leak-safe under heavy notification streams.

### What should be done in the future

- Wire sketch scripts into OAP-04 acceptance checklists once helpers are implemented.
- Add one minimal “runs today” compatibility sketch for each scenario once Phase 1 helpers land.

### Code review instructions

- Where to start:
  - `openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/design/01-improved-js-harness-api-design.md`
  - `openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/00-sketches-readme.md`
- How to validate:
  - Confirm implementation blueprint sections map directly to current Go file structure.
  - Confirm each sketch script demonstrates at least one proposed helper and durable `db.*` usage.

### Technical details

- New sketch scenarios:
  - scientific research orchestration,
  - production incident triage,
  - release readiness gate,
  - security vulnerability triage,
  - customer support escalation.

## Step 5: Sequenced Expansion (Design First, Then Sketches)

In this step I followed the requested sequence explicitly: expand the design document first, then update and extend the sketch script pack. The design work focused on clearer onboarding and implementation clarity for new contributors, with paragraph-first sections that explain how the harness lifecycle, resumability, and script-owned schema decisions fit together. After that was complete, I revised the sketch scripts to make resumability behavior more consistent across scenarios.

The second half of this step converted sketches that were previously mostly persistence-focused into explicit phase/checkpoint flows. This means the examples now better demonstrate how a script can survive restarts and continue work safely, which was a key user intent across OAP-03 and OAP-04 requests.

### Prompt Context

**User prompt (verbatim):** "start with design additions first. Then the sketch scripts."

**Assistant interpretation:** Perform the remaining work in strict order by expanding the design first, then editing/adding script sketches.

**Inferred user intent:** Enforce implementation sequencing and deliver clearer, more practical handoff artifacts.

**Commit (code):** N/A (documentation + sketch scripts only)

### What I did

- Expanded `design/01-improved-js-harness-api-design.md` with new sections:
  - intern onboarding walkthrough in plain language,
  - script-owned DB resumability contract and API surface,
  - more elegant direction for `05-final-full-smoke.js`,
  - script author lifecycle template with schema migration hook,
  - concrete migration checklist for OAP-01/OAP-02 scripts.
- Updated `scripts/sketches/00-sketches-readme.md` with stronger explanation of schema ownership and resumability intent.
- Reworked sketches to include explicit checkpoint/resume phases where missing:
  - `03-release-readiness-gate-db.js`
  - `04-security-vulnerability-triage-db.js`
  - `05-customer-support-escalation-db.js`
- Added new real-world sketches:
  - `06-vendor-due-diligence-risk-review-db.js`
  - `07-data-pipeline-quality-guardian-db.js`

### Why

- The user requested strict ordering and clearer practical examples.
- The next engineer needs both API theory and scenario-level resumability patterns.

### What worked

- The design additions improved readability for onboarding by explaining lifecycle concepts before deep API detail.
- The phase/checkpoint model generalized cleanly to additional domains beyond RLM-style research.

### What didn't work

- N/A. No command failures occurred in this step.

### What I learned

- Resumability examples are significantly clearer when each script has explicit `resumeOrCreate` + phase transitions.
- Sketch quality improves when they show both schema ownership and lifecycle checkpoints, not one without the other.

### What was tricky to build

- The main challenge was keeping examples realistic while avoiding false signals that they are fully runtime-ready today.
- The root cause is that OAP-04 sketches target proposed APIs that may not exist yet in production code.
- I addressed this by preserving the “sketch” framing while still writing deterministic phase/checkpoint flows.

### What warrants a second pair of eyes

- Ensure checkpoint payload conventions are consistent enough to document as a shared guideline.
- Validate that approval policy examples in write-enabled scenarios remain safe defaults.

### What should be done in the future

- Once OAP-04 helpers are implemented, promote one or two sketches into runnable smoke scripts and add execution tests.

### Code review instructions

- Where to start:
  - `openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/design/01-improved-js-harness-api-design.md`
  - `openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/00-sketches-readme.md`
- How to validate:
  - Confirm new sections 24-28 in the design align with existing implementation blueprint sections.
  - Confirm updated/new sketches include explicit phase + checkpoint resume flow.

### Technical details

- New/updated sketch inventory:
  - `01-scientific-rlm-research-db.js`
  - `02-production-incident-triage-db.js`
  - `03-release-readiness-gate-db.js` (rewritten for resumability)
  - `04-security-vulnerability-triage-db.js` (rewritten for resumability)
  - `05-customer-support-escalation-db.js` (rewritten for resumability)
  - `06-vendor-due-diligence-risk-review-db.js` (new)
  - `07-data-pipeline-quality-guardian-db.js` (new)

## Step 4: Implement the First Five Helper APIs and Validate With Runtime Tests

I implemented the first five helper groups from the OAP-04 plan directly in `pkg/js`: identity extraction helpers, event waiter helpers, turn completion waiter, event metrics, and approval policy tools. The runtime now supports deterministic wait-based orchestration and policy-driven approval responses instead of repeated ad hoc script logic.

I also added targeted runtime tests for each helper and ran full repository tests. This closes the requested helper implementation set while keeping Phase 3 (canonical script migration) as the next explicit step.

### Prompt Context

**User prompt (verbatim):** "alright, add tasks to the current ticket and build all 5"

**Assistant interpretation:** Add detailed implementation tasks to OAP-04 and implement all five priority helpers immediately.

**Inferred user intent:** Move from design-only OAP-04 to concrete runtime capability that directly improves branch-coverage testability.

**Commit (code):** pending

### What I did

- Expanded OAP-04 tasks into phased checklist structure and helper-level subtasks.
- Implemented helper APIs:
  - `session.ids.thread(...)` and `session.ids.turn(...)`
  - `session.waitFor({ method, timeoutMs, where })`
  - `thread.turn.waitCompleted({ turnId?, timeoutMs? })`
  - `session.events.metrics()` with `countByMethod`, `totalNotifications`, `totalRequests`, `reset`
  - `session.approvals.setPolicy({ command, fileChange, fallback })`
  - `session.approvals.respond(reqOrId, decision)`
- Added runtime internals:
  - event waiter registry with timeout settlement
  - event metric counters
  - approval policy callback routing on inbound approval requests
- Added tests in `pkg/js/runtime_test.go` for:
  - helper surface availability
  - ID extraction behavior
  - `waitFor` notification gating
  - `waitCompleted` turn filtering
  - metrics counting/reset
  - policy-driven approval responses
- Ran validation:
  - `go test ./pkg/js -count=1`
  - `go test ./... -count=1`

### Why

- These helpers are the smallest additive set that removes the biggest coverage/testing pain points observed in OAP-02 live scripts.

### What worked

- Helper APIs integrated without breaking existing wrapper routes.
- Waiters and policy responses behaved deterministically in tests.
- Full repository tests passed after integration.

### What didn't work

- Initial compile pass failed because promise resolve/reject types from goja were `func(any) error`, not `func(goja.Value) error`.
- Fix: adjusted waiter resolver/rejector field types and reran tests.

### What I learned

- Waiters and metrics belong in runtime-level infrastructure rather than ad hoc script helpers to keep semantics consistent across harness scripts.

### What was tricky to build

- The trickiest part was ensuring timeout-driven waiter rejection is safe under concurrent event delivery.
- Root cause: waiters can resolve from event dispatch and timeout goroutines at nearly the same time.
- Approach: single `settleEventWaiter` path with lock-protected `done` state and one-time map removal.

### What warrants a second pair of eyes

- Approval policy + manual `onRequest` handlers can both be active; scripts should avoid double-response patterns when `setPolicy` is enabled.

### What should be done in the future

- Phase 3: migrate canonical `05-final-full-smoke` script to v2 helper APIs and add richer `ui.emit` diagnostics for branch-miss causes.

### Code review instructions

- Where to start:
  - `openai-app-server/pkg/js/runtime.go`
  - `openai-app-server/pkg/js/module_codex.go`
  - `openai-app-server/pkg/js/runtime_test.go`
  - `openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/tasks.md`
- How to validate:
  - `go test ./pkg/js -count=1`
  - `go test ./... -count=1`

### Technical details

- Waiter default timeout:
  - `30s` when `timeoutMs` is omitted.
- Event matching sources:
  - `session.waitFor` defaults to both requests and notifications, configurable via `includeRequests` / `includeNotifications`.

## Step 6: Migrate Canonical Final Smoke Script to v2 Helpers With Branch-Miss Diagnostics

I migrated the canonical smoke script into OAP-04 as `scripts/05-final-full-smoke.js`, rewritten to use the new helper APIs added in Step 4. The script now exercises helper-level lifecycle and policy flows directly, instead of re-implementing request routing and timing gates in ad hoc script code.

I also added detailed diagnostics so missed coverage branches are explicit: method-level event counts, sampled requests/notifications, elapsed checkpoints, approval wait timeout details, turn-completion wait timeout details, and an explicit `branchMissReasons` list in the final completion payload.

### Prompt Context

**User prompt (verbatim):** "go ahead."

**Assistant interpretation:** Continue into the next phase and complete the remaining script migration work.

**Inferred user intent:** Finish OAP-04’s canonical script migration now that the helper APIs exist.

**Commit (code):** pending

### What I did

- Added canonical v2 script:
  - `ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/05-final-full-smoke.js`
- Migrated script to helper APIs:
  - `session.ids.thread(...)`
  - `session.ids.turn(...)`
  - `session.waitFor(...)`
  - `thread.turn.waitCompleted(...)`
  - `session.events.metrics()`
  - `session.approvals.setPolicy(...)`
- Added diagnostics:
  - checkpointed timing emits (`final-full-smoke-v2-checkpoint`)
  - sampled request/notification envelopes
  - approval wait matched/timeout events
  - turn wait matched/timeout events
  - final `summary.branchMissReasons`
- Updated tasks:
  - checked off canonical migration + diagnostics items in Phase 3.
- Revalidated codebase:
  - `go test ./... -count=1`

### Why

- OAP-04’s value depends on showing the improved API in a real canonical script, not only unit-tested helper primitives.

### What worked

- Script now uses all core helper surfaces in one place.
- Full repository tests still pass after migration.

### What didn't work

- N/A in this step.

### What I learned

- Embedding coverage diagnostics into the script output sharply reduces ambiguity during live harness validation.

### What was tricky to build

- The hard part was making pass/fail strict enough for branch evidence while preserving enough context to debug misses quickly.
- Approach: keep strict branch conditions but include method counts, checkpoints, and sample payloads in the same completion event.

### What warrants a second pair of eyes

- Strict branch checks may need minor timeout tuning after first repeated live runs in slower environments.

### What should be done in the future

- Commit Phase 3 script/doc updates.
- Prepare the real-run command for the v2 script and stop for approval before execution.

### Code review instructions

- Where to start:
  - `openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/05-final-full-smoke.js`
  - `openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/tasks.md`
- How to validate:
  - `go test ./... -count=1`
  - inspect `final-full-smoke-v2-complete` payload shape in a future live run.

### Technical details

- Completion marker:
  - `final-full-smoke-v2-complete`
- Branch-miss field:
  - `summary.branchMissReasons`
