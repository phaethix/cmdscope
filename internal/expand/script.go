package expand

import (
	"strings"

	"github.com/phaethix/runmark/internal/ir"
	"github.com/phaethix/runmark/internal/shell"
)

// ExpandShellScript analyzes `sh|bash|dash|zsh -c <literal>`. Dynamic -c
// payloads become interpreter_dynamic_code; literals are re-parsed by the
// internal shell parser only — never executed.
func ExpandShellScript(cmd *shell.SimpleCommand, stage int) ExpansionResult {
	if cmd == nil || len(cmd.Words) == 0 {
		return ExpansionResult{}
	}
	if !isShellInterpreterName(commandBasename(cmd.Words[0].Text)) {
		return ExpansionResult{}
	}
	bodyWord, ok := findDashCPayload(cmd.Words)
	if !ok {
		return ExpansionResult{}
	}

	body, literal := unwrapScriptLiteral(bodyWord.Text)
	if !literal || body == "" {
		return ExpansionResult{
			Applied: true,
			Unknowns: []ir.Unknown{expandUnknown(
				ir.UnknownInterpreterDynamicCode, stage, cmd.Words[0],
				"shell -c payload is dynamic and cannot be analyzed statically",
			)},
		}
	}

	toks, err := shell.Lex(body)
	if err != nil {
		return ExpansionResult{
			Applied: true,
			Unknowns: []ir.Unknown{expandUnknown(
				ir.UnknownParseError, stage, cmd.Words[0],
				"shell -c payload could not be lexed",
			)},
			Evidence: []ir.Evidence{commandWordEvidence(bodyWord)},
		}
	}
	root, err := shell.Parse(toks)
	if err != nil && root == nil {
		return ExpansionResult{
			Applied: true,
			Unknowns: []ir.Unknown{expandUnknown(
				ir.UnknownParseError, stage, cmd.Words[0],
				"shell -c payload could not be parsed",
			)},
			Evidence: []ir.Evidence{commandWordEvidence(bodyWord)},
		}
	}
	return ExpansionResult{
		Applied:  true,
		Nodes:    flattenCommands(root),
		Evidence: []ir.Evidence{commandWordEvidence(bodyWord)},
	}
}

// ExpandPython conservatively handles `python|python3 -c …`. Python bodies are
// never lowered to file effects: only a blocking interpreter_dynamic_code
// unknown is emitted (outer process comes from the process extractor).
func ExpandPython(cmd *shell.SimpleCommand, stage int) ExpansionResult {
	if cmd == nil || len(cmd.Words) == 0 {
		return ExpansionResult{}
	}
	if !isPythonName(commandBasename(cmd.Words[0].Text)) {
		return ExpansionResult{}
	}
	bodyWord, ok := findDashCPayload(cmd.Words)
	if !ok {
		return ExpansionResult{}
	}
	return ExpansionResult{
		Applied: true,
		Unknowns: []ir.Unknown{expandUnknown(
			ir.UnknownInterpreterDynamicCode, stage, cmd.Words[0],
			"python -c bodies are not statically analyzed for file effects",
		)},
		Evidence: []ir.Evidence{commandWordEvidence(bodyWord)},
	}
}

func isShellInterpreterName(name string) bool {
	switch name {
	case "sh", "bash", "dash", "zsh":
		return true
	default:
		return false
	}
}

func isPythonName(name string) bool {
	switch name {
	case "python", "python3":
		return true
	default:
		return false
	}
}

func findDashCPayload(words []shell.Word) (shell.Word, bool) {
	for i := 1; i < len(words); i++ {
		w := words[i].Text
		if w == "-c" {
			if i+1 >= len(words) {
				return shell.Word{}, true // matched -c but missing body → caller treats as dynamic
			}
			return words[i+1], true
		}
		// Combined short form is uncommon; reject as unmatched so we do not
		// silently invent a body from -cSCRIPT without a separator.
	}
	return shell.Word{}, false
}

// unwrapScriptLiteral strips one layer of matching quotes. Single-quoted
// bodies are always static (the outer shell does not expand them). Double-
// quoted or unquoted bodies that still contain $ or backticks are dynamic.
func unwrapScriptLiteral(text string) (body string, literal bool) {
	if text == "" {
		return "", false
	}
	if len(text) >= 2 && text[0] == '\'' && text[len(text)-1] == '\'' {
		body = text[1 : len(text)-1]
		if body == "" {
			return "", false
		}
		return body, true
	}
	body = text
	if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		body = text[1 : len(text)-1]
	}
	if body == "" || strings.ContainsAny(body, "$`") {
		return "", false
	}
	return body, true
}

func commandWordEvidence(w shell.Word) ir.Evidence {
	ev := ir.Evidence{Source: ir.EvidenceCommand, Snippet: w.Text}
	if w.Start >= 0 && w.End > w.Start {
		ev.StartByte = new(w.Start)
		ev.EndByte = new(w.End)
	}
	return ev
}
