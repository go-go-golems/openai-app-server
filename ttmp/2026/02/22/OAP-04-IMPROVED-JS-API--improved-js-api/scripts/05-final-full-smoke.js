const codex = require("codex");
const ui = require("ui");
const clock = require("clock");

const session = codex.connect();
const startMs = clock.nowMs();

const summary = {
  threadId: "",
  turnId: "",
  approvalsAccepted: 0,
  approvalsDeclined: 0,
  approvalWaitMatched: false,
  approvalWaitError: "",
  turnCompleted: false,
  turnCompletedError: "",
  threadReadWorked: false,
  requestsObserved: 0,
  notificationsObserved: 0,
  totalRequests: 0,
  totalNotifications: 0,
  methodCounts: {},
  branchMissReasons: [],
  errors: []
};

const checkpoints = [];
const eventSamples = {
  requests: [],
  notifications: []
};

function asString(v) {
  return typeof v === "string" ? v : "";
}

function elapsedMs() {
  return clock.nowMs() - startMs;
}

function checkpoint(stage, extra) {
  const payload = Object.assign({
    stage,
    elapsedMs: elapsedMs()
  }, extra || {});
  checkpoints.push(payload);
  ui.emit(Object.assign({ type: "final-full-smoke-v2-checkpoint" }, payload));
}

function pushSample(kind, evt) {
  const maxSamples = 8;
  const list = kind === "request" ? eventSamples.requests : eventSamples.notifications;
  if (list.length >= maxSamples) {
    return;
  }
  const method = asString(evt && evt.method);
  const params = (evt && evt.params) || {};
  const sample = {
    method,
    keys: Object.keys(params)
  };
  if (kind === "request") {
    sample.id = evt && evt.id;
  }
  list.push(sample);
}

session.onRequest((evt) => {
  summary.requestsObserved += 1;
  pushSample("request", evt);
});

session.onNotification((evt) => {
  summary.notificationsObserved += 1;
  pushSample("notification", evt);

  if (asString(evt && evt.method) === "turn/completed") {
    summary.turnCompleted = true;
    ui.emit({
      type: "final-full-smoke-v2-turn-completed-observed",
      elapsedMs: elapsedMs(),
      params: (evt && evt.params) || {}
    });
  }
});

session.approvals.setPolicy({
  command: (req) => {
    const command = asString(req && req.commandText);
    if (command.indexOf("curl") === 0 || command.indexOf("curl ") >= 0) {
      summary.approvalsAccepted += 1;
      ui.emit({
        type: "final-full-smoke-v2-approval",
        method: asString(req && req.method),
        decision: "acceptForSession",
        command,
        elapsedMs: elapsedMs()
      });
      return "acceptForSession";
    }

    summary.approvalsDeclined += 1;
    ui.emit({
      type: "final-full-smoke-v2-approval",
      method: asString(req && req.method),
      decision: "decline",
      command,
      elapsedMs: elapsedMs()
    });
    return "decline";
  },
  fileChange: (req) => {
    summary.approvalsDeclined += 1;
    ui.emit({
      type: "final-full-smoke-v2-approval",
      method: asString(req && req.method),
      decision: "decline",
      elapsedMs: elapsedMs()
    });
    return "decline";
  },
  fallback: () => {
    summary.approvalsDeclined += 1;
    return "decline";
  }
});

ui.emit({ type: "final-full-smoke-v2-start" });
checkpoint("start");

const approvalWaitPromise = session
  .waitFor({
    method: "item/commandExecution/requestApproval",
    includeRequests: true,
    includeNotifications: false,
    timeoutMs: 120000
  })
  .then((evt) => {
    summary.approvalWaitMatched = true;
    ui.emit({
      type: "final-full-smoke-v2-approval-wait-matched",
      elapsedMs: elapsedMs(),
      method: asString(evt && evt.method)
    });
    return evt;
  })
  .catch((err) => {
    const msg = String(err);
    summary.approvalWaitError = msg;
    ui.emit({
      type: "final-full-smoke-v2-approval-wait-timeout",
      elapsedMs: elapsedMs(),
      error: msg
    });
    return null;
  });

session.threads
  .start({
    model: "gpt-5",
    cwd: "/home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server",
    approvalPolicy: "on-request",
    sandbox: "read-only"
  })
  .then((startResult) => {
    summary.threadId = session.ids.thread(startResult);
    if (!summary.threadId) {
      throw new Error("thread/start result missing thread id");
    }

    checkpoint("thread-started", { threadId: summary.threadId });
    ui.emit({ type: "final-full-smoke-v2-thread-started", threadId: summary.threadId });

    const thread = session.thread(summary.threadId);
    return thread.turn.start({
      input: [
        {
          type: "text",
          text: "You must run exactly this shell command now: `curl -I https://example.com`. After running it, reply with one sentence containing the HTTP status line from the command output."
        }
      ]
    });
  })
  .then((turnResult) => {
    summary.turnId = session.ids.turn(turnResult);
    checkpoint("turn-started", { turnId: summary.turnId });
    ui.emit({ type: "final-full-smoke-v2-turn-started", turnId: summary.turnId });

    const thread = session.thread(summary.threadId);
    return thread.turn
      .waitCompleted({ turnId: summary.turnId, timeoutMs: 180000 })
      .then((evt) => {
        summary.turnCompleted = true;
        checkpoint("turn-wait-completed", { turnId: summary.turnId });
        ui.emit({
          type: "final-full-smoke-v2-turn-wait-completed",
          elapsedMs: elapsedMs(),
          params: (evt && evt.params) || {}
        });
      })
      .catch((err) => {
        summary.turnCompletedError = String(err);
        ui.emit({
          type: "final-full-smoke-v2-turn-wait-timeout",
          elapsedMs: elapsedMs(),
          error: summary.turnCompletedError
        });
      });
  })
  .then(() => approvalWaitPromise)
  .then(() => {
    return session.threads.read({ threadId: summary.threadId, includeTurns: true }).then((threadRead) => {
      summary.threadReadWorked = !!threadRead;
      checkpoint("thread-read", { ok: summary.threadReadWorked });
      ui.emit({ type: "final-full-smoke-v2-thread-read", ok: summary.threadReadWorked, threadId: summary.threadId });
    });
  })
  .then(() => {
    const metrics = session.events.metrics();
    summary.methodCounts = metrics.countByMethod();
    summary.totalRequests = metrics.totalRequests();
    summary.totalNotifications = metrics.totalNotifications();

    if (!summary.turnCompleted) {
      summary.branchMissReasons.push("turn/completed branch not observed");
    }
    if (!summary.approvalWaitMatched) {
      summary.branchMissReasons.push("approval request branch not observed");
    }
    if (summary.approvalsAccepted + summary.approvalsDeclined === 0) {
      summary.branchMissReasons.push("approval policy callback was never invoked");
    }
    if (!summary.threadReadWorked) {
      summary.branchMissReasons.push("thread/read returned empty result");
    }

    const ok = summary.errors.length === 0 && summary.branchMissReasons.length === 0;
    ui.emit({
      type: "final-full-smoke-v2-complete",
      ok,
      elapsedMs: elapsedMs(),
      checkpoints,
      eventSamples,
      summary
    });
  })
  .catch((err) => {
    summary.errors.push(String(err));
    ui.emit({
      type: "final-full-smoke-v2-complete",
      ok: false,
      elapsedMs: elapsedMs(),
      checkpoints,
      eventSamples,
      error: String(err),
      summary
    });
  });
