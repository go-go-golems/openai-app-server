package codexrpc

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestBuildInitializeParamsWithOptOutMethods(t *testing.T) {
	params := BuildInitializeParams(InitializeOptions{
		ClientName:                "openai-app-server",
		ClientVersion:             "0.1.0",
		OptOutNotificationMethods: []string{"item/agentMessage/delta", "turn/plan/updated"},
	})

	clientInfo, ok := params["clientInfo"].(map[string]any)
	if !ok {
		t.Fatalf("expected clientInfo map, got %#v", params["clientInfo"])
	}
	if clientInfo["name"] != "openai-app-server" {
		t.Fatalf("unexpected clientInfo.name: %#v", clientInfo)
	}

	caps, ok := params["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("expected capabilities map, got %#v", params["capabilities"])
	}
	optOut, ok := caps["optOutNotificationMethods"].([]string)
	if !ok {
		t.Fatalf("expected []string optOutNotificationMethods, got %#v", caps["optOutNotificationMethods"])
	}
	want := []string{"item/agentMessage/delta", "turn/plan/updated"}
	if !reflect.DeepEqual(optOut, want) {
		t.Fatalf("unexpected opt-out methods: got=%v want=%v", optOut, want)
	}
}

func TestRequestWithRetryRetryableResponseError(t *testing.T) {
	var ft *fakeTransport
	attempts := 0
	ft = newFakeTransport(func(msg *Message) {
		switch msg.Method {
		case "initialize":
			ft.push(&Message{ID: msg.ID, Result: []byte(`{"capabilities":{}}`)})
		case "thread/list":
			attempts++
			if attempts < 3 {
				ft.push(&Message{ID: msg.ID, Error: &RPCError{Code: RetryableServerOverloadedCode, Message: "overloaded"}})
				return
			}
			ft.push(&Message{ID: msg.ID, Result: []byte(`{"threads":[]}`)})
		}
	})

	c := NewClient(ft)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Connect(ctx, map[string]any{"clientInfo": map[string]any{"name": "test"}}); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	_, err := c.RequestWithRetry(ctx, "thread/list", map[string]any{"limit": 1}, RetryPolicy{
		MaxAttempts:    4,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     2 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("RequestWithRetry() error = %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestRequestWithRetryNonRetryableStops(t *testing.T) {
	var ft *fakeTransport
	attempts := 0
	ft = newFakeTransport(func(msg *Message) {
		switch msg.Method {
		case "initialize":
			ft.push(&Message{ID: msg.ID, Result: []byte(`{"capabilities":{}}`)})
		case "thread/list":
			attempts++
			ft.push(&Message{ID: msg.ID, Error: &RPCError{Code: -32602, Message: "invalid params"}})
		}
	})

	c := NewClient(ft)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Connect(ctx, map[string]any{"clientInfo": map[string]any{"name": "test"}}); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	_, err := c.RequestWithRetry(ctx, "thread/list", map[string]any{"limit": 1}, RetryPolicy{
		MaxAttempts:    4,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     2 * time.Millisecond,
	})
	if err == nil {
		t.Fatalf("expected non-retryable error")
	}
	var respErr *ResponseError
	if !errors.As(err, &respErr) || respErr.Code != -32602 {
		t.Fatalf("expected invalid-params response error, got %v", err)
	}
	if attempts != 1 {
		t.Fatalf("expected single attempt for non-retryable error, got %d", attempts)
	}
}

func TestClientEventBuffersAreBounded(t *testing.T) {
	c := NewClientWithOptions(newFakeTransport(nil), ClientOptions{
		EventBuffer: EventBufferConfig{
			MaxNotifications: 2,
			MaxRequests:      1,
		},
	})

	c.dispatchNotification(&Message{Method: "n-1"})
	c.dispatchNotification(&Message{Method: "n-2"})
	c.dispatchNotification(&Message{Method: "n-3"})

	notifs := c.RecentNotifications()
	if len(notifs) != 2 {
		t.Fatalf("expected 2 notifications retained, got %d", len(notifs))
	}
	if notifs[0].Method != "n-2" || notifs[1].Method != "n-3" {
		t.Fatalf("unexpected notification retention order: %#v", notifs)
	}

	c.dispatchRequest(&Message{ID: 1, Method: "r-1"})
	c.dispatchRequest(&Message{ID: 2, Method: "r-2"})

	reqs := c.RecentRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request retained, got %d", len(reqs))
	}
	if reqs[0].Method != "r-2" {
		t.Fatalf("unexpected request retention: %#v", reqs)
	}
}
