// Package analyzer orchestrates the local deterministic analysis pipeline.
package analyzer

import (
	"github.com/phaethix/cmdscope/internal/ir"
)

// Placeholder exists so schemacheck can pin this package in the import graph.
var Placeholder = struct {
	IR string
}{
	IR: ir.Placeholder,
}
