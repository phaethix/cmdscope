package claude_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/phaethix/runmark/internal/adapter/claude"
	"github.com/stretchr/testify/require"
)

// TestHandleDelegatesSharedPipeline pins the adapter's only job: its wire
// contract reaches the shared pipeline unchanged. Full pipeline behavior is
// covered by shared's own tests.
func TestHandleDelegatesSharedPipeline(t *testing.T) {
	in := `{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"echo hi"},"cwd":"logical://workspace"}`
	var stdout, stderr bytes.Buffer
	code := claude.Handle(context.Background(), strings.NewReader(in), &stdout, &stderr)
	require.Equal(t, 0, code)
	require.Empty(t, stderr.String())
	require.Contains(t, stdout.String(), `"hookEventName":"PreToolUse"`)
}
