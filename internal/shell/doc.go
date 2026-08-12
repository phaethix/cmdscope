// Package shell provides lexer, AST, parser, and stage splitting for shell commands.
package shell

import "github.com/phaethix/runmark/internal/ir"

// Placeholder exists so schemacheck can pin this package in the import graph.
var Placeholder = struct {
	IR string
}{
	IR: ir.Placeholder,
}
