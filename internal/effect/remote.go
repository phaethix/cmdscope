package effect

import (
	"strconv"

	"github.com/phaethix/cmdscope/internal/ir"
	"github.com/phaethix/cmdscope/internal/shell"
)

// ExtractRemote detects curl/wget piped into a shell interpreter in one stage.
// Remote bodies are never fetched, so the match yields execute_remote plus a
// blocking remote_content unknown instead of invented filesystem writes.
func ExtractRemote(commands []shell.Node, stage int, cond ir.Condition) ([]ir.Effect, []ir.Unknown, []ir.Flag) {
	for i := 0; i+1 < len(commands); i++ {
		producer, ok := asSimpleCommand(commands[i])
		if !ok {
			continue
		}
		consumer, ok := asSimpleCommand(commands[i+1])
		if !ok {
			continue
		}
		prodName := commandBasename(producer.Words[0].Text)
		if prodName != "curl" && prodName != "wget" {
			continue
		}
		if !isShellInterpreter(commandBasename(consumer.Words[0].Text)) {
			continue
		}
		urls := networkURLOperands(producer.Words[1:], prodName)
		if len(urls) == 0 {
			continue
		}
		return remoteMatch(producer, consumer, urls, stage, cond)
	}
	return nil, nil, nil
}

func remoteMatch(producer, consumer *shell.SimpleCommand, urls []shell.Word, stage int, cond ir.Condition) ([]ir.Effect, []ir.Unknown, []ir.Flag) {
	effects := make([]ir.Effect, 0, len(urls)+3)
	for _, w := range urls {
		effects = append(effects, remoteEffect(ir.EffectNetwork, w.Text, w.Text, stage, cond, w.Start, w.End, w.Text))
	}
	prod0 := producer.Words[0]
	cons0 := consumer.Words[0]
	effects = append(effects,
		remoteEffect(ir.EffectProcess, prod0.Text, prod0.Text, stage, cond, prod0.Start, prod0.End, prod0.Text),
		remoteEffect(ir.EffectProcess, cons0.Text, cons0.Text, stage, cond, cons0.Start, cons0.End, cons0.Text),
	)
	primary := urls[0]
	effects = append(effects, remoteEffect(
		ir.EffectExecuteRemote, primary.Text, primary.Text, stage, cond,
		primary.Start, primary.End, primary.Text,
	))

	unk := ir.Unknown{
		Code:     ir.UnknownRemoteContent,
		Scope:    "stage:" + strconv.Itoa(stage),
		Message:  "remote content is not statically knowable",
		Evidence: []ir.Evidence{commandEvidence(primary.Start, primary.End, primary.Text)},
		Blocking: true,
	}
	flags := []ir.Flag{ir.FlagExternalNetwork, ir.FlagRemoteContent}
	return effects, []ir.Unknown{unk}, flags
}

func remoteEffect(kind ir.EffectKind, raw, target string, stage int, cond ir.Condition, start, end int, snippet string) ir.Effect {
	ef := ir.Effect{
		Kind:       kind,
		RawTarget:  raw,
		Target:     target,
		Stage:      stage,
		Certainty:  ir.Certain,
		Provenance: ir.FromCommand,
		Condition:  cond,
		Evidence:   []ir.Evidence{commandEvidence(start, end, snippet)},
	}
	ef.ID = ir.EffectID(ir.SchemaVersion, ef)
	return ef
}

func asSimpleCommand(n shell.Node) (*shell.SimpleCommand, bool) {
	cmd, ok := n.(*shell.SimpleCommand)
	if !ok || cmd == nil || len(cmd.Words) == 0 {
		return nil, false
	}
	return cmd, true
}

func isShellInterpreter(name string) bool {
	switch name {
	case "sh", "bash", "dash", "zsh":
		return true
	default:
		return false
	}
}
