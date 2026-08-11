package app

import (
	"github.com/phaethix/cmdscope/internal/analyzer"
	"github.com/phaethix/cmdscope/internal/ir"
)

// Placeholder exists so schemacheck can pin this package in the import graph.
var Placeholder = struct {
	Analyzer string
	IR       string
}{
	Analyzer: analyzer.Placeholder.IR,
	IR:       ir.Placeholder,
}
