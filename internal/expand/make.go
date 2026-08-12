package expand

import (
	"maps"
	"slices"
	"strings"

	"github.com/phaethix/runmark/internal/ir"
	"github.com/phaethix/runmark/internal/shell"
)

const makefileName = "Makefile"

// ExpandMake expands `make` / `make <target>` using a caller-supplied Makefile.
// It never invokes make(1); recipes are only re-lexed by the internal shell parser.
func ExpandMake(cmd *shell.SimpleCommand, files map[string]string, stage int) ExpansionResult {
	if cmd == nil || len(cmd.Words) == 0 {
		return ExpansionResult{}
	}
	if commandBasename(cmd.Words[0].Text) != "make" {
		return ExpansionResult{}
	}

	target, dynamic, missing := makeTargetName(cmd.Words)
	if dynamic {
		return ExpansionResult{
			Applied: true,
			Unknowns: []ir.Unknown{expandUnknown(
				ir.UnknownScriptDynamicPath, stage, cmd.Words[0],
				"make target name is dynamic and cannot be resolved statically",
			)},
		}
	}

	raw, ok := files[makefileName]
	if !ok {
		return ExpansionResult{
			Applied: true,
			Unknowns: []ir.Unknown{expandUnknown(
				ir.UnknownContextMissing, stage, cmd.Words[0],
				"Makefile was not provided in analysis context",
			)},
		}
	}

	mf, dynFeat := parseMakefile(raw)
	if dynFeat {
		return ExpansionResult{
			Applied: true,
			Unknowns: []ir.Unknown{expandUnknown(
				ir.UnknownUnsupportedCommand, stage, cmd.Words[0],
				"Makefile uses include/eval/$(shell) which cannot be expanded statically",
			)},
			Evidence: []ir.Evidence{{
				Source: ir.EvidenceWorkspaceFile,
				Path:   makefileName,
				Field:  "Makefile",
			}},
		}
	}

	if missing {
		// `make` with no target → first defined target, if any.
		if len(mf.order) == 0 {
			return ExpansionResult{
				Applied: true,
				Unknowns: []ir.Unknown{expandUnknown(
					ir.UnknownUnsupportedCommand, stage, cmd.Words[0],
					"make is missing a target name",
				)},
			}
		}
		target = mf.order[0]
	}

	if _, ok := mf.targets[target]; !ok {
		return ExpansionResult{
			Applied: true,
			Unknowns: []ir.Unknown{expandUnknown(
				ir.UnknownUnsupportedCommand, stage, cmd.Words[0],
				"Makefile has no target "+target,
			)},
		}
	}

	st := &makeState{mf: mf, stage: stage}
	st.expandTarget(target, nil)
	st.result.Applied = true
	return st.result
}

type makeTarget struct {
	deps    []string
	recipes []string
}

type makefile struct {
	vars    map[string]string
	targets map[string]*makeTarget
	order   []string
}

type makeState struct {
	mf     *makefile
	stage  int
	nodes  int
	result ExpansionResult
}

func (st *makeState) expandTarget(name string, active []string) {
	key := makeTargetKey(makefileName, name)
	if i := indexOf(active, key); i >= 0 {
		cycle := append(append([]string{}, active[i:]...), key)
		st.result.Unknowns = append(st.result.Unknowns, ir.Unknown{
			Code:     ir.UnknownExpansionCycle,
			Scope:    stageScope(st.stage),
			Message:  makeCycleMessage(cycle),
			Evidence: []ir.Evidence{makeEvidence(name)},
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

	t, ok := st.mf.targets[name]
	if !ok {
		st.result.Unknowns = append(st.result.Unknowns, ir.Unknown{
			Code:     ir.UnknownUnsupportedCommand,
			Scope:    stageScope(st.stage),
			Message:  "Makefile has no target " + name,
			Evidence: []ir.Evidence{makeEvidence(name)},
			Blocking: true,
		})
		return
	}

	st.nodes++
	st.result.Evidence = append(st.result.Evidence, makeEvidence(name))
	active = append(active, key)

	for _, dep := range t.deps {
		st.expandTarget(dep, active)
	}
	for _, line := range t.recipes {
		expanded := substituteMakeVars(line, st.mf.vars)
		if recipeHasDynamic(expanded) {
			st.result.Unknowns = append(st.result.Unknowns, ir.Unknown{
				Code:     ir.UnknownUnsupportedCommand,
				Scope:    stageScope(st.stage),
				Message:  "Makefile recipe uses dynamic make features",
				Evidence: []ir.Evidence{makeEvidence(name)},
				Blocking: true,
			})
			continue
		}
		toks, err := shell.Lex(expanded)
		if err != nil {
			st.result.Unknowns = append(st.result.Unknowns, ir.Unknown{
				Code:     ir.UnknownParseError,
				Scope:    stageScope(st.stage),
				Message:  "make recipe could not be lexed",
				Evidence: []ir.Evidence{makeEvidence(name)},
				Blocking: true,
			})
			continue
		}
		root, err := shell.Parse(toks)
		if err != nil && root == nil {
			st.result.Unknowns = append(st.result.Unknowns, ir.Unknown{
				Code:     ir.UnknownParseError,
				Scope:    stageScope(st.stage),
				Message:  "make recipe could not be parsed",
				Evidence: []ir.Evidence{makeEvidence(name)},
				Blocking: true,
			})
			continue
		}
		st.result.Nodes = append(st.result.Nodes, flattenCommands(root)...)
	}
}

func (st *makeState) hitLimit(code string) {
	if !slices.Contains(st.result.Limits, code) {
		st.result.Limits = append(st.result.Limits, code)
	}
	st.result.Unknowns = append(st.result.Unknowns, ir.Unknown{
		Code:     ir.UnknownExpansionLimit,
		Scope:    stageScope(st.stage),
		Message:  "make expansion exceeded " + code,
		Evidence: []ir.Evidence{},
		Blocking: true,
	})
}

func makeTargetName(words []shell.Word) (name string, dynamic, missing bool) {
	for _, w := range words[1:] {
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

func parseMakefile(raw string) (*makefile, bool) {
	mf := &makefile{
		vars:    map[string]string{},
		targets: map[string]*makeTarget{},
	}
	var current *makeTarget
	dynamic := false
	for line := range strings.SplitSeq(raw, "\n") {
		if strings.HasPrefix(line, "\t") {
			if current != nil {
				if recipe, ok := strings.CutPrefix(line, "\t"); ok {
					current.recipes = append(current.recipes, recipe)
				}
			}
			continue
		}
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		if isDynamicMakeDirective(trim) {
			dynamic = true
			continue
		}
		if name, val, ok := parseMakeAssign(trim); ok {
			mf.vars[name] = val
			current = nil
			continue
		}
		if name, deps, ok := parseMakeRule(trim); ok {
			t := &makeTarget{deps: deps}
			mf.targets[name] = t
			mf.order = append(mf.order, name)
			current = t
			continue
		}
		current = nil
	}
	return mf, dynamic
}

func parseMakeAssign(line string) (name, val string, ok bool) {
	// Only literal recursive/simple assignment. ?= / += / != are not statically
	// knowable (or are shell assigns) and must not populate the var table.
	for _, sep := range []string{":=", "="} {
		if before, after, found := strings.Cut(line, sep); found {
			name = strings.TrimSpace(before)
			if name == "" || strings.ContainsAny(name, " \t:") {
				return "", "", false
			}
			return name, strings.TrimSpace(after), true
		}
	}
	return "", "", false
}

func parseMakeRule(line string) (name string, deps []string, ok bool) {
	name, rest, found := strings.Cut(line, ":")
	if !found {
		return "", nil, false
	}
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsAny(name, " \t") {
		return "", nil, false
	}
	for dep := range strings.FieldsSeq(rest) {
		deps = append(deps, dep)
	}
	return name, deps, true
}

func substituteMakeVars(line string, vars map[string]string) string {
	if len(vars) == 0 {
		return line
	}
	// Map iteration order is randomized; sort names and re-apply until stable
	// so nested refs like A=$(B) resolve the same way on every run.
	names := slices.Sorted(maps.Keys(vars))
	out := line
	for range len(vars) + 1 {
		prev := out
		for _, name := range names {
			val := vars[name]
			out = strings.ReplaceAll(out, "$("+name+")", val)
			out = strings.ReplaceAll(out, "${"+name+"}", val)
		}
		if out == prev {
			break
		}
	}
	return out
}

func recipeHasDynamic(line string) bool {
	return strings.Contains(line, "$(shell") || strings.Contains(line, "${shell") ||
		strings.Contains(line, "$(eval") || strings.Contains(line, "${eval") ||
		strings.ContainsAny(line, "$")
}

func isDynamicMakeDirective(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))
	switch {
	case strings.HasPrefix(lower, "include "),
		strings.HasPrefix(lower, "-include "),
		strings.HasPrefix(lower, "sinclude "),
		strings.HasPrefix(lower, "eval "),
		strings.HasPrefix(lower, "export "),
		strings.HasPrefix(lower, "unexport "):
		return true
	}
	if strings.Contains(line, "$(shell") || strings.Contains(line, "${shell") ||
		strings.Contains(line, "$(eval") || strings.Contains(line, "${eval") {
		return true
	}
	// Shell assignment: NAME != cmd
	if before, _, found := strings.Cut(line, "!="); found {
		name := strings.TrimSpace(before)
		if name != "" && !strings.ContainsAny(name, " \t:") {
			return true
		}
	}
	return false
}

func makeTargetKey(file, name string) string {
	return "make:" + file + ":target:" + name
}

func makeCycleMessage(keys []string) string {
	names := make([]string, len(keys))
	for i, k := range keys {
		if _, name, ok := strings.Cut(k, ":target:"); ok {
			names[i] = name
		} else {
			names[i] = k
		}
	}
	return "expansion cycle " + strings.Join(names, " -> ")
}

func makeEvidence(target string) ir.Evidence {
	return ir.Evidence{
		Source: ir.EvidenceWorkspaceFile,
		Path:   makefileName,
		Field:  target,
	}
}
