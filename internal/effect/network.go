package effect

import (
	"strings"

	"github.com/phaethix/runmark/internal/ir"
	"github.com/phaethix/runmark/internal/shell"
)

// ExtractNetwork records curl/wget URL operands as network effects without
// fetching them. Targets stay raw URL text so path normalization cannot join
// schemes like https:// onto cwd.
func ExtractNetwork(cmd *shell.SimpleCommand, stage int, cond ir.Condition) ([]ir.Effect, []ir.Unknown) {
	if cmd == nil || len(cmd.Words) == 0 {
		return nil, nil
	}
	name := commandBasename(cmd.Words[0].Text)
	if name != "curl" && name != "wget" {
		return nil, nil
	}

	urls := networkURLOperands(cmd.Words[1:], name)
	if len(urls) == 0 {
		return nil, []ir.Unknown{insufficientOperandUnknown(cmd.Words[0], stage, name)}
	}

	effects := make([]ir.Effect, 0, len(urls))
	for _, w := range urls {
		ef := ir.Effect{
			Kind:       ir.EffectNetwork,
			RawTarget:  w.Text,
			Target:     w.Text,
			Stage:      stage,
			Certainty:  ir.Certain,
			Provenance: ir.FromCommand,
			Condition:  cond,
			Evidence:   []ir.Evidence{commandEvidence(w.Start, w.End, w.Text)},
		}
		ef.ID = ir.EffectID(ir.SchemaVersion, ef)
		effects = append(effects, ef)
	}
	return effects, nil
}

func networkURLOperands(words []shell.Word, cmdName string) []shell.Word {
	var out []shell.Word
	endOpts := false
	for i := 0; i < len(words); i++ {
		w := words[i]
		if !endOpts {
			if w.Text == "--" {
				endOpts = true
				continue
			}
			if strings.HasPrefix(w.Text, "-") && w.Text != "-" {
				if urls, ok := urlOptionValues(w, words, &i); ok {
					out = append(out, urls...)
					continue
				}
				if networkOptionTakesArg(cmdName, w.Text) && i+1 < len(words) && !strings.HasPrefix(words[i+1].Text, "-") {
					i++
				}
				continue
			}
		}
		if w.Text == "-" {
			continue
		}
		out = append(out, w)
	}
	return out
}

// urlOptionValues pulls URLs carried by --url / --url= so they are not dropped
// as opaque option arguments.
func urlOptionValues(opt shell.Word, words []shell.Word, i *int) ([]shell.Word, bool) {
	if !strings.HasPrefix(opt.Text, "--") {
		return nil, false
	}
	name, val, hasVal := strings.Cut(opt.Text, "=")
	if name != "--url" {
		return nil, false
	}
	if hasVal {
		if val == "" || val == "-" {
			return nil, true
		}
		return []shell.Word{{Text: val, Start: opt.Start, End: opt.End}}, true
	}
	if *i+1 < len(words) && !strings.HasPrefix(words[*i+1].Text, "-") {
		*i++
		u := words[*i]
		if u.Text == "-" {
			return nil, true
		}
		return []shell.Word{u}, true
	}
	return nil, true
}

func networkOptionTakesArg(cmdName, opt string) bool {
	if strings.HasPrefix(opt, "--") {
		name, _, ok := strings.Cut(opt, "=")
		if ok {
			return false
		}
		switch name {
		case "--output", "--header", "--data", "--user", "--user-agent",
			"--cookie", "--referer", "--max-time", "--connect-timeout",
			"--proxy", "--output-document", "--directory-prefix":
			return true
		}
		return false
	}
	if len(opt) != 2 || opt[0] != '-' {
		return false
	}
	flag := opt[1]
	switch cmdName {
	case "curl":
		switch flag {
		case 'o', 'e', 'H', 'd', 'u', 'A', 'b', 'c', 'w', 'T', 'Y', 'y', 'm', 'K', 'E':
			return true
		}
	case "wget":
		switch flag {
		case 'O', 'o', 'P', 'U', 't', 'T', 'w', 'Q':
			return true
		}
	}
	return false
}
