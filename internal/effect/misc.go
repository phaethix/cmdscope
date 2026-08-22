package effect

import (
	"strings"

	"github.com/phaethix/runmark/internal/ir"
	"github.com/phaethix/runmark/internal/logicalpath"
	"github.com/phaethix/runmark/internal/shell"
)

// ExtractMisc covers tee, touch, truncate, ln, and rmdir path effects.
func ExtractMisc(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string) ([]ir.Effect, []ir.Unknown) {
	if cmd == nil || len(cmd.Words) == 0 {
		return nil, nil
	}

	name := commandBasename(cmd.Words[0].Text)
	switch name {
	case "tee":
		return teeEffects(cmd, stage, cond, cwd)
	case "touch":
		return touchEffects(cmd, stage, cond, cwd)
	case "truncate":
		return truncateEffects(cmd, stage, cond, cwd)
	case "ln":
		return lnEffects(cmd, stage, cond, cwd)
	case "rmdir":
		return rmdirEffects(cmd, stage, cond, cwd)
	default:
		return nil, nil
	}
}

func teeEffects(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string) ([]ir.Effect, []ir.Unknown) {
	paths := positionalOperands(cmd.Words[1:], teeOptionTakesArg)
	cert := ir.Certain
	if hasBoolOption(cmd.Words[1:], "a", "--append") {
		cert = ir.Conditional
	}
	return writeEffects(paths, stage, cond, cwd, cert)
}

func touchEffects(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string) ([]ir.Effect, []ir.Unknown) {
	paths := positionalOperands(cmd.Words[1:], touchOptionTakesArg)
	return writeEffects(paths, stage, cond, cwd, ir.Certain)
}

func truncateEffects(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string) ([]ir.Effect, []ir.Unknown) {
	paths := positionalOperands(cmd.Words[1:], truncateOptionTakesArg)
	return writeEffects(paths, stage, cond, cwd, ir.Certain)
}

func lnEffects(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string) ([]ir.Effect, []ir.Unknown) {
	paths := positionalOperands(cmd.Words[1:], lnOptionTakesArg)
	if len(paths) < 2 {
		return nil, nil
	}
	// The last positional is the link; preceding are the sources read.
	srcs := paths[:len(paths)-1]
	dst := paths[len(paths)-1]

	var effects []ir.Effect
	for _, w := range srcs {
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
	dstTarget, _ := logicalpath.NormalizeLogicalPath(dst.Text, cwd)
	ef := ir.Effect{
		Kind:       ir.EffectWrite,
		RawTarget:  dst.Text,
		Target:     dstTarget,
		Stage:      stage,
		Certainty:  ir.Certain,
		Provenance: ir.FromCommand,
		Condition:  cond,
		Evidence:   []ir.Evidence{commandEvidence(dst.Start, dst.End, dst.Text)},
	}
	ef.ID = ir.EffectID(ir.SchemaVersion, ef)
	effects = append(effects, ef)
	return effects, nil
}

func rmdirEffects(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string) ([]ir.Effect, []ir.Unknown) {
	paths := positionalOperands(cmd.Words[1:], rmdirOptionTakesArg)
	return deleteEffects(paths, stage, cond, cwd)
}

func writeEffects(paths []shell.Word, stage int, cond ir.Condition, cwd string, cert ir.Certainty) ([]ir.Effect, []ir.Unknown) {
	var effects []ir.Effect
	for _, w := range paths {
		if w.Text == "-" {
			continue
		}
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
		effects = append(effects, ef)
	}
	return effects, nil
}

func deleteEffects(paths []shell.Word, stage int, cond ir.Condition, cwd string) ([]ir.Effect, []ir.Unknown) {
	var effects []ir.Effect
	for _, w := range paths {
		if w.Text == "-" {
			continue
		}
		target, _ := logicalpath.NormalizeLogicalPath(w.Text, cwd)
		ef := ir.Effect{
			Kind:       ir.EffectDelete,
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

func hasBoolOption(words []shell.Word, short, long string) bool {
	for _, w := range words {
		if w.Text == "-"+short || w.Text == long {
			return true
		}
	}
	return false
}

func teeOptionTakesArg(opt string) bool {
	if strings.HasPrefix(opt, "--") {
		name, _, ok := strings.Cut(opt, "=")
		if ok {
			return false
		}
		switch name {
		case "--output", "--suffix", "--size":
			return true
		}
		return false
	}
	// tee: -a (append) and -i (ignore interrupts) are boolean.
	return false
}

func touchOptionTakesArg(opt string) bool {
	if strings.HasPrefix(opt, "--") {
		name, _, ok := strings.Cut(opt, "=")
		if ok {
			return false
		}
		switch name {
		case "--reference", "--date", "--time":
			return true
		}
		return false
	}
	if len(opt) != 2 || opt[0] != '-' {
		return false
	}
	switch opt[1] {
	case 'r', 't', 'd':
		return true
	}
	return false
}

func truncateOptionTakesArg(opt string) bool {
	if strings.HasPrefix(opt, "--") {
		name, _, ok := strings.Cut(opt, "=")
		if ok {
			return false
		}
		return name == "--size"
	}
	if len(opt) != 2 || opt[0] != '-' {
		return false
	}
	return opt[1] == 's'
}

func lnOptionTakesArg(opt string) bool {
	if strings.HasPrefix(opt, "--") {
		name, _, ok := strings.Cut(opt, "=")
		if ok {
			return false
		}
		return name == "--target-directory"
	}
	if len(opt) != 2 || opt[0] != '-' {
		return false
	}
	// -t DIR takes an argument; -s/-f/-n/-i/-v/-b/-r/-T/-P/-L are boolean.
	return opt[1] == 't'
}

func rmdirOptionTakesArg(opt string) bool {
	// All rmdir options are boolean (-p/--parents, --ignore-fail-on-non-empty,
	// --verbose); treating any as taking an argument would drop a delete
	// target.
	return false
}
