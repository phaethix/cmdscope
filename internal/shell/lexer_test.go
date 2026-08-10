package shell

import (
	"strings"
	"testing"
)

// lexTokens is a thin helper that drops the trailing EOF token so assertions
// focus on the meaningful lexemes produced by Lex.
func lexTokens(t *testing.T, input string) []Token {
	t.Helper()
	toks, err := Lex(input)
	if err != nil {
		t.Fatalf("Lex(%q) returned unexpected error: %v", input, err)
	}
	if n := len(toks); n == 0 || toks[n-1].Kind != TokenEOF {
		t.Fatalf("Lex(%q) must end with an EOF token, got %d tokens", input, n)
	}
	return toks[:len(toks)-1]
}

func assertTokens(t *testing.T, input string, want []Token) {
	t.Helper()
	got := lexTokens(t, input)
	if len(got) != len(want) {
		t.Errorf("Lex(%q) got %d tokens, want %d (got=%v want=%v)", input, len(got), len(want), got, want)
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Lex(%q) token[%d] = %+v, want %+v", input, i, got[i], want[i])
		}
	}
}

func TestLexerSimpleWord(t *testing.T) {
	assertTokens(t, "echo hi", []Token{
		{Kind: TokenWord, Text: "echo", Start: 0, End: 4},
		{Kind: TokenWord, Text: "hi", Start: 5, End: 7},
	})
}

func TestLexerSkippsInternalWhitespace(t *testing.T) {
	assertTokens(t, "  a   b  ", []Token{
		{Kind: TokenWord, Text: "a", Start: 2, End: 3},
		{Kind: TokenWord, Text: "b", Start: 6, End: 7},
	})
}

func TestLexerSingleQuote(t *testing.T) {
	assertTokens(t, "echo 'a b'", []Token{
		{Kind: TokenWord, Text: "echo", Start: 0, End: 4},
		{Kind: TokenSingleQuote, Text: "'a b'", Start: 5, End: 10},
	})
}

func TestLexerDoubleQuote(t *testing.T) {
	assertTokens(t, "echo \"a b\"", []Token{
		{Kind: TokenWord, Text: "echo", Start: 0, End: 4},
		{Kind: TokenDoubleQuote, Text: "\"a b\"", Start: 5, End: 10},
	})
}

func TestLexerEscapeMaskingWhitespace(t *testing.T) {
	// `\ ` keeps the escaped space inside one escape token; the following
	// character is a separate word.
	assertTokens(t, "a\\ b", []Token{
		{Kind: TokenWord, Text: "a", Start: 0, End: 1},
		{Kind: TokenEscape, Text: "\\ ", Start: 1, End: 3},
		{Kind: TokenWord, Text: "b", Start: 3, End: 4},
	})
}

func TestLexerOperators(t *testing.T) {
	assertTokens(t, "a && b || c ; d | e", []Token{
		{Kind: TokenWord, Text: "a", Start: 0, End: 1},
		{Kind: TokenAndAnd, Text: "&&", Start: 2, End: 4},
		{Kind: TokenWord, Text: "b", Start: 5, End: 6},
		{Kind: TokenOrOr, Text: "||", Start: 7, End: 9},
		{Kind: TokenWord, Text: "c", Start: 10, End: 11},
		{Kind: TokenSemi, Text: ";", Start: 12, End: 13},
		{Kind: TokenWord, Text: "d", Start: 14, End: 15},
		{Kind: TokenPipe, Text: "|", Start: 16, End: 17},
		{Kind: TokenWord, Text: "e", Start: 18, End: 19},
	})
}

func TestLexerRedirectOperators(t *testing.T) {
	assertTokens(t, "a>>b", []Token{
		{Kind: TokenWord, Text: "a", Start: 0, End: 1},
		{Kind: TokenGTGT, Text: ">>", Start: 1, End: 3},
		{Kind: TokenWord, Text: "b", Start: 3, End: 4},
	})
	assertTokens(t, "a> b", []Token{
		{Kind: TokenWord, Text: "a", Start: 0, End: 1},
		{Kind: TokenGT, Text: ">", Start: 1, End: 2},
		{Kind: TokenWord, Text: "b", Start: 3, End: 4},
	})
	assertTokens(t, "a<< b", []Token{
		{Kind: TokenWord, Text: "a", Start: 0, End: 1},
		{Kind: TokenLTLT, Text: "<<", Start: 1, End: 3},
		{Kind: TokenWord, Text: "b", Start: 4, End: 5},
	})
	assertTokens(t, "<in", []Token{
		{Kind: TokenLT, Text: "<", Start: 0, End: 1},
		{Kind: TokenWord, Text: "in", Start: 1, End: 3},
	})
}

func TestLexerPipeAndAmp(t *testing.T) {
	assertTokens(t, "a |& b", []Token{
		{Kind: TokenWord, Text: "a", Start: 0, End: 1},
		{Kind: TokenPipeAmp, Text: "|&", Start: 2, End: 4},
		{Kind: TokenWord, Text: "b", Start: 5, End: 6},
	})
	assertTokens(t, "cmd &", []Token{
		{Kind: TokenWord, Text: "cmd", Start: 0, End: 3},
		{Kind: TokenAmp, Text: "&", Start: 4, End: 5},
	})
}

func TestLexerParens(t *testing.T) {
	assertTokens(t, "(a)", []Token{
		{Kind: TokenLParen, Text: "(", Start: 0, End: 1},
		{Kind: TokenWord, Text: "a", Start: 1, End: 2},
		{Kind: TokenRParen, Text: ")", Start: 2, End: 3},
	})
}

func TestLexerCommandSubstitution(t *testing.T) {
	assertTokens(t, "echo $(ls -la)", []Token{
		{Kind: TokenWord, Text: "echo", Start: 0, End: 4},
		{Kind: TokenCommandSub, Text: "$(ls -la)", Start: 5, End: 14},
	})
}

func TestLexerCommandSubstitutionNested(t *testing.T) {
	// The lexer must match the outermost closing parenthesis for $(...).
	// "$(echo $(pwd))" is 14 bytes; the whole span must be captured.
	assertTokens(t, "$(echo $(pwd))", []Token{
		{Kind: TokenCommandSub, Text: "$(echo $(pwd))", Start: 0, End: 14},
	})
}

func TestLexerBackticks(t *testing.T) {
	assertTokens(t, "echo `pwd`", []Token{
		{Kind: TokenWord, Text: "echo", Start: 0, End: 4},
		{Kind: TokenBacktick, Text: "`pwd`", Start: 5, End: 10},
	})
}

// TestLexerUTF8ByteSpan verifies that Start/End are byte offsets, not rune
// counts, for non-ASCII input.
func TestLexerUTF8ByteSpan(t *testing.T) {
	input := "嗯 \\:ë"
	toks := lexTokens(t, input)
	if len(toks) != 3 {
		t.Fatalf("Lex(%q) got %d tokens, want 3 (%v)", input, len(toks), toks)
	}
	// "嗯" occupies 3 bytes; a space follows at byte 3; the escape "\:" spans
	// bytes 4-6; "ë" occupies 2 bytes (bytes 6-8). The input is 8 bytes total.
	if toks[0] != (Token{Kind: TokenWord, Text: "嗯", Start: 0, End: 3}) {
		t.Errorf("first token = %+v, want 嗯 @ [0,3)", toks[0])
	}
	if toks[1] != (Token{Kind: TokenEscape, Text: "\\:", Start: 4, End: 6}) {
		t.Errorf("second token = %+v, want escape \\: @ [4,6)", toks[1])
	}
	if toks[2] != (Token{Kind: TokenWord, Text: "ë", Start: 6, End: 8}) {
		t.Errorf("third token = %+v, want word ë @ [6,8)", toks[2])
	}
}

func TestLexerUnterminatedSingleQuote(t *testing.T) {
	_, err := Lex("echo 'abc")
	if err == nil {
		t.Fatal("Lex(\"echo 'abc\") should return an error for unterminated single quote")
	}
	if !strings.Contains(err.Error(), "unterminated") {
		t.Errorf("unterminated error should mention unterminated, got %q", err)
	}
}

func TestLexerUnterminatedDoubleQuote(t *testing.T) {
	_, err := Lex("echo \"abc")
	if err == nil {
		t.Fatal("Lex(\"echo \\\"abc\") should return an error for unterminated double quote")
	}
}

func TestLexerUnterminatedCommandSub(t *testing.T) {
	if _, err := Lex("echo $(abc"); err == nil {
		t.Fatal("Lex(\"echo $(abc\") should return an error for unterminated command substitution")
	}
}

func TestLexerUnterminatedBacktick(t *testing.T) {
	if _, err := Lex("echo `abc"); err == nil {
		t.Fatal("Lex(\"echo `abc\") should return an error for unterminated backtick")
	}
}

func TestLexerDoesNotPanicOnEmptyInput(t *testing.T) {
	toks := lexTokens(t, "")
	if len(toks) != 0 {
		t.Errorf("Lex(\"\") got %d tokens, want none", len(toks))
	}
}
