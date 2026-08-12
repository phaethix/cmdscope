package analyzer

import (
	"maps"
	"strings"

	"github.com/phaethix/runmark/internal/ir"
)

// ContextFiles is the only way expanders may read caller-supplied workspace
// files. Implementations must not touch the host filesystem.
type ContextFiles interface {
	Lookup(path string) (content string, ok bool)
}

type mapContextFiles struct {
	files map[string]string
}

// NewContextFiles builds a read-only view of AnalysisContext.Files. Size and
// key-shape limits are re-checked here so a raw map that skipped
// ValidateRequest cannot enter expansion. A nil context or nil Files map
// yields an empty store; CWD/Env/Platform are ignored (Files-only API).
func NewContextFiles(ctx *ir.AnalysisContext) (ContextFiles, error) {
	if ctx == nil || ctx.Files == nil {
		return &mapContextFiles{files: map[string]string{}}, nil
	}
	if err := validateContextFiles(ctx.Files); err != nil {
		return nil, err
	}
	files := make(map[string]string, len(ctx.Files))
	maps.Copy(files, ctx.Files)
	return &mapContextFiles{files: files}, nil
}

func validateContextFiles(files map[string]string) error {
	var totalBytes int
	for key, content := range files {
		if !isContextFileKey(key) {
			return ir.NewValidationError(ir.ErrCodeInvalidContextPath, "context file key must be a relative POSIX path without .. or ./ segments")
		}
		size := len(content)
		if size > ir.MaxContextFileBytes {
			return ir.NewValidationError(ir.ErrCodeContextFileTooLarge, "context file exceeds maximum size")
		}
		totalBytes += size
		if totalBytes > ir.MaxTotalContextBytes {
			return ir.NewValidationError(ir.ErrCodeContextFileTooLarge, "total context size exceeds maximum")
		}
	}
	return nil
}

// Lookup returns caller-supplied content for a relative POSIX path. Backslashes
// are normalized to '/'; absolute paths, ".." segments, and other invalid
// shapes are treated as misses rather than errors (the interface has no error
// channel) and never fall through to the OS.
func (m *mapContextFiles) Lookup(path string) (string, bool) {
	key, ok := normalizeLookupKey(path)
	if !ok {
		return "", false
	}
	content, found := m.files[key]
	return content, found
}

func normalizeLookupKey(path string) (string, bool) {
	if path == "" {
		return "", false
	}
	key := strings.ReplaceAll(path, `\`, "/")
	if !isContextFileKey(key) {
		return "", false
	}
	return key, true
}

// isContextFileKey mirrors ir.isValidContextFileKey so Lookup and construct
// reject the same key shapes.
func isContextFileKey(key string) bool {
	if key == "" {
		return false
	}
	if strings.Contains(key, "\\") {
		return false
	}
	if len(key) >= 2 && key[1] == ':' {
		c := key[0]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			return false
		}
	}
	if strings.HasPrefix(key, "/") {
		return false
	}
	if strings.Contains(key, "//") {
		return false
	}
	for part := range strings.SplitSeq(key, "/") {
		if part == "." || part == ".." {
			return false
		}
	}
	return true
}
