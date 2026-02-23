package builtin

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-go-golems/openai-app-server/pkg/harness"
)

type TDDTestResult struct {
	Passed bool
	Output string
}

type TDDLoopController interface {
	RunTests(ctx context.Context, threadID string) (TDDTestResult, error)
	StartFollowupTurn(ctx context.Context, threadID string, input string) error
}

type TDDLoopConfig struct {
	Controller       TDDLoopController
	MaxIterations    int
	FollowupTemplate string
}

type tddLoopState struct {
	mu         sync.Mutex
	iterations map[string]int
}

func NewTDDLoopHarness(cfg TDDLoopConfig) harness.Harness {
	if cfg.MaxIterations <= 0 {
		cfg.MaxIterations = 3
	}
	if cfg.FollowupTemplate == "" {
		cfg.FollowupTemplate = "Tests failed. Please fix the failures and rerun tests.\n\nTest output:\n%s"
	}

	state := &tddLoopState{iterations: map[string]int{}}

	return harness.Harness{
		Name: "tdd-loop",
		Notifications: []harness.NotificationBinding{
			{
				Method: "turn/completed",
				Handler: func(ctx context.Context, evt harness.NotificationEvent) error {
					if cfg.Controller == nil {
						return nil
					}

					threadID, _ := evt.Params["threadId"].(string)
					if threadID == "" {
						return nil
					}
					status := extractTurnStatus(evt.Params)
					if status != "completed" {
						return nil
					}

					state.mu.Lock()
					state.iterations[threadID]++
					iter := state.iterations[threadID]
					state.mu.Unlock()

					if iter > cfg.MaxIterations {
						return nil
					}

					result, err := cfg.Controller.RunTests(ctx, threadID)
					if err != nil {
						return err
					}
					if result.Passed {
						return nil
					}

					prompt := fmt.Sprintf(cfg.FollowupTemplate, result.Output)
					return cfg.Controller.StartFollowupTurn(ctx, threadID, prompt)
				},
			},
		},
	}
}

func extractTurnStatus(params map[string]any) string {
	if turn, ok := params["turn"].(map[string]any); ok {
		if status, ok := turn["status"].(string); ok {
			return status
		}
	}
	if status, ok := params["status"].(string); ok {
		return status
	}
	return ""
}
