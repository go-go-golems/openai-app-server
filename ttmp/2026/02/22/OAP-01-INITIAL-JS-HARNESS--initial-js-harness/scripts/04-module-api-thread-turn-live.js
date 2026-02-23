const codex = require("codex");
const rpc = require("rpc");
const ui = require("ui");
const clock = require("clock");

const session = codex.connect();
const seenMethods = [];
let threadId = "";
let turnId = "";

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
  if (method) {
    seenMethods.push(method);
  }
  ui.emit({ type: "module-api-thread-turn-notification", method, params: (evt && evt.params) || {} });
});

rpc.request("thread/start", {
  model: "gpt-5",
  cwd: "/home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server",
  approvalPolicy: "on-request",
  sandbox: "workspace-write"
})
  .then((thread) => {
    threadId = extractThreadId(thread);
    ui.emit({ type: "module-api-thread-start", ok: !!threadId, threadId, thread });
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
    ui.emit({ type: "module-api-turn-start", ok: !!turnId, turnId, turn });
    return clock.sleep(1500);
  })
  .then(() => {
    ui.emit({
      type: "module-api-thread-turn-complete",
      ok: true,
      threadId,
      turnId,
      notifications: seenMethods.slice(0, 30)
    });
  })
  .catch((err) => {
    ui.emit({
      type: "module-api-thread-turn-complete",
      ok: false,
      error: String(err),
      threadId,
      turnId,
      notifications: seenMethods.slice(0, 30)
    });
  });
