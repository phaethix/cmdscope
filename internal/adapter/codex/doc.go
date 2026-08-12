// Package codex adapts runmark analysis for Codex PreToolUse hook events.
//
// It translates host protocol JSON only: analysis stays in analyzer/facts.
package codex

import (
	"github.com/phaethix/runmark/internal/facts"
	"github.com/phaethix/runmark/internal/ir"
)

// Placeholder exists so schemacheck can pin this package in the import graph.
var Placeholder = struct {
	Facts string
	IR    string
}{
	Facts: facts.SchemaVersion,
	IR:    ir.Placeholder,
}
