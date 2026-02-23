const codex = require("codex");
const session = codex.connect();

session.onNotification((evt) => {
  __host.ui.emit({ type: "live-notif", method: evt.method, params: evt.params || {} });
});

session.request("thread/start", {
  model: "gpt-5",
  cwd: "/home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server",
  approvalPolicy: "on-request",
  sandbox: "workspace-write"
}).then((thread) => {
  __host.ui.emit({ type: "thread-start-result", ok: true, thread });
  const threadId = (thread && (thread.threadId || thread.id || (thread.thread && thread.thread.id))) || "";
  if (!threadId) {
    throw new Error("no thread id in thread/start response");
  }

  return session.request("turn/start", {
    threadId,
    input: [{ type: "text", text: "Say hello in one short sentence." }]
  });
}).then((turn) => {
  __host.ui.emit({ type: "turn-start-result", ok: true, turn });
}).catch((err) => {
  __host.ui.emit({ type: "second-real-test-error", ok: false, error: String(err) });
});
