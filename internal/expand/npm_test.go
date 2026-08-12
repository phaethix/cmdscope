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

func TestNPMExpandBuild(t *testing.T) {
	files := fixtureFiles(t)
	cmd := parseSimple(t, "npm run build")
	res := expand.ExpandNPM(cmd, files, 0)
	require.True(t, res.Applied)
	require.Empty(t, res.Unknowns)
	require.NotEmpty(t, res.Nodes)
	require.NotEmpty(t, res.Evidence)
	assert.Equal(t, ir.EvidenceWorkspaceFile, res.Evidence[0].Source)
	assert.Equal(t, "package.json", res.Evidence[0].Path)
	assert.Equal(t, "scripts.build", res.Evidence[0].Field)
	echo, ok := res.Nodes[0].(*shell.SimpleCommand)
	require.True(t, ok)
	require.NotEmpty(t, echo.Words)
	assert.Equal(t, "echo", echo.Words[0].Text)
}

func TestNPMExpandCycle(t *testing.T) {
	files := fixtureFiles(t)
	res := expand.ExpandNPM(parseSimple(t, "npm run a"), files, 0)
	require.True(t, res.Applied)
	require.True(t, hasUnknown(res.Unknowns, ir.UnknownExpansionCycle))
	msg := findUnknown(res.Unknowns, ir.UnknownExpansionCycle).Message
	assert.Contains(t, msg, "a -> b -> a")
}

func TestNPMExpandSharedDiamond(t *testing.T) {
	files := fixtureFiles(t)
	// One expand that reaches shared via two parents must not false-cycle.
	res := expand.ExpandNPM(parseSimple(t, "npm run root"), files, 0)
	require.True(t, res.Applied)
	assert.False(t, hasUnknown(res.Unknowns, ir.UnknownExpansionCycle))
	require.NotEmpty(t, res.Nodes)
}

func TestNPMExpandNestedDynamicScriptName(t *testing.T) {
	files := fixtureFiles(t)
	res := expand.ExpandNPM(parseSimple(t, "npm run outer"), files, 0)
	require.True(t, res.Applied)
	require.True(t, hasUnknown(res.Unknowns, ir.UnknownScriptDynamicPath))
}

func TestNPMExpandMissingPackage(t *testing.T) {
	res := expand.ExpandNPM(parseSimple(t, "npm run build"), map[string]string{}, 0)
	require.True(t, res.Applied)
	require.True(t, hasUnknown(res.Unknowns, ir.UnknownContextMissing))
	assert.True(t, findUnknown(res.Unknowns, ir.UnknownContextMissing).Blocking)
}

func TestNPMExpandMissingScript(t *testing.T) {
	files := fixtureFiles(t)
	res := expand.ExpandNPM(parseSimple(t, "npm run nope"), files, 0)
	require.True(t, res.Applied)
	require.True(t, hasUnknown(res.Unknowns, ir.UnknownUnsupportedCommand))
}

func TestNPMExpandDynamicScriptName(t *testing.T) {
	files := fixtureFiles(t)
	res := expand.ExpandNPM(parseSimple(t, "npm run $NAME"), files, 0)
	require.True(t, res.Applied)
	require.True(t, hasUnknown(res.Unknowns, ir.UnknownScriptDynamicPath))
}

func TestNPMNotApplied(t *testing.T) {
	res := expand.ExpandNPM(parseSimple(t, "npm install"), fixtureFiles(t), 0)
	assert.False(t, res.Applied)
}

func TestPNPMExpandBuild(t *testing.T) {
	files := fixtureFiles(t)
	res := expand.ExpandPNPM(parseSimple(t, "pnpm run build"), files, 1)
	require.True(t, res.Applied)
	require.Empty(t, res.Unknowns)
	require.NotEmpty(t, res.Nodes)
	echo, ok := res.Nodes[0].(*shell.SimpleCommand)
	require.True(t, ok)
	assert.Equal(t, "echo", echo.Words[0].Text)
}

func fixtureFiles(t *testing.T) map[string]string {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "fixtures", "package.json")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return map[string]string{"package.json": string(data)}
}

func parseSimple(t *testing.T, command string) *shell.SimpleCommand {
	t.Helper()
	toks, err := shell.Lex(command)
	require.NoError(t, err)
	root, err := shell.Parse(toks)
	require.NoError(t, err)
	cmd, ok := root.(*shell.SimpleCommand)
	require.True(t, ok, "got %T", root)
	return cmd
}

func hasUnknown(us []ir.Unknown, code ir.UnknownCode) bool {
	for _, u := range us {
		if u.Code == code {
			return true
		}
	}
	return false
}

func findUnknown(us []ir.Unknown, code ir.UnknownCode) ir.Unknown {
	for _, u := range us {
		if u.Code == code {
			return u
		}
	}
	return ir.Unknown{}
}
