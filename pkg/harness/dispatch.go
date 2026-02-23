package harness

import (
	"context"
	"fmt"
	"sync"
)

type Dispatcher struct {
	mu sync.RWMutex

	notificationHandlers map[string][]NotificationHandler
	requestHandlers      map[string][]RequestHandler
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		notificationHandlers: map[string][]NotificationHandler{},
		requestHandlers:      map[string][]RequestHandler{},
	}
}

func (d *Dispatcher) OnNotification(method string, handler NotificationHandler) {
	if handler == nil {
		return
	}
	if method == "" {
		method = "*"
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.notificationHandlers[method] = append(d.notificationHandlers[method], handler)
}

func (d *Dispatcher) OnRequest(method string, handler RequestHandler) {
	if handler == nil {
		return
	}
	if method == "" {
		method = "*"
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.requestHandlers[method] = append(d.requestHandlers[method], handler)
}

func (d *Dispatcher) DispatchNotification(ctx context.Context, evt NotificationEvent) error {
	handlers := d.notificationHandlersFor(evt.Method)
	for _, h := range handlers {
		if err := h(ctx, evt); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dispatcher) DispatchRequest(rc *RequestContext) (RequestDiagnostics, error) {
	if rc == nil {
		return RequestDiagnostics{}, fmt.Errorf("harness: request context is nil")
	}

	req := rc.Request()
	handlers := d.requestHandlersFor(req.Method)
	for _, h := range handlers {
		rc.MarkHandlerInvoked()
		if err := h(rc); err != nil {
			return rc.Diagnostics(), err
		}
	}

	diag := rc.Diagnostics()
	if !diag.Responded {
		return diag, fmt.Errorf("%w: method=%s id=%v handlers=%d", ErrRequestUnanswered, req.Method, req.ID, diag.HandlerInvoked)
	}
	return diag, nil
}

func (d *Dispatcher) notificationHandlersFor(method string) []NotificationHandler {
	d.mu.RLock()
	defer d.mu.RUnlock()

	handlers := append([]NotificationHandler{}, d.notificationHandlers[method]...)
	handlers = append(handlers, d.notificationHandlers["*"]...)
	return handlers
}

func (d *Dispatcher) requestHandlersFor(method string) []RequestHandler {
	d.mu.RLock()
	defer d.mu.RUnlock()

	handlers := append([]RequestHandler{}, d.requestHandlers[method]...)
	handlers = append(handlers, d.requestHandlers["*"]...)
	return handlers
}
