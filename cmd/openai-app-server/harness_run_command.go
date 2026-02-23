package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/openai-app-server/pkg/codexrpc"
	"github.com/go-go-golems/openai-app-server/pkg/config"
	"github.com/go-go-golems/openai-app-server/pkg/js"
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
	TimeoutMS      int    `glazed:"timeout-ms"`
	SettleMS       int    `glazed:"settle-ms"`
	DryRun         bool   `glazed:"dry-run"`
}

type harnessRPCBridge struct {
	client *codexrpc.Client
}

func (b *harnessRPCBridge) Request(ctx context.Context, method string, params any) (any, error) {
	resp, err := b.client.Request(ctx, method, params)
	if err != nil {
		return nil, err
	}
	if len(resp.Result) == 0 {
		return map[string]any{}, nil
	}
	var out any
	if err := json.Unmarshal(resp.Result, &out); err != nil {
		return string(resp.Result), nil
	}
	return out, nil
}

func (b *harnessRPCBridge) Notify(ctx context.Context, method string, params any) error {
	return b.client.Notify(ctx, method, params)
}

type stdoutUIBridge struct{}

func (b *stdoutUIBridge) Emit(_ context.Context, event any) error {
	raw, err := json.Marshal(event)
	if err != nil {
		return err
	}
	fmt.Printf("ui.emit %s\n", string(raw))
	return nil
}

var newHarnessRunClient = func(ctx context.Context, s *harnessRunSettings) (*codexrpc.Client, error) {
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
			"title":   "OpenAI App Server Harness Runner",
			"version": "0.1.0",
		},
	}); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}

var newHarnessRuntime = func(rpc js.RPCBridge, ui js.UIBridge) (*js.Runtime, error) {
	return js.NewRuntime(js.Options{
		Name: "openai-app-server-harness-run",
		RPC:  rpc,
		UI:   ui,
	})
}

func newHarnessRunCommand(defaults config.Defaults) (*harnessRunCommand, error) {
	desc := cmds.NewCommandDescription(
		"run",
		cmds.WithShort("Run a harness script"),
		cmds.WithLong("Execute a JS harness script against a codexrpc client transport."),
		cmds.WithFlags(
			fields.New("script", fields.TypeString, fields.WithHelp("Path to harness script")),
			fields.New("transport", fields.TypeString, fields.WithDefault(defaults.Transport), fields.WithHelp("Transport backend: stdio|websocket")),
			fields.New("stdio-command", fields.TypeString, fields.WithDefault(defaults.StdioCommand), fields.WithHelp("Stdio command for app-server process")),
			fields.New("stdio-args", fields.TypeString, fields.WithDefault(strings.Join(defaults.StdioArgs, " ")), fields.WithHelp("Stdio command arguments (space-separated)")),
			fields.New("model", fields.TypeString, fields.WithDefault(defaults.Model), fields.WithHelp("Default model for new thread sessions")),
			fields.New("cwd", fields.TypeString, fields.WithDefault(defaults.Cwd), fields.WithHelp("Working directory for thread sessions")),
			fields.New("approval-policy", fields.TypeString, fields.WithDefault(defaults.ApprovalPolicy), fields.WithHelp("Approval policy (phase-1 placeholder)")),
			fields.New("sandbox-policy", fields.TypeString, fields.WithDefault(defaults.SandboxPolicy), fields.WithHelp("Sandbox policy (phase-1 placeholder)")),
			fields.New("timeout-ms", fields.TypeInteger, fields.WithDefault(30000), fields.WithHelp("Handshake and RPC timeout in milliseconds")),
			fields.New("settle-ms", fields.TypeInteger, fields.WithDefault(250), fields.WithHelp("Post-script settle wait for async callbacks (milliseconds)")),
			fields.New("dry-run", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Print resolved settings without launching harness")),
		),
	)

	return &harnessRunCommand{CommandDescription: desc}, nil
}

func (c *harnessRunCommand) Run(ctx context.Context, vals *values.Values) error {
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
	fmt.Printf("timeout-ms=%d\n", s.TimeoutMS)
	fmt.Printf("settle-ms=%d\n", s.SettleMS)
	fmt.Printf("dry-run=%t\n", s.DryRun)

	if s.DryRun {
		return nil
	}

	if s.TimeoutMS <= 0 {
		s.TimeoutMS = 30000
	}
	if s.SettleMS < 0 {
		s.SettleMS = 0
	}

	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(s.TimeoutMS)*time.Millisecond)
	defer cancel()

	client, err := newHarnessRunClient(reqCtx, s)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	runtime, err := newHarnessRuntime(&harnessRPCBridge{client: client}, &stdoutUIBridge{})
	if err != nil {
		return err
	}
	defer func() { _ = runtime.Close() }()

	scriptBytes, err := os.ReadFile(s.ScriptPath)
	if err != nil {
		return fmt.Errorf("read script: %w", err)
	}
	if _, err := runtime.RunString(string(scriptBytes)); err != nil {
		return fmt.Errorf("execute script: %w", err)
	}

	if s.SettleMS > 0 {
		time.Sleep(time.Duration(s.SettleMS) * time.Millisecond)
	}
	fmt.Println("harness.run completed")
	return nil
}
