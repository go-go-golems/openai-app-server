package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHarnessStateReplayCommand(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatalf("newRootCommand() error = %v", err)
	}

	tmpDir := t.TempDir()
	eventsPath := filepath.Join(tmpDir, "events.json")
	events := `[
  {"method":"thread/started","params":{"thread":{"id":"thread-1","status":"active","cwd":"/repo","model":"gpt-5"}}},
  {"method":"turn/started","params":{"threadId":"thread-1","turn":{"id":"turn-1","status":"inProgress"}}},
  {"method":"turn/diff/updated","params":{"threadId":"thread-1","turnId":"turn-1","diff":"--- a\\n+++ b"}},
  {"method":"item/completed","params":{"threadId":"thread-1","turnId":"turn-1","item":{"id":"item-1","type":"agentMessage"}}}
]`
	if err := os.WriteFile(eventsPath, []byte(events), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	root.SetArgs([]string{"harness", "state-replay", "--events-file", eventsPath})
	out, err := captureStdout(func() error { return root.Execute() })
	if err != nil {
		t.Fatalf("root.Execute() error = %v", err)
	}

	if !strings.Contains(out, "thread-1") || !strings.Contains(out, "turn-1") || !strings.Contains(out, "--- a") {
		t.Fatalf("unexpected state replay output:\n%s", out)
	}
}
