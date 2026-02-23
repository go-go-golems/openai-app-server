package codexrpc

import "context"

// Transport carries JSON-RPC envelopes between the client and Codex App Server.
type Transport interface {
	Send(ctx context.Context, msg *Message) error
	Recv(ctx context.Context) (*Message, error)
	Close() error
}
