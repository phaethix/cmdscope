package effect

import (
	"regexp"
	"strings"

	"github.com/phaethix/runmark/internal/shell"
)

// StripWrapperPrefix returns the command hidden behind one sudo/doas/env
// wrapper layer: wrapper flags and NAME=value assignments are dropped while
// words keep their original evidence spans. Redirects are deliberately not
// carried over — the wrapper command itself already extracted them — and the
// boolean is false when the command does not start with such a wrapper, so
// callers can chain-strip nested wrappers (env sudo rm ...) until false.
func StripWrapperPrefix(cmd *shell.SimpleCommand) (*shell.SimpleCommand, bool) {
	if cmd == nil || len(cmd.Words) == 0 {
		return nil, false
	}
	var inner []shell.Word
	switch name := commandBasename(cmd.Words[0].Text); name {
	case "sudo", "doas":
		inner = skipDashOptions(cmd.Words[1:], wrapperOptionTakesArg(name))
	case "env":
		inner = skipEnvAssignments(skipDashOptions(cmd.Words[1:], envOptionTakesArg))
	default:
		return nil, false
	}
	if len(inner) == 0 {
		return nil, false
	}
	return &shell.SimpleCommand{
		Words: inner,
		Start: inner[0].Start,
		End:   inner[len(inner)-1].End,
	}, true
}

// envAssignment matches the leading NAME= form env itself interprets; a word
// like rm=x is an assignment to env, not the utility, and must not become a
// command name here either.
var envAssignment = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

func skipEnvAssignments(words []shell.Word) []shell.Word {
	out := words
	for len(out) > 0 && envAssignment.MatchString(out[0].Text) {
		out = out[1:]
	}
	return out
}

// wrapperOptionTakesArg keeps the flag-arity tables close to the wrappers
// they decode; a wrong arity either swallows a real operand or leaks an
// option value as one, so only well-known flags are listed.
func wrapperOptionTakesArg(cmdName string) func(string) bool {
	if cmdName == "doas" {
		return doasOptionTakesArg
	}
	return sudoOptionTakesArg
}

func sudoOptionTakesArg(opt string) bool {
	if strings.HasPrefix(opt, "--") {
		if strings.Contains(opt, "=") {
			return false
		}
		switch opt {
		case "--user", "--group", "--prompt", "--close-from", "--role",
			"--type", "--timeout", "--other-user":
			return true
		}
		return false
	}
	if len(opt) != 2 || opt[0] != '-' {
		return false
	}
	switch opt[1] {
	case 'u', 'g', 'p', 'C', 'r', 't', 'T':
		return true
	}
	return false
}

func doasOptionTakesArg(opt string) bool {
	if strings.HasPrefix(opt, "--") || len(opt) != 2 || opt[0] != '-' {
		return false
	}
	switch opt[1] {
	case 'a', 'C', 'u':
		return true
	}
	return false
}

func envOptionTakesArg(opt string) bool {
	if strings.HasPrefix(opt, "--") {
		if strings.Contains(opt, "=") {
			return false
		}
		switch opt {
		case "--unset", "--split-string", "--block-signal",
			"--default-signal", "--ignore-signal":
			return true
		}
		return false
	}
	if len(opt) != 2 || opt[0] != '-' {
		return false
	}
	switch opt[1] {
	case 'u', 'S':
		return true
	}
	return false
}
