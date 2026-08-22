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

// directExtractorEffects runs every command-level extractor against one
// command layer. ExtractProcess and ExtractRemote stay in the synthesis
// stage instead: process must fire even when no family extractor claims the
// command, and remote detection spans two pipeline members, not one command.
func directExtractorEffects(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string) ([]ir.Effect, []ir.Unknown, []ir.Flag) {
	var effects []ir.Effect
	var unknowns []ir.Unknown
	var flags []ir.Flag
	add := func(efs []ir.Effect, unks []ir.Unknown) {
		effects = append(effects, efs...)
		unknowns = append(unknowns, unks...)
	}
	addFlags := func(efs []ir.Effect, unks []ir.Unknown, fls []ir.Flag) {
		add(efs, unks)
		flags = append(flags, fls...)
	}
	add(effect.ExtractWrite(cmd, stage, cond, cwd))
	add(effect.ExtractRead(cmd, stage, cond, cwd))
	add(effect.ExtractMutate(cmd, stage, cond, cwd))
	addFlags(effect.ExtractGit(cmd, stage, cond, cwd))
	add(effect.ExtractMisc(cmd, stage, cond, cwd))
	add(effect.ExtractSed(cmd, stage, cond, cwd))
	add(effect.ExtractFind(cmd, stage, cond, cwd))
	add(effect.ExtractArchive(cmd, stage, cond, cwd))
	add(effect.ExtractPrivilege(cmd, stage, cond, cwd))
	add(effect.ExtractNetwork(cmd, stage, cond))
	add(effect.ExtractInstall(cmd, stage, cond))
	add(effect.ExtractXargs(cmd, stage, cond))
	return effects, unknowns, flags
}

// extractDirectEffects substitutes caller-supplied env values first, then
// runs the command-level extractors over the command and every sudo/doas/env
// wrapper layer beneath it. Layers dedupe by EffectID so nested wrappers do
// not multiply the same fact; redirects stay on the outermost layer because
// StripWrapperPrefix deliberately does not carry them inward.
func extractDirectEffects(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string, env map[string]string) ([]ir.Effect, []ir.Unknown, []ir.Flag) {
	if cmd == nil {
		return nil, nil, nil
	}

	substituted, replaced := effect.SubstituteEnvWords(cmd, env)

	var effects []ir.Effect
	var unknowns []ir.Unknown
	var flags []ir.Flag
	seen := map[string]bool{}

	collectLayer := func(layer *shell.SimpleCommand) {
		efs, unks, fls := directExtractorEffects(layer, stage, cond, cwd)
		for i := range efs {
			efs[i].Certainty = ApplyConditionCertainty(efs[i].Certainty, cond)
			EnsureEffectHasEvidence(&efs[i])
			if seen[efs[i].ID] {
				continue
			}
			seen[efs[i].ID] = true
			effects = append(effects, efs[i])
		}
		unknowns = append(unknowns, unks...)
		flags = append(flags, fls...)
	}

	collectLayer(substituted)
	for layer, ok := effect.StripWrapperPrefix(substituted); ok; layer, ok = effect.StripWrapperPrefix(layer) {
		collectLayer(layer)
	}
	annotateCallerContext(effects, replaced)
	SortEffects(effects)
	return effects, unknowns, flags
}

// annotateCallerContext retags effects derived from words whose $refs the
// caller actually resolved: those targets exist only because caller context
// supplied values, so provenance moves off FromCommand and the pre-
// substitution text joins the evidence trail. Provenance feeds EffectID, so
// retagged ids are recomputed.
func annotateCallerContext(effects []ir.Effect, replaced map[int]string) {
	if len(replaced) == 0 {
		return
	}
	for i := range effects {
		ef := &effects[i]
		retagged := false
		for _, ev := range ef.Evidence {
			if ev.StartByte == nil {
				continue
			}
			orig, ok := replaced[*ev.StartByte]
			if !ok {
				continue
			}
			ef.Provenance = ir.FromCallerContext
			ef.Evidence = append(ef.Evidence, ir.Evidence{
				Source:    ir.EvidenceCallerContext,
				Snippet:   orig,
				StartByte: ev.StartByte,
				EndByte:   ev.EndByte,
			})
			retagged = true
			break
		}
		if retagged {
			ef.ID = ir.EffectID(ir.SchemaVersion, *ef)
		}
	}
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

// analyzeSimpleCommand expands (bounded) then extracts direct effects.
// Expansion uses only the caller-supplied files map — never the host
// filesystem.
func analyzeSimpleCommand(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string, files map[string]string, env map[string]string, depth int) (effects []ir.Effect, unknowns []ir.Unknown, limits []string, flags []ir.Flag) {
	if cmd == nil {
		return nil, nil, nil, nil
	}
	if depth > maxExpandDepth {
		return nil, []ir.Unknown{{
			Code:     ir.UnknownExpansionLimit,
			Scope:    "stage:" + strconv.Itoa(stage),
			Message:  "script expansion exceeded max_script_depth:8",
			Evidence: []ir.Evidence{},
			Blocking: true,
		}}, []string{"max_script_depth:8"}, nil
	}

	hit, applied := tryExpand(cmd, files, stage)
	outerEffects, outerUnknowns, outerFlags := extractDirectEffects(cmd, stage, cond, cwd, env)
	if !applied {
		return outerEffects, outerUnknowns, nil, outerFlags
	}

	effects = append(effects, outerEffects...)
	unknowns = append(unknowns, outerUnknowns...)
	flags = append(flags, outerFlags...)
	unknowns = append(unknowns, hit.result.Unknowns...)
	limits = append(limits, hit.result.Limits...)

	for _, n := range hit.result.Nodes {
		child, ok := n.(*shell.SimpleCommand)
		if !ok {
			continue
		}
		childEffects, childUnknowns, childLimits, childFlags := analyzeSimpleCommand(child, stage, cond, cwd, files, env, depth+1)
		annotateExpandedEffects(childEffects, hit.provenance, hit.result.Evidence)
		effects = append(effects, childEffects...)
		unknowns = append(unknowns, childUnknowns...)
		limits = append(limits, childLimits...)
		flags = append(flags, childFlags...)
	}
	return effects, unknowns, limits, flags
}

// fillStageEffects resolves per-stage working directories (cd tracking),
// then walks each stage's simple commands through bounded expansion and
// direct extraction, merging effects, flags, unknowns, and limits. Non-simple
// nodes are skipped.
func fillStageEffects(stages []ir.Stage, shellStages []shell.Stage, root string, files map[string]string, env map[string]string) (outStages []ir.Stage, unknowns []ir.Unknown, limits []string, flags []ir.Flag) {
	stageCWDs, cwdUnknowns := resolveStageCWDs(root, shellStages, env)
	unknowns = append(unknowns, cwdUnknowns...)

	for i := range stages {
		if i >= len(shellStages) || i >= len(stageCWDs) {
			break
		}
		cond := stages[i].Condition
		var effects []ir.Effect
		for _, n := range shellStages[i].Commands {
			cmd, ok := n.(*shell.SimpleCommand)
			if !ok {
				continue
			}
			efs, unks, lims, fls := analyzeSimpleCommand(cmd, stages[i].Index, cond, stageCWDs[i], files, env, 0)
			effects = append(effects, efs...)
			unknowns = append(unknowns, unks...)
			limits = append(limits, lims...)
			flags = append(flags, fls...)
		}
		if effects == nil {
			effects = []ir.Effect{}
		}
		SortEffects(effects)
		stages[i].Effects = effects
	}
	return stages, unknowns, limits, flags
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
