package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/openai-app-server/pkg/state"
)

type harnessStateReplayCommand struct {
	*cmds.CommandDescription
}

var _ cmds.BareCommand = (*harnessStateReplayCommand)(nil)

type harnessStateReplaySettings struct {
	EventsFile string `glazed:"events-file"`
	ThreadID   string `glazed:"thread-id"`
	MaxThreads int    `glazed:"max-threads"`
	MaxTurns   int    `glazed:"max-turns"`
	MaxItems   int    `glazed:"max-items"`
}

type replayEvent struct {
	Method string         `json:"method"`
	Params map[string]any `json:"params"`
}

func newHarnessStateReplayCommand() (*harnessStateReplayCommand, error) {
	desc := cmds.NewCommandDescription(
		"state-replay",
		cmds.WithShort("Project state from recorded event JSON"),
		cmds.WithLong("Replay JSON event records into the in-memory state projector and print the projected state snapshot."),
		cmds.WithFlags(
			fields.New("events-file", fields.TypeString, fields.WithHelp("Path to JSON file containing an array of {method,params} events")),
			fields.New("thread-id", fields.TypeString, fields.WithHelp("If set, output only one projected thread by id")),
			fields.New("max-threads", fields.TypeInteger, fields.WithDefault(100), fields.WithHelp("Retention bound for thread projections")),
			fields.New("max-turns", fields.TypeInteger, fields.WithDefault(100), fields.WithHelp("Retention bound per thread for turn projections")),
			fields.New("max-items", fields.TypeInteger, fields.WithDefault(200), fields.WithHelp("Retention bound per turn for item projections")),
		),
	)
	return &harnessStateReplayCommand{CommandDescription: desc}, nil
}

func (c *harnessStateReplayCommand) Run(_ context.Context, vals *values.Values) error {
	s := &harnessStateReplaySettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}
	if strings.TrimSpace(s.EventsFile) == "" {
		return fmt.Errorf("--events-file is required")
	}

	b, err := os.ReadFile(s.EventsFile)
	if err != nil {
		return fmt.Errorf("read events file: %w", err)
	}

	events := []replayEvent{}
	if err := json.Unmarshal(b, &events); err != nil {
		return fmt.Errorf("decode events file: %w", err)
	}

	store := state.NewStore(state.Config{
		MaxThreads:        s.MaxThreads,
		MaxTurnsPerThread: s.MaxTurns,
		MaxItemsPerTurn:   s.MaxItems,
	})
	projector := state.NewProjector(store)

	for _, evt := range events {
		projector.Apply(evt.Method, evt.Params)
	}

	if threadID := strings.TrimSpace(s.ThreadID); threadID != "" {
		thread, ok := store.Thread(threadID)
		if !ok {
			return fmt.Errorf("thread %q not found in projected state", threadID)
		}
		out := map[string]any{"thread": thread}
		return printIndentedJSON(out)
	}

	out := map[string]any{"threads": store.ListThreads()}
	return printIndentedJSON(out)
}

func printIndentedJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
