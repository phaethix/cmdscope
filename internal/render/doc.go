// Package render produces deterministic JSON and text output from ImpactReport IR.
package render

import "github.com/phaethix/cmdscope/internal/ir"

// Placeholder exists so schemacheck can pin this package in the import graph.
var Placeholder = struct {
	IR string
}{
	IR: ir.Placeholder,
}

// MarshalReport is the JSON entry point used by early callers. It refuses to
// emit any bytes when Validate fails (same gate as JSON).
func MarshalReport(r ir.ImpactReport) ([]byte, error) {
	return JSON(r)
}
