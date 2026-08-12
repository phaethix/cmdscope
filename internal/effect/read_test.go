package effect_test

import (
	"testing"

	"github.com/phaethix/runmark/internal/effect"
	"github.com/phaethix/runmark/internal/ir"
	"github.com/phaethix/runmark/internal/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadEffects(t *testing.T) {
	cases := []struct {
		name        string
		command     string
		cwd         string
		wantTargets []string // normalized targets in order
		wantRaws    []string
	}{
		{
			name:        "cat file",
			command:     "cat f",
			cwd:         "/ws",
			wantTargets: []string{"/ws/f"},
			wantRaws:    []string{"f"},
		},
		{
			name:        "cat options then files",
			command:     "cat -n a b",
			cwd:         "/ws",
			wantTargets: []string{"/ws/a", "/ws/b"},
			wantRaws:    []string{"a", "b"},
		},
		{
			name:        "head with -n arg",
			command:     "head -n 10 f",
			cwd:         "/ws",
			wantTargets: []string{"/ws/f"},
			wantRaws:    []string{"f"},
		},
		{
			name:        "grep pattern then file",
			command:     "grep PAT f",
			cwd:         "/ws",
			wantTargets: []string{"/ws/f"},
			wantRaws:    []string{"f"},
		},
		{
			name:        "grep -n pattern files",
			command:     "grep -n PAT a b",
			cwd:         "/ws",
			wantTargets: []string{"/ws/a", "/ws/b"},
			wantRaws:    []string{"a", "b"},
		},
		{
			name:        "grep only pattern no file",
			command:     "grep PAT",
			cwd:         "/ws",
			wantTargets: nil,
		},
		{
			name:        "not a read command",
			command:     "echo hi",
			cwd:         "/ws",
			wantTargets: nil,
		},
		{
			name:        "absolute operand ignores cwd join",
			command:     "cat /abs/x",
			cwd:         "/ws",
			wantTargets: []string{"/abs/x"},
			wantRaws:    []string{"/abs/x"},
		},
		{
			name:        "grep -e pattern then file",
			command:     "grep -e PAT f",
			cwd:         "/ws",
			wantTargets: []string{"/ws/f"},
			wantRaws:    []string{"f"},
		},
		{
			name:        "grep --regexp= pattern then file",
			command:     "grep --regexp=PAT f",
			cwd:         "/ws",
			wantTargets: []string{"/ws/f"},
			wantRaws:    []string{"f"},
		},
		{
			name:        "cat stdin dash",
			command:     "cat -",
			cwd:         "/ws",
			wantTargets: nil,
		},
		{
			name:        "grep pattern then stdin dash",
			command:     "grep PAT -",
			cwd:         "/ws",
			wantTargets: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := parseSimple(t, tc.command)
			cond := ir.Condition{Kind: ir.ConditionAlways}
			effects, unknowns := effect.ExtractRead(cmd, 0, cond, tc.cwd)
			require.Empty(t, unknowns)
			require.Len(t, effects, len(tc.wantTargets))
			for i, ef := range effects {
				assert.Equal(t, ir.EffectRead, ef.Kind)
				assert.Equal(t, tc.wantRaws[i], ef.RawTarget)
				assert.Equal(t, tc.wantTargets[i], ef.Target)
				assert.Equal(t, ir.Certain, ef.Certainty)
				assert.Equal(t, ir.FromCommand, ef.Provenance)
				assert.Equal(t, ir.EffectID(ir.SchemaVersion, ef), ef.ID)
				require.NotEmpty(t, ef.Evidence)
			}
		})
	}
}

func TestReadEffectsNil(t *testing.T) {
	effects, unknowns := effect.ExtractRead(nil, 0, ir.Condition{Kind: ir.ConditionAlways}, "/ws")
	assert.Empty(t, effects)
	assert.Empty(t, unknowns)
	effects, unknowns = effect.ExtractRead(&shell.SimpleCommand{}, 0, ir.Condition{Kind: ir.ConditionAlways}, "/ws")
	assert.Empty(t, effects)
	assert.Empty(t, unknowns)
}
