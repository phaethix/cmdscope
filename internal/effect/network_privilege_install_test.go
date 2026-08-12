package effect_test

import (
	"testing"

	"github.com/phaethix/runmark/internal/effect"
	"github.com/phaethix/runmark/internal/ir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetworkEffects(t *testing.T) {
	t.Run("curl url", func(t *testing.T) {
		cmd := parseSimple(t, "curl https://example.com/a")
		effects, unknowns := effect.ExtractNetwork(cmd, 0, ir.Condition{Kind: ir.ConditionAlways})
		require.Empty(t, unknowns)
		require.Len(t, effects, 1)
		assert.Equal(t, ir.EffectNetwork, effects[0].Kind)
		assert.Equal(t, "https://example.com/a", effects[0].RawTarget)
		assert.Equal(t, "https://example.com/a", effects[0].Target)
		assert.Equal(t, ir.Certain, effects[0].Certainty)
		assert.Equal(t, ir.EffectID(ir.SchemaVersion, effects[0]), effects[0].ID)
	})

	t.Run("wget with options", func(t *testing.T) {
		cmd := parseSimple(t, "wget -q https://example.com/x")
		effects, unknowns := effect.ExtractNetwork(cmd, 0, ir.Condition{Kind: ir.ConditionAlways})
		require.Empty(t, unknowns)
		require.Len(t, effects, 1)
		assert.Equal(t, "https://example.com/x", effects[0].Target)
	})

	t.Run("curl missing url", func(t *testing.T) {
		cmd := parseSimple(t, "curl -I")
		effects, unknowns := effect.ExtractNetwork(cmd, 0, ir.Condition{Kind: ir.ConditionAlways})
		assert.Empty(t, effects)
		require.Len(t, unknowns, 1)
		assert.Equal(t, ir.UnknownUnsupportedCommand, unknowns[0].Code)
	})

	t.Run("curl -O keeps url", func(t *testing.T) {
		cmd := parseSimple(t, "curl -O https://example.com/a")
		effects, unknowns := effect.ExtractNetwork(cmd, 0, ir.Condition{Kind: ir.ConditionAlways})
		require.Empty(t, unknowns)
		require.Len(t, effects, 1)
		assert.Equal(t, "https://example.com/a", effects[0].Target)
	})

	t.Run("curl --url", func(t *testing.T) {
		cmd := parseSimple(t, "curl --url https://example.com/b")
		effects, unknowns := effect.ExtractNetwork(cmd, 0, ir.Condition{Kind: ir.ConditionAlways})
		require.Empty(t, unknowns)
		require.Len(t, effects, 1)
		assert.Equal(t, "https://example.com/b", effects[0].Target)
	})

	t.Run("not curl", func(t *testing.T) {
		cmd := parseSimple(t, "echo hi")
		effects, unknowns := effect.ExtractNetwork(cmd, 0, ir.Condition{Kind: ir.ConditionAlways})
		assert.Empty(t, effects)
		assert.Empty(t, unknowns)
	})
}

func TestPrivilegeEffects(t *testing.T) {
	cond := ir.Condition{Kind: ir.ConditionAlways}

	t.Run("sudo", func(t *testing.T) {
		cmd := parseSimple(t, "sudo apt update")
		effects, unknowns := effect.ExtractPrivilege(cmd, 0, cond, "/ws")
		require.Empty(t, unknowns)
		require.Len(t, effects, 1)
		assert.Equal(t, ir.EffectPrivilege, effects[0].Kind)
		assert.Equal(t, "sudo", effects[0].Target)
		assert.Equal(t, ir.Certain, effects[0].Certainty)
	})

	t.Run("chmod file", func(t *testing.T) {
		cmd := parseSimple(t, "chmod 755 bin/app")
		effects, unknowns := effect.ExtractPrivilege(cmd, 0, cond, "/ws")
		require.Empty(t, unknowns)
		require.Len(t, effects, 2)
		assert.Equal(t, ir.EffectPrivilege, effects[0].Kind)
		assert.Equal(t, "/ws/bin/app", effects[0].Target)
		assert.Equal(t, ir.EffectWrite, effects[1].Kind)
		assert.Equal(t, "/ws/bin/app", effects[1].Target)
	})

	t.Run("chown file", func(t *testing.T) {
		cmd := parseSimple(t, "chown root:root a")
		effects, unknowns := effect.ExtractPrivilege(cmd, 0, cond, "/ws")
		require.Empty(t, unknowns)
		require.Len(t, effects, 2)
		assert.Equal(t, ir.EffectPrivilege, effects[0].Kind)
		assert.Equal(t, "/ws/a", effects[0].Target)
		assert.Equal(t, ir.EffectWrite, effects[1].Kind)
	})

	t.Run("chmod missing path", func(t *testing.T) {
		cmd := parseSimple(t, "chmod 644")
		effects, unknowns := effect.ExtractPrivilege(cmd, 0, cond, "/ws")
		assert.Empty(t, effects)
		require.Len(t, unknowns, 1)
		assert.Equal(t, ir.UnknownUnsupportedCommand, unknowns[0].Code)
	})
}

func TestInstallEffects(t *testing.T) {
	cond := ir.Condition{Kind: ir.ConditionAlways}

	t.Run("npm install packages", func(t *testing.T) {
		cmd := parseSimple(t, "npm install lodash")
		effects, unknowns := effect.ExtractInstall(cmd, 0, cond)
		require.Empty(t, unknowns)
		require.Len(t, effects, 2)
		assert.Equal(t, ir.EffectInstall, effects[0].Kind)
		assert.Equal(t, "lodash", effects[0].Target)
		assert.Equal(t, ir.Certain, effects[0].Certainty)
		assert.Equal(t, ir.EffectNetwork, effects[1].Kind)
		assert.Equal(t, "registry", effects[1].Target)
		assert.Equal(t, ir.Possible, effects[1].Certainty)
	})

	t.Run("npm i bare", func(t *testing.T) {
		cmd := parseSimple(t, "npm i")
		effects, unknowns := effect.ExtractInstall(cmd, 0, cond)
		require.Empty(t, unknowns)
		require.Len(t, effects, 2)
		assert.Equal(t, ".", effects[0].Target)
		assert.Equal(t, ir.EffectNetwork, effects[1].Kind)
		assert.Equal(t, ir.Possible, effects[1].Certainty)
	})

	t.Run("pnpm install", func(t *testing.T) {
		cmd := parseSimple(t, "pnpm install")
		effects, unknowns := effect.ExtractInstall(cmd, 0, cond)
		require.Empty(t, unknowns)
		require.Len(t, effects, 2)
		assert.Equal(t, ir.EffectInstall, effects[0].Kind)
	})

	t.Run("npm run not install", func(t *testing.T) {
		cmd := parseSimple(t, "npm run build")
		effects, unknowns := effect.ExtractInstall(cmd, 0, cond)
		assert.Empty(t, effects)
		assert.Empty(t, unknowns)
	})
}
