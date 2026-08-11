package shell

import "testing"

// parseFail lexes input, runs Parse, and asserts that Parse returns a
// non-nil *ParseError. It returns the error's concrete fields for assertion.
func parseFail(t *testing.T, input string) *ParseError {
	t.Helper()
	toks, err := Lex(input)
	if err != nil {
		t.Fatalf("Lex(%q) returned unexpected lexer error: %v", input, err)
	}
	_, perr := Parse(toks)
	if perr == nil {
		t.Fatalf("Parse(%q) succeeded, want a parse error", input)
	}
	pe, ok := perr.(*ParseError)
	if !ok {
		t.Fatalf("Parse(%q) returned %T (%v), want *ParseError", input, perr, perr)
	}
	return pe
}

func TestParseUnsupportedBackgroundAmp(t *testing.T) {
	pe := parseFail(t, "cmd &")
	if pe.Kind != KindUnsupported {
		t.Errorf("Kind = %q, want %q", pe.Kind, KindUnsupported)
	}
	if pe.Token != "&" || pe.At != 4 {
		t.Errorf("unsupported token = %q @ %d, want \"&\" @ 4", pe.Token, pe.At)
	}
}

func TestParseUnsupportedPipeAmp(t *testing.T) {
	pe := parseFail(t, "a |& b")
	if pe.Kind != KindUnsupported {
		t.Errorf("Kind = %q, want %q", pe.Kind, KindUnsupported)
	}
	if pe.Token != "|&" {
		t.Errorf("Token = %q, want |&", pe.Token)
	}
}

func TestParseUnsupportedHereDoc(t *testing.T) {
	pe := parseFail(t, "cat << EOF")
	if pe.Kind != KindUnsupported {
		t.Errorf("Kind = %q, want %q", pe.Kind, KindUnsupported)
	}
	if pe.Token != "<<" {
		t.Errorf("Token = %q, want <<", pe.Token)
	}
}

func TestParseIsUnsupportedSyntaxHelper(t *testing.T) {
	toks, err := Lex("cmd &")
	if err != nil {
		t.Fatalf("Lex error: %v", err)
	}
	_, perr := Parse(toks)
	if !IsUnsupportedSyntax(perr) {
		t.Fatalf("IsUnsupportedSyntax(&) = false, want true (err=%v)", perr)
	}
}

func TestParseStructuralRedirectMissingTarget(t *testing.T) {
	pe := parseFail(t, "echo >")
	if pe.Kind != KindStructural {
		t.Errorf("Kind = %q, want %q", pe.Kind, KindStructural)
	}
	if pe.At != 5 {
		t.Errorf("At = %d, want 5 (> operator offset)", pe.At)
	}
}

func TestParseStructuralUnmatchedParen(t *testing.T) {
	pe := parseFail(t, "(a")
	if pe.Kind != KindStructural {
		t.Errorf("Kind = %q, want %q", pe.Kind, KindStructural)
	}
	if !stringsContains(pe.Msg, "closing") {
		t.Errorf("Msg = %q, want mention of missing closing paren", pe.Msg)
	}
}

func TestParseStructuralEmptyCommandAfterOperator(t *testing.T) {
	pe := parseFail(t, "a &&")
	if pe.Kind != KindStructural {
		t.Errorf("Kind = %q, want %q", pe.Kind, KindStructural)
	}
}

func TestParseStructuralIsNotUnsupported(t *testing.T) {
	toks, err := Lex("a &&")
	if err != nil {
		t.Fatalf("Lex error: %v", err)
	}
	_, perr := Parse(toks)
	if IsUnsupportedSyntax(perr) {
		t.Fatalf("IsUnsupportedSyntax(structural) = true, want false (err=%v)", perr)
	}
	if _, ok := perr.(*ParseError); !ok {
		t.Fatalf("structural error should still be a *ParseError, got %T", perr)
	}
}

func TestParseNilStreamReportsStructural(t *testing.T) {
	_, perr := Parse(nil)
	pe, ok := perr.(*ParseError)
	if !ok {
		t.Fatalf("Parse(nil) returned %T, want *ParseError", perr)
	}
	if pe.Kind != KindStructural {
		t.Errorf("Kind = %q, want %q", pe.Kind, KindStructural)
	}
}

// stringsContains is a tiny substring check that keeps the test free of the
// strings import churn across assertions.
func stringsContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
