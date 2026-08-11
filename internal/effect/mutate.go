package effect

import (
	"strconv"
	"strings"

	"github.com/phaethix/cmdscope/internal/ir"
	"github.com/phaethix/cmdscope/internal/logicalpath"
	"github.com/phaethix/cmdscope/internal/shell"
)

// ExtractMutate covers rm/cp/mv path effects. Globs stay literal and raise
// glob_runtime_dependent so we never invent an expansion; missing operands
// become unsupported_command instead of guessing paths.
func ExtractMutate(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string) ([]ir.Effect, []ir.Unknown) {
	if cmd == nil || len(cmd.Words) == 0 {
		return nil, nil
	}

	name := commandBasename(cmd.Words[0].Text)
	switch name {
	case "rm", "cp", "mv":
	default:
		return nil, nil
	}

	args := parseMutateArgs(cmd.Words[1:], name)
	switch name {
	case "rm":
		if len(args.paths) == 0 {
			return nil, []ir.Unknown{insufficientOperandUnknown(cmd.Words[0], stage, name)}
		}
		return pathEffects(args.paths, stage, cond, cwd, ir.EffectDelete)
	case "cp":
		return copyOrMoveFromArgs(args, cmd.Words[0], stage, cond, cwd, name, ir.EffectRead)
	default: // mv
		return copyOrMoveFromArgs(args, cmd.Words[0], stage, cond, cwd, name, ir.EffectDelete)
	}
}

func copyOrMoveFromArgs(args mutateArgs, argv0 shell.Word, stage int, cond ir.Condition, cwd, name string, srcKind ir.EffectKind) ([]ir.Effect, []ir.Unknown) {
	var srcs []shell.Word
	var dst shell.Word
	if args.destFromOpt != nil {
		// -t already named the destination; remaining positionals are sources only.
		if len(args.paths) == 0 {
			return nil, []ir.Unknown{insufficientOperandUnknown(argv0, stage, name)}
		}
		srcs = args.paths
		dst = *args.destFromOpt
	} else {
		if len(args.paths) < 2 {
			return nil, []ir.Unknown{insufficientOperandUnknown(argv0, stage, name)}
		}
		dst = args.paths[len(args.paths)-1]
		srcs = args.paths[:len(args.paths)-1]
	}
	return copyOrMoveEffects(srcs, dst, stage, cond, cwd, srcKind)
}

type mutateArgs struct {
	paths       []shell.Word
	destFromOpt *shell.Word
}

func copyOrMoveEffects(srcs []shell.Word, dst shell.Word, stage int, cond ir.Condition, cwd string, srcKind ir.EffectKind) ([]ir.Effect, []ir.Unknown) {
	var effects []ir.Effect
	var unknowns []ir.Unknown
	for _, src := range srcs {
		ef, unk := onePathEffect(src, stage, cond, cwd, srcKind)
		effects = append(effects, ef)
		if unk != nil {
			unknowns = append(unknowns, *unk)
		}
	}
	ef, unk := onePathEffect(dst, stage, cond, cwd, ir.EffectWrite)
	effects = append(effects, ef)
	if unk != nil {
		unknowns = append(unknowns, *unk)
	}
	return effects, unknowns
}

func pathEffects(operands []shell.Word, stage int, cond ir.Condition, cwd string, kind ir.EffectKind) ([]ir.Effect, []ir.Unknown) {
	effects := make([]ir.Effect, 0, len(operands))
	var unknowns []ir.Unknown
	for _, w := range operands {
		ef, unk := onePathEffect(w, stage, cond, cwd, kind)
		effects = append(effects, ef)
		if unk != nil {
			unknowns = append(unknowns, *unk)
		}
	}
	return effects, unknowns
}

func onePathEffect(w shell.Word, stage int, cond ir.Condition, cwd string, kind ir.EffectKind) (ir.Effect, *ir.Unknown) {
	target, flags := logicalpath.NormalizeLogicalPath(w.Text, cwd)
	ef := ir.Effect{
		Kind:       kind,
		RawTarget:  w.Text,
		Target:     target,
		Stage:      stage,
		Certainty:  ir.Certain,
		Provenance: ir.FromCommand,
		Condition:  cond,
		Evidence:   []ir.Evidence{commandEvidence(w.Start, w.End, w.Text)},
	}
	ef.ID = ir.EffectID(ir.SchemaVersion, ef)
	if !flags.Has(logicalpath.PathHasGlob) {
		return ef, nil
	}
	unk := ir.Unknown{
		Code:     ir.UnknownGlobRuntimeDependent,
		Scope:    "stage:" + strconv.Itoa(stage),
		Message:  "glob pattern is runtime-dependent " + strconv.Quote(w.Text),
		Evidence: []ir.Evidence{commandEvidence(w.Start, w.End, w.Text)},
		Blocking: false,
	}
	return ef, &unk
}

func insufficientOperandUnknown(argv0 shell.Word, stage int, name string) ir.Unknown {
	return ir.Unknown{
		Code:     ir.UnknownUnsupportedCommand,
		Scope:    "stage:" + strconv.Itoa(stage),
		Message:  name + " is missing required path operands",
		Evidence: []ir.Evidence{commandEvidence(argv0.Start, argv0.End, argv0.Text)},
		Blocking: false,
	}
}

func parseMutateArgs(words []shell.Word, cmdName string) mutateArgs {
	var args mutateArgs
	endOpts := false
	for i := 0; i < len(words); i++ {
		w := words[i]
		if !endOpts {
			if w.Text == "--" {
				endOpts = true
				continue
			}
			if strings.HasPrefix(w.Text, "-") && w.Text != "-" {
				if isTargetDirOption(cmdName, w.Text) {
					if dest, ok := targetDirOperand(w, words, &i); ok {
						args.destFromOpt = &dest
					}
					continue
				}
				if mutateOptionTakesArg(cmdName, w.Text) && i+1 < len(words) && !strings.HasPrefix(words[i+1].Text, "-") {
					i++
				}
				continue
			}
		}
		if w.Text == "-" {
			continue
		}
		args.paths = append(args.paths, w)
	}
	return args
}

func isTargetDirOption(cmdName, opt string) bool {
	if cmdName != "cp" && cmdName != "mv" {
		return false
	}
	if strings.HasPrefix(opt, "--") {
		name, _, _ := strings.Cut(opt, "=")
		return name == "--target-directory"
	}
	return opt == "-t"
}

func targetDirOperand(opt shell.Word, words []shell.Word, i *int) (shell.Word, bool) {
	if strings.HasPrefix(opt.Text, "--") {
		if _, val, ok := strings.Cut(opt.Text, "="); ok {
			// Sticky --target-directory=DEST: no separate Word span; reuse opt span.
			return shell.Word{Text: val, Start: opt.Start, End: opt.End}, val != "" && val != "-"
		}
	}
	if *i+1 < len(words) && !strings.HasPrefix(words[*i+1].Text, "-") {
		*i++
		dest := words[*i]
		if dest.Text == "-" {
			return shell.Word{}, false
		}
		return dest, true
	}
	return shell.Word{}, false
}

func mutateOptionTakesArg(cmdName, opt string) bool {
	if strings.HasPrefix(opt, "--") {
		name, _, ok := strings.Cut(opt, "=")
		if ok {
			return false
		}
		switch name {
		case "--suffix":
			return cmdName == "cp" || cmdName == "mv"
		case "--context":
			return cmdName == "cp"
		}
		return false
	}
	if len(opt) != 2 || opt[0] != '-' {
		return false
	}
	return opt[1] == 'S' && (cmdName == "cp" || cmdName == "mv")
}
