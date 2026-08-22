package effect

import (
	"strconv"
	"strings"

	"github.com/phaethix/runmark/internal/ir"
	"github.com/phaethix/runmark/internal/logicalpath"
	"github.com/phaethix/runmark/internal/shell"
)

// ExtractGit covers git subcommands that touch files, the network, or leave
// destructive worktree side-effects. It does not enumerate worktree changes
// for reset --hard / clean because those are runtime-only.
func ExtractGit(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string) ([]ir.Effect, []ir.Unknown, []ir.Flag) {
	if cmd == nil || len(cmd.Words) < 2 {
		return nil, nil, nil
	}
	if commandBasename(cmd.Words[0].Text) != "git" {
		return nil, nil, nil
	}

	sub := cmd.Words[1].Text
	switch sub {
	case "push":
		return gitPush(cmd, stage, cond)
	case "reset":
		return gitReset(cmd, stage, cond)
	case "clean":
		return gitClean(cmd, stage, cond)
	case "rm":
		return gitRm(cmd, stage, cond, cwd)
	case "checkout":
		return gitCheckout(cmd, stage, cond, cwd)
	case "restore":
		return gitRestore(cmd, stage, cond, cwd)
	case "add", "commit":
		return gitWriteDotGit(cmd, stage, cond, cwd)
	default:
		return nil, nil, nil
	}
}

func gitPush(cmd *shell.SimpleCommand, stage int, cond ir.Condition) ([]ir.Effect, []ir.Unknown, []ir.Flag) {
	hasForce := false
	for _, w := range cmd.Words[2:] {
		switch w.Text {
		case "--force", "-f", "--force-with-lease":
			hasForce = true
		}
	}
	ef := ir.Effect{
		Kind:       ir.EffectNetwork,
		RawTarget:  "git-remote",
		Target:     "git-remote",
		Stage:      stage,
		Certainty:  ir.Certain,
		Provenance: ir.FromCommand,
		Condition:  cond,
		Evidence:   []ir.Evidence{commandEvidence(cmd.Words[0].Start, cmd.Words[1].End, cmd.Words[0].Text+" "+cmd.Words[1].Text)},
	}
	ef.ID = ir.EffectID(ir.SchemaVersion, ef)
	if hasForce {
		return []ir.Effect{ef}, nil, []ir.Flag{ir.FlagDestructive}
	}
	return []ir.Effect{ef}, nil, nil
}

func gitReset(cmd *shell.SimpleCommand, stage int, cond ir.Condition) ([]ir.Effect, []ir.Unknown, []ir.Flag) {
	hasHard := false
	for _, w := range cmd.Words[2:] {
		// Exact match only: substrings would let unrelated words like a
		// branch named "hardware" flip the destructive flag.
		if w.Text == "--hard" {
			hasHard = true
		}
	}
	if !hasHard {
		return nil, nil, nil
	}
	unk := ir.Unknown{
		Code:     ir.UnknownEffectsRuntimeDependent,
		Scope:    "stage:" + strconv.Itoa(stage),
		Message:  "git reset --hard may destroy uncommitted worktree changes",
		Evidence: []ir.Evidence{commandEvidence(cmd.Words[0].Start, cmd.Words[1].End, cmd.Words[0].Text+" "+cmd.Words[1].Text)},
		Blocking: false,
	}
	return nil, []ir.Unknown{unk}, []ir.Flag{ir.FlagDestructive}
}

func gitClean(cmd *shell.SimpleCommand, stage int, cond ir.Condition) ([]ir.Effect, []ir.Unknown, []ir.Flag) {
	unk := ir.Unknown{
		Code:     ir.UnknownEffectsRuntimeDependent,
		Scope:    "stage:" + strconv.Itoa(stage),
		Message:  "git clean may remove untracked files",
		Evidence: []ir.Evidence{commandEvidence(cmd.Words[0].Start, cmd.Words[1].End, cmd.Words[0].Text+" "+cmd.Words[1].Text)},
		Blocking: false,
	}
	return nil, []ir.Unknown{unk}, []ir.Flag{ir.FlagDestructive}
}

func gitRm(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string) ([]ir.Effect, []ir.Unknown, []ir.Flag) {
	paths := positionalOperands(cmd.Words[2:], noOptionArgs)
	if len(paths) == 0 {
		return nil, nil, nil
	}
	effects, unknowns := pathEffects(paths, stage, cond, cwd, ir.EffectDelete)
	return effects, unknowns, nil
}

func gitCheckout(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string) ([]ir.Effect, []ir.Unknown, []ir.Flag) {
	paths := gitCheckoutPaths(cmd.Words[2:])
	if len(paths) == 0 {
		return nil, nil, nil
	}
	effects, unknowns := pathEffects(paths, stage, cond, cwd, ir.EffectWrite)
	return effects, unknowns, nil
}

func gitRestore(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string) ([]ir.Effect, []ir.Unknown, []ir.Flag) {
	paths := positionalOperands(cmd.Words[2:], noOptionArgs)
	if len(paths) == 0 {
		return nil, nil, nil
	}
	effects, unknowns := pathEffects(paths, stage, cond, cwd, ir.EffectWrite)
	return effects, unknowns, nil
}

func gitWriteDotGit(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string) ([]ir.Effect, []ir.Unknown, []ir.Flag) {
	target, _ := logicalpath.NormalizeLogicalPath("./.git", cwd)
	ef := ir.Effect{
		Kind:       ir.EffectWrite,
		RawTarget:  "./.git",
		Target:     target,
		Stage:      stage,
		Certainty:  ir.Certain,
		Provenance: ir.FromCommand,
		Condition:  cond,
		Evidence:   []ir.Evidence{commandEvidence(cmd.Words[0].Start, cmd.Words[1].End, cmd.Words[0].Text+" "+cmd.Words[1].Text)},
	}
	ef.ID = ir.EffectID(ir.SchemaVersion, ef)
	return []ir.Effect{ef}, nil, nil
}

func gitCheckoutPaths(words []shell.Word) []shell.Word {
	sawDashDash := false
	var out []shell.Word
	for _, w := range words {
		if w.Text == "--" {
			sawDashDash = true
			continue
		}
		if !sawDashDash {
			if strings.HasPrefix(w.Text, "-") && w.Text != "-" {
				continue
			}
			continue
		}
		if w.Text == "-" {
			continue
		}
		out = append(out, w)
	}
	return out
}
