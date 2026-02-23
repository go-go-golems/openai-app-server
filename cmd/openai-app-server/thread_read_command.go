package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/openai-app-server/pkg/config"
)

type threadReadCommand struct {
	*cmds.CommandDescription
}

var _ cmds.BareCommand = (*threadReadCommand)(nil)

type threadReadSettings struct {
	Transport    string `glazed:"transport"`
	StdioCommand string `glazed:"stdio-command"`
	StdioArgs    string `glazed:"stdio-args"`
	ThreadID     string `glazed:"thread-id"`
	IncludeTurns bool   `glazed:"include-turns"`
}

func newThreadReadCommand(defaults config.Defaults) (*threadReadCommand, error) {
	desc := cmds.NewCommandDescription(
		"read",
		cmds.WithShort("Read thread details (skeleton)"),
		cmds.WithLong("Phase-3 skeleton for thread/read output shape. Full RPC wiring will follow after thread/list stabilization."),
		cmds.WithFlags(
			fields.New("transport", fields.TypeString, fields.WithDefault(defaults.Transport), fields.WithHelp("Transport backend: stdio|websocket")),
			fields.New("stdio-command", fields.TypeString, fields.WithDefault(defaults.StdioCommand), fields.WithHelp("Stdio command for app-server process")),
			fields.New("stdio-args", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Stdio command arguments (space-separated)")),
			fields.New("thread-id", fields.TypeString, fields.WithHelp("Thread identifier to read")),
			fields.New("include-turns", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Include turn data in read output")),
		),
	)
	return &threadReadCommand{CommandDescription: desc}, nil
}

func (c *threadReadCommand) Run(_ context.Context, vals *values.Values) error {
	s := &threadReadSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}
	if strings.TrimSpace(s.ThreadID) == "" {
		return fmt.Errorf("--thread-id is required")
	}

	out := map[string]any{
		"thread": map[string]any{
			"id":           s.ThreadID,
			"includeTurns": s.IncludeTurns,
			"status":       "phase-3-skeleton",
		},
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
