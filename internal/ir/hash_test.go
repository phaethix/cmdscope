package ir_test

import (
	"strings"
	"testing"

	"github.com/phaethix/cmdscope/internal/ir"
	"github.com/stretchr/testify/require"
)

func TestEffectIDStableFormula(t *testing.T) {
	ef := ir.Effect{
		Kind:       ir.EffectProcess,
		RawTarget:  "go",
		Target:     "go",
		Stage:      0,
		Provenance: ir.FromCommand,
		Condition:  ir.Condition{Kind: ir.ConditionAlways, DependsOn: 0},
	}
	got := ir.EffectID(ir.SchemaVersion, ef)
	require.True(t, strings.HasPrefix(got, "sha256:"))
	require.Equal(t, ir.EffectID(ir.SchemaVersion, ef), got)

	ef.Target = "other"
	require.NotEqual(t, got, ir.EffectID(ir.SchemaVersion, ef))
}
