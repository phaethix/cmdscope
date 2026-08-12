package analyzer_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"testing"

	"github.com/phaethix/runmark/internal/analyzer"
	"github.com/phaethix/runmark/internal/ir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderEffectIDMatchesValidateFormula(t *testing.T) {
	ef := ir.Effect{
		Kind:       ir.EffectWrite,
		RawTarget:  "out.txt",
		Target:     "/ws/out.txt",
		Stage:      0,
		Provenance: ir.FromCommand,
		Condition:  ir.Condition{Kind: ir.ConditionAlways, DependsOn: 0},
	}
	got := analyzer.EffectID(ir.SchemaVersion, ef)
	require.Equal(t, wantEffectID(ir.SchemaVersion, ef), got)
	require.True(t, len(got) > len("sha256:"))
	assert.Equal(t, "sha256:", got[:7])
}

func TestOrderSortEffects(t *testing.T) {
	effects := []ir.Effect{
		{Stage: 1, Kind: ir.EffectWrite, Target: "b", ID: "sha256:2"},
		{Stage: 0, Kind: ir.EffectRead, Target: "z", ID: "sha256:9"},
		{Stage: 0, Kind: ir.EffectRead, Target: "a", ID: "sha256:1"},
		{Stage: 0, Kind: ir.EffectWrite, Target: "a", ID: "sha256:3"},
		{Stage: 1, Kind: ir.EffectWrite, Target: "b", ID: "sha256:1"},
	}
	// Shuffle then sort twice — order must be identical (stability + determinism).
	r := rand.New(rand.NewPCG(42, 1))
	r.Shuffle(len(effects), func(i, j int) { effects[i], effects[j] = effects[j], effects[i] })

	analyzer.SortEffects(effects)
	first := slices.Clone(effects)
	analyzer.SortEffects(effects)
	require.Equal(t, first, effects)

	require.Equal(t, []int{0, 0, 0, 1, 1}, stagesOf(effects))
	assert.Equal(t, ir.EffectRead, effects[0].Kind)
	assert.Equal(t, "a", effects[0].Target)
	assert.Equal(t, ir.EffectRead, effects[1].Kind)
	assert.Equal(t, "z", effects[1].Target)
	assert.Equal(t, ir.EffectWrite, effects[2].Kind)
	assert.Equal(t, "sha256:1", effects[3].ID)
	assert.Equal(t, "sha256:2", effects[4].ID)
}

func TestOrderSortUnknowns(t *testing.T) {
	unknowns := []ir.Unknown{
		{Code: ir.UnknownParseError, Scope: "stage:1", Message: "b"},
		{Code: ir.UnknownContextMissing, Scope: "report", Message: "a"},
		{Code: ir.UnknownParseError, Scope: "stage:0", Message: "a"},
		{Code: ir.UnknownParseError, Scope: "stage:1", Message: "a"},
	}
	r := rand.New(rand.NewPCG(7, 1))
	r.Shuffle(len(unknowns), func(i, j int) { unknowns[i], unknowns[j] = unknowns[j], unknowns[i] })

	analyzer.SortUnknowns(unknowns)
	first := slices.Clone(unknowns)
	analyzer.SortUnknowns(unknowns)
	require.Equal(t, first, unknowns)

	assert.Equal(t, ir.UnknownContextMissing, unknowns[0].Code)
	assert.Equal(t, "stage:0", unknowns[1].Scope)
	assert.Equal(t, "a", unknowns[2].Message)
	assert.Equal(t, "b", unknowns[3].Message)
}

func stagesOf(effects []ir.Effect) []int {
	out := make([]int, len(effects))
	for i, ef := range effects {
		out[i] = ef.Stage
	}
	return out
}

// wantEffectID mirrors ir.ValidateReport's private formula so analyzer IDs
// stay honest without importing unexported ir helpers.
func wantEffectID(schemaVersion string, ef ir.Effect) string {
	canon := fmt.Sprintf(`{"kind":%q,"depends_on":%d}`, string(ef.Condition.Kind), ef.Condition.DependsOn)
	payload := schemaVersion + strconv.Itoa(ef.Stage) + string(ef.Kind) + ef.RawTarget + ef.Target + canon + string(ef.Provenance)
	sum := sha256.Sum256([]byte(payload))
	return "sha256:" + hex.EncodeToString(sum[:])
}
