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

type threadListCommand struct {
	*cmds.CommandDescription
}

var _ cmds.BareCommand = (*threadListCommand)(nil)

type threadListSettings struct {
	Transport    string `glazed:"transport"`
	StdioCommand string `glazed:"stdio-command"`
	StdioArgs    string `glazed:"stdio-args"`
	TimeoutMS    int    `glazed:"timeout-ms"`
	Limit        int    `glazed:"limit"`
}

var newThreadListClient = func(ctx context.Context, s *threadListSettings) (*codexrpc.Client, error) {
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

func newThreadListCommand(defaults config.Defaults) (*threadListCommand, error) {
	desc := cmds.NewCommandDescription(
		"list",
		cmds.WithShort("List threads"),
		cmds.WithLong("List threads through codexrpc client transport. Use stdio transport in normal runs; tests can inject an in-memory transport."),
		cmds.WithFlags(
			fields.New("transport", fields.TypeString, fields.WithDefault(defaults.Transport), fields.WithHelp("Transport backend: stdio|websocket")),
			fields.New("stdio-command", fields.TypeString, fields.WithDefault(defaults.StdioCommand), fields.WithHelp("Stdio command for app-server process")),
			fields.New("stdio-args", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Stdio command arguments (space-separated)")),
			fields.New("timeout-ms", fields.TypeInteger, fields.WithDefault(15000), fields.WithHelp("Request timeout in milliseconds")),
			fields.New("limit", fields.TypeInteger, fields.WithDefault(20), fields.WithHelp("Max threads to list")),
		),
	)
	return &threadListCommand{CommandDescription: desc}, nil
}

func (c *threadListCommand) Run(ctx context.Context, vals *values.Values) error {
	s := &threadListSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}

	if s.TimeoutMS <= 0 {
		s.TimeoutMS = 15000
	}

	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(s.TimeoutMS)*time.Millisecond)
	defer cancel()

	client, err := newThreadListClient(reqCtx, s)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	threads, err := client.ThreadList(reqCtx, s.Limit)
	if err != nil {
		return err
	}

	out := map[string]any{"threads": threads}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
