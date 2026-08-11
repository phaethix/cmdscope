package analyzer

import "strings"

// PathFlags carries soft outcomes of logical path cleaning that later stages
// turn into unknowns. Keeping bits here leaves this helper IR-free so effect
// extractors decide how each signal is surfaced.
type PathFlags uint8

const (
	// PathHasGlob means a segment still contains glob metacharacters.
	// The pattern must stay literal: expanding it would invent filesystem facts.
	PathHasGlob PathFlags = 1 << iota

	// PathUnprovenDotDot means a ".." had no concrete parent to collapse into
	// (root escape or a glob parent). Keep the ".." — clamping to "/" would
	// pretend the path stayed inside the logical workspace.
	PathUnprovenDotDot
)

// Has reports whether f includes every bit in bit.
func (f PathFlags) Has(bit PathFlags) bool { return f&bit == bit }

const logicalScheme = "logical://"

// NormalizeLogicalPath produces the logical effect target for raw under cwd.
// Only concrete parents prove a ".."; the real filesystem, symlinks, and glob
// expansion are never consulted so two hosts always agree on the same string.
func NormalizeLogicalPath(raw, cwd string) (string, PathFlags) {
	if raw == "" {
		return "", 0
	}

	raw = toSlash(raw)
	cwd = toSlash(cwd)

	if isAbs(raw) {
		return cleanPath(raw, true)
	}
	if rest, ok := strings.CutPrefix(cwd, logicalScheme); ok {
		// Strip the scheme before segment cleaning so the empty piece inside
		// "://" is not treated as a path root.
		return cleanLogical(raw, rest)
	}
	if cwd == "" {
		return cleanPath(raw, false)
	}
	joined := joinSlash(cwd, raw)
	return cleanPath(joined, isAbs(joined))
}

func cleanLogical(raw, rest string) (string, PathFlags) {
	joined := joinSlash(strings.Trim(rest, "/"), raw)
	target, flags := cleanPath(joined, false)
	return logicalScheme + target, flags
}

func joinSlash(base, rel string) string {
	base = strings.TrimRight(base, "/")
	rel = strings.TrimLeft(rel, "/")
	switch {
	case base == "":
		return rel
	case rel == "":
		return base
	default:
		return base + "/" + rel
	}
}

func toSlash(p string) string {
	if !strings.Contains(p, `\`) {
		return p
	}
	return strings.ReplaceAll(p, `\`, "/")
}

func isAbs(p string) bool { return strings.HasPrefix(p, "/") }

func isGlob(seg string) bool { return strings.ContainsAny(seg, "*?[") }

func isConcrete(seg string) bool { return seg != ".." && !isGlob(seg) }

// segmentStack owns the collapse rules so join/prefix orchestration stays
// free of stack bookkeeping (small stateful helper, not a package-level algorithm).
type segmentStack struct {
	segs  []string
	flags PathFlags
}

func (s *segmentStack) apply(seg string) {
	switch seg {
	case "", ".":
		return
	case "..":
		if n := len(s.segs); n > 0 && isConcrete(s.segs[n-1]) {
			s.segs = s.segs[:n-1]
			return
		}
		s.segs = append(s.segs, seg)
		s.flags |= PathUnprovenDotDot
	default:
		if isGlob(seg) {
			s.flags |= PathHasGlob
		}
		s.segs = append(s.segs, seg)
	}
}

func (s *segmentStack) join(abs bool) string {
	body := strings.Join(s.segs, "/")
	if abs {
		return "/" + body
	}
	return body
}

func cleanPath(joined string, abs bool) (string, PathFlags) {
	stack := segmentStack{
		segs: make([]string, 0, strings.Count(joined, "/")+1),
	}
	for seg := range strings.SplitSeq(joined, "/") {
		stack.apply(seg)
	}
	return stack.join(abs), stack.flags
}
