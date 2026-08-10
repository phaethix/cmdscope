// Package analyzer orchestrates the local deterministic analysis pipeline.
package analyzer

import (
	"github.com/phaethix/cmdscope/internal/ir"
)

// Placeholder anchors the analyzer package boundary until the pipeline is implemented.
var Placeholder = struct {
	IR string
}{
	IR: ir.Placeholder,
}
