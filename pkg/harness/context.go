package harness

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrRequestAlreadyResponded = errors.New("harness: request already responded")
	ErrRequestUnanswered       = errors.New("harness: request was not answered by any handler")
)

type NotificationEvent struct {
	Method string
	Params map[string]any
}

type RequestEvent struct {
	ID     any
	Method string
	Params map[string]any
}

type ResponseSender interface {
	Respond(ctx context.Context, id any, result any) error
	RespondError(ctx context.Context, id any, code int, message string, data any) error
}

type RequestDiagnostics struct {
	RequestID      any
	Method         string
	Responded      bool
	ResponseKind   string
	ResponseCount  int
	HandlerInvoked int
}

type RequestContext struct {
	ctx     context.Context
	request RequestEvent
	sender  ResponseSender

	mu             sync.Mutex
	responded      bool
	responseKind   string
	responseCount  int
	handlerInvoked int
}

func NewRequestContext(ctx context.Context, request RequestEvent, sender ResponseSender) *RequestContext {
	if ctx == nil {
		ctx = context.Background()
	}
	return &RequestContext{
		ctx:     ctx,
		request: request,
		sender:  sender,
	}
}

func (rc *RequestContext) Request() RequestEvent {
	return rc.request
}

func (rc *RequestContext) Context() context.Context {
	return rc.ctx
}

func (rc *RequestContext) MarkHandlerInvoked() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.handlerInvoked++
}

func (rc *RequestContext) Respond(result any) error {
	rc.mu.Lock()
	if rc.responded {
		rc.responseCount++
		rc.mu.Unlock()
		return fmt.Errorf("%w: method=%s id=%v", ErrRequestAlreadyResponded, rc.request.Method, rc.request.ID)
	}
	rc.responded = true
	rc.responseKind = "result"
	rc.responseCount++
	rc.mu.Unlock()

	if rc.sender == nil {
		return nil
	}
	return rc.sender.Respond(rc.ctx, rc.request.ID, result)
}

func (rc *RequestContext) RespondError(code int, message string, data any) error {
	rc.mu.Lock()
	if rc.responded {
		rc.responseCount++
		rc.mu.Unlock()
		return fmt.Errorf("%w: method=%s id=%v", ErrRequestAlreadyResponded, rc.request.Method, rc.request.ID)
	}
	rc.responded = true
	rc.responseKind = "error"
	rc.responseCount++
	rc.mu.Unlock()

	if rc.sender == nil {
		return nil
	}
	return rc.sender.RespondError(rc.ctx, rc.request.ID, code, message, data)
}

func (rc *RequestContext) Diagnostics() RequestDiagnostics {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return RequestDiagnostics{
		RequestID:      rc.request.ID,
		Method:         rc.request.Method,
		Responded:      rc.responded,
		ResponseKind:   rc.responseKind,
		ResponseCount:  rc.responseCount,
		HandlerInvoked: rc.handlerInvoked,
	}
}
