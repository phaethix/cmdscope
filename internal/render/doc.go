// Package render produces deterministic JSON and text output from ImpactReport IR.
package render

import (
	"encoding/json"

	"github.com/phaethix/cmdscope/internal/ir"
)

// Placeholder anchors the render package boundary until renderers are implemented.
var Placeholder = struct {
	IR string
}{
	IR: ir.Placeholder,
}

// MarshalReport is a placeholder for the deterministic JSON encoder.
// The real implementation MUST produce byte-for-byte identical output for
// equivalent inputs — see CONTRIBUTING.md § Determinism requirement.
func MarshalReport(r ir.ImpactReport) ([]byte, error) {
	return json.Marshal(r)
}
