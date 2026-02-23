const codex = require("codex");
const rpc = require("rpc");
const ui = require("ui");
const clock = require("clock");

const session = codex.connect();
const seenMethods = [];
let threadId = "";
let turnId = "";
let turnCompleted = false;

function extractThreadId(v) {
  return (
    (v && (v.threadId || v.id)) ||
    (v && v.thread && (v.thread.threadId || v.thread.id)) ||
    ""
  );
}

function extractTurnId(v) {
  return (
    (v && (v.turnId || v.id)) ||
    (v && v.turn && (v.turn.turnId || v.turn.id)) ||
    ""
  );
}

session.onNotification((evt) => {
  const method = evt && evt.method ? evt.method : "";
  const params = (evt && evt.params) || {};
  if (method) {
    seenMethods.push(method);
  }
  if (method === "turn/completed") {
    const notifTurnId = extractTurnId(params.turn || params);
    const notifThreadId = extractThreadId(params.thread || params);
    if ((!turnId || notifTurnId === turnId) && (!threadId || notifThreadId === threadId || !notifThreadId)) {
      turnCompleted = true;
    }
  }
  ui.emit({ type: "module-api-turn-gate-notification", method, params });
});

function waitForTurnCompleted(maxMs) {
  const started = clock.nowMs();
  function loop() {
    if (turnCompleted) {
      return Promise.resolve();
    }
    if (clock.nowMs() - started > maxMs) {
      throw new Error("timed out waiting for matching turn/completed notification");
    }
    return clock.sleep(100).then(loop);
  }
  return loop();
}

rpc.request("thread/start", {
  model: "gpt-5",
  cwd: "/home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server",
  approvalPolicy: "on-request",
  sandbox: "workspace-write"
})
  .then((thread) => {
    threadId = extractThreadId(thread);
    ui.emit({ type: "module-api-turn-gate-thread-start", ok: !!threadId, threadId, thread });
    if (!threadId) {
      throw new Error("thread/start response missing thread id");
    }
    return rpc.request("turn/start", {
      threadId,
      input: [{ type: "text", text: "Say hello in one short sentence." }]
    });
  })
  .then((turn) => {
    turnId = extractTurnId(turn);
    ui.emit({ type: "module-api-turn-gate-turn-start", ok: !!turnId, turnId, turn });
    if (!turnId) {
      throw new Error("turn/start response missing turn id");
    }
    return waitForTurnCompleted(15000);
  })
  .then(() => {
    ui.emit({
      type: "module-api-turn-gate-complete",
      ok: true,
      threadId,
      turnId,
      turnCompleted,
      notifications: seenMethods.slice(0, 50)
    });
  })
  .catch((err) => {
    ui.emit({
      type: "module-api-turn-gate-complete",
      ok: false,
      error: String(err),
      threadId,
      turnId,
      turnCompleted,
      notifications: seenMethods.slice(0, 50)
    });
  });
