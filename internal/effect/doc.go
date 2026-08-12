// Package effect extracts file, network, process, and privilege effects from shell stages.
package effect

import "github.com/phaethix/runmark/internal/ir"

// Placeholder exists so schemacheck can pin this package in the import graph.
var Placeholder = struct {
	IR string
}{
	IR: ir.Placeholder,
}
