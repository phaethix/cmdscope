package effect

import (
	"strings"

	"github.com/phaethix/runmark/internal/shell"
)

// positionalOperands returns the non-option words of one simple command:
// "--" ends option parsing, dash-prefixed words are skipped (consuming their
// value when optTakesArg says so), and a bare "-" never names a path because
// it stands for stdin/stdout. Family extractors share this walker so operand
// roles stay consistent across commands.
func positionalOperands(words []shell.Word, optTakesArg func(string) bool) []shell.Word {
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
				if optTakesArg(w.Text) && i+1 < len(words) && !strings.HasPrefix(words[i+1].Text, "-") {
					i++
				}
				continue
			}
		}
		if w.Text == "-" {
			continue
		}
		out = append(out, w)
	}
	return out
}

// noOptionArgs is the arity table for commands whose every option is boolean,
// so no dash-word may legally swallow the operand after it.
func noOptionArgs(string) bool { return false }
