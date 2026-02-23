package codexrpc

import (
	"errors"
	"fmt"
)

var (
	ErrHandshakeRequired     = errors.New("codexrpc: handshake required")
	ErrInitializeAlreadyDone = errors.New("codexrpc: initialize already completed")
	ErrInitializedOutOfOrder = errors.New("codexrpc: initialized notification out of order")
	ErrClientClosed          = errors.New("codexrpc: client closed")
)

// ResponseError wraps an error returned by the remote JSON-RPC endpoint.
type ResponseError struct {
	Method  string
	Code    int
	Message string
}

func (e *ResponseError) Error() string {
	if e == nil {
		return "codexrpc: response error"
	}
	return fmt.Sprintf("codexrpc: method %s failed (code=%d): %s", e.Method, e.Code, e.Message)
}
