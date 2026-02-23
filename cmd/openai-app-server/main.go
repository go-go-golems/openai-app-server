package main

import (
	"fmt"
	"os"
)

func main() {
	root, err := newRootCommand()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to build root command: %v\n", err)
		os.Exit(1)
	}

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
