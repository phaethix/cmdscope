package effect

import (
	"strings"

	"github.com/phaethix/runmark/internal/ir"
	"github.com/phaethix/runmark/internal/shell"
)

// ExtractInstall covers package-manager install subcommands: npm/pnpm install,
// pip/pip3 install, cargo install, yarn/bun add, and npx <pkg>. Registry
// contact is possible (not certain) because we never resolve or fetch the
// registry; dependency trees are intentionally not expanded.
func ExtractInstall(cmd *shell.SimpleCommand, stage int, cond ir.Condition) ([]ir.Effect, []ir.Unknown) {
	if cmd == nil || len(cmd.Words) == 0 {
		return nil, nil
	}
	name := commandBasename(cmd.Words[0].Text)

	installSub, ok := installSubcommand(name)
	if !ok {
		return nil, nil
	}

	pkgs, isInstall := installPackages(cmd.Words[1:], name, installSub)
	if !isInstall {
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

func installPackages(words []shell.Word, cmdName, installSub string) (pkgs []string, isInstall bool) {
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
			// npx takes the package name directly as its first operand;
			// everything after it is that package's arguments, not more
			// packages to install.
			if installSub == "" {
				sawSub = true
				pkgs = append(pkgs, w.Text)
				break
			}
			if !isInstallSubcommand(cmdName, w.Text, installSub) {
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

func installSubcommand(name string) (string, bool) {
	switch name {
	case "npm", "pnpm", "pip", "pip3", "cargo":
		return "install", true
	case "yarn", "bun":
		return "add", true
	case "npx":
		return "", true
	default:
		return "", false
	}
}

func isInstallSubcommand(cmdName, sub, installSub string) bool {
	if sub == installSub {
		return true
	}
	// npm also accepts the shorthand `i` for install.
	if cmdName == "npm" && sub == "i" {
		return installSub == "install"
	}
	return false
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
	case 'C', 'w', 'r', 'e', 'i', 'f', 'p':
		return true
	}
	return false
}
