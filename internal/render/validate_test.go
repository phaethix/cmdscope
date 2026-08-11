package render_test

import (
	"strings"
	"testing"

	"github.com/phaethix/cmdscope/internal/ir"
	"github.com/phaethix/cmdscope/internal/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderValidation(t *testing.T) {
	t.Run("valid report encodes", func(t *testing.T) {
		r := renderValidReport()
		require.NoError(t, render.Validate(r))
		out, err := render.JSON(r)
		require.NoError(t, err)
		require.NotEmpty(t, out)
		assert.Equal(t, byte('{'), out[0])
	})

	t.Run("tampered effect id rejects and emits no json", func(t *testing.T) {
		r := renderValidReport()
		r.Stages[0].Effects[0].ID = "sha256:" + strings.Repeat("ab", 32)
		require.Error(t, render.Validate(r))

		out, err := render.JSON(r)
		require.Error(t, err)
		assert.Nil(t, out)

		out2, err2 := render.MarshalReport(r)
		require.Error(t, err2)
		assert.Nil(t, out2)

		var cv *ir.ContractViolationError
		require.ErrorAs(t, err, &cv)
		assert.Equal(t, ir.ContractViolationErrorCode, cv.Code)
	})
}

func renderValidReport() ir.ImpactReport {
	cond := ir.Condition{Kind: ir.ConditionAlways, DependsOn: 0}
	ef := ir.Effect{
		Kind:       ir.EffectWrite,
		RawTarget:  "output.txt",
		Target:     "/repo/output.txt",
		Stage:      0,
		Certainty:  ir.Certain,
		Provenance: ir.FromCommand,
		Condition:  cond,
		Evidence: []ir.Evidence{{
			Source:    ir.EvidenceCommand,
			StartByte: new(5),
			EndByte:   new(11),
			Snippet:   "> output.txt",
		}},
	}
	ef.ID = ir.EffectID(ir.SchemaVersion, ef)
	return ir.ImpactReport{
		SchemaVersion: ir.SchemaVersion,
		Command:       "echo hi > output.txt",
		CWD:           "/repo",
		Analysis: ir.AnalysisMeta{
			Coverage:     ir.CoverageComplete,
			Completeness: ir.CompletenessComplete,
			Limits:       []string{},
			Parser:       "shell",
		},
		Stages: []ir.Stage{{
			Index:        0,
			Command:      "echo hi > output.txt",
			Condition:    cond,
			Completeness: ir.CompletenessComplete,
			Effects:      []ir.Effect{ef},
		}},
		Unknowns: []ir.Unknown{},
		Flags:    []ir.Flag{},
		Summary:  "write output.txt",
	}
}
