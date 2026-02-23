package config

import "os"

// Defaults centralizes phase-1 command defaults for transport/session flags.
type Defaults struct {
	Transport      string
	StdioCommand   string
	StdioArgs      []string
	Model          string
	Cwd            string
	ApprovalPolicy string
	SandboxPolicy  string
}

func NewDefaults() (Defaults, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Defaults{}, err
	}

	return Defaults{
		Transport:      "stdio",
		StdioCommand:   "codex-app-server",
		StdioArgs:      []string{},
		Model:          "gpt-5",
		Cwd:            cwd,
		ApprovalPolicy: "on-request",
		SandboxPolicy:  "workspace-write",
	}, nil
}
