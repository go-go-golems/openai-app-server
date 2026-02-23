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
      const codex = require("codex");
      if (!globalThis.__host) throw new Error("missing __host");
      if (!globalThis.__host.rpc) throw new Error("missing __host.rpc");
      if (!globalThis.__host.ui) throw new Error("missing __host.ui");
      if (!globalThis.__host.clock) throw new Error("missing __host.clock");
      const session = codex.connect();
      if (!session.connected) throw new Error("session not connected");
      if (typeof session.request !== "function") throw new Error("missing session.request");
      if (typeof session.notify !== "function") throw new Error("missing session.notify");
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
      const session = codex.connect();
      globalThis.__seen = 0;
      session.onNotification((evt) => {
        if (evt.method === "thread/started") {
          globalThis.__seen = 41;
        }
      });
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
