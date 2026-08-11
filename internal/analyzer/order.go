package analyzer

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strconv"

	"github.com/phaethix/cmdscope/internal/ir"
)

// EffectID returns the stable effect identifier that ValidateReport recomputes.
// The payload layout is fixed so gold cases and adapters can rely on the same
// string across hosts; changing it is a contract break.
func EffectID(schemaVersion string, ef ir.Effect) string {
	canon := fmt.Sprintf(`{"kind":%q,"depends_on":%d}`, string(ef.Condition.Kind), ef.Condition.DependsOn)
	payload := schemaVersion + strconv.Itoa(ef.Stage) + string(ef.Kind) + ef.RawTarget + ef.Target + canon + string(ef.Provenance)
	sum := sha256.Sum256([]byte(payload))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// SortEffects orders by stage, kind, target, then id. Stable so equal keys keep
// relative order and repeated runs cannot reshuffle reports.
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

// SortUnknowns orders by code, scope, then message for the same determinism
// reason as SortEffects.
func SortUnknowns(unknowns []ir.Unknown) {
	slices.SortStableFunc(unknowns, func(a, b ir.Unknown) int {
		return cmp.Or(
			cmp.Compare(string(a.Code), string(b.Code)),
			cmp.Compare(a.Scope, b.Scope),
			cmp.Compare(a.Message, b.Message),
		)
	})
}
