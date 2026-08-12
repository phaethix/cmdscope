package analyzer_test

import (
	"context"
	"testing"

	"github.com/phaethix/runmark/internal/analyzer"
	"github.com/phaethix/runmark/internal/ir"
	"github.com/stretchr/testify/require"
)

// workspaceCWD is a stable logical root so every test exercises the same
// request shape without touching the host filesystem.
const workspaceCWD = "logical://workspace"

func TestAnalyzeSkeletonSimpleCommand(t *testing.T) {
	req := ir.AnalyzeRequest{
		Command: "echo hi",
		Context: &ir.AnalysisContext{CWD: workspaceCWD},
	}
	report, err := analyzer.Analyze(context.Background(), req)
	require.NoError(t, err)
	require.NoError(t, ir.ValidateReport(report))
	require.Equal(t, "echo hi", report.Command)
	require.Equal(t, workspaceCWD, report.CWD)
	require.Len(t, report.Stages, 1)
	st := report.Stages[0]
	require.Equal(t, 0, st.Index)
	require.Equal(t, "echo hi", st.Command)
	require.Equal(t, ir.ConditionAlways, st.Condition.Kind)
	require.Equal(t, 0, st.Condition.DependsOn)
	require.Empty(t, st.Effects)
	require.Empty(t, report.Unknowns)
	require.Equal(t, ir.CoverageComplete, report.Analysis.Coverage)
	require.Equal(t, ir.CompletenessComplete, report.Analysis.Completeness)
}

func TestAnalyzeSkeletonCompoundStages(t *testing.T) {
	const cmd = "a && echo b || c; d | e"
	req := ir.AnalyzeRequest{
		Command: cmd,
		Context: &ir.AnalysisContext{CWD: workspaceCWD},
	}
	report, err := analyzer.Analyze(context.Background(), req)
	require.NoError(t, err)
	require.NoError(t, ir.ValidateReport(report))
	want := []struct {
		command   string
		condKind  ir.ConditionKind
		dependsOn int
	}{
		{"a", ir.ConditionAlways, 0},
		{"echo b", ir.ConditionOnSuccess, 0}, // 1-based dep 1 -> 0-based 0
		{"c", ir.ConditionOnFailure, 1},      // 1-based dep 2 -> 0-based 1
		{"d | e", ir.ConditionAlways, 0},
	}
	require.Len(t, report.Stages, len(want))
	for i, w := range want {
		st := report.Stages[i]
		require.Equal(t, i, st.Index, "Stages[%d].Index", i)
		require.Equal(t, w.command, st.Command, "Stages[%d].Command", i)
		require.Equal(t, w.condKind, st.Condition.Kind, "Stages[%d].Condition.Kind", i)
		require.Equal(t, w.dependsOn, st.Condition.DependsOn, "Stages[%d].Condition.DependsOn", i)
		require.Empty(t, st.Effects, "Stages[%d].Effects", i)
	}
}

func TestAnalyzeSkeletonUnsupportedSyntaxUnknown(t *testing.T) {
	// Background '&' is lexically valid but outside the L0 surface, so it must
	// surface as an unsupported_syntax unknown rather than a silent drop.
	req := ir.AnalyzeRequest{
		Command: "echo hi &",
		Context: &ir.AnalysisContext{CWD: workspaceCWD},
	}
	report, err := analyzer.Analyze(context.Background(), req)
	require.NoError(t, err)
	require.NoError(t, ir.ValidateReport(report))
	unk, ok := findUnknown(report.Unknowns, ir.UnknownUnsupportedSyntax)
	require.True(t, ok, "unknowns = %+v, want a unsupported_syntax unknown", reportUnknowns(report))
	require.True(t, unk.Blocking)
	require.Equal(t, ir.CoveragePartial, report.Analysis.Coverage)
	require.Equal(t, ir.CompletenessUnknown, report.Analysis.Completeness)
}

func TestAnalyzeSkeletonParseErrorUnknown(t *testing.T) {
	// A redirect missing its target is structurally invalid and must be
	// reported as a parse_error unknown rather than returned as a Go error.
	req := ir.AnalyzeRequest{
		Command: "echo hi >",
		Context: &ir.AnalysisContext{CWD: workspaceCWD},
	}
	report, err := analyzer.Analyze(context.Background(), req)
	require.NoError(t, err)
	require.NoError(t, ir.ValidateReport(report))
	unk, ok := findUnknown(report.Unknowns, ir.UnknownParseError)
	require.True(t, ok, "unknowns = %v, want a parse_error unknown", reportUnknowns(report))
	require.True(t, unk.Blocking)
	require.Equal(t, ir.CoveragePartial, report.Analysis.Coverage)
	require.Equal(t, ir.CompletenessUnknown, report.Analysis.Completeness)
}

func TestAnalyzeSkeletonCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := ir.AnalyzeRequest{
		Command: "echo hi",
		Context: &ir.AnalysisContext{CWD: workspaceCWD},
	}
	report, err := analyzer.Analyze(ctx, req)
	require.NoError(t, err, "cancellation is structured, not an error")
	require.NoError(t, ir.ValidateReport(report))
	unk, ok := findUnknown(report.Unknowns, ir.UnknownAnalysisTimeout)
	require.True(t, ok, "unknowns = %v, want an analysis_timeout unknown on cancellation", reportUnknowns(report))
	require.True(t, unk.Blocking)
}

func TestAnalyzeSkeletonRejectsEmptyCommand(t *testing.T) {
	req := ir.AnalyzeRequest{Command: "   "}
	_, err := analyzer.Analyze(context.Background(), req)
	require.Error(t, err)
	require.Contains(t, err.Error(), ir.ErrCodeEmptyCommand)
}

// Direct path extraction is the S01 orchestration slice: Analyze must call
// existing write/read/mutate extractors without expanders or new command rules.
func TestAnalyzeDirectPathEffects(t *testing.T) {
	cases := []struct {
		name       string
		command    string
		wantKind   ir.EffectKind
		wantRaw    string
		wantTarget string
	}{
		{
			name:       "write redirect",
			command:    "echo hi > output.txt",
			wantKind:   ir.EffectWrite,
			wantRaw:    "output.txt",
			wantTarget: "logical://workspace/output.txt",
		},
		{
			name:       "read cat",
			command:    "cat README.md",
			wantKind:   ir.EffectRead,
			wantRaw:    "README.md",
			wantTarget: "logical://workspace/README.md",
		},
		{
			name:       "delete rm",
			command:    "rm -rf build",
			wantKind:   ir.EffectDelete,
			wantRaw:    "build",
			wantTarget: "logical://workspace/build",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := ir.AnalyzeRequest{
				Command: tc.command,
				Context: &ir.AnalysisContext{CWD: workspaceCWD},
			}
			report, err := analyzer.Analyze(context.Background(), req)
			require.NoError(t, err)
			require.NoError(t, ir.ValidateReport(report))
			require.Len(t, report.Stages, 1)
			st := report.Stages[0]
			require.Equal(t, ir.ConditionAlways, st.Condition.Kind)
			require.NotEmpty(t, st.Effects, "Analyze must not return empty effects for %q", tc.command)

			var matched *ir.Effect
			for i := range st.Effects {
				ef := &st.Effects[i]
				if ef.Kind == tc.wantKind && ef.RawTarget == tc.wantRaw {
					matched = ef
					break
				}
			}
			require.NotNil(t, matched, "effects = %+v", st.Effects)
			require.Equal(t, tc.wantTarget, matched.Target)
			require.Equal(t, st.Condition, matched.Condition)
			require.Equal(t, 0, matched.Stage)
			require.NotEmpty(t, matched.Evidence)
			require.Equal(t, ir.EffectID(ir.SchemaVersion, *matched), matched.ID)
		})
	}
}

func TestAnalyzeDirectPathEffectsPreserveStageCondition(t *testing.T) {
	req := ir.AnalyzeRequest{
		Command: "true && echo hi > output.txt",
		Context: &ir.AnalysisContext{CWD: workspaceCWD},
	}
	report, err := analyzer.Analyze(context.Background(), req)
	require.NoError(t, err)
	require.NoError(t, ir.ValidateReport(report))
	require.GreaterOrEqual(t, len(report.Stages), 2)

	st := report.Stages[1]
	require.Equal(t, ir.ConditionOnSuccess, st.Condition.Kind)
	require.NotEmpty(t, st.Effects)
	for _, ef := range st.Effects {
		require.Equal(t, st.Condition, ef.Condition)
		require.Equal(t, ir.Conditional, ef.Certainty, "gated stage must downgrade certain writes")
		require.NotEmpty(t, ef.Evidence)
	}
}

func findUnknown(unknowns []ir.Unknown, code ir.UnknownCode) (ir.Unknown, bool) {
	for _, u := range unknowns {
		if u.Code == code {
			return u, true
		}
	}
	return ir.Unknown{}, false
}

func reportUnknowns(report ir.ImpactReport) []ir.Unknown {
	return report.Unknowns
}
