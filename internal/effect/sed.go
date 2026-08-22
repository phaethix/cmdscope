package effect

import (
	"strings"

	"github.com/phaethix/runmark/internal/ir"
	"github.com/phaethix/runmark/internal/logicalpath"
	"github.com/phaethix/runmark/internal/shell"
)

// ExtractSed covers sed path effects. -i triggers read+write; without -i, only
// read. The first non-option argument is the script expression and is skipped
// because it is never a filesystem path.
func ExtractSed(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string) ([]ir.Effect, []ir.Unknown) {
	if cmd == nil || len(cmd.Words) == 0 {
		return nil, nil
	}
	if commandBasename(cmd.Words[0].Text) != "sed" {
		return nil, nil
	}

	inPlace := false
	words := cmd.Words[1:]
	endOpts := false
	var fileWords []shell.Word
	scriptSeen := false

	for i := 0; i < len(words); i++ {
		w := words[i]
		if !endOpts {
			if w.Text == "--" {
				endOpts = true
				continue
			}
			if strings.HasPrefix(w.Text, "-") && w.Text != "-" {
				if isSedInPlace(w.Text) {
					inPlace = true
				}
				if sedScriptOption(w.Text) {
					scriptSeen = true
					if !strings.Contains(w.Text, "=") && i+1 < len(words) {
						i++
					}
					continue
				}
				if sedOptionTakesArg(w.Text) && i+1 < len(words) && !strings.HasPrefix(words[i+1].Text, "-") {
					i++
				}
				continue
			}
		}
		if !scriptSeen {
			// Without an explicit -e/-f the first positional operand is the
			// script expression, never a filesystem path.
			scriptSeen = true
			continue
		}
		if w.Text == "-" {
			continue
		}
		fileWords = append(fileWords, w)
	}

	if len(fileWords) == 0 {
		return nil, nil
	}

	var effects []ir.Effect
	for _, w := range fileWords {
		target, _ := logicalpath.NormalizeLogicalPath(w.Text, cwd)
		read := ir.Effect{
			Kind:       ir.EffectRead,
			RawTarget:  w.Text,
			Target:     target,
			Stage:      stage,
			Certainty:  ir.Certain,
			Provenance: ir.FromCommand,
			Condition:  cond,
			Evidence:   []ir.Evidence{commandEvidence(w.Start, w.End, w.Text)},
		}
		read.ID = ir.EffectID(ir.SchemaVersion, read)
		effects = append(effects, read)
		if inPlace {
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
			effects = append(effects, write)
		}
	}
	return effects, nil
}

func isSedInPlace(opt string) bool {
	if opt == "-i" {
		return true
	}
	if strings.HasPrefix(opt, "-i") && len(opt) > 2 {
		return true
	}
	if strings.HasPrefix(opt, "--in-place") {
		return true
	}
	return false
}

// sedScriptOption reports options that carry the script expression. Once the
// script arrives through one of these, later positionals are all file
// operands; the = forms embed their value and consume nothing extra.
func sedScriptOption(opt string) bool {
	switch opt {
	case "-e", "--expression", "-f", "--file":
		return true
	}
	return strings.HasPrefix(opt, "--expression=") || strings.HasPrefix(opt, "--file=")
}

// sedOptionTakesArg stays false across the board: sed's only argument-taking
// options carry the script itself and are handled by sedScriptOption, so
// treating anything else as consuming a value would swallow an input file.
func sedOptionTakesArg(opt string) bool {
	return false
}
