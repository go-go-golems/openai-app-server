package state

import "testing"

func TestStoreBounds(t *testing.T) {
	s := NewStore(Config{
		MaxThreads:        2,
		MaxTurnsPerThread: 1,
		MaxItemsPerTurn:   1,
	})

	s.UpsertThread(ThreadState{ID: "thread-1"})
	s.UpsertThread(ThreadState{ID: "thread-2"})
	s.UpsertThread(ThreadState{ID: "thread-3"})

	threads := s.ListThreads()
	if len(threads) != 2 {
		t.Fatalf("expected 2 threads after eviction, got %d", len(threads))
	}
	if threads[0].ID != "thread-2" || threads[1].ID != "thread-3" {
		t.Fatalf("unexpected thread order after eviction: %#v", threads)
	}

	s.UpsertTurn("thread-3", TurnState{ID: "turn-1", Status: "inProgress"})
	s.UpsertTurn("thread-3", TurnState{ID: "turn-2", Status: "completed"})

	thread3, ok := s.Thread("thread-3")
	if !ok {
		t.Fatalf("expected thread-3 to exist")
	}
	if len(thread3.Turns) != 1 || thread3.Turns[0].ID != "turn-2" {
		t.Fatalf("unexpected turns after eviction: %#v", thread3.Turns)
	}

	s.UpsertItem("thread-3", "turn-2", ItemState{ID: "item-1", Type: "agentMessage"})
	s.UpsertItem("thread-3", "turn-2", ItemState{ID: "item-2", Type: "commandExecution"})

	turn2, ok := s.Turn("thread-3", "turn-2")
	if !ok {
		t.Fatalf("expected turn-2 to exist")
	}
	if len(turn2.Items) != 1 || turn2.Items[0].ID != "item-2" {
		t.Fatalf("unexpected items after eviction: %#v", turn2.Items)
	}
}

func TestStoreSnapshotIsolation(t *testing.T) {
	s := NewStore(Config{})
	s.UpsertThread(ThreadState{
		ID: "thread-1",
		TokenUsage: map[string]any{
			"total": map[string]any{"inputTokens": float64(10)},
		},
	})
	s.UpsertTurn("thread-1", TurnState{ID: "turn-1", LatestPlan: map[string]any{"steps": []any{"a"}}})

	thread1, ok := s.Thread("thread-1")
	if !ok {
		t.Fatalf("expected thread-1")
	}

	total := thread1.TokenUsage["total"].(map[string]any)
	total["inputTokens"] = float64(999)

	turn1 := thread1.Turns[0]
	plan := turn1.LatestPlan.(map[string]any)
	steps := plan["steps"].([]any)
	steps[0] = "mutated"

	threadAgain, ok := s.Thread("thread-1")
	if !ok {
		t.Fatalf("expected thread-1 on second read")
	}

	totalAgain := threadAgain.TokenUsage["total"].(map[string]any)
	if got := totalAgain["inputTokens"].(float64); got != 10 {
		t.Fatalf("expected tokenUsage isolation, got %v", got)
	}
	planAgain := threadAgain.Turns[0].LatestPlan.(map[string]any)
	stepsAgain := planAgain["steps"].([]any)
	if got := stepsAgain[0].(string); got != "a" {
		t.Fatalf("expected latestPlan isolation, got %q", got)
	}
}
