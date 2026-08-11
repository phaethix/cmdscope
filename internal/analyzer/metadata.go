package analyzer

import (
	"cmp"
	"slices"

	"github.com/phaethix/cmdscope/internal/ir"
)

// ApplyConditionCertainty downgrades certain → conditional when a stage gate
// applies. Confidence about habit-only or unknown targets is left alone; the
// gate does not make those claims stronger or weaker.
func ApplyConditionCertainty(c ir.Certainty, cond ir.Condition) ir.Certainty {
	if c == ir.Certain && cond.Kind != ir.ConditionAlways {
		return ir.Conditional
	}
	return c
}

// DefaultProvenance is the command-layer provenance used when an extractor
// has not yet been attributed to workspace/script/context expansion.
func DefaultProvenance() ir.Provenance {
	return ir.FromCommand
}

// AggregateFlags collects factual report flags from effects and unknowns.
// It never invents a risk_level: callers that need policy conclusions must
// derive them outside core analysis.
func AggregateFlags(effects []ir.Effect, unknowns []ir.Unknown, extra ...ir.Flag) []ir.Flag {
	seen := map[ir.Flag]bool{}
	var out []ir.Flag
	add := func(f ir.Flag) {
		if seen[f] {
			return
		}
		seen[f] = true
		out = append(out, f)
	}
	for _, ef := range effects {
		switch ef.Kind {
		case ir.EffectDelete:
			add(ir.FlagDestructive)
		case ir.EffectNetwork:
			add(ir.FlagExternalNetwork)
		case ir.EffectPrivilege:
			add(ir.FlagPrivilegeChange)
		case ir.EffectExecuteRemote:
			add(ir.FlagExternalNetwork)
			add(ir.FlagRemoteContent)
		}
	}
	for _, u := range unknowns {
		switch u.Code {
		case ir.UnknownRemoteContent:
			add(ir.FlagRemoteContent)
		case ir.UnknownInterpreterDynamicCode:
			add(ir.FlagOpaqueScript)
		case ir.UnknownContextMissing:
			add(ir.FlagContextMissing)
		case ir.UnknownUnsupportedCommand, ir.UnknownUnsupportedSyntax:
			add(ir.FlagUnsupported)
		case ir.UnknownAnalysisTimeout:
			add(ir.FlagAnalysisTimeout)
		}
	}
	for _, f := range extra {
		add(f)
	}
	slices.SortFunc(out, func(a, b ir.Flag) int {
		return cmp.Compare(string(a), string(b))
	})
	if out == nil {
		return []ir.Flag{}
	}
	return out
}
