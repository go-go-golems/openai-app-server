---
Title: Imported App Server JS Requirements
Ticket: OAP-01-INITIAL-JS-HARNESS
Status: active
Topics:
    - goja
    - openai-app-server
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
    - https://developers.openai.com/codex/app-server/
Summary: "Imported source requirements document used as baseline for JS harness design and validation."
LastUpdated: 2026-02-23T06:35:00-05:00
WhatFor: "Capture upstream design/requirements text imported from /tmp for traceable ticket analysis."
WhenToUse: "Reference when validating architecture and playbook assumptions against imported guidance."
---

Below is a JS-first “harness” API design that sits *on top of* Codex App Server’s bidirectional JSON‑RPC stream (threads → turns → items), with examples of harnesses that react to events/approvals and steer the agent. I’m going to align the API tightly to the App Server concepts because those primitives are already designed to be UI-friendly and composable. ([OpenAI Developers][1])

---

## What the App Server gives you (and what the JS API should embrace)

Key protocol facts your JS layer should treat as “ground truth”:

* **Transport + protocol**: bidirectional JSON‑RPC 2.0 (with the `"jsonrpc":"2.0"` header omitted on the wire), over **stdio JSONL** (default) or **WebSocket** (experimental). ([OpenAI Developers][1])
* **Handshake**: every connection must do exactly one `initialize` request and then an `initialized` notification *before any other methods*. ([OpenAI Developers][1])
* **Conversation primitives**: **Thread → Turn → Item** with explicit lifecycles; a single user request unfolds into a stream of structured updates (items, deltas, diffs), not a single response. ([OpenAI Developers][1])
* **Events are UI-ready**: the app server translates the internal Codex core event stream into a stable, smaller set of “UI-ready” notifications (`thread/*`, `turn/*`, `item/*`, etc.). ([OpenAI][2])
* **Server-initiated requests**: the server can **pause a turn** and send a JSON‑RPC request when it needs input (approvals, token refresh, etc.). ([OpenAI][2])
* **Approvals**: command execution + file changes may require approval via server requests like `item/commandExecution/requestApproval` and `item/fileChange/requestApproval`, and clients respond with decisions such as `accept`, `acceptForSession`, `decline`, etc. ([OpenAI Developers][1])
* **Useful UI artifacts**: `turn/diff/updated` streams an aggregated unified diff; `turn/plan/updated` streams plan steps + statuses; item deltas stream assistant text and command output. ([OpenAI Developers][1])

Your JS API should basically be: “a stateful event router + ergonomic wrappers + policy hooks + UI bridge.”

---

## Design goals for the JS harness API

1. **Event-native**: handlers for notifications and server requests; no “just call and get a reply” mental model.
2. **Composable orchestration**: harnesses as plugins/middleware that can be stacked (like Koa/Express middleware, but for agent events).
3. **Turn-aware steering**: first-class `turn.start`, `turn.steer`, `turn.interrupt`, plus higher-level loops (`untilPass`, “critic then fix”, etc.). ([OpenAI Developers][1])
4. **Approval policies**: treat approvals as *programmable gates* with optional UI prompts. ([OpenAI Developers][1])
5. **UI as a peer**: harnesses should be able to declare UI panels/actions and react to user interactions (approve, pause, interrupt, retry, etc.).
6. **Goja-friendly**: avoid requiring Node APIs; assume the host injects primitives (`rpc`, `fs`, `ui`, timers).

---

## Proposed API surface

Think of three layers:

1. **Low-level RPC binding (host-provided)**
2. **Codex client wrapper (JS library)**
3. **Harness runtime + UI bridge (JS library)**

### 1) Host primitives (what Go exposes into goja)

These are minimal “kernel” objects you expose from Go:

```js
// Provided by Go host (suggested minimal primitives)
globalThis.__host = {
  rpc: {
    request(method, params) -> Promise<{result}|{error}>
    notify(method, params) -> void
    onNotification(handler) -> unsubscribe
    onRequest(handler) -> unsubscribe   // server-initiated JSON-RPC requests
  },

  ui: {
    emit(event) -> void                 // send UI events to host renderer
    onEvent(handler) -> unsubscribe     // receive UI interactions (buttons, forms)
  },

  clock: { nowMs() -> number, sleep(ms) -> Promise<void> },

  // optional, you said you’ll expose:
  fs: { readFile, writeFile, glob, ... }
  os: { exec, env, ... }  // but note Codex has command/exec too
};
```

### 2) `codex` client wrapper (thin typed/ergonomic layer)

This wraps App Server methods (`thread/start`, `turn/start`, etc.) and normalizes events into a coherent stream.

**Key idea**: Every incoming JSON-RPC message becomes a **CodexEvent** with:

* `kind`: `"notification"` | `"request"` | `"response"` | `"error"`
* `method` and `params`
* best-effort `threadId`, `turnId`, `itemId` (if present)
* plus convenience handles: `thread`, `turn`, `item` (resolved from state)

The wrapper also maintains derived state per thread/turn for UI (latest diff, plan, running items).

```js
// codex.connect() performs initialize + initialized handshake once per connection.
// (Required by app-server) :contentReference[oaicite:10]{index=10}
const session = await codex.connect({
  clientInfo: { name, title, version },
  capabilities: {
    experimentalApi: true,
    optOutNotificationMethods: ["item/agentMessage/delta"], // optional :contentReference[oaicite:11]{index=11}
  }
});

// Common app-server calls (wrappers around JSON-RPC)
const thread = await session.threads.start({ model, cwd, approvalPolicy, sandbox /*...*/ });
const turn = await thread.turn.start({ input: [{type:"text", text:"Run tests"}], /* overrides */ });

await thread.review.start({ delivery: "inline", target: { type: "uncommittedChanges" } }); // :contentReference[oaicite:12]{index=12}

const models = await session.models.list({ limit: 50, includeHidden: false }); // :contentReference[oaicite:13]{index=13}
```

Where these wrappers map directly to documented methods like `turn/start`, `turn/steer`, `turn/interrupt`, and `review/start`. ([OpenAI Developers][1])

### 3) Harness runtime (plugins + policies + UI)

Harnesses are plugins that register handlers and optionally expose UI actions.

#### Harness shape

```js
export default codex.defineHarness({
  name: "my-harness",
  // called once after session is ready
  async onStart(ctx) {},

  // register handlers by JSON-RPC method name
  on: {
    "thread/started": async (ctx, evt) => {},
    "turn/plan/updated": async (ctx, evt) => {},
    "item/agentMessage/delta": async (ctx, evt) => {},

    // server-initiated request: must respond
    "item/commandExecution/requestApproval": async (ctx, req) => {},
    "item/fileChange/requestApproval": async (ctx, req) => {},
  }
});
```

#### Context object (`ctx`)

`ctx` is what makes harness writing pleasant:

```js
ctx = {
  session,               // CodexSession wrapper
  thread, turn, item,    // resolved when possible (null if not applicable)
  state,                 // persistent key/value store (global + per-thread + per-turn)
  ui,                    // UI bridge
  log,                   // structured logging to host
  respond(result|error), // for server requests
  approvals: {           // helpers when handling approval requests
    accept(), acceptForSession(), decline(), cancel(),
    acceptWithExecpolicyAmendment(execpolicy_amendment) // command approvals :contentReference[oaicite:15]{index=15}
  }
}
```

#### Composition model

You want “orchestration-like” composition, so make harnesses stackable:

```js
export default codex.composeHarnesses([
  uiDashboard(),
  autopilotApprovals(),
  tddLoop(),
  autoCompact()
]);
```

Composition rules:

* handlers run in order
* first handler can “consume” an event (stop propagation) or allow others
* for server requests (approvals), enforce “exactly one response” semantics

---

## Harness examples

These are written against the proposed harness API, but every action is ultimately:

* send `turn/start`, `turn/steer`, `turn/interrupt` ([OpenAI Developers][1])
* listen to `turn/*` and `item/*` events ([OpenAI Developers][1])
* respond to approval requests ([OpenAI Developers][1])

### Harness 1: Autopilot approvals (guardrails + “ask when risky”)

Purpose:

* automatically approve safe commands and safe file edits
* prompt user when risky
* record decisions in UI

```js
function autopilotApprovals() {
  const SAFE_PREFIXES = [
    ["go", "test"],
    ["npm", "test"],
    ["pnpm", "test"],
    ["pytest"],
    ["git", "status"],
    ["git", "diff"],
  ];

  const PROTECTED_PATHS = [
    ".env", ".env.local", "secrets/", ".ssh/", "id_rsa"
  ];

  function isSafeCmd(argv = []) {
    return SAFE_PREFIXES.some(prefix =>
      prefix.every((p, i) => argv[i] === p)
    );
  }

  function touchesProtected(changes = []) {
    return changes.some(ch =>
      PROTECTED_PATHS.some(p => ch.path === p || ch.path.startsWith(p))
    );
  }

  return codex.defineHarness({
    name: "autopilot-approvals",

    on: {
      async "item/commandExecution/requestApproval"(ctx, req) {
        // req.params often includes threadId/turnId/itemId + optional command/cwd/reason :contentReference[oaicite:19]{index=19}
        const argv = req.params.command ?? [];
        const reason = req.params.reason ?? "";

        if (isSafeCmd(argv)) {
          ctx.ui.emit({ type: "approval/auto", kind: "command", argv, reason, decision: "acceptForSession" });
          return ctx.respond("acceptForSession"); // allowed decision :contentReference[oaicite:20]{index=20}
        }

        // Ask user
        const choice = await ctx.ui.prompt({
          title: "Approve command?",
          body: `${argv.join(" ")}\n\nReason: ${reason}`,
          buttons: ["accept", "acceptForSession", "decline", "cancel"]
        });

        ctx.ui.emit({ type: "approval/user", kind: "command", argv, decision: choice });
        return ctx.respond(choice);
      },

      async "item/fileChange/requestApproval"(ctx, req) {
        // request includes itemId/threadId/turnId + optional reason/grantRoot :contentReference[oaicite:21]{index=21}
        const item = ctx.item; // should be the pending fileChange item from item/started
        const changes = item?.changes ?? [];

        if (touchesProtected(changes)) {
          ctx.ui.emit({ type: "approval/auto", kind: "fileChange", decision: "decline", changes });
          return ctx.respond("decline");
        }

        // Small diffs can auto-accept, large ones prompt
        const totalLines = changes.reduce((n, ch) => n + (ch.diff?.split("\n").length ?? 0), 0);
        if (totalLines < 120) {
          ctx.ui.emit({ type: "approval/auto", kind: "fileChange", decision: "acceptForSession", totalLines });
          return ctx.respond("acceptForSession");
        }

        const choice = await ctx.ui.prompt({
          title: "Approve file changes?",
          body: `Proposed changes across ${changes.length} file(s). (${totalLines} lines of diff)`,
          buttons: ["accept", "acceptForSession", "decline", "cancel"]
        });

        ctx.ui.emit({ type: "approval/user", kind: "fileChange", decision: choice, totalLines });
        return ctx.respond(choice);
      }
    }
  });
}
```

Why this maps cleanly:

* approvals are explicitly server-initiated requests and must be answered, and the decision vocabulary is defined (`accept`, `acceptForSession`, etc.). ([OpenAI Developers][1])

UI angle:

* approvals show up as a queue/modal, but the harness decides the policy.

---

### Harness 2: Plan-gated execution (steer/interrupt based on plan updates)

Purpose:

* enforce: “show plan first, wait for approval”
* use `turn/plan/updated` as the trigger (and optionally `turn/steer` to continue) ([OpenAI Developers][1])

```js
function planGate() {
  return codex.defineHarness({
    name: "plan-gate",

    on: {
      async "turn/plan/updated"(ctx, evt) {
        // evt.params.plan is a list of {step, status} :contentReference[oaicite:24]{index=24}
        const plan = evt.params.plan ?? [];
        ctx.state.turn(ctx.turn.id).set("latestPlan", plan);

        ctx.ui.emit({ type: "plan/update", threadId: ctx.thread.id, turnId: ctx.turn.id, plan });

        // Only gate once per turn
        if (ctx.state.turn(ctx.turn.id).get("planApproved") !== true) {
          const choice = await ctx.ui.prompt({
            title: "Approve plan to proceed?",
            body: plan.map((p, i) => `${i + 1}. [${p.status}] ${p.step}`).join("\n"),
            buttons: ["approve", "requestChanges", "interrupt"]
          });

          if (choice === "approve") {
            ctx.state.turn(ctx.turn.id).set("planApproved", true);
            // Steer the active turn: append input without new turn :contentReference[oaicite:25]{index=25}
            await ctx.thread.turn.steer({
              expectedTurnId: ctx.turn.id,
              input: [{ type: "text", text: "Plan approved. Proceed." }]
            });
          } else if (choice === "requestChanges") {
            await ctx.thread.turn.steer({
              expectedTurnId: ctx.turn.id,
              input: [{ type: "text", text: "Revise the plan. Do not edit files yet." }]
            });
          } else {
            // Stop the turn :contentReference[oaicite:26]{index=26}
            await ctx.thread.turn.interrupt({ turnId: ctx.turn.id });
          }
        }
      }
    }
  });
}
```

UI angle:

* a plan panel that updates live, and a one-time “Approve plan” modal.

---

### Harness 3: TDD loop (run tests after changes; iterate until green)

Purpose:

* after a turn finishes, run a test command and feed failures back as the next turn
* keeps iterating until passing or max iterations

This one uses `command/exec` to run tests without starting a new agent turn (fast feedback). ([OpenAI Developers][1])

```js
function tddLoop({ testCmd = ["go", "test", "./..."], maxIters = 5 } = {}) {
  return codex.defineHarness({
    name: "tdd-loop",

    on: {
      async "turn/completed"(ctx, evt) {
        const status = evt.params.turn?.status;
        if (status !== "completed") return;

        const iterKey = `tddIters:${ctx.thread.id}`;
        const iter = (ctx.state.thread(ctx.thread.id).get(iterKey) ?? 0) + 1;
        ctx.state.thread(ctx.thread.id).set(iterKey, iter);

        // Optional: only run tests if there was any diff
        const lastDiff = ctx.state.turn(ctx.turn.id).get("latestDiff");
        if (!lastDiff || lastDiff.trim() === "") return;

        ctx.ui.emit({ type: "tdd/testStart", iter, cmd: testCmd });

        // Run tests via app-server sandboxed command/exec :contentReference[oaicite:28]{index=28}
        const res = await ctx.session.command.exec({
          command: testCmd,
          cwd: ctx.thread.cwd,
          sandboxPolicy: { type: "workspaceWrite" },
          timeoutMs: 10 * 60 * 1000
        });

        ctx.ui.emit({
          type: "tdd/testResult",
          iter,
          exitCode: res.exitCode,
          stdout: res.stdout,
          stderr: res.stderr
        });

        if (res.exitCode === 0) {
          ctx.ui.emit({ type: "tdd/green", iter });
          return;
        }

        if (iter >= maxIters) {
          ctx.ui.emit({ type: "tdd/giveUp", iter });
          return;
        }

        // Ask agent to fix failures in a new turn
        await ctx.thread.turn.start({
          input: [{
            type: "text",
            text:
              `Tests failed (iteration ${iter}). Fix them.\n\n` +
              `Command: ${testCmd.join(" ")}\n\n` +
              `STDOUT:\n${res.stdout}\n\nSTDERR:\n${res.stderr}`
          }]
        });
      },

      // Capture aggregated diff updates for gating
      "turn/diff/updated"(ctx, evt) {
        ctx.state.turn(ctx.turn.id).set("latestDiff", evt.params.diff); // unified diff :contentReference[oaicite:29]{index=29}
        ctx.ui.emit({ type: "diff/update", diff: evt.params.diff });
      }
    }
  });
}
```

UI angle:

* a “TDD” panel: iteration count, last test output, green/red status, and a “Stop after this” action.

---

### Harness 4: Critic/reviewer loop (Codex reviewer as an automated gate)

Purpose:

* when agent produces changes, run `review/start` on uncommitted changes
* if review looks bad, ask agent to address review comments in a follow-up turn

Codex reviewer supports inline or detached review threads. ([OpenAI Developers][1])

```js
function reviewGate() {
  return codex.defineHarness({
    name: "review-gate",

    on: {
      async "turn/completed"(ctx, evt) {
        const status = evt.params.turn?.status;
        if (status !== "completed") return;

        // Trigger review on uncommitted changes :contentReference[oaicite:31]{index=31}
        const reviewTurn = await ctx.thread.review.start({
          delivery: "inline",
          target: { type: "uncommittedChanges" }
        });

        ctx.ui.emit({ type: "review/started", reviewTurnId: reviewTurn.id });

        // Wait until we see exitedReviewMode item completed, then read it from state
        // (Your runtime can expose a helper: await ctx.thread.waitForItem(type))
      },

      "item/completed"(ctx, evt) {
        const item = evt.params.item;

        if (item.type === "exitedReviewMode") {
          const reviewText = item.review ?? "";
          ctx.ui.emit({ type: "review/completed", reviewText });

          // Super simple heuristic; you can do better
          const looksBad =
            /critical|security|data loss|broken|doesn't work/i.test(reviewText);

          if (looksBad) {
            ctx.thread.turn.start({
              input: [{
                type: "text",
                text:
                  "Address the reviewer feedback below. Make the minimal safe fixes.\n\n" +
                  reviewText
              }]
            });
          }
        }
      }
    }
  });
}
```

UI angle:

* a “Review” tab that shows entered/exited review mode output; the doc explicitly recommends using the `exitedReviewMode` notification/item to render reviewer output. ([OpenAI Developers][1])

---

### Harness 5: RLM-ish recursive decomposition (manager/worker with subthreads)

This is the “recursive LM / orchestration” style: treat the coding task as a tree; spawn sub-agents for subtasks; merge guidance back to the main agent.

Mechanics:

* use `outputSchema` on an initial “planner” turn to get structured subtasks ([OpenAI Developers][1])
* spawn subthreads in **read-only** mode to generate guidance/patch suggestions (or just “how to implement”)
* feed aggregated results into the main thread as context, then let the main agent implement

```js
function rlmDecompose({ maxDepth = 2, maxWorkers = 4 } = {}) {
  async function plan(ctx, goalText) {
    // Ask for a structured plan using outputSchema :contentReference[oaicite:34]{index=34}
    const turn = await ctx.thread.turn.start({
      input: [{ type: "text", text: `Decompose into implementable subtasks:\n${goalText}` }],
      outputSchema: {
        type: "object",
        properties: {
          subtasks: {
            type: "array",
            items: {
              type: "object",
              properties: {
                title: { type: "string" },
                prompt: { type: "string" }
              },
              required: ["title", "prompt"],
              additionalProperties: false
            }
          }
        },
        required: ["subtasks"],
        additionalProperties: false
      }
    });

    return turn; // runtime can parse the final structured output from completed items
  }

  async function runWorker(ctx, { title, prompt }) {
    // Start a separate thread for sub-agent work
    const workerThread = await ctx.session.threads.start({
      model: ctx.thread.model,
      cwd: ctx.thread.cwd,
      approvalPolicy: "never",
      sandbox: "readOnly" // keep it analysis-only
    });

    const workerTurn = await workerThread.turn.start({
      input: [{
        type: "text",
        text:
          `You are a sub-agent. Produce concrete implementation guidance.\n` +
          `Task: ${title}\n\n${prompt}\n\n` +
          `Return:\n- key files\n- suggested diff snippets\n- pitfalls\n`
      }]
    });

    const summary = await workerTurn.waitForFinalAgentMessage(); // helper your runtime provides
    return { title, summary };
  }

  return codex.defineHarness({
    name: "rlm-decompose",

    on: {
      async "ui/submitPrompt"(ctx, uiEvt) {
        const goal = uiEvt.text;

        // 1) get subtasks
        ctx.ui.emit({ type: "rlm/planning", goal });
        const planTurn = await plan(ctx, goal);

        const subtasks = await planTurn.getStructuredOutput().subtasks;

        ctx.ui.emit({ type: "rlm/subtasks", subtasks });

        // 2) run workers (bounded concurrency)
        const results = [];
        for (const batch of chunk(subtasks, maxWorkers)) {
          const batchResults = await Promise.all(batch.map(st => runWorker(ctx, st)));
          results.push(...batchResults);
          ctx.ui.emit({ type: "rlm/workerBatchDone", results: batchResults });
        }

        // 3) feed results back to main agent and implement
        const packed = results.map(r => `## ${r.title}\n${r.summary}`).join("\n\n");
        await ctx.thread.turn.start({
          input: [{
            type: "text",
            text:
              `Implement the feature now. Here is sub-agent guidance:\n\n${packed}\n\n` +
              `Proceed step-by-step and run tests.`
          }]
        });
      }
    }
  });
}
```

UI angle:

* a “task graph” view: subtasks as nodes, each worker thread as a lane, status updates as events.

This is the pattern that feels most “RLM-ish”: recursive decomposition + spawning specialized calls + merging back.

---

### Harness 6: Auto-compaction when token usage grows (keep context healthy)

The app-server emits `thread/tokenUsage/updated` and supports `thread/compact/start`, with compaction progress streaming as normal turn/item events and a `contextCompaction` item lifecycle. ([OpenAI Developers][1])

```js
function autoCompact({ softLimit = 0.8 } = {}) {
  return codex.defineHarness({
    name: "auto-compact",

    on: {
      async "thread/tokenUsage/updated"(ctx, evt) {
        const usage = evt.params; // depends on schema; treat as opaque
        const ratio = usage.ratio ?? 0;

        ctx.ui.emit({ type: "tokens/update", usage });

        if (ratio > softLimit && !ctx.state.thread(ctx.thread.id).get("compactionRequested")) {
          ctx.state.thread(ctx.thread.id).set("compactionRequested", true);
          ctx.ui.emit({ type: "tokens/compactStart", ratio });

          await ctx.thread.compact.start(); // wrapper around thread/compact/start :contentReference[oaicite:36]{index=36}
        }
      }
    }
  });
}
```

UI angle:

* token meter + “compact now” button + compaction progress item.

---

## UI design: what your JS API should make easy

Because the App Server stream is already structured into primitives and item lifecycles, you can build UIs as *projections of the event stream*. ([OpenAI][2])

### Suggested UI primitives to expose to JS

Keep it minimal and composable:

1. **Panels**

   * `ui.panel(id, {title, kind})`
   * `panel.setMarkdown(text)` / `panel.setText(text)` / `panel.setJSON(obj)`
2. **Timeline**

   * `ui.timeline.append({threadId, turnId, type, title, status, payload})`
3. **Approvals**

   * `ui.prompt(...)` returning a Promise, used in approval handlers
4. **Actions / command palette**

   * `ui.actions.register({id, title, shortcut}, handler)`
   * actions like: interrupt, rollback, run review, run tests, compact, fork thread, export log
5. **Thread list**

   * render from `thread/list` and `thread/read` for history UI ([OpenAI Developers][1])
6. **Diff + Plan views**

   * Diff from `turn/diff/updated` (aggregated unified diff) ([OpenAI Developers][1])
   * Plan from `turn/plan/updated` ([OpenAI Developers][1])
7. **Streaming chat**

   * Use `item/agentMessage/delta` to append partial assistant text, finalize on `item/completed` ([OpenAI Developers][1])
8. **Command output console**

   * Stream with `item/commandExecution/outputDelta`, finalize on `item/completed` ([OpenAI Developers][1])

### UI control patterns enabled by this API

* **“Pause at plan”**: plan-gate harness prompts user on `turn/plan/updated`, then `turn/steer` to continue or `turn/interrupt` to stop. ([OpenAI Developers][1])
* **Approval queue**: server requests pause the turn; your UI shows a modal/queue; harness responds with the decision vocabulary. ([OpenAI][2])
* **Agent steering as UI action**: “Add constraint”, “Focus tests”, “Stop editing files” => `turn/steer`. ([OpenAI Developers][1])
* **Fork for experimentation**: allow user to “Try alternative approach” via `thread/fork` and show both lanes. ([OpenAI Developers][1])
* **Review lane**: `review/start` inline or detached; show reviewer output as entered/exited review mode items. ([OpenAI Developers][1])

---

## A few implementation notes that should influence the JS API shape

These are worth baking into the runtime because they directly affect harness ergonomics:

* **Backpressure / overload**: in WebSocket mode, the server can reject requests with `-32001` “Server overloaded; retry later.” Your runtime should centralize retry w/ exponential backoff so harness authors don’t reinvent it. ([OpenAI Developers][1])
* **Notification suppression**: allow harness authors (or UI configs) to opt out of high-volume notifications with `initialize.params.capabilities.optOutNotificationMethods`. ([OpenAI Developers][1])
* **State reconstruction**: because threads are persisted and list/read APIs exist, you can rebuild UI state on reconnect by replaying history (`thread/read includeTurns`). ([OpenAI Developers][1])

---

## Summary of the API “core” you should implement

If you implement only ~8 things well, harness authors can build almost anything:

1. `codex.connect({clientInfo, capabilities})` that performs initialize handshake. ([OpenAI Developers][1])
2. `session.rpc` passthrough + message router (notification vs request). ([OpenAI Developers][1])
3. A `Thread` / `Turn` object model with helpers for `turn/start`, `turn/steer`, `turn/interrupt`, `review/start`. ([OpenAI Developers][1])
4. Event subscription by method name (`"turn/diff/updated"`, `"item/agentMessage/delta"`, etc.). ([OpenAI Developers][1])
5. First-class approval handling helpers (respond exactly once, with decision vocabulary). ([OpenAI Developers][1])
6. A small persistent store (`state.thread(id)`, `state.turn(id)`) for orchestration/UI.
7. Harness composition (`composeHarnesses`, middleware-ish).
8. A minimal UI bridge (`ui.emit`, `ui.prompt`, `ui.actions.register`).

With those, the harness examples above (autopilot approvals, plan gating, TDD loop, review gate, RLM decomposition, compaction) are all straightforward.

If you want, I can also sketch a **canonical event schema** for `CodexEvent` and a **default UI layout** (thread list + timeline + diff/plan/console panes) that your Go host can render, but the core harness API above is the foundation.

[1]: https://developers.openai.com/codex/app-server/ "https://developers.openai.com/codex/app-server/"
[2]: https://openai.com/index/unlocking-the-codex-harness/ "https://openai.com/index/unlocking-the-codex-harness/"
