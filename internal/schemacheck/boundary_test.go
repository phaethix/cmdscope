package schemacheck_test

// These boundary checks deliberately live in the schemacheck black-box test
// package (package schemacheck_test) rather than internal/integration:
// enforcing package layout and import direction is part of schemacheck's
// future contract-checking responsibility, and internal/integration is not
// created yet. The os/exec calls below only run the go toolchain
// (go env / go list) to inspect module metadata; they never execute the
// command under analysis, so they stay within the §9 red lines.

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
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
	if err != nil {
		t.Fatalf("go env GOMOD: %v", err)
	}
	modPath := strings.TrimSpace(string(out))
	if modPath == "" {
		t.Fatal("GOMOD is empty")
	}
	return filepath.Dir(modPath)
}

func goList(t *testing.T, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = repoRoot(t)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go %s: %v", strings.Join(args, " "), err)
	}
	return out
}

func TestRequiredPackagesExist(t *testing.T) {
	out := goList(t, "list", "./...")
	listed := strings.Fields(string(out))
	sort.Strings(listed)

	for _, pkg := range requiredPackages {
		idx := sort.SearchStrings(listed, pkg)
		if idx >= len(listed) || listed[idx] != pkg {
			t.Errorf("missing required package %s", pkg)
		}
	}
}

func TestForbiddenDirectoriesAbsent(t *testing.T) {
	root := repoRoot(t)
	for _, rel := range forbiddenPaths {
		path := filepath.Join(root, rel)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("forbidden path exists: %s", rel)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", rel, err)
		}
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

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
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
			if _, ok := allowed[imp]; !ok {
				t.Errorf("%s must not import %s", pkg, imp)
			}
		}
	}
}
