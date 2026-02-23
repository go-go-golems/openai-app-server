package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/go-go-golems/openai-app-server/pkg/codexrpc"
)

func TestThreadReadCommandWithMemoryTransport(t *testing.T) {
	mem := codexrpc.NewMemoryTransport()
	mem.OnSend = func(msg *codexrpc.Message) {
		switch msg.Method {
		case "initialize":
			mem.Push(&codexrpc.Message{ID: msg.ID, Result: []byte(`{"capabilities":{}}`)})
		case "thread/read":
			mem.Push(&codexrpc.Message{ID: msg.ID, Result: []byte(`{"thread":{"id":"thread-123","status":"active","cwd":"/tmp/work","turns":[{"id":"turn-1"}]}}`)})
		}
	}

	oldFactory := newThreadReadClient
	newThreadReadClient = func(ctx context.Context, s *threadReadSettings) (*codexrpc.Client, error) {
		c := codexrpc.NewClient(mem)
		if err := c.Connect(ctx, map[string]any{"clientInfo": map[string]any{"name": "test"}}); err != nil {
			return nil, err
		}
		return c, nil
	}
	defer func() {
		newThreadReadClient = oldFactory
	}()

	root, err := newRootCommand()
	if err != nil {
		t.Fatalf("newRootCommand() error = %v", err)
	}
	root.SetArgs([]string{"thread", "read", "--thread-id", "thread-123", "--include-turns"})

	out, err := captureStdout(func() error {
		return root.Execute()
	})
	if err != nil {
		t.Fatalf("root.Execute() error = %v", err)
	}

	if !strings.Contains(out, "thread-123") || !strings.Contains(out, "turn-1") {
		t.Fatalf("expected output to contain thread and turn ids, got:\n%s", out)
	}

	sent := mem.SentSnapshot()
	if len(sent) < 3 {
		t.Fatalf("expected at least three outbound messages (initialize, initialized, thread/read), got %d", len(sent))
	}
	if sent[0].Method != "initialize" || sent[1].Method != "initialized" || sent[2].Method != "thread/read" {
		t.Fatalf("unexpected outbound method sequence: %q, %q, %q", sent[0].Method, sent[1].Method, sent[2].Method)
	}

	params := map[string]any{}
	if err := json.Unmarshal(sent[2].Params, &params); err != nil {
		t.Fatalf("unmarshal thread/read params: %v", err)
	}
	if params["threadId"] != "thread-123" {
		t.Fatalf("expected threadId=thread-123, got %#v", params["threadId"])
	}
	if includeTurns, ok := params["includeTurns"].(bool); !ok || !includeTurns {
		t.Fatalf("expected includeTurns=true, got %#v", params["includeTurns"])
	}
}
