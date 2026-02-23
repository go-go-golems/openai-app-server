package main

import (
	"context"
	"fmt"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
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
	Limit        int    `glazed:"limit"`
}

func newThreadListCommand(defaults config.Defaults) (*threadListCommand, error) {
	desc := cmds.NewCommandDescription(
		"list",
		cmds.WithShort("List threads (phase-1 skeleton)"),
		cmds.WithLong("Phase-1 skeleton for thread listing. Real codex RPC integration is added in phase-2/phase-3."),
		cmds.WithFlags(
			fields.New("transport", fields.TypeString, fields.WithDefault(defaults.Transport), fields.WithHelp("Transport backend: stdio|websocket")),
			fields.New("stdio-command", fields.TypeString, fields.WithDefault(defaults.StdioCommand), fields.WithHelp("Stdio command for app-server process")),
			fields.New("stdio-args", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Stdio command arguments (space-separated)")),
			fields.New("limit", fields.TypeInteger, fields.WithDefault(20), fields.WithHelp("Max threads to list")),
		),
	)
	return &threadListCommand{CommandDescription: desc}, nil
}

func (c *threadListCommand) Run(_ context.Context, vals *values.Values) error {
	s := &threadListSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}

	fmt.Printf("phase-1 thread.list skeleton\n")
	fmt.Printf("transport=%s\n", s.Transport)
	fmt.Printf("stdio.command=%s\n", s.StdioCommand)
	fmt.Printf("stdio.args=%s\n", s.StdioArgs)
	fmt.Printf("limit=%d\n", s.Limit)
	fmt.Println("threads=[]")
	return nil
}
