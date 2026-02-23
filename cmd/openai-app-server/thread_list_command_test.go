package main

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/go-go-golems/openai-app-server/pkg/codexrpc"
)

func TestThreadListCommandWithMemoryTransport(t *testing.T) {
	mem := codexrpc.NewMemoryTransport()
	mem.OnSend = func(msg *codexrpc.Message) {
		switch msg.Method {
		case "initialize":
			mem.Push(&codexrpc.Message{ID: msg.ID, Result: []byte(`{"capabilities":{}}`)})
		case "thread/list":
			mem.Push(&codexrpc.Message{ID: msg.ID, Result: []byte(`{"threads":[{"id":"thread-1","status":"active"},{"id":"thread-2","status":"completed"}]}`)})
		}
	}

	oldFactory := newThreadListClient
	newThreadListClient = func(ctx context.Context, s *threadListSettings) (*codexrpc.Client, error) {
		c := codexrpc.NewClient(mem)
		if err := c.Connect(ctx, map[string]any{"clientInfo": map[string]any{"name": "test"}}); err != nil {
			return nil, err
		}
		return c, nil
	}
	defer func() {
		newThreadListClient = oldFactory
	}()

	root, err := newRootCommand()
	if err != nil {
		t.Fatalf("newRootCommand() error = %v", err)
	}
	root.SetArgs([]string{"thread", "list", "--limit", "2"})

	out, err := captureStdout(func() error {
		return root.Execute()
	})
	if err != nil {
		t.Fatalf("root.Execute() error = %v", err)
	}

	if !strings.Contains(out, "thread-1") || !strings.Contains(out, "thread-2") {
		t.Fatalf("expected command output to contain thread ids, got:\n%s", out)
	}

	sent := mem.SentSnapshot()
	if len(sent) < 3 {
		t.Fatalf("expected at least three outbound messages (initialize, initialized, thread/list), got %d", len(sent))
	}
	if sent[0].Method != "initialize" || sent[1].Method != "initialized" || sent[2].Method != "thread/list" {
		t.Fatalf("unexpected outbound method sequence: %q, %q, %q", sent[0].Method, sent[1].Method, sent[2].Method)
	}
}

func captureStdout(fn func() error) (string, error) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w
	runErr := fn()
	_ = w.Close()
	os.Stdout = oldStdout

	b, readErr := io.ReadAll(r)
	_ = r.Close()
	if readErr != nil {
		return "", readErr
	}
	return string(b), runErr
}
