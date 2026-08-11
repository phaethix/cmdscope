package effect

import (
	"strings"

	"github.com/phaethix/cmdscope/internal/ir"
	"github.com/phaethix/cmdscope/internal/logicalpath"
	"github.com/phaethix/cmdscope/internal/shell"
)

// ExtractWrite covers > / >> redirects and mkdir path operands only.
// Append is conditional because success depends on the prior stream state;
// PathFlags stay for later unknown surfacing.
func ExtractWrite(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string) ([]ir.Effect, []ir.Unknown) {
	if cmd == nil {
		return nil, nil
	}

	var effects []ir.Effect
	for _, r := range cmd.Redirects {
		cert, ok := redirectWriteCertainty(r.Operator)
		if !ok || r.Target.Text == "-" {
			// "<" is not a write; "-" would invent a cwd-joined fake path.
			continue
		}
		effects = append(effects, newWriteEffect(r.Target, stage, cond, cwd, cert))
	}

	if len(cmd.Words) > 0 && commandBasename(cmd.Words[0].Text) == "mkdir" {
		for _, w := range mkdirOperands(cmd.Words[1:]) {
			if w.Text == "-" {
				continue
			}
			effects = append(effects, newWriteEffect(w, stage, cond, cwd, ir.Certain))
		}
	}
	return effects, nil
}

func redirectWriteCertainty(op string) (ir.Certainty, bool) {
	switch op {
	case ">":
		return ir.Certain, true
	case ">>":
		return ir.Conditional, true
	default:
		return "", false
	}
}

func newWriteEffect(w shell.Word, stage int, cond ir.Condition, cwd string, cert ir.Certainty) ir.Effect {
	target, _ := logicalpath.NormalizeLogicalPath(w.Text, cwd)
	ef := ir.Effect{
		Kind:       ir.EffectWrite,
		RawTarget:  w.Text,
		Target:     target,
		Stage:      stage,
		Certainty:  cert,
		Provenance: ir.FromCommand,
		Condition:  cond,
		Evidence:   []ir.Evidence{commandEvidence(w.Start, w.End, w.Text)},
	}
	ef.ID = ir.EffectID(ir.SchemaVersion, ef)
	return ef
}

func mkdirOperands(words []shell.Word) []shell.Word {
	var out []shell.Word
	endOpts := false
	for i := 0; i < len(words); i++ {
		w := words[i]
		if !endOpts {
			if w.Text == "--" {
				endOpts = true
				continue
			}
			if strings.HasPrefix(w.Text, "-") && w.Text != "-" {
				// Lone -m MODE must not treat the mode bits as a directory path.
				if w.Text == "-m" && i+1 < len(words) && !strings.HasPrefix(words[i+1].Text, "-") {
					i++
				}
				continue
			}
		}
		out = append(out, w)
	}
	return out
}
