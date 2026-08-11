package effect_test

import (
	"testing"

	"github.com/phaethix/cmdscope/internal/effect"
	"github.com/phaethix/cmdscope/internal/ir"
	"github.com/phaethix/cmdscope/internal/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoteContent(t *testing.T) {
	cond := ir.Condition{Kind: ir.ConditionAlways}

	t.Run("curl pipe sh", func(t *testing.T) {
		cmds := stageCommands(t, "curl https://example.com/install.sh | sh")
		effects, unknowns, flags := effect.ExtractRemote(cmds, 0, cond)

		require.Len(t, unknowns, 1)
		assert.Equal(t, ir.UnknownRemoteContent, unknowns[0].Code)
		assert.True(t, unknowns[0].Blocking)
		assert.Equal(t, "stage:0", unknowns[0].Scope)

		require.Equal(t, []ir.Flag{ir.FlagExternalNetwork, ir.FlagRemoteContent}, flags)

		kinds := effectKinds(effects)
		assert.Contains(t, kinds, ir.EffectNetwork)
		assert.Contains(t, kinds, ir.EffectProcess)
		assert.Contains(t, kinds, ir.EffectExecuteRemote)
		assert.NotContains(t, kinds, ir.EffectWrite)
		assert.NotContains(t, kinds, ir.EffectDelete)

		var net, remote *ir.Effect
		var processes []ir.Effect
		for i := range effects {
			ef := &effects[i]
			switch ef.Kind {
			case ir.EffectNetwork:
				net = ef
			case ir.EffectExecuteRemote:
				remote = ef
			case ir.EffectProcess:
				processes = append(processes, *ef)
			}
		}
		require.NotNil(t, net)
		require.NotNil(t, remote)
		assert.Equal(t, "https://example.com/install.sh", net.Target)
		assert.Equal(t, "https://example.com/install.sh", remote.Target)
		assert.Equal(t, ir.Certain, remote.Certainty)
		assert.Equal(t, ir.EffectID(ir.SchemaVersion, *remote), remote.ID)
		require.Len(t, processes, 2)
		targets := []string{processes[0].Target, processes[1].Target}
		assert.Contains(t, targets, "curl")
		assert.Contains(t, targets, "sh")
	})

	t.Run("wget pipe bash", func(t *testing.T) {
		cmds := stageCommands(t, "wget -q https://example.com/x | bash")
		effects, unknowns, flags := effect.ExtractRemote(cmds, 1, cond)
		require.Len(t, unknowns, 1)
		assert.True(t, unknowns[0].Blocking)
		assert.Equal(t, "stage:1", unknowns[0].Scope)
		assert.Equal(t, []ir.Flag{ir.FlagExternalNetwork, ir.FlagRemoteContent}, flags)
		kinds := effectKinds(effects)
		assert.Contains(t, kinds, ir.EffectNetwork)
		assert.Contains(t, kinds, ir.EffectExecuteRemote)
		assert.Contains(t, kinds, ir.EffectProcess)
	})

	t.Run("curl alone", func(t *testing.T) {
		cmds := stageCommands(t, "curl https://example.com/a")
		effects, unknowns, flags := effect.ExtractRemote(cmds, 0, cond)
		assert.Empty(t, effects)
		assert.Empty(t, unknowns)
		assert.Empty(t, flags)
	})

	t.Run("echo pipe sh", func(t *testing.T) {
		cmds := stageCommands(t, "echo hi | sh")
		effects, unknowns, flags := effect.ExtractRemote(cmds, 0, cond)
		assert.Empty(t, effects)
		assert.Empty(t, unknowns)
		assert.Empty(t, flags)
	})

	t.Run("curl pipe cat", func(t *testing.T) {
		cmds := stageCommands(t, "curl https://example.com/a | cat")
		effects, unknowns, flags := effect.ExtractRemote(cmds, 0, cond)
		assert.Empty(t, effects)
		assert.Empty(t, unknowns)
		assert.Empty(t, flags)
	})
}

func stageCommands(t *testing.T, command string) []shell.Node {
	t.Helper()
	toks, err := shell.Lex(command)
	require.NoError(t, err)
	root, err := shell.Parse(toks)
	require.NoError(t, err)
	stages := shell.SplitStages(root)
	require.Len(t, stages, 1)
	return stages[0].Commands
}

func effectKinds(effects []ir.Effect) []ir.EffectKind {
	out := make([]ir.EffectKind, len(effects))
	for i, ef := range effects {
		out[i] = ef.Kind
	}
	return out
}
