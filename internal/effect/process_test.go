package effect_test

import (
	"testing"

	"github.com/phaethix/cmdscope/internal/effect"
	"github.com/phaethix/cmdscope/internal/ir"
	"github.com/phaethix/cmdscope/internal/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessEffects(t *testing.T) {
	cases := []struct {
		name            string
		command         string
		wantEffects     int
		wantTarget      string
		wantUnsupported bool
	}{
		{
			name:        "go build",
			command:     "go build ./cmd/cmdscope",
			wantEffects: 1,
			wantTarget:  "go",
		},
		{
			name:            "unknown tool",
			command:         "unknown-tool --flag",
			wantEffects:     1,
			wantTarget:      "unknown-tool",
			wantUnsupported: true,
		},
		{
			name:        "echo builtin",
			command:     "echo hi",
			wantEffects: 0,
		},
		{
			name:        "echo via path still builtin",
			command:     "/usr/bin/echo x",
			wantEffects: 0,
		},
		{
			name:            "unknown via path",
			command:         "/usr/bin/unknown-tool",
			wantEffects:     1,
			wantTarget:      "/usr/bin/unknown-tool",
			wantUnsupported: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := parseSimple(t, tc.command)
			cond := ir.Condition{Kind: ir.ConditionAlways}
			effects, unknowns := effect.ExtractProcess(cmd, 0, cond)

			require.Len(t, effects, tc.wantEffects)
			if tc.wantEffects == 0 {
				assert.Empty(t, unknowns)
				return
			}

			ef := effects[0]
			assert.Equal(t, ir.EffectProcess, ef.Kind)
			assert.Equal(t, tc.wantTarget, ef.RawTarget)
			assert.Equal(t, tc.wantTarget, ef.Target)
			assert.Equal(t, ir.Certain, ef.Certainty)
			assert.Equal(t, ir.FromCommand, ef.Provenance)
			assert.Equal(t, 0, ef.Stage)
			assert.Equal(t, cond, ef.Condition)
			require.NotEmpty(t, ef.Evidence)
			assert.Equal(t, ir.EffectID(ir.SchemaVersion, ef), ef.ID)

			if tc.wantUnsupported {
				require.Len(t, unknowns, 1)
				unk := unknowns[0]
				assert.Equal(t, ir.UnknownUnsupportedCommand, unk.Code)
				assert.Equal(t, "stage:0", unk.Scope)
				assert.False(t, unk.Blocking)
				require.NotEmpty(t, unk.Evidence)
			} else {
				assert.Empty(t, unknowns)
			}
		})
	}
}

func TestProcessValidateReportSmoke(t *testing.T) {
	cmd := parseSimple(t, "go build ./cmd/cmdscope")
	cond := ir.Condition{Kind: ir.ConditionAlways}
	effects, unknowns := effect.ExtractProcess(cmd, 0, cond)
	require.Len(t, effects, 1)
	require.Empty(t, unknowns)

	report := ir.ImpactReport{
		SchemaVersion: ir.SchemaVersion,
		Command:       "go build ./cmd/cmdscope",
		Analysis: ir.AnalysisMeta{
			Coverage:     ir.CoverageComplete,
			Completeness: ir.CompletenessComplete,
			Limits:       []string{},
			Parser:       "shell",
		},
		Stages: []ir.Stage{{
			Index:        0,
			Command:      "go build ./cmd/cmdscope",
			Condition:    cond,
			Completeness: ir.CompletenessComplete,
			Effects:      effects,
		}},
		Unknowns: []ir.Unknown{},
		Flags:    []ir.Flag{},
	}
	require.NoError(t, ir.ValidateReport(report))
}

func TestProcessNilOrEmpty(t *testing.T) {
	effects, unknowns := effect.ExtractProcess(nil, 0, ir.Condition{Kind: ir.ConditionAlways})
	assert.Empty(t, effects)
	assert.Empty(t, unknowns)

	effects, unknowns = effect.ExtractProcess(&shell.SimpleCommand{}, 1, ir.Condition{Kind: ir.ConditionAlways})
	assert.Empty(t, effects)
	assert.Empty(t, unknowns)
}

func parseSimple(t *testing.T, command string) *shell.SimpleCommand {
	t.Helper()
	toks, err := shell.Lex(command)
	require.NoError(t, err)
	root, err := shell.Parse(toks)
	require.NoError(t, err)
	cmd, ok := root.(*shell.SimpleCommand)
	require.True(t, ok, "want *SimpleCommand, got %T", root)
	return cmd
}
