package app

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionIsFixed(t *testing.T) {
	require.Equal(t, "0.1.0", Version)
}

func TestPrintVersion(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, PrintVersion(&buf))
	require.Equal(t, "cmdscope 0.1.0\n", buf.String())
}
