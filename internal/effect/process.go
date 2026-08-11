package effect

import (
	"strconv"
	"strings"

	"github.com/phaethix/cmdscope/internal/ir"
	"github.com/phaethix/cmdscope/internal/shell"
)

// ExtractProcess: builtins are silent; every other argv[0] still yields
// process so unknown tools never produce an empty extraction. Unrecognized
// names add unsupported_command on top of that process effect.
func ExtractProcess(cmd *shell.SimpleCommand, stage int, cond ir.Condition) ([]ir.Effect, []ir.Unknown) {
	if cmd == nil || len(cmd.Words) == 0 {
		return nil, nil
	}

	argv0 := cmd.Words[0]
	name := commandBasename(argv0.Text)
	if builtins[name] {
		return nil, nil
	}

	ef := ir.Effect{
		Kind:       ir.EffectProcess,
		RawTarget:  argv0.Text,
		Target:     argv0.Text,
		Stage:      stage,
		Certainty:  ir.Certain,
		Provenance: ir.FromCommand,
		Condition:  cond,
		Evidence:   []ir.Evidence{commandEvidence(argv0.Start, argv0.End, argv0.Text)},
	}
	ef.ID = ir.EffectID(ir.SchemaVersion, ef)

	effects := []ir.Effect{ef}
	if knownCommands[name] {
		return effects, nil
	}

	unk := ir.Unknown{
		Code:     ir.UnknownUnsupportedCommand,
		Scope:    "stage:" + strconv.Itoa(stage),
		Message:  "unsupported command " + strconv.Quote(argv0.Text),
		Evidence: []ir.Evidence{commandEvidence(argv0.Start, argv0.End, argv0.Text)},
		Blocking: false,
	}
	return effects, []ir.Unknown{unk}
}

// commandEvidence duplicates the analyzer helper on purpose: importing
// analyzer here would cycle once Analyze wires extractors.
func commandEvidence(start, end int, snippet string) ir.Evidence {
	ev := ir.Evidence{Source: ir.EvidenceCommand, Snippet: snippet}
	if start >= 0 && end > start {
		ev.StartByte = intPtr(start)
		ev.EndByte = intPtr(end)
	}
	return ev
}

func intPtr(n int) *int { return &n }

func commandBasename(argv0 string) string {
	argv0 = strings.ReplaceAll(argv0, `\`, "/")
	if i := strings.LastIndex(argv0, "/"); i >= 0 {
		return argv0[i+1:]
	}
	return argv0
}

var builtins = map[string]bool{
	"echo":   true,
	"true":   true,
	"false":  true,
	":":      true,
	"cd":     true,
	"export": true,
	"unset":  true,
	"pwd":    true,
	"test":   true,
	"[":      true,
}

// knownCommands: names with dedicated extractors (plus go) must not also
// raise unsupported_command. Builtins are included so a single-map check
// remains valid if a caller skips the builtins table.
var knownCommands = map[string]bool{
	"echo": true, "true": true, "false": true, ":": true,
	"cd": true, "export": true, "unset": true, "pwd": true,
	"test": true, "[": true,
	"go":  true,
	"cat": true, "head": true, "grep": true,
	"mkdir": true, "rm": true, "cp": true, "mv": true,
	"curl": true, "wget": true,
	"sudo": true, "chmod": true, "chown": true,
	"npm": true, "pnpm": true,
}
