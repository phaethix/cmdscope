package shell

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stageExpect describes the observable shape of one Stage in a SplitStages result.
type stageExpect struct {
	index     int
	kind      ConditionKind
	dependsOn int
	nCommands int
}

// assertStages verifies the shape of a []Stage returned by SplitStages.
// It checks index order, condition, and command count per stage, but not the
// exact identity of command nodes (which is intentional: the contract here is
// stage partitioning + condition wiring, not node contents).
func assertStages(t *testing.T, got []Stage, want []stageExpect) {
	t.Helper()
	require.Len(t, got, len(want), "got: %+v", got)
	for i, s := range got {
		w := want[i]
		assert.Equal(t, w.index, s.Index, "stage[%d].Index", i)
		assert.Equal(t, w.kind, s.Condition.Kind, "stage[%d] Condition.Kind", i)
		assert.Equal(t, w.dependsOn, s.Condition.DependsOn, "stage[%d] Condition.DependsOn", i)
		assert.Len(t, s.Commands, w.nCommands, "stage[%d] commands", i)
	}
}

func TestStageSemisplitToAlways(t *testing.T) {
	n := parseNode(t, "echo a ; echo b")
	assertStages(t, SplitStages(n), []stageExpect{
		{index: 1, kind: ConditionAlways, dependsOn: 0, nCommands: 1},
		{index: 2, kind: ConditionAlways, dependsOn: 0, nCommands: 1},
	})
}

func TestStageAndWiresOnSuccess(t *testing.T) {
	n := parseNode(t, "a && b")
	assertStages(t, SplitStages(n), []stageExpect{
		{index: 1, kind: ConditionAlways, dependsOn: 0, nCommands: 1},
		{index: 2, kind: ConditionOnSuccess, dependsOn: 1, nCommands: 1},
	})
}

func TestStageOrWiresOnFailure(t *testing.T) {
	n := parseNode(t, "a || b")
	assertStages(t, SplitStages(n), []stageExpect{
		{index: 1, kind: ConditionAlways, dependsOn: 0, nCommands: 1},
		{index: 2, kind: ConditionOnFailure, dependsOn: 1, nCommands: 1},
	})
}

func TestStageChainAndOr(t *testing.T) {
	// a && b || c ; d | e
	// precedence: && > ||, left-assoc => ((a && b) || c); then ; splits
	// a=1 always, b=2 on_success(1), c=3 on_failure(2), d|e=4 always (2 cmds)
	n := parseNode(t, "a && b || c; d | e")
	assertStages(t, SplitStages(n), []stageExpect{
		{index: 1, kind: ConditionAlways, dependsOn: 0, nCommands: 1},
		{index: 2, kind: ConditionOnSuccess, dependsOn: 1, nCommands: 1},
		{index: 3, kind: ConditionOnFailure, dependsOn: 2, nCommands: 1},
		{index: 4, kind: ConditionAlways, dependsOn: 0, nCommands: 2},
	})
}

func TestStageSubshellGlobalIndex(t *testing.T) {
	// (a; b) && c
	// subshell body expands on the global index: a=1 always, b=2 always,
	// then c=3 on_success(2) because the whole subshell is the && LHS.
	n := parseNode(t, "(a; b) && c")
	assertStages(t, SplitStages(n), []stageExpect{
		{index: 1, kind: ConditionAlways, dependsOn: 0, nCommands: 1},
		{index: 2, kind: ConditionAlways, dependsOn: 0, nCommands: 1},
		{index: 3, kind: ConditionOnSuccess, dependsOn: 2, nCommands: 1},
	})
}

func TestStageRedirectKeepsSingleStage(t *testing.T) {
	n := parseNode(t, "echo hi > out")
	assertStages(t, SplitStages(n), []stageExpect{
		{index: 1, kind: ConditionAlways, dependsOn: 0, nCommands: 1},
	})
}

func TestStageNilRoot(t *testing.T) {
	st := SplitStages(nil)
	require.Empty(t, st)
}
