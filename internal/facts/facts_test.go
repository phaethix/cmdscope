package facts_test

import (
	"encoding/json"
	"testing"

	"github.com/phaethix/runmark/internal/facts"
	"github.com/stretchr/testify/require"
)

func TestSchemaVersionExperimental(t *testing.T) {
	require.Equal(t, "0.1-touch-experimental", facts.SchemaVersion)
}

func TestEmptyValidatesAndMarshalsArrays(t *testing.T) {
	f := facts.Empty()
	require.NoError(t, facts.Validate(f))

	raw, err := json.Marshal(f)
	require.NoError(t, err)
	var asMap map[string]any
	require.NoError(t, json.Unmarshal(raw, &asMap))

	touches, ok := asMap["touches"].(map[string]any)
	require.True(t, ok)
	for _, key := range []string{"read", "write", "delete"} {
		arr, ok := touches[key].([]any)
		require.True(t, ok, "touches.%s must be an array", key)
		require.NotNil(t, arr)
		require.Len(t, arr, 0)
	}
	for _, key := range []string{"scripts", "unknown_reasons", "evidence"} {
		arr, ok := asMap[key].([]any)
		require.True(t, ok, "%s must be an array", key)
		require.NotNil(t, arr)
		require.Len(t, arr, 0)
	}
	require.Equal(t, facts.SchemaVersion, asMap["schema_version"])
}

func TestValidateRejectsNilSlices(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*facts.RunmarkFacts)
	}{
		{"touches.read", func(f *facts.RunmarkFacts) { f.Touches.Read = nil }},
		{"touches.write", func(f *facts.RunmarkFacts) { f.Touches.Write = nil }},
		{"touches.delete", func(f *facts.RunmarkFacts) { f.Touches.Delete = nil }},
		{"scripts", func(f *facts.RunmarkFacts) { f.Scripts = nil }},
		{"unknown_reasons", func(f *facts.RunmarkFacts) { f.UnknownReasons = nil }},
		{"evidence", func(f *facts.RunmarkFacts) { f.Evidence = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := facts.Empty()
			tc.mutate(&f)
			require.Error(t, facts.Validate(f))
		})
	}
}

func TestValidateRejectsSchemaDrift(t *testing.T) {
	f := facts.Empty()
	f.SchemaVersion = "1.0"
	require.Error(t, facts.Validate(f))
}

func TestNormalizeStableSort(t *testing.T) {
	f := facts.Empty()
	f.Touches.Read = []string{"b", "a", "a"}
	f.Touches.Write = []string{"z", "m"}
	f.Touches.Delete = []string{"2", "1"}
	f.UnknownReasons = []string{"beta", "alpha"}
	f.Scripts = []facts.ScriptEntry{
		{Tool: "npm", Name: "b", Source: "package.json"},
		{Tool: "make", Name: "a", Source: "Makefile"},
		{Tool: "npm", Name: "a", Source: "package.json"},
	}
	f.Evidence = []facts.FactEvidence{
		{Source: "command", Snippet: "b"},
		{Source: "command", Snippet: "a"},
		{Source: "aggregate", Path: "x"},
	}

	facts.Normalize(&f)
	require.NoError(t, facts.Validate(f))
	require.Equal(t, []string{"a", "a", "b"}, f.Touches.Read)
	require.Equal(t, []string{"m", "z"}, f.Touches.Write)
	require.Equal(t, []string{"1", "2"}, f.Touches.Delete)
	require.Equal(t, []string{"alpha", "beta"}, f.UnknownReasons)
	require.Equal(t, "make", f.Scripts[0].Tool)
	require.Equal(t, "npm", f.Scripts[1].Tool)
	require.Equal(t, "a", f.Scripts[1].Name)
	require.Equal(t, "b", f.Scripts[2].Name)
	require.Equal(t, "aggregate", f.Evidence[0].Source)
	require.Equal(t, "a", f.Evidence[1].Snippet)
	require.Equal(t, "b", f.Evidence[2].Snippet)

	again := f
	facts.Normalize(&again)
	require.Equal(t, f, again, "Normalize must be idempotent")
}
