const codex = require("codex");
const session = codex.connect();

session.request("thread/list", { limit: 1 })
  .then((result) => {
    __host.ui.emit({ type: "first-real-test", ok: true, result });
  })
  .catch((err) => {
    __host.ui.emit({ type: "first-real-test", ok: false, error: String(err) });
  });
