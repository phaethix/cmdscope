package effect

import (
	"strconv"
	"strings"

	"github.com/phaethix/runmark/internal/ir"
	"github.com/phaethix/runmark/internal/logicalpath"
	"github.com/phaethix/runmark/internal/shell"
)

// ExtractArchive covers tar / zip / unzip file effects. tar extraction writes
// into the -C directory (or the current tree when -C is absent), and archive
// creation writes the named archive file.
func ExtractArchive(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string) ([]ir.Effect, []ir.Unknown) {
	if cmd == nil || len(cmd.Words) == 0 {
		return nil, nil
	}
	name := commandBasename(cmd.Words[0].Text)
	switch name {
	case "tar":
		return tarEffects(cmd, stage, cond, cwd)
	case "zip", "unzip":
		return zipEffects(cmd, stage, cond, cwd)
	default:
		return nil, nil
	}
}

func tarEffects(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string) ([]ir.Effect, []ir.Unknown) {
	extract, create_, extractDir, archiveFile, hasArchive := parseTarOptions(cmd.Words)

	var effects []ir.Effect
	var unknowns []ir.Unknown

	if extract {
		if extractDir.Text != "" {
			target, _ := logicalpath.NormalizeLogicalPath(extractDir.Text, cwd)
			ef := newArchiveWrite(extractDir, target, stage, cond)
			effects = append(effects, ef)
		} else {
			// No -C: extraction lands in the current tree, which we cannot
			// enumerate statically.
			target, _ := logicalpath.NormalizeLogicalPath("./**", cwd)
			ef := ir.Effect{
				Kind:       ir.EffectWrite,
				RawTarget:  "./**",
				Target:     target,
				Stage:      stage,
				Certainty:  ir.Certain,
				Provenance: ir.FromCommand,
				Condition:  cond,
				Evidence:   []ir.Evidence{commandEvidence(cmd.Words[0].Start, cmd.Words[0].End, cmd.Words[0].Text)},
			}
			ef.ID = ir.EffectID(ir.SchemaVersion, ef)
			effects = append(effects, ef)
			unknowns = append(unknowns, ir.Unknown{
				Code:     ir.UnknownGlobRuntimeDependent,
				Scope:    "stage:" + strconv.Itoa(stage),
				Message:  "extraction target is runtime-dependent (no -C directory given)",
				Evidence: []ir.Evidence{commandEvidence(cmd.Words[0].Start, cmd.Words[0].End, cmd.Words[0].Text)},
				Blocking: false,
			})
		}
	}
	if create_ && hasArchive {
		target, _ := logicalpath.NormalizeLogicalPath(archiveFile.Text, cwd)
		ef := newArchiveWrite(archiveFile, target, stage, cond)
		effects = append(effects, ef)
	}
	return effects, unknowns
}

func newArchiveWrite(w shell.Word, target string, stage int, cond ir.Condition) ir.Effect {
	ef := ir.Effect{
		Kind:       ir.EffectWrite,
		RawTarget:  w.Text,
		Target:     target,
		Stage:      stage,
		Certainty:  ir.Certain,
		Provenance: ir.FromCommand,
		Condition:  cond,
		Evidence:   []ir.Evidence{commandEvidence(w.Start, w.End, w.Text)},
	}
	ef.ID = ir.EffectID(ir.SchemaVersion, ef)
	return ef
}

// parseTarOptions decodes tar's short-option clusters (e.g. -xzf) and long
// options. It returns the extract/create flags, the -C directory, and the
// -f archive operand.
func parseTarOptions(words []shell.Word) (extract, create bool, extractDir, archiveFile shell.Word, hasArchive bool) {
	i := 1
	for i < len(words) {
		w := words[i]
		if strings.HasPrefix(w.Text, "-") && w.Text != "-" {
			if strings.HasPrefix(w.Text, "--") {
				switch {
				case w.Text == "--extract" || w.Text == "--get":
					extract = true
				case w.Text == "--create":
					create = true
				case strings.HasPrefix(w.Text, "--directory"):
					if val, ok := optValue(w, words, &i); ok {
						extractDir = val
					}
				case strings.HasPrefix(w.Text, "--file"):
					if val, ok := optValue(w, words, &i); ok {
						archiveFile = val
						hasArchive = true
					}
				}
				i++
				continue
			}
			opts := w.Text[1:]
			for j := 0; j < len(opts); j++ {
				switch opts[j] {
				case 'x':
					extract = true
				case 'c':
					create = true
				case 'C':
					if j+1 < len(opts) {
						extractDir = shell.Word{Text: opts[j+1:], Start: w.Start, End: w.End}
						j = len(opts)
					} else if i+1 < len(words) && !strings.HasPrefix(words[i+1].Text, "-") {
						extractDir = words[i+1]
						i++
					}
				case 'f':
					if j+1 < len(opts) {
						archiveFile = shell.Word{Text: opts[j+1:], Start: w.Start, End: w.End}
						hasArchive = true
						j = len(opts)
					} else if i+1 < len(words) && !strings.HasPrefix(words[i+1].Text, "-") {
						archiveFile = words[i+1]
						hasArchive = true
						i++
					}
				}
			}
			i++
			continue
		}
		break
	}
	return extract, create, extractDir, archiveFile, hasArchive
}

func optValue(opt shell.Word, words []shell.Word, i *int) (shell.Word, bool) {
	name, val, hasVal := strings.Cut(opt.Text, "=")
	_ = name
	if hasVal {
		if val == "" || val == "-" {
			return shell.Word{}, false
		}
		return shell.Word{Text: val, Start: opt.Start, End: opt.End}, true
	}
	if *i+1 < len(words) && !strings.HasPrefix(words[*i+1].Text, "-") {
		*i++
		dst := words[*i]
		if dst.Text == "-" {
			return shell.Word{}, false
		}
		return dst, true
	}
	return shell.Word{}, false
}

func zipEffects(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string) ([]ir.Effect, []ir.Unknown) {
	name := commandBasename(cmd.Words[0].Text)
	var archiveFile shell.Word
	var hasArchive bool
	var extractDir shell.Word

	if name == "zip" {
		// zip archive.zip src... → archive operand is the first positional.
		for i := 1; i < len(cmd.Words); i++ {
			w := cmd.Words[i]
			if strings.HasPrefix(w.Text, "-") && w.Text != "-" {
				if strings.HasPrefix(w.Text, "--") {
					if name2, val, ok := strings.Cut(w.Text, "="); ok && (name2 == "--file" || name2 == "-f") {
						if val != "" && val != "-" {
							archiveFile = shell.Word{Text: val, Start: w.Start, End: w.End}
							hasArchive = true
						}
					}
					continue
				}
				if i+1 < len(cmd.Words) && (w.Text == "-r" || w.Text == "--recurse-paths") {
					continue
				}
				continue
			}
			archiveFile = w
			hasArchive = true
			break
		}
		if hasArchive {
			target, _ := logicalpath.NormalizeLogicalPath(archiveFile.Text, cwd)
			ef := newArchiveWrite(archiveFile, target, stage, cond)
			return []ir.Effect{ef}, nil
		}
		return nil, nil
	}

	// unzip archive.zip [-d dir]
	for i := 1; i < len(cmd.Words); i++ {
		w := cmd.Words[i]
		if strings.HasPrefix(w.Text, "-") && w.Text != "-" {
			if w.Text == "-d" && i+1 < len(cmd.Words) {
				extractDir = cmd.Words[i+1]
				i++
			}
			continue
		}
		if !hasArchive {
			archiveFile = w
			hasArchive = true
		}
	}
	if !hasArchive {
		return nil, nil
	}
	if extractDir.Text != "" {
		target, _ := logicalpath.NormalizeLogicalPath(extractDir.Text, cwd)
		ef := newArchiveWrite(extractDir, target, stage, cond)
		return []ir.Effect{ef}, nil
	}
	target, _ := logicalpath.NormalizeLogicalPath("./**", cwd)
	ef := ir.Effect{
		Kind:       ir.EffectWrite,
		RawTarget:  "./**",
		Target:     target,
		Stage:      stage,
		Certainty:  ir.Certain,
		Provenance: ir.FromCommand,
		Condition:  cond,
		Evidence:   []ir.Evidence{commandEvidence(cmd.Words[0].Start, cmd.Words[0].End, cmd.Words[0].Text)},
	}
	ef.ID = ir.EffectID(ir.SchemaVersion, ef)
	unk := ir.Unknown{
		Code:     ir.UnknownGlobRuntimeDependent,
		Scope:    "stage:" + strconv.Itoa(stage),
		Message:  "extraction target is runtime-dependent (no -d directory given)",
		Evidence: []ir.Evidence{commandEvidence(cmd.Words[0].Start, cmd.Words[0].End, cmd.Words[0].Text)},
		Blocking: false,
	}
	return []ir.Effect{ef}, []ir.Unknown{unk}
}
