// Package codex adapts cmdscope analysis for Codex PreToolUse hook events.
package codex

import (
	"github.com/phaethix/cmdscope/internal/app"
	"github.com/phaethix/cmdscope/internal/ir"
)

// Placeholder exists so schemacheck can pin this package in the import graph.
var Placeholder = struct {
	AppAnalyzer string
	IR          string
}{
	AppAnalyzer: app.Placeholder.Analyzer,
	IR:          ir.Placeholder,
}
