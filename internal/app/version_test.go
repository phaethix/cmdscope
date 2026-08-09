package app

import (
	"bytes"
	"testing"
)

func TestVersionIsFixed(t *testing.T) {
	const want = "0.1.0"
	if got := Version; got != want {
		t.Fatalf("Version = %q, want %q", got, want)
	}
}

func TestPrintVersion(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintVersion(&buf); err != nil {
		t.Fatalf("PrintVersion returned error: %v", err)
	}
	const want = "cmdscope 0.1.0\n"
	if got := buf.String(); got != want {
		t.Fatalf("PrintVersion output = %q, want %q", got, want)
	}
}
