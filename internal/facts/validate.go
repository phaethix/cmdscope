package facts

import (
	"cmp"
	"fmt"
	"slices"
)

// Empty returns a Validate-ready facts value with every slice non-nil so JSON
// marshals as [] rather than null.
func Empty() RunmarkFacts {
	return RunmarkFacts{
		SchemaVersion: SchemaVersion,
		Touches: TouchSet{
			Read:   []string{},
			Write:  []string{},
			Delete: []string{},
		},
		Scripts:        []ScriptEntry{},
		UnknownReasons: []string{},
		Evidence:       []FactEvidence{},
	}
}

// Normalize sorts list fields for deterministic output. It does not dedupe
// paths — that is projection policy and belongs with ImpactReport→Facts.
func Normalize(f *RunmarkFacts) {
	if f == nil {
		return
	}
	if f.Touches.Read == nil {
		f.Touches.Read = []string{}
	}
	if f.Touches.Write == nil {
		f.Touches.Write = []string{}
	}
	if f.Touches.Delete == nil {
		f.Touches.Delete = []string{}
	}
	if f.Scripts == nil {
		f.Scripts = []ScriptEntry{}
	}
	if f.UnknownReasons == nil {
		f.UnknownReasons = []string{}
	}
	if f.Evidence == nil {
		f.Evidence = []FactEvidence{}
	}

	slices.Sort(f.Touches.Read)
	slices.Sort(f.Touches.Write)
	slices.Sort(f.Touches.Delete)
	slices.Sort(f.UnknownReasons)
	slices.SortFunc(f.Scripts, func(a, b ScriptEntry) int {
		return cmp.Or(
			cmp.Compare(a.Tool, b.Tool),
			cmp.Compare(a.Name, b.Name),
			cmp.Compare(a.Source, b.Source),
		)
	})
	slices.SortFunc(f.Evidence, func(a, b FactEvidence) int {
		return cmp.Or(
			cmp.Compare(a.Source, b.Source),
			cmp.Compare(a.Path, b.Path),
			cmp.Compare(a.Field, b.Field),
			cmp.Compare(a.Snippet, b.Snippet),
		)
	})
}

// Validate checks experimental wire invariants. It rejects nil slices and
// schema_version drift; it does not invent projection results.
func Validate(f RunmarkFacts) error {
	if f.SchemaVersion != SchemaVersion {
		return fmt.Errorf("facts: schema_version %q want %q", f.SchemaVersion, SchemaVersion)
	}
	if f.Touches.Read == nil {
		return fmt.Errorf("facts: touches.read must be non-nil")
	}
	if f.Touches.Write == nil {
		return fmt.Errorf("facts: touches.write must be non-nil")
	}
	if f.Touches.Delete == nil {
		return fmt.Errorf("facts: touches.delete must be non-nil")
	}
	if f.Scripts == nil {
		return fmt.Errorf("facts: scripts must be non-nil")
	}
	if f.UnknownReasons == nil {
		return fmt.Errorf("facts: unknown_reasons must be non-nil")
	}
	if f.Evidence == nil {
		return fmt.Errorf("facts: evidence must be non-nil")
	}
	return nil
}
