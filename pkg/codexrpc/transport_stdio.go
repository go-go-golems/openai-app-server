package codexrpc

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

// StdioTransport exchanges JSON-RPC envelopes over a spawned process stdio stream.
type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader

	writeMu sync.Mutex
	closeMu sync.Mutex
	closed  bool
}

func NewStdioTransport(command string, args ...string) (*StdioTransport, error) {
	if command == "" {
		return nil, fmt.Errorf("codexrpc: stdio command is required")
	}

	cmd := exec.Command(command, args...)
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("codexrpc: create stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("codexrpc: create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("codexrpc: start stdio command: %w", err)
	}

	return &StdioTransport{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
	}, nil
}

func (t *StdioTransport) Send(ctx context.Context, msg *Message) error {
	if msg == nil {
		return fmt.Errorf("codexrpc: send nil message")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("codexrpc: marshal message: %w", err)
	}
	b = append(b, '\n')

	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	if t.isClosed() {
		return ErrClientClosed
	}
	if _, err := t.stdin.Write(b); err != nil {
		return fmt.Errorf("codexrpc: write message: %w", err)
	}
	return nil
}

func (t *StdioTransport) Recv(ctx context.Context) (*Message, error) {
	type recvResult struct {
		line []byte
		err  error
	}
	ch := make(chan recvResult, 1)
	go func() {
		line, err := t.stdout.ReadBytes('\n')
		ch <- recvResult{line: line, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		if res.err != nil {
			return nil, res.err
		}
		msg := &Message{}
		if err := json.Unmarshal(res.line, msg); err != nil {
			return nil, fmt.Errorf("codexrpc: decode message: %w", err)
		}
		return msg, nil
	}
}

func (t *StdioTransport) Close() error {
	t.closeMu.Lock()
	defer t.closeMu.Unlock()
	if t.closed {
		return nil
	}
	t.closed = true

	if t.stdin != nil {
		_ = t.stdin.Close()
	}
	if t.cmd != nil && t.cmd.Process != nil {
		_ = t.cmd.Process.Kill()
		_ = t.cmd.Wait()
	}
	return nil
}

func (t *StdioTransport) isClosed() bool {
	t.closeMu.Lock()
	defer t.closeMu.Unlock()
	return t.closed
}
