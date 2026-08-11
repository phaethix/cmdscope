package expand_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phaethix/cmdscope/internal/expand"
	"github.com/phaethix/cmdscope/internal/ir"
	"github.com/phaethix/cmdscope/internal/shell"
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

func TestMakeNotApplied(t *testing.T) {
	res := expand.ExpandMake(parseSimple(t, "echo hi"), makeFixtureFiles(t), 0)
	assert.False(t, res.Applied)
}

func makeFixtureFiles(t *testing.T) map[string]string {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "fixtures", "Makefile")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return map[string]string{"Makefile": string(data)}
}
