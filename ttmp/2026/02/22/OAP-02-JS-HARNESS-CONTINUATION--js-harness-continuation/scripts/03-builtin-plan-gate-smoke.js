const codex = require("codex");
const ui = require("ui");
const clock = require("clock");

const session = codex.connect();
let gateTriggered = false;
let gateAction = "none";

session.onNotification((evt) => {
  const method = evt && evt.method ? evt.method : "";
  const params = (evt && evt.params) || {};

  if (method !== "turn/plan/updated") {
    return;
  }

  const threadId = params.threadId || "";
  const turnId = params.turnId || "";
  if (!threadId || !turnId || gateTriggered) {
    return;
  }

  gateTriggered = true;
  gateAction = "steer";
  ui.emit({ type: "plan-gate-smoke-plan", threadId, turnId, plan: params.plan || null });

  const handle = session.thread(threadId);
  handle.turn
    .steer({
      expectedTurnId: turnId,
      input: [{ type: "text", text: "Plan approved. Continue with implementation now." }]
    })
    .then(() => {
      ui.emit({ type: "plan-gate-smoke-steer-sent", threadId, turnId });
    })
    .catch((err) => {
      gateAction = "error";
      ui.emit({ type: "plan-gate-smoke-steer-error", error: String(err), threadId, turnId });
    });
});

ui.emit({ type: "plan-gate-smoke-start" });

session
  .threads
  .start({
    model: "gpt-5",
    cwd: "/home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server",
    approvalPolicy: "on-request",
    sandbox: "workspace-write"
  })
  .then((startResult) => {
    const thread = (startResult && startResult.thread) || startResult || {};
    const threadId = thread.id || thread.threadId || "";
    if (!threadId) {
      throw new Error("thread/start result missing id");
    }

    ui.emit({ type: "plan-gate-smoke-thread-started", threadId });

    const handle = session.thread(threadId);
    return handle.turn.start({
      input: [
        {
          type: "text",
          text: "First output a short step-by-step plan, then execute it."
        }
      ]
    });
  })
  .then((turnResult) => {
    ui.emit({ type: "plan-gate-smoke-turn-started", turnResult });
    return clock.sleep(15000);
  })
  .then(() => {
    ui.emit({ type: "plan-gate-smoke-complete", ok: true, gateTriggered, gateAction });
  })
  .catch((err) => {
    ui.emit({ type: "plan-gate-smoke-complete", ok: false, error: String(err), gateTriggered, gateAction });
  });
