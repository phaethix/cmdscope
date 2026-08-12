package effect_test

import (
	"testing"

	"github.com/phaethix/runmark/internal/effect"
	"github.com/phaethix/runmark/internal/ir"
	"github.com/phaethix/runmark/internal/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMutationEffects(t *testing.T) {
	t.Run("rm path", func(t *testing.T) {
		effects, unknowns := extractMutate(t, "rm a", "/ws")
		require.Empty(t, unknowns)
		require.Len(t, effects, 1)
		assertEffect(t, effects[0], ir.EffectDelete, "a", "/ws/a", ir.Certain)
	})

	t.Run("rm options then paths", func(t *testing.T) {
		effects, unknowns := extractMutate(t, "rm -rf a b", "/ws")
		require.Empty(t, unknowns)
		require.Len(t, effects, 2)
		assertEffect(t, effects[0], ir.EffectDelete, "a", "/ws/a", ir.Certain)
		assertEffect(t, effects[1], ir.EffectDelete, "b", "/ws/b", ir.Certain)
	})

	t.Run("rm missing operand", func(t *testing.T) {
		effects, unknowns := extractMutate(t, "rm -f", "/ws")
		assert.Empty(t, effects)
		require.Len(t, unknowns, 1)
		assert.Equal(t, ir.UnknownUnsupportedCommand, unknowns[0].Code)
		assert.Equal(t, "stage:0", unknowns[0].Scope)
		assert.False(t, unknowns[0].Blocking)
	})

	t.Run("rm glob", func(t *testing.T) {
		effects, unknowns := extractMutate(t, "rm dist/*.js", "/ws")
		require.Len(t, effects, 1)
		assertEffect(t, effects[0], ir.EffectDelete, "dist/*.js", "/ws/dist/*.js", ir.Certain)
		require.Len(t, unknowns, 1)
		assert.Equal(t, ir.UnknownGlobRuntimeDependent, unknowns[0].Code)
		assert.False(t, unknowns[0].Blocking)
		require.NotEmpty(t, unknowns[0].Evidence)
	})

	t.Run("cp src dst", func(t *testing.T) {
		effects, unknowns := extractMutate(t, "cp src.txt dst.txt", "/ws")
		require.Empty(t, unknowns)
		require.Len(t, effects, 2)
		assertEffect(t, effects[0], ir.EffectRead, "src.txt", "/ws/src.txt", ir.Certain)
		assertEffect(t, effects[1], ir.EffectWrite, "dst.txt", "/ws/dst.txt", ir.Certain)
	})

	t.Run("cp insufficient", func(t *testing.T) {
		effects, unknowns := extractMutate(t, "cp only", "/ws")
		assert.Empty(t, effects)
		require.Len(t, unknowns, 1)
		assert.Equal(t, ir.UnknownUnsupportedCommand, unknowns[0].Code)
	})

	t.Run("mv src dst", func(t *testing.T) {
		effects, unknowns := extractMutate(t, "mv src.txt dst.txt", "/ws")
		require.Empty(t, unknowns)
		require.Len(t, effects, 2)
		assertEffect(t, effects[0], ir.EffectDelete, "src.txt", "/ws/src.txt", ir.Certain)
		assertEffect(t, effects[1], ir.EffectWrite, "dst.txt", "/ws/dst.txt", ir.Certain)
	})

	t.Run("mv insufficient", func(t *testing.T) {
		effects, unknowns := extractMutate(t, "mv only", "/ws")
		assert.Empty(t, effects)
		require.Len(t, unknowns, 1)
		assert.Equal(t, ir.UnknownUnsupportedCommand, unknowns[0].Code)
	})

	t.Run("cp glob src", func(t *testing.T) {
		effects, unknowns := extractMutate(t, "cp a/*.txt dest/", "/ws")
		require.Len(t, effects, 2)
		assertEffect(t, effects[0], ir.EffectRead, "a/*.txt", "/ws/a/*.txt", ir.Certain)
		assertEffect(t, effects[1], ir.EffectWrite, "dest/", "/ws/dest", ir.Certain)
		require.Len(t, unknowns, 1)
		assert.Equal(t, ir.UnknownGlobRuntimeDependent, unknowns[0].Code)
	})

	t.Run("cp -t dest sources", func(t *testing.T) {
		effects, unknowns := extractMutate(t, "cp -t dest a b", "/ws")
		require.Empty(t, unknowns)
		require.Len(t, effects, 3)
		assertEffect(t, effects[0], ir.EffectRead, "a", "/ws/a", ir.Certain)
		assertEffect(t, effects[1], ir.EffectRead, "b", "/ws/b", ir.Certain)
		assertEffect(t, effects[2], ir.EffectWrite, "dest", "/ws/dest", ir.Certain)
	})

	t.Run("rm dash skipped", func(t *testing.T) {
		effects, unknowns := extractMutate(t, "rm -", "/ws")
		assert.Empty(t, effects)
		require.Len(t, unknowns, 1)
		assert.Equal(t, ir.UnknownUnsupportedCommand, unknowns[0].Code)
	})

	t.Run("not a mutate command", func(t *testing.T) {
		effects, unknowns := extractMutate(t, "echo hi", "/ws")
		assert.Empty(t, effects)
		assert.Empty(t, unknowns)
	})
}

func TestMutationEffectsNil(t *testing.T) {
	effects, unknowns := effect.ExtractMutate(nil, 0, ir.Condition{Kind: ir.ConditionAlways}, "/ws")
	assert.Empty(t, effects)
	assert.Empty(t, unknowns)
	effects, unknowns = effect.ExtractMutate(&shell.SimpleCommand{}, 0, ir.Condition{Kind: ir.ConditionAlways}, "/ws")
	assert.Empty(t, effects)
	assert.Empty(t, unknowns)
}

func extractMutate(t *testing.T, command, cwd string) ([]ir.Effect, []ir.Unknown) {
	t.Helper()
	cmd := parseSimple(t, command)
	return effect.ExtractMutate(cmd, 0, ir.Condition{Kind: ir.ConditionAlways}, cwd)
}

func assertEffect(t *testing.T, ef ir.Effect, kind ir.EffectKind, raw, target string, cert ir.Certainty) {
	t.Helper()
	assert.Equal(t, kind, ef.Kind)
	assert.Equal(t, raw, ef.RawTarget)
	assert.Equal(t, target, ef.Target)
	assert.Equal(t, cert, ef.Certainty)
	assert.Equal(t, ir.FromCommand, ef.Provenance)
	assert.Equal(t, ir.EffectID(ir.SchemaVersion, ef), ef.ID)
	require.NotEmpty(t, ef.Evidence)
}
