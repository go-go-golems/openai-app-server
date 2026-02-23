const codex = require("codex");
const approval = require("approval");
const ui = require("ui");
const clock = require("clock");

const session = codex.connect();

const summary = {
  requestsObserved: 0,
  approvalsAccepted: 0,
  approvalsDeclined: 0,
  turnCompleted: false,
  threadReadWorked: false,
  errors: []
};

function asString(v) {
  if (typeof v === "string") {
    return v;
  }
  return "";
}

session.onRequest((evt) => {
  const id = evt && evt.id;
  const method = asString(evt && evt.method);
  const params = (evt && evt.params) || {};

  summary.requestsObserved += 1;

  try {
    if (method === "item/commandExecution/requestApproval") {
      const command = asString(params.command);
      if (command.indexOf("curl") === 0 || command.indexOf("curl ") >= 0) {
        approval.acceptForSession(id);
        summary.approvalsAccepted += 1;
        ui.emit({ type: "final-full-smoke-approval", method, decision: "acceptForSession", command });
      } else {
        approval.decline(id);
        summary.approvalsDeclined += 1;
        ui.emit({ type: "final-full-smoke-approval", method, decision: "decline", command });
      }
    } else if (method === "item/fileChange/requestApproval") {
      approval.decline(id);
      summary.approvalsDeclined += 1;
      ui.emit({ type: "final-full-smoke-approval", method, decision: "decline" });
    }
  } catch (err) {
    summary.errors.push(String(err));
    ui.emit({ type: "final-full-smoke-approval-error", error: String(err), method });
  }
});

session.onNotification((evt) => {
  const method = asString(evt && evt.method);
  const params = (evt && evt.params) || {};

  if (method === "turn/completed") {
    summary.turnCompleted = true;
    ui.emit({ type: "final-full-smoke-turn-completed", params });
  }
});

ui.emit({ type: "final-full-smoke-start" });

let threadId = "";

session.threads
  .start({
    model: "gpt-5",
    cwd: "/home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server",
    approvalPolicy: "on-request",
    sandbox: "read-only"
  })
  .then((startResult) => {
    const thread = (startResult && startResult.thread) || startResult || {};
    threadId = asString(thread.id || thread.threadId);
    if (!threadId) {
      throw new Error("thread/start result missing thread id");
    }

    ui.emit({ type: "final-full-smoke-thread-started", threadId });

    const handle = session.thread(threadId);
    return handle.turn.start({
      input: [
        {
          type: "text",
          text: "Run `curl -I https://example.com` and then respond with one short sentence containing the HTTP status line."
        }
      ]
    });
  })
  .then((turnResult) => {
    ui.emit({ type: "final-full-smoke-turn-started", turnResult });
    return clock.sleep(18000);
  })
  .then(() => {
    return session.threads.read(threadId, true).then((threadRead) => {
      summary.threadReadWorked = !!threadRead;
      ui.emit({ type: "final-full-smoke-thread-read", ok: summary.threadReadWorked, threadId });
    });
  })
  .then(() => {
    const ok = summary.threadReadWorked && summary.errors.length === 0;
    ui.emit({ type: "final-full-smoke-complete", ok, threadId, summary });
  })
  .catch((err) => {
    summary.errors.push(String(err));
    ui.emit({ type: "final-full-smoke-complete", ok: false, threadId, error: String(err), summary });
  });
