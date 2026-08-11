package effect

import (
	"strings"

	"github.com/phaethix/cmdscope/internal/ir"
	"github.com/phaethix/cmdscope/internal/shell"
)

// ExtractInstall covers npm/pnpm install only. Registry contact is possible
// (not certain) because we never resolve or fetch the registry; dependency
// trees are intentionally not expanded.
func ExtractInstall(cmd *shell.SimpleCommand, stage int, cond ir.Condition) ([]ir.Effect, []ir.Unknown) {
	if cmd == nil || len(cmd.Words) == 0 {
		return nil, nil
	}
	name := commandBasename(cmd.Words[0].Text)
	if name != "npm" && name != "pnpm" {
		return nil, nil
	}

	pkgs, ok := installPackages(cmd.Words[1:], name)
	if !ok {
		return nil, nil
	}

	installTarget := strings.Join(pkgs, " ")
	if installTarget == "" {
		installTarget = "."
	}
	installRaw := installTarget

	install := ir.Effect{
		Kind:       ir.EffectInstall,
		RawTarget:  installRaw,
		Target:     installTarget,
		Stage:      stage,
		Certainty:  ir.Certain,
		Provenance: ir.FromCommand,
		Condition:  cond,
		Evidence:   []ir.Evidence{commandEvidence(cmd.Words[0].Start, cmd.Words[0].End, cmd.Words[0].Text)},
	}
	install.ID = ir.EffectID(ir.SchemaVersion, install)

	network := ir.Effect{
		Kind:       ir.EffectNetwork,
		RawTarget:  "registry",
		Target:     "registry",
		Stage:      stage,
		Certainty:  ir.Possible,
		Provenance: ir.FromCommand,
		Condition:  cond,
		Evidence:   []ir.Evidence{commandEvidence(cmd.Words[0].Start, cmd.Words[0].End, cmd.Words[0].Text)},
	}
	network.ID = ir.EffectID(ir.SchemaVersion, network)

	return []ir.Effect{install, network}, nil
}

func installPackages(words []shell.Word, cmdName string) (pkgs []string, isInstall bool) {
	endOpts := false
	sawSub := false
	for i := 0; i < len(words); i++ {
		w := words[i]
		if !endOpts && !sawSub {
			if w.Text == "--" {
				endOpts = true
				continue
			}
			if strings.HasPrefix(w.Text, "-") && w.Text != "-" {
				if installOptionTakesArg(w.Text) && i+1 < len(words) && !strings.HasPrefix(words[i+1].Text, "-") {
					i++
				}
				continue
			}
			if !isInstallSubcommand(cmdName, w.Text) {
				return nil, false
			}
			sawSub = true
			continue
		}
		if !sawSub {
			continue
		}
		if !endOpts && strings.HasPrefix(w.Text, "-") && w.Text != "-" {
			if installOptionTakesArg(w.Text) && i+1 < len(words) && !strings.HasPrefix(words[i+1].Text, "-") {
				i++
			}
			continue
		}
		if w.Text == "-" {
			continue
		}
		pkgs = append(pkgs, w.Text)
	}
	return pkgs, sawSub
}

func isInstallSubcommand(cmdName, sub string) bool {
	switch sub {
	case "install":
		return true
	case "i":
		return cmdName == "npm"
	default:
		return false
	}
}

func installOptionTakesArg(opt string) bool {
	if strings.HasPrefix(opt, "--") {
		name, _, ok := strings.Cut(opt, "=")
		if ok {
			return false
		}
		switch name {
		case "--registry", "--prefix", "--workspace", "--userconfig", "--cache":
			return true
		}
		return false
	}
	if len(opt) != 2 || opt[0] != '-' {
		return false
	}
	switch opt[1] {
	case 'C', 'w':
		return true
	}
	return false
}
