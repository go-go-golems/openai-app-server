package builtin

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-go-golems/openai-app-server/pkg/harness"
)

type ReviewResult struct {
	NeedsFollowup bool
	Summary       string
}

type ReviewGateController interface {
	StartReview(ctx context.Context, threadID string, turnID string) (ReviewResult, error)
	StartFollowupTurn(ctx context.Context, threadID string, input string) error
}

type ReviewGateConfig struct {
	Controller       ReviewGateController
	FollowupTemplate string
}

type reviewGateState struct {
	mu   sync.Mutex
	seen map[string]bool
}

func NewReviewGateHarness(cfg ReviewGateConfig) harness.Harness {
	if cfg.FollowupTemplate == "" {
		cfg.FollowupTemplate = "Please address the following review feedback before continuing:\n\n%s"
	}

	state := &reviewGateState{seen: map[string]bool{}}

	return harness.Harness{
		Name: "review-gate",
		Notifications: []harness.NotificationBinding{
			{
				Method: "turn/completed",
				Handler: func(ctx context.Context, evt harness.NotificationEvent) error {
					if cfg.Controller == nil {
						return nil
					}

					threadID, _ := evt.Params["threadId"].(string)
					turnID := extractTurnID(evt.Params)
					if threadID == "" || turnID == "" {
						return nil
					}
					if extractTurnStatus(evt.Params) != "completed" {
						return nil
					}

					key := fmt.Sprintf("%s:%s", threadID, turnID)
					state.mu.Lock()
					if state.seen[key] {
						state.mu.Unlock()
						return nil
					}
					state.seen[key] = true
					state.mu.Unlock()

					review, err := cfg.Controller.StartReview(ctx, threadID, turnID)
					if err != nil {
						return err
					}
					if !review.NeedsFollowup {
						return nil
					}
					prompt := fmt.Sprintf(cfg.FollowupTemplate, review.Summary)
					return cfg.Controller.StartFollowupTurn(ctx, threadID, prompt)
				},
			},
		},
	}
}

func extractTurnID(params map[string]any) string {
	if turn, ok := params["turn"].(map[string]any); ok {
		if id, ok := turn["id"].(string); ok {
			return id
		}
	}
	if turnID, ok := params["turnId"].(string); ok {
		return turnID
	}
	return ""
}
