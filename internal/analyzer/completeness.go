package analyzer

import (
	"strconv"

	"github.com/phaethix/runmark/internal/ir"
)

// StageCompleteness answers whether the impact set for one stage may still be
// missing pieces. Blocking unknowns (or effects with certainty=unknown) force
// unknown; non-blocking unknowns or possible-only effects force partial.
func StageCompleteness(effects []ir.Effect, stageUnknowns []ir.Unknown) ir.Completeness {
	for _, u := range stageUnknowns {
		if u.Blocking {
			return ir.CompletenessUnknown
		}
	}
	for _, ef := range effects {
		if ef.Certainty == ir.CertaintyUnknown {
			return ir.CompletenessUnknown
		}
	}
	for _, u := range stageUnknowns {
		if !u.Blocking {
			return ir.CompletenessPartial
		}
	}
	for _, ef := range effects {
		if ef.Certainty == ir.Possible {
			return ir.CompletenessPartial
		}
	}
	return ir.CompletenessComplete
}

// ReportCompleteness takes the most conservative stage result and folds
// report-scoped unknowns. unknown beats partial beats complete.
func ReportCompleteness(stages []ir.Stage, allUnknowns []ir.Unknown) ir.Completeness {
	worst := ir.CompletenessComplete
	for _, st := range stages {
		c := StageCompleteness(st.Effects, unknownsForStage(allUnknowns, st.Index))
		worst = worseCompleteness(worst, c)
	}
	for _, u := range allUnknowns {
		if u.Scope != "report" {
			continue
		}
		if u.Blocking {
			return ir.CompletenessUnknown
		}
		worst = worseCompleteness(worst, ir.CompletenessPartial)
	}
	return worst
}

func unknownsForStage(all []ir.Unknown, index int) []ir.Unknown {
	want := "stage:" + strconv.Itoa(index)
	var out []ir.Unknown
	for _, u := range all {
		if u.Scope == want {
			out = append(out, u)
		}
	}
	return out
}

func worseCompleteness(a, b ir.Completeness) ir.Completeness {
	if completenessRank(b) > completenessRank(a) {
		return b
	}
	return a
}

func completenessRank(c ir.Completeness) int {
	switch c {
	case ir.CompletenessUnknown:
		return 2
	case ir.CompletenessPartial:
		return 1
	default:
		return 0
	}
}
