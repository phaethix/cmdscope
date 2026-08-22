package effect

import (
	"strconv"

	"github.com/phaethix/runmark/internal/ir"
	"github.com/phaethix/runmark/internal/shell"
)

// ExtractXargs conservatively records that xargs runs a command per input
// line. The command and its effects are not statically knowable, so only a
// process effect plus a non-blocking unknown are emitted.
func ExtractXargs(cmd *shell.SimpleCommand, stage int, cond ir.Condition) ([]ir.Effect, []ir.Unknown) {
	if cmd == nil || len(cmd.Words) == 0 {
		return nil, nil
	}
	if commandBasename(cmd.Words[0].Text) != "xargs" {
		return nil, nil
	}
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
	unk := ir.Unknown{
		Code:     ir.UnknownEffectsRuntimeDependent,
		Scope:    "stage:" + strconv.Itoa(stage),
		Message:  "xargs runs a command per input; effects are not statically knowable",
		Evidence: []ir.Evidence{commandEvidence(cmd.Words[0].Start, cmd.Words[0].End, cmd.Words[0].Text)},
		Blocking: false,
	}
	return []ir.Effect{ef}, []ir.Unknown{unk}
}
