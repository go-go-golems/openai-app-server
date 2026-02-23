const codex = require("codex");
const rpc = require("rpc");
const approval = require("approval");
const ui = require("ui");
const clock = require("clock");

const session = codex.connect();
const requestEvents = [];
const responseErrors = [];

let totalNotificationCount = 0;
const interestingNotifications = [];

const cases = [
  {
    name: "on-request_workspace-write",
    approvalPolicy: "on-request",
    sandbox: "workspace-write",
    waitMs: 12000
  },
  {
    name: "on-request_read-only",
    approvalPolicy: "on-request",
    sandbox: "read-only",
    waitMs: 12000
  },
  {
    name: "on-request_danger-full-access",
    approvalPolicy: "on-request",
    sandbox: "danger-full-access",
    waitMs: 12000
  }
];

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

function shouldEmitNotification(method) {
  if (!method) {
    return false;
  }

  if (
    method === "codex/event/reasoning_content_delta" ||
    method === "codex/event/agent_reasoning_delta" ||
    method === "item/reasoning/summaryTextDelta" ||
    method === "item/reasoning/summaryPartAdded"
  ) {
    return false;
  }

  if (
    method.indexOf("thread/") === 0 ||
    method.indexOf("turn/") === 0 ||
    method.indexOf("item/") === 0 ||
    method.indexOf("codex/event/exec_command") === 0 ||
    method === "codex/event/task_started" ||
    method === "codex/event/task_complete" ||
    method === "codex/event/mcp_startup_update" ||
    method === "codex/event/mcp_startup_complete"
  ) {
    return true;
  }

  return false;
}

session.onNotification((evt) => {
  totalNotificationCount += 1;
  const method = evt && evt.method ? evt.method : "";
  if (shouldEmitNotification(method)) {
    interestingNotifications.push(method);
    ui.emit({ type: "sandbox-variant-notification", method, params: (evt && evt.params) || {} });
  }
});

session.onRequest((evt) => {
  const id = evt && evt.id;
  const method = evt && evt.method ? evt.method : "";
  requestEvents.push({ id, method });
  ui.emit({ type: "sandbox-variant-request", id, method, params: (evt && evt.params) || {} });
  try {
    if (method === "item/commandExecution/requestApproval" || method === "item/fileChange/requestApproval") {
      approval.acceptForSession(id);
    } else {
      rpc.respond(id, { accepted: true });
    }
    ui.emit({ type: "sandbox-variant-response", ok: true, id, method });
  } catch (err) {
    responseErrors.push(String(err));
    ui.emit({ type: "sandbox-variant-response", ok: false, id, method, error: String(err) });
  }
});

function runCase(caseCfg) {
  const startRequestCount = requestEvents.length;
  const startTotalNotificationCount = totalNotificationCount;
  const startInterestingCount = interestingNotifications.length;
  let threadId = "";
  let turnId = "";

  ui.emit({
    type: "sandbox-variant-case-start",
    caseName: caseCfg.name,
    approvalPolicy: caseCfg.approvalPolicy,
    sandbox: caseCfg.sandbox
  });

  return rpc
    .request("thread/start", {
      model: "gpt-5",
      cwd: "/home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server",
      approvalPolicy: caseCfg.approvalPolicy,
      sandbox: caseCfg.sandbox
    })
    .then((thread) => {
      threadId = extractThreadId(thread);
      ui.emit({
        type: "sandbox-variant-thread-start",
        caseName: caseCfg.name,
        ok: !!threadId,
        threadId,
        thread
      });
      if (!threadId) {
        throw new Error("thread/start response missing thread id");
      }
      return rpc.request("turn/start", {
        threadId,
        input: [
          {
            type: "text",
            text: "Run `curl -I https://example.com` and return exactly one short sentence with the HTTP status line."
          }
        ]
      });
    })
    .then((turn) => {
      turnId = extractTurnId(turn);
      ui.emit({
        type: "sandbox-variant-turn-start",
        caseName: caseCfg.name,
        ok: !!turnId,
        turnId,
        turn
      });
      return clock.sleep(caseCfg.waitMs || 12000);
    })
    .then(() => {
      const requestDelta = requestEvents.length - startRequestCount;
      const totalNotificationDelta = totalNotificationCount - startTotalNotificationCount;
      const interestingNotificationDelta = interestingNotifications.length - startInterestingCount;
      const status = requestDelta > 0 ? "request_observed" : "no_request_observed";
      const result = {
        caseName: caseCfg.name,
        ok: true,
        status,
        approvalPolicy: caseCfg.approvalPolicy,
        sandbox: caseCfg.sandbox,
        threadId,
        turnId,
        requestDelta,
        totalNotificationDelta,
        interestingNotificationDelta
      };
      ui.emit({ type: "sandbox-variant-case-complete", result });
      return result;
    })
    .catch((err) => {
      const requestDelta = requestEvents.length - startRequestCount;
      const totalNotificationDelta = totalNotificationCount - startTotalNotificationCount;
      const interestingNotificationDelta = interestingNotifications.length - startInterestingCount;
      const result = {
        caseName: caseCfg.name,
        ok: false,
        status: "case_error",
        error: String(err),
        approvalPolicy: caseCfg.approvalPolicy,
        sandbox: caseCfg.sandbox,
        threadId,
        turnId,
        requestDelta,
        totalNotificationDelta,
        interestingNotificationDelta
      };
      ui.emit({ type: "sandbox-variant-case-complete", result });
      return result;
    });
}

const results = [];
let chain = Promise.resolve();

for (let i = 0; i < cases.length; i += 1) {
  const c = cases[i];
  chain = chain.then(() => runCase(c)).then((result) => {
    results.push(result);
    return clock.sleep(1000);
  });
}

chain
  .then(() => {
    let allCasesOk = true;
    for (let i = 0; i < results.length; i += 1) {
      if (!results[i].ok) {
        allCasesOk = false;
        break;
      }
    }

    ui.emit({
      type: "sandbox-variant-probe-complete",
      ok: allCasesOk && responseErrors.length === 0,
      sawAnyRequest: requestEvents.length > 0,
      totalRequestsObserved: requestEvents.length,
      totalNotificationCount,
      interestingNotificationCount: interestingNotifications.length,
      responseErrors: responseErrors.slice(0, 10),
      results
    });
  })
  .catch((err) => {
    ui.emit({
      type: "sandbox-variant-probe-complete",
      ok: false,
      error: String(err),
      sawAnyRequest: requestEvents.length > 0,
      totalRequestsObserved: requestEvents.length,
      totalNotificationCount,
      interestingNotificationCount: interestingNotifications.length,
      responseErrors: responseErrors.slice(0, 10),
      results
    });
  });
