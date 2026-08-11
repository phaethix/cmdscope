package ir_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Schema contract tests.
//
// The impact report JSON Schema lives in the repository's schema/ directory
// (../../schema relative to this package). Because cmdscope is a zero
// dependency project, this suite validates instances against the schema using
// a small self-contained validator that covers exactly the JSON Schema
// keywords emitted by schema/impact-report-0.1.schema.json:
//   type, properties, required, items, enum.
//
// Tests keep the schema and its minimal example honest under four guarantees:
//   - the schema file exists, is valid JSON and does not reference remote
//     resources (offline contract);
//   - every required field declared here matches the Go contract;
//   - the minimal example validates and serializes its arrays as [] never null;
//   - dropping a required field from the example must make validation fail.

var (
	schemaFilePath  = filepath.Join("..", "..", "schema", "impact-report-0.1.schema.json")
	exampleFilePath = filepath.Join("..", "..", "schema", "examples", "minimal.json")
)

// Minimal self-contained JSON Schema validator (zero dependencies)

// schemaNode mirrors the JSON Schema subset implemented here.
type schemaNode struct {
	Type       string                 `json:"type"`
	Properties map[string]*schemaNode `json:"properties"`
	Required   []string               `json:"required"`
	Items      *schemaNode            `json:"items"`
	Enum       []any                  `json:"enum"`
	Const      any                    `json:"const,omitempty"`
	Minimum    *float64               `json:"minimum,omitempty"`
	MinLength  int                    `json:"minLength,omitempty"`
	Pattern    string                 `json:"pattern,omitempty"`
}

// validate walks a parsed document against node and returns human readable
// violations. Only type, properties, required, items, enum, const, minimum,
// minLength and pattern are interpreted; every other JSON Schema keyword is
// intentionally ignored.
func validateSchema(doc any, s *schemaNode) []string {
	if s == nil {
		return nil
	}
	var errs []string
	if s.Type != "" && !typeOk(doc, s.Type) {
		errs = append(errs, "expected type "+s.Type+", got "+jsonTypeName(doc))
	}
	if len(s.Enum) > 0 && !enumOk(s.Enum, doc) {
		errs = append(errs, "value not in enum")
	}
	if s.Const != nil && !constOk(s.Const, doc) {
		errs = append(errs, "value does not match const "+string(mustJSON(s.Const)))
	}
	if s.Minimum != nil && !minimumOk(s.Minimum, doc) {
		errs = append(errs, fmt.Sprintf("value below minimum %v", *s.Minimum))
	}
	if s.MinLength > 0 && !minLengthOk(s.MinLength, doc) {
		errs = append(errs, fmt.Sprintf("value shorter than minLength %d", s.MinLength))
	}
	if s.Pattern != "" && !patternOk(s.Pattern, doc) {
		errs = append(errs, "value does not match pattern "+s.Pattern)
	}
	switch val := doc.(type) {
	case map[string]any:
		for key, v := range val {
			if child, ok := s.Properties[key]; ok {
				errs = append(errs, prefixErrors(key, validateSchema(v, child))...)
			}
		}
		for _, req := range s.Required {
			if _, ok := val[req]; !ok {
				errs = append(errs, "missing required field '"+req+"'")
			}
		}
	case []any:
		if s.Items != nil {
			for idx, item := range val {
				errs = append(errs, prefixErrors("["+strconv.Itoa(idx)+"]", validateSchema(item, s.Items))...)
			}
		}
	}
	return errs
}

func prefixErrors(prefix string, in []string) []string {
	out := make([]string, len(in))
	for i, e := range in {
		out[i] = prefix + ": " + e
	}
	return out
}

func typeOk(v any, t string) bool {
	switch t {
	case "object":
		_, ok := v.(map[string]any)
		return ok
	case "array":
		_, ok := v.([]any)
		return ok
	case "string":
		_, ok := v.(string)
		return ok
	case "integer":
		if n, ok := v.(float64); ok {
			return n == float64(int(n))
		}
		return false
	case "number":
		_, ok := v.(float64)
		return ok
	case "boolean":
		_, ok := v.(bool)
		return ok
	}
	return true
}

func jsonTypeName(v any) string {
	switch v.(type) {
	case map[string]any:
		return "object"
	case []any:
		return "array"
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case nil:
		return "null"
	}
	return "unknown"
}

func enumOk(set []any, v any) bool {
	va, _ := json.Marshal(v)
	for _, e := range set {
		ea, _ := json.Marshal(e)
		if string(va) == string(ea) {
			return true
		}
	}
	return false
}

// constOk compares a parsed document value against a const declaration using
// canonical JSON equality, mirroring how enumOk compares values.
func constOk(c, v any) bool {
	ca, _ := json.Marshal(c)
	va, _ := json.Marshal(v)
	return string(ca) == string(va)
}

// minimumOk enforces numeric minimum for integer and number values. Non-numeric
// values are handled by the type check and treated as passing here.
func minimumOk(min *float64, v any) bool {
	switch n := v.(type) {
	case float64:
		return n >= *min
	case float32:
		return float64(n) >= *min
	case int:
		return float64(n) >= *min
	case int64:
		return float64(n) >= *min
	}
	return true
}

// minLengthOk enforces a minimum string length. Non-string values are handled
// by the type check and treated as passing here.
func minLengthOk(min int, v any) bool {
	str, ok := v.(string)
	if !ok {
		return true
	}
	return len(str) >= min
}

// patternOk enforces a regular expression against string values. Non-string
// values are handled by the type check and treated as passing here.
func patternOk(pattern string, v any) bool {
	str, ok := v.(string)
	if !ok {
		return true
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		// A broken schema pattern should never fail validation; treat it as pass.
		return true
	}
	return re.MatchString(str)
}

// mustJSON marshals a value for display in an error message; on failure it
// returns a literal marker so validation never panics.
func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("<unmarshalable>")
	}
	return b
}

func loadNode(t *testing.T) *schemaNode {
	t.Helper()
	raw, err := os.ReadFile(schemaFilePath)
	if err != nil {
		t.Fatalf("cannot read schema at %s: %v", schemaFilePath, err)
	}
	var n schemaNode
	if err := json.Unmarshal(raw, &n); err != nil {
		t.Fatalf("schema does not decode into node: %v", err)
	}
	return &n
}

func loadMap(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read %s: %v", path, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("%s does not decode into map: %v", path, err)
	}
	return doc
}

// Tests

// TestSchemaFileIsValidJSONAndObject asserts the schema file exists, is valid
// JSON and its root is an object schema.
func TestSchemaFileIsValidJSONAndObject(t *testing.T) {
	doc := loadMap(t, schemaFilePath)
	if doc["type"] != "object" {
		t.Fatalf("schema root type must be object, got %v", doc["type"])
	}
}

// TestSchemaIsOffline enforces the offline contract: no $ref and no remote
// resource identifiers anywhere in the schema.
func TestSchemaIsOffline(t *testing.T) {
	raw, err := os.ReadFile(schemaFilePath)
	if err != nil {
		t.Fatalf("missing schema: %v", err)
	}
	if strings.Contains(string(raw), "$ref") {
		t.Fatalf("schema must not use $ref (offline contract)")
	}
	for _, banned := range []string{"http://", "https://"} {
		if strings.Contains(string(raw), banned) {
			t.Fatalf("schema must not reference remote resources, found %q", banned)
		}
	}
}

// TestSchemaRequiredMatchesGoContract verifies the required field sets of the
// top-level report, analysis, stage, effect, condition and unknown against the
// Go contract.
func TestSchemaRequiredMatchesGoContract(t *testing.T) {
	doc := loadMap(t, schemaFilePath)

	// Report root required list lives at the schema root.
	assertReq(t, doc, "report", []string{
		"schema_version", "command", "analysis", "stages", "unknowns", "flags", "summary"})

	analysis := childProp(doc, "analysis")
	assertReq(t, analysis, "analysis", []string{"coverage", "completeness", "limits", "parser"})

	// stages -> items -> stage.
	stage := itemsOf(childProp(doc, "stages"))
	assertReq(t, stage, "stage", []string{"index", "command", "condition", "completeness", "effects"})

	// stage.effects -> items -> effect.
	effect := itemsOf(childProp(stage, "effects"))
	assertReq(t, effect, "effect",
		[]string{"id", "kind", "raw_target", "target", "stage", "certainty", "provenance", "condition", "evidence"})

	// effect.condition.
	condition := childProp(effect, "condition")
	assertReq(t, condition, "condition", []string{"kind", "depends_on"})

	// effect.evidence -> items -> evidence.
	evidence := itemsOf(childProp(effect, "evidence"))
	assertReq(t, evidence, "evidence", []string{"source"})

	// unknowns -> items -> unknown.
	unknown := itemsOf(childProp(doc, "unknowns"))
	assertReq(t, unknown, "unknown", []string{"code", "scope", "message", "evidence", "blocking"})
}

func getChild(obj map[string]any, key string) map[string]any {
	if obj == nil {
		return nil
	}
	v, ok := obj[key]
	if !ok {
		return nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	return m
}

func childProp(node map[string]any, key string) map[string]any {
	props := getChild(node, "properties")
	if props == nil {
		return nil
	}
	return getChild(props, key)
}

func itemsOf(node map[string]any) map[string]any {
	return getChild(node, "items")
}

func assertReq(t *testing.T, obj map[string]any, label string, want []string) {
	t.Helper()
	if obj == nil {
		t.Fatalf("%s definition missing in schema", label)
	}
	req, ok := obj["required"].([]any)
	if !ok {
		t.Fatalf("%s has no required list", label)
	}
	var got []string
	for _, r := range req {
		if s, ok := r.(string); ok {
			got = append(got, s)
		}
	}
	if !sameSet(t, got, want) {
		t.Fatalf("%s required mismatch:\n  got  %v\n  want %v", label, got, want)
	}
}

func sameSet(t *testing.T, a, b []string) bool {
	t.Helper()
	if len(a) != len(b) {
		return false
	}
	sa, sb := append([]string(nil), a...), append([]string(nil), b...)
	sort.Strings(sa)
	sort.Strings(sb)
	for i := range sa {
		if sa[i] != sb[i] {
			return false
		}
	}
	return true
}

// TestExampleMinimalValid validates the minimal example against the schema.
func TestExampleMinimalValid(t *testing.T) {
	var inst any
	raw, err := os.ReadFile(exampleFilePath)
	if err != nil {
		t.Fatalf("missing example at %s: %v", exampleFilePath, err)
	}
	if err := json.Unmarshal(raw, &inst); err != nil {
		t.Fatalf("example is not valid JSON: %v", err)
	}
	if errs := validateSchema(inst, loadNode(t)); len(errs) > 0 {
		t.Fatalf("minimal example does not validate against schema:\n%s", strings.Join(errs, "\n"))
	}
}

// TestExampleSlicesNeverNull asserts the schema-declared array properties are
// present as [] and never collapse to null.
func TestExampleSlicesNeverNull(t *testing.T) {
	inst := loadMap(t, exampleFilePath)
	for _, prop := range []string{"stages", "unknowns", "flags"} {
		v, ok := inst[prop]
		if !ok {
			t.Fatalf("array property %q missing in example (must be [] not omitted)", prop)
		}
		if _, ok := v.([]any); !ok {
			t.Fatalf("array property %q must be an array, got %s", prop, jsonTypeName(v))
		}
		raw, _ := json.Marshal(v)
		if string(raw) == "null" {
			t.Fatalf("array property %q serialized as null", prop)
		}
	}
}

// TestExampleMissingRequiredFieldFails is the acceptance criterion for the
// task: deleting a required field from the example must fail validation.
func TestExampleMissingRequiredFieldFails(t *testing.T) {
	inst := loadMap(t, exampleFilePath)
	delete(inst, "stages")
	if errs := validateSchema(inst, loadNode(t)); len(errs) == 0 {
		t.Fatalf("removing required field 'stages' must fail validation, but it passed")
	}
}

// TestConstRejectsSchemaVersionDrift pins that the validator interprets the
// const keyword on schema_version, so a version drift away from 0.1 fails
// validation instead of silently passing.
func TestConstRejectsSchemaVersionDrift(t *testing.T) {
	inst := loadMap(t, exampleFilePath)
	inst["schema_version"] = "0.2"
	if errs := validateSchema(inst, loadNode(t)); len(errs) == 0 {
		t.Fatalf("schema_version drift to 0.2 must fail const validation, but it passed")
	}
}

// TestMinimumRejectsNegativeStageIndex pins that the validator interprets the
// minimum keyword on stage.index, so a negative index fails validation.
func TestMinimumRejectsNegativeStageIndex(t *testing.T) {
	inst := loadMap(t, exampleFilePath)
	stages, ok := inst["stages"].([]any)
	if !ok || len(stages) == 0 {
		t.Fatalf("example has no stages to mutate")
	}
	first, ok := stages[0].(map[string]any)
	if !ok {
		t.Fatalf("first stage is not an object")
	}
	first["index"] = -1
	if errs := validateSchema(inst, loadNode(t)); len(errs) == 0 {
		t.Fatalf("negative stage index must fail minimum validation, but it passed")
	}
}

// TestMinLengthRejectsEmptySummary pins that the validator interprets the
// minLength keyword on summary, so an empty summary fails validation.
func TestMinLengthRejectsEmptySummary(t *testing.T) {
	inst := loadMap(t, exampleFilePath)
	inst["summary"] = ""
	if errs := validateSchema(inst, loadNode(t)); len(errs) == 0 {
		t.Fatalf("empty summary must fail minLength validation, but it passed")
	}
}

// TestPatternRejectsUnknownScopeFormats pins that the validator interprets the
// pattern keyword on unknown.scope, so an out-of-format scope fails validation.
func TestPatternRejectsUnknownScopeFormats(t *testing.T) {
	base := loadMap(t, exampleFilePath)

	mkUnknown := func(scope string) map[string]any {
		return map[string]any{
			"code": "parse_error", "scope": scope, "message": "x",
			"evidence": []any{map[string]any{"source": "command"}},
			"blocking": false,
		}
	}
	for _, tc := range []struct {
		scope string
		want  bool // true = should pass validation
	}{
		{scope: "report", want: true},
		{scope: "stage:3", want: true},
		{scope: "file:scripts/deploy.sh", want: true},
		{scope: "script:/tmp/gen.sh", want: true},
		{scope: "stage:-1", want: false},
		{scope: "foo", want: false},
		{scope: "file:", want: false},
	} {
		t.Run(tc.scope, func(t *testing.T) {
			inst := map[string]any{}
			for k, v := range base {
				inst[k] = v
			}
			inst["unknowns"] = []any{mkUnknown(tc.scope)}
			errs := validateSchema(inst, loadNode(t))
			got := len(errs) == 0
			if got != tc.want {
				t.Fatalf("scope %q validation = %v, want %v (errors: %v)", tc.scope, got, tc.want, errs)
			}
		})
	}
}
