// Sketch: Release readiness gate harness (real-world, non-RLM).
// Focus: durable check state, blockers, and resumable go/no-go decision flow.

const codex = require("codex");
const db = require("db");
const ui = require("ui");

const CHECKPOINT_KEY = "active";

const session = codex.connect({
  defaults: {
    model: "gpt-5",
    cwd: "/home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server",
    approvalPolicy: "on-request",
    sandbox: "workspace-write"
  }
});

session.approvals.setPolicy({
  command: (req) => {
    const cmd = req.commandText || "";
    if (cmd.startsWith("go test") || cmd.startsWith("make lint") || cmd.startsWith("git diff")) {
      return "acceptForSession";
    }
    return "decline";
  },
  fileChange: () => "decline",
  fallback: () => "decline"
});

function rid(prefix) {
  return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 1e6)}`;
}

async function initDb() {
  await db.exec(`
    CREATE TABLE IF NOT EXISTS release_gate_v1_runs (
      run_id TEXT PRIMARY KEY,
      release_tag TEXT NOT NULL,
      status TEXT NOT NULL,
      thread_id TEXT,
      created_at INTEGER NOT NULL,
      updated_at INTEGER NOT NULL
    )
  `);

  await db.exec(`
    CREATE TABLE IF NOT EXISTS release_gate_v1_checks (
      check_id TEXT PRIMARY KEY,
      run_id TEXT NOT NULL,
      check_name TEXT NOT NULL,
      status TEXT NOT NULL,
      detail_text TEXT,
      updated_at INTEGER NOT NULL
    )
  `);

  await db.exec(`
    CREATE TABLE IF NOT EXISTS release_gate_v1_blockers (
      blocker_id TEXT PRIMARY KEY,
      run_id TEXT NOT NULL,
      title TEXT NOT NULL,
      severity TEXT NOT NULL,
      owner TEXT,
      status TEXT NOT NULL,
      created_at INTEGER NOT NULL,
      updated_at INTEGER NOT NULL
    )
  `);

  await db.exec(`
    CREATE TABLE IF NOT EXISTS release_gate_v1_checkpoints (
      checkpoint_key TEXT PRIMARY KEY,
      payload_json TEXT NOT NULL,
      updated_at INTEGER NOT NULL
    )
  `);
}

async function loadCheckpoint() {
  const row = await db.queryOne(
    `SELECT payload_json FROM release_gate_v1_checkpoints WHERE checkpoint_key = ?`,
    [CHECKPOINT_KEY]
  );
  return row ? JSON.parse(row.payload_json) : null;
}

async function saveCheckpoint(payload) {
  await db.exec(
    `INSERT INTO release_gate_v1_checkpoints(checkpoint_key, payload_json, updated_at)
     VALUES (?, json(?), unixepoch())
     ON CONFLICT(checkpoint_key) DO UPDATE SET payload_json=excluded.payload_json, updated_at=excluded.updated_at`,
    [CHECKPOINT_KEY, JSON.stringify(payload)]
  );
}

async function upsertCheck(runId, checkName, status, detail) {
  const checkId = `${runId}:${checkName}`;
  await db.exec(
    `INSERT INTO release_gate_v1_checks(check_id, run_id, check_name, status, detail_text, updated_at)
     VALUES (?, ?, ?, ?, ?, unixepoch())
     ON CONFLICT(check_id) DO UPDATE SET status=excluded.status, detail_text=excluded.detail_text, updated_at=excluded.updated_at`,
    [checkId, runId, checkName, status, detail || ""]
  );
}

async function addBlocker(runId, title, severity, owner) {
  await db.exec(
    `INSERT INTO release_gate_v1_blockers(blocker_id, run_id, title, severity, owner, status, created_at, updated_at)
     VALUES (?, ?, ?, ?, ?, 'open', unixepoch(), unixepoch())`,
    [rid("blk"), runId, title, severity, owner || null]
  );
}

async function resumeOrCreateRun(releaseTag) {
  const cp = await loadCheckpoint();
  if (cp && cp.runId) {
    const run = await db.queryOne(`SELECT * FROM release_gate_v1_runs WHERE run_id=?`, [cp.runId]);
    if (run) {
      return {
        runId: run.run_id,
        releaseTag: run.release_tag,
        phase: cp.phase || "evaluate",
        threadId: cp.threadId || run.thread_id || null
      };
    }
  }

  const runId = rid("rel");
  await db.exec(
    `INSERT INTO release_gate_v1_runs(run_id, release_tag, status, created_at, updated_at)
     VALUES (?, ?, 'evaluate', unixepoch(), unixepoch())`,
    [runId, releaseTag]
  );

  const state = { runId, releaseTag, phase: "evaluate", threadId: null };
  await saveCheckpoint(state);
  return state;
}

async function ensureThread(state) {
  if (state.threadId) return state.threadId;
  const started = await session.threads.start();
  state.threadId = session.ids.thread(started);
  await db.exec(`UPDATE release_gate_v1_runs SET thread_id=?, updated_at=unixepoch() WHERE run_id=?`, [state.threadId, state.runId]);
  await saveCheckpoint(state);
  return state.threadId;
}

async function evaluate(state) {
  const thread = session.thread(await ensureThread(state));
  const turn = await thread.turn.start({
    input: [{
      type: "text",
      text:
        `You are release manager assistant.\n` +
        `Evaluate release readiness for tag ${state.releaseTag}.\n` +
        `Assess tests, lint, migration risk, rollback readiness, and observability.\n` +
        `Return checks, blockers, and decision.`
    }],
    outputSchema: {
      type: "object",
      properties: {
        checks: {
          type: "array",
          items: {
            type: "object",
            properties: {
              name: { type: "string" },
              status: { type: "string" },
              detail: { type: "string" }
            },
            required: ["name", "status"]
          }
        },
        blockers: {
          type: "array",
          items: {
            type: "object",
            properties: {
              title: { type: "string" },
              severity: { type: "string" },
              owner: { type: "string" }
            },
            required: ["title", "severity"]
          }
        },
        decision: { type: "string" }
      },
      required: ["checks", "decision"]
    }
  });

  await thread.turn.waitCompleted({ turnId: session.ids.turn(turn), timeoutMs: 180000 });
  const result = await turn.getStructuredOutput();

  await db.tx(async () => {
    for (const c of result.checks || []) {
      await upsertCheck(state.runId, c.name, c.status, c.detail || "");
    }
    for (const b of result.blockers || []) {
      await addBlocker(state.runId, b.title, b.severity, b.owner || "unassigned");
    }

    await db.exec(
      `UPDATE release_gate_v1_runs SET status=?, updated_at=unixepoch() WHERE run_id=?`,
      [result.decision === "go" ? "go" : "blocked", state.runId]
    );
  });

  state.phase = "complete";
  await saveCheckpoint(state);
  ui.emit({ type: "release-gate-complete", runId: state.runId, releaseTag: state.releaseTag, result });
}

async function main(releaseTag) {
  await initDb();
  const state = await resumeOrCreateRun(releaseTag);
  ui.emit({ type: "release-gate-start", runId: state.runId, releaseTag: state.releaseTag, phase: state.phase, threadId: state.threadId });

  if (state.phase === "evaluate") {
    await evaluate(state);
  }
}

main("v1.12.0").catch((err) => ui.emit({ type: "release-gate-failed", error: String(err) }));
