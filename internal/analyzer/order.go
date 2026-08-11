package analyzer

import (
	"cmp"
	"slices"

	"github.com/phaethix/cmdscope/internal/ir"
)

// EffectID is a thin alias of ir.EffectID so analyzer callers and extractors
// share one formula without effect importing this package (cycle risk).
func EffectID(schemaVersion string, ef ir.Effect) string {
	return ir.EffectID(schemaVersion, ef)
}

// SortEffects is stable so equal keys keep relative order and repeated runs
// cannot reshuffle reports.
func SortEffects(effects []ir.Effect) {
	slices.SortStableFunc(effects, func(a, b ir.Effect) int {
		return cmp.Or(
			cmp.Compare(a.Stage, b.Stage),
			cmp.Compare(string(a.Kind), string(b.Kind)),
			cmp.Compare(a.Target, b.Target),
			cmp.Compare(a.ID, b.ID),
		)
	})
}

// SortUnknowns uses the same determinism rationale as SortEffects.
func SortUnknowns(unknowns []ir.Unknown) {
	slices.SortStableFunc(unknowns, func(a, b ir.Unknown) int {
		return cmp.Or(
			cmp.Compare(string(a.Code), string(b.Code)),
			cmp.Compare(a.Scope, b.Scope),
			cmp.Compare(a.Message, b.Message),
		)
	})
}
