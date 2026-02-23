package main

import (
	"fmt"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/openai-app-server/pkg/config"
	"github.com/spf13/cobra"
)

func newRootCommand() (*cobra.Command, error) {
	defaults, err := config.NewDefaults()
	if err != nil {
		return nil, err
	}

	root := &cobra.Command{
		Use:   "openai-app-server",
		Short: "Run JS harnesses against Codex App Server",
		Long:  "OpenAI App Server CLI with Glazed commands for progressive harness development.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return logging.InitLoggerFromViper()
		},
	}

	harnessGroup := &cobra.Command{
		Use:   "harness",
		Short: "Harness authoring and execution commands",
	}
	threadGroup := &cobra.Command{
		Use:   "thread",
		Short: "Thread inspection and control commands",
	}

	harnessRun, err := newHarnessRunCommand(defaults)
	if err != nil {
		return nil, fmt.Errorf("build harness run command: %w", err)
	}
	harnessRunCobra, err := cli.BuildCobraCommand(harnessRun,
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpSections: []string{schema.DefaultSlug},
			MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("wrap harness run command: %w", err)
	}

	threadList, err := newThreadListCommand(defaults)
	if err != nil {
		return nil, fmt.Errorf("build thread list command: %w", err)
	}
	threadListCobra, err := cli.BuildCobraCommand(threadList,
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpSections: []string{schema.DefaultSlug},
			MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("wrap thread list command: %w", err)
	}

	threadRead, err := newThreadReadCommand(defaults)
	if err != nil {
		return nil, fmt.Errorf("build thread read command: %w", err)
	}
	threadReadCobra, err := cli.BuildCobraCommand(threadRead,
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpSections: []string{schema.DefaultSlug},
			MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("wrap thread read command: %w", err)
	}

	harnessGroup.AddCommand(harnessRunCobra)
	threadGroup.AddCommand(threadListCobra)
	threadGroup.AddCommand(threadReadCobra)
	root.AddCommand(harnessGroup)
	root.AddCommand(threadGroup)

	return root, nil
}
