package ir

// SchemaVersion is the fixed report contract version for the first release.
const SchemaVersion = "0.1"

// ImpactReport is the structured analysis result for one command.
// Callers must initialize every slice field to an empty slice before
// rendering so it serializes as [] (never null); ValidateReport enforces
// this at runtime. Empty strings in RawTarget/Target only mean
// "no more specific operand".
type ImpactReport struct {
	SchemaVersion string       `json:"schema_version"`
	Command       string       `json:"command"`
	CWD           string       `json:"cwd,omitempty"`
	Analysis      AnalysisMeta `json:"analysis"`
	Stages        []Stage      `json:"stages"`
	Unknowns      []Unknown    `json:"unknowns"`
	Flags         []Flag       `json:"flags"`
	Summary       string       `json:"summary"`
}

// AnalysisMeta carries analysis-wide metadata.
type AnalysisMeta struct {
	Coverage     Coverage     `json:"coverage"`
	Completeness Completeness `json:"completeness"`
	Limits       []string     `json:"limits"`
	Parser       string       `json:"parser"`
}

// Stage is one ordered execution stage of the analyzed command.
type Stage struct {
	Index        int          `json:"index"`
	Command      string       `json:"command"`
	Condition    Condition    `json:"condition"`
	Completeness Completeness `json:"completeness"`
	Effects      []Effect     `json:"effects"`
}

// Effect is one concrete impact extracted from a stage.
type Effect struct {
	ID         string     `json:"id"`
	Kind       EffectKind `json:"kind"`
	RawTarget  string     `json:"raw_target"`
	Target     string     `json:"target"`
	Stage      int        `json:"stage"`
	Certainty  Certainty  `json:"certainty"`
	Provenance Provenance `json:"provenance"`
	Condition  Condition  `json:"condition"`
	Evidence   []Evidence `json:"evidence"`
}

// Condition is the stage condition that gates an effect.
// DependsOn is always serialized, even when zero.
type Condition struct {
	Kind      ConditionKind `json:"kind"`
	DependsOn int           `json:"depends_on"`
}

// Evidence links an effect or unknown back to its source text.
// StartByte/EndByte are half-open byte offsets [start,end) in the original
// UTF-8 command or context file; nil means "cannot be located".
type Evidence struct {
	Source    EvidenceSource `json:"source"`
	Path      string         `json:"path,omitempty"`
	Field     string         `json:"field,omitempty"`
	StartByte *int           `json:"start_byte,omitempty"`
	EndByte   *int           `json:"end_byte,omitempty"`
	Snippet   string         `json:"snippet,omitempty"`
}

// Unknown is a documented analysis uncertainty.
// Scope uses the fixed formats: report, stage:<index>, file:<path>, script:<path>.
type Unknown struct {
	Code     UnknownCode `json:"code"`
	Scope    string      `json:"scope"`
	Message  string      `json:"message"`
	Evidence []Evidence  `json:"evidence"`
	Blocking bool        `json:"blocking"`
}
