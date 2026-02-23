package codexrpc

import (
	"context"
	"encoding/json"
	"fmt"
)

type ThreadSummary struct {
	ID     string `json:"id"`
	Status string `json:"status,omitempty"`
	Cwd    string `json:"cwd,omitempty"`
	Model  string `json:"model,omitempty"`
}

type Thread struct {
	ID            string `json:"id"`
	Status        string `json:"status,omitempty"`
	Cwd           string `json:"cwd,omitempty"`
	Model         string `json:"model,omitempty"`
	CliVersion    string `json:"cliVersion,omitempty"`
	ModelProvider string `json:"modelProvider,omitempty"`
	Path          string `json:"path,omitempty"`
	Preview       string `json:"preview,omitempty"`
	Source        string `json:"source,omitempty"`
	CreatedAt     int64  `json:"createdAt,omitempty"`
	UpdatedAt     int64  `json:"updatedAt,omitempty"`
	GitInfo       any    `json:"gitInfo,omitempty"`
	Turns         []any  `json:"turns,omitempty"`
}

type threadListResult struct {
	Threads []ThreadSummary `json:"threads"`
}

type threadReadResult struct {
	Thread *Thread `json:"thread"`
}

func (c *Client) ThreadList(ctx context.Context, limit int) ([]ThreadSummary, error) {
	params := map[string]any{"limit": limit}
	resp, err := c.Request(ctx, "thread/list", params)
	if err != nil {
		return nil, err
	}
	if len(resp.Result) == 0 {
		return []ThreadSummary{}, nil
	}

	var wrapped threadListResult
	if err := json.Unmarshal(resp.Result, &wrapped); err == nil && wrapped.Threads != nil {
		return wrapped.Threads, nil
	}

	var flat []ThreadSummary
	if err := json.Unmarshal(resp.Result, &flat); err == nil {
		return flat, nil
	}

	return nil, fmt.Errorf("codexrpc: unexpected thread/list result payload")
}

func (c *Client) ThreadRead(ctx context.Context, threadID string, includeTurns bool) (*Thread, error) {
	params := map[string]any{
		"threadId":     threadID,
		"includeTurns": includeTurns,
	}
	resp, err := c.Request(ctx, "thread/read", params)
	if err != nil {
		return nil, err
	}
	if len(resp.Result) == 0 {
		return nil, fmt.Errorf("codexrpc: empty thread/read result payload")
	}

	var wrapped threadReadResult
	if err := json.Unmarshal(resp.Result, &wrapped); err == nil && wrapped.Thread != nil {
		return wrapped.Thread, nil
	}

	var direct Thread
	if err := json.Unmarshal(resp.Result, &direct); err == nil {
		if direct.ID != "" || direct.Path != "" || direct.Cwd != "" || len(direct.Turns) > 0 {
			return &direct, nil
		}
	}

	return nil, fmt.Errorf("codexrpc: unexpected thread/read result payload")
}
