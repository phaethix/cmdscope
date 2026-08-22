package shared_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phaethix/runmark/internal/adapter/shared"
	"github.com/stretchr/testify/require"
)

func TestHandleBashPreToolUseInjectsAdditionalContext(t *testing.T) {
	in := `{
  "hook_event_name": "PreToolUse",
  "tool_name": "Bash",
  "tool_input": {"command": "echo hi > out.txt"},
  "cwd": "logical://workspace"
}`
	var stdout, stderr bytes.Buffer
	code := shared.HandlePreToolUseBash(context.Background(), strings.NewReader(in), &stdout, &stderr)
	require.Equal(t, 0, code)
	require.Empty(t, stderr.String())

	var out map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &out))
	hso, ok := out["hookSpecificOutput"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "PreToolUse", hso["hookEventName"])
	ctx, ok := hso["additionalContext"].(string)
	require.True(t, ok)
	require.Contains(t, ctx, "out.txt")
	require.Contains(t, ctx, "write:")
	_, hasDecision := hso["permissionDecision"]
	require.False(t, hasDecision)
}

func TestHandleInjectsWorkspaceFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"build":"rm -rf dist"}}`), 0o600))

	in := `{
  "hook_event_name": "PreToolUse",
  "tool_name": "Bash",
  "tool_input": {"command": "npm run build"},
  "cwd": "` + dir + `"
}`
	var stdout, stderr bytes.Buffer
	code := shared.HandlePreToolUseBash(context.Background(), strings.NewReader(in), &stdout, &stderr)
	require.Equal(t, 0, code)
	require.Empty(t, stderr.String())

	var out map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &out))
	hso := out["hookSpecificOutput"].(map[string]any)
	require.Contains(t, hso["additionalContext"], "dist")
}

func TestHandleNoOpNonTargetEvents(t *testing.T) {
	cases := []string{
		`{"hook_event_name":"PostToolUse","tool_name":"Bash","tool_input":{"command":"echo hi"},"cwd":"logical://workspace"}`,
		`{"hook_event_name":"PreToolUse","tool_name":"Edit","tool_input":{"command":"echo hi"},"cwd":"logical://workspace"}`,
		`{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{},"cwd":"logical://workspace"}`,
		`{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"  "},"cwd":"logical://workspace"}`,
	}
	for _, in := range cases {
		var stdout, stderr bytes.Buffer
		code := shared.HandlePreToolUseBash(context.Background(), strings.NewReader(in), &stdout, &stderr)
		require.Equal(t, 0, code, in)
		require.Empty(t, stdout.String(), in)
		require.Empty(t, stderr.String(), in)
	}
}

func TestHandleInvalidJSONIsSilentNoOp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := shared.HandlePreToolUseBash(context.Background(), strings.NewReader("{not-json"), &stdout, &stderr)
	require.Equal(t, 0, code)
	require.Empty(t, stdout.String())
	require.Empty(t, stderr.String())
}

func TestHandleAnalysisFailureDoesNotLeakCommand(t *testing.T) {
	// Invalid cwd forces ValidateRequest failure after we attempt analysis.
	secret := "SUPER_SECRET_RM_COMMAND_xyz"
	in := `{
  "hook_event_name": "PreToolUse",
  "tool_name": "Bash",
  "tool_input": {"command": "` + secret + `"},
  "cwd": "not-a-valid-cwd"
}`
	var stdout, stderr bytes.Buffer
	code := shared.HandlePreToolUseBash(context.Background(), strings.NewReader(in), &stdout, &stderr)
	require.Equal(t, 0, code)
	require.Empty(t, stdout.String())
	require.NotContains(t, stderr.String(), secret)
	require.Contains(t, stderr.String(), "runmark: hook analysis failed")
}
