package render

import (
	"encoding/json"

	"github.com/phaethix/runmark/internal/ir"
)

// Validate is the renderer gate: every JSON/text path must call it before
// emitting bytes. A contract violation must abort the entire render so callers
// never see a partial ImpactReport on stdout.
func Validate(report ir.ImpactReport) error {
	return ir.ValidateReport(report)
}

// JSON encodes a report only after Validate succeeds. On failure the byte
// slice is nil so a careless caller cannot print a half-built body.
func JSON(report ir.ImpactReport) ([]byte, error) {
	if err := Validate(report); err != nil {
		return nil, err
	}
	return json.Marshal(report)
}
