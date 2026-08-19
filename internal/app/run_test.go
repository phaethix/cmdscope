package app_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/phaethix/runmark/internal/app"
	"github.com/phaethix/runmark/internal/facts"
	"github.com/phaethix/runmark/internal/ir"
	"github.com/stretchr/testify/require"
)

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := app.Run([]string{"version"}, &stdout, &stderr)
	require.Equal(t, 0, code)
	require.Equal(t, "runmark 0.1.0\n", stdout.String())
	require.Empty(t, stderr.String())
}

func TestRunUsageWhenNoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := app.Run(nil, &stdout, &stderr)
	require.Equal(t, 2, code)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "usage:")
}

func TestRunAnalyzeFactsDefault(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := app.Run([]string{
		"analyze", "echo hi > out.txt",
		"--cwd", "logical://workspace",
	}, &stdout, &stderr)
	require.Equal(t, 0, code, stderr.String())
	require.Empty(t, stderr.String())

	var got facts.RunmarkFacts
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &got))
	require.Equal(t, facts.SchemaVersion, got.SchemaVersion)
	require.Contains(t, got.Touches.Write, "logical://workspace/out.txt")
}

func TestRunAnalyzeUnsupportedCommandFacts(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := app.Run([]string{
		"analyze", "unsupported-tool --flag",
		"--cwd", "logical://workspace",
		"--format", "facts",
	}, &stdout, &stderr)
	require.Equal(t, 0, code, stderr.String())

	var got facts.RunmarkFacts
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &got))
	require.True(t, got.Unknown, "unsupported commands must not project as no-impact")
	require.Contains(t, got.UnknownReasons, string(ir.UnknownUnsupportedCommand))
	require.Empty(t, got.Touches.Read)
	require.Empty(t, got.Touches.Write)
	require.Empty(t, got.Touches.Delete)
}

func TestRunAnalyzeFormatImpact(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := app.Run([]string{
		"analyze", "echo hi > out.txt",
		"--cwd", "logical://workspace",
		"--format", "impact",
	}, &stdout, &stderr)
	require.Equal(t, 0, code, stderr.String())

	var report ir.ImpactReport
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &report))
	require.Equal(t, ir.SchemaVersion, report.SchemaVersion)
	require.NotEmpty(t, report.Stages)
}

func TestRunAnalyzeFormatText(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := app.Run([]string{
		"analyze", "echo hi > out.txt",
		"--cwd", "logical://workspace",
		"--format", "text",
	}, &stdout, &stderr)
	require.Equal(t, 0, code, stderr.String())
	out := stdout.String()
	require.Contains(t, out, "schema_version:")
	require.Contains(t, out, "write:")
	require.Contains(t, out, "out.txt")
}

func TestRunAnalyzeMissingCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := app.Run([]string{"analyze", "--cwd", "logical://workspace"}, &stdout, &stderr)
	require.Equal(t, 2, code)
	require.Empty(t, stdout.String())
	require.NotEmpty(t, stderr.String())
}

func TestRunAnalyzeUnknownFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := app.Run([]string{
		"analyze", "echo hi",
		"--format", "html",
	}, &stdout, &stderr)
	require.Equal(t, 2, code)
	require.Empty(t, stdout.String())
}

func TestRunAnalyzeContextFile(t *testing.T) {
	dir := t.TempDir()
	ctxPath := filepath.Join(dir, "context.json")
	payload := `{
  "cwd": "logical://workspace",
  "files": {
    "package.json": "{\"scripts\":{\"build\":\"rm -rf dist\"}}"
  }
}`
	require.NoError(t, os.WriteFile(ctxPath, []byte(payload), 0o600))

	var stdout, stderr bytes.Buffer
	code := app.Run([]string{
		"analyze", "npm run build",
		"--context-file", ctxPath,
		"--format", "facts",
	}, &stdout, &stderr)
	require.Equal(t, 0, code, stderr.String())

	var got facts.RunmarkFacts
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &got))
	require.True(t, got.Boundary.Destructive)
	require.NotEmpty(t, got.Scripts)
	require.Equal(t, "npm", got.Scripts[0].Tool)
}

func TestRunAnalyzeCWDOverridesContextFile(t *testing.T) {
	dir := t.TempDir()
	ctxPath := filepath.Join(dir, "context.json")
	payload := `{"cwd":"logical://other","files":{}}`
	require.NoError(t, os.WriteFile(ctxPath, []byte(payload), 0o600))

	var stdout, stderr bytes.Buffer
	code := app.Run([]string{
		"analyze", "echo hi > out.txt",
		"--context-file", ctxPath,
		"--cwd", "logical://workspace",
		"--format", "impact",
	}, &stdout, &stderr)
	require.Equal(t, 0, code, stderr.String())

	var report ir.ImpactReport
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &report))
	require.Equal(t, "logical://workspace", report.CWD)
}
