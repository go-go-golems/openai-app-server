---
Title: Script-Owned SQLite Schemas and Resume Patterns
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
      Note: Command entrypoint for db options and startup/shutdown checkpoint orchestration
    - Path: openai-app-server/pkg/js/module_codex.go
      Note: Session module that script resume logic coordinates with
    - Path: openai-app-server/pkg/js/runtime.go
      Note: Runtime bridge where db module lifecycle hooks can call script init/resume/checkpoint methods
    - Path: openai-app-server/ttmp/2026/02/22/OAP-01-INITIAL-JS-HARNESS--initial-js-harness/sources/local/01-app-server-js.md
      Note: Source harness examples mapped to script-owned schema patterns
    - Path: openai-app-server/ttmp/2026/02/22/OAP-03-RESUMING-SESSIONS--resuming-sessions/design/02-sqlite-db-object-for-harness-resume-and-state.md
      Note: Base db-object design extended here to script-owned schemas
ExternalSources:
    - local:01-app-server-js.md
Summary: Detailed design for allowing each harness script to own its SQLite schema, migration lifecycle, and resume logic while preserving operational safety and consistency.
LastUpdated: 2026-02-23T13:25:00-05:00
WhatFor: Define a practical contract where scripts can freely design their own persistence model and resume semantics.
WhenToUse: Use when implementing script-level db initialization and resume orchestration in JS harnesses.
---

# Script-Owned SQLite Schemas and Resume Patterns

## 1. Purpose and Audience

This document is written for a new engineer joining the project, including someone who has never worked with this harness system before. The goal is to explain, in plain language, why we want script-owned database schemas, how they fit into the harness runtime, and exactly what a script author needs to implement to get reliable resumability.

When we say "script-owned schema," we mean that each harness script can create its own SQLite tables and design its own persistence model. The host runtime still provides the same `db` API, but it does not force every script into one global table layout. This gives each script freedom to store only what it needs while still following a small set of safety conventions.

If you read only one section before coding, read Section 6 and Section 7. Those two sections define the minimal contract and startup lifecycle a script must implement to be production-safe.

## 2. System Context: What Is a Harness in This Repo?

In this repository, a harness script is a JavaScript program that runs inside a Go-hosted goja runtime. The script talks to Codex App Server through JSON-RPC methods and events. The common objects a script sees are `codex`, `rpc`, `ui`, `approval`, and eventually `db`.

Codex conversation state is structured as thread -> turn -> item. A thread is a long-running conversation container, a turn is a single request-response execution segment, and items are streamed events inside that turn (messages, commands, plans, diffs, approvals, and so on). The harness listens to events, decides policies, and can start/steer/interrupt turns.

Without persistence, all in-memory script state is lost when the process exits. That includes iteration counters, pending gates, prior approval decisions, and other workflow context that the remote thread alone does not always fully encode. A `db` object solves this by giving scripts durable local state.

## 3. Core Question and Direct Answer

The question is whether scripts should be allowed to design their own schema and initialize/resume themselves. The direct answer is yes, this generally makes the system nicer and more maintainable.

It is better because different scripts solve different orchestration problems. A plan-gate script needs durable plan approvals. A TDD loop script needs iteration history. A recursive decomposition script needs durable job and subtask state. Trying to fit all of those into one universal schema usually produces a rigid, confusing model.

The tradeoff is discipline. If scripts are fully free-form, schemas drift and quality drops. The solution is to keep freedom for data modeling but require a tiny cross-script contract for migrations, checkpointing, and resume semantics.

## 4. Decision: Flexible Ownership With Minimal Contract

The recommended architecture is "script-owned schema with shared guardrails." That means every script can create and evolve tables, indexes, and views for its own domain, but every script must still follow the same lifecycle contract and naming conventions.

This approach gives the right balance. Script authors iterate fast without waiting for host schema changes, while operators still get predictable behavior on restart. It also keeps future platform work possible, because shared conventions make scripts understandable to people who did not author them.

In short, the host should provide durable SQLite and safe primitives, and scripts should own domain-specific persistence behavior.

## 5. Mental Model: Two Truths That Must Be Reconciled

A resumable harness always has two truths to reconcile on startup: remote truth and local truth. Remote truth comes from `thread/read` and related Codex APIs. Local truth comes from the script's SQLite tables.

Remote truth tells you what happened in Codex from the server's perspective. Local truth tells you what the harness intended and where it was in local workflow logic. The script's `resume()` routine must merge these two and decide an anchor: continue active turn, start a new turn, re-run a gate, or branch.

This is the key idea for interns: resume is not "load file and continue." Resume is always a reconciliation decision between remote state and local workflow state.

## 6. Minimal Required Contract for Every Script

Every script should implement five functions, even if the internals are small. This creates consistency across scripts and makes incident response much easier.

```ts
interface HarnessPersistenceContract {
  initDb(): Promise<void>;
  loadState(): Promise<any>;
  reconcileRemote(remoteThread: any, localState: any): Promise<any>;
  saveState(nextState: any): Promise<void>;
  checkpoint(reason: string): Promise<void>;
}
```

`initDb()` must be idempotent so running it twice is harmless. `loadState()` should return `null` safely when no prior state exists. `reconcileRemote()` is where you choose resume behavior deterministically. `saveState()` and `checkpoint()` should use transactions so half-written state does not corrupt future restarts.

If a script exports this contract consistently, another engineer can quickly understand and maintain it, even without deep context.

## 7. Canonical Startup and Resume Lifecycle

A script should follow the same high-level flow every run. The exact schema differs per script, but the startup rhythm should stay stable so operators and reviewers always know where to look when debugging.

```text
start process
  -> initDb()
  -> loadState()
  -> resolve thread target
  -> fetch remote thread/read
  -> reconcileRemote()
  -> emit resume summary to UI
  -> run normal event loop
  -> checkpoint periodically and on exit
```

Here is a concrete startup skeleton:

```js
const rpc = require("rpc");
const ui = require("ui");

async function bootstrap() {
  await initDb();
  const local = await loadState();

  const threadId = await resolveThreadId(local);
  const remote = threadId
    ? await rpc.request("thread/read", { threadId, includeTurns: true })
    : null;

  const decision = await reconcileRemote(remote, local);
  ui.emit({ type: "resume-decision", decision });

  await saveState({ ...local, decision, bootstrappedAt: new Date().toISOString() });
  return decision;
}
```

For an intern, this is the first implementation target: get this bootstrap path correct before optimizing anything else.

## 8. Naming and Namespace Conventions

When multiple scripts share one DB file, naming conventions prevent collisions and confusion. Each script should use a stable `SCRIPT_ID`, and every table should begin with that prefix.

For example, a script with `SCRIPT_ID = "plan_gate_v1"` would define tables like `plan_gate_v1_meta`, `plan_gate_v1_checkpoints`, and `plan_gate_v1_turn_state`. This keeps schemas readable and avoids accidental cross-script updates.

Conventions are especially important for onboarding. A new engineer can inspect the DB and immediately identify which rows belong to which script.

```js
const SCRIPT_ID = "plan_gate_v1";

function table(name) {
  return `${SCRIPT_ID}_${name}`;
}
```

## 9. Migration Strategy: How Scripts Evolve Safely

Allowing script-owned schema means schema evolution is normal. The safe pattern is to keep migrations versioned and transactional, and to persist the current version in a small shared migration table.

Each migration should be an idempotent unit. If the process crashes, rerunning startup should either continue cleanly or reapply safely without corrupting state. Never bump version before the migration body succeeds.

```js
const db = require("db");

async function ensureMigrationTable() {
  await db.exec(`
    CREATE TABLE IF NOT EXISTS script_schema_versions (
      script_id TEXT PRIMARY KEY,
      version INTEGER NOT NULL,
      updated_at INTEGER NOT NULL
    )
  `);
}

async function currentVersion(scriptId) {
  const row = await db.queryOne(
    `SELECT version FROM script_schema_versions WHERE script_id = ?`,
    [scriptId]
  );
  return row ? row.version : 0;
}

async function setVersion(scriptId, version) {
  await db.exec(
    `
    INSERT INTO script_schema_versions(script_id, version, updated_at)
    VALUES (?, ?, unixepoch())
    ON CONFLICT(script_id) DO UPDATE SET
      version = excluded.version,
      updated_at = excluded.updated_at
    `,
    [scriptId, version]
  );
}

async function runMigrations(scriptId, migrations) {
  await ensureMigrationTable();
  const v = await currentVersion(scriptId);

  for (const m of migrations) {
    if (m.version <= v) continue;
    await db.tx(async () => {
      await m.up();
      await setVersion(scriptId, m.version);
    });
  }
}
```

## 10. Template: A Complete Script-Owned Checkpoint Layer

A script should maintain one durable checkpoint row that represents the current orchestrator state. This is the fastest way to recover state on restart, while detailed history can live in additional tables.

```js
const SCRIPT_ID = "my_harness_v1";
const CHECKPOINT_KEY = "default";

async function initCheckpointTable() {
  await db.exec(`
    CREATE TABLE IF NOT EXISTS my_harness_v1_checkpoints (
      checkpoint_key TEXT PRIMARY KEY,
      payload_json TEXT NOT NULL,
      updated_at INTEGER NOT NULL
    )
  `);
}

async function loadCheckpoint() {
  const row = await db.queryOne(
    `SELECT payload_json FROM my_harness_v1_checkpoints WHERE checkpoint_key = ?`,
    [CHECKPOINT_KEY]
  );
  return row ? JSON.parse(row.payload_json) : null;
}

async function saveCheckpoint(payload) {
  await db.exec(
    `
    INSERT INTO my_harness_v1_checkpoints(checkpoint_key, payload_json, updated_at)
    VALUES (?, json(?), unixepoch())
    ON CONFLICT(checkpoint_key) DO UPDATE SET
      payload_json = excluded.payload_json,
      updated_at = excluded.updated_at
    `,
    [CHECKPOINT_KEY, JSON.stringify(payload)]
  );
}
```

For beginners: this checkpoint is your script's "save game." Keep it compact and deterministic.

## 11. Resume Reconciliation Algorithm (Intern-Friendly)

On startup, you should pick behavior by deterministic rules. Do not make ad hoc decisions in event handlers; keep resume logic in one place.

A simple and safe policy is:

1. If local state has no thread ID, start a new thread.
2. If local thread exists but remote thread is missing, either fail or start fresh based on explicit policy.
3. If latest remote turn is active, continue active.
4. Otherwise start a new turn on same thread.

```js
async function reconcileRemote(remote, local) {
  if (!local || !local.threadId) {
    return { action: "start-new-thread", reason: "no-local-thread" };
  }

  if (!remote) {
    return { action: "start-new-thread", reason: "remote-missing" };
  }

  const turns = Array.isArray(remote.turns) ? remote.turns : [];
  const latest = turns[turns.length - 1];

  if (latest && latest.status !== "completed") {
    return {
      action: "continue-active",
      threadId: local.threadId,
      turnId: latest.id,
      reason: "latest-turn-active"
    };
  }

  return {
    action: "start-new-turn",
    threadId: local.threadId,
    fromTurnId: latest ? latest.id : null,
    reason: "latest-completed"
  };
}
```

## 12. Pattern Examples From Source Harness Designs

The imported source document includes several harness patterns. Below is how each pattern benefits from script-owned schemas, with concrete examples.

### 12.1 Autopilot Approvals

An approvals harness should store both policy memory and event history. Policy memory avoids asking the same question after restart. History helps audit and debugging.

```sql
CREATE TABLE IF NOT EXISTS approvals_v1_policies (
  policy_key TEXT PRIMARY KEY,
  decision TEXT NOT NULL,
  rationale TEXT,
  updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS approvals_v1_history (
  request_id TEXT PRIMARY KEY,
  method TEXT NOT NULL,
  thread_id TEXT,
  turn_id TEXT,
  signature TEXT,
  decision TEXT NOT NULL,
  created_at INTEGER NOT NULL
);
```

```js
function approvalKey(evt) {
  const p = evt.params || {};
  return `${evt.method}|${p.threadId || ""}|${JSON.stringify(p.command || [])}`;
}

async function decideApproval(evt) {
  const key = approvalKey(evt);
  const saved = await db.queryOne(
    `SELECT decision FROM approvals_v1_policies WHERE policy_key = ?`,
    [key]
  );
  return saved ? saved.decision : "prompt-user";
}
```

### 12.2 Plan Gate

A plan-gate harness should persist plan snapshots and approval state by `thread_id + turn_id`. This prevents duplicate prompts after restart.

```sql
CREATE TABLE IF NOT EXISTS plan_gate_v1_turns (
  thread_id TEXT NOT NULL,
  turn_id TEXT NOT NULL,
  latest_plan_json TEXT,
  approved INTEGER NOT NULL DEFAULT 0,
  approved_at INTEGER,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (thread_id, turn_id)
);
```

If a restart happens after user approval, the script loads the row and skips re-prompting. If approval was never saved, it re-renders plan and asks again.

### 12.3 TDD Loop

A TDD loop should persist iteration cursor and test outputs. That way restart continues at the right iteration and keeps prior failure context.

```sql
CREATE TABLE IF NOT EXISTS tdd_v1_state (
  thread_id TEXT PRIMARY KEY,
  current_iter INTEGER NOT NULL,
  max_iters INTEGER NOT NULL,
  status TEXT NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS tdd_v1_results (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  thread_id TEXT NOT NULL,
  turn_id TEXT,
  iter INTEGER NOT NULL,
  cmd_json TEXT NOT NULL,
  exit_code INTEGER,
  stdout_text TEXT,
  stderr_text TEXT,
  created_at INTEGER NOT NULL
);
```

```js
async function tddNextIter(threadId, maxIters) {
  const row = await db.queryOne(`SELECT * FROM tdd_v1_state WHERE thread_id = ?`, [threadId]);
  if (!row) {
    await db.exec(
      `INSERT INTO tdd_v1_state(thread_id, current_iter, max_iters, status, updated_at)
       VALUES (?, 1, ?, 'running-tests', unixepoch())`,
      [threadId, maxIters]
    );
    return 1;
  }

  await db.exec(
    `UPDATE tdd_v1_state SET current_iter = current_iter + 1, status='running-tests', updated_at=unixepoch()
     WHERE thread_id = ?`,
    [threadId]
  );

  const next = await db.queryOne(`SELECT current_iter FROM tdd_v1_state WHERE thread_id = ?`, [threadId]);
  return next.current_iter;
}
```

### 12.4 Review Gate

A review gate should store whether review output has already triggered remediation. This avoids duplicate follow-up turns after restart.

```sql
CREATE TABLE IF NOT EXISTS review_gate_v1_runs (
  review_turn_id TEXT PRIMARY KEY,
  source_thread_id TEXT NOT NULL,
  source_turn_id TEXT,
  status TEXT NOT NULL,
  review_text TEXT,
  followup_started INTEGER NOT NULL DEFAULT 0,
  updated_at INTEGER NOT NULL
);
```

If `followup_started=1`, restart should not generate another remediation turn for the same review turn.

### 12.5 Recursive Decomposition

Recursive/worker harnesses should persist jobs and subtasks, because the highest failure risk is duplicated or lost subtask work during restarts.

```sql
CREATE TABLE IF NOT EXISTS rlm_v1_jobs (
  job_id TEXT PRIMARY KEY,
  parent_thread_id TEXT NOT NULL,
  goal_text TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS rlm_v1_subtasks (
  subtask_id TEXT PRIMARY KEY,
  job_id TEXT NOT NULL,
  title TEXT NOT NULL,
  prompt_text TEXT NOT NULL,
  worker_thread_id TEXT,
  status TEXT NOT NULL,
  summary_text TEXT,
  updated_at INTEGER NOT NULL
);
```

On restart, query `status IN ('queued','running')` and continue only unfinished subtasks.

### 12.6 Auto-Compaction

Compaction automation should store last compaction timestamp and cooldown settings. This prevents compaction storms on restart.

```sql
CREATE TABLE IF NOT EXISTS compact_v1_state (
  thread_id TEXT PRIMARY KEY,
  last_compact_at INTEGER,
  cooldown_seconds INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
```

## 13. Host Responsibilities When Scripts Own Schema

Even with script-owned schema, the host still has important responsibilities. The host should expose stable DB primitives and runtime hooks but avoid forcing script table design.

The host should also provide operational flags such as `--db-path`, `--db-read-only`, and possibly `--db-busy-timeout-ms`. These make script behavior predictable in different environments.

Suggested optional lifecycle hooks:

```js
module.exports = {
  async initDb() {},
  async resume() {},
  async checkpoint(reason) {}
};
```

The host can call these automatically if exported, which standardizes startup/shutdown behavior across scripts.

## 14. Operational Safety Practices

If scripts can do anything with SQL, safety practices become mandatory. Use transactions for multi-step updates, keep checkpoints small, and never mutate state from more than one process unless you have lock/lease rules.

Set SQLite pragmas thoughtfully at connection time. WAL mode and sensible busy timeout greatly improve reliability. Keep schema evolution additive when possible, and write migration tests before changing production scripts.

For sensitive payloads, scripts should redact before persistence. Not every event payload belongs in durable storage.

## 15. Failure Handling and Recovery

A reliable system assumes failures will happen. Scripts should explicitly handle migration failures, JSON decode failures, remote thread disappearance, and lock contention.

If checkpoint JSON cannot be parsed, scripts should log a diagnostic event, preserve the bad row for forensics, and continue with safe fallback state. If remote thread is missing, scripts should choose between fail-fast and start-fresh by explicit policy, not by implicit default.

This section is critical for interns: graceful degradation is more important than perfect completeness.

## 16. Testing Plan for New Engineers

An intern should implement these tests first for any new script-owned schema:

1. Fresh DB initialization works.
2. Re-running initialization is idempotent.
3. Migration N -> N+1 works.
4. Resume after clean shutdown keeps cursor/state.
5. Resume after simulated crash does not duplicate critical actions.

Simple integration scenario:

```text
run #1: init -> start thread -> write checkpoint -> terminate
run #2: init -> load checkpoint -> thread/read -> reconcile -> continue
assert: no duplicated gate prompts, counters preserved
```

If these tests pass, most practical resume bugs are already prevented.

## 17. First-Week Onboarding Checklist for Interns

If a new intern joins and needs to become productive quickly, this is the recommended path. On day one, have them read this document and inspect one existing script table set in SQLite. On day two, have them add a tiny migration and verify idempotence. On day three, have them implement one resume branch and write a crash-recovery test.

By the end of the first week, the intern should be able to create a small script that initializes schema, stores checkpoints, resumes from local+remote state, and avoids duplicate side effects on restart. That is the practical milestone for independent contribution.

## 18. Final Recommendation

Adopt script-owned schema/resume as the default model, because it fits the real diversity of harness workflows better than a single global schema. Keep the system understandable by enforcing a minimal shared contract and naming conventions, and by providing host-level lifecycle hooks that scripts can opt into.

This model gives flexibility to script authors, clarity to operators, and a clear learning path for new team members.
