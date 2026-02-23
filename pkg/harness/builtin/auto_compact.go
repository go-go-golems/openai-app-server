package builtin

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-go-golems/openai-app-server/pkg/harness"
)

type AutoCompactController interface {
	StartCompaction(ctx context.Context, threadID string) error
}

type AutoCompactConfig struct {
	Controller      AutoCompactController
	SoftLimitRatio  float64
	ResetBelowRatio float64
}

type autoCompactState struct {
	mu        sync.Mutex
	requested map[string]bool
}

func NewAutoCompactHarness(cfg AutoCompactConfig) harness.Harness {
	if cfg.SoftLimitRatio <= 0 {
		cfg.SoftLimitRatio = 0.75
	}
	if cfg.ResetBelowRatio <= 0 || cfg.ResetBelowRatio >= cfg.SoftLimitRatio {
		cfg.ResetBelowRatio = cfg.SoftLimitRatio * 0.8
	}

	state := &autoCompactState{requested: map[string]bool{}}

	return harness.Harness{
		Name: "auto-compact",
		Notifications: []harness.NotificationBinding{
			{
				Method: "thread/tokenUsage/updated",
				Handler: func(ctx context.Context, evt harness.NotificationEvent) error {
					if cfg.Controller == nil {
						return nil
					}
					threadID, _ := evt.Params["threadId"].(string)
					if threadID == "" {
						return nil
					}

					ratio, ok := tokenUsageRatio(evt.Params)
					if !ok {
						return nil
					}

					state.mu.Lock()
					defer state.mu.Unlock()

					if ratio < cfg.ResetBelowRatio {
						state.requested[threadID] = false
						return nil
					}

					if ratio >= cfg.SoftLimitRatio && !state.requested[threadID] {
						if err := cfg.Controller.StartCompaction(ctx, threadID); err != nil {
							return err
						}
						state.requested[threadID] = true
					}
					return nil
				},
			},
		},
	}
}

func tokenUsageRatio(params map[string]any) (float64, bool) {
	tokenUsage, ok := params["tokenUsage"].(map[string]any)
	if !ok {
		return 0, false
	}
	modelContextWindow, ok := asFloat64(tokenUsage["modelContextWindow"])
	if !ok || modelContextWindow <= 0 {
		return 0, false
	}
	total, ok := tokenUsage["total"].(map[string]any)
	if !ok {
		return 0, false
	}
	totalTokens, ok := asFloat64(total["totalTokens"])
	if !ok || totalTokens < 0 {
		return 0, false
	}
	return totalTokens / modelContextWindow, true
}

func asFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

func formatRatio(ratio float64) string {
	return fmt.Sprintf("%.4f", ratio)
}
