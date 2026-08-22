package analyzer

import (
	"slices"

	"github.com/phaethix/runmark/internal/effect"
	"github.com/phaethix/runmark/internal/ir"
	"github.com/phaethix/runmark/internal/shell"
)

// synthesizeReport folds remote/process extraction, uncertainty collection,
// flags, deterministic ordering, and completeness onto a path/expand report.
// It does not re-parse the command or read the host filesystem.
func synthesizeReport(report ir.ImpactReport, shellStages []shell.Stage, env map[string]string, extractFlags []ir.Flag) ir.ImpactReport {
	report.Unknowns = append(report.Unknowns, CollectUncertainties(shellStages, env)...)

	var extraFlags []ir.Flag
	for i := range report.Stages {
		if i >= len(shellStages) {
			break
		}
		cond := report.Stages[i].Condition
		stage := report.Stages[i].Index

		remoteEffects, remoteUnknowns, remoteFlags := effect.ExtractRemote(shellStages[i].Commands, stage, cond)
		for j := range remoteEffects {
			remoteEffects[j].Certainty = ApplyConditionCertainty(remoteEffects[j].Certainty, cond)
			EnsureEffectHasEvidence(&remoteEffects[j])
			remoteEffects[j].ID = ir.EffectID(ir.SchemaVersion, remoteEffects[j])
		}
		report.Stages[i].Effects = mergeEffectsByID(report.Stages[i].Effects, remoteEffects)
		report.Unknowns = append(report.Unknowns, remoteUnknowns...)
		extraFlags = append(extraFlags, remoteFlags...)

		for _, n := range shellStages[i].Commands {
			cmd, ok := n.(*shell.SimpleCommand)
			if !ok {
				continue
			}
			procEffects, procUnknowns := effect.ExtractProcess(cmd, stage, cond)
			for j := range procEffects {
				procEffects[j].Certainty = ApplyConditionCertainty(procEffects[j].Certainty, cond)
				EnsureEffectHasEvidence(&procEffects[j])
				procEffects[j].ID = ir.EffectID(ir.SchemaVersion, procEffects[j])
			}
			report.Stages[i].Effects = mergeEffectsByID(report.Stages[i].Effects, procEffects)
			report.Unknowns = append(report.Unknowns, procUnknowns...)
		}

		SortEffects(report.Stages[i].Effects)
	}

	SortUnknowns(report.Unknowns)

	allEffects := flattenEffects(report.Stages)
	// slices.Concat, not append: extractFlags belongs to the caller, and
	// appending into its backing array would corrupt it.
	report.Flags = AggregateFlags(allEffects, report.Unknowns, slices.Concat(extractFlags, extraFlags)...)

	for i := range report.Stages {
		report.Stages[i].Completeness = StageCompleteness(
			report.Stages[i].Effects,
			unknownsForStage(report.Unknowns, report.Stages[i].Index),
		)
	}
	report.Analysis.Completeness = ReportCompleteness(report.Stages, report.Unknowns)
	// An unsupported command means its effects were never enumerated, so the
	// report must not claim complete coverage over the command surface.
	if report.Analysis.Coverage == ir.CoverageComplete {
		for _, u := range report.Unknowns {
			if u.Code == ir.UnknownUnsupportedCommand {
				report.Analysis.Coverage = ir.CoveragePartial
				break
			}
		}
	}
	return report
}

func mergeEffectsByID(dst, extra []ir.Effect) []ir.Effect {
	seen := make(map[string]bool, len(dst))
	for _, ef := range dst {
		seen[ef.ID] = true
	}
	for _, ef := range extra {
		if seen[ef.ID] {
			continue
		}
		seen[ef.ID] = true
		dst = append(dst, ef)
	}
	if dst == nil {
		return []ir.Effect{}
	}
	return dst
}

func flattenEffects(stages []ir.Stage) []ir.Effect {
	var out []ir.Effect
	for _, st := range stages {
		out = append(out, st.Effects...)
	}
	return out
}

func contextEnv(ctx *ir.AnalysisContext) map[string]string {
	if ctx == nil || ctx.Env == nil {
		return map[string]string{}
	}
	return ctx.Env
}

// attachTimeout preserves already-proven stages/effects and records cancellation
// as a structured unknown instead of replacing the report with an empty shell.
func attachTimeout(report ir.ImpactReport) ir.ImpactReport {
	for _, u := range report.Unknowns {
		if u.Code == ir.UnknownAnalysisTimeout {
			return report
		}
	}
	report.Unknowns = append(report.Unknowns, ir.Unknown{
		Code:     ir.UnknownAnalysisTimeout,
		Scope:    "report",
		Message:  "analysis context was cancelled before the pipeline could complete",
		Evidence: []ir.Evidence{},
		Blocking: true,
	})
	report.Flags = AggregateFlags(flattenEffects(report.Stages), report.Unknowns)
	report.Analysis.Coverage = ir.CoverageMinimal
	report.Analysis.Completeness = ir.CompletenessUnknown
	SortUnknowns(report.Unknowns)
	return report
}
