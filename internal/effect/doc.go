// Package effect extracts file, network, process, and privilege effects from shell stages.
package effect

import "github.com/phaethix/cmdscope/internal/ir"

// Placeholder marks the effect package boundary for import graph checks.
var Placeholder = struct {
	IR string
}{
	IR: ir.Placeholder,
}
