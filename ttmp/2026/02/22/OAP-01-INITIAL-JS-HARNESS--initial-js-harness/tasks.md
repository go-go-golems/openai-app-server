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
