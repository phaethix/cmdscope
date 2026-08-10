// Package expand provides bounded static expansion for npm, pnpm, make, and scripts.
package expand

import "github.com/phaethix/cmdscope/internal/ir"

// Placeholder marks the expand package boundary for import graph checks.
var Placeholder = struct {
	IR string
}{
	IR: ir.Placeholder,
}
