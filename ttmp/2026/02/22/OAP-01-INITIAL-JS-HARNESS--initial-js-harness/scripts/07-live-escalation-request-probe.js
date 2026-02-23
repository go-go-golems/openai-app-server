const codex = require("codex");
const rpc = require("rpc");
const ui = require("ui");
const clock = require("clock");

const session = codex.connect();
let threadId = "";
let turnId = "";
let sawRequest = false;
let respondedToRequest = false;
const requestMethods = [];
const notifications = [];
const responseErrors = [];

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
  ui.emit({ type: "escalation-probe-notification", method, params: (evt && evt.params) || {} });
});

session.onRequest((evt) => {
  const id = evt && evt.id;
  const method = evt && evt.method ? evt.method : "";
  sawRequest = true;
  if (method) {
    requestMethods.push(method);
  }
  ui.emit({ type: "escalation-probe-request", id, method, params: (evt && evt.params) || {} });
  try {
    rpc.respond(id, { approved: true, decision: "approve", note: "auto-approved by escalation probe" });
    respondedToRequest = true;
    ui.emit({ type: "escalation-probe-response", ok: true, id, method });
  } catch (err) {
    responseErrors.push(String(err));
    ui.emit({ type: "escalation-probe-response", ok: false, id, method, error: String(err) });
  }
});

rpc.request("thread/start", {
  model: "gpt-5",
  cwd: "/home/manuel/workspaces/2026-02-22/app-server-js/openai-app-server",
  approvalPolicy: "on-request",
  sandbox: "workspace-write"
})
  .then((thread) => {
    threadId = extractThreadId(thread);
    ui.emit({ type: "escalation-probe-thread-start", ok: !!threadId, threadId, thread });
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
    ui.emit({ type: "escalation-probe-turn-start", ok: !!turnId, turnId, turn });
    return clock.sleep(12000);
  })
  .then(() => {
    const ok = responseErrors.length === 0;
    const probeStatus = sawRequest
      ? (respondedToRequest ? "request_observed_and_responded" : "request_observed_but_not_responded")
      : "no_inbound_request_observed";
    ui.emit({
      type: "escalation-probe-complete",
      ok,
      probeStatus,
      threadId,
      turnId,
      sawRequest,
      respondedToRequest,
      requestMethods: requestMethods.slice(0, 20),
      responseErrors: responseErrors.slice(0, 10),
      notifications: notifications.slice(0, 50)
    });
  })
  .catch((err) => {
    ui.emit({
      type: "escalation-probe-complete",
      ok: false,
      probeStatus: "script_error",
      error: String(err),
      threadId,
      turnId,
      sawRequest,
      respondedToRequest,
      requestMethods: requestMethods.slice(0, 20),
      responseErrors: responseErrors.slice(0, 10),
      notifications: notifications.slice(0, 50)
    });
  });
