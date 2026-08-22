package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/phaethix/runmark/internal/adapter/shared"
	"github.com/phaethix/runmark/internal/analyzer"
	"github.com/phaethix/runmark/internal/facts"
	"github.com/phaethix/runmark/internal/ir"
)

const (
	exitOK = 0
)

// Handle translates one Codex hook stdin JSON event to stdout. It only emits
// additionalContext for PreToolUse+Bash with a non-empty command; everything
// else is a silent no-op (exit 0, empty stdout). It never denies or asks, and
// never echoes the command or context onto stderr.
func Handle(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) int {
	raw, err := io.ReadAll(stdin)
	if err != nil {
		fmt.Fprintln(stderr, "runmark: hook read failed")
		return exitOK
	}
	ev, ok := parseEvent(raw)
	if !ok {
		return exitOK
	}
	if !isTarget(ev) {
		return exitOK
	}

	req := ir.AnalyzeRequest{Command: ev.command}
	if ev.cwd != "" {
		req.Context = &ir.AnalysisContext{
			CWD:   ev.cwd,
			Files: map[string]string{},
			Env:   map[string]string{},
		}
	}
	shared.InjectContext(req.Context)
	report, err := analyzer.Analyze(ctx, req)
	if err != nil {
		fmt.Fprintln(stderr, "runmark: hook analysis failed")
		return exitOK
	}
	if err := ir.ValidateReport(report); err != nil {
		fmt.Fprintln(stderr, "runmark: hook analysis failed")
		return exitOK
	}
	f := facts.Project(report)
	if err := facts.Validate(f); err != nil {
		fmt.Fprintln(stderr, "runmark: hook analysis failed")
		return exitOK
	}
	facts.Normalize(&f)

	out := hookOutput{
		HookSpecificOutput: hookSpecificOutput{
			HookEventName:     "PreToolUse",
			AdditionalContext: facts.FormatText(f),
		},
	}
	body, err := json.Marshal(out)
	if err != nil {
		fmt.Fprintln(stderr, "runmark: hook analysis failed")
		return exitOK
	}
	if _, err := stdout.Write(append(body, '\n')); err != nil {
		fmt.Fprintln(stderr, "runmark: hook write failed")
		return 1
	}
	return exitOK
}

type event struct {
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

func parseEvent(raw []byte) (event, bool) {
	var wire struct {
		HookEventName string          `json:"hook_event_name"`
		ToolName      string          `json:"tool_name"`
		CWD           string          `json:"cwd"`
		ToolInput     json.RawMessage `json:"tool_input"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return event{}, false
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
	return event{
		command: cmd,
		cwd:     wire.CWD,
		name:    wire.HookEventName,
		tool:    wire.ToolName,
	}, true
}

func isTarget(ev event) bool {
	if ev.name != "PreToolUse" {
		return false
	}
	if ev.tool != "Bash" {
		return false
	}
	return strings.TrimSpace(ev.command) != ""
}
