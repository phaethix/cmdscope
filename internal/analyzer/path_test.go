package analyzer_test

import (
	"testing"

	"github.com/phaethix/cmdscope/internal/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compatibility smoke: analyzer re-exports must keep the prior public path API.
func TestPathNormalizeReexport(t *testing.T) {
	got, flags := analyzer.NormalizeLogicalPath("a", "/ws")
	require.Equal(t, "/ws/a", got)
	assert.Equal(t, analyzer.PathFlags(0), flags)
	require.True(t, (analyzer.PathHasGlob).Has(analyzer.PathHasGlob))
}
