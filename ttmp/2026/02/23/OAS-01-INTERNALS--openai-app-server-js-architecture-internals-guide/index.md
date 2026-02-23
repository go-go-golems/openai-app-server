---
Title: OpenAI App Server JS - Architecture & Internals Guide
Ticket: OAS-01-INTERNALS
Status: active
Topics:
    - architecture
    - documentation
    - onboarding
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: openai-app-server/cmd/openai-app-server/harness_run_command.go
      Note: Harness run orchestrator
    - Path: openai-app-server/pkg/codexrpc/client.go
      Note: JSON-RPC 2.0 client with handshake FSM
    - Path: openai-app-server/pkg/harness/dispatch.go
      Note: Harness event dispatcher
    - Path: openai-app-server/pkg/js/module_codex.go
      Note: Main JS API - session
    - Path: openai-app-server/pkg/js/runtime.go
      Note: Core JS runtime - event bridging
    - Path: openai-app-server/pkg/state/projector.go
      Note: Event-to-state projection
    - Path: openai-app-server/pkg/state/store.go
      Note: Thread-safe in-memory state store
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-23T09:57:11.435377092-05:00
WhatFor: ""
WhenToUse: ""
---


# OpenAI App Server JS - Architecture & Internals Guide

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- architecture
- documentation
- onboarding

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
