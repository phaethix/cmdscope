package expand_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phaethix/runmark/internal/expand"
	"github.com/phaethix/runmark/internal/ir"
	"github.com/phaethix/runmark/internal/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeExpandBuild(t *testing.T) {
	files := makeFixtureFiles(t)
	res := expand.ExpandMake(parseSimple(t, "make build"), files, 0)
	require.True(t, res.Applied)
	require.Empty(t, res.Unknowns)
	require.NotEmpty(t, res.Nodes)
	require.NotEmpty(t, res.Evidence)
	assert.Equal(t, "Makefile", res.Evidence[0].Path)
	echo, ok := res.Nodes[0].(*shell.SimpleCommand)
	require.True(t, ok)
	require.GreaterOrEqual(t, len(echo.Words), 2)
	assert.Equal(t, "echo", echo.Words[0].Text)
	assert.Equal(t, "building", echo.Words[1].Text)
}

func TestMakeExpandCycle(t *testing.T) {
	files := makeFixtureFiles(t)
	res := expand.ExpandMake(parseSimple(t, "make a"), files, 0)
	require.True(t, res.Applied)
	require.True(t, hasUnknown(res.Unknowns, ir.UnknownExpansionCycle))
	assert.Contains(t, findUnknown(res.Unknowns, ir.UnknownExpansionCycle).Message, "a -> b -> a")
}

func TestMakeExpandSharedDiamond(t *testing.T) {
	files := makeFixtureFiles(t)
	res := expand.ExpandMake(parseSimple(t, "make root"), files, 0)
	require.True(t, res.Applied)
	assert.False(t, hasUnknown(res.Unknowns, ir.UnknownExpansionCycle))
	require.NotEmpty(t, res.Nodes)
}

func TestMakeExpandMissingMakefile(t *testing.T) {
	res := expand.ExpandMake(parseSimple(t, "make build"), map[string]string{}, 0)
	require.True(t, res.Applied)
	require.True(t, hasUnknown(res.Unknowns, ir.UnknownContextMissing))
	assert.True(t, findUnknown(res.Unknowns, ir.UnknownContextMissing).Blocking)
}

func TestMakeExpandMissingTarget(t *testing.T) {
	files := makeFixtureFiles(t)
	res := expand.ExpandMake(parseSimple(t, "make nope"), files, 0)
	require.True(t, res.Applied)
	require.True(t, hasUnknown(res.Unknowns, ir.UnknownUnsupportedCommand))
}

func TestMakeExpandDynamicTarget(t *testing.T) {
	files := makeFixtureFiles(t)
	res := expand.ExpandMake(parseSimple(t, "make $T"), files, 0)
	require.True(t, res.Applied)
	require.True(t, hasUnknown(res.Unknowns, ir.UnknownScriptDynamicPath))
}

func TestMakeExpandDynamicInclude(t *testing.T) {
	files := map[string]string{
		"Makefile": "include other.mk\nbuild:\n\techo hi\n",
	}
	res := expand.ExpandMake(parseSimple(t, "make build"), files, 0)
	require.True(t, res.Applied)
	require.NotEmpty(t, res.Unknowns)
	assert.True(t, res.Unknowns[0].Blocking)
}

func TestMakeExpandNestedVarsDeterministic(t *testing.T) {
	files := map[string]string{
		"Makefile": "A = $(B)\nB = hello\nbuild:\n\techo $(A)\n",
	}
	var first expand.ExpansionResult
	for i := range 50 {
		res := expand.ExpandMake(parseSimple(t, "make build"), files, 0)
		require.True(t, res.Applied)
		require.Empty(t, res.Unknowns, "run %d: nested literal vars must fully resolve", i)
		require.NotEmpty(t, res.Nodes)
		if i == 0 {
			first = res
			continue
		}
		require.Equal(t, len(first.Nodes), len(res.Nodes))
		a, ok := first.Nodes[0].(*shell.SimpleCommand)
		require.True(t, ok)
		b, ok := res.Nodes[0].(*shell.SimpleCommand)
		require.True(t, ok)
		require.Equal(t, wordTexts(a), wordTexts(b), "run %d", i)
	}
}

func wordTexts(cmd *shell.SimpleCommand) []string {
	out := make([]string, len(cmd.Words))
	for i, w := range cmd.Words {
		out[i] = w.Text
	}
	return out
}

func makeFixtureFiles(t *testing.T) map[string]string {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "fixtures", "Makefile")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return map[string]string{"Makefile": string(data)}
}
