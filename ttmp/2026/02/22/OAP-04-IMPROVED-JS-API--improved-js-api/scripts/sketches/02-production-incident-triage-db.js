// Sketch: Production incident triage harness (real-world, non-RLM).
// Focus: durable incident timeline, hypothesis tracking, remediation planning.

const codex = require("codex");
const db = require("db");
const ui = require("ui");

const SCRIPT_ID = "incident_triage_v1";
const INCIDENT_KEY = "active";

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
    // Example: allow diagnostics, block mutating shell commands by default.
    if (cmd.startsWith("date") || cmd.startsWith("ls ") || cmd.startsWith("git status")) {
      return "acceptForSession";
    }
    return "decline";
  },
  fileChange: () => "decline",
  fallback: () => "decline"
});

async function initDb() {
  await db.exec(`
    CREATE TABLE IF NOT EXISTS incident_triage_v1_incidents (
      incident_id TEXT PRIMARY KEY,
      title TEXT NOT NULL,
      severity TEXT NOT NULL,
      status TEXT NOT NULL,
      thread_id TEXT,
      started_at INTEGER NOT NULL,
      updated_at INTEGER NOT NULL
    )
  `);

  await db.exec(`
    CREATE TABLE IF NOT EXISTS incident_triage_v1_timeline (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      incident_id TEXT NOT NULL,
      ts INTEGER NOT NULL,
      stage TEXT NOT NULL,
      message TEXT NOT NULL,
      payload_json TEXT
    )
  `);

  await db.exec(`
    CREATE TABLE IF NOT EXISTS incident_triage_v1_hypotheses (
      hypothesis_id TEXT PRIMARY KEY,
      incident_id TEXT NOT NULL,
      statement TEXT NOT NULL,
      confidence REAL,
      status TEXT NOT NULL,
      updated_at INTEGER NOT NULL
    )
  `);

  await db.exec(`
    CREATE TABLE IF NOT EXISTS incident_triage_v1_checkpoints (
      checkpoint_key TEXT PRIMARY KEY,
      payload_json TEXT NOT NULL,
      updated_at INTEGER NOT NULL
    )
  `);
}

async function logTimeline(incidentId, stage, message, payload) {
  await db.exec(
    `INSERT INTO incident_triage_v1_timeline(incident_id, ts, stage, message, payload_json)
     VALUES (?, unixepoch(), ?, ?, json(?))`,
    [incidentId, stage, message, JSON.stringify(payload || {})]
  );
}

async function loadCheckpoint() {
  const row = await db.queryOne(
    `SELECT payload_json FROM incident_triage_v1_checkpoints WHERE checkpoint_key = ?`,
    [INCIDENT_KEY]
  );
  return row ? JSON.parse(row.payload_json) : null;
}

async function saveCheckpoint(payload) {
  await db.exec(
    `INSERT INTO incident_triage_v1_checkpoints(checkpoint_key, payload_json, updated_at)
     VALUES (?, json(?), unixepoch())
     ON CONFLICT(checkpoint_key) DO UPDATE SET payload_json=excluded.payload_json, updated_at=excluded.updated_at`,
    [INCIDENT_KEY, JSON.stringify(payload)]
  );
}

function incidentId() {
  return `inc_${Date.now()}_${Math.floor(Math.random() * 1e6)}`;
}

async function ensureIncident(title, severity) {
  const cp = await loadCheckpoint();
  if (cp && cp.incidentId) {
    const existing = await db.queryOne(`SELECT * FROM incident_triage_v1_incidents WHERE incident_id = ?`, [cp.incidentId]);
    if (existing) return existing;
  }

  const id = incidentId();
  await db.exec(
    `INSERT INTO incident_triage_v1_incidents(incident_id, title, severity, status, started_at, updated_at)
     VALUES (?, ?, ?, 'triage', unixepoch(), unixepoch())`,
    [id, title, severity]
  );
  await saveCheckpoint({ incidentId: id, phase: "triage" });
  return db.queryOne(`SELECT * FROM incident_triage_v1_incidents WHERE incident_id = ?`, [id]);
}

async function runIncidentTriage({ title, severity, alertContext }) {
  await initDb();
  const incident = await ensureIncident(title, severity);

  const started = await session.threads.start();
  const threadId = session.ids.thread(started);
  const thread = session.thread(threadId);

  await db.exec(`UPDATE incident_triage_v1_incidents SET thread_id=?, updated_at=unixepoch() WHERE incident_id=?`, [threadId, incident.incident_id]);
  await logTimeline(incident.incident_id, "thread-start", "Triage thread started", { threadId });

  const triageTurn = await thread.turn.start({
    input: [{
      type: "text",
      text:
        `You are incident commander assistant.\n` +
        `Incident: ${title} (severity ${severity})\n` +
        `Alert context:\n${JSON.stringify(alertContext)}\n\n` +
        `Return: likely causes, immediate containment actions, and top 3 diagnostic commands.`
    }],
    outputSchema: {
      type: "object",
      properties: {
        causes: { type: "array", items: { type: "string" } },
        containment: { type: "array", items: { type: "string" } },
        diagnostics: { type: "array", items: { type: "string" } }
      },
      required: ["causes", "containment"]
    }
  });

  await thread.turn.waitCompleted({ turnId: session.ids.turn(triageTurn), timeoutMs: 180000 });
  const result = await triageTurn.getStructuredOutput();

  await db.tx(async () => {
    for (const c of result.causes || []) {
      const hid = `hyp_${Date.now()}_${Math.floor(Math.random() * 1e6)}`;
      await db.exec(
        `INSERT INTO incident_triage_v1_hypotheses(hypothesis_id, incident_id, statement, confidence, status, updated_at)
         VALUES (?, ?, ?, ?, 'open', unixepoch())`,
        [hid, incident.incident_id, c, null]
      );
    }
  });

  await logTimeline(incident.incident_id, "triage-result", "Initial triage complete", result);
  await saveCheckpoint({ incidentId: incident.incident_id, phase: "containment-planning", threadId });

  ui.emit({ type: "incident-triage-complete", incidentId: incident.incident_id, threadId, result });
}

runIncidentTriage({
  title: "Elevated 5xx errors on /checkout",
  severity: "SEV-1",
  alertContext: { service: "payments-api", errorRate: 0.27, region: "us-east-1" }
}).catch((err) => ui.emit({ type: "incident-triage-failed", error: String(err) }));
