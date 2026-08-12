package spike_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phaethix/runmark/internal/spike"
	"github.com/stretchr/testify/require"
)

func TestSpikeCases(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "spike")
	cases, err := spike.Discover(root)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(cases), 3, "S07 seed cases must be present")

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			got, err := spike.RunCase(c)
			require.NoError(t, err, "analyze/project")
			require.Equal(t, c.Expected, got, "facts mismatch for case %s", c.Name)

			body, err := os.ReadFile(filepath.Join(c.Dir, "baseline.md"))
			require.NoError(t, err)
			require.Contains(t, string(body), "## Simple string guard")
			require.Contains(t, string(body), "## RunmarkFacts")
		})
	}
}

func TestLoadCaseRejectsMissingExpected(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "request.json"), []byte(`{"command":"true"}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "context.json"), []byte(`{"cwd":"logical://workspace"}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "baseline.md"), []byte("## Simple string guard\n## RunmarkFacts\n"), 0o600))
	_, err := spike.LoadCase(dir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected-facts.json")
}
