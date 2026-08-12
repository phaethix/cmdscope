// Package spike runs Spike differentiation fixtures under testdata/spike.
//
// Cases assert RunmarkFacts, not CLI text. baseline.md is narrative only and
// must not invent third-party Guardrail version claims.
package spike

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/phaethix/runmark/internal/analyzer"
	"github.com/phaethix/runmark/internal/facts"
	"github.com/phaethix/runmark/internal/ir"
)

// Case is one Spike fixture directory after load.
type Case struct {
	Name     string
	Dir      string
	Command  string
	Context  *ir.AnalysisContext
	Expected facts.RunmarkFacts
}

type requestFile struct {
	Command string `json:"command"`
}

// Discover loads every immediate subdirectory of root as a Spike case.
func Discover(root string) ([]Case, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("spike: read %s: %w", root, err)
	}
	var out []Case
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		c, err := LoadCase(dir)
		if err != nil {
			return nil, fmt.Errorf("spike: case %s: %w", e.Name(), err)
		}
		out = append(out, c)
	}
	slices.SortFunc(out, func(a, b Case) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return out, nil
}

// LoadCase reads the four required fixture files from dir.
func LoadCase(dir string) (Case, error) {
	name := filepath.Base(dir)
	reqBytes, err := os.ReadFile(filepath.Join(dir, "request.json"))
	if err != nil {
		return Case{}, fmt.Errorf("request.json: %w", err)
	}
	var req requestFile
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return Case{}, fmt.Errorf("request.json: %w", err)
	}
	if req.Command == "" {
		return Case{}, fmt.Errorf("request.json: command must not be empty")
	}

	ctxBytes, err := os.ReadFile(filepath.Join(dir, "context.json"))
	if err != nil {
		return Case{}, fmt.Errorf("context.json: %w", err)
	}
	ctx, err := ir.ParseAnalysisContextJSON(ctxBytes)
	if err != nil {
		return Case{}, fmt.Errorf("context.json: %w", err)
	}

	expBytes, err := os.ReadFile(filepath.Join(dir, "expected-facts.json"))
	if err != nil {
		return Case{}, fmt.Errorf("expected-facts.json: %w", err)
	}
	var expected facts.RunmarkFacts
	if err := json.Unmarshal(expBytes, &expected); err != nil {
		return Case{}, fmt.Errorf("expected-facts.json: %w", err)
	}
	facts.Normalize(&expected)

	if _, err := os.Stat(filepath.Join(dir, "baseline.md")); err != nil {
		return Case{}, fmt.Errorf("baseline.md: %w", err)
	}

	return Case{
		Name:     name,
		Dir:      dir,
		Command:  req.Command,
		Context:  ctx,
		Expected: expected,
	}, nil
}

// RunCase analyzes the case the same way the CLI facts path does: Analyze →
// ValidateReport → Project → Validate/Normalize. It does not shell out.
func RunCase(c Case) (facts.RunmarkFacts, error) {
	report, err := analyzer.Analyze(context.Background(), ir.AnalyzeRequest{
		Command: c.Command,
		Context: c.Context,
	})
	if err != nil {
		return facts.RunmarkFacts{}, err
	}
	if err := ir.ValidateReport(report); err != nil {
		return facts.RunmarkFacts{}, err
	}
	got := facts.Project(report)
	if err := facts.Validate(got); err != nil {
		return facts.RunmarkFacts{}, err
	}
	facts.Normalize(&got)
	return got, nil
}
