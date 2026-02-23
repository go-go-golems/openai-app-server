package js

import (
	"context"
	"testing"
	"time"
)

type fakeRPCBridge struct{}

func (f fakeRPCBridge) Request(_ context.Context, method string, _ any) (any, error) {
	return map[string]any{"ok": true, "method": method}, nil
}

func (f fakeRPCBridge) Notify(_ context.Context, _ string, _ any) error {
	return nil
}

func (f fakeRPCBridge) Respond(_ context.Context, _ any, _ any) error {
	return nil
}

func (f fakeRPCBridge) RespondError(_ context.Context, _ any, _ int, _ string, _ any) error {
	return nil
}

type fakeUIBridge struct{}

func (f fakeUIBridge) Emit(_ context.Context, _ any) error {
	return nil
}

func TestRuntimeInstallsHostAndCodexModule(t *testing.T) {
	rt, err := NewRuntime(Options{
		Name: "runtime-test",
		RPC:  fakeRPCBridge{},
		UI:   fakeUIBridge{},
	})
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
	defer func() { _ = rt.Close() }()

	_, err = rt.RunString(`
      const rpc = require("rpc");
      const ui = require("ui");
      const clock = require("clock");
      const codex = require("codex");
      if (typeof globalThis.__host !== "undefined") throw new Error("unexpected __host global");
      if (typeof rpc.request !== "function") throw new Error("missing rpc.request");
      if (typeof rpc.notify !== "function") throw new Error("missing rpc.notify");
      if (typeof rpc.respond !== "function") throw new Error("missing rpc.respond");
      if (typeof rpc.respondError !== "function") throw new Error("missing rpc.respondError");
      if (typeof ui.emit !== "function") throw new Error("missing ui.emit");
      if (typeof clock.nowMs !== "function") throw new Error("missing clock.nowMs");
      if (typeof clock.sleep !== "function") throw new Error("missing clock.sleep");
      const session = codex.connect();
      if (!session.connected) throw new Error("session not connected");
      if (typeof session.request !== "function") throw new Error("missing session.request");
      if (typeof session.notify !== "function") throw new Error("missing session.notify");
      if (typeof session.respond !== "function") throw new Error("missing session.respond");
      if (typeof session.respondError !== "function") throw new Error("missing session.respondError");
      if (typeof session.onNotification !== "function") throw new Error("missing session.onNotification");
      if (codex.version !== "0.1.0") throw new Error("unexpected codex version");
    `)
	if err != nil {
		t.Fatalf("RunString() error = %v", err)
	}
}

func TestRuntimeNotificationCallbackDispatch(t *testing.T) {
	rt, err := NewRuntime(Options{
		Name: "runtime-test-notify",
		RPC:  fakeRPCBridge{},
		UI:   fakeUIBridge{},
	})
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
	defer func() { _ = rt.Close() }()

	_, err = rt.RunString(`
      const codex = require("codex");
      const ui = require("ui");
      const session = codex.connect();
      globalThis.__seen = 0;
      session.onNotification((evt) => {
        if (evt.method === "thread/started") {
          globalThis.__seen = 41;
        }
      });
      ui.emit({ type: "runtime-test" });
    `)
	if err != nil {
		t.Fatalf("RunString() setup error = %v", err)
	}

	if err := rt.EmitRPCNotification("thread/started", map[string]any{"id": "thread-1"}); err != nil {
		t.Fatalf("EmitRPCNotification() error = %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		v, runErr := rt.RunString("globalThis.__seen")
		if runErr != nil {
			t.Fatalf("RunString() readback error = %v", runErr)
		}
		if v != nil && v.ToInteger() == 41 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("notification callback did not update state before deadline")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
