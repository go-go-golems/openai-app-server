// Sketch: Data pipeline quality guardian harness (real-world, non-RLM).
// Focus: anomaly triage, root-cause hypotheses, and resumable remediation tracking.

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
    if (cmd.startsWith("ls ") || cmd.startsWith("cat ") || cmd.startsWith("wc ")) {
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

async function initSchema() {
  await db.exec(`
    CREATE TABLE IF NOT EXISTS pipeline_quality_guardian_v1_runs (
      run_id TEXT PRIMARY KEY,
      pipeline_name TEXT NOT NULL,
      status TEXT NOT NULL,
      thread_id TEXT,
      created_at INTEGER NOT NULL,
      updated_at INTEGER NOT NULL
    )
  `);

  await db.exec(`
    CREATE TABLE IF NOT EXISTS pipeline_quality_guardian_v1_anomalies (
      anomaly_id TEXT PRIMARY KEY,
      run_id TEXT NOT NULL,
      metric_name TEXT NOT NULL,
      observed_value REAL,
      expected_range TEXT,
      status TEXT NOT NULL,
      hypothesis TEXT,
      remediation TEXT,
      updated_at INTEGER NOT NULL
    )
  `);

  await db.exec(`
    CREATE TABLE IF NOT EXISTS pipeline_quality_guardian_v1_checkpoints (
      checkpoint_key TEXT PRIMARY KEY,
      payload_json TEXT NOT NULL,
      updated_at INTEGER NOT NULL
    )
  `);
}

async function loadCheckpoint() {
  const row = await db.queryOne(
    `SELECT payload_json FROM pipeline_quality_guardian_v1_checkpoints WHERE checkpoint_key = ?`,
    [CHECKPOINT_KEY]
  );
  return row ? JSON.parse(row.payload_json) : null;
}

async function saveCheckpoint(payload) {
  await db.exec(
    `INSERT INTO pipeline_quality_guardian_v1_checkpoints(checkpoint_key, payload_json, updated_at)
     VALUES (?, json(?), unixepoch())
     ON CONFLICT(checkpoint_key) DO UPDATE SET payload_json=excluded.payload_json, updated_at=excluded.updated_at`,
    [CHECKPOINT_KEY, JSON.stringify(payload)]
  );
}

async function resumeOrCreate(input) {
  const cp = await loadCheckpoint();
  if (cp && cp.runId) {
    const run = await db.queryOne(`SELECT * FROM pipeline_quality_guardian_v1_runs WHERE run_id=?`, [cp.runId]);
    if (run) return { runId: run.run_id, phase: cp.phase || "triage", threadId: cp.threadId || run.thread_id || null };
  }

  const runId = rid("run");
  await db.exec(
    `INSERT INTO pipeline_quality_guardian_v1_runs(run_id, pipeline_name, status, created_at, updated_at)
     VALUES (?, ?, 'triage', unixepoch(), unixepoch())`,
    [runId, input.pipelineName]
  );

  await db.tx(async () => {
    for (const a of input.anomalies) {
      await db.exec(
        `INSERT INTO pipeline_quality_guardian_v1_anomalies(
           anomaly_id, run_id, metric_name, observed_value, expected_range, status, updated_at
         ) VALUES (?, ?, ?, ?, ?, 'queued', unixepoch())`,
        [rid("anom"), runId, a.metricName, a.observedValue, a.expectedRange]
      );
    }
  });

  await saveCheckpoint({ runId, phase: "triage", threadId: null });
  return { runId, phase: "triage", threadId: null };
}

async function ensureThread(state) {
  if (state.threadId) return state.threadId;
  const started = await session.threads.start();
  state.threadId = session.ids.thread(started);
  await db.exec(`UPDATE pipeline_quality_guardian_v1_runs SET thread_id=?, updated_at=unixepoch() WHERE run_id=?`, [state.threadId, state.runId]);
  await saveCheckpoint(state);
  return state.threadId;
}

async function triageQueuedAnomalies(state) {
  const thread = session.thread(await ensureThread(state));
  const queued = await db.queryAll(
    `SELECT * FROM pipeline_quality_guardian_v1_anomalies WHERE run_id=? AND status='queued' ORDER BY anomaly_id`,
    [state.runId]
  );

  for (const anomaly of queued) {
    await db.exec(
      `UPDATE pipeline_quality_guardian_v1_anomalies SET status='running', updated_at=unixepoch() WHERE anomaly_id=?`,
      [anomaly.anomaly_id]
    );

    const turn = await thread.turn.start({
      input: [{
        type: "text",
        text:
          `You are data reliability analyst.\n` +
          `Anomaly metric=${anomaly.metric_name}, observed=${anomaly.observed_value}, expected=${anomaly.expected_range}.\n` +
          `Return a likely root-cause hypothesis and one remediation action.`
      }],
      outputSchema: {
        type: "object",
        properties: {
          hypothesis: { type: "string" },
          remediation: { type: "string" }
        },
        required: ["hypothesis", "remediation"]
      }
    });

    await thread.turn.waitCompleted({ turnId: session.ids.turn(turn), timeoutMs: 180000 });
    const out = await turn.getStructuredOutput();

    await db.exec(
      `UPDATE pipeline_quality_guardian_v1_anomalies
       SET status='triaged', hypothesis=?, remediation=?, updated_at=unixepoch()
       WHERE anomaly_id=?`,
      [out.hypothesis || "", out.remediation || "", anomaly.anomaly_id]
    );

    ui.emit({ type: "pipeline-quality-anomaly-triaged", runId: state.runId, anomalyId: anomaly.anomaly_id });
  }

  state.phase = "finalize";
  await db.exec(`UPDATE pipeline_quality_guardian_v1_runs SET status='finalize', updated_at=unixepoch() WHERE run_id=?`, [state.runId]);
  await saveCheckpoint(state);
}

async function finalizeRun(state) {
  const totals = await db.queryOne(
    `SELECT COUNT(*) AS total, SUM(CASE WHEN status='triaged' THEN 1 ELSE 0 END) AS triaged
     FROM pipeline_quality_guardian_v1_anomalies WHERE run_id=?`,
    [state.runId]
  );

  await db.exec(`UPDATE pipeline_quality_guardian_v1_runs SET status='complete', updated_at=unixepoch() WHERE run_id=?`, [state.runId]);
  state.phase = "complete";
  await saveCheckpoint(state);

  ui.emit({
    type: "pipeline-quality-complete",
    runId: state.runId,
    totals: { total: totals ? totals.total : 0, triaged: totals ? totals.triaged : 0 }
  });
}

async function main() {
  await initSchema();
  const state = await resumeOrCreate({
    pipelineName: "warehouse-orders-daily",
    anomalies: [
      { metricName: "null_rate.customer_id", observedValue: 0.18, expectedRange: "0.00-0.01" },
      { metricName: "row_count.delta", observedValue: -0.42, expectedRange: "-0.05-0.05" }
    ]
  });

  ui.emit({ type: "pipeline-quality-start", runId: state.runId, phase: state.phase });

  if (state.phase === "triage") {
    await triageQueuedAnomalies(state);
  }
  if (state.phase === "finalize") {
    await finalizeRun(state);
  }
}

main().catch((err) => ui.emit({ type: "pipeline-quality-failed", error: String(err) }));
