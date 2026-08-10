package ir

// EffectKind classifies the kind of impact an effect represents.
type EffectKind string

// Effect kinds emitted by the analyzer.
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

// Certainty expresses how sure the analyzer is that the effect occurs.
type Certainty string

// Certainty levels, from definite to unknown.
const (
	Certain          Certainty = "certain"
	Conditional      Certainty = "conditional"
	Possible         Certainty = "possible"
	CertaintyUnknown Certainty = "unknown"
)

// ConditionKind is the stage condition under which an effect runs.
type ConditionKind string

// Stage condition kinds.
const (
	ConditionAlways    ConditionKind = "always"
	ConditionOnSuccess ConditionKind = "on_success"
	ConditionOnFailure ConditionKind = "on_failure"
)

// Provenance describes which layer an effect's semantics come from.
type Provenance string

// Provenance sources.
const (
	FromCommand       Provenance = "command"
	FromWorkspaceFile Provenance = "workspace_file"
	FromScript        Provenance = "script"
	Inferred          Provenance = "inferred"
	FromCallerContext Provenance = "caller_context"
)

// Coverage reports how much of the command surface was analyzed.
type Coverage string

// Coverage levels.
const (
	CoverageComplete Coverage = "complete"
	CoveragePartial  Coverage = "partial"
	CoverageMinimal  Coverage = "minimal"
)

// Completeness reports whether the result is complete or partial.
type Completeness string

// Completeness levels.
const (
	CompletenessComplete Completeness = "complete"
	CompletenessPartial  Completeness = "partial"
	CompletenessUnknown  Completeness = "unknown"
)

// EvidenceSource is where a piece of evidence is stored.
type EvidenceSource string

// Evidence sources.
const (
	EvidenceCommand       EvidenceSource = "command"
	EvidenceWorkspaceFile EvidenceSource = "workspace_file"
	EvidenceScript        EvidenceSource = "script"
	EvidenceCallerContext EvidenceSource = "caller_context"
)

// UnknownCode is a stable identifier for an analysis uncertainty.
type UnknownCode string

// Unknown codes emitted by the analyzer.
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
type Flag string

// Flags attached to reports.
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
