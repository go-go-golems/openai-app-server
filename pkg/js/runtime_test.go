package js

import (
	"context"
	"reflect"
	"sync"
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

type capturedResponse struct {
	id     any
	result any
}

type recordingRPCBridge struct {
	mu        sync.Mutex
	responses []capturedResponse
}

func (r *recordingRPCBridge) Request(_ context.Context, method string, _ any) (any, error) {
	return map[string]any{"ok": true, "method": method}, nil
}

func (r *recordingRPCBridge) Notify(_ context.Context, _ string, _ any) error {
	return nil
}

func (r *recordingRPCBridge) Respond(_ context.Context, id any, result any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.responses = append(r.responses, capturedResponse{id: id, result: result})
	return nil
}

func (r *recordingRPCBridge) RespondError(_ context.Context, _ any, _ int, _ string, _ any) error {
	return nil
}

func (r *recordingRPCBridge) snapshot() []capturedResponse {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]capturedResponse, len(r.responses))
	copy(out, r.responses)
	return out
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
      const approval = require("approval");
      const codex = require("codex");
      if (typeof globalThis.__host !== "undefined") throw new Error("unexpected __host global");
      if (typeof rpc.request !== "function") throw new Error("missing rpc.request");
      if (typeof rpc.notify !== "function") throw new Error("missing rpc.notify");
      if (typeof rpc.respond !== "function") throw new Error("missing rpc.respond");
      if (typeof rpc.respondError !== "function") throw new Error("missing rpc.respondError");
      if (typeof approval.accept !== "function") throw new Error("missing approval.accept");
      if (typeof approval.acceptForSession !== "function") throw new Error("missing approval.acceptForSession");
      if (typeof approval.decline !== "function") throw new Error("missing approval.decline");
      if (typeof approval.cancel !== "function") throw new Error("missing approval.cancel");
      if (typeof approval.acceptWithExecpolicyAmendment !== "function") throw new Error("missing approval.acceptWithExecpolicyAmendment");
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

func TestApprovalModuleDecisionResponses(t *testing.T) {
	rpcBridge := &recordingRPCBridge{}
	rt, err := NewRuntime(Options{
		Name: "runtime-test-approval",
		RPC:  rpcBridge,
		UI:   fakeUIBridge{},
	})
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
	defer func() { _ = rt.Close() }()

	_, err = rt.RunString(`
      const approval = require("approval");
      approval.accept("id-1");
      approval.acceptForSession({ id: "id-2" });
      approval.decline("id-3");
      approval.cancel("id-4");
      approval.acceptWithExecpolicyAmendment("id-5", ["curl", "-I"]);
    `)
	if err != nil {
		t.Fatalf("RunString() error = %v", err)
	}

	got := rpcBridge.snapshot()
	if len(got) != 5 {
		t.Fatalf("expected 5 responses, got %d", len(got))
	}

	check := func(i int, wantID any, wantResult any) {
		if got[i].id != wantID {
			t.Fatalf("response[%d] id mismatch: got=%v want=%v", i, got[i].id, wantID)
		}
		if !reflect.DeepEqual(got[i].result, wantResult) {
			t.Fatalf("response[%d] result mismatch: got=%#v want=%#v", i, got[i].result, wantResult)
		}
	}

	check(0, "id-1", map[string]any{"decision": "accept"})
	check(1, "id-2", map[string]any{"decision": "acceptForSession"})
	check(2, "id-3", map[string]any{"decision": "decline"})
	check(3, "id-4", map[string]any{"decision": "cancel"})
	check(4, "id-5", map[string]any{
		"decision": map[string]any{
			"acceptWithExecpolicyAmendment": map[string]any{
				"execpolicy_amendment": []any{"curl", "-I"},
			},
		},
	})
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
