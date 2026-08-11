package analyzer

import "github.com/phaethix/cmdscope/internal/logicalpath"

// Path helpers live in logicalpath so effect extractors can normalize targets
// without importing analyzer (which would create a cycle once Analyze wires
// extractors).

type PathFlags = logicalpath.PathFlags

const (
	PathHasGlob        = logicalpath.PathHasGlob
	PathUnprovenDotDot = logicalpath.PathUnprovenDotDot
)

// NormalizeLogicalPath is a compatibility wrapper around logicalpath.NormalizeLogicalPath.
func NormalizeLogicalPath(raw, cwd string) (string, PathFlags) {
	return logicalpath.NormalizeLogicalPath(raw, cwd)
}
