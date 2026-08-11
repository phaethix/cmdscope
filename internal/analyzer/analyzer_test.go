package analyzer_test

import (
	"context"
	"strings"
	"testing"

	"github.com/phaethix/cmdscope/internal/analyzer"
	"github.com/phaethix/cmdscope/internal/ir"
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
	if err != nil {
		t.Fatalf("Analyze() error = %v, want nil", err)
	}
	if err := ir.ValidateReport(report); err != nil {
		t.Fatalf("ValidateReport(report) = %v, want nil", err)
	}
	if report.Command != "echo hi" {
		t.Errorf("report.Command = %q, want %q", report.Command, "echo hi")
	}
	if report.CWD != workspaceCWD {
		t.Errorf("report.CWD = %q, want %q", report.CWD, workspaceCWD)
	}
	if len(report.Stages) != 1 {
		t.Fatalf("len(Stages) = %d, want 1", len(report.Stages))
	}
	st := report.Stages[0]
	if st.Index != 0 {
		t.Errorf("Stages[0].Index = %d, want 0", st.Index)
	}
	if st.Command != "echo hi" {
		t.Errorf("Stages[0].Command = %q, want %q", st.Command, "echo hi")
	}
	if st.Condition.Kind != ir.ConditionAlways || st.Condition.DependsOn != 0 {
		t.Errorf("Stages[0].Condition = %+v, want always/0", st.Condition)
	}
	if len(st.Effects) != 0 {
		t.Errorf("Stages[0].Effects = %d, want 0 (no effects in skeleton)", len(st.Effects))
	}
	if len(report.Unknowns) != 0 {
		t.Errorf("len(Unknowns) = %d, want 0", len(report.Unknowns))
	}
	if report.Analysis.Coverage != ir.CoverageComplete {
		t.Errorf("Analysis.Coverage = %q, want complete", report.Analysis.Coverage)
	}
	if report.Analysis.Completeness != ir.CompletenessComplete {
		t.Errorf("Analysis.Completeness = %q, want complete", report.Analysis.Completeness)
	}
}

func TestAnalyzeSkeletonCompoundStages(t *testing.T) {
	const cmd = "a && echo b || c; d | e"
	req := ir.AnalyzeRequest{
		Command: cmd,
		Context: &ir.AnalysisContext{CWD: workspaceCWD},
	}
	report, err := analyzer.Analyze(context.Background(), req)
	if err != nil {
		t.Fatalf("Analyze() error = %v, want nil", err)
	}
	if err := ir.ValidateReport(report); err != nil {
		t.Fatalf("ValidateReport(report) = %v, want nil", err)
	}
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
	if len(report.Stages) != len(want) {
		t.Fatalf("len(Stages) = %d, want %d", len(report.Stages), len(want))
	}
	for i, w := range want {
		st := report.Stages[i]
		if st.Index != i {
			t.Errorf("Stages[%d].Index = %d, want %d", i, st.Index, i)
		}
		if st.Command != w.command {
			t.Errorf("Stages[%d].Command = %q, want %q", i, st.Command, w.command)
		}
		if st.Condition.Kind != w.condKind || st.Condition.DependsOn != w.dependsOn {
			t.Errorf("Stages[%d].Condition = %+v, want Kind=%s depends=%d", i, st.Condition, w.condKind, w.dependsOn)
		}
		if len(st.Effects) != 0 {
			t.Errorf("Stages[%d].Effects = %d, want 0", i, len(st.Effects))
		}
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
	if err != nil {
		t.Fatalf("Analyze() error = %v, want nil", err)
	}
	if err := ir.ValidateReport(report); err != nil {
		t.Fatalf("ValidateReport(report) = %v, want nil", err)
	}
	unk, ok := findUnknown(report.Unknowns, ir.UnknownUnsupportedSyntax)
	if !ok {
		t.Fatalf("unknowns = %+v, want a unsupported_syntax unknown", reportUnknowns(report))
	}
	if !unk.Blocking {
		t.Errorf("unsupported_syntax unknown Blocking = false, want true")
	}
	if report.Analysis.Coverage != ir.CoveragePartial {
		t.Errorf("Analysis.Coverage = %q, want partial", report.Analysis.Coverage)
	}
	if report.Analysis.Completeness != ir.CompletenessUnknown {
		t.Errorf("Analysis.Completeness = %q, want unknown", report.Analysis.Completeness)
	}
}

func TestAnalyzeSkeletonParseErrorUnknown(t *testing.T) {
	// A redirect missing its target is structurally invalid and must be
	// reported as a parse_error unknown rather than returned as a Go error.
	req := ir.AnalyzeRequest{
		Command: "echo hi >",
		Context: &ir.AnalysisContext{CWD: workspaceCWD},
	}
	report, err := analyzer.Analyze(context.Background(), req)
	if err != nil {
		t.Fatalf("Analyze() error = %v, want nil", err)
	}
	if err := ir.ValidateReport(report); err != nil {
		t.Fatalf("ValidateReport(report) = %v, want nil", err)
	}
	unk, ok := findUnknown(report.Unknowns, ir.UnknownParseError)
	if !ok {
		t.Fatalf("unknowns = %v, want a parse_error unknown", reportUnknowns(report))
	}
	if !unk.Blocking {
		t.Errorf("parse_error unknown Blocking = false, want true")
	}
	if report.Analysis.Coverage != ir.CoveragePartial {
		t.Errorf("Analysis.Coverage = %q, want partial", report.Analysis.Coverage)
	}
	if report.Analysis.Completeness != ir.CompletenessUnknown {
		t.Errorf("Analysis.Completeness = %q, want unknown", report.Analysis.Completeness)
	}
}

func TestAnalyzeSkeletonCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := ir.AnalyzeRequest{
		Command: "echo hi",
		Context: &ir.AnalysisContext{CWD: workspaceCWD},
	}
	report, err := analyzer.Analyze(ctx, req)
	if err != nil {
		t.Fatalf("Analyze() error = %v, want nil (cancellation is structured, not an error)", err)
	}
	if err := ir.ValidateReport(report); err != nil {
		t.Fatalf("ValidateReport(report) = %v, want nil", err)
	}
	unk, ok := findUnknown(report.Unknowns, ir.UnknownAnalysisTimeout)
	if !ok {
		t.Fatalf("unknowns = %v, want an analysis_timeout unknown on cancellation", reportUnknowns(report))
	}
	if !unk.Blocking {
		t.Errorf("analysis_timeout unknown Blocking = false, want true")
	}
}

func TestAnalyzeSkeletonRejectsEmptyCommand(t *testing.T) {
	req := ir.AnalyzeRequest{Command: "   "}
	_, err := analyzer.Analyze(context.Background(), req)
	if err == nil {
		t.Fatal("Analyze() err = nil, want a request validation error")
	}
	if !strings.Contains(err.Error(), ir.ErrCodeEmptyCommand) {
		t.Errorf("Analyze() err = %v, want empty_command code", err)
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
