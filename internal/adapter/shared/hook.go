package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/phaethix/runmark/internal/analyzer"
	"github.com/phaethix/runmark/internal/facts"
	"github.com/phaethix/runmark/internal/ir"
)

const hookExitOK = 0

// HandlePreToolUseBash is the hook pipeline both client adapters share: parse
// one stdin event, inject bounded workspace context, analyze, project facts,
// and print a single additionalContext JSON line. Every failure is fail-open
// (exit 0, empty stdout, one stderr line that never echoes the command or
// context) so broken analysis cannot block an agent session; only losing
// stdout returns non-zero.
func HandlePreToolUseBash(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) int {
	raw, err := io.ReadAll(stdin)
	if err != nil {
		fmt.Fprintln(stderr, "runmark: hook read failed")
		return hookExitOK
	}
	ev, ok := parseHookEvent(raw)
	if !ok {
		return hookExitOK
	}
	if !isBashPreToolUse(ev) {
		return hookExitOK
	}

	req := ir.AnalyzeRequest{Command: ev.command}
	if ev.cwd != "" {
		req.Context = &ir.AnalysisContext{
			CWD:   ev.cwd,
			Files: map[string]string{},
			Env:   map[string]string{},
		}
	}
	InjectContext(req.Context)

	report, err := analyzer.Analyze(ctx, req)
	if err != nil {
		fmt.Fprintln(stderr, "runmark: hook analysis failed")
		return hookExitOK
	}
	if err := ir.ValidateReport(report); err != nil {
		fmt.Fprintln(stderr, "runmark: hook analysis failed")
		return hookExitOK
	}
	f := facts.Project(report)
	if err := facts.Validate(f); err != nil {
		fmt.Fprintln(stderr, "runmark: hook analysis failed")
		return hookExitOK
	}
	facts.Normalize(&f)

	body, err := json.Marshal(hookOutput{
		HookSpecificOutput: hookSpecificOutput{
			HookEventName:     "PreToolUse",
			AdditionalContext: facts.FormatText(f),
		},
	})
	if err != nil {
		fmt.Fprintln(stderr, "runmark: hook analysis failed")
		return hookExitOK
	}
	if _, err := stdout.Write(append(body, '\n')); err != nil {
		fmt.Fprintln(stderr, "runmark: hook write failed")
		return 1
	}
	return hookExitOK
}

type hookEvent struct {
	command string
	cwd     string
	name    string
	tool    string
}

type hookOutput struct {
	HookSpecificOutput hookSpecificOutput `json:"hookSpecificOutput"`
}

type hookSpecificOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

func parseHookEvent(raw []byte) (hookEvent, bool) {
	var wire struct {
		HookEventName string          `json:"hook_event_name"`
		ToolName      string          `json:"tool_name"`
		CWD           string          `json:"cwd"`
		ToolInput     json.RawMessage `json:"tool_input"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return hookEvent{}, false
	}
	cmd := ""
	if len(wire.ToolInput) > 0 && string(wire.ToolInput) != "null" {
		var input struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal(wire.ToolInput, &input); err == nil {
			cmd = input.Command
		}
	}
	return hookEvent{
		command: cmd,
		cwd:     wire.CWD,
		name:    wire.HookEventName,
		tool:    wire.ToolName,
	}, true
}

func isBashPreToolUse(ev hookEvent) bool {
	if ev.name != "PreToolUse" {
		return false
	}
	if ev.tool != "Bash" {
		return false
	}
	return strings.TrimSpace(ev.command) != ""
}
