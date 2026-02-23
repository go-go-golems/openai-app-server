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

type capturedRequest struct {
	method string
	params any
}

type recordingRPCBridge struct {
	mu        sync.Mutex
	requests  []capturedRequest
	responses []capturedResponse
}

func (r *recordingRPCBridge) Request(_ context.Context, method string, params any) (any, error) {
	r.mu.Lock()
	r.requests = append(r.requests, capturedRequest{method: method, params: params})
	r.mu.Unlock()
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

func (r *recordingRPCBridge) snapshotRequests() []capturedRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]capturedRequest, len(r.requests))
	copy(out, r.requests)
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
      if (typeof session.waitFor !== "function") throw new Error("missing session.waitFor");
      if (typeof session.threads !== "object") throw new Error("missing session.threads");
      if (typeof session.threads.start !== "function") throw new Error("missing session.threads.start");
      if (typeof session.threads.list !== "function") throw new Error("missing session.threads.list");
      if (typeof session.threads.read !== "function") throw new Error("missing session.threads.read");
      if (typeof session.threads.byId !== "function") throw new Error("missing session.threads.byId");
      if (typeof session.thread !== "function") throw new Error("missing session.thread");
      if (typeof session.ids !== "object") throw new Error("missing session.ids");
      if (typeof session.ids.thread !== "function") throw new Error("missing session.ids.thread");
      if (typeof session.ids.turn !== "function") throw new Error("missing session.ids.turn");
      if (typeof session.events !== "object") throw new Error("missing session.events");
      if (typeof session.events.metrics !== "function") throw new Error("missing session.events.metrics");
      if (typeof session.approvals !== "object") throw new Error("missing session.approvals");
      if (typeof session.approvals.setPolicy !== "function") throw new Error("missing session.approvals.setPolicy");
      if (typeof session.approvals.respond !== "function") throw new Error("missing session.approvals.respond");
      const metrics = session.events.metrics();
      if (typeof metrics.countByMethod !== "function") throw new Error("missing metrics.countByMethod");
      if (typeof metrics.totalNotifications !== "function") throw new Error("missing metrics.totalNotifications");
      if (typeof metrics.totalRequests !== "function") throw new Error("missing metrics.totalRequests");
      if (typeof metrics.reset !== "function") throw new Error("missing metrics.reset");
      const thread = session.thread("thread-check");
      if (typeof thread.turn.waitCompleted !== "function") throw new Error("missing thread.turn.waitCompleted");
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

func TestCodexSessionThreadWrappersRouteRequests(t *testing.T) {
	rpcBridge := &recordingRPCBridge{}
	rt, err := NewRuntime(Options{
		Name: "runtime-test-thread-wrappers",
		RPC:  rpcBridge,
		UI:   fakeUIBridge{},
	})
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
	defer func() { _ = rt.Close() }()

	_, err = rt.RunString(`
      const codex = require("codex");
      const session = codex.connect();

      session.threads.start({ model: "gpt-5", cwd: "/repo" });
      session.threads.list({ limit: 5 });
      session.threads.read("thread-abc", true);

      const thread = session.thread("thread-abc");
      thread.turn.start({ input: [{ type: "text", text: "hello" }] });
      thread.turn.steer({ expectedTurnId: "turn-1", input: [{ type: "text", text: "continue" }] });
      thread.turn.interrupt({ turnId: "turn-1" });
      thread.review.start({ delivery: "inline" });

      const nested = session.threads.byId({ thread: { id: "thread-nested" } });
      nested.turn.start({ input: [{ type: "text", text: "nested" }] });
    `)
	if err != nil {
		t.Fatalf("RunString() error = %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	var reqs []capturedRequest
	for {
		reqs = rpcBridge.snapshotRequests()
		if len(reqs) >= 8 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for wrapper requests, got %d", len(reqs))
		}
		time.Sleep(10 * time.Millisecond)
	}

	counts := map[string]int{}
	for _, r := range reqs {
		counts[r.method]++
	}

	expectedCounts := map[string]int{
		"thread/start":   1,
		"thread/list":    1,
		"thread/read":    1,
		"turn/start":     2,
		"turn/steer":     1,
		"turn/interrupt": 1,
		"review/start":   1,
	}
	for method, want := range expectedCounts {
		if counts[method] < want {
			t.Fatalf("expected at least %d request(s) for %s, got %d", want, method, counts[method])
		}
	}

	var sawReadThreadID bool
	var sawReadIncludeTurns bool
	var sawTurnStartABC bool
	var sawTurnStartNested bool
	var sawSteerABC bool
	var sawInterruptABC bool
	var sawReviewABC bool

	for _, r := range reqs {
		params, _ := r.params.(map[string]any)
		switch r.method {
		case "thread/read":
			if params["threadId"] == "thread-abc" {
				sawReadThreadID = true
			}
			if includeTurns, ok := params["includeTurns"].(bool); ok && includeTurns {
				sawReadIncludeTurns = true
			}
		case "turn/start":
			if params["threadId"] == "thread-abc" {
				sawTurnStartABC = true
			}
			if params["threadId"] == "thread-nested" {
				sawTurnStartNested = true
			}
		case "turn/steer":
			if params["threadId"] == "thread-abc" {
				sawSteerABC = true
			}
		case "turn/interrupt":
			if params["threadId"] == "thread-abc" {
				sawInterruptABC = true
			}
		case "review/start":
			if params["threadId"] == "thread-abc" {
				sawReviewABC = true
			}
		}
	}

	if !sawReadThreadID || !sawReadIncludeTurns {
		t.Fatalf("thread/read wrapper did not normalize params correctly: %#v", reqs)
	}
	if !sawTurnStartABC || !sawTurnStartNested {
		t.Fatalf("turn/start wrapper did not inject thread ids correctly: %#v", reqs)
	}
	if !sawSteerABC || !sawInterruptABC || !sawReviewABC {
		t.Fatalf("thread handle wrappers missing threadId injection: %#v", reqs)
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

func TestCodexIdentityHelpersExtractIDs(t *testing.T) {
	rt, err := NewRuntime(Options{
		Name: "runtime-test-ids",
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
      globalThis.__ids = {
        threadDirect: session.ids.thread("thread-a"),
        threadNested: session.ids.thread({ thread: { id: "thread-b" } }),
        turnDirect: session.ids.turn("turn-a"),
        turnNested: session.ids.turn({ turn: { turnId: "turn-b" } })
      };
    `)
	if err != nil {
		t.Fatalf("RunString() error = %v", err)
	}

	threadDirect, err := rt.RunString(`globalThis.__ids.threadDirect`)
	if err != nil {
		t.Fatalf("RunString() threadDirect error = %v", err)
	}
	if threadDirect.String() != "thread-a" {
		t.Fatalf("threadDirect mismatch: got=%q want=%q", threadDirect.String(), "thread-a")
	}

	threadNested, err := rt.RunString(`globalThis.__ids.threadNested`)
	if err != nil {
		t.Fatalf("RunString() threadNested error = %v", err)
	}
	if threadNested.String() != "thread-b" {
		t.Fatalf("threadNested mismatch: got=%q want=%q", threadNested.String(), "thread-b")
	}

	turnDirect, err := rt.RunString(`globalThis.__ids.turnDirect`)
	if err != nil {
		t.Fatalf("RunString() turnDirect error = %v", err)
	}
	if turnDirect.String() != "turn-a" {
		t.Fatalf("turnDirect mismatch: got=%q want=%q", turnDirect.String(), "turn-a")
	}

	turnNested, err := rt.RunString(`globalThis.__ids.turnNested`)
	if err != nil {
		t.Fatalf("RunString() turnNested error = %v", err)
	}
	if turnNested.String() != "turn-b" {
		t.Fatalf("turnNested mismatch: got=%q want=%q", turnNested.String(), "turn-b")
	}
}

func TestCodexWaitForMatchesNotification(t *testing.T) {
	rt, err := NewRuntime(Options{
		Name: "runtime-test-waitfor",
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
      globalThis.__wait = { done: false, method: "", error: "" };
      session.waitFor({ method: "turn/completed", timeoutMs: 2000 }).then(
        (evt) => {
          globalThis.__wait.done = true;
          globalThis.__wait.method = evt.method || "";
        },
        (err) => {
          globalThis.__wait.done = true;
          globalThis.__wait.error = String(err);
        }
      );
    `)
	if err != nil {
		t.Fatalf("RunString() setup error = %v", err)
	}

	if emitErr := rt.EmitRPCNotification("turn/completed", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-1",
	}); emitErr != nil {
		t.Fatalf("EmitRPCNotification() error = %v", emitErr)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		doneValue, runErr := rt.RunString(`globalThis.__wait.done`)
		if runErr != nil {
			t.Fatalf("RunString() done check error = %v", runErr)
		}
		if doneValue.ToBoolean() {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("waitFor did not resolve before deadline")
		}
		time.Sleep(10 * time.Millisecond)
	}

	methodValue, err := rt.RunString(`globalThis.__wait.method`)
	if err != nil {
		t.Fatalf("RunString() method error = %v", err)
	}
	if methodValue.String() != "turn/completed" {
		t.Fatalf("waitFor method mismatch: got=%q want=%q", methodValue.String(), "turn/completed")
	}

	errorValue, err := rt.RunString(`globalThis.__wait.error`)
	if err != nil {
		t.Fatalf("RunString() error field read failed: %v", err)
	}
	if errorValue.String() != "" {
		t.Fatalf("waitFor should not fail, got error=%q", errorValue.String())
	}
}

func TestCodexTurnWaitCompletedFiltersByTurnID(t *testing.T) {
	rt, err := NewRuntime(Options{
		Name: "runtime-test-waitcompleted",
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
      const thread = session.thread("thread-abc");
      globalThis.__waitTurn = { done: false, matchedTurnId: "", error: "" };
      thread.turn.waitCompleted({ turnId: "turn-2", timeoutMs: 2000 }).then(
        (evt) => {
          globalThis.__waitTurn.done = true;
          const params = evt.params || {};
          globalThis.__waitTurn.matchedTurnId = params.turnId || "";
        },
        (err) => {
          globalThis.__waitTurn.done = true;
          globalThis.__waitTurn.error = String(err);
        }
      );
    `)
	if err != nil {
		t.Fatalf("RunString() setup error = %v", err)
	}

	if emitErr := rt.EmitRPCNotification("turn/completed", map[string]any{
		"threadId": "thread-abc",
		"turnId":   "turn-1",
	}); emitErr != nil {
		t.Fatalf("EmitRPCNotification() first event error = %v", emitErr)
	}
	if emitErr := rt.EmitRPCNotification("turn/completed", map[string]any{
		"threadId": "thread-abc",
		"turnId":   "turn-2",
	}); emitErr != nil {
		t.Fatalf("EmitRPCNotification() second event error = %v", emitErr)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		doneValue, runErr := rt.RunString(`globalThis.__waitTurn.done`)
		if runErr != nil {
			t.Fatalf("RunString() done check error = %v", runErr)
		}
		if doneValue.ToBoolean() {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("waitCompleted did not resolve before deadline")
		}
		time.Sleep(10 * time.Millisecond)
	}

	matchedTurnID, err := rt.RunString(`globalThis.__waitTurn.matchedTurnId`)
	if err != nil {
		t.Fatalf("RunString() matchedTurnId error = %v", err)
	}
	if matchedTurnID.String() != "turn-2" {
		t.Fatalf("waitCompleted matchedTurnId mismatch: got=%q want=%q", matchedTurnID.String(), "turn-2")
	}

	errorValue, err := rt.RunString(`globalThis.__waitTurn.error`)
	if err != nil {
		t.Fatalf("RunString() error field read failed: %v", err)
	}
	if errorValue.String() != "" {
		t.Fatalf("waitCompleted should not fail, got error=%q", errorValue.String())
	}
}

func TestCodexEventMetricsCountsAndReset(t *testing.T) {
	rt, err := NewRuntime(Options{
		Name: "runtime-test-metrics",
		RPC:  fakeRPCBridge{},
		UI:   fakeUIBridge{},
	})
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
	defer func() { _ = rt.Close() }()

	_, err = rt.RunString(`
      const codex = require("codex");
      globalThis.__session = codex.connect();
    `)
	if err != nil {
		t.Fatalf("RunString() setup error = %v", err)
	}

	if emitErr := rt.EmitRPCNotification("thread/started", map[string]any{"id": "thread-1"}); emitErr != nil {
		t.Fatalf("EmitRPCNotification() first error = %v", emitErr)
	}
	if emitErr := rt.EmitRPCNotification("turn/completed", map[string]any{"threadId": "thread-1", "turnId": "turn-1"}); emitErr != nil {
		t.Fatalf("EmitRPCNotification() second error = %v", emitErr)
	}
	if emitErr := rt.EmitRPCRequest("req-1", "item/commandExecution/requestApproval", map[string]any{"command": "curl -I https://example.com"}); emitErr != nil {
		t.Fatalf("EmitRPCRequest() error = %v", emitErr)
	}

	_, err = rt.RunString(`
      const metrics = globalThis.__session.events.metrics();
      globalThis.__metrics = {
        counts: metrics.countByMethod(),
        notifications: metrics.totalNotifications(),
        requests: metrics.totalRequests()
      };
    `)
	if err != nil {
		t.Fatalf("RunString() metrics snapshot error = %v", err)
	}

	notifications, err := rt.RunString(`globalThis.__metrics.notifications`)
	if err != nil {
		t.Fatalf("RunString() notifications error = %v", err)
	}
	if notifications.ToInteger() != 2 {
		t.Fatalf("notifications mismatch: got=%d want=%d", notifications.ToInteger(), 2)
	}

	requests, err := rt.RunString(`globalThis.__metrics.requests`)
	if err != nil {
		t.Fatalf("RunString() requests error = %v", err)
	}
	if requests.ToInteger() != 1 {
		t.Fatalf("requests mismatch: got=%d want=%d", requests.ToInteger(), 1)
	}

	countCommandApproval, err := rt.RunString(`globalThis.__metrics.counts["item/commandExecution/requestApproval"]`)
	if err != nil {
		t.Fatalf("RunString() countByMethod command error = %v", err)
	}
	if countCommandApproval.ToInteger() != 1 {
		t.Fatalf("countByMethod command mismatch: got=%d want=%d", countCommandApproval.ToInteger(), 1)
	}

	_, err = rt.RunString(`
      const metrics2 = globalThis.__session.events.metrics();
      metrics2.reset();
      globalThis.__afterReset = {
        notifications: metrics2.totalNotifications(),
        requests: metrics2.totalRequests(),
        methodCountKeys: Object.keys(metrics2.countByMethod()).length
      };
    `)
	if err != nil {
		t.Fatalf("RunString() reset error = %v", err)
	}

	afterResetNotifications, err := rt.RunString(`globalThis.__afterReset.notifications`)
	if err != nil {
		t.Fatalf("RunString() after reset notifications error = %v", err)
	}
	if afterResetNotifications.ToInteger() != 0 {
		t.Fatalf("after reset notifications mismatch: got=%d want=0", afterResetNotifications.ToInteger())
	}

	afterResetRequests, err := rt.RunString(`globalThis.__afterReset.requests`)
	if err != nil {
		t.Fatalf("RunString() after reset requests error = %v", err)
	}
	if afterResetRequests.ToInteger() != 0 {
		t.Fatalf("after reset requests mismatch: got=%d want=0", afterResetRequests.ToInteger())
	}

	afterResetMethodKeys, err := rt.RunString(`globalThis.__afterReset.methodCountKeys`)
	if err != nil {
		t.Fatalf("RunString() after reset methodCountKeys error = %v", err)
	}
	if afterResetMethodKeys.ToInteger() != 0 {
		t.Fatalf("after reset methodCountKeys mismatch: got=%d want=0", afterResetMethodKeys.ToInteger())
	}
}

func TestCodexApprovalsSetPolicyAndRespond(t *testing.T) {
	rpcBridge := &recordingRPCBridge{}
	rt, err := NewRuntime(Options{
		Name: "runtime-test-approvals-policy",
		RPC:  rpcBridge,
		UI:   fakeUIBridge{},
	})
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
	defer func() { _ = rt.Close() }()

	_, err = rt.RunString(`
      const codex = require("codex");
      const session = codex.connect();
      session.approvals.setPolicy({
        command: (req) => req.commandText && req.commandText.startsWith("curl ") ? "acceptForSession" : "decline",
        fileChange: (_req) => "decline",
        fallback: (_req) => "cancel"
      });
      session.approvals.respond("manual-accept", "accept");
    `)
	if err != nil {
		t.Fatalf("RunString() setup error = %v", err)
	}

	if emitErr := rt.EmitRPCRequest("req-command", "item/commandExecution/requestApproval", map[string]any{"command": "curl -I https://example.com"}); emitErr != nil {
		t.Fatalf("EmitRPCRequest() command error = %v", emitErr)
	}
	if emitErr := rt.EmitRPCRequest("req-file", "item/fileChange/requestApproval", map[string]any{"path": "README.md"}); emitErr != nil {
		t.Fatalf("EmitRPCRequest() file error = %v", emitErr)
	}
	if emitErr := rt.EmitRPCRequest("req-fallback", "item/other/requestApproval", map[string]any{}); emitErr != nil {
		t.Fatalf("EmitRPCRequest() fallback error = %v", emitErr)
	}

	deadline := time.Now().Add(2 * time.Second)
	var responses []capturedResponse
	for {
		responses = rpcBridge.snapshot()
		if len(responses) >= 4 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for approval policy responses, got=%d", len(responses))
		}
		time.Sleep(10 * time.Millisecond)
	}

	decisionByID := map[string]any{}
	for _, response := range responses {
		id, _ := response.id.(string)
		if id == "" {
			continue
		}
		decisionByID[id] = response.result
	}

	expectDecision := func(id string, want any) {
		got, ok := decisionByID[id]
		if !ok {
			t.Fatalf("missing response for id=%s (all=%#v)", id, decisionByID)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("response mismatch for id=%s: got=%#v want=%#v", id, got, want)
		}
	}

	expectDecision("manual-accept", map[string]any{"decision": "accept"})
	expectDecision("req-command", map[string]any{"decision": "acceptForSession"})
	expectDecision("req-file", map[string]any{"decision": "decline"})
	expectDecision("req-fallback", map[string]any{"decision": "cancel"})
}
