package ir

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ContractViolationErrorCode is the stable code returned by ValidateReport
// when a runtime invariant is violated. Renderers and adapters map it to the
// product-level "internal_contract_violation" signal (CLI exit code 3).
const ContractViolationErrorCode = "internal_contract_violation"

// ContractViolationError is returned by ValidateReport when a report breaks a
// documented runtime invariant. It never panics and leaves the caller free to
// abort rendering before emitting (partial) output.
type ContractViolationError struct {
	Code    string `json:"error_code"`
	Message string `json:"message"`
}

// Error returns the human-readable text form for logs and CLI stderr.
func (e *ContractViolationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// ValidateReport checks the runtime invariants that the JSON Schema alone
// cannot express (architecture §3.2). It is a contract, not a test helper:
// every renderer and adapter must call it before serializing. It returns a
// *ContractViolationError and never panics.
func ValidateReport(report ImpactReport) error {
	if err := validateReportGlobal(report); err != nil {
		return err
	}
	return nil
}

func violation(detail string) error {
	return &ContractViolationError{Code: ContractViolationErrorCode, Message: detail}
}

func validateReportGlobal(report ImpactReport) error {
	if report.Stages == nil {
		return violation("stages must not be nil")
	}
	if report.Unknowns == nil {
		return violation("unknowns must not be nil")
	}
	if report.Flags == nil {
		return violation("flags must not be nil")
	}
	if report.Analysis.Limits == nil {
		return violation("analysis.limits must not be nil")
	}
	if !validCoverage(report.Analysis.Coverage) {
		return violation("invalid analysis.coverage " + strconv.Quote(string(report.Analysis.Coverage)))
	}
	if !validCompleteness(report.Analysis.Completeness) {
		return violation("invalid analysis.completeness " + strconv.Quote(string(report.Analysis.Completeness)))
	}

	// Stage.Index must be exactly 0..n-1, unique and continuous.
	seenIndex := make(map[int]bool)
	for i, st := range report.Stages {
		if st.Index != i {
			return violation(fmt.Sprintf("position %d must carry stage index %d, got %d", i, i, st.Index))
		}
		if seenIndex[st.Index] {
			return violation(fmt.Sprintf("duplicate stage index %d", st.Index))
		}
		seenIndex[st.Index] = true
	}

	for _, st := range report.Stages {
		if err := validateStage(report, st); err != nil {
			return err
		}
	}
	for _, unk := range report.Unknowns {
		if err := validateUnknown(report, unk); err != nil {
			return err
		}
	}
	for _, fl := range report.Flags {
		if !validFlag(fl) {
			return violation("invalid flag value " + strconv.Quote(string(fl)))
		}
	}
	return nil
}

func validateStage(report ImpactReport, st Stage) error {
	if !validCompleteness(st.Completeness) {
		return violation(fmt.Sprintf("stage %d invalid completeness %q", st.Index, st.Completeness))
	}
	if !validConditionKind(st.Condition.Kind) {
		return violation(fmt.Sprintf("stage %d invalid condition.kind %q", st.Index, st.Condition.Kind))
	}
	if err := validateDependsOn(report, st.Index, st.Condition); err != nil {
		return err
	}
	if st.Effects == nil {
		return violation(fmt.Sprintf("stage %d effects must not be nil", st.Index))
	}
	for _, ef := range st.Effects {
		if err := validateEffect(report, st, ef); err != nil {
			return err
		}
	}
	return nil
}

func validateEffect(report ImpactReport, st Stage, ef Effect) error {
	if ef.Stage != st.Index {
		return violation(fmt.Sprintf("effect in stage %d declares stage %d", st.Index, ef.Stage))
	}
	if !validEffectKind(ef.Kind) {
		return violation(fmt.Sprintf("effect %q invalid kind %q", ef.ID, ef.Kind))
	}
	if !validCertainty(ef.Certainty) {
		return violation(fmt.Sprintf("effect %q invalid certainty %q", ef.ID, ef.Certainty))
	}
	if !validProvenance(ef.Provenance) {
		return violation(fmt.Sprintf("effect %q invalid provenance %q", ef.ID, ef.Provenance))
	}
	// Stage.Condition must deep-equal the Effect.Condition.
	if ef.Condition != st.Condition {
		return violation(fmt.Sprintf("effect %q condition drifts from its stage: got %+v want %+v", ef.ID, ef.Condition, st.Condition))
	}
	if len(ef.Evidence) == 0 {
		return violation(fmt.Sprintf("effect %q must have at least one evidence", ef.ID))
	}
	for _, ev := range ef.Evidence {
		if err := validateEvidence(ev); err != nil {
			return violation(fmt.Sprintf("effect %q: %s", ef.ID, err.Error()))
		}
	}
	// Effect ID must equal the recomputation over the normalized inputs.
	want := effectID(report.SchemaVersion, ef)
	if ef.ID != want {
		return violation(fmt.Sprintf("effect ID mismatch: got %q want recomputed %q", ef.ID, want))
	}
	return nil
}

func validateEvidence(ev Evidence) error {
	if !validEvidenceSource(ev.Source) {
		return violation("invalid evidence.source " + strconv.Quote(string(ev.Source)))
	}
	if (ev.StartByte == nil) != (ev.EndByte == nil) {
		return violation("evidence span must be present in pairs")
	}
	if ev.StartByte != nil && ev.EndByte != nil && *ev.StartByte >= *ev.EndByte {
		return violation("evidence span must satisfy start < end")
	}
	return nil
}

func validateUnknown(report ImpactReport, unk Unknown) error {
	if !validUnknownCode(unk.Code) {
		return violation("invalid unknown.code " + strconv.Quote(string(unk.Code)))
	}
	if err := validateScope(report, unk.Scope); err != nil {
		return err
	}
	for _, ev := range unk.Evidence {
		if err := validateEvidence(ev); err != nil {
			return violation(fmt.Sprintf("unknown %q: %s", unk.Scope, err.Error()))
		}
	}
	return nil
}

// validateScope applies the documented fixed scope formats and, for stage:N,
// requires the referenced stage to exist.
func validateScope(report ImpactReport, scope string) error {
	switch {
	case scope == "report":
		return nil
	case strings.HasPrefix(scope, "file:"):
		if len(scope) > len("file:") {
			return nil
		}
		return violation("file scope must carry a path")
	case strings.HasPrefix(scope, "script:"):
		if len(scope) > len("script:") {
			return nil
		}
		return violation("script scope must carry a path")
	case strings.HasPrefix(scope, "stage:"):
		idx, err := strconv.Atoi(scope[len("stage:"):])
		if err != nil || idx < 0 {
			return violation("stage scope must reference a non-negative integer stage index")
		}
		if idx >= len(report.Stages) {
			return violation(fmt.Sprintf("stage scope %q references missing stage", scope))
		}
		return nil
	default:
		return violation("unknown scope must be report, stage:<index>, file:<path> or script:<path>")
	}
}

// validateDependsOn enforces Condition.DependsOn legality:
//   - always => DependsOn must be 0;
//   - on_success / on_failure => DependsOn must reference a smaller existing
//     stage index.
func validateDependsOn(report ImpactReport, stageIndex int, c Condition) error {
	switch c.Kind {
	case ConditionAlways:
		if c.DependsOn != 0 {
			return violation(fmt.Sprintf("stage %d always condition must have depends_on 0, got %d", stageIndex, c.DependsOn))
		}
	case ConditionOnSuccess, ConditionOnFailure:
		if c.DependsOn < 0 || c.DependsOn >= stageIndex || c.DependsOn >= len(report.Stages) {
			return violation(fmt.Sprintf("stage %d condition %s must reference a smaller existing stage index, got %d", stageIndex, c.Kind, c.DependsOn))
		}
	default:
		return violation(fmt.Sprintf("stage %d invalid condition kind %q", stageIndex, c.Kind))
	}
	return nil
}

// effectID recomputes the stable identifier described in the learning guide:
//
//	sha256(schema_version + stage + kind + raw_target + target + condition_canonical + provenance)
//
// condition_canonical is the canonical JSON of the condition
// ({"kind":"...","depends_on":N}) so the recombination is deterministic.
func effectID(schemaVersion string, ef Effect) string {
	canon := fmt.Sprintf(`{"kind":%q,"depends_on":%d}`, string(ef.Condition.Kind), ef.Condition.DependsOn)
	payload := schemaVersion + strconv.Itoa(ef.Stage) + string(ef.Kind) + ef.RawTarget + ef.Target + canon + string(ef.Provenance)
	sum := sha256.Sum256([]byte(payload))
	return "sha256:" + fmt.Sprintf("%x", sum[:])
}

func validEffectKind(k EffectKind) bool         { return enumContains(effectKindSet, string(k)) }
func validCertainty(c Certainty) bool           { return enumContains(certaintySet, string(c)) }
func validConditionKind(k ConditionKind) bool   { return enumContains(conditionKindSet, string(k)) }
func validProvenance(p Provenance) bool         { return enumContains(provenanceSet, string(p)) }
func validCoverage(c Coverage) bool             { return enumContains(coverageSet, string(c)) }
func validCompleteness(c Completeness) bool     { return enumContains(completenessSet, string(c)) }
func validEvidenceSource(s EvidenceSource) bool { return enumContains(evidenceSourceSet, string(s)) }
func validUnknownCode(c UnknownCode) bool       { return enumContains(unknownCodeSet, string(c)) }
func validFlag(f Flag) bool                     { return enumContains(flagSet, string(f)) }

func enumContains(set []string, v string) bool {
	i := sort.SearchStrings(set, v)
	return i < len(set) && set[i] == v
}

func sortedValues(in []string) []string {
	s := append([]string(nil), in...)
	sort.Strings(s)
	return s
}

var (
	coverageSet       = sortedValues([]string{string(CoverageComplete), string(CoveragePartial), string(CoverageMinimal)})
	completenessSet   = sortedValues([]string{string(CompletenessComplete), string(CompletenessPartial), string(CompletenessUnknown)})
	effectKindSet     = sortedValues([]string{string(EffectRead), string(EffectWrite), string(EffectDelete), string(EffectNetwork), string(EffectProcess), string(EffectPrivilege), string(EffectExecuteRemote), string(EffectInstall)})
	certaintySet      = sortedValues([]string{string(Certain), string(Conditional), string(Possible), string(CertaintyUnknown)})
	conditionKindSet  = sortedValues([]string{string(ConditionAlways), string(ConditionOnSuccess), string(ConditionOnFailure)})
	provenanceSet     = sortedValues([]string{string(FromCommand), string(FromWorkspaceFile), string(FromScript), string(Inferred), string(FromCallerContext)})
	evidenceSourceSet = sortedValues([]string{string(EvidenceCommand), string(EvidenceWorkspaceFile), string(EvidenceScript), string(EvidenceCallerContext)})
	unknownCodeSet    = sortedValues([]string{
		string(UnknownUnsupportedSyntax), string(UnknownUnsupportedCommand), string(UnknownContextMissing),
		string(UnknownScriptNotProvided), string(UnknownScriptDynamicPath), string(UnknownEnvMissing),
		string(UnknownGlobRuntimeDependent), string(UnknownCommandSubstitution), string(UnknownRemoteContent),
		string(UnknownInterpreterDynamicCode), string(UnknownPlatformDependent), string(UnknownParseError),
		string(UnknownInputTooLarge), string(UnknownExpansionLimit), string(UnknownAnalysisTimeout),
		string(UnknownExpansionCycle),
	})
	flagSet = sortedValues([]string{
		string(FlagDestructive), string(FlagExternalNetwork), string(FlagPrivilegeChange),
		string(FlagOpaqueScript), string(FlagRemoteContent), string(FlagContextMissing),
		string(FlagUnsupported), string(FlagAnalysisTimeout),
	})
)
