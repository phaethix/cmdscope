package effect

import (
	"strconv"
	"strings"

	"github.com/phaethix/runmark/internal/ir"
	"github.com/phaethix/runmark/internal/logicalpath"
	"github.com/phaethix/runmark/internal/shell"
)

// ExtractFind covers find path effects: -delete removes the starting points,
// -exec/-execdir runs a command per match (not statically knowable), and a
// bare find only traverses (reads) the starting points.
func ExtractFind(cmd *shell.SimpleCommand, stage int, cond ir.Condition, cwd string) ([]ir.Effect, []ir.Unknown) {
	if cmd == nil || len(cmd.Words) == 0 {
		return nil, nil
	}
	if commandBasename(cmd.Words[0].Text) != "find" {
		return nil, nil
	}

	startPoints, hasDelete, hasExec := parseFindArgs(cmd.Words)
	if len(startPoints) == 0 {
		return nil, nil
	}

	var effects []ir.Effect
	var unknowns []ir.Unknown

	switch {
	case hasDelete:
		for _, sp := range startPoints {
			target, flags := logicalpath.NormalizeLogicalPath(sp.Text, cwd)
			ef := ir.Effect{
				Kind:       ir.EffectDelete,
				RawTarget:  sp.Text,
				Target:     target,
				Stage:      stage,
				Certainty:  ir.Certain,
				Provenance: ir.FromCommand,
				Condition:  cond,
				Evidence:   []ir.Evidence{commandEvidence(sp.Start, sp.End, sp.Text)},
			}
			ef.ID = ir.EffectID(ir.SchemaVersion, ef)
			effects = append(effects, ef)
			if flags.Has(logicalpath.PathHasGlob) {
				unknowns = append(unknowns, ir.Unknown{
					Code:     ir.UnknownGlobRuntimeDependent,
					Scope:    "stage:" + strconv.Itoa(stage),
					Message:  "glob pattern is runtime-dependent " + strconv.Quote(sp.Text),
					Evidence: []ir.Evidence{commandEvidence(sp.Start, sp.End, sp.Text)},
					Blocking: false,
				})
			}
		}
	case hasExec:
		ef := ir.Effect{
			Kind:       ir.EffectProcess,
			RawTarget:  cmd.Words[0].Text,
			Target:     cmd.Words[0].Text,
			Stage:      stage,
			Certainty:  ir.Certain,
			Provenance: ir.FromCommand,
			Condition:  cond,
			Evidence:   []ir.Evidence{commandEvidence(cmd.Words[0].Start, cmd.Words[0].End, cmd.Words[0].Text)},
		}
		ef.ID = ir.EffectID(ir.SchemaVersion, ef)
		effects = append(effects, ef)
		unknowns = append(unknowns, ir.Unknown{
			Code:     ir.UnknownEffectsRuntimeDependent,
			Scope:    "stage:" + strconv.Itoa(stage),
			Message:  "find -exec runs a command per match; the matches are not statically knowable",
			Evidence: []ir.Evidence{commandEvidence(cmd.Words[0].Start, cmd.Words[0].End, cmd.Words[0].Text)},
			Blocking: false,
		})
	default:
		for _, sp := range startPoints {
			target, _ := logicalpath.NormalizeLogicalPath(sp.Text, cwd)
			ef := ir.Effect{
				Kind:       ir.EffectRead,
				RawTarget:  sp.Text,
				Target:     target,
				Stage:      stage,
				Certainty:  ir.Certain,
				Provenance: ir.FromCommand,
				Condition:  cond,
				Evidence:   []ir.Evidence{commandEvidence(sp.Start, sp.End, sp.Text)},
			}
			ef.ID = ir.EffectID(ir.SchemaVersion, ef)
			effects = append(effects, ef)
		}
	}
	return effects, unknowns
}

// parseFindArgs splits find's global options and starting points from the
// expression, then scans the expression for -delete / -exec / -execdir.
func parseFindArgs(words []shell.Word) (startPoints []shell.Word, hasDelete, hasExec bool) {
	i := 1
	for i < len(words) {
		w := words[i]
		if strings.HasPrefix(w.Text, "-") && w.Text != "-" {
			switch w.Text {
			case "-H", "-L", "-P":
				i++
				continue
			case "-D", "-O":
				i += 2
				continue
			}
			break
		}
		startPoints = append(startPoints, w)
		i++
	}
	for ; i < len(words); i++ {
		w := words[i]
		switch w.Text {
		case "-delete":
			hasDelete = true
		case "-exec", "-execdir":
			hasExec = true
			// Skip the command and its args until the terminator ; or +.
			for j := i + 1; j < len(words); j++ {
				if words[j].Text == ";" || words[j].Text == "+" {
					i = j
					break
				}
			}
		}
	}
	return startPoints, hasDelete, hasExec
}
