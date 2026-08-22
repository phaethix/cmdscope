package expand

import (
	"encoding/json"
	"slices"
	"strconv"
	"strings"

	"github.com/phaethix/runmark/internal/ir"
	"github.com/phaethix/runmark/internal/shell"
)

const (
	packageJSONFile   = "package.json"
	maxScriptDepth    = 8
	maxExpansionNodes = 64
)

// ExpansionResult is the bounded outcome of a matched expander. Applied=false
// means the command was not this expander's surface; Applied=true with
// Unknowns means the expander owned the command but could not fully expand it.
type ExpansionResult struct {
	Nodes    []shell.Node
	Evidence []ir.Evidence
	Unknowns []ir.Unknown
	Limits   []string
	Applied  bool
}

// ExpandNPM expands `npm run <script>` using caller-supplied package.json only.
func ExpandNPM(cmd *shell.SimpleCommand, files map[string]string, stage int) ExpansionResult {
	return expandPackageRun("npm", cmd, files, stage)
}

// ExpandPNPM expands `pnpm run <script>` with the same package.json rules as npm.
func ExpandPNPM(cmd *shell.SimpleCommand, files map[string]string, stage int) ExpansionResult {
	return expandPackageRun("pnpm", cmd, files, stage)
}

func expandPackageRun(tool string, cmd *shell.SimpleCommand, files map[string]string, stage int) ExpansionResult {
	if cmd == nil || len(cmd.Words) < 2 {
		return ExpansionResult{}
	}
	if commandBasename(cmd.Words[0].Text) != tool {
		return ExpansionResult{}
	}

	sub := cmd.Words[1].Text
	switch sub {
	case "run":
		name, dynamic, missing := runScriptName(cmd.Words)
		if dynamic {
			return ExpansionResult{
				Applied: true,
				Unknowns: []ir.Unknown{expandUnknown(
					ir.UnknownScriptDynamicPath, stage, cmd.Words[0],
					"npm/pnpm script name is dynamic and cannot be resolved statically",
				)},
			}
		}
		if missing {
			return ExpansionResult{
				Applied: true,
				Unknowns: []ir.Unknown{expandUnknown(
					ir.UnknownUnsupportedCommand, stage, cmd.Words[0],
					tool+` run is missing a script name`,
				)},
			}
		}
		return expandScriptByName(tool, name, cmd, files, stage)
	default:
		if !isImplicitScript(tool, sub) {
			return ExpansionResult{}
		}
		return expandImplicitScript(tool, sub, cmd, files, stage)
	}
}

// isImplicitScript reports whether a bare (non-`run`) subcommand should be
// treated as an implicit npm/pnpm script run. pnpm treats any non-reserved
// bare subcommand as a possible script; npm only reserves the lifecycle
// scripts test/start/stop/restart.
func isImplicitScript(tool, sub string) bool {
	switch tool {
	case "npm":
		switch sub {
		case "test", "start", "stop", "restart":
			return true
		}
		return false
	case "pnpm":
		switch sub {
		case "add", "install", "i", "remove", "rm", "uninstall", "update", "up",
			"link", "unlink", "exec", "dlx", "create", "init", "rebuild",
			"publish", "pack", "audit", "outdated", "why", "list", "ls",
			"store", "config", "env":
			return false
		}
		return true
	default:
		return false
	}
}

// expandImplicitScript expands a bare subcommand only when it names a real
// package.json script. When the script is absent we return an empty (unapplied)
// result rather than fabricating npm/pnpm built-in behavior.
func expandImplicitScript(tool, name string, cmd *shell.SimpleCommand, files map[string]string, stage int) ExpansionResult {
	raw, ok := files[packageJSONFile]
	if !ok {
		return ExpansionResult{
			Applied: true,
			Unknowns: []ir.Unknown{expandUnknown(
				ir.UnknownContextMissing, stage, cmd.Words[0],
				"package.json was not provided in analysis context",
			)},
		}
	}
	scripts, err := parsePackageScripts(raw)
	if err != nil {
		return ExpansionResult{
			Applied: true,
			Unknowns: []ir.Unknown{expandUnknown(
				ir.UnknownContextMissing, stage, cmd.Words[0],
				"package.json scripts could not be parsed",
			)},
		}
	}
	if _, ok := scripts[name]; !ok {
		return ExpansionResult{}
	}
	return expandScriptByName(tool, name, cmd, files, stage)
}

func expandScriptByName(tool, name string, cmd *shell.SimpleCommand, files map[string]string, stage int) ExpansionResult {
	raw, ok := files[packageJSONFile]
	if !ok {
		return ExpansionResult{
			Applied: true,
			Unknowns: []ir.Unknown{expandUnknown(
				ir.UnknownContextMissing, stage, cmd.Words[0],
				"package.json was not provided in analysis context",
			)},
		}
	}

	scripts, err := parsePackageScripts(raw)
	if err != nil {
		return ExpansionResult{
			Applied: true,
			Unknowns: []ir.Unknown{expandUnknown(
				ir.UnknownContextMissing, stage, cmd.Words[0],
				"package.json scripts could not be parsed",
			)},
		}
	}

	st := &expandState{
		tool:    tool,
		scripts: scripts,
		stage:   stage,
	}
	st.expandScript(name, nil)
	st.result.Applied = true
	return st.result
}

type expandState struct {
	tool    string
	scripts map[string]string
	stage   int
	nodes   int
	result  ExpansionResult
}

func (st *expandState) expandScript(name string, active []string) {
	key := scriptKey(packageJSONFile, name)
	if i := indexOf(active, key); i >= 0 {
		cycle := append(append([]string{}, active[i:]...), key)
		st.result.Unknowns = append(st.result.Unknowns, ir.Unknown{
			Code:     ir.UnknownExpansionCycle,
			Scope:    stageScope(st.stage),
			Message:  cycleMessage(cycle),
			Evidence: []ir.Evidence{scriptEvidence(name, st.scripts[name])},
			Blocking: true,
		})
		return
	}
	if len(active) >= maxScriptDepth {
		st.hitLimit("max_script_depth:8")
		return
	}
	if st.nodes >= maxExpansionNodes {
		st.hitLimit("max_expansion_nodes:64")
		return
	}

	body, ok := st.scripts[name]
	if !ok {
		st.result.Unknowns = append(st.result.Unknowns, ir.Unknown{
			Code:     ir.UnknownUnsupportedCommand,
			Scope:    stageScope(st.stage),
			Message:  "package.json has no scripts." + name,
			Evidence: []ir.Evidence{scriptEvidence(name, "")},
			Blocking: true,
		})
		return
	}

	st.nodes++
	st.result.Evidence = append(st.result.Evidence, scriptEvidence(name, body))

	toks, err := shell.Lex(body)
	if err != nil {
		st.result.Unknowns = append(st.result.Unknowns, ir.Unknown{
			Code:     ir.UnknownParseError,
			Scope:    stageScope(st.stage),
			Message:  "script " + name + " could not be lexed",
			Evidence: []ir.Evidence{scriptEvidence(name, body)},
			Blocking: true,
		})
		return
	}
	root, err := shell.Parse(toks)
	if err != nil && root == nil {
		st.result.Unknowns = append(st.result.Unknowns, ir.Unknown{
			Code:     ir.UnknownParseError,
			Scope:    stageScope(st.stage),
			Message:  "script " + name + " could not be parsed",
			Evidence: []ir.Evidence{scriptEvidence(name, body)},
			Blocking: true,
		})
		return
	}

	active = append(active, key)
	for _, n := range flattenCommands(root) {
		if nested, ok := n.(*shell.SimpleCommand); ok {
			if _, script, matched := matchPackageRun(nested); matched {
				if script == "" {
					_, dynamic, missing := runScriptName(nested.Words)
					switch {
					case dynamic:
						st.result.Unknowns = append(st.result.Unknowns, expandUnknown(
							ir.UnknownScriptDynamicPath, st.stage, nested.Words[0],
							"npm/pnpm script name is dynamic and cannot be resolved statically",
						))
					case missing:
						st.result.Unknowns = append(st.result.Unknowns, expandUnknown(
							ir.UnknownUnsupportedCommand, st.stage, nested.Words[0],
							"npm/pnpm run is missing a script name",
						))
					}
					continue
				}
				st.expandScript(script, active)
				continue
			}
		}
		st.result.Nodes = append(st.result.Nodes, n)
	}
}

func (st *expandState) hitLimit(code string) {
	if !slices.Contains(st.result.Limits, code) {
		st.result.Limits = append(st.result.Limits, code)
	}
	st.result.Unknowns = append(st.result.Unknowns, ir.Unknown{
		Code:     ir.UnknownExpansionLimit,
		Scope:    stageScope(st.stage),
		Message:  "script expansion exceeded " + code,
		Evidence: []ir.Evidence{},
		Blocking: true,
	})
}

func matchPackageRun(cmd *shell.SimpleCommand) (tool, script string, ok bool) {
	if cmd == nil || len(cmd.Words) < 2 {
		return "", "", false
	}
	base := commandBasename(cmd.Words[0].Text)
	if (base != "npm" && base != "pnpm") || cmd.Words[1].Text != "run" {
		return "", "", false
	}
	name, dynamic, missing := runScriptName(cmd.Words)
	if dynamic || missing {
		return base, "", true
	}
	return base, name, true
}

func runScriptName(words []shell.Word) (name string, dynamic, missing bool) {
	for _, w := range words[2:] {
		if strings.HasPrefix(w.Text, "-") && w.Text != "-" {
			continue
		}
		if strings.ContainsAny(w.Text, "$`") {
			return "", true, false
		}
		return w.Text, false, false
	}
	return "", false, true
}

func parsePackageScripts(raw string) (map[string]string, error) {
	var doc struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return nil, err
	}
	if doc.Scripts == nil {
		doc.Scripts = map[string]string{}
	}
	return doc.Scripts, nil
}

func flattenCommands(n shell.Node) []shell.Node {
	switch v := n.(type) {
	case nil:
		return nil
	case *shell.SimpleCommand:
		return []shell.Node{v}
	case *shell.Sequence:
		var out []shell.Node
		for _, item := range v.Items {
			out = append(out, flattenCommands(item)...)
		}
		return out
	case *shell.Pipeline:
		return append([]shell.Node{}, v.Commands...)
	case *shell.Binary:
		return append(flattenCommands(v.Left), flattenCommands(v.Right)...)
	case *shell.Subshell:
		return flattenCommands(v.Body)
	default:
		return []shell.Node{n}
	}
}

func scriptKey(pkg, name string) string {
	return "npm:" + pkg + ":script:" + name
}

func cycleMessage(keys []string) string {
	names := make([]string, len(keys))
	for i, k := range keys {
		if _, name, ok := strings.Cut(k, ":script:"); ok {
			names[i] = name
		} else {
			names[i] = k
		}
	}
	return "expansion cycle " + strings.Join(names, " -> ")
}

func scriptEvidence(name, snippet string) ir.Evidence {
	return ir.Evidence{
		Source:  ir.EvidenceWorkspaceFile,
		Path:    packageJSONFile,
		Field:   "scripts." + name,
		Snippet: snippet,
	}
}

func expandUnknown(code ir.UnknownCode, stage int, argv0 shell.Word, msg string) ir.Unknown {
	return ir.Unknown{
		Code:    code,
		Scope:   stageScope(stage),
		Message: msg,
		Evidence: []ir.Evidence{{
			Source:    ir.EvidenceCommand,
			StartByte: new(argv0.Start),
			EndByte:   new(argv0.End),
			Snippet:   argv0.Text,
		}},
		Blocking: true,
	}
}

func stageScope(stage int) string {
	return "stage:" + strconv.Itoa(stage)
}

func commandBasename(argv0 string) string {
	argv0 = strings.ReplaceAll(argv0, `\`, "/")
	if i := strings.LastIndex(argv0, "/"); i >= 0 {
		return argv0[i+1:]
	}
	return argv0
}

func indexOf(path []string, key string) int {
	return slices.Index(path, key)
}
