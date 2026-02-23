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
