// Sketch: Customer support escalation harness (real-world, non-RLM).
// Focus: durable escalation state, customer update drafts, engineering action items, and resumable phases.

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

async function initDb() {
  await db.exec(`
    CREATE TABLE IF NOT EXISTS support_escalation_v1_cases (
      case_id TEXT PRIMARY KEY,
      customer_name TEXT NOT NULL,
      severity TEXT NOT NULL,
      status TEXT NOT NULL,
      thread_id TEXT,
      created_at INTEGER NOT NULL,
      updated_at INTEGER NOT NULL
    )
  `);

  await db.exec(`
    CREATE TABLE IF NOT EXISTS support_escalation_v1_updates (
      update_id TEXT PRIMARY KEY,
      case_id TEXT NOT NULL,
      audience TEXT NOT NULL,
      message_text TEXT NOT NULL,
      created_at INTEGER NOT NULL
    )
  `);

  await db.exec(`
    CREATE TABLE IF NOT EXISTS support_escalation_v1_actions (
      action_id TEXT PRIMARY KEY,
      case_id TEXT NOT NULL,
      owner TEXT,
      action_text TEXT NOT NULL,
      priority TEXT NOT NULL,
      status TEXT NOT NULL,
      updated_at INTEGER NOT NULL
    )
  `);

  await db.exec(`
    CREATE TABLE IF NOT EXISTS support_escalation_v1_checkpoints (
      checkpoint_key TEXT PRIMARY KEY,
      payload_json TEXT NOT NULL,
      updated_at INTEGER NOT NULL
    )
  `);
}

async function loadCheckpoint() {
  const row = await db.queryOne(
    `SELECT payload_json FROM support_escalation_v1_checkpoints WHERE checkpoint_key = ?`,
    [CHECKPOINT_KEY]
  );
  return row ? JSON.parse(row.payload_json) : null;
}

async function saveCheckpoint(payload) {
  await db.exec(
    `INSERT INTO support_escalation_v1_checkpoints(checkpoint_key, payload_json, updated_at)
     VALUES (?, json(?), unixepoch())
     ON CONFLICT(checkpoint_key) DO UPDATE SET payload_json=excluded.payload_json, updated_at=excluded.updated_at`,
    [CHECKPOINT_KEY, JSON.stringify(payload)]
  );
}

async function resumeOrCreate(caseInput) {
  const cp = await loadCheckpoint();
  if (cp && cp.caseId) {
    const existing = await db.queryOne(`SELECT * FROM support_escalation_v1_cases WHERE case_id=?`, [cp.caseId]);
    if (existing) {
      return {
        caseId: existing.case_id,
        phase: cp.phase || "investigate",
        threadId: cp.threadId || existing.thread_id || null,
        caseInput
      };
    }
  }

  const caseId = rid("case");
  await db.exec(
    `INSERT INTO support_escalation_v1_cases(case_id, customer_name, severity, status, created_at, updated_at)
     VALUES (?, ?, ?, 'investigating', unixepoch(), unixepoch())`,
    [caseId, caseInput.customerName, caseInput.severity]
  );

  const state = { caseId, phase: "investigate", threadId: null, caseInput };
  await saveCheckpoint(state);
  return state;
}

async function ensureThread(state) {
  if (state.threadId) return state.threadId;
  const started = await session.threads.start();
  state.threadId = session.ids.thread(started);
  await db.exec(`UPDATE support_escalation_v1_cases SET thread_id=?, updated_at=unixepoch() WHERE case_id=?`, [state.threadId, state.caseId]);
  await saveCheckpoint(state);
  return state.threadId;
}

async function investigate(state) {
  const thread = session.thread(await ensureThread(state));

  const investigateTurn = await thread.turn.start({
    input: [{
      type: "text",
      text:
        `Support escalation context:\n${JSON.stringify(state.caseInput)}\n\n` +
        `Return:\n` +
        `1) likely root causes\n` +
        `2) immediate customer-safe mitigations\n` +
        `3) internal engineering action items\n` +
        `4) a short customer-facing status update`
    }],
    outputSchema: {
      type: "object",
      properties: {
        likelyRootCauses: { type: "array", items: { type: "string" } },
        mitigations: { type: "array", items: { type: "string" } },
        actions: {
          type: "array",
          items: {
            type: "object",
            properties: {
              owner: { type: "string" },
              action: { type: "string" },
              priority: { type: "string" }
            },
            required: ["action", "priority"]
          }
        },
        customerUpdate: { type: "string" }
      },
      required: ["actions", "customerUpdate"]
    }
  });

  await thread.turn.waitCompleted({ turnId: session.ids.turn(investigateTurn), timeoutMs: 180000 });
  const out = await investigateTurn.getStructuredOutput();

  await db.tx(async () => {
    await db.exec(
      `INSERT INTO support_escalation_v1_updates(update_id, case_id, audience, message_text, created_at)
       VALUES (?, ?, 'customer', ?, unixepoch())`,
      [rid("upd"), state.caseId, out.customerUpdate || ""]
    );

    for (const a of out.actions || []) {
      await db.exec(
        `INSERT INTO support_escalation_v1_actions(action_id, case_id, owner, action_text, priority, status, updated_at)
         VALUES (?, ?, ?, ?, ?, 'open', unixepoch())`,
        [rid("act"), state.caseId, a.owner || null, a.action || "", a.priority || "medium"]
      );
    }

    await db.exec(`UPDATE support_escalation_v1_cases SET status='response-drafted', updated_at=unixepoch() WHERE case_id=?`, [state.caseId]);
  });

  state.phase = "complete";
  await saveCheckpoint(state);
  ui.emit({ type: "support-escalation-complete", caseId: state.caseId, customerUpdate: out.customerUpdate || "" });
}

async function main() {
  await initDb();
  const state = await resumeOrCreate({
    customerName: "Acme Health",
    severity: "high",
    ticketId: "SUP-98231",
    symptoms: ["Intermittent timeouts", "Dashboard stale by 15 minutes"],
    environment: "production-us"
  });

  ui.emit({ type: "support-escalation-start", caseId: state.caseId, phase: state.phase, threadId: state.threadId || null });

  if (state.phase === "investigate") {
    await investigate(state);
  }
}

main().catch((err) => ui.emit({ type: "support-escalation-failed", error: String(err) }));
