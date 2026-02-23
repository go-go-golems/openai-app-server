const codex = require("codex");
const rpc = require("rpc");
const ui = require("ui");
const clock = require("clock");

const session = codex.connect();
session.onNotification((evt) => {
  ui.emit({ type: "module-api-notification", method: evt.method, params: evt.params || {} });
});

rpc.request("thread/list", { limit: 3 })
  .then((result) => {
    const threads = result && Array.isArray(result.threads) ? result.threads : [];
    ui.emit({ type: "module-api-thread-list", ok: true, count: threads.length, result });
    return clock.sleep(200);
  })
  .then(() => {
    ui.emit({ type: "module-api-smoke-complete", ok: true });
  })
  .catch((err) => {
    ui.emit({ type: "module-api-smoke-complete", ok: false, error: String(err) });
  });
