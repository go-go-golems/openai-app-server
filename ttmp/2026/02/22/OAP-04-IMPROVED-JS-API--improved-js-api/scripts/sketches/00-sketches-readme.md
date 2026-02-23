---
Title: Sketch Scripts Index
Ticket: OAP-04-IMPROVED-JS-API
Status: active
Topics:
    - goja
    - openai-app-server
    - codex
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Index and intent for OAP-04 scenario sketch scripts with schema ownership and resumability patterns.
LastUpdated: 2026-02-23T14:55:00-05:00
WhatFor: Quick map of the non-production sketch scripts used to stress proposed APIs.
WhenToUse: Use when reviewing scenario examples for the improved JS API.
---

# OAP-04 Sketch Scripts

These scripts are design sketches for the proposed improved JS API in OAP-04.

They are intentionally written against the target API (for example `session.ids`, `thread.turn.waitCompleted`, `session.approvals.setPolicy`) and the proposed `db.*` module from OAP-03.

Each sketch assumes script-owned schema. That means the script defines its own tables, migration/versioning behavior, and checkpoint format. This is the core pattern for reliable resumability: script authors choose the state model that best fits their workflow instead of forcing every scenario into one shared generic schema.

They are not expected to run unchanged on the current runtime. They should be read as target ergonomics and implementation acceptance references.

## Included sketches

1. `01-scientific-rlm-research-db.js`
- Multi-worker scientific research orchestration with durable subtasks and synthesis.

2. `02-production-incident-triage-db.js`
- Incident response orchestration with durable timeline, hypotheses, and remediation plan.

3. `03-release-readiness-gate-db.js`
- Release go/no-go gate with persisted checks, blockers, and final decision report.

4. `04-security-vulnerability-triage-db.js`
- Vulnerability triage pipeline with severity scoring and prioritized mitigation backlog.

5. `05-customer-support-escalation-db.js`
- Escalation handling flow that produces customer-facing updates and engineering actions.

6. `06-vendor-due-diligence-risk-review-db.js`
- Vendor risk review flow for procurement/security with durable findings and approval memo generation.

7. `07-data-pipeline-quality-guardian-db.js`
- Data pipeline anomaly triage flow with resumable checkpoints and remediation backlog.
