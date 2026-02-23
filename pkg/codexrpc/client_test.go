package codexrpc

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

type fakeTransport struct {
	sendHook func(msg *Message)

	mu     sync.Mutex
	sent   []*Message
	recvCh chan *Message
	closed bool
}

func newFakeTransport(sendHook func(msg *Message)) *fakeTransport {
	return &fakeTransport{
		sendHook: sendHook,
		recvCh:   make(chan *Message, 32),
	}
}

func (f *fakeTransport) Send(ctx context.Context, msg *Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	f.mu.Lock()
	f.sent = append(f.sent, msg)
	f.mu.Unlock()

	if f.sendHook != nil {
		f.sendHook(msg)
	}
	return nil
}

func (f *fakeTransport) Recv(ctx context.Context) (*Message, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case msg, ok := <-f.recvCh:
		if !ok {
			return nil, io.EOF
		}
		return msg, nil
	}
}

func (f *fakeTransport) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return nil
	}
	f.closed = true
	close(f.recvCh)
	return nil
}

func (f *fakeTransport) push(msg *Message) {
	f.recvCh <- msg
}

func (f *fakeTransport) sentSnapshot() []*Message {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*Message, len(f.sent))
	copy(out, f.sent)
	return out
}

func TestConnectHandshakeSuccess(t *testing.T) {
	var ft *fakeTransport
	ft = newFakeTransport(func(msg *Message) {
		if msg.Method == "initialize" {
			ft.push(&Message{ID: msg.ID, Result: []byte(`{"capabilities":{}}`)})
		}
	})

	c := NewClient(ft)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := c.Connect(ctx, map[string]any{"clientInfo": map[string]any{"name": "test"}}); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	sent := ft.sentSnapshot()
	if len(sent) < 2 {
		t.Fatalf("expected at least two outbound messages, got %d", len(sent))
	}
	if sent[0].Method != "initialize" {
		t.Fatalf("first message must be initialize, got %q", sent[0].Method)
	}
	if sent[1].Method != "initialized" {
		t.Fatalf("second message must be initialized, got %q", sent[1].Method)
	}
}

func TestRequestRejectsBeforeHandshake(t *testing.T) {
	ft := newFakeTransport(nil)
	c := NewClient(ft)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := c.Request(ctx, "thread/list", map[string]any{"limit": 10})
	if !errors.Is(err, ErrHandshakeRequired) {
		t.Fatalf("expected ErrHandshakeRequired, got %v", err)
	}

	if got := len(ft.sentSnapshot()); got != 0 {
		t.Fatalf("expected no outbound message, got %d", got)
	}
}

func TestInitializeRejectedAfterHandshake(t *testing.T) {
	var ft *fakeTransport
	ft = newFakeTransport(func(msg *Message) {
		if msg.Method == "initialize" {
			ft.push(&Message{ID: msg.ID, Result: []byte(`{"capabilities":{}}`)})
		}
	})

	c := NewClient(ft)
	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := c.Connect(ctx, map[string]any{"clientInfo": map[string]any{"name": "test"}}); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	_, err := c.Request(ctx, "initialize", map[string]any{})
	if !errors.Is(err, ErrInitializeAlreadyDone) {
		t.Fatalf("expected ErrInitializeAlreadyDone, got %v", err)
	}
}

func TestNotificationWildcardHandler(t *testing.T) {
	ft := newFakeTransport(nil)
	c := NewClient(ft)
	defer func() { _ = c.Close() }()

	got := make(chan string, 1)
	_ = c.OnNotification("*", func(_ context.Context, msg *Message) {
		got <- msg.Method
	})

	c.ensureReadLoop()
	ft.push(&Message{Method: "thread/started", Params: []byte(`{"id":"thread-1"}`)})

	select {
	case method := <-got:
		if method != "thread/started" {
			t.Fatalf("unexpected method: %s", method)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for wildcard notification handler")
	}
}
