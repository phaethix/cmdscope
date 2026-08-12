package analyzer

import (
	"github.com/phaethix/runmark/internal/effect"
	"github.com/phaethix/runmark/internal/ir"
	"github.com/phaethix/runmark/internal/shell"
)

// extractDirectPathEffects runs the path-touching extractors only. Expand and
// process/network/privilege extractors stay out of this slice so S02/S03 can
// extend orchestration without rewriting command rules.
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

// fillStageEffects walks each stage's simple commands and attaches direct path
// effects. Non-simple nodes (e.g. raw command substitution stages) are skipped
// so we never invent paths; later tasks own their unknowns.
func fillStageEffects(stages []ir.Stage, shellStages []shell.Stage, cwd string) ([]ir.Stage, []ir.Unknown) {
	var unknowns []ir.Unknown
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
			efs, unks := extractDirectPathEffects(cmd, stages[i].Index, cond, cwd)
			effects = append(effects, efs...)
			unknowns = append(unknowns, unks...)
		}
		if effects == nil {
			effects = []ir.Effect{}
		}
		SortEffects(effects)
		stages[i].Effects = effects
	}
	return stages, unknowns
}
