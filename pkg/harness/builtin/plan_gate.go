package builtin

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-go-golems/openai-app-server/pkg/harness"
)

type PlanGateController interface {
	SteerTurn(ctx context.Context, threadID string, turnID string, instruction string) error
	InterruptTurn(ctx context.Context, threadID string, turnID string) error
}

type PlanGateConfig struct {
	Controller       PlanGateController
	SteerInstruction string
	Approve          func(threadID string, turnID string, plan any) (bool, error)
}

type planGateState struct {
	mu   sync.Mutex
	seen map[string]bool
}

func NewPlanGateHarness(cfg PlanGateConfig) harness.Harness {
	if cfg.SteerInstruction == "" {
		cfg.SteerInstruction = "Plan approved. Continue with implementation."
	}

	state := &planGateState{seen: map[string]bool{}}

	return harness.Harness{
		Name: "plan-gate",
		Notifications: []harness.NotificationBinding{
			{
				Method: "turn/plan/updated",
				Handler: func(ctx context.Context, evt harness.NotificationEvent) error {
					threadID, _ := evt.Params["threadId"].(string)
					turnID, _ := evt.Params["turnId"].(string)
					if threadID == "" || turnID == "" {
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

					approved := true
					if cfg.Approve != nil {
						ok, err := cfg.Approve(threadID, turnID, evt.Params["plan"])
						if err != nil {
							return err
						}
						approved = ok
					}

					if cfg.Controller == nil {
						return nil
					}
					if approved {
						return cfg.Controller.SteerTurn(ctx, threadID, turnID, cfg.SteerInstruction)
					}
					return cfg.Controller.InterruptTurn(ctx, threadID, turnID)
				},
			},
		},
	}
}
