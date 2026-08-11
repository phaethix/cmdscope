package effect

import (
	"strings"

	"github.com/phaethix/cmdscope/internal/ir"
	"github.com/phaethix/cmdscope/internal/logicalpath"
	"github.com/phaethix/cmdscope/internal/shell"
)

// ExtractRead emits read effects for cat/head/grep file operands.
// Options are never treated as paths; grep's positional pattern is skipped
// unless -e/-f (or long forms) already supplied one. PathFlags from
// normalization are ignored here (surfaced later as unknowns).
func ExtractRead(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string) ([]ir.Effect, []ir.Unknown) {
	if cmd == nil || len(cmd.Words) == 0 {
		return nil, nil
	}

	name := commandBasename(cmd.Words[0].Text)
	switch name {
	case "cat", "head", "grep":
	default:
		return nil, nil
	}

	operands, patternFromOpt := readOperands(cmd.Words[1:], name)
	if name == "grep" && !patternFromOpt {
		if len(operands) == 0 {
			return nil, nil
		}
		operands = operands[1:]
	}

	effects := make([]ir.Effect, 0, len(operands))
	for _, w := range operands {
		if w.Text == "-" {
			// Explicit stdin placeholder is not a filesystem path.
			continue
		}
		target, _ := logicalpath.NormalizeLogicalPath(w.Text, cwd)
		ef := ir.Effect{
			Kind:       ir.EffectRead,
			RawTarget:  w.Text,
			Target:     target,
			Stage:      stage,
			Certainty:  ir.Certain,
			Provenance: ir.FromCommand,
			Condition:  cond,
			Evidence:   []ir.Evidence{commandEvidence(w.Start, w.End, w.Text)},
		}
		ef.ID = ir.EffectID(ir.SchemaVersion, ef)
		effects = append(effects, ef)
	}
	return effects, nil
}

func readOperands(words []shell.Word, cmdName string) (operands []shell.Word, patternFromOpt bool) {
	endOpts := false
	for i := 0; i < len(words); i++ {
		w := words[i]
		if !endOpts {
			if w.Text == "--" {
				endOpts = true
				continue
			}
			if strings.HasPrefix(w.Text, "-") && w.Text != "-" {
				if isPatternOption(cmdName, w.Text) {
					patternFromOpt = true
				}
				if optionTakesArg(cmdName, w.Text) && i+1 < len(words) && !strings.HasPrefix(words[i+1].Text, "-") {
					i++
				}
				continue
			}
		}
		operands = append(operands, w)
	}
	return operands, patternFromOpt
}

func isPatternOption(cmdName, opt string) bool {
	if cmdName != "grep" {
		return false
	}
	if strings.HasPrefix(opt, "--") {
		name, _, _ := strings.Cut(opt, "=")
		return name == "--regexp" || name == "--file"
	}
	if len(opt) >= 2 && opt[0] == '-' && !strings.HasPrefix(opt, "--") {
		// -ePAT / -e / -fFILE / clusters containing e or f as pattern carriers.
		// Lone -e/-f (len 2) or sticky -ePAT: treat as pattern-providing.
		if len(opt) == 2 {
			return opt[1] == 'e' || opt[1] == 'f'
		}
		// Sticky short form -ePAT or -fFILE.
		return opt[1] == 'e' || opt[1] == 'f'
	}
	return false
}

// optionTakesArg reports whether a discrete short/long option consumes the next
// argv word. Sticky forms like -n10 are not detected here and do not swallow.
func optionTakesArg(cmdName, opt string) bool {
	if strings.HasPrefix(opt, "--") {
		name, _, ok := strings.Cut(opt, "=")
		if ok {
			return false // --opt=value already carries its argument
		}
		switch cmdName {
		case "head":
			return name == "--lines" || name == "--bytes"
		case "grep":
			switch name {
			case "--regexp", "--file", "--max-count",
				"--after-context", "--before-context", "--context":
				return true
			}
		}
		return false
	}
	// Short cluster: only a lone -X (length 2) may take a separate argument.
	if len(opt) != 2 || opt[0] != '-' {
		return false
	}
	flag := opt[1]
	switch cmdName {
	case "head":
		return flag == 'n' || flag == 'c'
	case "grep":
		switch flag {
		case 'e', 'f', 'm', 'A', 'B', 'C', 'D':
			return true
		}
	}
	return false
}
