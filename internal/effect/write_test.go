package effect_test

import (
	"testing"

	"github.com/phaethix/cmdscope/internal/effect"
	"github.com/phaethix/cmdscope/internal/ir"
	"github.com/phaethix/cmdscope/internal/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteEffects(t *testing.T) {
	cases := []struct {
		name        string
		command     string
		cwd         string
		wantRaws    []string
		wantTargets []string
		wantCert    []ir.Certainty
	}{
		{
			name:        "redirect truncate",
			command:     "echo hi > output.txt",
			cwd:         "/ws",
			wantRaws:    []string{"output.txt"},
			wantTargets: []string{"/ws/output.txt"},
			wantCert:    []ir.Certainty{ir.Certain},
		},
		{
			name:        "redirect append",
			command:     "echo hi >> output.txt",
			cwd:         "/ws",
			wantRaws:    []string{"output.txt"},
			wantTargets: []string{"/ws/output.txt"},
			wantCert:    []ir.Certainty{ir.Conditional},
		},
		{
			name:        "mkdir -p",
			command:     "mkdir -p dist",
			cwd:         "/ws",
			wantRaws:    []string{"dist"},
			wantTargets: []string{"/ws/dist"},
			wantCert:    []ir.Certainty{ir.Certain},
		},
		{
			name:        "mkdir multiple",
			command:     "mkdir a b",
			cwd:         "/ws",
			wantRaws:    []string{"a", "b"},
			wantTargets: []string{"/ws/a", "/ws/b"},
			wantCert:    []ir.Certainty{ir.Certain, ir.Certain},
		},
		{
			name:     "input redirect ignored",
			command:  "cat < input.txt",
			cwd:      "/ws",
			wantRaws: nil,
		},
		{
			name:     "redirect to stdin dash",
			command:  "echo hi > -",
			cwd:      "/ws",
			wantRaws: nil,
		},
		{
			name:     "not mkdir and no redirect",
			command:  "echo hi",
			cwd:      "/ws",
			wantRaws: nil,
		},
		{
			name:        "mkdir -m swallows mode",
			command:     "mkdir -m 755 dist",
			cwd:         "/ws",
			wantRaws:    []string{"dist"},
			wantTargets: []string{"/ws/dist"},
			wantCert:    []ir.Certainty{ir.Certain},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := parseSimple(t, tc.command)
			cond := ir.Condition{Kind: ir.ConditionAlways}
			effects, unknowns := effect.ExtractWrite(cmd, 0, cond, tc.cwd)
			require.Empty(t, unknowns)
			require.Len(t, effects, len(tc.wantRaws))
			for i, ef := range effects {
				assert.Equal(t, ir.EffectWrite, ef.Kind)
				assert.Equal(t, tc.wantRaws[i], ef.RawTarget)
				assert.Equal(t, tc.wantTargets[i], ef.Target)
				assert.Equal(t, tc.wantCert[i], ef.Certainty)
				assert.Equal(t, ir.FromCommand, ef.Provenance)
				assert.Equal(t, ir.EffectID(ir.SchemaVersion, ef), ef.ID)
				require.NotEmpty(t, ef.Evidence)
			}
		})
	}
}

func TestWriteEffectsNil(t *testing.T) {
	effects, unknowns := effect.ExtractWrite(nil, 0, ir.Condition{Kind: ir.ConditionAlways}, "/ws")
	assert.Empty(t, effects)
	assert.Empty(t, unknowns)
	effects, unknowns = effect.ExtractWrite(&shell.SimpleCommand{}, 0, ir.Condition{Kind: ir.ConditionAlways}, "/ws")
	assert.Empty(t, effects)
	assert.Empty(t, unknowns)
}
