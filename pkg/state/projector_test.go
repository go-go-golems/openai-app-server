package state

import "testing"

func TestProjectorReplayAppliesThreadTurnItemDiffPlan(t *testing.T) {
	store := NewStore(Config{})
	projector := NewProjector(store)

	projector.Apply("thread/started", map[string]any{
		"thread": map[string]any{
			"id":     "thread-1",
			"status": "active",
			"cwd":    "/repo",
			"model":  "gpt-5",
		},
	})
	projector.Apply("turn/started", map[string]any{
		"threadId": "thread-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "inProgress",
		},
	})
	projector.Apply("item/started", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-1",
		"item": map[string]any{
			"id":   "item-1",
			"type": "agentMessage",
		},
	})
	projector.Apply("turn/diff/updated", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-1",
		"diff":     "--- a\n+++ b",
	})
	projector.Apply("turn/plan/updated", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-1",
		"plan":     map[string]any{"steps": []any{"s1", "s2"}},
	})
	projector.Apply("item/completed", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-1",
		"item": map[string]any{
			"id":   "item-1",
			"type": "agentMessage",
		},
	})
	projector.Apply("thread/tokenUsage/updated", map[string]any{
		"threadId": "thread-1",
		"tokenUsage": map[string]any{
			"total": map[string]any{"inputTokens": float64(123)},
		},
	})

	thread, ok := store.Thread("thread-1")
	if !ok {
		t.Fatalf("expected thread-1 to exist")
	}
	if thread.Status != "active" || thread.Cwd != "/repo" || thread.Model != "gpt-5" {
		t.Fatalf("unexpected thread projection: %#v", thread)
	}
	if len(thread.Turns) != 1 || thread.Turns[0].ID != "turn-1" {
		t.Fatalf("unexpected turn projection: %#v", thread.Turns)
	}

	turn, ok := store.Turn("thread-1", "turn-1")
	if !ok {
		t.Fatalf("expected turn-1 to exist")
	}
	if turn.Status != "inProgress" {
		t.Fatalf("unexpected turn status: %s", turn.Status)
	}
	if turn.LatestDiff != "--- a\n+++ b" {
		t.Fatalf("unexpected latest diff: %q", turn.LatestDiff)
	}
	if turn.LatestPlan == nil {
		t.Fatalf("expected latest plan to be set")
	}
	if len(turn.Items) != 1 {
		t.Fatalf("expected one item, got %d", len(turn.Items))
	}
	if turn.Items[0].Status != "completed" {
		t.Fatalf("expected completed item status, got %q", turn.Items[0].Status)
	}

	if thread.TokenUsage == nil {
		t.Fatalf("expected token usage to be projected")
	}
	total := thread.TokenUsage["total"].(map[string]any)
	if got := total["inputTokens"].(float64); got != 123 {
		t.Fatalf("unexpected token usage: %v", got)
	}
}

func TestProjectorIgnoresMalformedEvents(t *testing.T) {
	store := NewStore(Config{})
	projector := NewProjector(store)

	projector.Apply("thread/started", map[string]any{"thread": map[string]any{}})
	projector.Apply("turn/started", map[string]any{"threadId": "thread-1"})
	projector.Apply("item/started", map[string]any{"threadId": "thread-1", "turnId": "turn-1"})
	projector.Apply("unknown/method", map[string]any{"x": "y"})

	threads := store.ListThreads()
	if len(threads) != 0 {
		t.Fatalf("expected malformed events to be ignored, got %#v", threads)
	}
}
