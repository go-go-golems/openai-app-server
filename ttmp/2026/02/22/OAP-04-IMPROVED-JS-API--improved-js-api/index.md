---
Title: Improved JS API
Ticket: OAP-04-IMPROVED-JS-API
Status: active
Topics:
    - goja
    - openai-app-server
    - codex
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: openai-app-server/pkg/js/module_codex.go
      Note: Current module surface targeted by API redesign
    - Path: openai-app-server/pkg/js/runtime.go
      Note: Runtime callbacks and event dispatch constraints for proposed helpers
    - Path: openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/sources/local/01-app-server-js.md
      Note: Imported large harness examples used as high-level API benchmark
    - Path: openai-app-server/ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/05-final-full-smoke.js
      Note: Canonical migration target script for improved API
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/design/01-improved-js-harness-api-design.md
      Note: Primary OAP-04 improved API design document
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/reference/01-diary.md
      Note: Detailed working diary for audit and design steps
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/00-sketches-readme.md
      Note: Sketch script index and intent for proposed improved APIs
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/01-scientific-rlm-research-db.js
      Note: Scientific research RLM scenario using proposed APIs and db.* persistence
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/02-production-incident-triage-db.js
      Note: Production incident triage scenario with durable timeline and hypotheses
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/03-release-readiness-gate-db.js
      Note: Release readiness go/no-go gate scenario
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/04-security-vulnerability-triage-db.js
      Note: Security vulnerability triage and mitigation scenario
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/05-customer-support-escalation-db.js
      Note: Customer support escalation orchestration scenario
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/06-vendor-due-diligence-risk-review-db.js
      Note: Vendor due diligence risk review scenario with resumable phases
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/07-data-pipeline-quality-guardian-db.js
      Note: Data pipeline anomaly triage scenario with checkpointed progress
ExternalSources: []
Summary: OAP-04 design and examples for an improved JS harness API with resumability patterns and script-owned DB schema.
LastUpdated: 2026-02-23T15:08:00-05:00
WhatFor: Provide implementation-ready API design, migration guidance, and scenario sketches for next engineering steps.
WhenToUse: Use when implementing or reviewing OAP-04 API helpers and resumable harness scripts.
---




# Improved JS API

## Overview

Design and document a cleaner JS harness API that reduces script boilerplate and improves reliability while preserving low-level RPC escape hatches.

Primary outputs:

- `design/01-improved-js-harness-api-design.md`
- `reference/01-diary.md`
- `scripts/sketches/*.js` (real-world scenario API sketches)

Key design focus:

- eliminate repeated ID extraction and sleep-based waits,
- introduce first-class event waiters and approval policy helpers,
- provide migration guidance from OAP-01/OAP-02 scripts,
- include large real-life harness examples adapted from imported source guidance.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- goja
- openai-app-server
- codex

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
