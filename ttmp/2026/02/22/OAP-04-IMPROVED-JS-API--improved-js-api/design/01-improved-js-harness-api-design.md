---
Title: Improved JS Harness API Design
Ticket: OAP-04-IMPROVED-JS-API
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
      Note: |-
        Existing approval helpers informing policy-layer API design
        Existing approval helper semantics informing policy API
    - Path: openai-app-server/pkg/js/module_codex.go
      Note: |-
        Current codex wrapper shape and method signatures to evolve
        Current codex session API and wrapper shape baseline for redesign
    - Path: openai-app-server/pkg/js/module_rpc.go
      Note: |-
        Low-level RPC passthrough that should remain as escape hatch
        Low-level RPC escape hatch preserved in proposed layered API
    - Path: openai-app-server/pkg/js/runtime.go
      Note: |-
        Runtime event dispatch, handler registration, and async bridge constraints
        Runtime event dispatch and callback model for proposed waiters/policies
    - Path: openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/05-module-api-turn-completed-gate.js
      Note: |-
        Example of manual polling and custom event gate logic
        Manual polling example replaced by waitCompleted proposal
    - Path: openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/08-live-approval-matrix-probe.js
      Note: |-
        Multi-case probe that demonstrates repeated orchestration boilerplate
        Case-runner boilerplate informing scenario helper proposal
    - Path: openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/scripts/09-live-sandbox-variant-probe.js
      Note: |-
        Notification filtering and probe harness complexity baseline
        Notification filtering and metric aggregation baseline
    - Path: openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/sources/local/01-app-server-js.md
      Note: |-
        Larger harness examples and target long-form API direction
        Large real-life harness examples adapted in design
    - Path: openai-app-server/ttmp/2026/02/22/OAP-02-JS-HARNESS-CONTINUATION--js-harness-continuation/scripts/05-final-full-smoke.js
      Note: |-
        Most complete current smoke script and strongest ergonomic benchmark
        Primary before/after script in API redesign
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/00-sketches-readme.md
      Note: Companion sketch index referenced by implementation blueprint section
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/01-scientific-rlm-research-db.js
      Note: Scientific scenario reference for proposed helper ergonomics
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/02-production-incident-triage-db.js
      Note: Incident scenario reference for non-RLM workflows
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/03-release-readiness-gate-db.js
      Note: Release gate scenario reference for durable check state
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/04-security-vulnerability-triage-db.js
      Note: Security triage scenario reference
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/05-customer-support-escalation-db.js
      Note: Support escalation scenario reference
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/06-vendor-due-diligence-risk-review-db.js
      Note: Vendor risk review scenario reference
    - Path: openai-app-server/ttmp/2026/02/22/OAP-04-IMPROVED-JS-API--improved-js-api/scripts/sketches/07-data-pipeline-quality-guardian-db.js
      Note: Data pipeline quality scenario reference
ExternalSources:
    - local:01-app-server-js.md
Summary: Detailed proposal for a cleaner, more expressive JS harness API with concrete migration mapping from OAP-01/OAP-02 scripts and large production-style examples.
LastUpdated: 2026-02-23T14:45:00-05:00
WhatFor: Define an elegant next-generation JS API that reduces script boilerplate and enables complex harness patterns safely.
WhenToUse: Use when implementing OAP-02 continuation work or authoring new harness scripts with stronger ergonomics.
---




# Improved JS Harness API Design

## 1. Executive Summary

The current JS API in `openai-app-server/pkg/js` is functional but low-level. It can run real harness scripts and has already validated key protocol paths (thread start, turn start, approvals, thread read, review start). However, script authors are repeatedly rebuilding the same helper logic in userland: extracting IDs from variant payloads, waiting for lifecycle events with ad hoc sleeps, writing case runners by hand, and manually handling approval branching.

The goal of OAP-04 is not to remove low-level control. The goal is to add an elegant higher-level layer that preserves raw `rpc` power while giving script authors first-class primitives for common orchestration tasks. This should reduce script size, lower bug rate, and make complex patterns from the imported source document practical in day-to-day harness code.

The proposed design introduces a two-layer API:

1. Keep `rpc` and current low-level methods for escape-hatch control.
2. Add rich high-level session/thread/turn/event/approval helpers that encode the repeated patterns seen across OAP-01 and OAP-02.

## 2. Scope and Inputs

This design is based on three concrete inputs:

1. OAP-01 scripts (`01-first-real-harness.js` through `09-live-sandbox-variant-probe.js`).
2. OAP-02 scripts (`01-wrapper-api-smoke.js` through `05-final-full-smoke.js`).
3. Large “real-life” harness examples in `sources/local/01-app-server-js.md` (autopilot approvals, plan gate, TDD loop, review gate, recursive decomposition, auto-compaction).

Out of scope for this ticket:

- implementing the API in Go code,
- changing protocol semantics on the server,
- replacing existing scripts immediately.

In scope:

- API shape,
- ergonomic design,
- migration guidance,
- detailed code examples.

## 3. Findings From OAP-01 and OAP-02 Script Audit

Across both ticket script sets, we see recurring friction patterns.

### 3.1 Repeated ID extraction boilerplate

Many scripts manually parse `thread.id` vs `thread.threadId` and `turn.id` vs `turn.turnId` with fallback logic. This increases noise and introduces subtle inconsistencies.

Representative scripts:

- OAP-01 `04-module-api-thread-turn-live.js`
- OAP-01 `05-module-api-turn-completed-gate.js`
- OAP-02 `01-wrapper-api-smoke.js`
- OAP-02 `05-final-full-smoke.js`

### 3.2 Sleep-based completion instead of lifecycle gates

Most scripts use `clock.sleep(...)` after `turn/start` and hope enough events arrive during that window. This is brittle in slower environments and wastes time in fast environments.

Representative scripts:

- OAP-01 `03-module-api-live-smoke.js`
- OAP-02 `04-phase6-review-gate-smoke.js`
- OAP-02 `05-final-full-smoke.js`

### 3.3 Manual approval request branching in every script

Scripts repeatedly branch on request methods and call `approval.acceptForSession` or `approval.decline` manually. This is repetitive and hard to audit.

Representative scripts:

- OAP-01 `06-live-inbound-request-probe.js`
- OAP-01 `07-live-escalation-request-probe.js`
- OAP-02 `02-builtin-autopilot-smoke.js`
- OAP-02 `05-final-full-smoke.js`

### 3.4 Repeated case-runner scaffolding for probes

Probe scripts (approval matrix and sandbox variants) create similar orchestration pipelines repeatedly: setup case, start thread, run turn, wait, summarize deltas.

Representative scripts:

- OAP-01 `08-live-approval-matrix-probe.js`
- OAP-01 `09-live-sandbox-variant-probe.js`

### 3.5 Event filtering and metric aggregation are ad hoc

Scripts locally define notification filters, counters, and status aggregations. Useful for one script, but not standardized across harnesses.

Representative scripts:

- OAP-01 `09-live-sandbox-variant-probe.js`
- OAP-02 `05-final-full-smoke.js`

## 4. Design Principles for an Elegant API

An elegant API here should satisfy six principles:

1. Progressive disclosure:
- Simple tasks need short code.
- Complex tasks can still access raw RPC and detailed hooks.

2. Explicit lifecycle semantics:
- Waiting for turn completion should be event-gated, not sleep-based.

3. Stable identity helpers:
- Thread/turn/item ID extraction should be centralized.

4. Composable policies:
- Approval behavior should be declarative and reusable.

5. Observability by default:
- Event subscriptions, filters, and counters should be simple and standardized.

6. Backward compatibility:
- Existing OAP-01/OAP-02 scripts should continue to run.

## 5. Proposed API Architecture

Keep three layers:

1. `rpc` module: raw JSON-RPC escape hatch.
2. `codex` high-level client: lifecycle helpers and typed wrappers.
3. Optional harness helpers (`scenarios`, `policies`, `events`) for advanced scripts.

### 5.1 New `codex.connect` shape

```ts
declare module "codex" {
  interface ConnectOptions {
    defaults?: {
      model?: string;
      cwd?: string;
      approvalPolicy?: string;
      sandbox?: string;
    };
    capabilities?: {
      optOutNotificationMethods?: string[];
    };
  }

  function connect(options?: ConnectOptions): Session;
}
```

This preserves zero-arg usage and adds defaulting so every script does not repeat `model/cwd/approvalPolicy/sandbox` payloads.

### 5.2 Session API

```ts
interface Session {
  connected: boolean;

  request(method: string, params?: any): Promise<any>;
  notify(method: string, params?: any): void;
  respond(idOrReq: any, result: any): void;
  respondError(idOrReq: any, code: number, message: string, data?: any): void;

  onNotification(cb: (evt: RpcNotification) => void): () => void;
  onRequest(cb: (evt: RpcRequest) => void): () => void;

  threads: ThreadCollection;
  thread(id: string): ThreadHandle;

  events: EventTools;
  approvals: ApprovalPolicyTools;
  ids: IdentityTools;

  waitFor(spec: WaitForSpec): Promise<WaitForResult>;
}
```

Key additions:

- `waitFor(...)` first-class event gate,
- `events` helper namespace,
- `approvals` policy namespace,
- `ids` helper namespace.

### 5.3 Thread collection and handles

```ts
interface ThreadCollection {
  start(params?: ThreadStartParams): Promise<ThreadStartResult>;
  list(params?: { limit?: number }): Promise<ThreadSummary[]>;
  read(params: { threadId: string; includeTurns?: boolean }): Promise<ThreadReadResult>;

  // Optional convenience
  startAndRun(params: ThreadStartParams & { input: InputBlock[] }): Promise<{ threadId: string; turnId: string }>;
}

interface ThreadHandle {
  id: string;

  read(params?: { includeTurns?: boolean }): Promise<ThreadReadResult>;
  compact: {
    start(params?: any): Promise<any>;
  };

  turn: {
    start(params: TurnStartParams): Promise<TurnStartResult>;
    steer(params: TurnSteerParams): Promise<any>;
    interrupt(params?: { turnId?: string }): Promise<any>;

    waitCompleted(params?: { turnId?: string; timeoutMs?: number }): Promise<TurnCompletedEvent>;
    waitPlan(params?: { turnId?: string; timeoutMs?: number }): Promise<TurnPlanUpdatedEvent>;
  };

  review: {
    start(params: ReviewStartParams): Promise<ReviewStartResult>;
  };
}
```

Important refinement:

- `threads.read(threadId, true)` compatibility may remain, but preferred API is object-only (`{ threadId, includeTurns }`).

### 5.4 Identity helpers

```ts
interface IdentityTools {
  thread(v: any): string;
  turn(v: any): string;
  item(v: any): string;
}
```

These replace repeated `extractThreadId()` and `extractTurnId()` helpers in user scripts.

### 5.5 Event tools

```ts
interface EventTools {
  on(method: string, cb: (evt: any) => void): () => void;
  once(method: string, cb: (evt: any) => void): () => void;

  waitFor(spec: {
    method: string;
    timeoutMs?: number;
    where?: (evt: any) => boolean;
  }): Promise<any>;

  filter(methodPrefixes: string[]): (evt: any) => boolean;

  metrics(): {
    countByMethod(): Record<string, number>;
    totalNotifications(): number;
    reset(): void;
  };
}
```

This standardizes common probe patterns and removes hand-rolled counters in each script.

### 5.6 Approval policy tools

```ts
type ApprovalDecision =
  | "accept"
  | "acceptForSession"
  | "decline"
  | "cancel"
  | { acceptWithExecpolicyAmendment: { execpolicy_amendment: string[] } };

interface ApprovalPolicyTools {
  setPolicy(policy: {
    command?: (req: ApprovalRequestContext) => ApprovalDecision | Promise<ApprovalDecision>;
    fileChange?: (req: ApprovalRequestContext) => ApprovalDecision | Promise<ApprovalDecision>;
    fallback?: (req: ApprovalRequestContext) => ApprovalDecision | Promise<ApprovalDecision>;
  }): () => void;

  respond(req: any, decision: ApprovalDecision): void;
}
```

This lets scripts define declarative policy once instead of branching request-by-request.

## 6. Concrete Rewrite: OAP-02 `05-final-full-smoke.js`

### 6.1 Current pain points

- manual `asString` conversion,
- manual request method branching,
- manual ID extraction,
- sleep-based wait (`clock.sleep(18000)`),
- thread read via mixed signature.

### 6.2 Improved API rewrite

```js
const codex = require("codex");
const ui = require("ui");

async function run() {
  const session = codex.connect({
    defaults: {
      model: "gpt-5",
      cwd: "/home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server",
      approvalPolicy: "on-request",
      sandbox: "read-only"
    }
  });

  const summary = {
    requestsObserved: 0,
    approvalsAccepted: 0,
    approvalsDeclined: 0,
    turnCompleted: false,
    threadReadWorked: false,
    errors: []
  };

  session.approvals.setPolicy({
    command: (req) => {
      summary.requestsObserved += 1;
      const cmd = req.commandText || "";
      if (cmd.startsWith("curl ")) {
        summary.approvalsAccepted += 1;
        ui.emit({ type: "final-full-smoke-approval", method: req.method, decision: "acceptForSession", command: cmd });
        return "acceptForSession";
      }
      summary.approvalsDeclined += 1;
      ui.emit({ type: "final-full-smoke-approval", method: req.method, decision: "decline", command: cmd });
      return "decline";
    },
    fileChange: (req) => {
      summary.requestsObserved += 1;
      summary.approvalsDeclined += 1;
      ui.emit({ type: "final-full-smoke-approval", method: req.method, decision: "decline" });
      return "decline";
    }
  });

  ui.emit({ type: "final-full-smoke-start" });

  const started = await session.threads.start();
  const threadId = session.ids.thread(started);
  const thread = session.thread(threadId);

  ui.emit({ type: "final-full-smoke-thread-started", threadId });

  const turn = await thread.turn.start({
    input: [{
      type: "text",
      text: "Run `curl -I https://example.com` and then respond with one short sentence containing the HTTP status line."
    }]
  });

  const turnId = session.ids.turn(turn);
  ui.emit({ type: "final-full-smoke-turn-started", turnId });

  await thread.turn.waitCompleted({ turnId, timeoutMs: 120000 });
  summary.turnCompleted = true;
  ui.emit({ type: "final-full-smoke-turn-completed", turnId });

  const readResult = await thread.read({ includeTurns: true });
  summary.threadReadWorked = !!readResult;
  ui.emit({ type: "final-full-smoke-thread-read", ok: summary.threadReadWorked, threadId });

  ui.emit({ type: "final-full-smoke-complete", ok: summary.threadReadWorked && summary.errors.length === 0, threadId, summary });
}

run().catch((err) => {
  ui.emit({ type: "final-full-smoke-complete", ok: false, error: String(err) });
});
```

This reduces boilerplate and removes timing fragility.

## 7. Rewrites of Key OAP-01 Patterns

### 7.1 Turn completion gate without custom polling

Current OAP-01 `05-module-api-turn-completed-gate.js` manually loops with `clock.sleep(100)`.

Improved:

```js
const session = codex.connect();

const started = await session.threads.start({ ... });
const threadId = session.ids.thread(started);
const thread = session.thread(threadId);

const turn = await thread.turn.start({ input: [{ type: "text", text: "Say hello" }] });
const turnId = session.ids.turn(turn);

await thread.turn.waitCompleted({ turnId, timeoutMs: 15000 });
ui.emit({ type: "module-api-turn-gate-complete", ok: true, threadId, turnId });
```

### 7.2 Approval matrix probe as scenario runner

Current OAP-01 `08-live-approval-matrix-probe.js` has repeated case plumbing.

Proposed helper:

```ts
interface ScenarioTools {
  runCases<TCase, TResult>(spec: {
    cases: TCase[];
    runCase: (c: TCase, ctx: ScenarioCaseContext) => Promise<TResult>;
    delayBetweenMs?: number;
  }): Promise<{ results: TResult[] }>;
}
```

Usage:

```js
const result = await session.scenarios.runCases({
  cases,
  delayBetweenMs: 1000,
  runCase: async (cfg, ctx) => {
    const t = await session.threads.start({ approvalPolicy: cfg.approvalPolicy, sandbox: cfg.sandbox });
    const threadId = session.ids.thread(t);
    const thread = session.thread(threadId);

    const turn = await thread.turn.start({ input: [ ... ] });
    const turnId = session.ids.turn(turn);

    await session.waitFor({
      method: "turn/completed",
      timeoutMs: cfg.waitMs,
      where: (evt) => evt.params?.threadId === threadId
    });

    return ctx.summary({ threadId, turnId });
  }
});
```

## 8. Large Real-Life Examples (From Source Doc, Adapted)

The imported source document has six large harness examples. Below is how each becomes cleaner with the improved API.

### 8.1 Autopilot approvals

```js
function autopilotApprovals() {
  return codex.defineHarness({
    name: "autopilot-approvals",

    async onStart(ctx) {
      ctx.session.approvals.setPolicy({
        command: (req) => {
          if (req.commandArgv && req.commandArgv[0] === "go" && req.commandArgv[1] === "test") {
            return "acceptForSession";
          }
          return ctx.ui.promptApproval(req);
        },
        fileChange: (req) => {
          if (req.touchesProtectedPath) return "decline";
          return req.totalDiffLines < 120 ? "acceptForSession" : ctx.ui.promptApproval(req);
        }
      });
    }
  });
}
```

### 8.2 Plan-gate execution

```js
function planGate() {
  return codex.defineHarness({
    name: "plan-gate",

    on: {
      async "turn/plan/updated"(ctx, evt) {
        const turnId = ctx.ids.turn(evt.params);
        const approved = ctx.state.turn(turnId).get("planApproved");
        if (approved) return;

        const choice = await ctx.ui.promptPlan(evt.params.plan || []);
        if (choice === "approve") {
          ctx.state.turn(turnId).set("planApproved", true);
          await ctx.thread.turn.steer({ expectedTurnId: turnId, input: [{ type: "text", text: "Plan approved. Proceed." }] });
        } else if (choice === "requestChanges") {
          await ctx.thread.turn.steer({ expectedTurnId: turnId, input: [{ type: "text", text: "Revise the plan." }] });
        } else {
          await ctx.thread.turn.interrupt({ turnId });
        }
      }
    }
  });
}
```

### 8.3 TDD loop

```js
function tddLoop({ testCmd = ["go", "test", "./..."], maxIters = 5 } = {}) {
  return codex.defineHarness({
    name: "tdd-loop",

    on: {
      async "turn/completed"(ctx, evt) {
        const turnId = ctx.ids.turn(evt.params.turn || evt.params);
        if (!ctx.turn.wasSuccessful(turnId)) return;

        const iter = ctx.counters.inc(`tdd:${ctx.thread.id}`);
        if (iter > maxIters) return;

        const res = await ctx.session.command.exec({ command: testCmd, cwd: ctx.thread.cwd });
        ctx.ui.emit({ type: "tdd/testResult", iter, exitCode: res.exitCode });

        if (res.exitCode !== 0) {
          await ctx.thread.turn.start({
            input: [{ type: "text", text: `Fix failing tests.\n\nSTDERR:\n${res.stderr}` }]
          });
        }
      }
    }
  });
}
```

### 8.4 Review gate

```js
function reviewGate() {
  return codex.defineHarness({
    name: "review-gate",

    on: {
      async "turn/completed"(ctx, evt) {
        if (!ctx.turn.isCompleted(evt)) return;

        const review = await ctx.thread.review.start({ delivery: "inline", target: { type: "uncommittedChanges" } });
        const item = await ctx.thread.waitItem({ type: "exitedReviewMode", timeoutMs: 120000 });

        const text = item.review || "";
        if (ctx.review.looksRisky(text)) {
          await ctx.thread.turn.start({
            input: [{ type: "text", text: `Address reviewer feedback:\n\n${text}` }]
          });
        }
      }
    }
  });
}
```

### 8.5 Recursive decomposition (RLM-ish)

```js
function rlmDecompose({ maxWorkers = 4 } = {}) {
  return codex.defineHarness({
    name: "rlm-decompose",

    on: {
      async "ui/submitPrompt"(ctx, uiEvt) {
        const plan = await ctx.thread.turn.start({
          input: [{ type: "text", text: `Decompose task:\n${uiEvt.text}` }],
          outputSchema: ctx.schemas.subtaskPlan
        });

        const subtasks = await plan.getStructuredOutput().subtasks;

        const results = await ctx.pool(maxWorkers).map(subtasks, async (st) => {
          const worker = await ctx.session.threads.start({ sandbox: "read-only", approvalPolicy: "never" });
          const workerId = ctx.ids.thread(worker);

          const out = await ctx.session.thread(workerId).turn.start({
            input: [{ type: "text", text: `Subtask: ${st.title}\n${st.prompt}` }]
          });

          return { title: st.title, summary: await out.waitForFinalAgentMessage() };
        });

        await ctx.thread.turn.start({
          input: [{ type: "text", text: `Implement using this guidance:\n\n${ctx.formatWorkerResults(results)}` }]
        });
      }
    }
  });
}
```

### 8.6 Auto-compaction

```js
function autoCompact({ softLimit = 0.8 } = {}) {
  return codex.defineHarness({
    name: "auto-compact",

    on: {
      async "thread/tokenUsage/updated"(ctx, evt) {
        const ratio = evt.params?.ratio || 0;
        if (ratio < softLimit) return;
        if (ctx.state.thread(ctx.thread.id).get("compactionRequested")) return;

        ctx.state.thread(ctx.thread.id).set("compactionRequested", true);
        await ctx.thread.compact.start();
      }
    }
  });
}
```

## 9. Optional Built-in Helpers That Unlock Elegance

These helpers are not required for v1, but they offer strong leverage.

1. `session.turns.awaitCompleted(threadId, turnId, timeoutMs)`
2. `session.approvals.policy.presets.autopilot(...)`
3. `session.events.aggregate({ include, exclude })`
4. `session.probes.matrix(cases, runner)`
5. `session.thread(id).resume({ policy })`

These should be small wrappers built on top of existing primitives, not deep framework magic.

## 10. Backward Compatibility and Migration Strategy

### 10.1 Compatibility stance

- Keep existing methods (`session.request`, `session.onNotification`, etc.) unchanged.
- Keep `rpc` module unchanged.
- Keep `approval` module unchanged.

### 10.2 Migration order

1. Add identity helpers (`ids.thread`, `ids.turn`).
2. Add event waiters (`waitFor`, `thread.turn.waitCompleted`).
3. Add approval policy wrapper (`approvals.setPolicy`).
4. Add scenario runner for case-based probes.

### 10.3 Script migration examples

- OAP-02 `01-wrapper-api-smoke.js`: remove manual id extraction + sleep.
- OAP-02 `02-builtin-autopilot-smoke.js`: replace request branching with `approvals.setPolicy`.
- OAP-02 `05-final-full-smoke.js`: use full improved stack as canonical reference script.

## 11. Proposed Go-Side Surface Changes

### 11.1 `module_codex.go`

Add methods on built session object:

- `waitFor(...)`
- `events` object and utilities
- `approvals.setPolicy(...)`
- `ids` utilities
- `thread(...).read(...)`
- `thread(...).turn.waitCompleted(...)`

### 11.2 `runtime.go`

Add internal support for:

- event waiter registration and timeout handling
- event dispatch indexing by method
- policy callback routing for approval requests

### 11.3 Keep low-level modules unchanged

`rpc`, `ui`, `approval` remain thin and stable.

## 12. Testing Strategy

### 12.1 Unit tests

- `ids.thread/turn` extraction across payload variants.
- `waitFor` timeout, success, and predicate filtering.
- `approvals.setPolicy` method routing.

### 12.2 Integration tests (memory transport)

- start thread -> start turn -> wait completed event.
- inbound approval request auto-policy response.
- thread read object signature normalization.

### 12.3 Script-level smoke suite

- migrate OAP-02 `05-final-full-smoke.js` to v2 API and keep as reference.
- validate existing scripts still run unchanged.

## 13. Acceptance Criteria for OAP-04

The API update is successful when:

1. The final smoke script shrinks significantly while preserving behavior.
2. Sleep-based waits are replaced by lifecycle event waits.
3. At least one probe script uses scenario runner utilities.
4. Approval logic can be configured declaratively in one place.
5. Existing old scripts still run without modification.

## 14. Final Recommendation

Implement the improved API as additive wrappers over current transport and event infrastructure. Prioritize identity extraction, event waiters, and approval policy tools first, because those three changes eliminate most current script boilerplate.

Then migrate OAP-02 final smoke script into the new style and use it as the canonical documentation example. This delivers immediate elegance gains without destabilizing the lower-level runtime foundation.

## 15. Implementation Blueprint (File-by-File)

This section is the handoff-grade implementation map for the next engineer. It specifies which files to edit, what to add, and what to keep stable.

### 15.1 `pkg/js/runtime.go`

Add new runtime-owned state:

```go
type runtimeWaiter struct {
    id        int64
    method    string
    predicate goja.Callable // optional
    resolve   goja.Callable
    reject    goja.Callable
    timer     *time.Timer
}

type approvalPolicyCallbacks struct {
    command   goja.Callable
    fileChange goja.Callable
    fallback  goja.Callable
}
```

Add fields to `Runtime`:

```go
waitersMu sync.Mutex
waiters   map[int64]*runtimeWaiter
nextWaiterID atomic.Int64

approvalPolicyMu sync.RWMutex
approvalPolicy   *approvalPolicyCallbacks
```

Core runtime responsibilities:

1. Register/remove waiters.
2. Evaluate method + optional predicate when notifications arrive.
3. Resolve waiter promises on VM thread.
4. Reject on timeout with clear error message.
5. Route approval requests to policy callbacks if set.

### 15.2 `pkg/js/module_codex.go`

Add or expand these exports in `buildCodexSession`:

- `session.ids`
- `session.events`
- `session.waitFor`
- `session.approvals.setPolicy`
- `thread.read`
- `thread.turn.waitCompleted`
- `thread.turn.waitPlan`

Do not remove existing exports. Keep existing behavior for:

- `session.request`
- `session.notify`
- `session.onNotification`
- `session.onRequest`
- `session.threads.start/list/read`

### 15.3 `pkg/js/module_approval.go`

Keep existing explicit response helpers (`accept`, `decline`, etc.) as low-level API.

Add one internal helper callable from `module_codex.go`:

```go
func respondApprovalDecisionByValue(rt *Runtime, id any, decision any) error
```

This lets `approvals.setPolicy` return either a string decision or amendment object and still reuse one response serializer.

### 15.4 `pkg/js/runtime_test.go`

Add tests in three groups:

1. Waiter behavior:
- resolves on matching method
- respects predicate
- times out correctly
- unsubscribes/cleans up timer state

2. Approval policy behavior:
- command policy callback auto-responds with accepted decision
- file-change policy callback declines
- fallback policy used when method not recognized

3. Backward compatibility:
- existing direct `onNotification` and `approval.*` helpers still work unchanged

### 15.5 Optional new file: `pkg/js/module_codex_helpers.go`

If `module_codex.go` becomes too large, split helper builders into:

- identity helpers
- event helpers
- approval policy helper binding
- thread waiter helpers

This reduces future merge conflicts and makes testing easier.

## 16. Runtime Algorithms

### 16.1 Waiter registration algorithm

```text
JS session.waitFor(spec)
  -> validate spec.method
  -> allocate waiter id
  -> create Promise
  -> store waiter in map
  -> if timeoutMs > 0 start timer
  -> return Promise
```

Timer callback must post back to VM thread and reject the same promise only if waiter is still present in map.

### 16.2 Notification dispatch with waiter matching

```text
EmitRPCNotification(method, params)
  -> build payload
  -> existing callbacks first (for compatibility)
  -> iterate waiter map snapshot
     -> method match?
     -> predicate match? (if any)
     -> resolve and remove waiter
```

Important edge case:

- predicate throw should reject waiter and remove it (not crash runtime).

### 16.3 Approval policy routing algorithm

```text
EmitRPCRequest(id, method, params)
  -> emit to existing onRequest callbacks (compat)
  -> if method is approval request:
       if policy callback exists:
          call callback on VM thread with normalized request context
          accept return value as decision
          respond once
```

Normalization target for callback context:

```ts
interface ApprovalRequestContext {
  id: any;
  method: string;
  params: any;
  threadId?: string;
  turnId?: string;
  commandText?: string;
  commandArgv?: string[];
  fileChanges?: any[];
}
```

## 17. API Contract Details and Edge Cases

### 17.1 `session.ids.thread(v)` behavior

Extraction order:

1. `v.threadId` string
2. `v.id` string
3. `v.thread.threadId` string
4. `v.thread.id` string

Return:

- `string` on success
- empty string (`""`) on no match (do not throw)

Reason:

- lets scripts choose strictness themselves.
- strict variants can call `session.ids.requireThread(v)` in future.

### 17.2 `session.waitFor(spec)` behavior

Input requirements:

- `spec.method` required non-empty string
- `spec.timeoutMs` optional default 30000
- `spec.where` optional predicate

Resolution payload:

```ts
{
  method: string;
  params: any;
}
```

Timeout error text:

- `timed out waiting for notification method "<method>"`

### 17.3 `thread.turn.waitCompleted` behavior

Implementation:

- wrapper over `session.waitFor({ method: "turn/completed", ... })`
- auto-predicate filters by `threadId` and optional `turnId`

Pseudo:

```js
await thread.turn.waitCompleted({ turnId, timeoutMs: 60000 })
```

Equivalent internal predicate:

```js
evt.params?.threadId === thread.id &&
(!turnId || (evt.params?.turn?.id || evt.params?.turnId) === turnId)
```

### 17.4 `session.approvals.setPolicy(policy)`

Policy callback return types:

1. String decision:
- `"accept" | "acceptForSession" | "decline" | "cancel"`

2. Amendment object:

```js
{ acceptWithExecpolicyAmendment: { execpolicy_amendment: ["curl", "-I"] } }
```

Callback failure behavior:

- log and fallback to explicit `decline` (safe default)

Return value:

- unsubscribe/reset function that clears policy callbacks.

## 18. Implementation Sequence (Recommended PR Plan)

### PR 1: Identity + waiters

Changes:

- `session.ids`
- `session.waitFor`
- `thread.turn.waitCompleted`
- tests for waiters and id extraction

Why first:

- biggest script readability gain
- low protocol risk

### PR 2: Approval policy layer

Changes:

- `session.approvals.setPolicy`
- request context normalization
- decision serialization helper
- tests for command/file-change/fallback policy paths

### PR 3: Event tools + metrics

Changes:

- `session.events.on/once/waitFor/filter/metrics`
- tests for metrics aggregation and filter composition

### PR 4: Canonical script migration

Changes:

- migrate OAP-02 `scripts/05-final-full-smoke.js` to v2 style
- keep old script variant in archive for comparison if useful
- add playbook update to reflect new script behavior

## 19. Regression Risks and Mitigations

### 19.1 Risk: double-response to approval request

Cause:

- both old `onRequest` handler and new policy layer respond.

Mitigation:

- policy layer should only respond when enabled.
- document that mixed mode requires either policy-only or manual-only response discipline.
- optional future guard: runtime tracks request IDs responded by policy and warns on duplicate manual response.

### 19.2 Risk: waiter memory leaks

Cause:

- unresolved waiters remain in map after timeout/panic.

Mitigation:

- centralized cleanup in resolve/reject paths.
- test for map size returning to baseline after timeout.

### 19.3 Risk: breaking current scripts via signature hardening

Cause:

- changing `threads.read(threadId, includeTurns)` behavior abruptly.

Mitigation:

- keep compatibility normalization path.
- prefer object signature in docs and lint warnings only.

## 20. Detailed Test Cases (Copy/Paste List)

Suggested test names:

- `TestCodexIDsThreadExtractionVariants`
- `TestCodexIDsTurnExtractionVariants`
- `TestSessionWaitForResolvesOnMethodMatch`
- `TestSessionWaitForPredicateRejectsNonMatchingEvents`
- `TestSessionWaitForTimeout`
- `TestThreadTurnWaitCompletedFiltersThreadAndTurn`
- `TestApprovalsSetPolicyCommandAcceptForSession`
- `TestApprovalsSetPolicyFileChangeDecline`
- `TestApprovalsSetPolicyFallbackUsed`
- `TestApprovalsSetPolicyCallbackErrorFallsBackDecline`
- `TestLegacyOnRequestAndApprovalHelpersStillWork`

## 21. Developer Notes for the Next Engineer

1. Keep implementation additive. Do not remove or rename existing exported methods in initial rollout.
2. Add tests before migrating scripts.
3. Migrate only one script first (`05-final-full-smoke.js`) and verify parity.
4. Use `ui.emit` instrumentation events during rollout to make waiter/policy behavior observable.
5. Treat `rpc` module as the permanent low-level contract; high-level wrappers should delegate, not fork behavior.

## 22. Appendix: Minimal V2 Smoke Script Template

```js
const codex = require("codex");
const ui = require("ui");

async function main() {
  const session = codex.connect({ defaults: { model: "gpt-5", sandbox: "read-only" } });
  session.approvals.setPolicy({ command: () => "acceptForSession", fileChange: () => "decline" });

  const started = await session.threads.start();
  const threadId = session.ids.thread(started);
  const thread = session.thread(threadId);

  const turn = await thread.turn.start({ input: [{ type: "text", text: "hello" }] });
  const turnId = session.ids.turn(turn);

  await thread.turn.waitCompleted({ turnId, timeoutMs: 60000 });
  const full = await thread.read({ includeTurns: true });

  ui.emit({ type: "v2-smoke-complete", ok: !!full, threadId, turnId });
}

main().catch((err) => ui.emit({ type: "v2-smoke-complete", ok: false, error: String(err) }));
```

## 23. Companion Sketch Scripts

To make implementation and API review practical, companion script sketches were added under:

- `scripts/sketches/01-scientific-rlm-research-db.js`
- `scripts/sketches/02-production-incident-triage-db.js`
- `scripts/sketches/03-release-readiness-gate-db.js`
- `scripts/sketches/04-security-vulnerability-triage-db.js`
- `scripts/sketches/05-customer-support-escalation-db.js`
- `scripts/sketches/06-vendor-due-diligence-risk-review-db.js`
- `scripts/sketches/07-data-pipeline-quality-guardian-db.js`

These are intentionally not runtime-stable scripts yet. They are design-time examples written against the proposed OAP-04 high-level APIs and OAP-03-style `db.*` persistence. Their purpose is to provide concrete “target ergonomics” for the next implementation engineer.

When implementing new helpers, treat these sketches as acceptance-oriented references:

1. If a helper makes these scripts shorter and clearer, it is likely valuable.
2. If a helper forces these scripts into awkward workarounds, refine the API.
3. Keep raw `rpc` escape hatches available so sketches can drop down a level where needed.

## 24. Intern Onboarding Walkthrough (Plain Language)

If you are new to this codebase, the most important thing to understand is that the JS harness is an orchestration layer, not the model itself. Your script is a controller that opens a session, creates threads, starts turns, watches lifecycle notifications, and optionally responds to tool approval requests. The runtime is event-driven, so most reliability issues come from handling lifecycle events incorrectly rather than from prompt text quality.

A useful mental model is to think in three nested levels. A `session` is your network connection and event bus. A `thread` is a conversational workspace with history and artifacts. A `turn` is one request/response execution unit within a thread. Resumability means persisting enough information to reconnect these levels after a crash so the script can continue instead of restarting from zero.

In current scripts, engineers often write ad hoc code for extracting IDs and waiting for completion events. The improved API treats these as first-class responsibilities because every reliable harness needs them. New intern rule of thumb: avoid bare sleeps when waiting for model progress, and avoid hand-written ID parsing in each file. Use standardized helpers so behavior remains consistent across scripts and easier to debug.

ASCII lifecycle map:

```text
JS Script
  |
  | connect()
  v
Session -----------------------------+
  |                                   \
  | threads.start()                    \ notifications (turn/completed, plan/updated, ...)
  v                                     \
ThreadHandle                             +--> waiters/events/policies in runtime
  |
  | turn.start(input)
  v
TurnHandle
  |
  | waitCompleted()
  v
Persist outcome + checkpoint in db
```

Crash/resume map:

```text
Boot
  |
  +--> init schema (CREATE TABLE IF NOT EXISTS ...)
  |
  +--> load checkpoint row
          |
          +--> no checkpoint: create job/thread
          |
          +--> checkpoint exists: rebind to threadId/turnId and continue phase
```

## 25. DB-Backed Resumability Contract (Script-Owned Schema)

The preferred model is script-owned schema, where each script controls its own tables and migration logic. This is the cleanest way to support heterogeneous workflows because an incident triage harness, a release gate harness, and a research harness do not share the same state graph. Forcing one generic state table for all scripts sounds simple initially but becomes harder to evolve and reason about once scenario complexity grows.

Script-owned schema means each script is responsible for four explicit lifecycle functions: `initSchema`, `resumeOrCreate`, `runPhase`, and `saveCheckpoint`. This keeps state transitions intentional and reviewable. The runtime only provides a low-friction `db.*` primitive surface; business data model decisions remain with the script author.

Recommended runtime-level `db` module surface:

```ts
declare module "db" {
  function exec(sql: string, args?: any[]): Promise<void>;
  function queryOne<T = any>(sql: string, args?: any[]): Promise<T | null>;
  function queryAll<T = any>(sql: string, args?: any[]): Promise<T[]>;
  function tx<T>(fn: () => Promise<T>): Promise<T>;

  // Optional ergonomics (proposed)
  function scalar<T = any>(sql: string, args?: any[]): Promise<T | null>;
  function migrate(spec: {
    scriptId: string;
    targetVersion: number;
    up: Array<{ version: number; sql: string[] }>;
  }): Promise<{ from: number; to: number }>;
}
```

Recommended per-script schema minimum:

1. `jobs` table (or equivalent root entity).
2. `checkpoints` table with phase + pointer IDs.
3. Domain tables (subtasks/findings/actions/etc.).
4. Optional `schema_version` table when not using `db.migrate`.

Suggested checkpoint payload:

```json
{
  "jobId": "job_123",
  "phase": "workers-running",
  "threadId": "thread_abc",
  "activeTurnId": "turn_789",
  "retryCount": 1,
  "updatedAt": 1771900000
}
```

Phase-driven resumability pseudocode:

```js
async function main(input) {
  await initSchema();
  const state = await resumeOrCreate(input); // returns { phase, ids... }

  if (state.phase === "planning") {
    await runPlanning(state);
    await saveCheckpoint({ ...state, phase: "workers-running" });
  }

  if (state.phase === "workers-running") {
    await runWorkers(state);
    await saveCheckpoint({ ...state, phase: "synthesizing" });
  }

  if (state.phase === "synthesizing") {
    await runSynthesis(state);
    await saveCheckpoint({ ...state, phase: "complete" });
  }
}
```

This approach is intentionally explicit. Every phase transition is persisted immediately after success, so a restart resumes from the most recent durable boundary rather than replaying everything.

## 26. More Elegant API Direction for `05-final-full-smoke.js`

The current continuation smoke script is useful as an integration check, but it still reads like protocol plumbing instead of harness logic. The elegant target is a short, intention-first script where setup, policy, execution, wait, and validation are visually separated and each step has exactly one responsibility.

The main ergonomic upgrades for this script are: one-line thread bootstrap, built-in completion waiter, decision policy registration, and a compact read/assert phase. This reduces cognitive load for reviewers because the script no longer mixes transport-level concerns with test intent.

Proposed convenience signatures:

```ts
interface ThreadCollection {
  start(params?: ThreadStartParams): Promise<ThreadStartResult>;
  startHandle(params?: ThreadStartParams): Promise<ThreadHandle>; // proposed
}

interface ThreadHandle {
  run(input: InputBlock[], opts?: { timeoutMs?: number }): Promise<{
    turnId: string;
    completedEvent: any;
  }>; // proposed sugar over turn.start + waitCompleted
}
```

Example using these helpers:

```js
const session = codex.connect({
  defaults: { model: "gpt-5", sandbox: "read-only", approvalPolicy: "on-request" }
});

session.approvals.setPolicy({
  command: (req) => req.commandText?.startsWith("curl ") ? "acceptForSession" : "decline",
  fileChange: () => "decline",
  fallback: () => "decline"
});

const thread = await session.threads.startHandle();
const { turnId } = await thread.run([
  { type: "text", text: "Run `curl -I https://example.com` and report the status line." }
], { timeoutMs: 120000 });

const full = await thread.read({ includeTurns: true });
ui.emit({ type: "smoke-complete", ok: !!full, turnId, threadId: thread.id });
```

This is still transparent enough to debug while being much easier for new contributors to read and modify.

## 27. Script Author Template: Self-Managed Schema + Resume Hooks

Each script should expose a predictable internal contract so collaborators can quickly locate persistence logic. A practical pattern is to centralize lifecycle hooks in one object, then keep orchestration flow small in `main()`.

Template:

```js
const lifecycle = {
  async initSchema() {},
  async resumeOrCreate(input) {},
  async runPhase(state) {},
  async saveCheckpoint(state) {},
  async finalize(state) {}
};

async function main(input) {
  await lifecycle.initSchema();
  let state = await lifecycle.resumeOrCreate(input);
  while (state.phase !== "complete") {
    state = await lifecycle.runPhase(state);
    await lifecycle.saveCheckpoint(state);
  }
  await lifecycle.finalize(state);
}
```

Why this helps:

1. Reviewers can inspect correctness by following a known function order.
2. Resume bugs are easier to isolate because checkpoint writes are centralized.
3. Scenario evolution is safer because schema and phase transitions live together.

When a script needs custom schema evolution, add a dedicated migration function:

```js
async function initSchema() {
  await db.exec(`CREATE TABLE IF NOT EXISTS my_script_schema_version (version INTEGER NOT NULL)`);
  const row = await db.queryOne(`SELECT version FROM my_script_schema_version LIMIT 1`);
  const v = row ? row.version : 0;

  if (v < 1) {
    await db.exec(`CREATE TABLE IF NOT EXISTS my_script_jobs (...)`);
    await db.exec(`INSERT OR REPLACE INTO my_script_schema_version(version) VALUES (1)`);
  }
  if (v < 2) {
    await db.exec(`ALTER TABLE my_script_jobs ADD COLUMN retry_count INTEGER DEFAULT 0`);
    await db.exec(`UPDATE my_script_schema_version SET version = 2`);
  }
}
```

This keeps ownership local to the script while still being deterministic and testable.

## 28. Migration Checklist for OAP-01/OAP-02 Scripts

A consistent migration checklist makes it easier to modernize scripts incrementally without regressions. The checklist below should be applied script-by-script instead of doing one large rewrite.

1. Replace local `extractThreadId`/`extractTurnId` helpers with `session.ids.*`.
2. Replace raw sleeps waiting for completion with `thread.turn.waitCompleted` or `session.waitFor`.
3. Move per-request approval branching into `session.approvals.setPolicy`.
4. Normalize reads to object form: `thread.read({ includeTurns: true })`.
5. Add explicit checkpoint writes if the script has multi-phase behavior.
6. Keep raw `session.request(...)` calls only for unsupported edge methods.

Suggested migration batching:

```text
Batch A (low risk): 01-wrapper-api-smoke.js, 05-module-api-turn-completed-gate.js
Batch B (medium): 06/07 approval probe scripts
Batch C (high): 08/09 matrix+variant probes, 05-final-full-smoke.js
```

Validation checklist after each migrated script:

1. Script runs without transport-level errors.
2. Completion waiters are event-driven (no fixed sleep gate for correctness).
3. Approval requests always get exactly one response.
4. Final `ui.emit` summary includes thread/turn IDs and success/failure flags.
