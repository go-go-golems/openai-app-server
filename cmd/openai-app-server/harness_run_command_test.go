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
const ui = require("ui");
const session = codex.connect();
session.request("thread/list", { limit: 1 }).then((r) => {
  const threads = r.threads || [];
  ui.emit({ type: "threads", count: threads.length });
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

func TestHarnessRunForwardsServerNotificationsToJS(t *testing.T) {
	tmp := t.TempDir()
	scriptPath := filepath.Join(tmp, "harness-notif.js")
	script := `
const codex = require("codex");
const ui = require("ui");
const session = codex.connect();
session.onNotification((evt) => {
  if (evt.method === "thread/started") {
    ui.emit({ type: "notif-forward", ok: true, id: evt.params.id });
  }
});
session.request("thread/list", { limit: 1 }).then(() => {});
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
			mem.Push(&codexrpc.Message{Method: "thread/started", Params: []byte(`{"id":"thread-1"}`)})
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
	root.SetArgs([]string{"harness", "run", "--script", scriptPath, "--transport", "stdio", "--settle-ms", "400", "--timeout-ms", "3000"})

	out, err := captureStdout(func() error {
		return root.Execute()
	})
	if err != nil {
		t.Fatalf("root.Execute() error = %v", err)
	}

	if !strings.Contains(out, `"type":"notif-forward"`) {
		t.Fatalf("expected notification-forward ui payload in output, got:\n%s", out)
	}
}

func TestHarnessRunWaitForUIType(t *testing.T) {
	tmp := t.TempDir()
	scriptPath := filepath.Join(tmp, "harness-wait.js")
	script := `
const codex = require("codex");
const ui = require("ui");
const session = codex.connect();
session.request("thread/list", { limit: 1 }).then((r) => {
  ui.emit({ type: "done", ok: true, count: (r.threads || []).length });
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
	root.SetArgs([]string{
		"harness", "run",
		"--script", scriptPath,
		"--transport", "stdio",
		"--settle-ms", "0",
		"--timeout-ms", "3000",
		"--wait-for-ui-type", "done",
		"--wait-for-ui-timeout-ms", "1000",
	})

	out, err := captureStdout(func() error {
		return root.Execute()
	})
	if err != nil {
		t.Fatalf("root.Execute() error = %v", err)
	}
	if !strings.Contains(out, "wait-for-ui-type matched type=done") {
		t.Fatalf("expected wait-for-ui-type match log in output, got:\n%s", out)
	}
}

func TestHarnessRunHandlesServerRequestAndResponds(t *testing.T) {
	tmp := t.TempDir()
	scriptPath := filepath.Join(tmp, "harness-respond.js")
	script := `
const codex = require("codex");
const rpc = require("rpc");
const ui = require("ui");
const session = codex.connect();
session.onRequest((evt) => {
  if (evt.method === "approval/request") {
    rpc.respond(evt.id, { accepted: true });
    ui.emit({ type: "request-handled", id: evt.id });
  }
});
session.request("thread/list", { limit: 1 }).then(() => {});
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
			mem.Push(&codexrpc.Message{ID: "req-1", Method: "approval/request", Params: []byte(`{"threadId":"thread-1"}`)})
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
	root.SetArgs([]string{"harness", "run", "--script", scriptPath, "--transport", "stdio", "--settle-ms", "400", "--timeout-ms", "3000"})

	out, err := captureStdout(func() error {
		return root.Execute()
	})
	if err != nil {
		t.Fatalf("root.Execute() error = %v", err)
	}
	if !strings.Contains(out, `"type":"request-handled"`) {
		t.Fatalf("expected request-handled event in output, got:\n%s", out)
	}

	sent := mem.SentSnapshot()
	foundResponse := false
	for _, msg := range sent {
		if msg != nil && msg.ID == "req-1" && msg.Method == "" && len(msg.Result) > 0 {
			foundResponse = true
			break
		}
	}
	if !foundResponse {
		t.Fatalf("expected response message for req-1 in outbound transport messages")
	}
}
