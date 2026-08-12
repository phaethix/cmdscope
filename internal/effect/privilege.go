package effect

import (
	"strings"

	"github.com/phaethix/runmark/internal/ir"
	"github.com/phaethix/runmark/internal/logicalpath"
	"github.com/phaethix/runmark/internal/shell"
)

// ExtractPrivilege covers sudo elevation and chmod/chown metadata writes.
// sudo only yields privilege(sudo); the inner argv remains for ExtractProcess.
func ExtractPrivilege(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string) ([]ir.Effect, []ir.Unknown) {
	if cmd == nil || len(cmd.Words) == 0 {
		return nil, nil
	}
	name := commandBasename(cmd.Words[0].Text)
	switch name {
	case "sudo":
		ef := ir.Effect{
			Kind:       ir.EffectPrivilege,
			RawTarget:  cmd.Words[0].Text,
			Target:     commandBasename(cmd.Words[0].Text),
			Stage:      stage,
			Certainty:  ir.Certain,
			Provenance: ir.FromCommand,
			Condition:  cond,
			Evidence:   []ir.Evidence{commandEvidence(cmd.Words[0].Start, cmd.Words[0].End, cmd.Words[0].Text)},
		}
		ef.ID = ir.EffectID(ir.SchemaVersion, ef)
		return []ir.Effect{ef}, nil
	case "chmod", "chown":
		paths := privilegePaths(cmd.Words[1:])
		if len(paths) == 0 {
			return nil, []ir.Unknown{insufficientOperandUnknown(cmd.Words[0], stage, name)}
		}
		effects := make([]ir.Effect, 0, len(paths)*2)
		for _, w := range paths {
			target, _ := logicalpath.NormalizeLogicalPath(w.Text, cwd)
			priv := ir.Effect{
				Kind:       ir.EffectPrivilege,
				RawTarget:  w.Text,
				Target:     target,
				Stage:      stage,
				Certainty:  ir.Certain,
				Provenance: ir.FromCommand,
				Condition:  cond,
				Evidence:   []ir.Evidence{commandEvidence(w.Start, w.End, w.Text)},
			}
			priv.ID = ir.EffectID(ir.SchemaVersion, priv)
			write := ir.Effect{
				Kind:       ir.EffectWrite,
				RawTarget:  w.Text,
				Target:     target,
				Stage:      stage,
				Certainty:  ir.Certain,
				Provenance: ir.FromCommand,
				Condition:  cond,
				Evidence:   []ir.Evidence{commandEvidence(w.Start, w.End, w.Text)},
			}
			write.ID = ir.EffectID(ir.SchemaVersion, write)
			effects = append(effects, priv, write)
		}
		return effects, nil
	default:
		return nil, nil
	}
}

func privilegePaths(words []shell.Word) []shell.Word {
	operands := skipDashOptions(words, privilegeOptionTakesArg)
	if len(operands) == 0 {
		return nil
	}
	// Mode (chmod) or owner (chown) is never a filesystem path effect target.
	return dropDashOperands(operands[1:])
}

func privilegeOptionTakesArg(string) bool {
	// Avoid claiming --reference/--from support until mode/owner skipping is
	// conditional; mis-handling drops the only remaining path operand.
	return false
}

func skipDashOptions(words []shell.Word, takesArg func(string) bool) []shell.Word {
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
				if takesArg(w.Text) && i+1 < len(words) && !strings.HasPrefix(words[i+1].Text, "-") {
					i++
				}
				continue
			}
		}
		out = append(out, w)
	}
	return out
}

func dropDashOperands(words []shell.Word) []shell.Word {
	var out []shell.Word
	for _, w := range words {
		if w.Text == "-" {
			continue
		}
		out = append(out, w)
	}
	return out
}
