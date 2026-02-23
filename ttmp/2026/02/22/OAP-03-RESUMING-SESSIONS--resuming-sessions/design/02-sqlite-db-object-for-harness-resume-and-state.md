---
Title: SQLite DB Object for Harness Resume and State
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
      Note: |-
        Harness CLI command where db path, migrations, and locking flags should be added
        Harness run command where db flags
    - Path: openai-app-server/pkg/js/module_codex.go
      Note: |-
        Codex JS API where resume helpers can consume DB-backed checkpoints
        Codex session helpers that can coordinate with DB-backed resume
    - Path: openai-app-server/pkg/js/runtime.go
      Note: |-
        Runtime host where a DB bridge can be injected alongside RPC/UI bridges
        Runtime host for injecting DB bridge and module registration
    - Path: openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/sources/local/01-app-server-js.md
      Note: |-
        Source document with harness examples adapted here to persistent SQLite-backed state
        Source harness examples adapted in this document
    - Path: openai-app-server/ttmp/2026/02/22/OAP-03-RESUMING-SESSIONS--resuming-sessions/design/01-js-harness-session-resume-architecture-analysis.md
      Note: |-
        First OAP-03 resume architecture document; this file extends it with DB-native design
        Companion first analysis for non-DB and hybrid resume approaches
ExternalSources:
    - local:01-app-server-js.md
Summary: In-depth design for passing a SQLite db object into JS harness scripts for resumability, workflow persistence, and durable operational state across restarts.
LastUpdated: 2026-02-23T12:30:00-05:00
WhatFor: Define a concrete SQLite-first persistence model and JS API for harness resume and data retention needs.
WhenToUse: Use when implementing DB-backed harness runtime, resume logic, and persistent orchestration patterns.
---


# SQLite DB Object for Harness Resume and State

## 1. Executive Summary

If we pass harness scripts a writable SQLite `db` object, resumability becomes much easier and much more deterministic.

Instead of trying to infer all internal harness state only from remote `thread/read`, scripts can persist their own workflow state:

- what they already approved
- what plan step was gated
- what test iteration they reached
- what review findings were pending
- what worker subtask was still running
- what compaction cooldown was active

The practical result is that a restarted harness can pick up exactly where it left off, not just where the remote thread happened to be.

This document proposes:

1. A host-provided JS `db` API built on SQLite.
2. Schema patterns for resume + general data keeping.
3. Adapted versions of all major harness examples from `01-app-server-js.md`.
4. Crash-safety, migration, and lock strategies.
5. Concrete implementation signatures for `openai-app-server`.

## 2. Why a DB Object Is Different From Plain Checkpoint Files

Checkpoint JSON files are useful, but a DB gives five major advantages:

1. Atomic updates with transactions.
2. Indexed queries for partial state reconstruction.
3. Event history plus snapshots in one place.
4. Multi-entity consistency (turn + approval + gate + metrics).
5. Better debugability and ad hoc inspection.

With SQLite WAL mode, durability and concurrency are strong enough for local harness workflows without adding network dependencies.

## 3. Proposed API Shape for `db` in JS Harness

## 3.1 Module shape

Expose a new native module:

```js
const db = require("db");
```

Minimal API:

```ts
declare module "db" {
  type Scalar = string | number | boolean | null;
  type Params = Scalar[] | Record<string, Scalar>;

  interface ExecResult {
    changes: number;
    lastInsertRowid?: number;
  }

  function exec(sql: string, params?: Params): Promise<ExecResult>;
  function queryOne<T = any>(sql: string, params?: Params): Promise<T | null>;
  function queryAll<T = any>(sql: string, params?: Params): Promise<T[]>;

  // Transaction boundary; commits on success, rollbacks on throw.
  function tx<T>(fn: () => Promise<T> | T): Promise<T>;

  // Optional helpers
  function pragma(name: string, value?: string | number): Promise<any>;
  function close(): Promise<void>;
}
```

## 3.2 Why Promise-based API

The rest of harness APIs (`rpc.request`, clock sleeps, UI prompts) are async. Keeping DB calls Promise-based prevents runtime design splits and supports background operations without blocking the event loop.

## 3.3 Optional typed repository helper on top of raw SQL

Raw SQL is powerful but repetitive. Provide a lightweight JS helper library:

```js
const store = {
  async upsertCheckpoint(cp) {
    await db.exec(
      `
      INSERT INTO harness_checkpoints (checkpoint_key, thread_id, turn_id, payload_json, updated_at)
      VALUES (?, ?, ?, json(?), unixepoch())
      ON CONFLICT(checkpoint_key) DO UPDATE SET
        thread_id=excluded.thread_id,
        turn_id=excluded.turn_id,
        payload_json=excluded.payload_json,
        updated_at=excluded.updated_at
      `,
      [cp.key, cp.threadId, cp.turnId || null, JSON.stringify(cp)]
    );
  }
};
```

## 4. SQLite Runtime Configuration

Recommended defaults on open:

```sql
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA foreign_keys = ON;
PRAGMA busy_timeout = 5000;
PRAGMA temp_store = MEMORY;
```

Rationale:

- `WAL`: good read/write concurrency.
- `busy_timeout`: better behavior under short lock contention.
- `foreign_keys`: keeps referential integrity across thread/turn tables.

## 5. Suggested Core Schema

A practical baseline schema for resumability and broader data retention:

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS harness_runs (
  run_id TEXT PRIMARY KEY,
  script_path TEXT NOT NULL,
  script_sha256 TEXT NOT NULL,
  started_at INTEGER NOT NULL,
  ended_at INTEGER,
  exit_status TEXT,
  host_pid INTEGER,
  app_version TEXT
);

CREATE TABLE IF NOT EXISTS harness_threads (
  thread_id TEXT PRIMARY KEY,
  model TEXT,
  cwd TEXT,
  approval_policy TEXT,
  sandbox_policy TEXT,
  created_at INTEGER NOT NULL,
  last_seen_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS harness_turns (
  thread_id TEXT NOT NULL,
  turn_id TEXT NOT NULL,
  status TEXT,
  started_at INTEGER,
  completed_at INTEGER,
  latest_plan_json TEXT,
  latest_diff_text TEXT,
  PRIMARY KEY (thread_id, turn_id)
);

CREATE TABLE IF NOT EXISTS harness_checkpoints (
  checkpoint_key TEXT PRIMARY KEY,
  thread_id TEXT,
  turn_id TEXT,
  payload_json TEXT NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS harness_events (
  event_id INTEGER PRIMARY KEY AUTOINCREMENT,
  run_id TEXT,
  ts INTEGER NOT NULL,
  direction TEXT NOT NULL,
  method TEXT,
  rpc_id TEXT,
  thread_id TEXT,
  turn_id TEXT,
  item_id TEXT,
  payload_json TEXT
);

CREATE INDEX IF NOT EXISTS idx_harness_events_thread_turn
  ON harness_events(thread_id, turn_id, event_id);

CREATE TABLE IF NOT EXISTS approval_decisions (
  request_id TEXT PRIMARY KEY,
  thread_id TEXT,
  turn_id TEXT,
  method TEXT NOT NULL,
  decision TEXT NOT NULL,
  rationale TEXT,
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS tdd_iterations (
  thread_id TEXT NOT NULL,
  turn_id TEXT,
  iter INTEGER NOT NULL,
  cmd_json TEXT NOT NULL,
  exit_code INTEGER,
  stdout_text TEXT,
  stderr_text TEXT,
  created_at INTEGER NOT NULL,
  PRIMARY KEY (thread_id, iter)
);

CREATE TABLE IF NOT EXISTS review_runs (
  review_turn_id TEXT PRIMARY KEY,
  source_thread_id TEXT NOT NULL,
  source_turn_id TEXT,
  status TEXT,
  review_text TEXT,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS worker_subtasks (
  subtask_id TEXT PRIMARY KEY,
  parent_thread_id TEXT NOT NULL,
  worker_thread_id TEXT,
  title TEXT NOT NULL,
  prompt_text TEXT NOT NULL,
  status TEXT NOT NULL,
  summary_text TEXT,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS token_usage_snapshots (
  thread_id TEXT NOT NULL,
  observed_at INTEGER NOT NULL,
  ratio REAL,
  usage_json TEXT,
  PRIMARY KEY (thread_id, observed_at)
);

CREATE TABLE IF NOT EXISTS ui_state (
  state_key TEXT PRIMARY KEY,
  payload_json TEXT NOT NULL,
  updated_at INTEGER NOT NULL
);
```

## 6. Resume Algorithm With DB Backing

## 6.1 High-level flow

```text
startup
  -> open sqlite
  -> migrate schema
  -> load checkpoint by key
  -> resolve thread target (flag > checkpoint > latest-thread table)
  -> hydrate remote thread/read
  -> reconcile remote + db rows
  -> choose resume anchor
  -> emit resume summary
  -> continue live event loop
  -> persist every important state transition
```

## 6.2 Anchor selection using DB + remote

```js
async function resolveAnchor({ checkpointKey, requestedThreadId, policy }) {
  const cp = await db.queryOne(
    `SELECT payload_json FROM harness_checkpoints WHERE checkpoint_key = ?`,
    [checkpointKey]
  );

  const cpState = cp ? JSON.parse(cp.payload_json) : null;
  const threadId = requestedThreadId || cpState?.threadId || await inferMostRecentThreadId();
  if (!threadId) return { type: "start-new-thread" };

  const remote = await rpc.request("thread/read", { threadId, includeTurns: true });
  const turns = (remote && remote.turns) || [];

  if (policy === "latest-completed") {
    for (let i = turns.length - 1; i >= 0; i -= 1) {
      if (turns[i].status === "completed") {
        return { type: "completed", threadId, turnId: turns[i].id };
      }
    }
  }

  if (policy === "continue-active") {
    const latest = turns[turns.length - 1];
    if (latest && latest.status !== "completed") {
      return { type: "active", threadId, turnId: latest.id };
    }
  }

  return { type: "start-new-turn", threadId };
}
```

## 7. Passing DB Into Harness Scripts

There are two implementation styles.

## 7.1 Style A: global module only

- `require("db")` available in all scripts.
- Scripts decide how to use it.

Pros:

- Maximum flexibility.

Cons:

- Requires script discipline for schema and migration.

## 7.2 Style B: module + structured helper

- `require("db")` for raw SQL.
- `require("harnessStore")` for stable tables and helper methods.

Pros:

- Better consistency across scripts.
- Easier migrations and shared tooling.

Cons:

- Slightly more host code.

Recommendation:

- Implement both.
- Keep `db` low-level, keep `harnessStore` convention-driven.

## 8. Example Adaptations for All Harness Patterns From `01-app-server-js.md`

This section answers the direct request: what it would look like for those examples to support resumability and broader data keeping with SQLite.

## 8.1 Autopilot approvals + DB

Goal:

- Persist decisions by request signature to avoid re-prompting after restart.

```js
const db = require("db");
const approval = require("approval");

async function lookupDecision(method, threadId, turnId, commandSig) {
  return db.queryOne(
    `
    SELECT decision
    FROM approval_decisions
    WHERE method = ?
      AND thread_id = ?
      AND (turn_id = ? OR turn_id IS NULL)
      AND rationale = ?
    ORDER BY created_at DESC
    LIMIT 1
    `,
    [method, threadId || null, turnId || null, commandSig]
  );
}

async function persistDecision({ requestId, threadId, turnId, method, decision, rationale }) {
  await db.exec(
    `
    INSERT OR REPLACE INTO approval_decisions
      (request_id, thread_id, turn_id, method, decision, rationale, created_at)
    VALUES (?, ?, ?, ?, ?, ?, unixepoch())
    `,
    [requestId, threadId || null, turnId || null, method, decision, rationale || ""]
  );
}

session.onRequest(async (evt) => {
  if (evt.method !== "item/commandExecution/requestApproval") return;

  const p = evt.params || {};
  const sig = JSON.stringify(p.command || []);
  const saved = await lookupDecision(evt.method, p.threadId, p.turnId, sig);

  if (saved && saved.decision === "acceptForSession") {
    approval.acceptForSession(evt.id);
    return;
  }

  // ask UI, then persist
  const decision = "acceptForSession";
  await persistDecision({
    requestId: String(evt.id),
    threadId: p.threadId,
    turnId: p.turnId,
    method: evt.method,
    decision,
    rationale: sig
  });

  approval.acceptForSession(evt.id);
});
```

Resumability benefit:

- Harness restarts do not lose approval policy memory.

Other data keeping benefit:

- Later audit query can explain why decisions were repeated.

## 8.2 Plan-gate harness + DB

Goal:

- Persist latest plan and approval state so restart does not prompt again.

```js
async function savePlan(threadId, turnId, plan) {
  await db.exec(
    `
    INSERT INTO harness_turns(thread_id, turn_id, status, latest_plan_json)
    VALUES (?, ?, 'active', json(?))
    ON CONFLICT(thread_id, turn_id) DO UPDATE SET
      latest_plan_json=excluded.latest_plan_json
    `,
    [threadId, turnId, JSON.stringify(plan)]
  );
}

async function savePlanApproval(threadId, turnId, approved) {
  await db.exec(
    `
    INSERT INTO ui_state(state_key, payload_json, updated_at)
    VALUES (?, json(?), unixepoch())
    ON CONFLICT(state_key) DO UPDATE SET
      payload_json=excluded.payload_json,
      updated_at=excluded.updated_at
    `,
    [`plan-approved:${threadId}:${turnId}`, JSON.stringify({ approved })]
  );
}

async function isPlanApproved(threadId, turnId) {
  const row = await db.queryOne(
    `SELECT payload_json FROM ui_state WHERE state_key = ?`,
    [`plan-approved:${threadId}:${turnId}`]
  );
  return row ? !!JSON.parse(row.payload_json).approved : false;
}
```

Resumability benefit:

- If harness crashed after user approved a plan, it can continue without duplicate plan-gate friction.

Other data keeping benefit:

- Historical plans can be inspected across turns.

## 8.3 TDD loop harness + DB

Goal:

- Persist iteration count and test outputs across restarts.

```js
async function nextIteration(threadId) {
  const row = await db.queryOne(
    `SELECT COALESCE(MAX(iter), 0) AS n FROM tdd_iterations WHERE thread_id = ?`,
    [threadId]
  );
  return (row?.n || 0) + 1;
}

async function recordTDD({ threadId, turnId, iter, cmd, exitCode, stdout, stderr }) {
  await db.exec(
    `
    INSERT INTO tdd_iterations
      (thread_id, turn_id, iter, cmd_json, exit_code, stdout_text, stderr_text, created_at)
    VALUES (?, ?, ?, json(?), ?, ?, ?, unixepoch())
    `,
    [threadId, turnId || null, iter, JSON.stringify(cmd), exitCode, stdout || "", stderr || ""]
  );
}
```

Resumability benefit:

- Loop can resume at the correct iteration number.

Other data keeping benefit:

- You retain full historical test evidence for debugging and reporting.

## 8.4 Review-gate harness + DB

Goal:

- Persist review lifecycle and unresolved findings.

```js
async function saveReviewStart(reviewTurnId, sourceThreadId, sourceTurnId) {
  await db.exec(
    `
    INSERT OR REPLACE INTO review_runs
      (review_turn_id, source_thread_id, source_turn_id, status, created_at, updated_at)
    VALUES (?, ?, ?, 'started', unixepoch(), unixepoch())
    `,
    [reviewTurnId, sourceThreadId, sourceTurnId || null]
  );
}

async function saveReviewCompleted(reviewTurnId, text) {
  await db.exec(
    `
    UPDATE review_runs
    SET status = 'completed', review_text = ?, updated_at = unixepoch()
    WHERE review_turn_id = ?
    `,
    [text, reviewTurnId]
  );
}
```

Resumability benefit:

- A restart can query incomplete reviews and continue only pending ones.

Other data keeping benefit:

- Review corpus can be mined for recurring issue patterns.

## 8.5 Recursive decomposition harness + DB

Goal:

- Persist task graph and worker output incrementally.

```js
async function createSubtask({ subtaskId, parentThreadId, title, prompt }) {
  await db.exec(
    `
    INSERT OR REPLACE INTO worker_subtasks
      (subtask_id, parent_thread_id, title, prompt_text, status, created_at, updated_at)
    VALUES (?, ?, ?, ?, 'queued', unixepoch(), unixepoch())
    `,
    [subtaskId, parentThreadId, title, prompt]
  );
}

async function markSubtaskRunning(subtaskId, workerThreadId) {
  await db.exec(
    `
    UPDATE worker_subtasks
    SET status='running', worker_thread_id=?, updated_at=unixepoch()
    WHERE subtask_id=?
    `,
    [workerThreadId, subtaskId]
  );
}

async function markSubtaskDone(subtaskId, summary) {
  await db.exec(
    `
    UPDATE worker_subtasks
    SET status='done', summary_text=?, updated_at=unixepoch()
    WHERE subtask_id=?
    `,
    [summary, subtaskId]
  );
}
```

Resumability benefit:

- On restart, query `status IN ('queued','running')` and continue remaining subtasks.

Other data keeping benefit:

- You accumulate reusable subtask solution history.

## 8.6 Auto-compaction harness + DB

Goal:

- Persist token usage trend and compaction cooldown.

```js
async function saveUsage(threadId, usage) {
  await db.exec(
    `
    INSERT INTO token_usage_snapshots(thread_id, observed_at, ratio, usage_json)
    VALUES (?, unixepoch(), ?, json(?))
    `,
    [threadId, usage.ratio || null, JSON.stringify(usage)]
  );
}

async function shouldCompact(threadId, ratio, softLimit) {
  if (ratio <= softLimit) return false;

  const last = await db.queryOne(
    `
    SELECT observed_at
    FROM token_usage_snapshots
    WHERE thread_id = ? AND ratio IS NOT NULL AND ratio > ?
    ORDER BY observed_at DESC
    LIMIT 1
    `,
    [threadId, softLimit]
  );

  // simple cooldown: only compact if last trigger older than 5 min
  if (!last) return true;
  return (Math.floor(Date.now() / 1000) - last.observed_at) > 300;
}
```

Resumability benefit:

- Restarted harness still knows recent compaction history and avoids repeated compaction thrash.

Other data keeping benefit:

- Long-term usage analysis becomes possible.

## 9. Additional Data-Keeping Use Cases Beyond Resume

A DB object also enables useful non-resume features:

1. Operator notes per thread (`ui_state` rows).
2. Prompt template library (table of reusable prompts).
3. Command allowlist statistics (approval safety tuning).
4. Per-script KPI dashboards (median turn duration, failure rates).
5. Incident forensics (event replay window from `harness_events`).

Example table:

```sql
CREATE TABLE IF NOT EXISTS prompt_templates (
  template_id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  body TEXT NOT NULL,
  tags_json TEXT,
  updated_at INTEGER NOT NULL
);
```

## 10. Concurrency and Locking Strategy

## 10.1 Single-writer expectation

For many harness runs, one process writes one DB file. This is simplest and recommended default.

## 10.2 Multi-process lock lease (optional)

If parallel harness processes can target same checkpoint key, add lease table:

```sql
CREATE TABLE IF NOT EXISTS checkpoint_leases (
  checkpoint_key TEXT PRIMARY KEY,
  owner_id TEXT NOT NULL,
  lease_until INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
```

Lease acquisition pattern:

```js
await db.tx(async () => {
  const now = Math.floor(Date.now() / 1000);
  const until = now + 30;

  await db.exec(
    `
    INSERT INTO checkpoint_leases(checkpoint_key, owner_id, lease_until, updated_at)
    VALUES (?, ?, ?, ?)
    ON CONFLICT(checkpoint_key) DO UPDATE SET
      owner_id = CASE WHEN checkpoint_leases.lease_until < excluded.updated_at THEN excluded.owner_id ELSE checkpoint_leases.owner_id END,
      lease_until = CASE WHEN checkpoint_leases.lease_until < excluded.updated_at THEN excluded.lease_until ELSE checkpoint_leases.lease_until END,
      updated_at = excluded.updated_at
    `,
    [checkpointKey, ownerId, until, now]
  );
});
```

If lease owner differs, script can warn or abort.

## 11. Migration and Compatibility

## 11.1 Migration policy

- Keep SQL migrations numbered and idempotent.
- Apply migrations before exposing `db` to script runtime.
- Record migration versions in `schema_migrations`.

## 11.2 Schema evolution examples

Migration v2 example:

```sql
ALTER TABLE approval_decisions ADD COLUMN rule_name TEXT;
CREATE INDEX IF NOT EXISTS idx_approval_rule_name ON approval_decisions(rule_name);
INSERT INTO schema_migrations(version, applied_at) VALUES (2, unixepoch());
```

## 11.3 Script compatibility strategy

Store script fingerprint in `harness_runs` and checkpoint payload:

- if schema supports migration: migrate and continue
- if incompatible: mark checkpoint stale and soft-reset local state

## 12. Security Boundaries

Since you requested that scripts can "do what they want" with DB, this section is about explicit boundaries so flexibility does not become unsafe.

Recommended boundaries:

1. DB path constrained to workspace unless `--db-allow-outside-workspace` set.
2. Optional read-only mode for scripts that should not mutate state.
3. Optional query timeout / statement step limit to avoid runaway SQL.
4. Redaction hook before inserting payloads from sensitive events.

## 13. Proposed CLI Flags

Add to `harness run`:

- `--db-path <path>` default `.openai-app-server/harness.db`
- `--db-migrate` default true
- `--db-read-only` default false
- `--db-busy-timeout-ms` default 5000
- `--db-wal` default true
- `--db-checkpoint-key <string>` optional explicit key

## 14. Proposed Go Interfaces

```go
// pkg/js/runtime.go additions
type DBBridge interface {
    Exec(ctx context.Context, sql string, params any) (map[string]any, error)
    QueryOne(ctx context.Context, sql string, params any) (map[string]any, error)
    QueryAll(ctx context.Context, sql string, params any) ([]map[string]any, error)
    WithTx(ctx context.Context, fn func(context.Context) error) error
    Close() error
}

type Options struct {
    Name string
    RPC  RPCBridge
    UI   UIBridge
    DB   DBBridge // new
}
```

`pkg/js/module_db.go` would bind bridge methods into goja.

## 15. End-to-End Resume Example

This is a full pattern combining DB and remote hydration.

```js
const codex = require("codex");
const db = require("db");
const rpc = require("rpc");
const ui = require("ui");

async function bootstrap() {
  await db.exec(`
    CREATE TABLE IF NOT EXISTS harness_checkpoints (
      checkpoint_key TEXT PRIMARY KEY,
      payload_json TEXT NOT NULL,
      updated_at INTEGER NOT NULL
    )
  `);
}

async function loadCheckpoint(key) {
  const row = await db.queryOne(
    `SELECT payload_json FROM harness_checkpoints WHERE checkpoint_key = ?`,
    [key]
  );
  return row ? JSON.parse(row.payload_json) : null;
}

async function saveCheckpoint(key, payload) {
  await db.exec(
    `
    INSERT INTO harness_checkpoints(checkpoint_key, payload_json, updated_at)
    VALUES (?, json(?), unixepoch())
    ON CONFLICT(checkpoint_key) DO UPDATE SET
      payload_json = excluded.payload_json,
      updated_at = excluded.updated_at
    `,
    [key, JSON.stringify(payload)]
  );
}

(async () => {
  await bootstrap();

  const checkpointKey = "main";
  const cp = await loadCheckpoint(checkpointKey);

  const session = codex.connect();
  let threadId = cp?.threadId;

  if (!threadId) {
    const started = await rpc.request("thread/start", {
      model: "gpt-5",
      cwd: "/workspace",
      approvalPolicy: "on-request",
      sandbox: "workspace-write"
    });
    threadId = started.threadId || started.id;
  }

  const thread = await rpc.request("thread/read", { threadId, includeTurns: true });

  ui.emit({ type: "resume-ready", threadId, turnCount: (thread.turns || []).length });

  await saveCheckpoint(checkpointKey, {
    threadId,
    lastSeenTurnId: cp?.lastSeenTurnId || null,
    ts: new Date().toISOString()
  });
})();
```

## 16. Testing Strategy for DB-Backed Harnesses

## 16.1 Unit tests

- SQL migration tests.
- checkpoint read/write tests.
- reconciliation tests under mismatch scenarios.

## 16.2 Integration tests (memory transport + temp sqlite)

- run script once -> create checkpoint rows
- restart script -> verify resumed thread and recovered gates
- crash simulation between request and response

## 16.3 Validation queries for operators

```sql
-- latest checkpoint
SELECT checkpoint_key, updated_at, json_extract(payload_json, '$.threadId') AS thread_id
FROM harness_checkpoints
ORDER BY updated_at DESC;

-- unresolved review runs
SELECT review_turn_id, source_thread_id, status
FROM review_runs
WHERE status != 'completed';

-- tdd failure history
SELECT thread_id, iter, exit_code
FROM tdd_iterations
WHERE exit_code != 0
ORDER BY created_at DESC
LIMIT 50;
```

## 17. Tradeoff Matrix

| Option | Resume quality | Complexity | Flexibility | Auditability |
|---|---|---|---|---|
| No DB (memory only) | Low | Low | Medium | Low |
| JSON checkpoint only | Medium | Low-Medium | Medium | Low-Medium |
| SQLite with minimal tables | High | Medium | High | High |
| SQLite + full event sourcing | Very high | High | Very high | Very high |

## 18. Recommended Incremental Plan

1. Phase A: Add `db` module with `exec/queryOne/queryAll/tx`.
2. Phase B: Add base schema and checkpoint integration.
3. Phase C: Migrate one harness pattern first (plan-gate or tdd-loop).
4. Phase D: Expand to all six example patterns and add runbook queries.
5. Phase E: Add lock leases and retention compaction if needed.

## 19. Direct Answer to "What would that look like?"

It looks like this in practice:

- Script receives `db` object via `require("db")`.
- On startup, script runs migrations and loads checkpoint rows.
- Script hydrates remote thread data.
- Script chooses resume anchor using both DB and remote state.
- During runtime, each meaningful event writes durable rows in a transaction.
- On crash/restart, the same script loads rows and continues deterministically.

For the source document's example harnesses, SQLite turns each pattern from "best-effort in-memory orchestration" into "durable workflow orchestration".

## 20. Conclusion

Passing a SQLite `db` object to harness scripts is the most practical way to guarantee resumability while also unlocking richer operational data. It is local-first, dependency-light, and fully compatible with the current goja runtime model.

The best implementation path is to keep raw SQL power available, provide stable helper conventions, and roll out schema-backed resume in phases tied to the six existing harness patterns.
