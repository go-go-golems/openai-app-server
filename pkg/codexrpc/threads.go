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

type threadListResult struct {
	Threads []ThreadSummary `json:"threads"`
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
