package analyzer

import (
	"strconv"
	"strings"

	"github.com/phaethix/cmdscope/internal/ir"
	"github.com/phaethix/cmdscope/internal/shell"
)

// CollectUncertainties walks stage commands for substitution, unexpanded
// globs, and env refs absent from env. It does not expand or resolve any of
// them — those facts are runtime-only.
func CollectUncertainties(stages []shell.Stage, env map[string]string) []ir.Unknown {
	var out []ir.Unknown
	for _, st := range stages {
		stage := max(st.Index-1, 0)
		for _, n := range st.Commands {
			out = append(out, walkUncertainty(n, stage, env)...)
		}
	}
	return out
}

func walkUncertainty(n shell.Node, stage int, env map[string]string) []ir.Unknown {
	switch v := n.(type) {
	case *shell.SimpleCommand:
		var out []ir.Unknown
		for _, w := range v.Words {
			out = append(out, wordUncertainties(w, stage, env)...)
		}
		for _, r := range v.Redirects {
			out = append(out, wordUncertainties(r.Target, stage, env)...)
		}
		return out
	case *shell.Pipeline:
		var out []ir.Unknown
		for _, c := range v.Commands {
			out = append(out, walkUncertainty(c, stage, env)...)
		}
		return out
	case *shell.Sequence:
		var out []ir.Unknown
		for _, item := range v.Items {
			out = append(out, walkUncertainty(item, stage, env)...)
		}
		return out
	case *shell.Binary:
		return append(walkUncertainty(v.Left, stage, env), walkUncertainty(v.Right, stage, env)...)
	case *shell.Subshell:
		return walkUncertainty(v.Body, stage, env)
	case *shell.CommandSubstitution:
		return []ir.Unknown{uncertainty(ir.UnknownCommandSubstitution, stage, v.Start, v.End, v.Raw, "command substitution is not expanded")}
	default:
		return nil
	}
}

func wordUncertainties(w shell.Word, stage int, env map[string]string) []ir.Unknown {
	// A substitution token owns the whole uncertainty for that span; scanning
	// inside it would double-count env/glob as if they were outer operands.
	if hasCommandSubstitution(w.Text) {
		return []ir.Unknown{uncertainty(ir.UnknownCommandSubstitution, stage, w.Start, w.End, w.Text, "command substitution is not expanded")}
	}
	var out []ir.Unknown
	if hasPathGlob(w.Text) {
		out = append(out, uncertainty(ir.UnknownGlobRuntimeDependent, stage, w.Start, w.End, w.Text, "glob pattern is runtime-dependent "+strconv.Quote(w.Text)))
	}
	for _, name := range envRefs(w.Text) {
		if _, ok := env[name]; ok {
			continue
		}
		out = append(out, uncertainty(ir.UnknownEnvMissing, stage, w.Start, w.End, w.Text, "environment variable "+strconv.Quote(name)+" is not provided in context"))
	}
	return out
}

func uncertainty(code ir.UnknownCode, stage, start, end int, snippet, msg string) ir.Unknown {
	return ir.Unknown{
		Code:     code,
		Scope:    "stage:" + strconv.Itoa(stage),
		Message:  msg,
		Evidence: []ir.Evidence{CommandEvidence(start, end, snippet)},
		Blocking: false,
	}
}

func hasCommandSubstitution(text string) bool {
	return strings.Contains(text, "$(") || strings.Contains(text, "`")
}

// hasPathGlob requires path-like context so "echo [debug]" and "$?" are not
// mistaken for filesystem globs.
func hasPathGlob(text string) bool {
	if !hasGlobMeta(text) {
		return false
	}
	return strings.Contains(text, "/") || strings.Contains(text, "*") || strings.Contains(text, ".")
}

func hasGlobMeta(text string) bool {
	for i := range len(text) {
		switch text[i] {
		case '[':
			return true
		case '*', '?':
			// "$?" / "$*" / "$@" are special parameters, not globs.
			if i > 0 && text[i-1] == '$' {
				continue
			}
			return true
		}
	}
	return false
}

func envRefs(text string) []string {
	var names []string
	seen := map[string]bool{}
	for i := 0; i < len(text); {
		if text[i] != '$' {
			i++
			continue
		}
		if i+1 < len(text) && text[i+1] == '(' {
			i += 2
			continue
		}
		if i+1 < len(text) && text[i+1] == '{' {
			j := i + 2
			name, ok := readIdent(text, &j)
			if !ok || j >= len(text) || text[j] != '}' {
				i++
				continue
			}
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
			i = j + 1
			continue
		}
		j := i + 1
		name, ok := readIdent(text, &j)
		if !ok {
			i++
			continue
		}
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
		i = j
	}
	return names
}

// L0 env names are ASCII identifiers only (not unicode.IsLetter).
func readIdent(text string, i *int) (string, bool) {
	if *i >= len(text) {
		return "", false
	}
	c := text[*i]
	if c != '_' && !isASCIILetter(c) {
		return "", false
	}
	start := *i
	*i++
	for *i < len(text) {
		c = text[*i]
		if c != '_' && !isASCIILetter(c) && !isASCIIDigit(c) {
			break
		}
		*i++
	}
	return text[start:*i], true
}

func isASCIILetter(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func isASCIIDigit(c byte) bool {
	return c >= '0' && c <= '9'
}
