const codex = require("codex");
const ui = require("ui");
const clock = require("clock");

const session = codex.connect();
let handledTurnId = "";
let reviewResult = null;
let followupStarted = false;

session.onNotification((evt) => {
  const method = evt && evt.method ? evt.method : "";
  const params = (evt && evt.params) || {};

  if (method !== "turn/completed") {
    return;
  }

  const threadId = params.threadId || "";
  const turn = params.turn || {};
  const turnId = turn.id || params.turnId || "";
  const status = turn.status || "";

  if (!threadId || !turnId || status !== "completed") {
    return;
  }

  if (handledTurnId) {
    return;
  }
  handledTurnId = turnId;

  ui.emit({ type: "phase6-review-gate-turn-completed", threadId, turnId });

  const thread = session.thread(threadId);
  thread.review
    .start({ delivery: "inline", target: { type: "uncommittedChanges" } })
    .then((review) => {
      reviewResult = review;
      ui.emit({ type: "phase6-review-gate-review-started", threadId, turnId, review });
      return thread.turn.start({
        input: [
          {
            type: "text",
            text: "Address any important review findings in one short patch and summarize in one sentence."
          }
        ]
      });
    })
    .then((followup) => {
      followupStarted = true;
      ui.emit({ type: "phase6-review-gate-followup-started", threadId, followup });
    })
    .catch((err) => {
      ui.emit({ type: "phase6-review-gate-review-error", threadId, error: String(err) });
    });
});

ui.emit({ type: "phase6-review-gate-smoke-start" });

session.threads
  .start({
    model: "gpt-5",
    cwd: "/home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server",
    approvalPolicy: "on-request",
    sandbox: "workspace-write"
  })
  .then((startResult) => {
    const threadData = (startResult && startResult.thread) || startResult || {};
    const threadId = threadData.id || threadData.threadId || "";
    if (!threadId) {
      throw new Error("thread/start result missing id");
    }

    ui.emit({ type: "phase6-review-gate-thread-started", threadId });

    const thread = session.thread(threadId);
    return thread.turn.start({
      input: [
        {
          type: "text",
          text: "Make one tiny, safe code quality improvement and summarize what changed in one short sentence."
        }
      ]
    });
  })
  .then((turnResult) => {
    ui.emit({ type: "phase6-review-gate-turn-started", turnResult });
    return clock.sleep(22000);
  })
  .then(() => {
    ui.emit({
      type: "phase6-review-gate-smoke-complete",
      ok: true,
      handledTurnId,
      followupStarted,
      reviewResult
    });
  })
  .catch((err) => {
    ui.emit({
      type: "phase6-review-gate-smoke-complete",
      ok: false,
      error: String(err),
      handledTurnId,
      followupStarted,
      reviewResult
    });
  });
