package analyzer_test

import (
	"context"
	"testing"

	"github.com/phaethix/runmark/internal/analyzer"
	"github.com/phaethix/runmark/internal/ir"
	"github.com/stretchr/testify/require"
)

func analyze(t *testing.T, command string, env map[string]string) ir.ImpactReport {
	t.Helper()
	ctx := &ir.AnalysisContext{CWD: workspaceCWD, Env: env}
	report, err := analyzer.Analyze(context.Background(), ir.AnalyzeRequest{Command: command, Context: ctx})
	require.NoError(t, err)
	require.NoError(t, ir.ValidateReport(report))
	return report
}

// TestAnalyzeCDTracking pins the session-cwd semantics: a successful cd moves
// the normalization base for later stages at the same depth.
func TestAnalyzeCDTracking(t *testing.T) {
	t.Run("cd then relative delete resolves under subdir", func(t *testing.T) {
		report := analyze(t, "cd sub && rm -rf .", nil)
		del := findEffect(t, report, ir.EffectDelete, ".")
		require.Equal(t, "logical://workspace/sub", del.Target)
	})

	t.Run("cd then cat resolves under subdir", func(t *testing.T) {
		report := analyze(t, "cd sub; cat .env", nil)
		read := findEffect(t, report, ir.EffectRead, ".env")
		require.Equal(t, "logical://workspace/sub/.env", read.Target)
	})

	t.Run("cd up escapes the workspace", func(t *testing.T) {
		report := analyze(t, "cd .. && cat .env", nil)
		read := findEffect(t, report, ir.EffectRead, ".env")
		require.Equal(t, "logical://.env", read.Target)
	})

	t.Run("failure branch sees pre-cd cwd", func(t *testing.T) {
		report := analyze(t, "cd sub || cat .env", nil)
		read := findEffect(t, report, ir.EffectRead, ".env")
		require.Equal(t, "logical://workspace/.env", read.Target)
		// Later unconditional stages are on two possible timelines; say so
		// instead of picking one.
		_, ok := findUnknown(report.Unknowns, ir.UnknownCwdRuntimeDependent)
		require.True(t, ok, "unknowns = %v", report.Unknowns)
	})

	t.Run("subshell cd does not leak out", func(t *testing.T) {
		report := analyze(t, "(cd sub && rm -rf build); cat .env", nil)
		del := findEffect(t, report, ir.EffectDelete, "build")
		require.Equal(t, "logical://workspace/sub/build", del.Target)
		read := findEffect(t, report, ir.EffectRead, ".env")
		require.Equal(t, "logical://workspace/.env", read.Target)
	})

	t.Run("dynamic cd keeps request cwd and flags unknown", func(t *testing.T) {
		report := analyze(t, "cd \"$SUB\" && cat .env", nil)
		read := findEffect(t, report, ir.EffectRead, ".env")
		require.Equal(t, "logical://workspace/.env", read.Target)
		_, ok := findUnknown(report.Unknowns, ir.UnknownCwdRuntimeDependent)
		require.True(t, ok)
		_, envMissing := findUnknown(report.Unknowns, ir.UnknownEnvMissing)
		require.True(t, envMissing)
	})

	t.Run("cd with provided env moves cwd", func(t *testing.T) {
		report := analyze(t, "cd \"$SUB\" && cat .env", map[string]string{"SUB": "dist"})
		read := findEffect(t, report, ir.EffectRead, ".env")
		require.Equal(t, "logical://workspace/dist/.env", read.Target)
	})

	t.Run("bare cd is runtime dependent", func(t *testing.T) {
		report := analyze(t, "cd && cat .env", nil)
		read := findEffect(t, report, ir.EffectRead, ".env")
		require.Equal(t, "logical://workspace/.env", read.Target)
		_, ok := findUnknown(report.Unknowns, ir.UnknownCwdRuntimeDependent)
		require.True(t, ok)
	})

	t.Run("cd - toggles OLDPWD at runtime", func(t *testing.T) {
		report := analyze(t, "cd - && cat .env", nil)
		read := findEffect(t, report, ir.EffectRead, ".env")
		require.Equal(t, "logical://workspace/.env", read.Target)
		require.True(t, hasUnknownCode(report.Unknowns, ir.UnknownCwdRuntimeDependent))
	})

	t.Run("cd in a pipeline member does not move cwd", func(t *testing.T) {
		// Pipeline members run in subshells, so a cd there cannot affect
		// later stages.
		report := analyze(t, "cd sub | cat .env && cat README.md", nil)
		read := findEffect(t, report, ir.EffectRead, "README.md")
		require.Equal(t, "logical://workspace/README.md", read.Target)
	})
}

// TestAnalyzeEnvSubstitution pins $VAR path resolution: values substitute only
// when the caller provided them, and derived facts stay traceable.
func TestAnalyzeEnvSubstitution(t *testing.T) {
	t.Run("provided env value substitutes into delete target", func(t *testing.T) {
		report := analyze(t, "rm -rf \"$OUT\"/*.tmp", map[string]string{"OUT": "build"})
		// The lexer splits "$OUT" and *.tmp into separate words; after env
		// substitution the first becomes "build" with a separate glob word.
		del := findEffect(t, report, ir.EffectDelete, "build")
		require.Equal(t, "logical://workspace/build", del.Target)
		require.Equal(t, ir.FromCallerContext, del.Provenance)
		require.False(t, hasUnknownCode(report.Unknowns, ir.UnknownEnvMissing))
		require.True(t, hasUnknownCode(report.Unknowns, ir.UnknownGlobRuntimeDependent))
	})

	t.Run("missing env keeps env_missing unknown", func(t *testing.T) {
		report := analyze(t, "rm -rf \"$OUT\"/*.tmp", nil)
		require.True(t, hasUnknownCode(report.Unknowns, ir.UnknownEnvMissing))
	})

	t.Run("braced form substitutes too", func(t *testing.T) {
		report := analyze(t, "cat ${CFG_FILE}", map[string]string{"CFG_FILE": "config/prod.env"})
		read := findEffect(t, report, ir.EffectRead, "config/prod.env")
		require.Equal(t, "logical://workspace/config/prod.env", read.Target)
	})

	t.Run("redirect target substitutes", func(t *testing.T) {
		report := analyze(t, "echo hi > \"$OUT\"", map[string]string{"OUT": "out/log.txt"})
		write := findEffect(t, report, ir.EffectWrite, "out/log.txt")
		require.Equal(t, "logical://workspace/out/log.txt", write.Target)
		require.Equal(t, ir.FromCallerContext, write.Provenance)
	})
}

// TestAnalyzeTildeUnknown pins that home-relative paths stay visible as
// runtime-dependent instead of being invented.
func TestAnalyzeTildeUnknown(t *testing.T) {
	report := analyze(t, "cat ~/.ssh/id_rsa", nil)
	require.True(t, hasUnknownCode(report.Unknowns, ir.UnknownTildeRuntimeDependent))
	read := findEffect(t, report, ir.EffectRead, "~/.ssh/id_rsa")
	require.Equal(t, "logical://workspace/~/.ssh/id_rsa", read.Target)
}

func hasUnknownCode(unknowns []ir.Unknown, code ir.UnknownCode) bool {
	_, ok := findUnknown(unknowns, code)
	return ok
}
