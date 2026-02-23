// Sketch: Vendor due-diligence risk review harness (real-world, non-RLM).
// Focus: procurement/security/legal risk scoring with resumable phase checkpoints.

const codex = require("codex");
const db = require("db");
const ui = require("ui");

const CHECKPOINT_KEY = "active";

const session = codex.connect({
  defaults: {
    model: "gpt-5",
    cwd: "/home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server",
    approvalPolicy: "never",
    sandbox: "read-only"
  }
});

session.approvals.setPolicy({
  command: () => "decline",
  fileChange: () => "decline",
  fallback: () => "decline"
});

function rid(prefix) {
  return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 1e6)}`;
}

async function initSchema() {
  await db.exec(`
    CREATE TABLE IF NOT EXISTS vendor_due_diligence_v1_runs (
      run_id TEXT PRIMARY KEY,
      program_name TEXT NOT NULL,
      status TEXT NOT NULL,
      coordinator_thread_id TEXT,
      created_at INTEGER NOT NULL,
      updated_at INTEGER NOT NULL
    )
  `);

  await db.exec(`
    CREATE TABLE IF NOT EXISTS vendor_due_diligence_v1_suppliers (
      supplier_id TEXT PRIMARY KEY,
      run_id TEXT NOT NULL,
      supplier_name TEXT NOT NULL,
      scope_text TEXT,
      status TEXT NOT NULL,
      updated_at INTEGER NOT NULL
    )
  `);

  await db.exec(`
    CREATE TABLE IF NOT EXISTS vendor_due_diligence_v1_findings (
      finding_id TEXT PRIMARY KEY,
      run_id TEXT NOT NULL,
      supplier_id TEXT NOT NULL,
      domain TEXT NOT NULL,
      severity TEXT NOT NULL,
      evidence_text TEXT,
      recommendation_text TEXT,
      status TEXT NOT NULL,
      updated_at INTEGER NOT NULL
    )
  `);

  await db.exec(`
    CREATE TABLE IF NOT EXISTS vendor_due_diligence_v1_checkpoints (
      checkpoint_key TEXT PRIMARY KEY,
      payload_json TEXT NOT NULL,
      updated_at INTEGER NOT NULL
    )
  `);
}

async function loadCheckpoint() {
  const row = await db.queryOne(
    `SELECT payload_json FROM vendor_due_diligence_v1_checkpoints WHERE checkpoint_key = ?`,
    [CHECKPOINT_KEY]
  );
  return row ? JSON.parse(row.payload_json) : null;
}

async function saveCheckpoint(payload) {
  await db.exec(
    `INSERT INTO vendor_due_diligence_v1_checkpoints(checkpoint_key, payload_json, updated_at)
     VALUES (?, json(?), unixepoch())
     ON CONFLICT(checkpoint_key) DO UPDATE SET payload_json=excluded.payload_json, updated_at=excluded.updated_at`,
    [CHECKPOINT_KEY, JSON.stringify(payload)]
  );
}

async function resumeOrCreate(programName, suppliers) {
  const cp = await loadCheckpoint();
  if (cp && cp.runId) {
    const run = await db.queryOne(`SELECT * FROM vendor_due_diligence_v1_runs WHERE run_id=?`, [cp.runId]);
    if (run) return { runId: run.run_id, phase: cp.phase || "ingest", coordinatorThreadId: cp.coordinatorThreadId || null };
  }

  const runId = rid("run");
  await db.exec(
    `INSERT INTO vendor_due_diligence_v1_runs(run_id, program_name, status, created_at, updated_at)
     VALUES (?, ?, 'ingest', unixepoch(), unixepoch())`,
    [runId, programName]
  );

  await db.tx(async () => {
    for (const s of suppliers) {
      await db.exec(
        `INSERT INTO vendor_due_diligence_v1_suppliers(supplier_id, run_id, supplier_name, scope_text, status, updated_at)
         VALUES (?, ?, ?, ?, 'queued', unixepoch())`,
        [rid("supplier"), runId, s.name, s.scope || ""]
      );
    }
  });

  await saveCheckpoint({ runId, phase: "ingest", coordinatorThreadId: null });
  return { runId, phase: "ingest", coordinatorThreadId: null };
}

async function ensureCoordinatorThread(state) {
  if (state.coordinatorThreadId) return state.coordinatorThreadId;
  const started = await session.threads.start();
  const threadId = session.ids.thread(started);
  await db.exec(
    `UPDATE vendor_due_diligence_v1_runs SET coordinator_thread_id=?, updated_at=unixepoch() WHERE run_id=?`,
    [threadId, state.runId]
  );
  state.coordinatorThreadId = threadId;
  await saveCheckpoint({ ...state, phase: state.phase || "ingest" });
  return threadId;
}

async function analyzeSuppliers(state) {
  const thread = session.thread(await ensureCoordinatorThread(state));
  const suppliers = await db.queryAll(
    `SELECT * FROM vendor_due_diligence_v1_suppliers WHERE run_id=? AND status!='done' ORDER BY supplier_id`,
    [state.runId]
  );

  for (const supplier of suppliers) {
    await db.exec(`UPDATE vendor_due_diligence_v1_suppliers SET status='running', updated_at=unixepoch() WHERE supplier_id=?`, [supplier.supplier_id]);

    const turn = await thread.turn.start({
      input: [{
        type: "text",
        text:
          `Perform due diligence triage for supplier:\n` +
          `name=${supplier.supplier_name}\n` +
          `scope=${supplier.scope_text}\n\n` +
          `Return procurement, security, legal, and privacy findings with severity and recommendations.`
      }],
      outputSchema: {
        type: "object",
        properties: {
          findings: {
            type: "array",
            items: {
              type: "object",
              properties: {
                domain: { type: "string" },
                severity: { type: "string" },
                evidence: { type: "string" },
                recommendation: { type: "string" }
              },
              required: ["domain", "severity", "recommendation"]
            }
          }
        },
        required: ["findings"]
      }
    });

    await thread.turn.waitCompleted({ turnId: session.ids.turn(turn), timeoutMs: 240000 });
    const out = await turn.getStructuredOutput();

    await db.tx(async () => {
      for (const f of out.findings || []) {
        await db.exec(
          `INSERT INTO vendor_due_diligence_v1_findings(
             finding_id, run_id, supplier_id, domain, severity, evidence_text, recommendation_text, status, updated_at
           ) VALUES (?, ?, ?, ?, ?, ?, ?, 'open', unixepoch())`,
          [
            rid("finding"),
            state.runId,
            supplier.supplier_id,
            f.domain || "general",
            f.severity || "medium",
            f.evidence || "",
            f.recommendation || ""
          ]
        );
      }
      await db.exec(`UPDATE vendor_due_diligence_v1_suppliers SET status='done', updated_at=unixepoch() WHERE supplier_id=?`, [supplier.supplier_id]);
    });

    ui.emit({ type: "vendor-risk-supplier-analyzed", runId: state.runId, supplierId: supplier.supplier_id });
  }

  state.phase = "memo";
  await db.exec(`UPDATE vendor_due_diligence_v1_runs SET status='memo', updated_at=unixepoch() WHERE run_id=?`, [state.runId]);
  await saveCheckpoint(state);
}

async function writeDecisionMemo(state) {
  const thread = session.thread(state.coordinatorThreadId);
  const findings = await db.queryAll(
    `SELECT supplier_id, domain, severity, recommendation_text FROM vendor_due_diligence_v1_findings WHERE run_id=? ORDER BY supplier_id`,
    [state.runId]
  );

  const memoTurn = await thread.turn.start({
    input: [{
      type: "text",
      text:
        `Create an executive decision memo for onboarding risk.\n` +
        `Include: accept/conditional/reject recommendation per supplier and required controls.\n\n` +
        `${JSON.stringify(findings)}`
    }]
  });

  await thread.turn.waitCompleted({ turnId: session.ids.turn(memoTurn), timeoutMs: 180000 });

  state.phase = "complete";
  await db.exec(`UPDATE vendor_due_diligence_v1_runs SET status='complete', updated_at=unixepoch() WHERE run_id=?`, [state.runId]);
  await saveCheckpoint(state);
  ui.emit({ type: "vendor-risk-complete", runId: state.runId });
}

async function main() {
  await initSchema();

  const state = await resumeOrCreate("Q2 strategic vendors", [
    { name: "Delta Data Systems", scope: "Analytics processing and storage" },
    { name: "Northwind Messaging", scope: "Transactional notification delivery" }
  ]);

  ui.emit({ type: "vendor-risk-start", runId: state.runId, phase: state.phase });

  if (state.phase === "ingest") {
    await analyzeSuppliers(state);
  }

  if (state.phase === "memo") {
    await writeDecisionMemo(state);
  }
}

main().catch((err) => ui.emit({ type: "vendor-risk-failed", error: String(err) }));
