package analyzer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phaethix/cmdscope/internal/analyzer"
	"github.com/phaethix/cmdscope/internal/ir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextFilesLookup(t *testing.T) {
	cf, err := analyzer.NewContextFiles(&ir.AnalysisContext{
		CWD: "/ws",
		Files: map[string]string{
			"package.json": `{"name":"x"}`,
			"scripts/a.sh": "echo hi",
		},
	})
	require.NoError(t, err)

	got, ok := cf.Lookup("package.json")
	require.True(t, ok)
	assert.Equal(t, `{"name":"x"}`, got)

	got, ok = cf.Lookup(`scripts\a.sh`)
	require.True(t, ok)
	assert.Equal(t, "echo hi", got)

	_, ok = cf.Lookup("missing.txt")
	assert.False(t, ok)
}

func TestContextFilesNilContext(t *testing.T) {
	cf, err := analyzer.NewContextFiles(nil)
	require.NoError(t, err)
	_, ok := cf.Lookup("package.json")
	assert.False(t, ok)

	cf, err = analyzer.NewContextFiles(&ir.AnalysisContext{})
	require.NoError(t, err)
	_, ok = cf.Lookup("package.json")
	assert.False(t, ok)
}

func TestContextFilesLookupRejectsBadPaths(t *testing.T) {
	cf, err := analyzer.NewContextFiles(&ir.AnalysisContext{
		CWD:   "/ws",
		Files: map[string]string{"ok.txt": "x"},
	})
	require.NoError(t, err)
	for _, path := range []string{"", "../x", "/abs", `C:\windows`} {
		_, ok := cf.Lookup(path)
		assert.False(t, ok, "path %q", path)
	}
}

func TestContextFilesRejectsOversizedSingleFile(t *testing.T) {
	_, err := analyzer.NewContextFiles(&ir.AnalysisContext{
		CWD:   "/ws",
		Files: map[string]string{"big.txt": strings.Repeat("x", ir.MaxContextFileBytes+1)},
	})
	require.Error(t, err)
	var ve *ir.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, ir.ErrCodeContextFileTooLarge, ve.Code)
}

func TestContextFilesRejectsOversizedTotal(t *testing.T) {
	chunk := strings.Repeat("x", ir.MaxContextFileBytes)
	_, err := analyzer.NewContextFiles(&ir.AnalysisContext{
		CWD: "/ws",
		Files: map[string]string{
			"a.txt": chunk,
			"b.txt": chunk,
			"c.txt": chunk,
			"d.txt": chunk,
			"e.txt": chunk,
			"f.txt": chunk,
			"g.txt": chunk,
			"h.txt": chunk,
			"i.txt": "y",
		},
	})
	require.Error(t, err)
	var ve *ir.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, ir.ErrCodeContextFileTooLarge, ve.Code)
}

func TestContextFilesRejectsInvalidKey(t *testing.T) {
	_, err := analyzer.NewContextFiles(&ir.AnalysisContext{
		CWD:   "/ws",
		Files: map[string]string{"../secret": "x"},
	})
	require.Error(t, err)
	var ve *ir.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, ir.ErrCodeInvalidContextPath, ve.Code)
}

func TestContextFilesDoesNotReadOSFilesystem(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "on-disk.txt")
	require.NoError(t, os.WriteFile(path, []byte("from-disk"), 0o600))

	cf, err := analyzer.NewContextFiles(&ir.AnalysisContext{
		CWD:   "/ws",
		Files: map[string]string{},
	})
	require.NoError(t, err)

	_, ok := cf.Lookup(path)
	assert.False(t, ok, "must not read host filesystem paths")
	_, ok = cf.Lookup("on-disk.txt")
	assert.False(t, ok)
}
