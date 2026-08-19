package facts

import (
	"path"
	"slices"
	"strings"

	"github.com/phaethix/runmark/internal/ir"
)

// Project maps an ImpactReport into experimental RunmarkFacts. It is a pure
// function over the report: it never reparses the command, never expands
// scripts, and never reads the host filesystem.
func Project(report ir.ImpactReport) RunmarkFacts {
	out := Empty()

	read := map[string]struct{}{}
	write := map[string]struct{}{}
	del := map[string]struct{}{}
	scripts := map[string]ScriptEntry{}
	evidence := map[string]FactEvidence{}
	reasons := map[string]struct{}{}

	addEvidence := func(list []ir.Evidence) {
		for _, ev := range list {
			fe := FactEvidence{
				Source:  string(ev.Source),
				Path:    ev.Path,
				Field:   ev.Field,
				Snippet: ev.Snippet,
			}
			evidence[evidenceKey(fe)] = fe
			if se, ok := scriptFromEvidence(ev); ok {
				scripts[scriptKey(se)] = se
			}
		}
	}

	for _, st := range report.Stages {
		for _, ef := range st.Effects {
			switch ef.Kind {
			case ir.EffectRead:
				if ef.Target != "" {
					read[ef.Target] = struct{}{}
				}
				addEvidence(ef.Evidence)
			case ir.EffectWrite:
				if ef.Target != "" {
					write[ef.Target] = struct{}{}
				}
				addEvidence(ef.Evidence)
			case ir.EffectDelete:
				if ef.Target != "" {
					del[ef.Target] = struct{}{}
				}
				out.Boundary.Destructive = true
				addEvidence(ef.Evidence)
			case ir.EffectNetwork, ir.EffectExecuteRemote:
				out.Boundary.ExternalNetwork = true
				addEvidence(ef.Evidence)
			default:
				// process/privilege/install stay out of public touches; still
				// harvest workspace script evidence if present.
				addEvidence(ef.Evidence)
			}
		}
	}

	for _, f := range report.Flags {
		switch f {
		case ir.FlagDestructive:
			out.Boundary.Destructive = true
		case ir.FlagExternalNetwork, ir.FlagRemoteContent:
			out.Boundary.ExternalNetwork = true
		case ir.FlagOpaqueScript:
			out.Boundary.OpaqueScript = true
		}
	}

	for _, u := range report.Unknowns {
		// Non-blocking unknowns (glob, command substitution, env-missing) still
		// mark the facts as partially undetermined; only blocking ones say the
		// command entered a script we could not reason about at all.
		out.Unknown = true
		reasons[string(u.Code)] = struct{}{}
		if u.Blocking && opaqueUnknown(u.Code) {
			out.Boundary.OpaqueScript = true
		}
		addEvidence(u.Evidence)
	}

	out.Touches.Read = keys(read)
	out.Touches.Write = keys(write)
	out.Touches.Delete = keys(del)
	out.UnknownReasons = keys(reasons)
	out.Scripts = scriptValues(scripts)
	out.Evidence = evidenceValues(evidence)

	for _, p := range out.Touches.Read {
		if outsideWorkspace(report.CWD, p) {
			out.Boundary.OutsideWorkspace = true
		}
		if isSensitivePath(p) {
			out.Boundary.SensitivePath = true
		}
	}
	for _, p := range out.Touches.Write {
		if outsideWorkspace(report.CWD, p) {
			out.Boundary.OutsideWorkspace = true
		}
		if isSensitivePath(p) {
			out.Boundary.SensitivePath = true
		}
	}
	for _, p := range out.Touches.Delete {
		if outsideWorkspace(report.CWD, p) {
			out.Boundary.OutsideWorkspace = true
		}
		if isSensitivePath(p) {
			out.Boundary.SensitivePath = true
		}
	}

	Normalize(&out)
	return out
}

func keys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func evidenceKey(ev FactEvidence) string {
	return ev.Source + "\x00" + ev.Path + "\x00" + ev.Field + "\x00" + ev.Snippet
}

func scriptKey(se ScriptEntry) string {
	return se.Tool + "\x00" + se.Name + "\x00" + se.Source
}

func scriptValues(m map[string]ScriptEntry) []ScriptEntry {
	out := make([]ScriptEntry, 0, len(m))
	for _, se := range m {
		out = append(out, se)
	}
	return out
}

func evidenceValues(m map[string]FactEvidence) []FactEvidence {
	out := make([]FactEvidence, 0, len(m))
	for _, ev := range m {
		out = append(out, ev)
	}
	return out
}

func scriptFromEvidence(ev ir.Evidence) (ScriptEntry, bool) {
	if ev.Source != ir.EvidenceWorkspaceFile {
		return ScriptEntry{}, false
	}
	switch ev.Path {
	case "package.json":
		name, ok := strings.CutPrefix(ev.Field, "scripts.")
		if !ok {
			name = ev.Field
		}
		return ScriptEntry{Tool: "npm", Name: name, Source: "package.json"}, true
	case "Makefile":
		return ScriptEntry{Tool: "make", Name: ev.Field, Source: "Makefile"}, true
	default:
		return ScriptEntry{}, false
	}
}

func opaqueUnknown(code ir.UnknownCode) bool {
	switch code {
	case ir.UnknownRemoteContent,
		ir.UnknownInterpreterDynamicCode,
		ir.UnknownScriptDynamicPath,
		ir.UnknownContextMissing,
		ir.UnknownScriptNotProvided,
		ir.UnknownExpansionLimit,
		ir.UnknownExpansionCycle:
		return true
	default:
		return false
	}
}

// outsideWorkspace is a logical/static judgment, not OS containment.
func outsideWorkspace(cwd, target string) bool {
	if target == "" {
		return false
	}
	target = strings.ReplaceAll(target, `\`, "/")
	cwd = strings.ReplaceAll(cwd, `\`, "/")

	for part := range strings.SplitSeq(target, "/") {
		if part == ".." {
			return true
		}
	}

	if rest, ok := strings.CutPrefix(cwd, "logical://"); ok {
		root := "logical://" + rest
		if target == root || strings.HasPrefix(target, root+"/") {
			return false
		}
		return true
	}

	if cwd == "" {
		return strings.HasPrefix(target, "/")
	}

	if strings.HasPrefix(cwd, "/") && strings.HasPrefix(target, "/") {
		if target == cwd || strings.HasPrefix(target, strings.TrimRight(cwd, "/")+"/") {
			return false
		}
		return true
	}
	return false
}

func isSensitivePath(target string) bool {
	target = strings.ReplaceAll(target, `\`, "/")
	base := path.Base(target)
	lower := strings.ToLower(target)
	baseLower := strings.ToLower(base)

	if slices.Contains(sensitiveBasenames, baseLower) {
		return true
	}
	for _, suf := range sensitiveSuffixes {
		if strings.HasSuffix(lower, suf) {
			return true
		}
	}
	for _, seg := range sensitiveSegments {
		if strings.Contains(lower, seg) {
			return true
		}
	}
	return false
}
