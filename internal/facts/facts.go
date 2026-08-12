package facts

// SchemaVersion marks the experimental touch-facts wire contract.
const SchemaVersion = "0.1-touch-experimental"

// RunmarkFacts is the experimental workspace/script facts surface for hooks.
type RunmarkFacts struct {
	SchemaVersion  string         `json:"schema_version"`
	Touches        TouchSet       `json:"touches"`
	Boundary       BoundaryFacts  `json:"boundary"`
	Scripts        []ScriptEntry  `json:"scripts"`
	Unknown        bool           `json:"unknown"`
	UnknownReasons []string       `json:"unknown_reasons"`
	Evidence       []FactEvidence `json:"evidence"`
}

// TouchSet holds dedupe-ready path lists. Callers must keep slices non-nil.
type TouchSet struct {
	Read   []string `json:"read"`
	Write  []string `json:"write"`
	Delete []string `json:"delete"`
}

// BoundaryFacts are factual boundary bits, not risk scores or sandbox claims.
type BoundaryFacts struct {
	OutsideWorkspace bool `json:"outside_workspace"`
	SensitivePath    bool `json:"sensitive_path"`
	Destructive      bool `json:"destructive"`
	ExternalNetwork  bool `json:"external_network"`
	OpaqueScript     bool `json:"opaque_script"`
}

// ScriptEntry names a workspace/script surface the analysis entered.
type ScriptEntry struct {
	Tool   string `json:"tool"`
	Name   string `json:"name,omitempty"`
	Source string `json:"source,omitempty"`
}

// FactEvidence is a compact evidence row for the facts projection.
type FactEvidence struct {
	Source  string `json:"source"`
	Path    string `json:"path,omitempty"`
	Field   string `json:"field,omitempty"`
	Snippet string `json:"snippet,omitempty"`
}
