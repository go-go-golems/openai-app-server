---
Title: JS Harness Session Resume Architecture Analysis
Ticket: OAP-03-RESUMING-SESSIONS
Status: active
Topics:
    - goja
    - openai-app-server
    - codex
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: openai-app-server/cmd/openai-app-server/harness_run_command.go
      Note: Current harness CLI entrypoint and runtime bootstrap
    - Path: openai-app-server/pkg/codexrpc/client.go
      Note: |-
        Handshake state machine and transport-bound RPC routing
        Handshake and transport-bound request routing relevant to reconnect/resume
    - Path: openai-app-server/pkg/js/module_codex.go
      Note: |-
        Current minimal codex module API (`connect`, handlers, request helpers)
        Codex JS API surface that will gain resume helpers
    - Path: openai-app-server/pkg/js/runtime.go
      Note: |-
        Runtime event dispatch and JS callback registration
        Runtime event dispatch and callback registry
    - Path: openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/sources/local/01-app-server-js.md
      Note: |-
        Imported target API document (includes reconnect/state reconstruction guidance)
        Imported requirements baseline including state reconstruction hints
ExternalSources:
    - local:01-app-server-js.md
Summary: Deep design analysis for resume-capable JS harness sessions, including API options, state-recovery models, tradeoffs, and phased implementation plan.
LastUpdated: 2026-02-23T11:50:00-05:00
WhatFor: Define how JS harnesses should resume remote codex threads and local harness state safely and predictably.
WhenToUse: Use before implementing resume flags/APIs in openai-app-server harness runtime and codex JS module.
---


# JS Harness Session Resume Architecture Analysis

## 1. Executive Summary

The current JS harness runtime can start fresh sessions and stream events, but it has no first-class notion of resume. Today, if the CLI process exits, the JS script loses all in-memory state (event counters, active turn tracking, approval context, UI projections), even when remote threads still exist server-side.

A robust resume design needs two distinct but coordinated capabilities:

1. Remote conversation resume: reconnect to an existing Codex App Server thread and continue turns there.
2. Local harness resume: reconstruct harness-internal state so behavior after restart is consistent with behavior before restart.

The strongest practical design for this codebase is a hybrid strategy:

- Protocol-driven hydration from `thread/read` (and optional `thread/list`) to rebuild durable conversation truth.
- Local checkpoint + event journal to restore harness operational state.
- Deterministic "resume anchor" algorithm that decides where execution continues (last completed turn, in-progress turn, or explicit user override).

This analysis details multiple technical options, then proposes a phased implementation that is additive to the current OAP-01/OAP-02 code.

## 2. Current Target Harness Reality

## 2.1 What exists now

From current implementation:

- CLI: `harness run` loads one JS script and wires `codexrpc.Client` to the goja runtime.
- Handshake: `initialize -> initialized` is done before script execution.
- JS modules: `codex`, `rpc`, `ui`, `clock`, `approval`.
- Event flow: notifications/requests are forwarded to JS callbacks in-process only.

Current behavior is intentionally minimal and stateless:

- `codex.connect()` returns a basic session object with `request/notify/respond` wrappers.
- No persisted session metadata across process runs.
- No persisted event sequence or checkpoint cursor.
- No helper APIs for `thread/read`, `resume`, or recovery decisions.

## 2.2 Why this matters for resume

If the harness process crashes or exits:

- Remote thread may continue to exist and be listable/readable.
- Local harness state disappears.
- Re-running the same script creates ambiguity:
  - Should it continue an existing thread?
  - Should it replay state?
  - Should it start a new thread?

Without explicit resume semantics, scripts either duplicate work or diverge from expected automation.

## 3. Resume Problem Decomposition

Resume is not one problem. It is four subproblems:

1. Session identity: which remote thread/session are we resuming?
2. Hydration: what remote data do we reload (`thread/read includeTurns` etc.)?
3. Local reconstruction: how do we rebuild harness-specific state (approval decisions, wait gates, per-turn projections)?
4. Continuation policy: do we continue, restart latest turn, or fork a new thread branch?

## 3.1 Vocabulary used below

- Resume target: remote session identity candidate (usually a `threadId`).
- Resume anchor: exact point from which harness continues execution.
- Hydration: fetching persisted remote truth and building local projections.
- Checkpoint: persisted local state snapshot written by harness runtime.
- Event journal: append-only local stream of runtime-observed events.

## 4. Solution Space: Technical Families

Below are the major solution families. They are intentionally broad to cover "all possible" practical implementation styles in this codebase.

## 4.1 Family A: Stateless Protocol Resume

Concept:

- Do not persist local harness state.
- On startup, pick a `threadId` and call `thread/read` with turn inclusion.
- Reconstruct only what can be derived from remote thread history.

How it works:

1. Resolve `threadId` from CLI flag or JS config.
2. Call `thread/read`.
3. Build derived state in memory (latest turn, plan/diff snapshots, etc.).
4. Continue with `turn/start` or `turn/steer`.

Pros:

- Minimal engineering effort.
- No local persistence corruption risk.
- Easy to reason about.

Cons:

- Loses purely local state (UI panel collapse state, debounce counters, custom middleware caches).
- Cannot recover transient approvals not reflected in durable server state.
- No crash-safe recovery for in-flight local workflows.

Best fit:

- Early phase implementation.
- Simple scripts where remote thread is source of truth.

## 4.2 Family B: Snapshot Checkpoint Resume

Concept:

- Periodically persist a compact local checkpoint.
- On restart, load checkpoint first, then hydrate deltas from server.

Example checkpoint payload:

```json
{
  "version": 1,
  "workspace": "/home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server",
  "script": "scripts/05-module-api-turn-completed-gate.js",
  "threadId": "thread_abc",
  "lastTurnId": "turn_123",
  "lastKnownTurnStatus": "completed",
  "seenMethods": ["thread/started", "turn/started", "turn/completed"],
  "gates": {
    "waitForTurnCompleted": true
  },
  "updatedAt": "2026-02-23T11:45:12Z"
}
```

Pros:

- Fast startup.
- Captures harness operational context.
- Easy to store as JSON file.

Cons:

- Snapshots can be stale.
- Requires merge logic with remote truth.
- Versioning/migration complexity over time.

Best fit:

- Medium complexity harnesses.
- CLI-centric workflows where fast recovery matters.

## 4.3 Family C: Event-Sourced Resume

Concept:

- Persist every inbound/outbound event in an append-only local journal.
- Rebuild state by replaying journal deterministically.
- Reconcile with remote thread history at startup.

Journal record example:

```json
{
  "seq": 1042,
  "ts": "2026-02-23T11:44:31.124Z",
  "direction": "inbound-notification",
  "method": "turn/completed",
  "threadId": "thread_abc",
  "turnId": "turn_123",
  "payload": {"turn": {"id": "turn_123"}}
}
```

Pros:

- Highest fidelity reconstruction.
- Debuggable timeline (excellent for incident analysis).
- Supports advanced "time-travel" replay and deterministic tests.

Cons:

- Highest implementation cost.
- Storage growth/rotation required.
- Reconciliation logic is more complex.

Best fit:

- Long-lived orchestrators.
- Reliability-critical automation.

## 4.4 Family D: Server-Managed Resume Token

Concept:

- Server provides opaque resume token/cursor.
- Client reconnects using token and receives replayed stream from server.

Pros:

- Simplifies local persistence.
- Best source-of-truth alignment.

Cons:

- Depends on upstream protocol capability.
- Not currently available in local openai-app-server harness code.

Best fit:

- Future protocol extension.

## 4.5 Family E: Hybrid (Recommended)

Concept:

- Use remote thread history as truth for conversation state.
- Persist local checkpoints and compact event journal for operational state.
- Resume via deterministic anchor algorithm.

Pros:

- Good reliability/cost balance.
- Works with existing protocol methods.
- Supports gradual rollout.

Cons:

- More moving parts than stateless approach.

Best fit:

- This repository, now.

## 5. Where Exactly Should Harness Resume?

The key design question from your prompt is: "How can the harness figure out where to resume its internal state?"

Answer: treat resume as an anchor-selection algorithm with explicit policy.

## 5.1 Resume anchor candidates

Given remote + local data, candidate anchors are:

1. Latest completed turn anchor
- Continue from last turn with status completed.
- Safe default for most scripts.

2. In-progress turn anchor
- If latest turn appears active/incomplete, either:
  - continue listening and steer, or
  - interrupt and start a follow-up turn.

3. Explicit turn anchor
- User or script chooses a specific historical turn.
- Useful for forensic replay or branching.

4. Fork anchor
- Start a new thread seeded from selected historical context.
- Useful when resuming old context but preserving audit separation.

## 5.2 Anchor resolution algorithm

```ts
function resolveResumeAnchor(input: {
  policy: "latest-completed" | "continue-active" | "explicit-turn" | "fork";
  requestedThreadId?: string;
  requestedTurnId?: string;
  localCheckpoint?: Checkpoint;
  remoteThread?: ThreadReadResult;
}): ResumeAnchor {
  const thread = input.remoteThread;
  if (!thread) throw new Error("remote thread missing");

  if (input.policy === "explicit-turn") {
    const turn = thread.turns.find(t => t.id === input.requestedTurnId);
    if (!turn) throw new Error("requested turn not found");
    return { type: "explicit", threadId: thread.id, turnId: turn.id };
  }

  const latest = thread.turns[thread.turns.length - 1];

  if (input.policy === "continue-active") {
    if (latest && latest.status !== "completed") {
      return { type: "active", threadId: thread.id, turnId: latest.id };
    }
    return { type: "completed", threadId: thread.id, turnId: latest?.id ?? "" };
  }

  if (input.policy === "fork") {
    const base = input.requestedTurnId
      ? thread.turns.find(t => t.id === input.requestedTurnId)
      : latest;
    if (!base) throw new Error("fork base turn not found");
    return { type: "fork", sourceThreadId: thread.id, baseTurnId: base.id };
  }

  // default: latest-completed
  for (let i = thread.turns.length - 1; i >= 0; i -= 1) {
    if (thread.turns[i].status === "completed") {
      return { type: "completed", threadId: thread.id, turnId: thread.turns[i].id };
    }
  }

  return { type: "start-new-turn", threadId: thread.id };
}
```

## 5.3 Internal state reconstruction layers

Use layered reconstruction to avoid over-coupling:

1. Durable remote layer
- Thread metadata, turns, status, plan, diff, item history.

2. Durable local layer
- Checkpoint metadata: last handled request ID, gate statuses, selected model/cwd, script fingerprint.

3. Ephemeral rebuild layer
- Recompute transient projections (method counters, active item map, UI panels) from remote + journal.

4. Live stream layer
- Subscribe and continue consuming notifications/requests.

## 6. Proposed JS API Extensions

The current `codex.connect()` API is too thin for resume orchestration. Below is a concrete signature proposal.

## 6.1 `codex.connect` with resume options

```ts
declare module "codex" {
  export type ResumePolicy =
    | "never"
    | "auto"
    | "latest-completed"
    | "continue-active"
    | "explicit-turn"
    | "fork";

  export interface ConnectOptions {
    threadId?: string;
    resume?: {
      policy?: ResumePolicy;
      turnId?: string;
      checkpointKey?: string;
      hydrateRemote?: boolean;      // default true
      hydrateIncludeTurns?: boolean; // default true
      failIfMissing?: boolean;      // default false
    };
  }

  export interface CodexSession {
    connected: boolean;
    request(method: string, params?: any): Promise<any>;
    notify(method: string, params?: any): void;
    onNotification(cb: (evt: RPCEvent) => void): () => void;
    onRequest(cb: (evt: RPCEvent) => void): () => void;

    // New
    resume(options?: ConnectOptions["resume"]): Promise<ResumeResult>;
    checkpoint(): Promise<void>;
    getState(): SessionProjection;
  }
}
```

## 6.2 `ResumeResult` shape

```ts
interface ResumeResult {
  resumed: boolean;
  threadId: string;
  anchor: {
    type: "completed" | "active" | "explicit" | "fork" | "start-new-turn";
    turnId?: string;
    sourceThreadId?: string;
    baseTurnId?: string;
  };
  hydration: {
    remoteTurnsLoaded: number;
    localCheckpointLoaded: boolean;
    journalEventsReplayed: number;
  };
  warnings: string[];
}
```

## 6.3 Harness-state helper module (optional but powerful)

Introduce a dedicated `harnessState` JS module:

```ts
declare module "harnessState" {
  export function load(key: string): Promise<any | null>;
  export function save(key: string, state: any): Promise<void>;
  export function appendEvent(key: string, event: any): Promise<void>;
  export function rotate(key: string, maxEvents: number): Promise<void>;
}
```

This cleanly separates codex RPC concerns from local persistence concerns.

## 7. Proposed CLI Surface for Resume

`harness run` should expose explicit resume controls.

Example flags:

- `--resume auto|never|latest-completed|continue-active|explicit-turn|fork`
- `--resume-thread-id <thread_id>`
- `--resume-turn-id <turn_id>`
- `--resume-checkpoint-key <string>`
- `--resume-fail-if-missing`
- `--checkpoint-dir <path>`

Example command:

```bash
go run ./cmd/openai-app-server harness run \
  --script ttmp/.../scripts/my-harness.js \
  --resume auto \
  --resume-checkpoint-key oap03-default \
  --resume-fail-if-missing
```

## 8. Runtime Architecture Changes (Go Side)

## 8.1 New components

Suggested package additions:

```text
pkg/harnessstate/
  store.go          // load/save checkpoint JSON
  journal.go        // append/read compact event journal
  model.go          // checkpoint/event structs + schema versioning
  reconcile.go      // merge local + remote hydration
```

Existing files impacted:

- `cmd/openai-app-server/harness_run_command.go`
- `pkg/js/module_codex.go`
- `pkg/js/runtime.go`
- optional: `pkg/codexrpc/threads.go` (add `ThreadRead` helper)

## 8.2 Resume lifecycle diagram

```text
+------------------------+
| harness run starts     |
+-----------+------------+
            |
            v
+------------------------+
| handshake (initialize) |
+-----------+------------+
            |
            v
+------------------------+
| resolve resume target  |
| (flags/checkpoint/js)  |
+-----------+------------+
            |
            v
+------------------------+
| hydrate remote thread  |
| thread/read/list       |
+-----------+------------+
            |
            v
+------------------------+
| load checkpoint/journal|
| reconcile + pick anchor|
+-----------+------------+
            |
            v
+------------------------+
| emit resume summary    |
| to JS + ui.emit        |
+-----------+------------+
            |
            v
+------------------------+
| run live event loop    |
| periodic checkpoint    |
+------------------------+
```

## 8.3 Internal state machine

```text
NEW -> CONNECTED -> HYDRATING -> RESUMED -> RUNNING -> CHECKPOINTING -> RUNNING
                                  |                         |
                                  +-> FAILED  <-------------+
```

## 9. Data Modeling for Safe Resume

## 9.1 Checkpoint schema v1

```ts
interface HarnessCheckpointV1 {
  schemaVersion: 1;
  key: string;
  scriptPath: string;
  scriptSha256: string;
  workspaceRoot: string;

  thread: {
    id: string;
    lastKnownTurnId?: string;
    lastKnownTurnStatus?: string;
  };

  runtime: {
    lastInboundSeq: number;
    lastRequestIdHandled?: string;
    gateState: Record<string, any>;
    counters: Record<string, number>;
  };

  updatedAt: string;
}
```

## 9.2 Event journal record

```ts
interface HarnessEventRecordV1 {
  schemaVersion: 1;
  seq: number;
  ts: string;
  direction: "inbound-notification" | "inbound-request" | "outbound-response";
  method?: string;
  id?: string | number;
  threadId?: string;
  turnId?: string;
  payload: any;
}
```

## 9.3 Why both checkpoint and journal

- Checkpoint gives quick startup.
- Journal gives deterministic rebuild and debugging.
- Journal can be compacted into checkpoint periodically.

## 10. JS Harness Patterns With Resume

## 10.1 Example: resume-aware approval harness

```js
const codex = require("codex");
const ui = require("ui");
const approval = require("approval");

const session = codex.connect({
  resume: {
    policy: "auto",
    checkpointKey: "approval-autopilot-main",
    hydrateRemote: true,
    hydrateIncludeTurns: true
  }
});

session.resume().then((info) => {
  ui.emit({ type: "resume-summary", ok: true, info });

  session.onRequest((evt) => {
    if (evt.method === "item/commandExecution/requestApproval") {
      approval.acceptForSession(evt.id);
    }
  });

  return session.checkpoint();
}).catch((err) => {
  ui.emit({ type: "resume-summary", ok: false, error: String(err) });
});
```

## 10.2 Example: explicit fork resume

```js
const session = codex.connect({
  threadId: "thread_old_123",
  resume: {
    policy: "fork",
    turnId: "turn_checkpoint_42"
  }
});

const result = await session.resume();
// result.anchor.type === "fork"
// result.anchor.sourceThreadId === "thread_old_123"
// subsequent turn/start happens on new thread
```

## 10.3 Example: deterministic wait-gate recovery

```js
function restoreTurnGateState(projection) {
  const gate = projection.gates && projection.gates.turnCompleted;
  return !!gate;
}

const r = await session.resume({ policy: "latest-completed" });
const state = session.getState();
const wasCompleted = restoreTurnGateState(state);
if (!wasCompleted) {
  // continue observing notifications until matching completion
}
```

## 11. Merge/Reconciliation Strategy

When local and remote disagree, the harness needs deterministic rules.

Recommended precedence:

1. Remote thread existence and turn history always win.
2. Remote turn status wins over local cached status.
3. Local journal supplements missing operational state only.
4. If mismatch is severe (e.g., checkpoint thread missing), emit warning and fall back to new thread unless `failIfMissing=true`.

Pseudocode:

```ts
function reconcile(local: Checkpoint | null, remote: ThreadReadResult | null, opts: ResumeOptions) {
  if (!remote) {
    if (opts.failIfMissing) throw new Error("resume target missing");
    return { action: "start-fresh", warnings: ["remote thread not found"] };
  }

  if (!local) {
    return { action: "hydrate-remote-only", warnings: [] };
  }

  if (local.thread.id !== remote.id) {
    return {
      action: "hydrate-remote-only",
      warnings: ["checkpoint thread mismatch; ignoring local thread pointer"]
    };
  }

  return {
    action: "merge",
    warnings: [],
    merged: {
      threadId: remote.id,
      lastTurnId: pickMostReliableTurnId(local, remote),
      gateState: local.runtime.gateState
    }
  };
}
```

## 12. Edge Cases and Failure Modes

## 12.1 In-flight approval request during crash

Issue:

- Harness crashed after receiving `requestApproval` but before responding.

Strategy:

- On resume, inspect latest remote turn/item status.
- If approval is still pending and request is replayed, handle normally.
- If request is gone, mark local pending approval as stale and continue.

## 12.2 In-progress turn with unknown completion

Issue:

- Last local state says active turn, but remote status may already be completed/cancelled.

Strategy:

- Treat remote as truth.
- Emit reconciliation event to UI: `resume/reconciled-turn-status`.

## 12.3 Script changed between runs

Issue:

- New script version may expect different checkpoint schema.

Strategy:

- Store `scriptSha256` + `schemaVersion`.
- If mismatch, either migrate checkpoint or soft-reset local state.

## 12.4 Multi-harness contention

Issue:

- Two harness processes resume same thread concurrently.

Strategy:

- Optional local lockfile per checkpoint key.
- Optional process identity metadata in checkpoint.
- Detect and warn rather than silently racing.

## 13. Security and Safety Considerations

- Checkpoint path must stay within workspace by default.
- Avoid persisting raw secrets from event payloads where possible.
- Support redaction hooks for checkpoint/journal payloads.
- Avoid automatic replay of privileged decisions without policy checks.

Suggested redaction example:

```ts
function redact(event) {
  if (event.method === "item/commandExecution/requestApproval") {
    delete event.payload?.environment;
  }
  return event;
}
```

## 14. Testing Strategy

## 14.1 Unit tests

- Anchor selection algorithm with synthetic thread histories.
- Checkpoint load/save schema compatibility.
- Reconciliation conflict handling.

## 14.2 Integration tests (memory transport)

- Start run -> checkpoint -> restart run -> resume same thread.
- Simulate in-progress turn before restart.
- Simulate missing remote thread and `failIfMissing` behaviors.

## 14.3 Chaos/failure tests

- Crash between inbound request and response.
- Crash during checkpoint write (verify atomic file replace).
- Corrupted checkpoint file fallback behavior.

## 14.4 Playbook-level live probes

Add OAP-03 scripts/playbooks analogous to OAP-01 probes:

- `01-resume-latest-completed.js`
- `02-resume-continue-active.js`
- `03-resume-fork-turn.js`
- `04-resume-corrupt-checkpoint.js`

## 15. Implementation Plan (Phased)

## Phase 1: Minimal resume API (fast path)

- Add CLI flags: `--resume`, `--resume-thread-id`.
- Add JS `session.resume()` that hydrates remote only.
- Add `codexrpc.ThreadRead` helper.

Outcome:

- Reliable stateless resume via remote thread history.

## Phase 2: Checkpoint persistence

- Add checkpoint store (`pkg/harnessstate/store.go`).
- Add periodic/exit checkpoint writing.
- Add `--resume-checkpoint-key`.

Outcome:

- Process restarts preserve operational state.

## Phase 3: Event journal + reconciliation

- Add compact local event journal.
- Replay + merge with remote hydration.
- Add diagnostics events and UI summaries.

Outcome:

- Deterministic high-fidelity resume.

## Phase 4: Advanced branching + locks

- Add fork/explicit-turn policies.
- Add local lockfile and contention detection.

Outcome:

- Safe multi-operator workflows and forensic replay.

## 16. Proposed Concrete API Signatures (Go)

```go
// pkg/codexrpc/threads.go
func (c *Client) ThreadRead(ctx context.Context, threadID string, includeTurns bool) (*ThreadReadResult, error)

// pkg/harnessstate/store.go
type Store interface {
    Load(ctx context.Context, key string) (*Checkpoint, error)
    Save(ctx context.Context, key string, cp *Checkpoint) error
}

// pkg/js/module_codex.go (session object methods)
// session.resume(options?) -> Promise<ResumeResult>
// session.checkpoint() -> Promise<void>
// session.getState() -> SessionProjection
```

## 17. Recommended Path for OAP-03

Recommended implementation order for this repository:

1. Build Phase 1 first: remote-only resume (`thread/read`-driven).
2. Add Phase 2 checkpointing quickly after, since harness scripts already maintain local gate state arrays/counters.
3. Defer full event sourcing (Phase 3) until there is evidence that checkpoint-only is insufficient.

Reason:

- It delivers immediate user value with low risk.
- It avoids blocking on speculative complexity.
- It aligns with current lightweight module design and ongoing implementation velocity.

## 18. Final Decision Framework

When choosing resume mode per harness, use this table:

| Harness type | Recommended mode | Why |
|---|---|---|
| Simple one-shot scripts | Stateless protocol resume | Minimal overhead |
| Long-running operator harness | Hybrid checkpoint resume | Better continuity |
| Compliance/audit-heavy workflows | Event-sourced resume | Full traceability |
| Future protocol with replay tokens | Server-managed token + local checkpoint | Strongest correctness |

## 19. Open Questions

1. Which exact `thread/read` response fields are stable enough to anchor on (status names, item schemas)?
2. Should checkpoint writes be timer-based, event-count-based, or both?
3. Do we require resume lockfiles from day one, or only after multi-process contention is observed?
4. Should `harness run` default to `--resume auto` or remain `never` for conservative rollout?

## 20. Conclusion

Session resuming for the JS harness should be designed as an explicit, observable contract, not an implicit convenience. The right model is: remote thread history as truth, local checkpoint/journal for operational continuity, and deterministic anchor selection that explains exactly where and why execution resumed.

This gives developers and operators predictable behavior after restarts, while preserving room for future protocol-native replay support.
