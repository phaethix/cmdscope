package ir_test

import (
	"strings"
	"testing"

	"github.com/phaethix/runmark/internal/ir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEffectIDStableFormula(t *testing.T) {
	ef := baseEffectIDSample()
	got := ir.EffectID(ir.SchemaVersion, ef)
	require.True(t, strings.HasPrefix(got, "sha256:"))
	require.Equal(t, 7+64, len(got))
	require.Equal(t, ir.EffectID(ir.SchemaVersion, ef), got)

	ef.Target = "other"
	require.NotEqual(t, got, ir.EffectID(ir.SchemaVersion, ef))
}

func TestEffectIDInputSensitivity(t *testing.T) {
	base := baseEffectIDSample()
	baseID := ir.EffectID(ir.SchemaVersion, base)

	t.Run("schemaVersion", func(t *testing.T) {
		assert.NotEqual(t, baseID, ir.EffectID("0.2", base))
	})

	cases := []struct {
		name string
		mut  func(*ir.Effect)
	}{
		{"stage", func(ef *ir.Effect) { ef.Stage = 1 }},
		{"kind", func(ef *ir.Effect) { ef.Kind = ir.EffectRead }},
		{"raw_target", func(ef *ir.Effect) { ef.RawTarget = "GO" }},
		{"target", func(ef *ir.Effect) { ef.Target = "elsewhere" }},
		{"condition_kind", func(ef *ir.Effect) {
			ef.Condition = ir.Condition{Kind: ir.ConditionOnSuccess, DependsOn: 0}
		}},
		{"condition_depends_on", func(ef *ir.Effect) {
			// Keep kind fixed so DependsOn alone must change the digest.
			ef.Condition = ir.Condition{Kind: ir.ConditionAlways, DependsOn: 1}
		}},
		{"provenance", func(ef *ir.Effect) { ef.Provenance = ir.Inferred }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ef := base
			tc.mut(&ef)
			assert.NotEqual(t, baseID, ir.EffectID(ir.SchemaVersion, ef), tc.name)
		})
	}

	t.Run("certainty and evidence excluded", func(t *testing.T) {
		ef := base
		ef.Certainty = ir.Possible
		ef.Evidence = []ir.Evidence{{Source: ir.EvidenceCommand, Snippet: "x"}}
		assert.Equal(t, baseID, ir.EffectID(ir.SchemaVersion, ef))
	})
}

func baseEffectIDSample() ir.Effect {
	return ir.Effect{
		Kind:       ir.EffectProcess,
		RawTarget:  "go",
		Target:     "go",
		Stage:      0,
		Provenance: ir.FromCommand,
		Condition:  ir.Condition{Kind: ir.ConditionAlways, DependsOn: 0},
	}
}
