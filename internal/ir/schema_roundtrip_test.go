package ir_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Review remediation for the schema contract tests.
//
// The core review finding is that the previous schema tests validated the
// minimal example against a validator that lived inside the test file itself,
// so the "contract consistency" guarantee was partly self-referential. These
// tests add two independent anchors:
//
//  1. TestGoMarshalValidatesAgainstSchema builds a full report and asserts
//     that encoding/json output is accepted by the on-disk schema. This closes
//     the gap where production-serialized JSON could drift from the schema
//     without any test noticing.
//
//  2. TestSchemaEnumSetsMatchGoConstants pins each schema enum to the exact Go
//     constant set, so future edits to either side cannot silently diverge
//     (including the intentional Provenance(5) vs EvidenceSource(4) split).
//
// No external dependencies are introduced; json.Marshal is the only marshaller
// used, mirroring what the render package does in production.

// roundtrippableReport returns a report exercising every declared enum and
// nested path, to feed TestGoMarshalValidatesAgainstSchema.
func roundtrippableReport() map[string]any {
	return map[string]any{
		"schema_version": "0.1",
		"command":        "cat a.txt | grep x",
		"cwd":            "/home/who",
		"analysis": map[string]any{
			"coverage":     "complete",
			"completeness": "complete",
			"limits":       []any{},
			"parser":       "test-parser",
		},
		"stages": []any{
			map[string]any{
				"index":        0,
				"command":      "cat a.txt",
				"condition":    map[string]any{"kind": "always", "depends_on": 0},
				"completeness": "complete",
				"effects": []any{
					map[string]any{
						"id":         "e1",
						"kind":       "read",
						"raw_target": "a.txt",
						"target":     "/home/who/a.txt",
						"stage":      0,
						"certainty":  "certain",
						"provenance": "inferred",
						"condition":  map[string]any{"kind": "always", "depends_on": 0},
						"evidence": []any{
							map[string]any{
								"source":     "script",
								"path":       "build.sh",
								"field":      "ops",
								"start_byte": 5,
								"end_byte":   5,
								"snippet":    "cat",
							},
						},
					},
				},
			},
		},
		"unknowns": []any{
			map[string]any{
				"code":     "glob_runtime_dependent",
				"scope":    "stage:0",
				"message":  "glob depends on cwd",
				"evidence": []any{map[string]any{"source": "command", "field": "arg"}},
				"blocking": true,
			},
		},
		"flags":   []any{"destructive"},
		"summary": "ok",
	}
}

// TestGoMarshalValidatesAgainstSchema serializes a full report and asserts the
// on-disk schema accepts it. If Go struct and schema ever drift (e.g. a field
// the struct omits being marked required), this fails.
func TestGoMarshalValidatesAgainstSchema(t *testing.T) {
	var inst any
	raw, err := json.Marshal(roundtrippableReport())
	require.NoError(t, err, "marshal report")
	require.NoError(t, json.Unmarshal(raw, &inst), "unmarshal marshaled report")
	errs := validateSchema(inst, loadNode(t))
	require.Empty(t, errs, "report does not validate against schema:\n%s\n--- json ---\n%s",
		joinErrs(errs), string(raw))
}

// TestSchemaEnumSetsMatchGoConstants pins every schema enum to the exact Go
// constant set. The Provenance(5) vs Evidence(4) split is asserted explicitly.
func TestSchemaEnumSetsMatchGoConstants(t *testing.T) {
	doc := loadMap(t, schemaFilePath)
	stage := itemsOf(childProp(doc, "stages"))
	effect := itemsOf(childProp(stage, "effects"))

	cases := []struct {
		label string
		node  map[string]any
		want  []string
	}{
		{"analysis.coverage", childProp(childProp(doc, "analysis"), "coverage"), []string{"complete", "partial", "minimal"}},
		{"analysis.completeness", childProp(childProp(doc, "analysis"), "completeness"), []string{"complete", "partial", "unknown"}},
		{"stage.completeness", childProp(stage, "completeness"), []string{"complete", "partial", "unknown"}},
		{"stage.condition.kind", childProp(childProp(stage, "condition"), "kind"), []string{"always", "on_success", "on_failure"}},
		{"effect.kind", childProp(effect, "kind"), []string{"read", "write", "delete", "network", "process", "privilege", "execute_remote", "install"}},
		{"effect.certainty", childProp(effect, "certainty"), []string{"certain", "conditional", "possible", "unknown"}},
		{"effect.provenance", childProp(effect, "provenance"), []string{"command", "workspace_file", "script", "inferred", "caller_context"}},
		{"effect.condition.kind", childProp(childProp(effect, "condition"), "kind"), []string{"always", "on_success", "on_failure"}},
		{"effect.evidence.source", childProp(itemsOf(childProp(effect, "evidence")), "source"), []string{"command", "workspace_file", "script", "caller_context"}},
		{"unknown.evidence.source", childProp(itemsOf(childProp(itemsOf(childProp(doc, "unknowns")), "evidence")), "source"), []string{"command", "workspace_file", "script", "caller_context"}},
		{"unknown.code", childProp(itemsOf(childProp(doc, "unknowns")), "code"), unknownCodes()},
		{"flags", getChild(childProp(doc, "flags"), "items"), flags()},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			got, errMsg := enumArray(tc.node)
			require.Empty(t, errMsg, "%s", tc.label)
			require.True(t, sameStringSet(got, tc.want), "%s enum mismatch:\n  got  %v\n  want %v", tc.label, got, tc.want)
		})
	}
}

// enumArray returns the string elements of an enum-holding schema node.
func enumArray(node map[string]any) ([]string, string) {
	if node == nil {
		return nil, "definition missing in schema"
	}
	raw, ok := node["enum"].([]any)
	if !ok {
		return nil, "node has no enum array"
	}
	out := make([]string, 0, len(raw))
	for _, e := range raw {
		s, ok := e.(string)
		if !ok {
			return nil, "enum element is not a string"
		}
		out = append(out, s)
	}
	return out, ""
}

func unknownCodes() []string {
	return strings.Fields(`unsupported_syntax unsupported_command context_missing script_not_provided
script_dynamic_path env_missing glob_runtime_dependent command_substitution
remote_content interpreter_dynamic_code platform_dependent parse_error
input_too_large expansion_limit analysis_timeout expansion_cycle`)
}

func flags() []string {
	return []string{
		"destructive", "external_network", "privilege_change", "opaque_script",
		"remote_content_executed", "context_missing", "unsupported", "analysis_timeout",
	}
}

func joinErrs(errs []string) string {
	var sb strings.Builder
	for i, e := range errs {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString("- ")
		sb.WriteString(e)
	}
	return sb.String()
}

func sameStringSet(a, b []string) bool {
	sa, sb := slices.Clone(a), slices.Clone(b)
	slices.Sort(sa)
	slices.Sort(sb)
	return slices.Equal(sa, sb)
}
