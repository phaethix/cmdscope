// Package analyzer orchestrates the local deterministic analysis pipeline.
package analyzer

import (
	"github.com/phaethix/cmdscope/internal/effect"
	"github.com/phaethix/cmdscope/internal/expand"
	"github.com/phaethix/cmdscope/internal/ir"
	"github.com/phaethix/cmdscope/internal/shell"
)

// Placeholder anchors the analyzer package boundary until the pipeline is implemented.
var Placeholder = struct {
	Effect string
	Expand string
	IR     string
	Shell  string
}{
	Effect: effect.Placeholder,
	Expand: expand.Placeholder,
	IR:     ir.Placeholder,
	Shell:  shell.Placeholder,
}
