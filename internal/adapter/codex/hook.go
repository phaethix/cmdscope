// Package codex adapts Codex Bash PreToolUse events to runmark facts. The
// wire format is structurally identical to Claude Code's today; the package
// stays as a per-client seam so future protocol divergence remains local.
package codex

import (
	"context"
	"io"

	"github.com/phaethix/runmark/internal/adapter/shared"
)

// Handle translates one Codex hook stdin JSON event to stdout; see
// shared.HandlePreToolUseBash for the fail-open contract.
func Handle(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) int {
	return shared.HandlePreToolUseBash(ctx, stdin, stdout, stderr)
}
