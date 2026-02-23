package codexrpc

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestClientThreadReadWrappedPayload(t *testing.T) {
	mem := NewMemoryTransport()
	mem.OnSend = func(msg *Message) {
		switch msg.Method {
		case "initialize":
			mem.Push(&Message{ID: msg.ID, Result: []byte(`{"capabilities":{}}`)})
		case "thread/read":
			mem.Push(&Message{ID: msg.ID, Result: []byte(`{"thread":{"id":"thread-1","status":"active","cwd":"/repo","turns":[{"id":"turn-1"}]}}`)})
		}
	}

	client := NewClient(mem)
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Connect(ctx, map[string]any{"clientInfo": map[string]any{"name": "test"}}); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	thread, err := client.ThreadRead(ctx, "thread-1", true)
	if err != nil {
		t.Fatalf("ThreadRead() error = %v", err)
	}
	if thread.ID != "thread-1" || thread.Cwd != "/repo" {
		t.Fatalf("unexpected thread payload: %#v", thread)
	}
	if len(thread.Turns) != 1 {
		t.Fatalf("expected turns to be populated, got %#v", thread.Turns)
	}
}

func TestClientThreadReadDirectPayload(t *testing.T) {
	mem := NewMemoryTransport()
	mem.OnSend = func(msg *Message) {
		switch msg.Method {
		case "initialize":
			mem.Push(&Message{ID: msg.ID, Result: []byte(`{"capabilities":{}}`)})
		case "thread/read":
			mem.Push(&Message{ID: msg.ID, Result: []byte(`{"id":"thread-2","status":"completed","cwd":"/repo-2"}`)})
		}
	}

	client := NewClient(mem)
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Connect(ctx, map[string]any{"clientInfo": map[string]any{"name": "test"}}); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	thread, err := client.ThreadRead(ctx, "thread-2", false)
	if err != nil {
		t.Fatalf("ThreadRead() error = %v", err)
	}
	if thread.ID != "thread-2" || thread.Status != "completed" {
		t.Fatalf("unexpected thread payload: %#v", thread)
	}
}

func TestClientThreadReadMalformedPayload(t *testing.T) {
	mem := NewMemoryTransport()
	mem.OnSend = func(msg *Message) {
		switch msg.Method {
		case "initialize":
			mem.Push(&Message{ID: msg.ID, Result: []byte(`{"capabilities":{}}`)})
		case "thread/read":
			mem.Push(&Message{ID: msg.ID, Result: []byte(`42`)})
		}
	}

	client := NewClient(mem)
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Connect(ctx, map[string]any{"clientInfo": map[string]any{"name": "test"}}); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	_, err := client.ThreadRead(ctx, "thread-x", false)
	if err == nil {
		t.Fatalf("expected malformed payload error")
	}
	if !strings.Contains(err.Error(), "unexpected thread/read result payload") {
		t.Fatalf("unexpected error: %v", err)
	}
}
