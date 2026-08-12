package analyzer

import (
	"slices"
	"strconv"

	"github.com/phaethix/runmark/internal/effect"
	"github.com/phaethix/runmark/internal/expand"
	"github.com/phaethix/runmark/internal/ir"
	"github.com/phaethix/runmark/internal/shell"
)

// Matches expand.maxScriptDepth so nested sh -c / script walks stop at the
// same bound expanders already enforce for npm/make active paths.
const maxExpandDepth = 8

type expandHit struct {
	result     expand.ExpansionResult
	provenance ir.Provenance
}

// extractDirectPathEffects runs the path-touching extractors only. Process and
// network extractors stay out so later tasks can extend without rewriting rules.
func extractDirectPathEffects(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string) ([]ir.Effect, []ir.Unknown) {
	if cmd == nil {
		return nil, nil
	}

	var effects []ir.Effect
	var unknowns []ir.Unknown
	appendExtract := func(efs []ir.Effect, unks []ir.Unknown) {
		effects = append(effects, efs...)
		unknowns = append(unknowns, unks...)
	}
	appendExtract(effect.ExtractWrite(cmd, stage, cond, cwd))
	appendExtract(effect.ExtractRead(cmd, stage, cond, cwd))
	appendExtract(effect.ExtractMutate(cmd, stage, cond, cwd))

	for i := range effects {
		effects[i].Certainty = ApplyConditionCertainty(effects[i].Certainty, cond)
		EnsureEffectHasEvidence(&effects[i])
	}
	SortEffects(effects)
	return effects, unknowns
}

// tryExpand returns the first expander that claims the command surface.
func tryExpand(cmd *shell.SimpleCommand, files map[string]string, stage int) (expandHit, bool) {
	if cmd == nil {
		return expandHit{}, false
	}
	if res := expand.ExpandNPM(cmd, files, stage); res.Applied {
		return expandHit{result: res, provenance: ir.FromScript}, true
	}
	if res := expand.ExpandPNPM(cmd, files, stage); res.Applied {
		return expandHit{result: res, provenance: ir.FromScript}, true
	}
	if res := expand.ExpandMake(cmd, files, stage); res.Applied {
		return expandHit{result: res, provenance: ir.FromWorkspaceFile}, true
	}
	if res := expand.ExpandShellScript(cmd, stage); res.Applied {
		return expandHit{result: res, provenance: ir.FromScript}, true
	}
	if res := expand.ExpandPython(cmd, stage); res.Applied {
		return expandHit{result: res, provenance: ir.FromScript}, true
	}
	return expandHit{}, false
}

// analyzeSimpleCommand expands (bounded) then extracts path effects. Expansion
// uses only the caller-supplied files map — never the host filesystem.
func analyzeSimpleCommand(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string, files map[string]string, depth int) (effects []ir.Effect, unknowns []ir.Unknown, limits []string) {
	if cmd == nil {
		return nil, nil, nil
	}
	if depth > maxExpandDepth {
		return nil, []ir.Unknown{{
			Code:     ir.UnknownExpansionLimit,
			Scope:    "stage:" + strconv.Itoa(stage),
			Message:  "script expansion exceeded max_script_depth:8",
			Evidence: []ir.Evidence{},
			Blocking: true,
		}}, []string{"max_script_depth:8"}
	}

	hit, applied := tryExpand(cmd, files, stage)
	outerEffects, outerUnknowns := extractDirectPathEffects(cmd, stage, cond, cwd)
	if !applied {
		return outerEffects, outerUnknowns, nil
	}

	effects = append(effects, outerEffects...)
	unknowns = append(unknowns, outerUnknowns...)
	unknowns = append(unknowns, hit.result.Unknowns...)
	limits = append(limits, hit.result.Limits...)

	for _, n := range hit.result.Nodes {
		child, ok := n.(*shell.SimpleCommand)
		if !ok {
			continue
		}
		childEffects, childUnknowns, childLimits := analyzeSimpleCommand(child, stage, cond, cwd, files, depth+1)
		annotateExpandedEffects(childEffects, hit.provenance, hit.result.Evidence)
		effects = append(effects, childEffects...)
		unknowns = append(unknowns, childUnknowns...)
		limits = append(limits, childLimits...)
	}
	return effects, unknowns, limits
}

// annotateExpandedEffects retags leaf effects so ImpactReport can tell command
// vs workspace/script derivation. Provenance is part of EffectID, so IDs are
// recomputed after the change.
func annotateExpandedEffects(effects []ir.Effect, provenance ir.Provenance, evidence []ir.Evidence) {
	for i := range effects {
		effects[i].Provenance = provenance
		if len(evidence) > 0 {
			merged := make([]ir.Evidence, 0, len(evidence)+len(effects[i].Evidence))
			merged = append(merged, evidence...)
			merged = append(merged, effects[i].Evidence...)
			effects[i].Evidence = merged
		}
		EnsureEffectHasEvidence(&effects[i])
		effects[i].ID = ir.EffectID(ir.SchemaVersion, effects[i])
	}
}

// fillStageEffects walks each stage's simple commands, runs bounded expansion,
// and attaches merged path effects. Non-simple nodes are skipped.
func fillStageEffects(stages []ir.Stage, shellStages []shell.Stage, cwd string, files map[string]string) (outStages []ir.Stage, unknowns []ir.Unknown, limits []string) {
	for i := range stages {
		if i >= len(shellStages) {
			break
		}
		cond := stages[i].Condition
		var effects []ir.Effect
		for _, n := range shellStages[i].Commands {
			cmd, ok := n.(*shell.SimpleCommand)
			if !ok {
				continue
			}
			efs, unks, lims := analyzeSimpleCommand(cmd, stages[i].Index, cond, cwd, files, 0)
			effects = append(effects, efs...)
			unknowns = append(unknowns, unks...)
			limits = append(limits, lims...)
		}
		if effects == nil {
			effects = []ir.Effect{}
		}
		SortEffects(effects)
		stages[i].Effects = effects
	}
	return stages, unknowns, limits
}

func mergeLimits(dst, extra []string) []string {
	for _, lim := range extra {
		if lim == "" || slices.Contains(dst, lim) {
			continue
		}
		dst = append(dst, lim)
	}
	if dst == nil {
		return []string{}
	}
	return dst
}

func filesFromContext(cf ContextFiles) map[string]string {
	if m, ok := cf.(*mapContextFiles); ok {
		return m.files
	}
	return map[string]string{}
}
