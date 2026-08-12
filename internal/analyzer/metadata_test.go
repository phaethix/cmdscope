package analyzer_test

import (
	"testing"

	"github.com/phaethix/runmark/internal/analyzer"
	"github.com/phaethix/runmark/internal/ir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetadataConditionCertainty(t *testing.T) {
	always := ir.Condition{Kind: ir.ConditionAlways}
	onOK := ir.Condition{Kind: ir.ConditionOnSuccess, DependsOn: 0}

	assert.Equal(t, ir.Certain, analyzer.ApplyConditionCertainty(ir.Certain, always))
	assert.Equal(t, ir.Conditional, analyzer.ApplyConditionCertainty(ir.Certain, onOK))
	assert.Equal(t, ir.Conditional, analyzer.ApplyConditionCertainty(ir.Conditional, onOK))
	assert.Equal(t, ir.Possible, analyzer.ApplyConditionCertainty(ir.Possible, onOK))
	assert.Equal(t, ir.CertaintyUnknown, analyzer.ApplyConditionCertainty(ir.CertaintyUnknown, onOK))
}

func TestMetadataDefaultProvenance(t *testing.T) {
	assert.Equal(t, ir.FromCommand, analyzer.DefaultProvenance())
}

func TestMetadataAggregateFlags(t *testing.T) {
	effects := []ir.Effect{
		{Kind: ir.EffectDelete},
		{Kind: ir.EffectNetwork},
		{Kind: ir.EffectPrivilege},
		{Kind: ir.EffectExecuteRemote},
	}
	unknowns := []ir.Unknown{
		{Code: ir.UnknownContextMissing},
		{Code: ir.UnknownUnsupportedCommand},
		{Code: ir.UnknownInterpreterDynamicCode},
		{Code: ir.UnknownAnalysisTimeout},
	}
	flags := analyzer.AggregateFlags(effects, unknowns)
	require.Equal(t, []ir.Flag{
		ir.FlagAnalysisTimeout,
		ir.FlagContextMissing,
		ir.FlagDestructive,
		ir.FlagExternalNetwork,
		ir.FlagOpaqueScript,
		ir.FlagPrivilegeChange,
		ir.FlagRemoteContent,
		ir.FlagUnsupported,
	}, flags)

	// Factual labels only — never a risk conclusion field.
	for _, f := range flags {
		assert.NotEqual(t, "risk_level", string(f))
	}

	merged := analyzer.AggregateFlags(nil, nil, ir.FlagDestructive, ir.FlagDestructive)
	assert.Equal(t, []ir.Flag{ir.FlagDestructive}, merged)

	fromUnknown := analyzer.AggregateFlags(nil, []ir.Unknown{{Code: ir.UnknownRemoteContent}})
	assert.Equal(t, []ir.Flag{ir.FlagRemoteContent}, fromUnknown)

	empty := analyzer.AggregateFlags(nil, nil)
	require.NotNil(t, empty)
	assert.Empty(t, empty)
}

func TestCompletenessComplete(t *testing.T) {
	effects := []ir.Effect{
		{Certainty: ir.Certain},
		{Certainty: ir.Conditional},
	}
	assert.Equal(t, ir.CompletenessComplete, analyzer.StageCompleteness(effects, nil))
}

func TestCompletenessPartialNonBlocking(t *testing.T) {
	effects := []ir.Effect{{Certainty: ir.Certain}}
	unknowns := []ir.Unknown{{Code: ir.UnknownEnvMissing, Blocking: false}}
	assert.Equal(t, ir.CompletenessPartial, analyzer.StageCompleteness(effects, unknowns))
}

func TestCompletenessPartialPossible(t *testing.T) {
	effects := []ir.Effect{{Certainty: ir.Possible}}
	assert.Equal(t, ir.CompletenessPartial, analyzer.StageCompleteness(effects, nil))
}

func TestCompletenessUnknownBlocking(t *testing.T) {
	effects := []ir.Effect{{Certainty: ir.Certain}}
	unknowns := []ir.Unknown{{Code: ir.UnknownRemoteContent, Blocking: true}}
	assert.Equal(t, ir.CompletenessUnknown, analyzer.StageCompleteness(effects, unknowns))
}

func TestCompletenessUnknownEffectCertainty(t *testing.T) {
	effects := []ir.Effect{{Certainty: ir.CertaintyUnknown}}
	assert.Equal(t, ir.CompletenessUnknown, analyzer.StageCompleteness(effects, nil))
}

func TestCompletenessReportRollup(t *testing.T) {
	stages := []ir.Stage{
		{Index: 0, Effects: []ir.Effect{{Certainty: ir.Certain}}},
		{Index: 1, Effects: []ir.Effect{{Certainty: ir.Certain}}},
	}
	assert.Equal(t, ir.CompletenessComplete, analyzer.ReportCompleteness(stages, nil))

	partial := []ir.Unknown{{Code: ir.UnknownGlobRuntimeDependent, Scope: "stage:1", Blocking: false}}
	assert.Equal(t, ir.CompletenessPartial, analyzer.ReportCompleteness(stages, partial))

	blocking := []ir.Unknown{{Code: ir.UnknownRemoteContent, Scope: "stage:0", Blocking: true}}
	assert.Equal(t, ir.CompletenessUnknown, analyzer.ReportCompleteness(stages, blocking))

	reportScope := []ir.Unknown{{Code: ir.UnknownAnalysisTimeout, Scope: "report", Blocking: true}}
	assert.Equal(t, ir.CompletenessUnknown, analyzer.ReportCompleteness(stages, reportScope))

	reportPartial := []ir.Unknown{{Code: ir.UnknownEnvMissing, Scope: "report", Blocking: false}}
	assert.Equal(t, ir.CompletenessPartial, analyzer.ReportCompleteness(stages, reportPartial))
}
