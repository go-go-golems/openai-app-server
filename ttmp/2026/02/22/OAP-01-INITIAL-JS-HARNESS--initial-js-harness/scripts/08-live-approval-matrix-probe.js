const codex = require("codex");
const rpc = require("rpc");
const approval = require("approval");
const ui = require("ui");
const clock = require("clock");

const session = codex.connect();
const notifications = [];
const requestEvents = [];
const responseErrors = [];

const cases = [
  {
    name: "on-request_workspace-write",
    approvalPolicy: "on-request",
    sandbox: "workspace-write",
    waitMs: 12000
  },
  {
    name: "on-failure_workspace-write",
    approvalPolicy: "on-failure",
    sandbox: "workspace-write",
    waitMs: 12000
  },
  {
    name: "never_workspace-write",
    approvalPolicy: "never",
    sandbox: "workspace-write",
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

session.onNotification((evt) => {
  const method = evt && evt.method ? evt.method : "";
  if (method) {
    notifications.push(method);
  }
  ui.emit({ type: "approval-matrix-notification", method, params: (evt && evt.params) || {} });
});

session.onRequest((evt) => {
  const id = evt && evt.id;
  const method = evt && evt.method ? evt.method : "";
  requestEvents.push({ id, method });
  ui.emit({ type: "approval-matrix-request", id, method, params: (evt && evt.params) || {} });
  try {
    if (method === "item/commandExecution/requestApproval" || method === "item/fileChange/requestApproval") {
      approval.acceptForSession(id);
    } else {
      rpc.respond(id, { accepted: true });
    }
    ui.emit({ type: "approval-matrix-response", ok: true, id, method });
  } catch (err) {
    responseErrors.push(String(err));
    ui.emit({ type: "approval-matrix-response", ok: false, id, method, error: String(err) });
  }
});

function runCase(caseCfg) {
  const startRequestCount = requestEvents.length;
  const startNotificationCount = notifications.length;
  let threadId = "";
  let turnId = "";

  ui.emit({
    type: "approval-matrix-case-start",
    caseName: caseCfg.name,
    approvalPolicy: caseCfg.approvalPolicy,
    sandbox: caseCfg.sandbox
  });

  return rpc.request("thread/start", {
    model: "gpt-5",
    cwd: "/home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server",
    approvalPolicy: caseCfg.approvalPolicy,
    sandbox: caseCfg.sandbox
  })
    .then((thread) => {
      threadId = extractThreadId(thread);
      ui.emit({
        type: "approval-matrix-thread-start",
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
        type: "approval-matrix-turn-start",
        caseName: caseCfg.name,
        ok: !!turnId,
        turnId,
        turn
      });
      return clock.sleep(caseCfg.waitMs || 12000);
    })
    .then(() => {
      const requestDelta = requestEvents.length - startRequestCount;
      const notificationDelta = notifications.length - startNotificationCount;
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
        notificationDelta
      };
      ui.emit({ type: "approval-matrix-case-complete", result });
      return result;
    })
    .catch((err) => {
      const requestDelta = requestEvents.length - startRequestCount;
      const notificationDelta = notifications.length - startNotificationCount;
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
        notificationDelta
      };
      ui.emit({ type: "approval-matrix-case-complete", result });
      return result;
    });
}

const results = [];
let chain = Promise.resolve();

for (let i = 0; i < cases.length; i += 1) {
  const c = cases[i];
  chain = chain
    .then(() => runCase(c))
    .then((result) => {
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
    const sawAnyRequest = requestEvents.length > 0;
    ui.emit({
      type: "approval-matrix-probe-complete",
      ok: allCasesOk && responseErrors.length === 0,
      sawAnyRequest,
      totalRequestsObserved: requestEvents.length,
      responseErrors: responseErrors.slice(0, 10),
      results
    });
  })
  .catch((err) => {
    ui.emit({
      type: "approval-matrix-probe-complete",
      ok: false,
      error: String(err),
      sawAnyRequest: requestEvents.length > 0,
      totalRequestsObserved: requestEvents.length,
      responseErrors: responseErrors.slice(0, 10),
      results
    });
  });
