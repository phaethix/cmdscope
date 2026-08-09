// Package codex adapts cmdscope analysis for Codex PreToolUse hook events.
package codex

import (
	"github.com/phaethix/cmdscope/internal/app"
	"github.com/phaethix/cmdscope/internal/ir"
)

// Placeholder anchors the codex adapter package boundary until hook wiring is implemented.
var Placeholder = struct {
	AppAnalyzer string
	IR          string
}{
	AppAnalyzer: app.Placeholder.Analyzer,
	IR:          ir.Placeholder,
}
