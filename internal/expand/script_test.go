package expand_test

import (
	"testing"

	"github.com/phaethix/runmark/internal/expand"
	"github.com/phaethix/runmark/internal/ir"
	"github.com/phaethix/runmark/internal/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShellScriptLiteral(t *testing.T) {
	res := expand.ExpandShellScript(parseSimple(t, `sh -c 'echo hi'`), 0)
	require.True(t, res.Applied)
	require.Empty(t, res.Unknowns)
	require.NotEmpty(t, res.Nodes)
	echo, ok := res.Nodes[0].(*shell.SimpleCommand)
	require.True(t, ok)
	require.NotEmpty(t, echo.Words)
	assert.Equal(t, "echo", echo.Words[0].Text)
	require.NotEmpty(t, res.Evidence)
	assert.Equal(t, ir.EvidenceCommand, res.Evidence[0].Source)
}

func TestShellScriptBashLiteral(t *testing.T) {
	res := expand.ExpandShellScript(parseSimple(t, `bash -c "rm tmp/x"`), 1)
	require.True(t, res.Applied)
	require.Empty(t, res.Unknowns)
	require.NotEmpty(t, res.Nodes)
	rm, ok := res.Nodes[0].(*shell.SimpleCommand)
	require.True(t, ok)
	assert.Equal(t, "rm", rm.Words[0].Text)
}

func TestShellScriptSingleQuotedWithDollar(t *testing.T) {
	res := expand.ExpandShellScript(parseSimple(t, `sh -c 'echo $HOME'`), 0)
	require.True(t, res.Applied)
	require.Empty(t, res.Unknowns)
	require.NotEmpty(t, res.Nodes)
	echo, ok := res.Nodes[0].(*shell.SimpleCommand)
	require.True(t, ok)
	assert.Equal(t, "echo", echo.Words[0].Text)
}

func TestShellScriptDynamic(t *testing.T) {
	res := expand.ExpandShellScript(parseSimple(t, `sh -c "$SCRIPT"`), 0)
	require.True(t, res.Applied)
	require.True(t, hasUnknown(res.Unknowns, ir.UnknownInterpreterDynamicCode))
	assert.True(t, findUnknown(res.Unknowns, ir.UnknownInterpreterDynamicCode).Blocking)
	assert.Empty(t, res.Nodes)
}

func TestShellScriptMissingBody(t *testing.T) {
	res := expand.ExpandShellScript(parseSimple(t, `sh -c`), 0)
	require.True(t, res.Applied)
	require.True(t, hasUnknown(res.Unknowns, ir.UnknownInterpreterDynamicCode))
}

func TestShellScriptNotApplied(t *testing.T) {
	assert.False(t, expand.ExpandShellScript(parseSimple(t, `echo hi`), 0).Applied)
	assert.False(t, expand.ExpandShellScript(parseSimple(t, `sh script.sh`), 0).Applied)
}

func TestPythonInlineConservative(t *testing.T) {
	res := expand.ExpandPython(parseSimple(t, `python3 -c 'print(1)'`), 0)
	require.True(t, res.Applied)
	require.True(t, hasUnknown(res.Unknowns, ir.UnknownInterpreterDynamicCode))
	assert.True(t, findUnknown(res.Unknowns, ir.UnknownInterpreterDynamicCode).Blocking)
	assert.Empty(t, res.Nodes, "must not invent file effects from Python source")
}

func TestPythonDynamic(t *testing.T) {
	res := expand.ExpandPython(parseSimple(t, `python -c "$CODE"`), 0)
	require.True(t, res.Applied)
	require.True(t, hasUnknown(res.Unknowns, ir.UnknownInterpreterDynamicCode))
	assert.Empty(t, res.Nodes)
}

func TestPythonNotApplied(t *testing.T) {
	assert.False(t, expand.ExpandPython(parseSimple(t, `python script.py`), 0).Applied)
	assert.False(t, expand.ExpandPython(parseSimple(t, `echo hi`), 0).Applied)
}
