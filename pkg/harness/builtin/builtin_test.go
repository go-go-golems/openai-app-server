package builtin

import (
	"context"
	"testing"

	"github.com/go-go-golems/openai-app-server/pkg/harness"
)

type sender struct {
	results []any
}

func (s *sender) Respond(_ context.Context, _ any, result any) error {
	s.results = append(s.results, result)
	return nil
}

func (s *sender) RespondError(_ context.Context, _ any, _ int, _ string, _ any) error {
	return nil
}

func dispatchRequest(t *testing.T, h harness.Harness, method string, params map[string]any) map[string]any {
	t.Helper()
	d := harness.Compose(h)
	s := &sender{}
	rc := harness.NewRequestContext(context.Background(), harness.RequestEvent{ID: "req-1", Method: method, Params: params}, s)
	_, err := d.DispatchRequest(rc)
	if err != nil {
		t.Fatalf("DispatchRequest() error = %v", err)
	}
	if len(s.results) != 1 {
		t.Fatalf("expected one response result, got %d", len(s.results))
	}
	result, ok := s.results[0].(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %#v", s.results[0])
	}
	return result
}

func TestAutopilotCommandApprovals(t *testing.T) {
	h := NewAutopilotHarness(AutopilotConfig{SafeCommandPrefixes: []string{"go test", "npm test"}})

	safe := dispatchRequest(t, h, "item/commandExecution/requestApproval", map[string]any{"command": "go test ./..."})
	if safe["decision"] != "acceptForSession" {
		t.Fatalf("expected safe command acceptance, got %#v", safe)
	}

	risky := dispatchRequest(t, h, "item/commandExecution/requestApproval", map[string]any{"command": "curl -I https://example.com"})
	if risky["decision"] != "decline" {
		t.Fatalf("expected risky command decline, got %#v", risky)
	}
}

func TestAutopilotFileChangeApprovals(t *testing.T) {
	h := NewAutopilotHarness(AutopilotConfig{
		DeniedPathPrefixes: []string{"secrets/", ".env"},
		MaxChangedLines:    20,
	})

	small := dispatchRequest(t, h, "item/fileChange/requestApproval", map[string]any{
		"changes": []any{
			map[string]any{"path": "pkg/main.go", "added": float64(3), "removed": float64(2)},
		},
	})
	if small["decision"] != "acceptForSession" {
		t.Fatalf("expected small safe change acceptance, got %#v", small)
	}

	deniedPath := dispatchRequest(t, h, "item/fileChange/requestApproval", map[string]any{
		"changes": []any{
			map[string]any{"path": "secrets/api.key", "added": float64(1), "removed": float64(0)},
		},
	})
	if deniedPath["decision"] != "decline" {
		t.Fatalf("expected denied path decline, got %#v", deniedPath)
	}

	large := dispatchRequest(t, h, "item/fileChange/requestApproval", map[string]any{
		"changes": []any{
			map[string]any{"path": "pkg/huge.go", "added": float64(25), "removed": float64(0)},
		},
	})
	if large["decision"] != "decline" {
		t.Fatalf("expected large change decline, got %#v", large)
	}
}

type controllerCall struct {
	action      string
	threadID    string
	turnID      string
	instruction string
}

type fakePlanController struct {
	calls []controllerCall
}

func (f *fakePlanController) SteerTurn(_ context.Context, threadID string, turnID string, instruction string) error {
	f.calls = append(f.calls, controllerCall{action: "steer", threadID: threadID, turnID: turnID, instruction: instruction})
	return nil
}

func (f *fakePlanController) InterruptTurn(_ context.Context, threadID string, turnID string) error {
	f.calls = append(f.calls, controllerCall{action: "interrupt", threadID: threadID, turnID: turnID})
	return nil
}

func TestPlanGateHarness(t *testing.T) {
	controller := &fakePlanController{}
	h := NewPlanGateHarness(PlanGateConfig{
		Controller:       controller,
		SteerInstruction: "Proceed",
		Approve: func(_ string, _ string, plan any) (bool, error) {
			if m, ok := plan.(map[string]any); ok {
				if approved, ok := m["approved"].(bool); ok {
					return approved, nil
				}
			}
			return true, nil
		},
	})

	d := harness.Compose(h)

	if err := d.DispatchNotification(context.Background(), harness.NotificationEvent{
		Method: "turn/plan/updated",
		Params: map[string]any{"threadId": "thread-1", "turnId": "turn-1", "plan": map[string]any{"approved": true}},
	}); err != nil {
		t.Fatalf("DispatchNotification() error = %v", err)
	}
	if len(controller.calls) != 1 || controller.calls[0].action != "steer" {
		t.Fatalf("expected one steer call, got %#v", controller.calls)
	}

	if err := d.DispatchNotification(context.Background(), harness.NotificationEvent{
		Method: "turn/plan/updated",
		Params: map[string]any{"threadId": "thread-1", "turnId": "turn-1", "plan": map[string]any{"approved": false}},
	}); err != nil {
		t.Fatalf("DispatchNotification() error = %v", err)
	}
	if len(controller.calls) != 1 {
		t.Fatalf("expected duplicate turn plan to be ignored, got %#v", controller.calls)
	}

	if err := d.DispatchNotification(context.Background(), harness.NotificationEvent{
		Method: "turn/plan/updated",
		Params: map[string]any{"threadId": "thread-1", "turnId": "turn-2", "plan": map[string]any{"approved": false}},
	}); err != nil {
		t.Fatalf("DispatchNotification() error = %v", err)
	}
	if len(controller.calls) != 2 || controller.calls[1].action != "interrupt" {
		t.Fatalf("expected interrupt call on rejected plan, got %#v", controller.calls)
	}
}

type fakeTDDController struct {
	runResults []TDDTestResult
	runCalls   []string
	followups  []struct {
		threadID string
		input    string
	}
}

func (f *fakeTDDController) RunTests(_ context.Context, threadID string) (TDDTestResult, error) {
	f.runCalls = append(f.runCalls, threadID)
	if len(f.runResults) == 0 {
		return TDDTestResult{Passed: true}, nil
	}
	out := f.runResults[0]
	f.runResults = f.runResults[1:]
	return out, nil
}

func (f *fakeTDDController) StartFollowupTurn(_ context.Context, threadID string, input string) error {
	f.followups = append(f.followups, struct {
		threadID string
		input    string
	}{threadID: threadID, input: input})
	return nil
}

func TestTDDLoopHarness(t *testing.T) {
	controller := &fakeTDDController{
		runResults: []TDDTestResult{
			{Passed: false, Output: "FAIL: test A"},
			{Passed: true, Output: "ok"},
		},
	}
	h := NewTDDLoopHarness(TDDLoopConfig{Controller: controller, MaxIterations: 2})
	d := harness.Compose(h)

	if err := d.DispatchNotification(context.Background(), harness.NotificationEvent{
		Method: "turn/completed",
		Params: map[string]any{"threadId": "thread-1", "turn": map[string]any{"id": "turn-1", "status": "completed"}},
	}); err != nil {
		t.Fatalf("DispatchNotification() error = %v", err)
	}

	if len(controller.followups) != 1 {
		t.Fatalf("expected one followup turn, got %#v", controller.followups)
	}
	if controller.followups[0].threadID != "thread-1" {
		t.Fatalf("unexpected followup thread id: %#v", controller.followups)
	}

	if err := d.DispatchNotification(context.Background(), harness.NotificationEvent{
		Method: "turn/completed",
		Params: map[string]any{"threadId": "thread-1", "turn": map[string]any{"id": "turn-2", "status": "completed"}},
	}); err != nil {
		t.Fatalf("DispatchNotification() error = %v", err)
	}

	if len(controller.followups) != 1 {
		t.Fatalf("expected no additional followup after passing tests, got %#v", controller.followups)
	}

	if err := d.DispatchNotification(context.Background(), harness.NotificationEvent{
		Method: "turn/completed",
		Params: map[string]any{"threadId": "thread-1", "turn": map[string]any{"id": "turn-3", "status": "completed"}},
	}); err != nil {
		t.Fatalf("DispatchNotification() error = %v", err)
	}

	if len(controller.runCalls) != 2 {
		t.Fatalf("expected max-iteration guard to stop third run, got calls=%d", len(controller.runCalls))
	}
}

type fakeReviewController struct {
	reviews []ReviewResult
	calls   []struct {
		threadID string
		turnID   string
	}
	followups []struct {
		threadID string
		input    string
	}
}

func (f *fakeReviewController) StartReview(_ context.Context, threadID string, turnID string) (ReviewResult, error) {
	f.calls = append(f.calls, struct {
		threadID string
		turnID   string
	}{threadID: threadID, turnID: turnID})
	if len(f.reviews) == 0 {
		return ReviewResult{}, nil
	}
	out := f.reviews[0]
	f.reviews = f.reviews[1:]
	return out, nil
}

func (f *fakeReviewController) StartFollowupTurn(_ context.Context, threadID string, input string) error {
	f.followups = append(f.followups, struct {
		threadID string
		input    string
	}{threadID: threadID, input: input})
	return nil
}

func TestReviewGateHarness(t *testing.T) {
	controller := &fakeReviewController{
		reviews: []ReviewResult{
			{NeedsFollowup: true, Summary: "Fix edge case"},
			{NeedsFollowup: false, Summary: "Looks good"},
		},
	}
	h := NewReviewGateHarness(ReviewGateConfig{Controller: controller})
	d := harness.Compose(h)

	if err := d.DispatchNotification(context.Background(), harness.NotificationEvent{
		Method: "turn/completed",
		Params: map[string]any{"threadId": "thread-1", "turn": map[string]any{"id": "turn-1", "status": "completed"}},
	}); err != nil {
		t.Fatalf("DispatchNotification() error = %v", err)
	}
	if len(controller.followups) != 1 {
		t.Fatalf("expected followup for needs-followup review, got %#v", controller.followups)
	}

	if err := d.DispatchNotification(context.Background(), harness.NotificationEvent{
		Method: "turn/completed",
		Params: map[string]any{"threadId": "thread-1", "turn": map[string]any{"id": "turn-1", "status": "completed"}},
	}); err != nil {
		t.Fatalf("DispatchNotification() error = %v", err)
	}
	if len(controller.calls) != 1 {
		t.Fatalf("expected duplicate turn completion to be ignored, got %#v", controller.calls)
	}

	if err := d.DispatchNotification(context.Background(), harness.NotificationEvent{
		Method: "turn/completed",
		Params: map[string]any{"threadId": "thread-1", "turn": map[string]any{"id": "turn-2", "status": "completed"}},
	}); err != nil {
		t.Fatalf("DispatchNotification() error = %v", err)
	}
	if len(controller.calls) != 2 {
		t.Fatalf("expected second review call, got %#v", controller.calls)
	}
	if len(controller.followups) != 1 {
		t.Fatalf("expected no extra followup for clean review, got %#v", controller.followups)
	}
}

type fakeCompactController struct {
	calls []string
}

func (f *fakeCompactController) StartCompaction(_ context.Context, threadID string) error {
	f.calls = append(f.calls, threadID)
	return nil
}

func TestAutoCompactHarness(t *testing.T) {
	controller := &fakeCompactController{}
	h := NewAutoCompactHarness(AutoCompactConfig{
		Controller:      controller,
		SoftLimitRatio:  0.7,
		ResetBelowRatio: 0.5,
	})
	d := harness.Compose(h)

	emit := func(total float64, window float64) {
		err := d.DispatchNotification(context.Background(), harness.NotificationEvent{
			Method: "thread/tokenUsage/updated",
			Params: map[string]any{
				"threadId": "thread-1",
				"tokenUsage": map[string]any{
					"modelContextWindow": window,
					"total": map[string]any{
						"totalTokens": total,
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("DispatchNotification() error = %v", err)
		}
	}

	emit(300, 1000)
	if len(controller.calls) != 0 {
		t.Fatalf("expected no compaction below threshold, got %#v", controller.calls)
	}

	emit(750, 1000)
	if len(controller.calls) != 1 {
		t.Fatalf("expected one compaction at threshold breach, got %#v", controller.calls)
	}

	emit(820, 1000)
	if len(controller.calls) != 1 {
		t.Fatalf("expected no duplicate compaction while still above threshold, got %#v", controller.calls)
	}

	emit(400, 1000)
	emit(760, 1000)
	if len(controller.calls) != 2 {
		t.Fatalf("expected compaction to retrigger after reset, got %#v", controller.calls)
	}
}
