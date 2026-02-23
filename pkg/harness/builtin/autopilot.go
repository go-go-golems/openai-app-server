package builtin

import (
	"strings"

	"github.com/go-go-golems/openai-app-server/pkg/harness"
)

type AutopilotConfig struct {
	SafeCommandPrefixes []string
	DeniedPathPrefixes  []string
	MaxChangedLines     int
}

func NewAutopilotHarness(cfg AutopilotConfig) harness.Harness {
	normalized := normalizeAutopilotConfig(cfg)
	return harness.Harness{
		Name: "autopilot-approvals",
		Requests: []harness.RequestBinding{
			{
				Method: "item/commandExecution/requestApproval",
				Handler: func(rc *harness.RequestContext) error {
					req := rc.Request()
					decision := commandDecision(req.Params, normalized.SafeCommandPrefixes)
					return rc.Respond(map[string]any{"decision": decision})
				},
			},
			{
				Method: "item/fileChange/requestApproval",
				Handler: func(rc *harness.RequestContext) error {
					req := rc.Request()
					decision := fileChangeDecision(req.Params, normalized.DeniedPathPrefixes, normalized.MaxChangedLines)
					return rc.Respond(map[string]any{"decision": decision})
				},
			},
		},
	}
}

func normalizeAutopilotConfig(cfg AutopilotConfig) AutopilotConfig {
	if len(cfg.SafeCommandPrefixes) == 0 {
		cfg.SafeCommandPrefixes = []string{
			"go test",
			"npm test",
			"pnpm test",
			"yarn test",
			"pytest",
			"cargo test",
			"make test",
		}
	}
	if cfg.MaxChangedLines <= 0 {
		cfg.MaxChangedLines = 400
	}
	return cfg
}

func commandDecision(params map[string]any, safePrefixes []string) string {
	command := extractCommand(params)
	for _, prefix := range safePrefixes {
		if strings.HasPrefix(command, strings.TrimSpace(prefix)) {
			return "acceptForSession"
		}
	}
	return "decline"
}

func fileChangeDecision(params map[string]any, deniedPathPrefixes []string, maxChangedLines int) string {
	changes, ok := params["changes"].([]any)
	if !ok || len(changes) == 0 {
		return "acceptForSession"
	}

	totalLines := 0
	for _, raw := range changes {
		change, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		path, _ := change["path"].(string)
		for _, prefix := range deniedPathPrefixes {
			if strings.HasPrefix(path, prefix) {
				return "decline"
			}
		}
		totalLines += intFromAny(change["added"]) + intFromAny(change["removed"])
	}

	if totalLines > maxChangedLines {
		return "decline"
	}
	return "acceptForSession"
}

func extractCommand(params map[string]any) string {
	if command, ok := params["command"].(string); ok && strings.TrimSpace(command) != "" {
		return command
	}
	if actions, ok := params["commandActions"].([]any); ok {
		for _, raw := range actions {
			action, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if command, ok := action["command"].(string); ok && strings.TrimSpace(command) != "" {
				return command
			}
		}
	}
	return ""
}

func intFromAny(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float64:
		return int(n)
	case float32:
		return int(n)
	default:
		return 0
	}
}
