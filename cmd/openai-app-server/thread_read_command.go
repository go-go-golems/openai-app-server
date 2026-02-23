package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/openai-app-server/pkg/codexrpc"
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
	TimeoutMS    int    `glazed:"timeout-ms"`
	ThreadID     string `glazed:"thread-id"`
	IncludeTurns bool   `glazed:"include-turns"`
}

var newThreadReadClient = func(ctx context.Context, s *threadReadSettings) (*codexrpc.Client, error) {
	if s.Transport != "stdio" {
		return nil, fmt.Errorf("unsupported transport %q", s.Transport)
	}

	stdioArgs := []string{}
	if trimmed := strings.TrimSpace(s.StdioArgs); trimmed != "" {
		stdioArgs = strings.Fields(trimmed)
	}

	transport, err := codexrpc.NewStdioTransport(s.StdioCommand, stdioArgs...)
	if err != nil {
		return nil, err
	}
	client := codexrpc.NewClient(transport)
	if err := client.Connect(ctx, map[string]any{
		"clientInfo": map[string]any{
			"name":    "openai-app-server",
			"version": "0.1.0",
		},
	}); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}

func newThreadReadCommand(defaults config.Defaults) (*threadReadCommand, error) {
	desc := cmds.NewCommandDescription(
		"read",
		cmds.WithShort("Read thread details"),
		cmds.WithLong("Read one thread through codexrpc client transport."),
		cmds.WithFlags(
			fields.New("transport", fields.TypeString, fields.WithDefault(defaults.Transport), fields.WithHelp("Transport backend: stdio|websocket")),
			fields.New("stdio-command", fields.TypeString, fields.WithDefault(defaults.StdioCommand), fields.WithHelp("Stdio command for app-server process")),
			fields.New("stdio-args", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Stdio command arguments (space-separated)")),
			fields.New("timeout-ms", fields.TypeInteger, fields.WithDefault(15000), fields.WithHelp("Request timeout in milliseconds")),
			fields.New("thread-id", fields.TypeString, fields.WithHelp("Thread identifier to read")),
			fields.New("include-turns", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Include turn data in read output")),
		),
	)
	return &threadReadCommand{CommandDescription: desc}, nil
}

func (c *threadReadCommand) Run(ctx context.Context, vals *values.Values) error {
	s := &threadReadSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}
	if strings.TrimSpace(s.ThreadID) == "" {
		return fmt.Errorf("--thread-id is required")
	}
	if s.TimeoutMS <= 0 {
		s.TimeoutMS = 15000
	}

	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(s.TimeoutMS)*time.Millisecond)
	defer cancel()

	client, err := newThreadReadClient(reqCtx, s)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	thread, err := client.ThreadRead(reqCtx, s.ThreadID, s.IncludeTurns)
	if err != nil {
		return err
	}

	out := map[string]any{
		"thread": thread,
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
