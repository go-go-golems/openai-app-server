package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-go-golems/openai-app-server/pkg/codexrpc"
)

func TestHarnessRunCommandWithMemoryTransport(t *testing.T) {
	tmp := t.TempDir()
	scriptPath := filepath.Join(tmp, "harness.js")
	script := `
const codex = require("codex");
const session = codex.connect();
session.request("thread/list", { limit: 1 }).then((r) => {
  const threads = r.threads || [];
  __host.ui.emit({ type: "threads", count: threads.length });
});
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	mem := codexrpc.NewMemoryTransport()
	mem.OnSend = func(msg *codexrpc.Message) {
		switch msg.Method {
		case "initialize":
			mem.Push(&codexrpc.Message{ID: msg.ID, Result: []byte(`{"capabilities":{}}`)})
		case "thread/list":
			mem.Push(&codexrpc.Message{ID: msg.ID, Result: []byte(`{"threads":[{"id":"thread-1"}]}`)})
		}
	}

	oldClientFactory := newHarnessRunClient
	newHarnessRunClient = func(ctx context.Context, _ *harnessRunSettings) (*codexrpc.Client, error) {
		c := codexrpc.NewClient(mem)
		if err := c.Connect(ctx, map[string]any{"clientInfo": map[string]any{"name": "test"}}); err != nil {
			return nil, err
		}
		return c, nil
	}
	defer func() {
		newHarnessRunClient = oldClientFactory
	}()

	root, err := newRootCommand()
	if err != nil {
		t.Fatalf("newRootCommand() error = %v", err)
	}
	root.SetArgs([]string{"harness", "run", "--script", scriptPath, "--transport", "stdio", "--settle-ms", "300", "--timeout-ms", "3000"})

	out, err := captureStdout(func() error {
		return root.Execute()
	})
	if err != nil {
		t.Fatalf("root.Execute() error = %v", err)
	}

	if !strings.Contains(out, `"type":"threads"`) {
		t.Fatalf("expected ui.emit threads payload in output, got:\n%s", out)
	}

	sent := mem.SentSnapshot()
	if len(sent) < 3 {
		t.Fatalf("expected at least three outbound messages, got %d", len(sent))
	}
	if sent[0].Method != "initialize" || sent[1].Method != "initialized" || sent[2].Method != "thread/list" {
		t.Fatalf("unexpected outbound method sequence: %q, %q, %q", sent[0].Method, sent[1].Method, sent[2].Method)
	}
}
