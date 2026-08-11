package ir_test

import (
	"encoding/json"
	"testing"

	"github.com/phaethix/cmdscope/internal/ir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReportSchemaVersionConstant locks the fixed first-release schema version.
func TestReportSchemaVersionConstant(t *testing.T) {
	require.Equal(t, "0.1", ir.SchemaVersion)
}

// TestReportConditionDependsOnAlwaysSerialized asserts Condition.DependsOn is
// emitted even when zero (architecture §3.2: no omitempty on depends_on).
func TestReportConditionDependsOnAlwaysSerialized(t *testing.T) {
	r := ir.ImpactReport{
		SchemaVersion: ir.SchemaVersion,
		Command:       "echo hi",
		Analysis: ir.AnalysisMeta{
			Coverage:     ir.CoverageComplete,
			Completeness: ir.CompletenessComplete,
			Limits:       []string{},
			Parser:       "bash-l0",
		},
		Stages: []ir.Stage{{
			Index:        0,
			Command:      "echo hi",
			Condition:    ir.Condition{Kind: ir.ConditionAlways, DependsOn: 0},
			Completeness: ir.CompletenessComplete,
			Effects:      []ir.Effect{},
		}},
		Unknowns: []ir.Unknown{},
		Flags:    []ir.Flag{},
	}
	data, err := json.Marshal(r)
	require.NoError(t, err)
	require.Contains(t, string(data), `"depends_on":0`)
}

// TestReportEffectRequiredFieldsAlwaysSerialized asserts raw_target and target
// are always present even when empty (architecture §3.2.1).
func TestReportEffectRequiredFieldsAlwaysSerialized(t *testing.T) {
	effect := ir.Effect{
		ID:         "sha256:test",
		Kind:       ir.EffectProcess,
		RawTarget:  "",
		Target:     "",
		Stage:      0,
		Certainty:  ir.Certain,
		Provenance: ir.FromCommand,
		Condition:  ir.Condition{Kind: ir.ConditionAlways},
		Evidence:   []ir.Evidence{{Source: ir.EvidenceCommand, Snippet: "echo"}},
	}
	data, err := json.Marshal(effect)
	require.NoError(t, err)
	got := string(data)
	// Soft multi-check: report every missing required wire field together.
	assert.Contains(t, got, `"raw_target":""`)
	assert.Contains(t, got, `"target":""`)
	assert.Contains(t, got, `"id":"sha256:test"`)
}

// TestReportEvidenceSpanPointers asserts *int span fields: nil is omitted and
// a zero-valued pointer is still emitted (architecture §3.2).
func TestReportEvidenceSpanPointers(t *testing.T) {
	zero := 0
	ten := 10
	withSpan := ir.Evidence{Source: ir.EvidenceCommand, StartByte: &zero, EndByte: &ten}
	data, err := json.Marshal(withSpan)
	require.NoError(t, err)
	got := string(data)
	assert.Contains(t, got, `"start_byte":0`)
	assert.Contains(t, got, `"end_byte":10`)

	nilSpan := ir.Evidence{Source: ir.EvidenceCommand}
	data, err = json.Marshal(nilSpan)
	require.NoError(t, err)
	got = string(data)
	assert.NotContains(t, got, "start_byte")
	assert.NotContains(t, got, "end_byte")
}

// TestJSONMinimalReportShape unmarshals the canonical example shape from
// architecture §3.2.2 and re-marshals it, asserting the contract fields.
func TestJSONMinimalReportShape(t *testing.T) {
	const input = `{
  "schema_version": "0.1",
  "command": "echo hi > output.txt",
  "cwd": "/workspace",
  "analysis": {
    "coverage": "complete",
    "completeness": "complete",
    "limits": [],
    "parser": "bash-l0"
  },
"stages": [
    {
      "index": 0,
      "command": "echo hi > output.txt",
      "condition": {"kind": "always", "depends_on": 0},
      "completeness": "complete",
      "effects": [
        {
          "id": "sha256:example",
          "kind": "write",
          "raw_target": "output.txt",
          "target": "/workspace/output.txt",
          "stage": 0,
          "certainty": "certain",
          "provenance": "command",
          "condition": {"kind": "always", "depends_on": 0},
          "evidence": [
            {"source": "command", "start_byte": 8, "end_byte": 20, "snippet": "> output.txt"}
          ]
        }
      ]
    }
  ],
  "unknowns": [],
  "flags": [],
  "summary": "1 write effect"
}`
	var r ir.ImpactReport
	require.NoError(t, json.Unmarshal([]byte(input), &r))
	assert.Equal(t, "0.1", r.SchemaVersion)
	assert.Equal(t, "/workspace", r.CWD)
	assert.Equal(t, ir.CoverageComplete, r.Analysis.Coverage)
	assert.Equal(t, ir.CompletenessComplete, r.Analysis.Completeness)
	assert.Equal(t, "bash-l0", r.Analysis.Parser)
	require.Len(t, r.Stages, 1)
	st := r.Stages[0]
	assert.Equal(t, 0, st.Index)
	assert.Equal(t, ir.ConditionAlways, st.Condition.Kind)
	assert.Equal(t, 0, st.Condition.DependsOn)
	require.Len(t, st.Effects, 1)
	ef := st.Effects[0]
	assert.Equal(t, ir.EffectWrite, ef.Kind)
	assert.Equal(t, "output.txt", ef.RawTarget)
	assert.Equal(t, "/workspace/output.txt", ef.Target)
	assert.Equal(t, 0, ef.Stage)
	assert.Equal(t, ir.Certain, ef.Certainty)
	assert.Equal(t, ir.FromCommand, ef.Provenance)
	require.Len(t, ef.Evidence, 1)
	ev := ef.Evidence[0]
	assert.Equal(t, ir.EvidenceCommand, ev.Source)
	assert.Equal(t, "> output.txt", ev.Snippet)
	require.NotNil(t, ev.StartByte)
	assert.Equal(t, 8, *ev.StartByte)
	require.NotNil(t, ev.EndByte)
	assert.Equal(t, 20, *ev.EndByte)
	assert.Empty(t, r.Unknowns)
	assert.Empty(t, r.Flags)

	out, err := json.Marshal(r)
	require.NoError(t, err)
	got := string(out)
	for _, want := range []string{
		`"schema_version":"0.1"`,
		`"cwd":"/workspace"`,
		`"coverage":"complete"`,
		`"completeness":"complete"`,
		`"limits":[]`,
		`"parser":"bash-l0"`,
		`"depends_on":0`,
		`"kind":"write"`,
		`"raw_target":"output.txt"`,
		`"target":"/workspace/output.txt"`,
		`"certainty":"certain"`,
		`"provenance":"command"`,
		`"start_byte":8`,
		`"end_byte":20`,
		`"unknowns":[]`,
		`"flags":[]`,
	} {
		assert.Contains(t, got, want)
	}
	assert.NotContains(t, got, `"stages":null`)
	assert.NotContains(t, got, `"unknowns":null`)
	assert.NotContains(t, got, `"flags":null`)
	assert.NotContains(t, got, `"limits":null`)
	assert.NotContains(t, got, `"effects":null`)
}

// TestReportEmptyArraysSerializeAsSlices pins that every array field on an
// empty report with initialized slices serializes as [] rather than null.
func TestReportEmptyArraysSerializeAsSlices(t *testing.T) {
	r := ir.ImpactReport{
		SchemaVersion: ir.SchemaVersion,
		Command:       "echo hi",
		Analysis: ir.AnalysisMeta{
			Coverage:     ir.CoverageComplete,
			Completeness: ir.CompletenessComplete,
			Limits:       []string{},
			Parser:       "bash-l0",
		},
		Stages:   []ir.Stage{},
		Unknowns: []ir.Unknown{},
		Flags:    []ir.Flag{},
	}
	data, err := json.Marshal(r)
	require.NoError(t, err)
	got := string(data)
	for _, want := range []string{`"limits":[]`, `"stages":[]`, `"unknowns":[]`, `"flags":[]`} {
		assert.Contains(t, got, want)
	}
}

// TestReportCWDIsOmitEmpty asserts an empty CWD is omitted from JSON.
func TestReportCWDIsOmitEmpty(t *testing.T) {
	r := ir.ImpactReport{
		SchemaVersion: ir.SchemaVersion,
		Command:       "echo hi",
		Analysis: ir.AnalysisMeta{
			Coverage:     ir.CoverageComplete,
			Completeness: ir.CompletenessComplete,
			Limits:       []string{},
			Parser:       "bash-l0",
		},
		Stages:   []ir.Stage{},
		Unknowns: []ir.Unknown{},
		Flags:    []ir.Flag{},
	}
	data, err := json.Marshal(r)
	require.NoError(t, err)
	assert.NotContains(t, string(data), `"cwd"`)
}
