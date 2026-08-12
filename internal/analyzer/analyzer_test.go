package analyzer_test

import (
	"context"
	"strings"
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

func TestAnalyzeBoundedExpansion(t *testing.T) {
	pkgWithDelete := `{"scripts":{"build":"rm -rf dist"}}`

	t.Run("npm run build with package.json", func(t *testing.T) {
		report := analyzeExpansion(t, "npm run build", map[string]string{
			"package.json": pkgWithDelete,
		})
		ef := requireEffect(t, report, ir.EffectDelete, "dist")
		require.Equal(t, ir.FromScript, ef.Provenance)
		require.True(t, hasEvidenceSource(ef.Evidence, ir.EvidenceWorkspaceFile) ||
			hasEvidenceFieldPrefix(ef.Evidence, "scripts."),
			"workspace/script evidence must survive expansion: %+v", ef.Evidence)
	})

	t.Run("npm run build without package.json", func(t *testing.T) {
		report := analyzeExpansion(t, "npm run build", nil)
		require.True(t, hasReportUnknown(report, ir.UnknownContextMissing))
		require.False(t, hasPathEffect(report), "must not invent path effects without package.json")
	})

	t.Run("pnpm run build with package.json", func(t *testing.T) {
		report := analyzeExpansion(t, "pnpm run build", map[string]string{
			"package.json": pkgWithDelete,
		})
		ef := requireEffect(t, report, ir.EffectDelete, "dist")
		require.Equal(t, ir.FromScript, ef.Provenance)
	})

	t.Run("make deploy static", func(t *testing.T) {
		report := analyzeExpansion(t, "make deploy", map[string]string{
			"Makefile": "deploy:\n\trm -rf build\n",
		})
		ef := requireEffect(t, report, ir.EffectDelete, "build")
		require.Equal(t, ir.FromWorkspaceFile, ef.Provenance)
		require.True(t, hasEvidenceSource(ef.Evidence, ir.EvidenceWorkspaceFile),
			"Makefile evidence must survive: %+v", ef.Evidence)
	})

	t.Run("make deploy dynamic makefile", func(t *testing.T) {
		report := analyzeExpansion(t, "make deploy", map[string]string{
			"Makefile": "include other.mk\ndeploy:\n\trm -rf build\n",
		})
		require.True(t, hasReportUnknown(report, ir.UnknownUnsupportedCommand))
		require.False(t, hasPathEffect(report), "dynamic Makefile must not invent recipe path effects")
	})

	t.Run("sh -c rm", func(t *testing.T) {
		report := analyzeExpansion(t, `sh -c 'rm -rf build'`, nil)
		ef := requireEffect(t, report, ir.EffectDelete, "build")
		require.Equal(t, ir.FromScript, ef.Provenance)
		require.NotEmpty(t, ef.Evidence)
	})

	t.Run("python -c opaque", func(t *testing.T) {
		report := analyzeExpansion(t, `python3 -c 'open("x").write("y")'`, nil)
		require.True(t, hasReportUnknown(report, ir.UnknownInterpreterDynamicCode))
		require.False(t, hasPathEffect(report), "python -c must not invent file effects")
	})
}

func TestAnalyzeEndToEndSynthesis(t *testing.T) {
	t.Run("curl pipe sh", func(t *testing.T) {
		report := analyzeExpansion(t, "curl https://example.com/install.sh | sh", nil)
		require.True(t, hasReportUnknown(report, ir.UnknownRemoteContent))
		unk, _ := findUnknown(report.Unknowns, ir.UnknownRemoteContent)
		require.True(t, unk.Blocking)
		require.Equal(t, ir.CompletenessUnknown, report.Analysis.Completeness)
		require.Contains(t, report.Flags, ir.FlagExternalNetwork)
		require.Contains(t, report.Flags, ir.FlagRemoteContent)
		require.NotNil(t, findEffectKind(report, ir.EffectExecuteRemote))
		require.NotNil(t, findEffectKind(report, ir.EffectNetwork))
		assertStableIDs(t, report)
	})

	t.Run("rm env glob", func(t *testing.T) {
		report := analyzeExpansion(t, `rm "$OUT"/*.tmp`, nil)
		require.True(t, hasReportUnknown(report, ir.UnknownEnvMissing) ||
			hasReportUnknown(report, ir.UnknownGlobRuntimeDependent),
			"unknowns=%+v", report.Unknowns)
		require.NotEqual(t, ir.CompletenessComplete, report.Analysis.Completeness)
		assertStableIDs(t, report)
	})

	t.Run("command substitution", func(t *testing.T) {
		report := analyzeExpansion(t, "echo $(cat secret.txt)", nil)
		require.True(t, hasReportUnknown(report, ir.UnknownCommandSubstitution))
		require.Equal(t, ir.CompletenessPartial, report.Analysis.Completeness)
		assertStableIDs(t, report)
	})

	t.Run("unsupported tool", func(t *testing.T) {
		report := analyzeExpansion(t, "unsupported-tool --flag", nil)
		require.True(t, hasReportUnknown(report, ir.UnknownUnsupportedCommand))
		require.NotEmpty(t, report.Stages[0].Effects, "must not emit complete+empty for unsupported tools")
		require.NotNil(t, findEffectKind(report, ir.EffectProcess))
		require.NotEqual(t, ir.CompletenessComplete, report.Analysis.Completeness,
			"unsupported must not look like a complete empty report")
		assertStableIDs(t, report)
	})

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		req := ir.AnalyzeRequest{
			Command: "echo hi > out.txt",
			Context: &ir.AnalysisContext{CWD: workspaceCWD},
		}
		report, err := analyzer.Analyze(ctx, req)
		require.NoError(t, err)
		require.NoError(t, ir.ValidateReport(report))
		require.True(t, hasReportUnknown(report, ir.UnknownAnalysisTimeout))
		require.Contains(t, report.Flags, ir.FlagAnalysisTimeout)
	})

	t.Run("stable order across runs", func(t *testing.T) {
		cmd := "curl https://example.com/a.sh | sh"
		a := analyzeExpansion(t, cmd, nil)
		b := analyzeExpansion(t, cmd, nil)
		require.Equal(t, effectIDs(a), effectIDs(b))
		require.Equal(t, unknownCodes(a), unknownCodes(b))
	})
}

func hasPathEffect(report ir.ImpactReport) bool {
	for _, st := range report.Stages {
		for _, ef := range st.Effects {
			switch ef.Kind {
			case ir.EffectRead, ir.EffectWrite, ir.EffectDelete:
				return true
			}
		}
	}
	return false
}

func findEffectKind(report ir.ImpactReport, kind ir.EffectKind) *ir.Effect {
	for i := range report.Stages {
		for j := range report.Stages[i].Effects {
			ef := &report.Stages[i].Effects[j]
			if ef.Kind == kind {
				return ef
			}
		}
	}
	return nil
}

func assertStableIDs(t *testing.T, report ir.ImpactReport) {
	t.Helper()
	for _, st := range report.Stages {
		for _, ef := range st.Effects {
			require.Equal(t, ir.EffectID(ir.SchemaVersion, ef), ef.ID)
		}
	}
}

func effectIDs(report ir.ImpactReport) []string {
	var ids []string
	for _, st := range report.Stages {
		for _, ef := range st.Effects {
			ids = append(ids, ef.ID)
		}
	}
	return ids
}

func unknownCodes(report ir.ImpactReport) []ir.UnknownCode {
	var codes []ir.UnknownCode
	for _, u := range report.Unknowns {
		codes = append(codes, u.Code)
	}
	return codes
}

func analyzeExpansion(t *testing.T, command string, files map[string]string) ir.ImpactReport {
	t.Helper()
	req := ir.AnalyzeRequest{
		Command: command,
		Context: &ir.AnalysisContext{CWD: workspaceCWD, Files: files},
	}
	report, err := analyzer.Analyze(context.Background(), req)
	require.NoError(t, err)
	require.NoError(t, ir.ValidateReport(report))
	require.NotEmpty(t, report.Stages)
	return report
}

func requireEffect(t *testing.T, report ir.ImpactReport, kind ir.EffectKind, raw string) ir.Effect {
	t.Helper()
	for _, st := range report.Stages {
		for _, ef := range st.Effects {
			if ef.Kind == kind && ef.RawTarget == raw {
				require.NotEmpty(t, ef.Evidence)
				require.Equal(t, ir.EffectID(ir.SchemaVersion, ef), ef.ID)
				return ef
			}
		}
	}
	require.FailNow(t, "effect not found", "kind=%s raw=%s stages=%+v", kind, raw, report.Stages)
	return ir.Effect{}
}

func hasReportUnknown(report ir.ImpactReport, code ir.UnknownCode) bool {
	_, ok := findUnknown(report.Unknowns, code)
	return ok
}

func hasEvidenceSource(evidence []ir.Evidence, source ir.EvidenceSource) bool {
	for _, ev := range evidence {
		if ev.Source == source {
			return true
		}
	}
	return false
}

func hasEvidenceFieldPrefix(evidence []ir.Evidence, prefix string) bool {
	for _, ev := range evidence {
		if strings.HasPrefix(ev.Field, prefix) {
			return true
		}
	}
	return false
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
