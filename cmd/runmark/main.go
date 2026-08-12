// Command runmark is the CLI for the runmark command impact analyzer.
//
// This entrypoint only assembles dependencies, parses the subcommand,
// and delegates to internal/app. Analyzer logic intentionally lives
// outside this package.
package main

import (
	"fmt"
	"os"

	"github.com/phaethix/runmark/internal/app"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "version" {
		if err := app.PrintVersion(os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "runmark: write version:", err)
			os.Exit(1)
		}
		return
	}

	fmt.Fprintln(os.Stderr, "usage: runmark version")
	os.Exit(2)
}
