package analyzer

import (
	"context"

	"github.com/phaethix/runmark/internal/ir"
	"github.com/phaethix/runmark/internal/shell"
)

// parserNote identifies which syntax layer produced the report's analysis
// meta. It is a stable label consumed by CLI/adapter diagnostics, not a
// version string.
const parserNote = "shell"

// Analyze is the entry point of the local deterministic analysis pipeline. It
// runs validate → normalize → lex → parse → stage → bounded expansion →
// direct path extraction. Remaining extractors and report synthesis land in
// later tasks.
//
// Only request-level validation failures are returned as Go errors. Every
// lex/parse failure is turned into a structured unknown on the report so that
// a partial result is always serializable and never silently dropped. Analyze
// never executes the command, never touches the network, and never reads files
// beyond the explicit request context.
func Analyze(ctx context.Context, req ir.AnalyzeRequest) (ir.ImpactReport, error) {
	if err := ir.ValidateRequest(req); err != nil {
		return ir.ImpactReport{}, err
	}

	command := normalizeCommand(req.Command)
	if err := ctx.Err(); err != nil {
		return timeoutReport(req, command), nil
	}

	tokens, lexErr := shell.Lex(command)
	if err := ctx.Err(); err != nil {
		return timeoutReport(req, command), nil
	}

	root, parseErr := shell.Parse(tokens)
	if err := ctx.Err(); err != nil {
		return timeoutReport(req, command), nil
	}

	cwd := ""
	if req.Context != nil {
		cwd = req.Context.CWD
	}
	cf, err := NewContextFiles(req.Context)
	if err != nil {
		return ir.ImpactReport{}, err
	}
	files := filesFromContext(cf)

	shellStages := shell.SplitStages(root)
	report := baseReport(req, command)
	report.Stages = mapStages(command, shellStages)
	var extractUnknowns []ir.Unknown
	var expandLimits []string
	report.Stages, extractUnknowns, expandLimits = fillStageEffects(report.Stages, shellStages, cwd, files)
	report.Unknowns = append(report.Unknowns, extractUnknowns...)
	report.Analysis.Limits = mergeLimits(report.Analysis.Limits, expandLimits)
	report = attachShellFailure(report, lexErr, parseErr)
	return report, nil
}

// baseReport initializes a fully report-shaped ImpactReport with every
// required slice set to an empty, non-nil slice so it serializes as [] and
// passes ValidateReport. Coverage/completeness default to complete and are
// downgraded by attachShellFailure when something could not be walked.
func baseReport(req ir.AnalyzeRequest, command string) ir.ImpactReport {
	report := ir.ImpactReport{
		SchemaVersion: ir.SchemaVersion,
		Command:       command,
		Analysis: ir.AnalysisMeta{
			Coverage:     ir.CoverageComplete,
			Completeness: ir.CompletenessComplete,
			Limits:       []string{},
			Parser:       parserNote,
		},
		Stages:   []ir.Stage{},
		Unknowns: []ir.Unknown{},
		Flags:    []ir.Flag{},
	}
	if req.Context != nil {
		report.CWD = req.Context.CWD
	}
	return report
}

// timeoutReport is returned when the caller's context is already cancelled
// before (or across) the pipeline. Cancellation is surfaced as a structured
// blocking analysis_timeout unknown plus its flag, never as a Go error, so the
// result remains serializable.
func timeoutReport(req ir.AnalyzeRequest, command string) ir.ImpactReport {
	report := baseReport(req, command)
	report.Analysis.Coverage = ir.CoverageMinimal
	report.Analysis.Completeness = ir.CompletenessUnknown
	report.Unknowns = append(report.Unknowns, ir.Unknown{
		Code:     ir.UnknownAnalysisTimeout,
		Scope:    "report",
		Message:  "analysis context was cancelled before the pipeline could complete",
		Evidence: []ir.Evidence{},
		Blocking: true,
	})
	report.Flags = append(report.Flags, ir.FlagAnalysisTimeout)
	return report
}

// mapStages lowers the shell stage graph (1-based indices) onto the IR stage
// contract (0-based, globally consecutive). Effects start empty and are filled
// by fillStageEffects after the stage graph is projected.
func mapStages(command string, stages []shell.Stage) []ir.Stage {
	out := make([]ir.Stage, 0, len(stages))
	for i, st := range stages {
		out = append(out, ir.Stage{
			Index:        i,
			Command:      stageCommand(command, st),
			Condition:    mapCondition(st.Condition),
			Completeness: ir.CompletenessComplete,
			Effects:      []ir.Effect{},
		})
	}
	return out
}

func mapCondition(c shell.Condition) ir.Condition {
	return ir.Condition{
		Kind:      ir.ConditionKind(c.Kind),
		DependsOn: irDependsOn(c.Kind, c.DependsOn),
	}
}

// irDependsOn re-bases the 1-based shell dependency onto the 0-based IR stage
// numbering. An "always" stage has no dependency and stays 0.
func irDependsOn(kind shell.ConditionKind, shellDependsOn int) int {
	if kind == shell.ConditionAlways {
		return 0
	}
	return shellDependsOn - 1
}

// stageCommand reconstructs the original text of a stage by cutting the
// contiguous byte range spanned by its commands out of the normalized command.
// It keeps evidence pointing at real source bytes rather than a re-derivation.
func stageCommand(command string, st shell.Stage) string {
	if len(st.Commands) == 0 {
		return ""
	}
	start, end := spanOf(st.Commands[0])
	for _, n := range st.Commands[1:] {
		s, e := spanOf(n)
		start = min(start, s)
		end = max(end, e)
	}
	if start < 0 || end < start || end > len(command) {
		return ""
	}
	return command[start:end]
}

// spanOf returns the byte range of any shell AST node the stage graph can
// hold. Every node carries its source span, so this is a pure projection; a
// node kind we do not know should never appear in a stage.
func spanOf(n shell.Node) (int, int) {
	switch v := n.(type) {
	case *shell.Word:
		return v.Start, v.End
	case *shell.Sequence:
		return v.Start, v.End
	case *shell.Binary:
		return v.Start, v.End
	case *shell.Pipeline:
		return v.Start, v.End
	case *shell.SimpleCommand:
		return v.Start, v.End
	case *shell.Subshell:
		return v.Start, v.End
	case *shell.CommandSubstitution:
		return v.Start, v.End
	default:
		return 0, 0
	}
}

// attachShellFailure turns a lex or parse failure into a structured blocking
// unknown instead of a Go error. Unsupported syntax (background '&', '|&',
// here-doc) maps to unsupported_syntax; every other malformed command maps to
// parse_error. Any failure downgrades the aggregate coverage/completeness so
// a reader cannot mistake a partial result for a whole one.
func attachShellFailure(report ir.ImpactReport, lexErr error, parseErr error) ir.ImpactReport {
	if lexErr == nil && parseErr == nil {
		return report
	}
	code := ir.UnknownParseError
	msg := "the command could not be fully parsed"
	if parseErr != nil && shell.IsUnsupportedSyntax(parseErr) {
		code = ir.UnknownUnsupportedSyntax
		if parseErr.Error() != "" {
			msg = parseErr.Error()
		}
	} else if lexErr != nil {
		msg = lexErr.Error()
	} else if parseErr != nil {
		msg = parseErr.Error()
	}
	report.Unknowns = append(report.Unknowns, ir.Unknown{
		Code:     code,
		Scope:    "report",
		Message:  msg,
		Evidence: []ir.Evidence{},
		Blocking: true,
	})
	report.Analysis.Coverage = ir.CoveragePartial
	report.Analysis.Completeness = ir.CompletenessUnknown
	return report
}
