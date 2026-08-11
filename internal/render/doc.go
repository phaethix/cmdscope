// Package render produces deterministic JSON and text output from ImpactReport IR.
package render

import (
	"encoding/json"

	"github.com/phaethix/cmdscope/internal/ir"
)

// Placeholder exists so schemacheck can pin this package in the import graph.
var Placeholder = struct {
	IR string
}{
	IR: ir.Placeholder,
}

// MarshalReport is a placeholder for the deterministic JSON encoder.
// The real implementation MUST produce byte-for-byte identical output for
// equivalent inputs (Determinism requirement in CONTRIBUTING.md).
func MarshalReport(r ir.ImpactReport) ([]byte, error) {
	return json.Marshal(r)
}
