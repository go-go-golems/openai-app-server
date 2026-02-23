# Changelog

## 2026-02-22

- Initial workspace created


## 2026-02-22

Step 1: Bootstrapped OAP-02 from OAP-01 handoff, authored continuation roadmap, prefilled multi-phase tasks, and established diary baseline after successful Phase 16 live validation.

### Related Files

- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/design/01-continuation-implementation-roadmap.md — Detailed forward implementation plan
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/reference/01-diary.md — Initial continuation diary entry
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/tasks.md — Prefilled phase-by-phase task checklist


## 2026-02-22

Step 2: Implemented Phase 1 thread/read live wiring with typed codexrpc ThreadRead decoder and integration/unit coverage.

### Related Files

- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/cmd/openai-app-server/thread_read_command.go — Replaced skeleton command with live RPC wiring
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/cmd/openai-app-server/thread_read_command_test.go — Added command integration test for thread read
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/pkg/codexrpc/threads.go — Added Thread model and ThreadRead method
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/pkg/codexrpc/threads_test.go — Added wrapped/direct/malformed payload tests
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/tasks.md — Phase 1 task progress updated


## 2026-02-22

Step 3: Implemented Phase 2 state projection package (models/store/projector) with bounded retention and replay tests.

### Related Files

- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/pkg/state/models.go — Thread/turn/item projection models
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/pkg/state/projector.go — Method-based notification projector
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/pkg/state/projector_test.go — Replay projection and malformed-event tests
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/pkg/state/store.go — Bounded in-memory projected state store
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/pkg/state/store_test.go — Bounds and snapshot isolation tests
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/tasks.md — Phase 2 task progress updated


## 2026-02-22

Step 4: Completed Phase 3 wrapper API expansion (threads + thread handles), added wrapper runtime routing tests, and created first OAP-02 wrapper smoke script/playbook assets.

### Related Files

- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/pkg/js/module_codex.go — Added session/thread wrapper API surface and helper builders
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/pkg/js/runtime.go — Added wrapper-oriented request promise helper
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/pkg/js/runtime_test.go — Added wrapper method routing coverage
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/playbooks/01-wrapper-api-smoke-plan.md — Wrapper smoke run plan and checks
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/01-wrapper-api-smoke.js — Initial wrapper smoke script
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/tasks.md — Phase 3 progress updated


## 2026-02-22

Step 5: Implemented Phase 4 harness core package (context/compose/dispatch) with exactly-once response guardrails and deterministic dispatch tests.

### Related Files

- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/pkg/harness/compose.go — Deterministic harness composition
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/pkg/harness/context.go — Request context and response diagnostics model
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/pkg/harness/dispatch.go — Notification/request dispatch pipeline
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/pkg/harness/dispatch_test.go — Ordering
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/tasks.md — Phase 4 task progress updated


## 2026-02-22

Step 6: Implemented Phase 5 built-ins (autopilot + plan gate), added deterministic built-in tests, and prepared Phase 5 live smoke scripts/playbooks.

### Related Files

- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/pkg/harness/builtin/autopilot.go — Autopilot approval policy harness
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/pkg/harness/builtin/builtin_test.go — Deterministic unit coverage for built-ins
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/pkg/harness/builtin/plan_gate.go — Plan-gate harness with steer/interrupt controller
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/02-builtin-autopilot-smoke.js — Autopilot smoke script
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/03-builtin-plan-gate-smoke.js — Plan-gate smoke script
- /home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server/ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/tasks.md — Phase 5 pre-run tasks completed

