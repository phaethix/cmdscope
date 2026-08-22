package effect

import (
	"strings"

	"github.com/phaethix/runmark/internal/shell"
)

// Shell quotes retain their delimiters in the lexer's Word.Text, so
// "$OUT" stays as the four-byte string "$OUT", not OUT. Substitution must
// strip the outermost quote pair first, otherwise the substituted value
// carries phantom quotes into path normalization.
func stripQuotes(s string) string {
	if len(s) >= 2 {
		if s[0] == '"' && s[len(s)-1] == '"' {
			return s[1 : len(s)-1]
		}
		if s[0] == '\'' && s[len(s)-1] == '\'' {
			// Single-quoted text has no expansions; but
			// SubstituteEnvWords is only called on words that $-refs
			// already detected, so stripping is safe here.
			return s[1 : len(s)-1]
		}
	}
	return s
}

// SubstituteEnvWords returns a copy of cmd with env-provided values
// substituted into words and redirect targets, plus the original text of
// every changed word keyed by span start. Spans stay the original's so
// evidence keeps pointing at real source bytes. The original cmd is returned
// unchanged when nothing was substituted.
func SubstituteEnvWords(cmd *shell.SimpleCommand, env map[string]string) (*shell.SimpleCommand, map[int]string) {
	if cmd == nil || len(env) == 0 {
		return cmd, nil
	}
	changed := map[int]string{}
	sub := &shell.SimpleCommand{
		Assignments: cmd.Assignments,
		Start:       cmd.Start,
		End:         cmd.End,
	}
	sub.Words = substituteWordSlice(cmd.Words, changed, env)
	sub.Redirects = make([]shell.Redirect, len(cmd.Redirects))
	for i, r := range cmd.Redirects {
		sub.Redirects[i] = r
		sub.Redirects[i].Target = substituteOneWord(r.Target, changed, env)
	}
	if len(changed) == 0 {
		return cmd, nil
	}
	return sub, changed
}

func substituteWordSlice(words []shell.Word, changed map[int]string, env map[string]string) []shell.Word {
	out := make([]shell.Word, len(words))
	for i, w := range words {
		out[i] = substituteOneWord(w, changed, env)
	}
	return out
}

func substituteOneWord(w shell.Word, changed map[int]string, env map[string]string) shell.Word {
	unquoted := stripQuotes(w.Text)
	text := SubstituteEnvRefs(unquoted, env)
	if text == unquoted {
		return w
	}
	changed[w.Start] = w.Text
	return shell.Word{Text: text, Start: w.Start, End: w.End}
}

// SubstituteEnvRefs replaces $NAME and ${NAME} references with caller-
// provided env values. Unknown names are left intact so downstream
// env_missing handling still fires; this never guesses a value.
func SubstituteEnvRefs(text string, env map[string]string) string {
	if len(env) == 0 || !strings.Contains(text, "$") {
		return text
	}
	var b strings.Builder
	b.Grow(len(text))
	for i := 0; i < len(text); {
		c := text[i]
		if c != '$' {
			b.WriteByte(c)
			i++
			continue
		}
		if i+1 < len(text) && text[i+1] == '{' {
			j := i + 2
			name, ok := readEnvName(text, &j)
			if ok && j < len(text) && text[j] == '}' {
				if val, known := env[name]; known {
					b.WriteString(val)
					i = j + 1
					continue
				}
			}
			b.WriteString(text[i : i+2])
			i += 2
			continue
		}
		j := i + 1
		name, ok := readEnvName(text, &j)
		if ok {
			if val, known := env[name]; known {
				b.WriteString(val)
				i = j
				continue
			}
			b.WriteByte('$')
			b.WriteString(name)
			i = j
			continue
		}
		b.WriteByte('$')
		i++
	}
	return b.String()
}

func readEnvName(text string, i *int) (string, bool) {
	if *i >= len(text) {
		return "", false
	}
	c := text[*i]
	if c != '_' && !isEnvLetter(c) {
		return "", false
	}
	start := *i
	*i++
	for *i < len(text) {
		c := text[*i]
		if c != '_' && !isEnvLetter(c) && !isEnvDigit(c) {
			break
		}
		*i++
	}
	return text[start:*i], true
}

func isEnvLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }
func isEnvDigit(c byte) bool  { return c >= '0' && c <= '9' }
