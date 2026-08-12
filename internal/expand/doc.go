// Package expand provides bounded static expansion for npm, pnpm, make, and scripts.
package expand

import "github.com/phaethix/runmark/internal/ir"

// Placeholder exists so schemacheck can pin this package in the import graph.
var Placeholder = struct {
	IR string
}{
	IR: ir.Placeholder,
}
