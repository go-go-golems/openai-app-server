package config

import (
	"os"
	"strings"
)

// Defaults centralizes phase-1 command defaults for transport/session flags.
type Defaults struct {
	Transport                 string
	StdioCommand              string
	StdioArgs                 []string
	OptOutNotificationMethods []string
	Model                     string
	Cwd                       string
	ApprovalPolicy            string
	SandboxPolicy             string
}

func NewDefaults() (Defaults, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Defaults{}, err
	}

	return Defaults{
		Transport:                 "stdio",
		StdioCommand:              "codex-app-server",
		StdioArgs:                 []string{},
		OptOutNotificationMethods: splitCSV(os.Getenv("OAP_OPT_OUT_NOTIFICATION_METHODS")),
		Model:                     "gpt-5",
		Cwd:                       cwd,
		ApprovalPolicy:            "on-request",
		SandboxPolicy:             "workspace-write",
	}, nil
}

func splitCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
