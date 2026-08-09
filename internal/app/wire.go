package app

import (
	"github.com/phaethix/cmdscope/internal/analyzer"
	"github.com/phaethix/cmdscope/internal/ir"
)

// Placeholder anchors the app package wiring boundary until analyze/validate commands land.
var Placeholder = struct {
	Analyzer string
	IR       string
}{
	Analyzer: analyzer.Placeholder.IR,
	IR:       ir.Placeholder,
}
