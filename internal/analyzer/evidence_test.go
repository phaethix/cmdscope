package analyzer_test

import (
	"testing"

	"github.com/phaethix/runmark/internal/analyzer"
	"github.com/phaethix/runmark/internal/ir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvidenceCommandSpan(t *testing.T) {
	ev := analyzer.CommandEvidence(8, 20, "> output.txt")
	require.Equal(t, ir.EvidenceCommand, ev.Source)
	require.NotNil(t, ev.StartByte)
	require.NotNil(t, ev.EndByte)
	assert.Equal(t, 8, *ev.StartByte)
	assert.Equal(t, 20, *ev.EndByte)
	assert.Equal(t, "> output.txt", ev.Snippet)
}

func TestEvidenceCommandInvalidSpanOmitsPointers(t *testing.T) {
	ev := analyzer.CommandEvidence(5, 5, "x")
	assert.Nil(t, ev.StartByte)
	assert.Nil(t, ev.EndByte)
	assert.Equal(t, "x", ev.Snippet)
	assert.Equal(t, ir.EvidenceCommand, ev.Source)
}

func TestEvidenceFile(t *testing.T) {
	ev := analyzer.FileEvidence(ir.EvidenceWorkspaceFile, "package.json", "scripts.build", "echo hi")
	assert.Equal(t, ir.EvidenceWorkspaceFile, ev.Source)
	assert.Equal(t, "package.json", ev.Path)
	assert.Equal(t, "scripts.build", ev.Field)
	assert.Equal(t, "echo hi", ev.Snippet)
	assert.Nil(t, ev.StartByte)
	assert.Nil(t, ev.EndByte)
}

func TestEvidenceEnsureEffectHasEvidence(t *testing.T) {
	ef := &ir.Effect{RawTarget: "out.txt", Target: "/ws/out.txt"}
	analyzer.EnsureEffectHasEvidence(ef)
	require.NotNil(t, ef.Evidence)
	require.Len(t, ef.Evidence, 1)
	assert.Equal(t, ir.EvidenceCommand, ef.Evidence[0].Source)
	assert.Equal(t, "out.txt", ef.Evidence[0].Snippet)

	// Idempotent when already present.
	analyzer.EnsureEffectHasEvidence(ef)
	require.Len(t, ef.Evidence, 1)

	ef2 := &ir.Effect{Evidence: []ir.Evidence{}}
	analyzer.EnsureEffectHasEvidence(ef2)
	require.Len(t, ef2.Evidence, 1)
}
