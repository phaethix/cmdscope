package analyzer_test

import (
	"context"
	"slices"
	"testing"

	"github.com/phaethix/runmark/internal/analyzer"
	"github.com/phaethix/runmark/internal/ir"
	"github.com/stretchr/testify/require"
)

// TestAnalyzeWiredExtractors pins the end-to-end presence of the extractors
// that were previously unit-tested in isolation but never called by the
// pipeline: standalone curl/wget URLs, npm/pnpm install, and sudo/chmod.
func TestAnalyzeWiredExtractors(t *testing.T) {
	cases := []struct {
		name        string
		command     string
		wantKind    ir.EffectKind
		wantRaw     string
		wantCertain ir.Certainty
	}{
		{
			name:        "curl url network",
			command:     "curl -fsSL https://example.com/a.txt",
			wantKind:    ir.EffectNetwork,
			wantRaw:     "https://example.com/a.txt",
			wantCertain: ir.Certain,
		},
		{
			name:        "wget url network",
			command:     "wget https://example.com/a.tar.gz",
			wantKind:    ir.EffectNetwork,
			wantRaw:     "https://example.com/a.tar.gz",
			wantCertain: ir.Certain,
		},
		{
			name:        "npm install",
			command:     "npm install left-pad",
			wantKind:    ir.EffectInstall,
			wantRaw:     "left-pad",
			wantCertain: ir.Certain,
		},
		{
			name:        "pnpm install",
			command:     "pnpm install",
			wantKind:    ir.EffectInstall,
			wantRaw:     ".",
			wantCertain: ir.Certain,
		},
		{
			name:        "sudo privilege",
			command:     "sudo -u root ls",
			wantKind:    ir.EffectPrivilege,
			wantRaw:     "sudo",
			wantCertain: ir.Certain,
		},
		{
			name:        "chmod metadata write",
			command:     "chmod +x deploy.sh",
			wantKind:    ir.EffectPrivilege,
			wantRaw:     "deploy.sh",
			wantCertain: ir.Certain,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report, err := analyzer.Analyze(context.Background(), ir.AnalyzeRequest{
				Command: tc.command,
				Context: &ir.AnalysisContext{CWD: workspaceCWD},
			})
			require.NoError(t, err)
			require.NoError(t, ir.ValidateReport(report))

			var matched *ir.Effect
			for i := range report.Stages {
				for j := range report.Stages[i].Effects {
					ef := &report.Stages[i].Effects[j]
					if ef.Kind == tc.wantKind && ef.RawTarget == tc.wantRaw {
						matched = ef
						break
					}
				}
			}
			require.NotNil(t, matched, "effects = %+v", report.Stages[0].Effects)
			require.Equal(t, tc.wantCertain, matched.Certainty)
			require.NotEmpty(t, matched.Evidence)
		})
	}
}

// TestAnalyzeWrapperPrefixStripping verifies sudo/doas/env wrappers do not
// hide the inner command from path extractors, and that nested wrappers do
// not duplicate facts.
func TestAnalyzeWrapperPrefixStripping(t *testing.T) {
	t.Run("sudo rm deletes target", func(t *testing.T) {
		report, err := analyzer.Analyze(context.Background(), ir.AnalyzeRequest{
			Command: "sudo rm -rf build",
			Context: &ir.AnalysisContext{CWD: workspaceCWD},
		})
		require.NoError(t, err)
		require.NoError(t, ir.ValidateReport(report))
		del := findEffect(t, report, ir.EffectDelete, "build")
		require.Equal(t, "logical://workspace/build", del.Target)
		require.NotNil(t, findEffect(t, report, ir.EffectPrivilege, "sudo"))
	})

	t.Run("sudo curl keeps network", func(t *testing.T) {
		report, err := analyzer.Analyze(context.Background(), ir.AnalyzeRequest{
			Command: "sudo curl https://example.com/install.sh",
			Context: &ir.AnalysisContext{CWD: workspaceCWD},
		})
		require.NoError(t, err)
		findEffect(t, report, ir.EffectNetwork, "https://example.com/install.sh")
	})

	t.Run("env assignment prefix strips to inner command", func(t *testing.T) {
		report, err := analyzer.Analyze(context.Background(), ir.AnalyzeRequest{
			Command: "env FOO=bar rm -rf dist",
			Context: &ir.AnalysisContext{CWD: workspaceCWD},
		})
		require.NoError(t, err)
		del := findEffect(t, report, ir.EffectDelete, "dist")
		require.Equal(t, "logical://workspace/dist", del.Target)
	})

	t.Run("env -i with unset strips flags and assignments", func(t *testing.T) {
		report, err := analyzer.Analyze(context.Background(), ir.AnalyzeRequest{
			Command: "env -i -u FOO BAR=baz cat secret.txt",
			Context: &ir.AnalysisContext{CWD: workspaceCWD},
		})
		require.NoError(t, err)
		findEffect(t, report, ir.EffectRead, "secret.txt")
	})

	t.Run("nested wrappers do not duplicate effects", func(t *testing.T) {
		report, err := analyzer.Analyze(context.Background(), ir.AnalyzeRequest{
			Command: "sudo env FOO=1 rm -rf build",
			Context: &ir.AnalysisContext{CWD: workspaceCWD},
		})
		require.NoError(t, err)
		require.NoError(t, ir.ValidateReport(report))
		count := 0
		for _, st := range report.Stages {
			for _, ef := range st.Effects {
				if ef.Kind == ir.EffectDelete && ef.RawTarget == "build" {
					count++
				}
			}
		}
		require.Equal(t, 1, count, "wrapper layers must not duplicate the same delete fact")
	})

	t.Run("wrapper redirect extracted once", func(t *testing.T) {
		report, err := analyzer.Analyze(context.Background(), ir.AnalyzeRequest{
			Command: "sudo sh -c 'echo hi' > out.txt",
			Context: &ir.AnalysisContext{CWD: workspaceCWD},
		})
		require.NoError(t, err)
		count := 0
		for _, st := range report.Stages {
			for _, ef := range st.Effects {
				if ef.Kind == ir.EffectWrite && ef.RawTarget == "out.txt" {
					count++
				}
			}
		}
		require.Equal(t, 1, count, "redirect already extracted on the wrapper command")
	})

	t.Run("env standalone has no inner command", func(t *testing.T) {
		report, err := analyzer.Analyze(context.Background(), ir.AnalyzeRequest{
			Command: "env FOO=bar",
			Context: &ir.AnalysisContext{CWD: workspaceCWD},
		})
		require.NoError(t, err)
		require.NoError(t, ir.ValidateReport(report))
	})
}

// TestAnalyzeInstallRegistryNetworkPossible pins that install keeps registry
// contact at possible certainty — we never resolve a registry during analysis,
// so certain would overstate the fact.
func TestAnalyzeInstallRegistryNetworkPossible(t *testing.T) {
	report, err := analyzer.Analyze(context.Background(), ir.AnalyzeRequest{
		Command: "npm install react",
		Context: &ir.AnalysisContext{CWD: workspaceCWD},
	})
	require.NoError(t, err)
	net := findEffect(t, report, ir.EffectNetwork, "registry")
	require.Equal(t, ir.Possible, net.Certainty)
}

func findEffect(t *testing.T, report ir.ImpactReport, kind ir.EffectKind, raw string) ir.Effect {
	t.Helper()
	for _, st := range report.Stages {
		for _, ef := range st.Effects {
			if ef.Kind == kind && ef.RawTarget == raw {
				return ef
			}
		}
	}
	t.Fatalf("no %s effect with raw target %q in report: stages=%+v", kind, raw, report.Stages)
	return ir.Effect{}
}

func hasReportFlag(report ir.ImpactReport, flag ir.Flag) bool {
	return slices.Contains(report.Flags, flag)
}

// TestAnalyzeP1Extractors pins that the command-family extractors are wired
// into the real pipeline (not only unit-tested in isolation): git write/
// delete/destructive, the write family (tee/touch/truncate/ln/sed), find
// -delete, xargs, tar -C, and npx package install.
func TestAnalyzeFamilyExtractorWiring(t *testing.T) {
	cases := []struct {
		name    string
		command string
		want    []struct {
			kind ir.EffectKind
			raw  string
		}
		wantFlag ir.Flag
		wantUnk  ir.UnknownCode
		noUnk    ir.UnknownCode
	}{
		{
			name:    "git rm deletes file",
			command: "git rm file.txt",
			want: []struct {
				kind ir.EffectKind
				raw  string
			}{{ir.EffectDelete, "file.txt"}},
		},
		{
			name:    "tee writes file",
			command: "tee out.txt",
			want: []struct {
				kind ir.EffectKind
				raw  string
			}{{ir.EffectWrite, "out.txt"}},
		},
		{
			name:    "sed -i read+write file",
			command: "sed -i 's/a/b/' file.txt",
			want: []struct {
				kind ir.EffectKind
				raw  string
			}{
				{ir.EffectRead, "file.txt"},
				{ir.EffectWrite, "file.txt"},
			},
		},
		{
			name:    "ln -s reads target writes link",
			command: "ln -s src link",
			want: []struct {
				kind ir.EffectKind
				raw  string
			}{
				{ir.EffectRead, "src"},
				{ir.EffectWrite, "link"},
			},
		},
		{
			name:    "touch writes file",
			command: "touch f.txt",
			want: []struct {
				kind ir.EffectKind
				raw  string
			}{{ir.EffectWrite, "f.txt"}},
		},
		{
			name:    "truncate writes file",
			command: "truncate -s 0 f.txt",
			want: []struct {
				kind ir.EffectKind
				raw  string
			}{{ir.EffectWrite, "f.txt"}},
		},
		{
			name:    "tar extract writes -C dir",
			command: "tar -xzf a.tar.gz -C /dest",
			want: []struct {
				kind ir.EffectKind
				raw  string
			}{{ir.EffectWrite, "/dest"}},
		},
		{
			name:    "find -delete deletes start point",
			command: "find . -name '*.txt' -delete",
			want: []struct {
				kind ir.EffectKind
				raw  string
			}{{ir.EffectDelete, "."}},
			wantUnk: ir.UnknownGlobRuntimeDependent,
		},
		{
			name:    "xargs process with runtime-dependent unknown",
			command: "xargs rm",
			want: []struct {
				kind ir.EffectKind
				raw  string
			}{{ir.EffectProcess, "xargs"}},
			wantUnk: ir.UnknownEffectsRuntimeDependent,
		},
		{
			name:    "npx installs package",
			command: "npx create-react-app myapp",
			want: []struct {
				kind ir.EffectKind
				raw  string
			}{{ir.EffectInstall, "create-react-app"}},
		},
		{
			name:     "git push --force is destructive not unsupported",
			command:  "git push --force origin main",
			wantFlag: ir.FlagDestructive,
			noUnk:    ir.UnknownUnsupportedCommand,
			want:     nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report, err := analyzer.Analyze(context.Background(), ir.AnalyzeRequest{
				Command: tc.command,
				Context: &ir.AnalysisContext{CWD: workspaceCWD},
			})
			require.NoError(t, err)
			require.NoError(t, ir.ValidateReport(report))

			for _, w := range tc.want {
				ef := findEffect(t, report, w.kind, w.raw)
				require.NotEmpty(t, ef.Evidence, "effect %s/%s should carry evidence", w.kind, w.raw)
			}
			if tc.wantFlag != "" {
				require.True(t, hasReportFlag(report, tc.wantFlag), "expected flag %s", tc.wantFlag)
			}
			if tc.wantUnk != "" {
				_, ok := findUnknown(report.Unknowns, tc.wantUnk)
				require.True(t, ok, "expected unknown %s", tc.wantUnk)
			}
			if tc.noUnk != "" {
				_, ok := findUnknown(report.Unknowns, tc.noUnk)
				require.False(t, ok, "did not expect unknown %s", tc.noUnk)
			}
		})
	}
}

// TestAnalyzeUnsupportedCommandDowngradesCoverage pins the honesty rule: a
// command we have no extractor for must not leave the report claiming
// complete coverage.
func TestAnalyzeUnsupportedCommandDowngradesCoverage(t *testing.T) {
	report, err := analyzer.Analyze(context.Background(), ir.AnalyzeRequest{
		Command: "docker system prune -af",
		Context: &ir.AnalysisContext{CWD: workspaceCWD},
	})
	require.NoError(t, err)
	require.NoError(t, ir.ValidateReport(report))
	require.Equal(t, ir.CoveragePartial, report.Analysis.Coverage)
	_, ok := findUnknown(report.Unknowns, ir.UnknownUnsupportedCommand)
	require.True(t, ok)

	known, err := analyzer.Analyze(context.Background(), ir.AnalyzeRequest{
		Command: "rm -rf build",
		Context: &ir.AnalysisContext{CWD: workspaceCWD},
	})
	require.NoError(t, err)
	require.Equal(t, ir.CoverageComplete, known.Analysis.Coverage)
}

// TestAnalyzeFdRedirectNoWrite pins that 2>&1 fd duplication must not produce
// a write fact — the target names a descriptor, not a file.
func TestAnalyzeFdRedirectNoWrite(t *testing.T) {
	report, err := analyzer.Analyze(context.Background(), ir.AnalyzeRequest{
		Command: "echo hi 2>&1",
		Context: &ir.AnalysisContext{CWD: workspaceCWD},
	})
	require.NoError(t, err)
	require.NoError(t, ir.ValidateReport(report))
	var writes []ir.Effect
	for _, st := range report.Stages {
		for _, ef := range st.Effects {
			if ef.Kind == ir.EffectWrite {
				writes = append(writes, ef)
			}
		}
	}
	require.Empty(t, writes)
}

// TestAnalyzeFdRedirectWriteCertainty pins the redirect-operator certainty
// convention end to end: overwrite is certain, append is conditional, and the
// both-streams forms behave like their single-stream counterparts.
func TestAnalyzeFdRedirectWriteCertainty(t *testing.T) {
	cases := []struct {
		name     string
		command  string
		wantRaw  string
		wantCert ir.Certainty
	}{
		{name: "stderr overwrite", command: "echo hi 2> err.txt", wantRaw: "err.txt", wantCert: ir.Certain},
		{name: "stderr append", command: "echo hi 2>> err.txt", wantRaw: "err.txt", wantCert: ir.Conditional},
		{name: "both streams overwrite", command: "echo x &> all.log", wantRaw: "all.log", wantCert: ir.Certain},
		{name: "both streams append", command: "echo x &>> all.log", wantRaw: "all.log", wantCert: ir.Conditional},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report, err := analyzer.Analyze(context.Background(), ir.AnalyzeRequest{
				Command: tc.command,
				Context: &ir.AnalysisContext{CWD: workspaceCWD},
			})
			require.NoError(t, err)
			require.NoError(t, ir.ValidateReport(report))
			write := findEffect(t, report, ir.EffectWrite, tc.wantRaw)
			require.Equal(t, tc.wantCert, write.Certainty)
		})
	}
}
