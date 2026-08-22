package shared_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phaethix/runmark/internal/adapter/shared"
	"github.com/phaethix/runmark/internal/ir"
	"github.com/stretchr/testify/require"
)

func TestInjectContextReadsWhitelistFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"build":"rm -rf dist"}}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Makefile"), []byte("build:\n\trm -rf out\n"), 0o600))

	ac := &ir.AnalysisContext{CWD: dir, Files: map[string]string{}, Env: map[string]string{}}
	shared.InjectContext(ac)

	require.Contains(t, ac.Files, "package.json")
	require.Contains(t, ac.Files, "Makefile")
	require.NotEmpty(t, ac.Platform)
}

func TestInjectContextSkipsLogicalCWD(t *testing.T) {
	ac := &ir.AnalysisContext{CWD: "logical://workspace", Files: map[string]string{}, Env: map[string]string{}}
	shared.InjectContext(ac)
	require.Empty(t, ac.Files)
	require.NotEmpty(t, ac.Platform)
}

func TestInjectContextDisabledByEnv(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0o600))
	t.Setenv("RUNMARK_HOOK_CONTEXT", "0")

	ac := &ir.AnalysisContext{CWD: dir, Files: map[string]string{}, Env: map[string]string{}}
	shared.InjectContext(ac)
	require.Empty(t, ac.Files)
}

func TestInjectContextDoesNotOverwriteExistingFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{}}`), 0o600))

	ac := &ir.AnalysisContext{CWD: dir, Files: map[string]string{"package.json": "caller-provided"}}
	shared.InjectContext(ac)
	require.Equal(t, "caller-provided", ac.Files["package.json"])
}

func TestInjectContextSkipsOversizedFiles(t *testing.T) {
	dir := t.TempDir()
	big := make([]byte, ir.MaxContextFileBytes+1)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), big, 0o600))

	ac := &ir.AnalysisContext{CWD: dir, Files: map[string]string{}, Env: map[string]string{}}
	shared.InjectContext(ac)
	_, ok := ac.Files["package.json"]
	require.False(t, ok)
}
