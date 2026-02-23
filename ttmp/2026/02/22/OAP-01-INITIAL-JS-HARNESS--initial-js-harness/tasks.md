# Tasks

## Phase 0: Planning and Baseline

- [x] Create ticket workspace and import source document
- [x] Analyze go-go-goja and geppetto goja integration patterns
- [x] Write 5+ page architecture design document for openai-app-server JS harness
- [x] Upload analysis documents to reMarkable
- [x] Record detailed diary entries for initial analysis/publication

## Phase 1: Bootstrap Repository for Real Implementation

- [x] Replace placeholder module path in `openai-app-server/go.mod`
- [x] Replace `cmd/XXX/main.go` with real `openai-app-server` CLI entrypoint
- [x] Add Glazed-based root command scaffolding with logging init
- [x] Add `harness` and `thread` command groups (initial skeleton)
- [x] Add config defaults package for transport/session defaults
- [x] Run `go test ./...` (bootstrap compile/test)
- [x] Commit Phase 1 bootstrap changes
- [x] Record diary step for Phase 1

## Phase 2: Codex RPC Protocol + Handshake Core

- [x] Add JSON-RPC protocol models (`request/response/error/notification`)
- [x] Add codex transport interface and stdio transport skeleton
- [x] Add codex client with strict handshake state machine (`initialize` then `initialized`)
- [x] Add request router (correlation map + notification handlers)
- [x] Add unit tests: handshake success, pre-handshake rejection, duplicate initialize rejection
- [x] Run `go test ./...` (protocol/handshake)
- [x] Commit Phase 2 protocol/handshake changes
- [x] Record diary step for Phase 2

## Phase 3: Progressive CLI Validation (No Real Harness Yet)

- [x] Add `thread list` command wired to codex client API surface
- [x] Add `thread read` command skeleton with structured output
- [x] Add integration test with fake/in-memory transport for `thread list`
- [x] Run `go test ./...` (CLI + fake transport)
- [x] Commit Phase 3 CLI changes
- [x] Record diary step for Phase 3

## Phase 4: JS Runtime and Harness API Skeleton

- [x] Add goja runtime bootstrap with runtimeowner runner
- [x] Expose host primitives (`rpc`, `ui`, `clock`) to JS
- [x] Add `require("codex")` module skeleton (`connect`, event subscription stubs)
- [x] Add JS runtime unit tests for module loading and callback threading invariants
- [x] Run `go test ./...` (runtime skeleton)
- [x] Commit Phase 4 runtime/harness skeleton changes
- [x] Record diary step for Phase 4

## Phase 5: First Real Harness Test Gate

- [x] Prepare real harness test plan (script + target app-server transport)
- [x] **STOP-GATE:** Notify user immediately before launching first real harness test
- [x] (Pending user confirmation) Run first real harness test
- [x] Archive all harness scripts under `ttmp/.../scripts/` (retroactive migration from `/tmp`)

## Phase 6: Replace `__host` with Native JS Modules

- [x] Add native modules `require("ui")`, `require("rpc")`, and `require("clock")`
- [x] Remove global `__host` injection from runtime bootstrap
- [x] Update harness code/tests/scripts to use `require("ui")` instead of `__host.ui`
- [x] Update runtime tests to validate module-based API and absence of `__host`
- [x] Run `go test ./...` after migration
- [x] Commit module API migration changes
- [x] Record diary/changelog updates for module API migration

## Phase 7: Module-API Real Run Gate (Post-Migration)

- [x] Update ticket design/playbook docs to reflect module API (`require("ui"|"rpc"|"clock")`)
- [x] Add dedicated module-API live-run script under `ttmp/.../scripts/`
- [x] Add explicit expected-output checks for module-API run in playbook
- [x] Run `go test ./...` preflight before live run
- [x] **STOP-GATE:** Explain next real-run script behavior + exact command and wait for approval
- [x] Execute module-API real harness run against live server (after approval)

## Phase 8: Module-API Live Thread/Turn Scenario Gate

- [x] Add module-API script for `thread/start` + `turn/start` flow under `ttmp/.../scripts/`
- [x] Add playbook for thread/turn scenario with explicit success/failure markers
- [x] Run `go test ./...` preflight before live run
- [x] **STOP-GATE:** Explain thread/turn script behavior + exact command and wait for approval
- [x] Execute thread/turn module-API live run against real server (after approval)

## Phase 9: Deterministic Turn-Completion Live Gate

- [x] Add module-API script that waits for matching `turn/completed` notification before success
- [x] Add playbook with deterministic completion markers and timeout behavior
- [x] Run `go test ./...` preflight before live run
- [x] **STOP-GATE:** Explain deterministic turn-completion script + exact command and wait for approval
- [x] Execute deterministic turn-completion live run (after approval)

## Phase 10: Request/Response Plumbing for Inbound RPC Requests

- [x] Add JSON-RPC response envelope helpers (`NewResponse`, `NewErrorResponse`)
- [x] Add `codexrpc.Client` response send APIs (`Respond`, `RespondError`)
- [x] Extend JS `rpc` module with `respond` and `respondError`
- [x] Extend `codex.connect()` session with `respond` and `respondError`
- [x] Add tests for response sending and request->response harness flow
- [x] Run `go test ./...` after response-path implementation
- [x] Prepare and gate next live scenario for real inbound request handling

## Phase 11: Live Inbound Request Probe Scenario

- [x] Add live probe script that attempts to trigger and handle inbound request(s)
- [x] Add playbook with explicit probe success/failure markers
- [x] Run `go test ./...` preflight before live probe run
- [x] Execute live inbound-request probe run and capture outcome

## Phase 12: Live Escalation Probe (Network Command)

- [x] Add escalation-focused probe script (network command) under `ttmp/.../scripts/`
- [x] Add playbook for escalation probe markers
- [x] Run `go test ./...` preflight before escalation probe
- [x] Execute escalation probe run and capture outcome
