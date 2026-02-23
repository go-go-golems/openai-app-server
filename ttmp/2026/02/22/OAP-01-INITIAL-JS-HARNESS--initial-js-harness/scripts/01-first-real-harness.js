const codex = require("codex");
const ui = require("ui");
const session = codex.connect();

session.request("thread/list", { limit: 1 })
  .then((result) => {
    ui.emit({ type: "first-real-test", ok: true, result });
  })
  .catch((err) => {
    ui.emit({ type: "first-real-test", ok: false, error: String(err) });
  });
