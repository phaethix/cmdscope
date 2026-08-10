package ir_test

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"testing"

	"github.com/phaethix/cmdscope/internal/ir"
)

// This suite pins the runtime contract enforced by ir.ValidateReport,
// independent of JSON Schema. Schema validity is necessary but not
// sufficient: reports reaching renderers must also satisfy the invariants
// described in the architecture (Stage.Index continuity, condition deep
// equality, evidence existence/spans, enum membership, effect ID stability).

// mustInt returns a pointer to v for instrumenting evidence spans.
func mustInt(v int) *int { return &v }

// validCondition builds an always condition with depends_on 0.
func validCondition() ir.Condition {
	return ir.Condition{Kind: ir.ConditionAlways, DependsOn: 0}
}

// validEffectID computes the expected stable identifier for an effect using
// the documented formula:
//
//	sha256(schema_version + stage + kind + raw_target + target + condition_canonical + provenance)
//
// where condition_canonical is the canonical JSON of the Condition
// ({"kind":...,"depends_on":...}). Keeping this in the test keeps the test
// independent from any production helper so it can detect drift.
func validEffectID(schemaVersion string, stage int, kind, rawTarget, target, provenance string, cond ir.Condition) string {
	canon := fmt.Sprintf(`{"kind":%q,"depends_on":%d}`, string(cond.Kind), cond.DependsOn)
	payload := schemaVersion + strconv.Itoa(stage) + kind + rawTarget + target + canon + provenance
	sum := sha256.Sum256([]byte(payload))
	return fmt.Sprintf("sha256:%x", sum[:])
}

// validEffect builds an effect wired to the owning stage and a recomputed ID.
func validEffect(stageIndex int, stageCond ir.Condition) ir.Effect {
	return ir.Effect{
		ID:         validEffectID("0.1", stageIndex, string(ir.EffectWrite), "output.txt", "/repo/output.txt", string(ir.FromCommand), stageCond),
		Kind:       ir.EffectWrite,
		RawTarget:  "output.txt",
		Target:     "/repo/output.txt",
		Stage:      stageIndex,
		Certainty:  ir.Certain,
		Provenance: ir.FromCommand,
		Condition:  stageCond,
		Evidence:   []ir.Evidence{{Source: ir.EvidenceCommand, StartByte: mustInt(5), EndByte: mustInt(11), Snippet: "> output.txt"}},
	}
}

// validStage builds a complete stage with the given index and one effect.
func validStage(index int) ir.Stage {
	c := validCondition()
	if index > 0 {
		c = ir.Condition{Kind: ir.ConditionOnSuccess, DependsOn: index - 1}
	}
	return ir.Stage{
		Index:        index,
		Command:      "echo hi > output.txt",
		Condition:    c,
		Completeness: ir.CompletenessComplete,
		Effects:      []ir.Effect{validEffect(index, c)},
	}
}

// validReport returns a report that must pass ValidateReport. Tests that want
// to expose a single invariant violation clone it and mutate one field.
func validReport() ir.ImpactReport {
	return ir.ImpactReport{
		SchemaVersion: "0.1",
		Command:       "echo hi > output.txt",
		CWD:           "/repo",
		Analysis: ir.AnalysisMeta{
			Coverage:     ir.CoverageComplete,
			Completeness: ir.CompletenessComplete,
			Limits:       []string{},
			Parser:       "shlex",
		},
		Stages:   []ir.Stage{validStage(0)},
		Unknowns: []ir.Unknown{},
		Flags:    []ir.Flag{},
		Summary:  "Single inherent stage writes to /repo/output.txt.",
	}
}

// TestValidateReportValid accepts a well-formed minimal report.
func TestValidateReportValid(t *testing.T) {
	if err := ir.ValidateReport(validReport()); err != nil {
		t.Fatalf("ValidateReport(valid) = %v, want nil", err)
	}
}

// TestValidateReportRejectsInvalidEnums rejects out-of-schema enum values.
func TestValidateReportRejectsInvalidEnums(t *testing.T) {
	setup := func(mut func(r *ir.ImpactReport)) ir.ImpactReport {
		r := validReport()
		mut(&r)
		return r
	}
	tests := []struct {
		name   string
		mutate func(r *ir.ImpactReport)
	}{
		{"coverage", func(r *ir.ImpactReport) { r.Analysis.Coverage = ir.Coverage("bogus") }},
		{"completeness", func(r *ir.ImpactReport) { r.Analysis.Completeness = ir.Completeness("bogus") }},
		{"effectKind", func(r *ir.ImpactReport) { r.Stages[0].Effects[0].Kind = "bogus" }},
		{"certainty", func(r *ir.ImpactReport) { r.Stages[0].Effects[0].Certainty = "bogus" }},
		{"provenance", func(r *ir.ImpactReport) { r.Stages[0].Effects[0].Provenance = "bogus" }},
		{"conditionKind", func(r *ir.ImpactReport) { r.Stages[0].Condition.Kind = "bogus" }},
		{"evidenceSource", func(r *ir.ImpactReport) { r.Stages[0].Effects[0].Evidence[0].Source = "bogus" }},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if err := ir.ValidateReport(setup(tc.mutate)); err == nil {
				t.Fatalf("ValidateReport with %s = nil, want error", tc.name)
			}
		})
	}
}

// TestValidateReportRejectsNilArrays rejects nil slice fields that would
// serialize as null instead of [].
func TestValidateReportRejectsNilArrays(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(r *ir.ImpactReport)
	}{
		{"stages", func(r *ir.ImpactReport) { r.Stages = nil }},
		{"unknowns", func(r *ir.ImpactReport) { r.Unknowns = nil }},
		{"flags", func(r *ir.ImpactReport) { r.Flags = nil }},
		{"limits", func(r *ir.ImpactReport) { r.Analysis.Limits = nil }},
		{"effects", func(r *ir.ImpactReport) { r.Stages[0].Effects = nil }},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			r := validReport()
			tc.mutate(&r)
			if err := ir.ValidateReport(r); err == nil {
				t.Fatalf("ValidateReport with nil %s = nil, want error", tc.name)
			}
		})
	}
}

// TestValidateReportRejectsStageIndexContiguity rejects non-zero-based,
// duplicate, or non-contiguous stage indexes.
func TestValidateReportRejectsStageIndexContiguity(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(r *ir.ImpactReport)
	}{
		{"startsAtOne", func(r *ir.ImpactReport) { r.Stages[0].Index = 1 }},
		{"duplicate", func(r *ir.ImpactReport) { r.Stages = []ir.Stage{validStage(0), validStage(0)} }},
		{"gap", func(r *ir.ImpactReport) { r.Stages = []ir.Stage{validStage(0), validStage(2)} }},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			r := validReport()
			tc.mutate(&r)
			if err := ir.ValidateReport(r); err == nil {
				t.Fatalf("ValidateReport with %s = nil, want error", tc.name)
			}
		})
	}
}

// TestValidateReportRejectsEffectStageMismatch rejects Effect.Stage that does
// not equal its owning Stage.Index.
func TestValidateReportRejectsEffectStageMismatch(t *testing.T) {
	r := validReport()
	r.Stages[0].Effects[0].Stage = 5
	if err := ir.ValidateReport(r); err == nil {
		t.Fatal("ValidateReport with effect stage mismatch = nil, want error")
	}
}

// TestValidateReportRejectsConditionDrift rejects Stage/Effect condition
// divergence.
func TestValidateReportRejectsConditionDrift(t *testing.T) {
	r := validReport()
	r.Stages[0].Effects[0].Condition = ir.Condition{Kind: ir.ConditionOnSuccess, DependsOn: 0}
	if err := ir.ValidateReport(r); err == nil {
		t.Fatal("ValidateReport with drifted effect condition = nil, want error")
	}
}

// TestValidateReportRejectsIllegalDependsOn applies the depends_on legality
// rules for non-always conditions.
func TestValidateReportRejectsIllegalDependsOn(t *testing.T) {
	// on_success referencing a stage index equal to/above the stage count.
	r := validReport()
	r.Stages[0].Condition = ir.Condition{Kind: ir.ConditionOnSuccess, DependsOn: 9}
	r.Stages[0].Effects[0].Condition = ir.Condition{Kind: ir.ConditionOnSuccess, DependsOn: 9}
	r.Stages[0].Effects[0].ID = validEffectID("0.1", 0, string(ir.EffectWrite), "output.txt", "/repo/output.txt", string(ir.FromCommand), r.Stages[0].Effects[0].Condition)
	if err := ir.ValidateReport(r); err == nil {
		t.Fatal("ValidateReport with on_success depends_on out of range = nil, want error")
	}

	// on_failure referencing its own stage (not strictly smaller).
	r = validReport()
	r.Stages = []ir.Stage{validStage(0), validStage(1)}
	r.Stages[1].Condition = ir.Condition{Kind: ir.ConditionOnFailure, DependsOn: 1}
	r.Stages[1].Effects[0].Condition = ir.Condition{Kind: ir.ConditionOnFailure, DependsOn: 1}
	r.Stages[1].Effects[0].ID = validEffectID("0.1", 1, string(ir.EffectWrite), "output.txt", "/repo/output.txt", string(ir.FromCommand), r.Stages[1].Effects[0].Condition)
	if err := ir.ValidateReport(r); err == nil {
		t.Fatal("ValidateReport with on_failure depends_on == own index = nil, want error")
	}
}

// TestValidateReportAcceptsLegalDependsOn accepts a chain of stages where
// non-root stages reference the previous stage.
func TestValidateReportAcceptsLegalDependsOn(t *testing.T) {
	r := validReport()
	r.Stages = []ir.Stage{validStage(0), validStage(1)}
	if err := ir.ValidateReport(r); err != nil {
		t.Fatalf("ValidateReport with legal depends_on chain = %v, want nil", err)
	}
}

// TestValidateReportRejectsMissingEvidence rejects effects with zero evidence.
func TestValidateReportRejectsMissingEvidence(t *testing.T) {
	r := validReport()
	r.Stages[0].Effects[0].Evidence = []ir.Evidence{}
	if err := ir.ValidateReport(r); err == nil {
		t.Fatal("ValidateReport with no effect evidence = nil, want error")
	}
}

// TestValidateReportRejectsUnpairedSpan rejects partially-present spans.
func TestValidateReportRejectsUnpairedSpan(t *testing.T) {
	// start present, end nil.
	r := validReport()
	r.Stages[0].Effects[0].Evidence[0].EndByte = nil
	if err := ir.ValidateReport(r); err == nil {
		t.Fatal("ValidateReport with unpaired start span = nil, want error")
	}

	// end present, start nil.
	r = validReport()
	r.Stages[0].Effects[0].Evidence[0].StartByte = nil
	if err := ir.ValidateReport(r); err == nil {
		t.Fatal("ValidateReport with unpaired end span = nil, want error")
	}
}

// TestValidateReportRejectsBadSpanOrder rejects spans where start >= end.
func TestValidateReportRejectsBadSpanOrder(t *testing.T) {
	r := validReport()
	r.Stages[0].Effects[0].Evidence[0].StartByte = mustInt(5)
	r.Stages[0].Effects[0].Evidence[0].EndByte = mustInt(3)
	if err := ir.ValidateReport(r); err == nil {
		t.Fatal("ValidateReport with start>=end span = nil, want error")
	}

	r = validReport()
	r.Stages[0].Effects[0].Evidence[0].StartByte = mustInt(4)
	r.Stages[0].Effects[0].Evidence[0].EndByte = mustInt(4)
	if err := ir.ValidateReport(r); err == nil {
		t.Fatal("ValidateReport with zero-length span = nil, want error")
	}
}

// TestValidateReportAcceptsNilNilSpan accepts a both-nil span when source and
// snippet are present.
func TestValidateReportAcceptsNilNilSpan(t *testing.T) {
	r := validReport()
	e := r.Stages[0].Effects[0].Evidence[0]
	e.StartByte = nil
	e.EndByte = nil
	e.Snippet = "> output.txt"
	r.Stages[0].Effects[0].Evidence[0] = e
	if err := ir.ValidateReport(r); err != nil {
		t.Fatalf("ValidateReport with nil/nil span = %v, want nil", err)
	}
}

// TestValidateReportUnknownScope accepts documented scope formats and rejects
// malformed ones.
func TestValidateReportUnknownScope(t *testing.T) {
	for _, scope := range []string{"report", "stage:0", "file:scripts/x.sh", "script:/tmp/x.sh"} {
		r := validReport()
		r.Unknowns = []ir.Unknown{{Code: ir.UnknownParseError, Scope: scope, Message: "x", Evidence: nil, Blocking: false}}
		if err := ir.ValidateReport(r); err != nil {
			t.Fatalf("ValidateReport with scope %q = %v, want nil", scope, err)
		}
	}

	r := validReport()
	r.Unknowns = []ir.Unknown{{Code: ir.UnknownParseError, Scope: "foo", Message: "x", Evidence: nil, Blocking: false}}
	if err := ir.ValidateReport(r); err == nil {
		t.Fatal("ValidateReport with invalid unknown scope 'foo' = nil, want error")
	}
}

// TestValidateReportRejectsStageScopeOutOfRange ensures a stage-scoped unknown
// references an existing stage.
func TestValidateReportRejectsStageScopeOutOfRange(t *testing.T) {
	r := validReport()
	r.Unknowns = []ir.Unknown{{Code: ir.UnknownParseError, Scope: "stage:9", Message: "x", Evidence: nil, Blocking: false}}
	if err := ir.ValidateReport(r); err == nil {
		t.Fatal("ValidateReport with stage scope referencing missing stage = nil, want error")
	}
}

// TestValidateReportRejectsMismatchedEffectID rejects an effect whose ID does
// not match the recomputed value.
func TestValidateReportRejectsMismatchedEffectID(t *testing.T) {
	r := validReport()
	r.Stages[0].Effects[0].ID = "sha256:deadbeef"
	if err := ir.ValidateReport(r); err == nil {
		t.Fatal("ValidateReport with mismatched effect ID = nil, want error")
	}
}

// TestValidateReportUndefinedCode rejects an unknown code outside the schema
// enum.
func TestValidateReportRejectsInvalidUnknownCode(t *testing.T) {
	r := validReport()
	r.Unknowns = []ir.Unknown{{Code: ir.UnknownCode("bogus"), Scope: "report", Message: "x", Evidence: nil, Blocking: false}}
	if err := ir.ValidateReport(r); err == nil {
		t.Fatal("ValidateReport with invalid unknown code = nil, want error")
	}
}

// TestValidateReportRejectsInvalidFlag rejects a flag outside the schema enum.
func TestValidateReportRejectsInvalidFlag(t *testing.T) {
	r := validReport()
	r.Flags = []ir.Flag{"bogus"}
	if err := ir.ValidateReport(r); err == nil {
		t.Fatal("ValidateReport with invalid flag = nil, want error")
	}
}
