package codexrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type handshakeState int

const (
	stateNew handshakeState = iota
	stateInitialized
	stateReady
	stateClosed
)

type NotificationHandler func(ctx context.Context, msg *Message)
type RequestHandler func(ctx context.Context, msg *Message)

type EventBufferConfig struct {
	MaxNotifications int
	MaxRequests      int
}

type ClientOptions struct {
	EventBuffer EventBufferConfig
}

type EventRecord struct {
	Timestamp time.Time
	Method    string
	ID        any
}

// Client manages JSON-RPC request/response routing and handshake sequencing.
type Client struct {
	transport Transport

	stateMu sync.RWMutex
	state   handshakeState

	nextID atomic.Int64

	pendingMu sync.Mutex
	pending   map[string]chan *Message

	notificationMu sync.RWMutex
	notifications  map[string][]NotificationHandler

	requestMu sync.RWMutex
	requests  map[string][]RequestHandler

	historyMu             sync.RWMutex
	maxNotifications      int
	maxRequests           int
	notificationEventRing []EventRecord
	requestEventRing      []EventRecord

	startOnce sync.Once
	closeOnce sync.Once
	done      chan struct{}
}

func NewClient(transport Transport) *Client {
	return NewClientWithOptions(transport, ClientOptions{})
}

func NewClientWithOptions(transport Transport, opts ClientOptions) *Client {
	maxNotifications := opts.EventBuffer.MaxNotifications
	if maxNotifications <= 0 {
		maxNotifications = 1000
	}
	maxRequests := opts.EventBuffer.MaxRequests
	if maxRequests <= 0 {
		maxRequests = 500
	}

	return &Client{
		transport:             transport,
		state:                 stateNew,
		pending:               map[string]chan *Message{},
		notifications:         map[string][]NotificationHandler{},
		requests:              map[string][]RequestHandler{},
		maxNotifications:      maxNotifications,
		maxRequests:           maxRequests,
		notificationEventRing: []EventRecord{},
		requestEventRing:      []EventRecord{},
		done:                  make(chan struct{}),
	}
}

func (c *Client) Connect(ctx context.Context, initializeParams any) error {
	if _, err := c.Request(ctx, "initialize", initializeParams); err != nil {
		return err
	}
	if err := c.Notify(ctx, "initialized", map[string]any{}); err != nil {
		return err
	}
	return nil
}

func (c *Client) Request(ctx context.Context, method string, params any) (*Message, error) {
	if err := c.validateRequestMethod(method); err != nil {
		return nil, err
	}
	if c.isClosed() {
		return nil, ErrClientClosed
	}
	c.ensureReadLoop()

	id := c.nextID.Add(1)
	msg, err := NewRequest(id, method, params)
	if err != nil {
		return nil, err
	}

	responseCh := make(chan *Message, 1)
	key := idKey(id)
	c.pendingMu.Lock()
	c.pending[key] = responseCh
	c.pendingMu.Unlock()

	if err := c.transport.Send(ctx, msg); err != nil {
		c.pendingMu.Lock()
		delete(c.pending, key)
		c.pendingMu.Unlock()
		return nil, err
	}

	select {
	case <-ctx.Done():
		c.pendingMu.Lock()
		delete(c.pending, key)
		c.pendingMu.Unlock()
		return nil, ctx.Err()
	case <-c.done:
		return nil, ErrClientClosed
	case resp := <-responseCh:
		if resp == nil {
			return nil, ErrClientClosed
		}
		if resp.Error != nil {
			return nil, &ResponseError{Method: method, Code: resp.Error.Code, Message: resp.Error.Message}
		}
		if method == "initialize" {
			c.setState(stateInitialized)
		}
		return resp, nil
	}
}

func (c *Client) Notify(ctx context.Context, method string, params any) error {
	if err := c.validateNotifyMethod(method); err != nil {
		return err
	}
	if c.isClosed() {
		return ErrClientClosed
	}
	c.ensureReadLoop()

	msg, err := NewNotification(method, params)
	if err != nil {
		return err
	}
	if err := c.transport.Send(ctx, msg); err != nil {
		return err
	}
	if method == "initialized" {
		c.setState(stateReady)
	}
	return nil
}

func (c *Client) Respond(ctx context.Context, id any, result any) error {
	if c.getState() != stateReady {
		return ErrHandshakeRequired
	}
	if c.isClosed() {
		return ErrClientClosed
	}
	c.ensureReadLoop()

	msg, err := NewResponse(id, result)
	if err != nil {
		return err
	}
	return c.transport.Send(ctx, msg)
}

func (c *Client) RespondError(ctx context.Context, id any, code int, message string, data any) error {
	if c.getState() != stateReady {
		return ErrHandshakeRequired
	}
	if c.isClosed() {
		return ErrClientClosed
	}
	c.ensureReadLoop()

	msg, err := NewErrorResponse(id, code, message, data)
	if err != nil {
		return err
	}
	return c.transport.Send(ctx, msg)
}

func (c *Client) OnNotification(method string, handler NotificationHandler) (unsubscribe func()) {
	c.notificationMu.Lock()
	defer c.notificationMu.Unlock()
	c.notifications[method] = append(c.notifications[method], handler)
	idx := len(c.notifications[method]) - 1
	return func() {
		c.notificationMu.Lock()
		defer c.notificationMu.Unlock()
		if idx < len(c.notifications[method]) {
			c.notifications[method][idx] = nil
		}
	}
}

func (c *Client) OnRequest(method string, handler RequestHandler) (unsubscribe func()) {
	c.requestMu.Lock()
	defer c.requestMu.Unlock()
	c.requests[method] = append(c.requests[method], handler)
	idx := len(c.requests[method]) - 1
	return func() {
		c.requestMu.Lock()
		defer c.requestMu.Unlock()
		if idx < len(c.requests[method]) {
			c.requests[method][idx] = nil
		}
	}
}

func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		c.setState(stateClosed)
		close(c.done)

		c.pendingMu.Lock()
		for key, ch := range c.pending {
			delete(c.pending, key)
			close(ch)
		}
		c.pendingMu.Unlock()
	})
	if c.transport != nil {
		return c.transport.Close()
	}
	return nil
}

func (c *Client) ensureReadLoop() {
	c.startOnce.Do(func() {
		go c.readLoop()
	})
}

func (c *Client) readLoop() {
	ctx := context.Background()
	for {
		select {
		case <-c.done:
			return
		default:
		}

		msg, err := c.transport.Recv(ctx)
		if err != nil {
			_ = c.Close()
			return
		}
		if msg == nil {
			continue
		}

		switch {
		case msg.IsResponse():
			c.dispatchResponse(msg)
		case msg.IsNotification():
			c.dispatchNotification(msg)
		case msg.IsRequest():
			c.dispatchRequest(msg)
		}
	}
}

func (c *Client) dispatchResponse(msg *Message) {
	key := idKey(msg.ID)
	c.pendingMu.Lock()
	ch, ok := c.pending[key]
	if ok {
		delete(c.pending, key)
	}
	c.pendingMu.Unlock()
	if !ok || ch == nil {
		return
	}
	ch <- msg
}

func (c *Client) dispatchNotification(msg *Message) {
	c.recordNotification(msg)

	c.notificationMu.RLock()
	handlers := append([]NotificationHandler{}, c.notifications[msg.Method]...)
	handlers = append(handlers, c.notifications["*"]...)
	c.notificationMu.RUnlock()

	for _, h := range handlers {
		if h != nil {
			h(context.Background(), msg)
		}
	}
}

func (c *Client) dispatchRequest(msg *Message) {
	c.recordRequest(msg)

	c.requestMu.RLock()
	handlers := append([]RequestHandler{}, c.requests[msg.Method]...)
	handlers = append(handlers, c.requests["*"]...)
	c.requestMu.RUnlock()

	for _, h := range handlers {
		if h != nil {
			h(context.Background(), msg)
		}
	}
}

func (c *Client) validateRequestMethod(method string) error {
	state := c.getState()
	switch method {
	case "initialize":
		if state != stateNew {
			return ErrInitializeAlreadyDone
		}
		return nil
	default:
		if state != stateReady {
			return ErrHandshakeRequired
		}
		return nil
	}
}

func (c *Client) validateNotifyMethod(method string) error {
	state := c.getState()
	switch method {
	case "initialized":
		if state != stateInitialized {
			return ErrInitializedOutOfOrder
		}
		return nil
	default:
		if state != stateReady {
			return ErrHandshakeRequired
		}
		return nil
	}
}

func (c *Client) getState() handshakeState {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.state
}

func (c *Client) setState(s handshakeState) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	c.state = s
}

func (c *Client) isClosed() bool {
	return c.getState() == stateClosed
}

func (c *Client) RecentNotifications() []EventRecord {
	c.historyMu.RLock()
	defer c.historyMu.RUnlock()
	out := make([]EventRecord, len(c.notificationEventRing))
	copy(out, c.notificationEventRing)
	return out
}

func (c *Client) RecentRequests() []EventRecord {
	c.historyMu.RLock()
	defer c.historyMu.RUnlock()
	out := make([]EventRecord, len(c.requestEventRing))
	copy(out, c.requestEventRing)
	return out
}

func (c *Client) recordNotification(msg *Message) {
	if msg == nil {
		return
	}
	c.historyMu.Lock()
	defer c.historyMu.Unlock()

	c.notificationEventRing = append(c.notificationEventRing, EventRecord{
		Timestamp: time.Now(),
		Method:    msg.Method,
		ID:        msg.ID,
	})
	if len(c.notificationEventRing) > c.maxNotifications {
		c.notificationEventRing = c.notificationEventRing[len(c.notificationEventRing)-c.maxNotifications:]
	}
}

func (c *Client) recordRequest(msg *Message) {
	if msg == nil {
		return
	}
	c.historyMu.Lock()
	defer c.historyMu.Unlock()

	c.requestEventRing = append(c.requestEventRing, EventRecord{
		Timestamp: time.Now(),
		Method:    msg.Method,
		ID:        msg.ID,
	})
	if len(c.requestEventRing) > c.maxRequests {
		c.requestEventRing = c.requestEventRing[len(c.requestEventRing)-c.maxRequests:]
	}
}

func idKey(id any) string {
	if id == nil {
		return ""
	}
	b, err := json.Marshal(id)
	if err != nil {
		return fmt.Sprintf("%v", id)
	}
	return string(b)
}
