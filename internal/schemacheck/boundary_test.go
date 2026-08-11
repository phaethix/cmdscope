package schemacheck_test

// These boundary checks deliberately live in the schemacheck black-box test
// package (package schemacheck_test) rather than internal/integration:
// enforcing package layout and import direction is part of schemacheck's
// future contract-checking responsibility, and internal/integration is not
// created yet. The os/exec calls below only run the go toolchain
// (go env / go list) to inspect module metadata; they never execute the
// command under analysis.

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const modulePath = "github.com/phaethix/cmdscope"

var requiredPackages = []string{
	modulePath + "/internal/app",
	modulePath + "/internal/ir",
	modulePath + "/internal/analyzer",
	modulePath + "/internal/shell",
	modulePath + "/internal/expand",
	modulePath + "/internal/effect",
	modulePath + "/internal/render",
	modulePath + "/internal/adapter/codex",
	modulePath + "/internal/schemacheck",
}

var forbiddenPaths = []string{
	"pkg",
	"internal/model",
	"internal/lexer",
	"internal/shellparse",
	"internal/effects",
	"internal/analyze",
}

// allowedInternalImports lists permitted cmdscope internal imports per package.
// Dependency direction: cmd -> app -> analyzer -> {shell,expand,effect,ir};
// render -> ir; adapter/codex -> {app,ir}; schemacheck -> ir.
var allowedInternalImports = map[string][]string{
	modulePath + "/cmd/cmdscope":           {modulePath + "/internal/app"},
	modulePath + "/internal/app":           {modulePath + "/internal/analyzer", modulePath + "/internal/ir"},
	modulePath + "/internal/analyzer":      {modulePath + "/internal/shell", modulePath + "/internal/expand", modulePath + "/internal/effect", modulePath + "/internal/ir"},
	modulePath + "/internal/render":        {modulePath + "/internal/ir"},
	modulePath + "/internal/adapter/codex": {modulePath + "/internal/app", modulePath + "/internal/ir"},
	modulePath + "/internal/schemacheck":   {modulePath + "/internal/ir"},
	modulePath + "/internal/ir":            {},
	modulePath + "/internal/shell":         {modulePath + "/internal/ir"},
	modulePath + "/internal/expand":        {modulePath + "/internal/ir"},
	modulePath + "/internal/effect":        {modulePath + "/internal/ir"},
}

func repoRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "env", "GOMOD").Output()
	require.NoError(t, err, "go env GOMOD")
	modPath := strings.TrimSpace(string(out))
	require.NotEmpty(t, modPath, "GOMOD is empty")
	return filepath.Dir(modPath)
}

func goList(t *testing.T, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = repoRoot(t)
	out, err := cmd.Output()
	require.NoError(t, err, "go %s", strings.Join(args, " "))
	return out
}

func TestRequiredPackagesExist(t *testing.T) {
	out := goList(t, "list", "./...")
	listed := strings.Fields(string(out))
	slices.Sort(listed)

	for _, pkg := range requiredPackages {
		// Soft multi-check: report every missing package in one run.
		_, found := slices.BinarySearch(listed, pkg)
		assert.True(t, found, "missing required package %s", pkg)
	}
}

func TestForbiddenDirectoriesAbsent(t *testing.T) {
	root := repoRoot(t)
	for _, rel := range forbiddenPaths {
		path := filepath.Join(root, rel)
		_, err := os.Stat(path)
		if err == nil {
			assert.Fail(t, "forbidden path exists", "%s", rel)
			continue
		}
		require.True(t, os.IsNotExist(err), "stat %s: %v", rel, err)
	}
}

func TestInternalImportBoundaries(t *testing.T) {
	out := goList(t, "list", "-deps", "-f", "{{.ImportPath}} {{join .Imports \" \"}}", "./...")

	allowedSet := make(map[string]map[string]struct{}, len(allowedInternalImports))
	for pkg, imports := range allowedInternalImports {
		set := make(map[string]struct{}, len(imports))
		for _, imp := range imports {
			set[imp] = struct{}{}
		}
		allowedSet[pkg] = set
	}

	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		pkg := fields[0]
		allowed, ok := allowedSet[pkg]
		if !ok {
			continue
		}
		for _, imp := range fields[1:] {
			if !strings.HasPrefix(imp, modulePath+"/internal/") {
				continue
			}
			_, ok := allowed[imp]
			// Soft multi-check: report every forbidden import edge together.
			assert.True(t, ok, "%s must not import %s", pkg, imp)
		}
	}
}
