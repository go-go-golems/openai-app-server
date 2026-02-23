const codex = require("codex");
const approval = require("approval");
const ui = require("ui");
const clock = require("clock");

const SAFE_PREFIXES = ["go test", "npm test", "pnpm test", "pytest", "make test"];

function isSafeCommand(command) {
  if (!command) {
    return false;
  }
  for (let i = 0; i < SAFE_PREFIXES.length; i += 1) {
    if (command.indexOf(SAFE_PREFIXES[i]) === 0) {
      return true;
    }
  }
  return false;
}

const session = codex.connect();
const decisions = [];

session.onRequest((evt) => {
  const id = evt && evt.id;
  const method = evt && evt.method ? evt.method : "";
  const params = (evt && evt.params) || {};

  if (method === "item/commandExecution/requestApproval") {
    const command = (params && params.command) || "";
    const decision = isSafeCommand(command) ? "acceptForSession" : "decline";
    if (decision === "acceptForSession") {
      approval.acceptForSession(id);
    } else {
      approval.decline(id);
    }
    decisions.push({ method, decision, command });
    ui.emit({ type: "autopilot-smoke-decision", method, decision, command });
    return;
  }

  if (method === "item/fileChange/requestApproval") {
    approval.decline(id);
    decisions.push({ method, decision: "decline" });
    ui.emit({ type: "autopilot-smoke-decision", method, decision: "decline" });
    return;
  }
});

ui.emit({ type: "autopilot-smoke-start" });

session
  .threads
  .start({
    model: "gpt-5",
    cwd: "/home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server",
    approvalPolicy: "on-request",
    sandbox: "read-only"
  })
  .then((startResult) => {
    const thread = (startResult && startResult.thread) || startResult || {};
    const threadId = thread.id || thread.threadId || "";
    if (!threadId) {
      throw new Error("thread/start result missing id");
    }
    ui.emit({ type: "autopilot-smoke-thread-started", threadId });

    const handle = session.thread(threadId);
    return handle.turn.start({
      input: [
        {
          type: "text",
          text: "Run `curl -I https://example.com` and reply with one short sentence."
        }
      ]
    });
  })
  .then((turnResult) => {
    ui.emit({ type: "autopilot-smoke-turn-started", turnResult });
    return clock.sleep(12000);
  })
  .then(() => {
    ui.emit({ type: "autopilot-smoke-complete", ok: true, decisions });
  })
  .catch((err) => {
    ui.emit({ type: "autopilot-smoke-complete", ok: false, error: String(err), decisions });
  });
