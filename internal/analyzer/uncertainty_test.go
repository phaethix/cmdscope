package analyzer_test

import (
	"testing"

	"github.com/phaethix/runmark/internal/analyzer"
	"github.com/phaethix/runmark/internal/ir"
	"github.com/phaethix/runmark/internal/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnknownCommandSubstitution(t *testing.T) {
	unknowns := collect(t, `echo $(cat secret.txt)`, nil)
	require.NotEmpty(t, unknowns)
	assert.True(t, hasCode(unknowns, ir.UnknownCommandSubstitution))
	for _, u := range unknowns {
		if u.Code == ir.UnknownCommandSubstitution {
			assert.Equal(t, "stage:0", u.Scope)
			assert.False(t, u.Blocking)
			require.NotEmpty(t, u.Evidence)
		}
	}
}

func TestUnknownGlob(t *testing.T) {
	unknowns := collect(t, `rm dist/*.js`, nil)
	require.NotEmpty(t, unknowns)
	assert.True(t, hasCode(unknowns, ir.UnknownGlobRuntimeDependent))
	assert.False(t, hasCode(unknowns, ir.UnknownEnvMissing))
}

func TestUnknownEnvMissing(t *testing.T) {
	unknowns := collect(t, `rm $OUT`, nil)
	require.NotEmpty(t, unknowns)
	assert.True(t, hasCode(unknowns, ir.UnknownEnvMissing))
	assert.Equal(t, "stage:0", unknowns[0].Scope)
	assert.False(t, unknowns[0].Blocking)
}

func TestUnknownEnvProvided(t *testing.T) {
	unknowns := collect(t, `rm $OUT`, map[string]string{"OUT": "/tmp/x"})
	assert.False(t, hasCode(unknowns, ir.UnknownEnvMissing))
}

func TestUnknownSubstitutionSkipsInnerEnv(t *testing.T) {
	unknowns := collect(t, `echo $(echo $OUT) $HOME`, nil)
	assert.True(t, hasCode(unknowns, ir.UnknownCommandSubstitution))
	// Inner $OUT lives inside substitution; only outer $HOME is env_missing.
	var envMsgs []string
	for _, u := range unknowns {
		if u.Code == ir.UnknownEnvMissing {
			envMsgs = append(envMsgs, u.Message)
		}
	}
	require.Len(t, envMsgs, 1)
	assert.Contains(t, envMsgs[0], "HOME")
	assert.NotContains(t, envMsgs[0], "OUT")
}

func TestUnknownSpecialParamNotGlob(t *testing.T) {
	for _, cmd := range []string{`rm $?`, `echo $*`, `echo $@`} {
		t.Run(cmd, func(t *testing.T) {
			unknowns := collect(t, cmd, nil)
			assert.False(t, hasCode(unknowns, ir.UnknownGlobRuntimeDependent))
		})
	}
}

func collect(t *testing.T, command string, env map[string]string) []ir.Unknown {
	t.Helper()
	toks, err := shell.Lex(command)
	require.NoError(t, err)
	root, err := shell.Parse(toks)
	require.NoError(t, err)
	stages := shell.SplitStages(root)
	return analyzer.CollectUncertainties(stages, env)
}

func hasCode(unknowns []ir.Unknown, code ir.UnknownCode) bool {
	for _, u := range unknowns {
		if u.Code == code {
			return true
		}
	}
	return false
}
