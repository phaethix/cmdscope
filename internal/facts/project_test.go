package facts_test

import (
	"encoding/json"
	"testing"

	"github.com/phaethix/runmark/internal/facts"
	"github.com/phaethix/runmark/internal/ir"
	"github.com/stretchr/testify/require"
)

func TestProjectTouchesDedupeAndSort(t *testing.T) {
	report := baseReport("logical://workspace")
	cond := ir.Condition{Kind: ir.ConditionAlways}
	report.Stages[0].Effects = []ir.Effect{
		pathEffect(ir.EffectRead, "b.txt", "logical://workspace/b.txt", cond),
		pathEffect(ir.EffectRead, "a.txt", "logical://workspace/a.txt", cond),
		pathEffect(ir.EffectRead, "a.txt", "logical://workspace/a.txt", cond),
		pathEffect(ir.EffectWrite, "out", "logical://workspace/out", cond),
		pathEffect(ir.EffectDelete, "tmp", "logical://workspace/tmp", cond),
		pathEffect(ir.EffectProcess, "rm", "rm", cond),
	}

	got := facts.Project(report)
	require.NoError(t, facts.Validate(got))
	require.Equal(t, []string{"logical://workspace/a.txt", "logical://workspace/b.txt"}, got.Touches.Read)
	require.Equal(t, []string{"logical://workspace/out"}, got.Touches.Write)
	require.Equal(t, []string{"logical://workspace/tmp"}, got.Touches.Delete)
	require.True(t, got.Boundary.Destructive)
	require.False(t, got.Boundary.ExternalNetwork)
}

func TestProjectOutsideWorkspace(t *testing.T) {
	report := baseReport("logical://workspace")
	cond := ir.Condition{Kind: ir.ConditionAlways}
	report.Stages[0].Effects = []ir.Effect{
		pathEffect(ir.EffectRead, "/etc/passwd", "/etc/passwd", cond),
	}
	got := facts.Project(report)
	require.True(t, got.Boundary.OutsideWorkspace)

	report2 := baseReport("logical://workspace")
	report2.Stages[0].Effects = []ir.Effect{
		pathEffect(ir.EffectRead, "a", "logical://other/a", cond),
	}
	require.True(t, facts.Project(report2).Boundary.OutsideWorkspace)

	report3 := baseReport("logical://workspace")
	report3.Stages[0].Effects = []ir.Effect{
		pathEffect(ir.EffectRead, "a", "logical://workspace/a", cond),
	}
	require.False(t, facts.Project(report3).Boundary.OutsideWorkspace)
}

func TestProjectSensitivePath(t *testing.T) {
	report := baseReport("logical://workspace")
	cond := ir.Condition{Kind: ir.ConditionAlways}
	report.Stages[0].Effects = []ir.Effect{
		pathEffect(ir.EffectRead, ".env", "logical://workspace/.env", cond),
	}
	require.True(t, facts.Project(report).Boundary.SensitivePath)
}

func TestProjectNetworkAndRemote(t *testing.T) {
	report := baseReport("logical://workspace")
	cond := ir.Condition{Kind: ir.ConditionAlways}
	report.Stages[0].Effects = []ir.Effect{
		pathEffect(ir.EffectNetwork, "https://x", "https://x", cond),
		pathEffect(ir.EffectExecuteRemote, "https://x", "https://x", cond),
	}
	got := facts.Project(report)
	require.True(t, got.Boundary.ExternalNetwork)
	require.Empty(t, got.Touches.Read)
	require.Empty(t, got.Touches.Write)
	require.Empty(t, got.Touches.Delete)
}

func TestProjectScriptEntryAndEvidence(t *testing.T) {
	report := baseReport("logical://workspace")
	cond := ir.Condition{Kind: ir.ConditionAlways}
	ef := pathEffect(ir.EffectDelete, "dist", "logical://workspace/dist", cond)
	ef.Provenance = ir.FromScript
	ef.Evidence = []ir.Evidence{{
		Source:  ir.EvidenceWorkspaceFile,
		Path:    "package.json",
		Field:   "scripts.build",
		Snippet: "rm -rf dist",
	}}
	ef.ID = ir.EffectID(ir.SchemaVersion, ef)
	report.Stages[0].Effects = []ir.Effect{ef}

	got := facts.Project(report)
	require.Len(t, got.Scripts, 1)
	require.Equal(t, "npm", got.Scripts[0].Tool)
	require.Equal(t, "build", got.Scripts[0].Name)
	require.Equal(t, "package.json", got.Scripts[0].Source)
	require.NotEmpty(t, got.Evidence)
	require.Equal(t, string(ir.EvidenceWorkspaceFile), got.Evidence[0].Source)
}

func TestProjectBlockingUnknownOpaque(t *testing.T) {
	report := baseReport("logical://workspace")
	report.Stages[0].Effects = []ir.Effect{}
	report.Unknowns = []ir.Unknown{{
		Code:     ir.UnknownRemoteContent,
		Scope:    "stage:0",
		Message:  "remote content is not statically knowable",
		Evidence: []ir.Evidence{{Source: ir.EvidenceCommand, Snippet: "https://x"}},
		Blocking: true,
	}}
	report.Flags = []ir.Flag{ir.FlagRemoteContent, ir.FlagExternalNetwork}

	got := facts.Project(report)
	require.True(t, got.Unknown)
	require.True(t, got.Boundary.OpaqueScript)
	require.True(t, got.Boundary.ExternalNetwork)
	require.Equal(t, []string{string(ir.UnknownRemoteContent)}, got.UnknownReasons)
}

func TestProjectByteStable(t *testing.T) {
	report := baseReport("logical://workspace")
	cond := ir.Condition{Kind: ir.ConditionAlways}
	report.Stages[0].Effects = []ir.Effect{
		pathEffect(ir.EffectWrite, "b", "logical://workspace/b", cond),
		pathEffect(ir.EffectWrite, "a", "logical://workspace/a", cond),
	}
	a, err := json.Marshal(facts.Project(report))
	require.NoError(t, err)
	b, err := json.Marshal(facts.Project(report))
	require.NoError(t, err)
	require.Equal(t, string(a), string(b))
}

func baseReport(cwd string) ir.ImpactReport {
	return ir.ImpactReport{
		SchemaVersion: ir.SchemaVersion,
		Command:       "fixture",
		CWD:           cwd,
		Analysis: ir.AnalysisMeta{
			Coverage:     ir.CoverageComplete,
			Completeness: ir.CompletenessComplete,
			Limits:       []string{},
			Parser:       "shell",
		},
		Stages: []ir.Stage{{
			Index:        0,
			Command:      "fixture",
			Condition:    ir.Condition{Kind: ir.ConditionAlways},
			Completeness: ir.CompletenessComplete,
			Effects:      []ir.Effect{},
		}},
		Unknowns: []ir.Unknown{},
		Flags:    []ir.Flag{},
	}
}

func pathEffect(kind ir.EffectKind, raw, target string, cond ir.Condition) ir.Effect {
	ef := ir.Effect{
		Kind:       kind,
		RawTarget:  raw,
		Target:     target,
		Stage:      0,
		Certainty:  ir.Certain,
		Provenance: ir.FromCommand,
		Condition:  cond,
		Evidence:   []ir.Evidence{{Source: ir.EvidenceCommand, Snippet: raw}},
	}
	ef.ID = ir.EffectID(ir.SchemaVersion, ef)
	return ef
}
