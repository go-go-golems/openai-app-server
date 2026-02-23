---
Title: Resuming Sessions
Ticket: OAP-03-RESUMING-SESSIONS
Status: active
Topics:
    - goja
    - openai-app-server
    - codex
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: openai-app-server/cmd/openai-app-server/harness_run_command.go
      Note: Harness run command where resume flags and bootstrap hooks will be added
    - Path: openai-app-server/pkg/codexrpc/client.go
      Note: Codex RPC client state machine relevant to reconnect/resume behavior
    - Path: openai-app-server/pkg/js/module_codex.go
      Note: Codex JS module where resume/session APIs are proposed
    - Path: openai-app-server/pkg/js/runtime.go
      Note: Runtime event pipeline where checkpoint and replay integration will hook in
    - Path: openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/sources/local/01-app-server-js.md
      Note: Baseline imported requirements used as primary reference
    - Path: openai-app-server/ttmp/2026/02/22/OAP-03-RESUMING-SESSIONS--resuming-sessions/design/01-js-harness-session-resume-architecture-analysis.md
      Note: Primary OAP-03 resume architecture analysis document
    - Path: openai-app-server/ttmp/2026/02/22/OAP-03-RESUMING-SESSIONS--resuming-sessions/design/02-sqlite-db-object-for-harness-resume-and-state.md
      Note: Second analysis document focused on SQLite db object for resumability and persistent workflow data
    - Path: openai-app-server/ttmp/2026/02/22/OAP-03-RESUMING-SESSIONS--resuming-sessions/design/03-script-owned-sqlite-schemas-and-resume-patterns.md
      Note: Third analysis document focused on script-owned schema and per-script init/resume methods
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-22T22:41:02.219474573-05:00
WhatFor: ""
WhenToUse: ""
---




# Resuming Sessions

## Overview

Define how the openai-app-server JS harness should resume existing remote Codex threads and recover local harness runtime state after process restarts.

Primary deliverable:

- `design/01-js-harness-session-resume-architecture-analysis.md`
- `design/02-sqlite-db-object-for-harness-resume-and-state.md`
- `design/03-script-owned-sqlite-schemas-and-resume-patterns.md`

These documents include technical options, API shapes, pseudocode, diagrams, phased implementation planning, and SQLite-backed persistence/resume patterns.

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
