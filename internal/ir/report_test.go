package ir_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/phaethix/cmdscope/internal/ir"
)

// TestReportSchemaVersionConstant locks the fixed first-release schema version.
func TestReportSchemaVersionConstant(t *testing.T) {
	if ir.SchemaVersion != "0.1" {
		t.Fatalf("SchemaVersion = %q, want %q", ir.SchemaVersion, "0.1")
	}
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
			Index:        1,
			Command:      "echo hi",
			Condition:    ir.Condition{Kind: ir.ConditionAlways, DependsOn: 0},
			Completeness: ir.CompletenessComplete,
			Effects:      []ir.Effect{},
		}},
		Unknowns: []ir.Unknown{},
		Flags:    []ir.Flag{},
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal() = %v", err)
	}
	got := string(data)
	if !strings.Contains(got, `"depends_on":0`) {
		t.Fatalf("JSON missing depends_on:0: %s", got)
	}
}

// TestReportEffectRequiredFieldsAlwaysSerialized asserts raw_target and target
// are always present even when empty (architecture §3.2.1).
func TestReportEffectRequiredFieldsAlwaysSerialized(t *testing.T) {
	effect := ir.Effect{
		ID:         "sha256:test",
		Kind:       ir.EffectProcess,
		RawTarget:  "",
		Target:     "",
		Stage:      1,
		Certainty:  ir.Certain,
		Provenance: ir.FromCommand,
		Condition:  ir.Condition{Kind: ir.ConditionAlways},
		Evidence:   []ir.Evidence{{Source: ir.EvidenceCommand, Snippet: "echo"}},
	}
	data, err := json.Marshal(effect)
	if err != nil {
		t.Fatalf("json.Marshal() = %v", err)
	}
	got := string(data)
	if !strings.Contains(got, `"raw_target":""`) {
		t.Errorf("JSON missing raw_target empty field: %s", got)
	}
	if !strings.Contains(got, `"target":""`) {
		t.Errorf("JSON missing target empty field: %s", got)
	}
	if !strings.Contains(got, `"id":"sha256:test"`) {
		t.Errorf("JSON missing id field: %s", got)
	}
}

// TestReportEvidenceSpanPointers asserts *int span fields: nil is omitted and
// a zero-valued pointer is still emitted (architecture §3.2).
func TestReportEvidenceSpanPointers(t *testing.T) {
	zero := 0
	ten := 10
	withSpan := ir.Evidence{Source: ir.EvidenceCommand, StartByte: &zero, EndByte: &ten}
	data, err := json.Marshal(withSpan)
	if err != nil {
		t.Fatalf("json.Marshal() = %v", err)
	}
	got := string(data)
	if !strings.Contains(got, `"start_byte":0`) {
		t.Errorf("JSON missing start_byte:0 for non-nil pointer: %s", got)
	}
	if !strings.Contains(got, `"end_byte":10`) {
		t.Errorf("JSON missing end_byte:10: %s", got)
	}

	nilSpan := ir.Evidence{Source: ir.EvidenceCommand}
	data, err = json.Marshal(nilSpan)
	if err != nil {
		t.Fatalf("json.Marshal() = %v", err)
	}
	got = string(data)
	if strings.Contains(got, "start_byte") {
		t.Errorf("JSON should omit nil start_byte: %s", got)
	}
	if strings.Contains(got, "end_byte") {
		t.Errorf("JSON should omit nil end_byte: %s", got)
	}
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
      "index": 1,
      "command": "echo hi > output.txt",
      "condition": {"kind": "always", "depends_on": 0},
      "completeness": "complete",
      "effects": [
        {
          "id": "sha256:example",
          "kind": "write",
          "raw_target": "output.txt",
          "target": "/workspace/output.txt",
          "stage": 1,
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
	if err := json.Unmarshal([]byte(input), &r); err != nil {
		t.Fatalf("json.Unmarshal() = %v", err)
	}
	if r.SchemaVersion != "0.1" {
		t.Errorf("SchemaVersion = %q, want 0.1", r.SchemaVersion)
	}
	if r.CWD != "/workspace" {
		t.Errorf("CWD = %q, want /workspace", r.CWD)
	}
	if r.Analysis.Coverage != ir.CoverageComplete || r.Analysis.Completeness != ir.CompletenessComplete {
		t.Errorf("Analysis coverage/completeness = %q/%q", r.Analysis.Coverage, r.Analysis.Completeness)
	}
	if r.Analysis.Parser != "bash-l0" {
		t.Errorf("Parser = %q, want bash-l0", r.Analysis.Parser)
	}
	if len(r.Stages) != 1 {
		t.Fatalf("len(Stages) = %d, want 1", len(r.Stages))
	}
	st := r.Stages[0]
	if st.Index != 1 || st.Condition.Kind != ir.ConditionAlways || st.Condition.DependsOn != 0 {
		t.Errorf("Stage = %+v", st)
	}
	if len(st.Effects) != 1 {
		t.Fatalf("len(Effects) = %d, want 1", len(st.Effects))
	}
	ef := st.Effects[0]
	if ef.Kind != ir.EffectWrite || ef.RawTarget != "output.txt" || ef.Target != "/workspace/output.txt" {
		t.Errorf("Effect = %+v", ef)
	}
	if ef.Stage != 1 || ef.Certainty != ir.Certain || ef.Provenance != ir.FromCommand {
		t.Errorf("Effect meta = %+v", ef)
	}
	if len(ef.Evidence) != 1 {
		t.Fatalf("len(Evidence) = %d, want 1", len(ef.Evidence))
	}
	ev := ef.Evidence[0]
	if ev.Source != ir.EvidenceCommand || ev.Snippet != "> output.txt" {
		t.Errorf("Evidence = %+v", ev)
	}
	if ev.StartByte == nil || *ev.StartByte != 8 {
		t.Errorf("StartByte = %v, want 8", ev.StartByte)
	}
	if ev.EndByte == nil || *ev.EndByte != 20 {
		t.Errorf("EndByte = %v, want 20", ev.EndByte)
	}
	if len(r.Unknowns) != 0 || len(r.Flags) != 0 {
		t.Errorf("Unknowns/Flags = %d/%d, want 0/0", len(r.Unknowns), len(r.Flags))
	}

	out, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal() = %v", err)
	}
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
		if !strings.Contains(got, want) {
			t.Errorf("marshaled JSON missing %s:\n%s", want, got)
		}
	}
	if strings.Contains(got, `"stages":null`) || strings.Contains(got, `"unknowns":null`) {
		t.Errorf("arrays must serialize as [] not null:\n%s", got)
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
	if err != nil {
		t.Fatalf("json.Marshal() = %v", err)
	}
	if strings.Contains(string(data), `"cwd"`) {
		t.Errorf("empty CWD should be omitted: %s", data)
	}
}
