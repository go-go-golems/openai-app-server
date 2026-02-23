// Sketch: Scientific RLM-style research harness with improved API + db.* storage.
// Target APIs: session.ids, thread.turn.waitCompleted, session.approvals.setPolicy, db.exec/queryOne/queryAll/tx.

const codex = require("codex");
const db = require("db");
const ui = require("ui");

const SCRIPT_ID = "science_rlm_v1";
const CHECKPOINT_KEY = "default";
const MAX_WORKERS = 3;

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
    CREATE TABLE IF NOT EXISTS science_rlm_v1_jobs (
      job_id TEXT PRIMARY KEY,
      goal_text TEXT NOT NULL,
      manager_thread_id TEXT,
      status TEXT NOT NULL,
      created_at INTEGER NOT NULL,
      updated_at INTEGER NOT NULL
    )
  `);

  await db.exec(`
    CREATE TABLE IF NOT EXISTS science_rlm_v1_subtasks (
      subtask_id TEXT PRIMARY KEY,
      job_id TEXT NOT NULL,
      title TEXT NOT NULL,
      question TEXT NOT NULL,
      worker_thread_id TEXT,
      status TEXT NOT NULL,
      summary_text TEXT,
      updated_at INTEGER NOT NULL
    )
  `);

  await db.exec(`
    CREATE TABLE IF NOT EXISTS science_rlm_v1_checkpoints (
      checkpoint_key TEXT PRIMARY KEY,
      payload_json TEXT NOT NULL,
      updated_at INTEGER NOT NULL
    )
  `);
}

async function saveCheckpoint(payload) {
  await db.exec(
    `INSERT INTO science_rlm_v1_checkpoints(checkpoint_key, payload_json, updated_at)
     VALUES (?, json(?), unixepoch())
     ON CONFLICT(checkpoint_key) DO UPDATE SET
       payload_json=excluded.payload_json,
       updated_at=excluded.updated_at`,
    [CHECKPOINT_KEY, JSON.stringify(payload)]
  );
}

async function loadCheckpoint() {
  const row = await db.queryOne(
    `SELECT payload_json FROM science_rlm_v1_checkpoints WHERE checkpoint_key = ?`,
    [CHECKPOINT_KEY]
  );
  return row ? JSON.parse(row.payload_json) : null;
}

async function ensureJob(goalText) {
  const cp = await loadCheckpoint();
  if (cp && cp.jobId) {
    const existing = await db.queryOne(`SELECT * FROM science_rlm_v1_jobs WHERE job_id = ?`, [cp.jobId]);
    if (existing) return existing;
  }

  const jobId = rid("job");
  await db.exec(
    `INSERT INTO science_rlm_v1_jobs(job_id, goal_text, status, created_at, updated_at)
     VALUES (?, ?, 'planning', unixepoch(), unixepoch())`,
    [jobId, goalText]
  );
  await saveCheckpoint({ jobId, phase: "planning" });
  return db.queryOne(`SELECT * FROM science_rlm_v1_jobs WHERE job_id = ?`, [jobId]);
}

async function ensureManagerThread(job) {
  if (job.manager_thread_id) return job.manager_thread_id;
  const started = await session.threads.start();
  const managerThreadId = session.ids.thread(started);
  await db.exec(`UPDATE science_rlm_v1_jobs SET manager_thread_id=?, updated_at=unixepoch() WHERE job_id=?`, [managerThreadId, job.job_id]);
  return managerThreadId;
}

async function planSubtasks(job, managerThreadId) {
  const existing = await db.queryAll(`SELECT * FROM science_rlm_v1_subtasks WHERE job_id = ?`, [job.job_id]);
  if (existing.length > 0) return existing;

  const manager = session.thread(managerThreadId);
  const planTurn = await manager.turn.start({
    input: [{
      type: "text",
      text:
        `Create 4-6 research subtasks for this scientific goal:\n${job.goal_text}\n\n` +
        `Each subtask must have a title and a precise research question. No fabricated citations.`
    }],
    outputSchema: {
      type: "object",
      properties: {
        subtasks: {
          type: "array",
          items: {
            type: "object",
            properties: {
              subtaskId: { type: "string" },
              title: { type: "string" },
              question: { type: "string" }
            },
            required: ["title", "question"]
          }
        }
      },
      required: ["subtasks"]
    }
  });

  await manager.turn.waitCompleted({ turnId: session.ids.turn(planTurn), timeoutMs: 180000 });
  const out = await planTurn.getStructuredOutput();
  const subtasks = out.subtasks || [];

  await db.tx(async () => {
    for (const st of subtasks) {
      await db.exec(
        `INSERT OR REPLACE INTO science_rlm_v1_subtasks(subtask_id, job_id, title, question, status, updated_at)
         VALUES (?, ?, ?, ?, 'queued', unixepoch())`,
        [st.subtaskId || rid("subtask"), job.job_id, st.title || "", st.question || ""]
      );
    }
    await db.exec(`UPDATE science_rlm_v1_jobs SET status='workers-running', updated_at=unixepoch() WHERE job_id=?`, [job.job_id]);
  });

  await saveCheckpoint({ jobId: job.job_id, phase: "workers-running", managerThreadId });
  return db.queryAll(`SELECT * FROM science_rlm_v1_subtasks WHERE job_id = ?`, [job.job_id]);
}

async function runWorker(job, subtask) {
  await db.exec(`UPDATE science_rlm_v1_subtasks SET status='running', updated_at=unixepoch() WHERE subtask_id=?`, [subtask.subtask_id]);

  const workerStarted = await session.threads.start({ sandbox: "read-only", approvalPolicy: "never" });
  const workerThreadId = session.ids.thread(workerStarted);
  await db.exec(`UPDATE science_rlm_v1_subtasks SET worker_thread_id=?, updated_at=unixepoch() WHERE subtask_id=?`, [workerThreadId, subtask.subtask_id]);

  const worker = session.thread(workerThreadId);
  const turn = await worker.turn.start({
    input: [{
      type: "text",
      text:
        `Research this question with evidence and explicit limitations:\n${subtask.question}\n\n` +
        `Return short structured findings and confidence.`
    }],
    outputSchema: {
      type: "object",
      properties: {
        summary: { type: "string" },
        confidence: { type: "number" }
      },
      required: ["summary"]
    }
  });

  await worker.turn.waitCompleted({ turnId: session.ids.turn(turn), timeoutMs: 240000 });
  const result = await turn.getStructuredOutput();

  await db.exec(
    `UPDATE science_rlm_v1_subtasks
     SET status='done', summary_text=?, updated_at=unixepoch()
     WHERE subtask_id=?`,
    [JSON.stringify(result), subtask.subtask_id]
  );

  ui.emit({ type: "science-rlm-worker-done", subtaskId: subtask.subtask_id, workerThreadId });
}

async function runWorkers(job) {
  while (true) {
    const queued = await db.queryAll(
      `SELECT * FROM science_rlm_v1_subtasks WHERE job_id=? AND status='queued' ORDER BY subtask_id LIMIT ?`,
      [job.job_id, MAX_WORKERS]
    );
    if (queued.length === 0) break;

    await Promise.all(queued.map(async (s) => {
      try {
        await runWorker(job, s);
      } catch (err) {
        await db.exec(
          `UPDATE science_rlm_v1_subtasks SET status='failed', summary_text=?, updated_at=unixepoch() WHERE subtask_id=?`,
          [String(err), s.subtask_id]
        );
      }
    }));
  }
}

async function synthesize(job, managerThreadId) {
  const findings = await db.queryAll(
    `SELECT subtask_id, title, summary_text FROM science_rlm_v1_subtasks WHERE job_id=? ORDER BY subtask_id`,
    [job.job_id]
  );

  const manager = session.thread(managerThreadId);
  const synthTurn = await manager.turn.start({
    input: [{
      type: "text",
      text:
        `Synthesize these scientific findings into:\n` +
        `1) consensus\n2) contested claims\n3) evidence gaps\n4) recommended next studies\n\n` +
        `${JSON.stringify(findings)}`
    }]
  });

  await manager.turn.waitCompleted({ turnId: session.ids.turn(synthTurn), timeoutMs: 180000 });
  await db.exec(`UPDATE science_rlm_v1_jobs SET status='complete', updated_at=unixepoch() WHERE job_id=?`, [job.job_id]);
  await saveCheckpoint({ jobId: job.job_id, phase: "complete", managerThreadId });
  ui.emit({ type: "science-rlm-complete", jobId: job.job_id });
}

async function main(goalText) {
  await initDb();
  const job = await ensureJob(goalText);
  const managerThreadId = await ensureManagerThread(job);

  ui.emit({ type: "science-rlm-start", jobId: job.job_id, managerThreadId, goalText });

  await planSubtasks(job, managerThreadId);
  await runWorkers(job);
  await synthesize(job, managerThreadId);
}

main("What does recent evidence suggest about microbiome interventions for depression outcomes?")
  .catch((err) => ui.emit({ type: "science-rlm-failed", error: String(err) }));
