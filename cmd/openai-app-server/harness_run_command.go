package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/openai-app-server/pkg/config"
)

type harnessRunCommand struct {
	*cmds.CommandDescription
}

var _ cmds.BareCommand = (*harnessRunCommand)(nil)

type harnessRunSettings struct {
	ScriptPath     string `glazed:"script"`
	Transport      string `glazed:"transport"`
	StdioCommand   string `glazed:"stdio-command"`
	StdioArgs      string `glazed:"stdio-args"`
	Model          string `glazed:"model"`
	Cwd            string `glazed:"cwd"`
	ApprovalPolicy string `glazed:"approval-policy"`
	SandboxPolicy  string `glazed:"sandbox-policy"`
	DryRun         bool   `glazed:"dry-run"`
}

func newHarnessRunCommand(defaults config.Defaults) (*harnessRunCommand, error) {
	desc := cmds.NewCommandDescription(
		"run",
		cmds.WithShort("Run a harness script (skeleton)"),
		cmds.WithLong("Phase-1 skeleton for harness execution wiring. Transport/session behavior is implemented in later phases."),
		cmds.WithFlags(
			fields.New("script", fields.TypeString, fields.WithHelp("Path to harness script")),
			fields.New("transport", fields.TypeString, fields.WithDefault(defaults.Transport), fields.WithHelp("Transport backend: stdio|websocket")),
			fields.New("stdio-command", fields.TypeString, fields.WithDefault(defaults.StdioCommand), fields.WithHelp("Stdio command for app-server process")),
			fields.New("stdio-args", fields.TypeString, fields.WithDefault(strings.Join(defaults.StdioArgs, " ")), fields.WithHelp("Stdio command arguments (space-separated)")),
			fields.New("model", fields.TypeString, fields.WithDefault(defaults.Model), fields.WithHelp("Default model for new thread sessions")),
			fields.New("cwd", fields.TypeString, fields.WithDefault(defaults.Cwd), fields.WithHelp("Working directory for thread sessions")),
			fields.New("approval-policy", fields.TypeString, fields.WithDefault(defaults.ApprovalPolicy), fields.WithHelp("Approval policy (phase-1 placeholder)")),
			fields.New("sandbox-policy", fields.TypeString, fields.WithDefault(defaults.SandboxPolicy), fields.WithHelp("Sandbox policy (phase-1 placeholder)")),
			fields.New("dry-run", fields.TypeBool, fields.WithDefault(true), fields.WithHelp("Print resolved settings without launching harness")),
		),
	)

	return &harnessRunCommand{CommandDescription: desc}, nil
}

func (c *harnessRunCommand) Run(_ context.Context, vals *values.Values) error {
	s := &harnessRunSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}

	if strings.TrimSpace(s.ScriptPath) == "" {
		return fmt.Errorf("--script is required")
	}

	fmt.Printf("phase-1 harness.run skeleton\n")
	fmt.Printf("script=%s\n", s.ScriptPath)
	fmt.Printf("transport=%s\n", s.Transport)
	fmt.Printf("stdio.command=%s\n", s.StdioCommand)
	fmt.Printf("stdio.args=%s\n", s.StdioArgs)
	fmt.Printf("model=%s\n", s.Model)
	fmt.Printf("cwd=%s\n", s.Cwd)
	fmt.Printf("approval-policy=%s\n", s.ApprovalPolicy)
	fmt.Printf("sandbox-policy=%s\n", s.SandboxPolicy)
	fmt.Printf("dry-run=%t\n", s.DryRun)
	return nil
}
