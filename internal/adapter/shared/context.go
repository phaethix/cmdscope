// Package shared holds hook-adapter machinery: the shared PreToolUse+Bash
// pipeline and the bounded workspace-context injection both client adapters
// use. None of this runs inside core analysis — core still receives an
// explicit AnalysisContext; only this adapter layer may read files.
package shared

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/phaethix/runmark/internal/ir"
)

// hookContextEnv, when set to "0", disables automatic context injection so a
// caller can rely solely on the files it already passed in.
const hookContextEnv = "RUNMARK_HOOK_CONTEXT"

// contextFiles are the only files an adapter may read from the hook cwd to
// populate AnalysisContext.Files. The set is deliberately tiny: only files
// whose absence would otherwise force every npm/make run into context_missing.
var contextFiles = []string{"package.json", "Makefile"}

// InjectContext fills an AnalysisContext with bounded workspace files read from
// the real cwd plus the host platform. It reads nothing for logical:// cwds
// (there is no filesystem there) and honors RUNMARK_HOOK_CONTEXT=0. Files the
// caller already supplied are left untouched.
func InjectContext(ac *ir.AnalysisContext) {
	if ac == nil {
		return
	}
	if os.Getenv(hookContextEnv) == "0" {
		return
	}
	if ac.Platform == "" {
		ac.Platform = runtime.GOOS
	}
	if ac.CWD == "" || strings.HasPrefix(ac.CWD, "logical://") {
		return
	}
	if ac.Files == nil {
		ac.Files = map[string]string{}
	}
	for _, name := range contextFiles {
		if _, exists := ac.Files[name]; exists {
			continue
		}
		data, err := os.ReadFile(filepath.Join(ac.CWD, name))
		if err != nil {
			continue
		}
		if len(data) > ir.MaxContextFileBytes {
			continue
		}
		ac.Files[name] = string(data)
	}
}
