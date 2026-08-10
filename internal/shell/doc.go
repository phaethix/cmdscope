// Package shell provides lexer, AST, parser, and stage splitting for shell commands.
package shell

import "github.com/phaethix/cmdscope/internal/ir"

// Placeholder marks the shell package boundary for import graph checks.
var Placeholder = struct {
	IR string
}{
	IR: ir.Placeholder,
}
