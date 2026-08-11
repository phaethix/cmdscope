package ir

// EffectKind classifies the kind of impact an effect represents.
type EffectKind string

const (
	EffectRead          EffectKind = "read"
	EffectWrite         EffectKind = "write"
	EffectDelete        EffectKind = "delete"
	EffectNetwork       EffectKind = "network"
	EffectProcess       EffectKind = "process"
	EffectPrivilege     EffectKind = "privilege"
	EffectExecuteRemote EffectKind = "execute_remote"
	EffectInstall       EffectKind = "install"
)

// Certainty is how sure the analyzer is that an effect occurs. It is orthogonal
// to ConditionKind: certainty is confidence, not whether the stage is gated.
type Certainty string

const (
	Certain          Certainty = "certain"
	Conditional      Certainty = "conditional"
	Possible         Certainty = "possible"
	CertaintyUnknown Certainty = "unknown"
)

// ConditionKind is the stage gate under which an effect runs (always /
// on_success / on_failure), distinct from Certainty.
type ConditionKind string

const (
	ConditionAlways    ConditionKind = "always"
	ConditionOnSuccess ConditionKind = "on_success"
	ConditionOnFailure ConditionKind = "on_failure"
)

// Provenance is which analysis layer supplied an effect's semantics.
// The set is intentionally larger than EvidenceSource (includes "inferred");
// shared wire strings such as "command" across the two types are deliberate.
type Provenance string

const (
	FromCommand       Provenance = "command"
	FromWorkspaceFile Provenance = "workspace_file"
	FromScript        Provenance = "script"
	Inferred          Provenance = "inferred"
	FromCallerContext Provenance = "caller_context"
)

// Coverage is how much of the command surface was walked. It is independent of
// Completeness, which answers whether the emitted result is whole or partial.
type Coverage string

const (
	CoverageComplete Coverage = "complete"
	CoveragePartial  Coverage = "partial"
	CoverageMinimal  Coverage = "minimal"
)

// Completeness is whether the emitted result is whole or partial — not how
// broadly the analyzer walked the command (see Coverage).
type Completeness string

const (
	CompletenessComplete Completeness = "complete"
	CompletenessPartial  Completeness = "partial"
	CompletenessUnknown  Completeness = "unknown"
)

// EvidenceSource is where a piece of evidence is stored. Unlike Provenance it
// has no "inferred" value: evidence always points at concrete text or context.
type EvidenceSource string

const (
	EvidenceCommand       EvidenceSource = "command"
	EvidenceWorkspaceFile EvidenceSource = "workspace_file"
	EvidenceScript        EvidenceSource = "script"
	EvidenceCallerContext EvidenceSource = "caller_context"
)

// UnknownCode is a stable wire identifier for an analysis uncertainty.
// String values are part of the public contract and must not be renamed lightly.
type UnknownCode string

const (
	UnknownUnsupportedSyntax      UnknownCode = "unsupported_syntax"
	UnknownUnsupportedCommand     UnknownCode = "unsupported_command"
	UnknownContextMissing         UnknownCode = "context_missing"
	UnknownScriptNotProvided      UnknownCode = "script_not_provided"
	UnknownScriptDynamicPath      UnknownCode = "script_dynamic_path"
	UnknownEnvMissing             UnknownCode = "env_missing"
	UnknownGlobRuntimeDependent   UnknownCode = "glob_runtime_dependent"
	UnknownCommandSubstitution    UnknownCode = "command_substitution"
	UnknownRemoteContent          UnknownCode = "remote_content"
	UnknownInterpreterDynamicCode UnknownCode = "interpreter_dynamic_code"
	UnknownPlatformDependent      UnknownCode = "platform_dependent"
	UnknownParseError             UnknownCode = "parse_error"
	UnknownInputTooLarge          UnknownCode = "input_too_large"
	UnknownExpansionLimit         UnknownCode = "expansion_limit"
	UnknownAnalysisTimeout        UnknownCode = "analysis_timeout"
	UnknownExpansionCycle         UnknownCode = "expansion_cycle"
)

// Flag is a factual label attached to a report, not a risk conclusion.
// Wire strings are part of the public contract and must not be renamed lightly.
type Flag string

const (
	FlagDestructive     Flag = "destructive"
	FlagExternalNetwork Flag = "external_network"
	FlagPrivilegeChange Flag = "privilege_change"
	FlagOpaqueScript    Flag = "opaque_script"
	FlagRemoteContent   Flag = "remote_content_executed"
	FlagContextMissing  Flag = "context_missing"
	FlagUnsupported     Flag = "unsupported"
	FlagAnalysisTimeout Flag = "analysis_timeout"
)
