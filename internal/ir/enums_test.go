package ir_test

import (
	"testing"

	"github.com/phaethix/runmark/internal/ir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// enumValues lists every enum type and its documented constant values.
// Each value must be unique within its own type; cross-type reuse
// (e.g. "command" in Provenance and EvidenceSource) is intentional.
func TestEnumValuesUniqueWithinType(t *testing.T) {
	cases := []struct {
		name   string
		values []string
	}{
		{"EffectKind", []string{
			string(ir.EffectRead),
			string(ir.EffectWrite),
			string(ir.EffectDelete),
			string(ir.EffectNetwork),
			string(ir.EffectProcess),
			string(ir.EffectPrivilege),
			string(ir.EffectExecuteRemote),
			string(ir.EffectInstall),
		}},
		{"Certainty", []string{
			string(ir.Certain),
			string(ir.Conditional),
			string(ir.Possible),
			string(ir.CertaintyUnknown),
		}},
		{"ConditionKind", []string{
			string(ir.ConditionAlways),
			string(ir.ConditionOnSuccess),
			string(ir.ConditionOnFailure),
		}},
		{"Provenance", []string{
			string(ir.FromCommand),
			string(ir.FromWorkspaceFile),
			string(ir.FromScript),
			string(ir.Inferred),
			string(ir.FromCallerContext),
		}},
		{"Coverage", []string{
			string(ir.CoverageComplete),
			string(ir.CoveragePartial),
			string(ir.CoverageMinimal),
		}},
		{"Completeness", []string{
			string(ir.CompletenessComplete),
			string(ir.CompletenessPartial),
			string(ir.CompletenessUnknown),
		}},
		{"EvidenceSource", []string{
			string(ir.EvidenceCommand),
			string(ir.EvidenceWorkspaceFile),
			string(ir.EvidenceScript),
			string(ir.EvidenceCallerContext),
		}},
		{"UnknownCode", []string{
			string(ir.UnknownUnsupportedSyntax),
			string(ir.UnknownUnsupportedCommand),
			string(ir.UnknownContextMissing),
			string(ir.UnknownScriptNotProvided),
			string(ir.UnknownScriptDynamicPath),
			string(ir.UnknownEnvMissing),
			string(ir.UnknownGlobRuntimeDependent),
			string(ir.UnknownCommandSubstitution),
			string(ir.UnknownRemoteContent),
			string(ir.UnknownInterpreterDynamicCode),
			string(ir.UnknownPlatformDependent),
			string(ir.UnknownParseError),
			string(ir.UnknownInputTooLarge),
			string(ir.UnknownExpansionLimit),
			string(ir.UnknownAnalysisTimeout),
			string(ir.UnknownExpansionCycle),
		}},
		{"Flag", []string{
			string(ir.FlagDestructive),
			string(ir.FlagExternalNetwork),
			string(ir.FlagPrivilegeChange),
			string(ir.FlagOpaqueScript),
			string(ir.FlagRemoteContent),
			string(ir.FlagContextMissing),
			string(ir.FlagUnsupported),
			string(ir.FlagAnalysisTimeout),
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.NotEmpty(t, tc.values, "enum has no values")
			seen := make(map[string]struct{}, len(tc.values))
			for _, v := range tc.values {
				// Soft multi-check: report every empty/duplicate value in one run.
				assert.NotEmpty(t, v, "enum %s has an empty value", tc.name)
				_, dup := seen[v]
				assert.False(t, dup, "enum %s has duplicate value %q", tc.name, v)
				seen[v] = struct{}{}
			}
		})
	}
}

// TestEnumConstantExactValues locks the exact wire strings so an accidental
// rename cannot silently change the JSON contract.
func TestEnumConstantExactValues(t *testing.T) {
	exact := map[string]string{
		"EffectRead":          string(ir.EffectRead),
		"EffectWrite":         string(ir.EffectWrite),
		"EffectDelete":        string(ir.EffectDelete),
		"EffectNetwork":       string(ir.EffectNetwork),
		"EffectProcess":       string(ir.EffectProcess),
		"EffectPrivilege":     string(ir.EffectPrivilege),
		"EffectExecuteRemote": string(ir.EffectExecuteRemote),
		"EffectInstall":       string(ir.EffectInstall),

		"Certain":            string(ir.Certain),
		"Conditional":        string(ir.Conditional),
		"Possible":           string(ir.Possible),
		"CertaintyUnknown":   string(ir.CertaintyUnknown),
		"ConditionAlways":    string(ir.ConditionAlways),
		"ConditionOnSuccess": string(ir.ConditionOnSuccess),
		"ConditionOnFailure": string(ir.ConditionOnFailure),

		"FromCommand":          string(ir.FromCommand),
		"FromWorkspaceFile":    string(ir.FromWorkspaceFile),
		"FromScript":           string(ir.FromScript),
		"Inferred":             string(ir.Inferred),
		"FromCallerContext":    string(ir.FromCallerContext),
		"CoverageComplete":     string(ir.CoverageComplete),
		"CoveragePartial":      string(ir.CoveragePartial),
		"CoverageMinimal":      string(ir.CoverageMinimal),
		"CompletenessComplete": string(ir.CompletenessComplete),
		"CompletenessPartial":  string(ir.CompletenessPartial),
		"CompletenessUnknown":  string(ir.CompletenessUnknown),

		"EvidenceCommand":               string(ir.EvidenceCommand),
		"EvidenceWorkspaceFile":         string(ir.EvidenceWorkspaceFile),
		"EvidenceScript":                string(ir.EvidenceScript),
		"EvidenceCallerContext":         string(ir.EvidenceCallerContext),
		"UnknownUnsupportedSyntax":      string(ir.UnknownUnsupportedSyntax),
		"UnknownUnsupportedCommand":     string(ir.UnknownUnsupportedCommand),
		"UnknownContextMissing":         string(ir.UnknownContextMissing),
		"UnknownScriptNotProvided":      string(ir.UnknownScriptNotProvided),
		"UnknownScriptDynamicPath":      string(ir.UnknownScriptDynamicPath),
		"UnknownEnvMissing":             string(ir.UnknownEnvMissing),
		"UnknownGlobRuntimeDependent":   string(ir.UnknownGlobRuntimeDependent),
		"UnknownCommandSubstitution":    string(ir.UnknownCommandSubstitution),
		"UnknownRemoteContent":          string(ir.UnknownRemoteContent),
		"UnknownInterpreterDynamicCode": string(ir.UnknownInterpreterDynamicCode),
		"UnknownPlatformDependent":      string(ir.UnknownPlatformDependent),
		"UnknownParseError":             string(ir.UnknownParseError),
		"UnknownInputTooLarge":          string(ir.UnknownInputTooLarge),
		"UnknownExpansionLimit":         string(ir.UnknownExpansionLimit),
		"UnknownAnalysisTimeout":        string(ir.UnknownAnalysisTimeout),
		"UnknownExpansionCycle":         string(ir.UnknownExpansionCycle),

		"FlagDestructive":     string(ir.FlagDestructive),
		"FlagExternalNetwork": string(ir.FlagExternalNetwork),
		"FlagPrivilegeChange": string(ir.FlagPrivilegeChange),
		"FlagOpaqueScript":    string(ir.FlagOpaqueScript),
		"FlagRemoteContent":   string(ir.FlagRemoteContent),
		"FlagContextMissing":  string(ir.FlagContextMissing),
		"FlagUnsupported":     string(ir.FlagUnsupported),
		"FlagAnalysisTimeout": string(ir.FlagAnalysisTimeout),
	}
	want := map[string]string{
		"EffectRead": "read", "EffectWrite": "write", "EffectDelete": "delete",
		"EffectNetwork": "network", "EffectProcess": "process", "EffectPrivilege": "privilege",
		"EffectExecuteRemote": "execute_remote", "EffectInstall": "install",
		"Certain": "certain", "Conditional": "conditional", "Possible": "possible",
		"CertaintyUnknown": "unknown",
		"ConditionAlways":  "always", "ConditionOnSuccess": "on_success", "ConditionOnFailure": "on_failure",
		"FromCommand": "command", "FromWorkspaceFile": "workspace_file", "FromScript": "script",
		"Inferred": "inferred", "FromCallerContext": "caller_context",
		"CoverageComplete": "complete", "CoveragePartial": "partial", "CoverageMinimal": "minimal",
		"CompletenessComplete": "complete", "CompletenessPartial": "partial", "CompletenessUnknown": "unknown",
		"EvidenceCommand": "command", "EvidenceWorkspaceFile": "workspace_file",
		"EvidenceScript": "script", "EvidenceCallerContext": "caller_context",
		"UnknownUnsupportedSyntax":      "unsupported_syntax",
		"UnknownUnsupportedCommand":     "unsupported_command",
		"UnknownContextMissing":         "context_missing",
		"UnknownScriptNotProvided":      "script_not_provided",
		"UnknownScriptDynamicPath":      "script_dynamic_path",
		"UnknownEnvMissing":             "env_missing",
		"UnknownGlobRuntimeDependent":   "glob_runtime_dependent",
		"UnknownCommandSubstitution":    "command_substitution",
		"UnknownRemoteContent":          "remote_content",
		"UnknownInterpreterDynamicCode": "interpreter_dynamic_code",
		"UnknownPlatformDependent":      "platform_dependent",
		"UnknownParseError":             "parse_error",
		"UnknownInputTooLarge":          "input_too_large",
		"UnknownExpansionLimit":         "expansion_limit",
		"UnknownAnalysisTimeout":        "analysis_timeout",
		"UnknownExpansionCycle":         "expansion_cycle",
		"FlagDestructive":               "destructive", "FlagExternalNetwork": "external_network",
		"FlagPrivilegeChange": "privilege_change", "FlagOpaqueScript": "opaque_script",
		"FlagRemoteContent": "remote_content_executed", "FlagContextMissing": "context_missing",
		"FlagUnsupported": "unsupported", "FlagAnalysisTimeout": "analysis_timeout",
	}
	for name, got := range exact {
		// Soft multi-check: surface every drifted wire string in one run.
		assert.Equal(t, want[name], got, "%s", name)
	}
}
