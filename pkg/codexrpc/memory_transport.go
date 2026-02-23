package codexrpc

import (
	"context"
	"io"
	"sync"
)

// MemoryTransport is an in-memory transport useful for integration tests.
type MemoryTransport struct {
	OnSend func(msg *Message)

	mu     sync.Mutex
	sent   []*Message
	recvCh chan *Message
	closed bool
}

func NewMemoryTransport() *MemoryTransport {
	return &MemoryTransport{recvCh: make(chan *Message, 64)}
}

func (m *MemoryTransport) Send(ctx context.Context, msg *Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	m.mu.Lock()
	m.sent = append(m.sent, msg)
	hook := m.OnSend
	m.mu.Unlock()

	if hook != nil {
		hook(msg)
	}
	return nil
}

func (m *MemoryTransport) Recv(ctx context.Context) (*Message, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case msg, ok := <-m.recvCh:
		if !ok {
			return nil, io.EOF
		}
		return msg, nil
	}
}

func (m *MemoryTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	close(m.recvCh)
	return nil
}

func (m *MemoryTransport) Push(msg *Message) {
	m.recvCh <- msg
}

func (m *MemoryTransport) SentSnapshot() []*Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Message, len(m.sent))
	copy(out, m.sent)
	return out
}
