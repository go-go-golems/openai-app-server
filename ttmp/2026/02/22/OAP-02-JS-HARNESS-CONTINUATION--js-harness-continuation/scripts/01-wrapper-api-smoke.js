const codex = require("codex");
const ui = require("ui");
const clock = require("clock");

const session = codex.connect();

ui.emit({ type: "wrapper-smoke-start" });

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

    ui.emit({ type: "wrapper-smoke-thread-started", threadId, startResult });

    const handle = session.thread(threadId);
    return handle.turn.start({
      input: [
        { type: "text", text: "Reply with exactly one short sentence: wrapper API smoke ok." }
      ]
    });
  })
  .then((turnResult) => {
    ui.emit({ type: "wrapper-smoke-turn-started", turnResult });
    return clock.sleep(6000);
  })
  .then(() => {
    ui.emit({ type: "wrapper-smoke-complete", ok: true });
  })
  .catch((err) => {
    ui.emit({ type: "wrapper-smoke-complete", ok: false, error: String(err) });
  });
