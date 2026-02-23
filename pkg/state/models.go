package state

// ThreadState is the projected in-memory view of a thread and its turns.
type ThreadState struct {
	ID         string         `json:"id"`
	Status     string         `json:"status,omitempty"`
	Cwd        string         `json:"cwd,omitempty"`
	Model      string         `json:"model,omitempty"`
	TokenUsage map[string]any `json:"tokenUsage,omitempty"`
	Turns      []*TurnState   `json:"turns,omitempty"`
}

// TurnState is the projected in-memory view of one turn.
type TurnState struct {
	ID         string       `json:"id"`
	ThreadID   string       `json:"threadId"`
	Status     string       `json:"status,omitempty"`
	Error      any          `json:"error,omitempty"`
	LatestDiff string       `json:"latestDiff,omitempty"`
	LatestPlan any          `json:"latestPlan,omitempty"`
	Items      []*ItemState `json:"items,omitempty"`
}

// ItemState is the projected in-memory view of one item.
type ItemState struct {
	ID       string         `json:"id"`
	ThreadID string         `json:"threadId"`
	TurnID   string         `json:"turnId"`
	Type     string         `json:"type,omitempty"`
	Status   string         `json:"status,omitempty"`
	Raw      map[string]any `json:"raw,omitempty"`
}
