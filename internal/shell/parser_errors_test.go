package shell

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseFail lexes input, runs Parse, and asserts that Parse returns a
// non-nil *ParseError. It returns the error's concrete fields for assertion.
func parseFail(t *testing.T, input string) *ParseError {
	t.Helper()
	toks, err := Lex(input)
	require.NoError(t, err, "Lex(%q) returned unexpected lexer error", input)
	_, perr := Parse(toks)
	require.Error(t, perr, "Parse(%q) succeeded, want a parse error", input)
	pe, ok := perr.(*ParseError)
	require.True(t, ok, "Parse(%q) returned %T (%v), want *ParseError", input, perr, perr)
	return pe
}

func TestParseUnsupportedBackgroundAmp(t *testing.T) {
	pe := parseFail(t, "cmd &")
	assert.Equal(t, KindUnsupported, pe.Kind)
	assert.Equal(t, "&", pe.Token)
	assert.Equal(t, 4, pe.At)
}

func TestParseUnsupportedPipeAmp(t *testing.T) {
	pe := parseFail(t, "a |& b")
	assert.Equal(t, KindUnsupported, pe.Kind)
	assert.Equal(t, "|&", pe.Token)
}

func TestParseUnsupportedHereDoc(t *testing.T) {
	pe := parseFail(t, "cat << EOF")
	assert.Equal(t, KindUnsupported, pe.Kind)
	assert.Equal(t, "<<", pe.Token)
}

func TestParseIsUnsupportedSyntaxHelper(t *testing.T) {
	toks, err := Lex("cmd &")
	require.NoError(t, err)
	_, perr := Parse(toks)
	require.True(t, IsUnsupportedSyntax(perr), "err=%v", perr)
}

func TestParseStructuralRedirectMissingTarget(t *testing.T) {
	pe := parseFail(t, "echo >")
	assert.Equal(t, KindStructural, pe.Kind)
	assert.Equal(t, 5, pe.At)
}

func TestParseStructuralUnmatchedParen(t *testing.T) {
	pe := parseFail(t, "(a")
	assert.Equal(t, KindStructural, pe.Kind)
	assert.Contains(t, pe.Msg, "closing")
}

func TestParseStructuralEmptyCommandAfterOperator(t *testing.T) {
	pe := parseFail(t, "a &&")
	assert.Equal(t, KindStructural, pe.Kind)
}

func TestParseStructuralIsNotUnsupported(t *testing.T) {
	toks, err := Lex("a &&")
	require.NoError(t, err)
	_, perr := Parse(toks)
	require.False(t, IsUnsupportedSyntax(perr), "err=%v", perr)
	_, ok := perr.(*ParseError)
	require.True(t, ok, "structural error should still be a *ParseError, got %T", perr)
}

func TestParseNilStreamReportsStructural(t *testing.T) {
	_, perr := Parse(nil)
	pe, ok := perr.(*ParseError)
	require.True(t, ok, "Parse(nil) returned %T, want *ParseError", perr)
	assert.Equal(t, KindStructural, pe.Kind)
}
