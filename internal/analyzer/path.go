package analyzer

import "github.com/phaethix/cmdscope/internal/logicalpath"

// Path symbols are re-exported from logicalpath so existing analyzer callers
// keep compiling while extractors import the leaf package (avoids a cycle).

type PathFlags = logicalpath.PathFlags

const (
	PathHasGlob        = logicalpath.PathHasGlob
	PathUnprovenDotDot = logicalpath.PathUnprovenDotDot
)

func NormalizeLogicalPath(raw, cwd string) (string, PathFlags) {
	return logicalpath.NormalizeLogicalPath(raw, cwd)
}
