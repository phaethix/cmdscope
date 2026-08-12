// Command runmark is the CLI for the runmark command impact analyzer.
//
// This entrypoint only assembles dependencies, parses the subcommand,
// and delegates to internal/app. Analyzer logic intentionally lives
// outside this package.
package main

import (
	"os"

	"github.com/phaethix/runmark/internal/app"
)

func main() {
	os.Exit(app.Run(os.Args[1:], os.Stdout, os.Stderr))
}
