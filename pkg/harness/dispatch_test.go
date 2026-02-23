package harness

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type recordingSender struct {
	results []any
	errors  []map[string]any
}

func (r *recordingSender) Respond(_ context.Context, _ any, result any) error {
	r.results = append(r.results, result)
	return nil
}

func (r *recordingSender) RespondError(_ context.Context, _ any, code int, message string, data any) error {
	r.errors = append(r.errors, map[string]any{
		"code":    code,
		"message": message,
		"data":    data,
	})
	return nil
}

func TestComposeDeterministicOrder(t *testing.T) {
	calls := []string{}

	d := Compose(
		Harness{
			Name: "h1",
			Notifications: []NotificationBinding{
				{Method: "turn/started", Handler: func(_ context.Context, _ NotificationEvent) error {
					calls = append(calls, "h1-notif")
					return nil
				}},
			},
			Requests: []RequestBinding{
				{Method: "item/commandExecution/requestApproval", Handler: func(rc *RequestContext) error {
					calls = append(calls, "h1-req")
					return rc.Respond(map[string]any{"ok": true})
				}},
			},
		},
		Harness{
			Name: "h2",
			Notifications: []NotificationBinding{
				{Method: "turn/started", Handler: func(_ context.Context, _ NotificationEvent) error {
					calls = append(calls, "h2-notif")
					return nil
				}},
			},
			Requests: []RequestBinding{
				{Method: "item/commandExecution/requestApproval", Handler: func(rc *RequestContext) error {
					calls = append(calls, "h2-req")
					return rc.Respond(map[string]any{"ok": true})
				}},
			},
		},
	)

	if err := d.DispatchNotification(context.Background(), NotificationEvent{Method: "turn/started"}); err != nil {
		t.Fatalf("DispatchNotification() error = %v", err)
	}

	sender := &recordingSender{}
	rc := NewRequestContext(context.Background(), RequestEvent{ID: "req-1", Method: "item/commandExecution/requestApproval"}, sender)
	_, err := d.DispatchRequest(rc)
	if err == nil {
		t.Fatalf("expected duplicate response error from second request handler")
	}
	if !errors.Is(err, ErrRequestAlreadyResponded) {
		t.Fatalf("expected ErrRequestAlreadyResponded, got %v", err)
	}

	if len(sender.results) != 1 {
		t.Fatalf("expected one response to be sent, got %d", len(sender.results))
	}

	wantPrefix := []string{"h1-notif", "h2-notif", "h1-req", "h2-req"}
	if !reflect.DeepEqual(calls, wantPrefix) {
		t.Fatalf("unexpected call order: got=%v want=%v", calls, wantPrefix)
	}
}

func TestDispatchRequestUnanswered(t *testing.T) {
	d := NewDispatcher()
	d.OnRequest("item/fileChange/requestApproval", func(_ *RequestContext) error {
		return nil
	})

	rc := NewRequestContext(context.Background(), RequestEvent{ID: "req-2", Method: "item/fileChange/requestApproval"}, &recordingSender{})
	diag, err := d.DispatchRequest(rc)
	if err == nil {
		t.Fatalf("expected unanswered request error")
	}
	if !errors.Is(err, ErrRequestUnanswered) {
		t.Fatalf("expected ErrRequestUnanswered, got %v", err)
	}
	if diag.Responded {
		t.Fatalf("expected diagnostics Responded=false, got %+v", diag)
	}
	if diag.HandlerInvoked != 1 {
		t.Fatalf("expected HandlerInvoked=1, got %+v", diag)
	}
}

func TestDispatchRequestWildcardFallback(t *testing.T) {
	d := NewDispatcher()
	d.OnRequest("*", func(rc *RequestContext) error {
		return rc.Respond(map[string]any{"accepted": true})
	})

	sender := &recordingSender{}
	rc := NewRequestContext(context.Background(), RequestEvent{ID: "req-3", Method: "unknown/method"}, sender)
	diag, err := d.DispatchRequest(rc)
	if err != nil {
		t.Fatalf("DispatchRequest() error = %v", err)
	}
	if !diag.Responded || diag.ResponseKind != "result" {
		t.Fatalf("unexpected diagnostics: %+v", diag)
	}
	if len(sender.results) != 1 {
		t.Fatalf("expected wildcard response to be sent")
	}
}
