package shell

// TokenKind classifies the lexical unit emitted by the lexer.
type TokenKind string

// Lexical token kinds recognized by the lexer. They cover the L0 syntax set
// described in the architecture document: words, quoting, escapes, pipelines,
// redirects, boolean control operators, grouping, command substitution,
// backticks and the background marker.
const (
	// TokenWord is a run of unquoted, unescaped literal characters.
	TokenWord TokenKind = "word"
	// TokenSingleQuote is a single-quoted string including the enclosing
	// quotes, e.g. 'a b'.
	TokenSingleQuote TokenKind = "single_quote"
	// TokenDoubleQuote is a double-quoted string including the enclosing
	// quotes, e.g. "a b".
	TokenDoubleQuote TokenKind = "double_quote"
	// TokenEscape is a backslash followed by the escaped character, e.g. \x.
	TokenEscape TokenKind = "escape"

	// TokenPipe is the | pipeline operator.
	TokenPipe TokenKind = "pipe"
	// TokenPipeAmp is the |& pipe-and-stderr operator.
	TokenPipeAmp TokenKind = "pipe_amp"
	// TokenGT is the > output redirect.
	TokenGT TokenKind = "gt"
	// TokenGTGT is the >> append redirect.
	TokenGTGT TokenKind = "gtgt"
	// TokenLT is the < input redirect.
	TokenLT TokenKind = "lt"
	// TokenLTLT is the << here-document marker.
	TokenLTLT TokenKind = "ltlt"

	// TokenAndAnd is the && boolean AND.
	TokenAndAnd TokenKind = "and_and"
	// TokenOrOr is the || boolean OR.
	TokenOrOr TokenKind = "or_or"
	// TokenSemi is the ; command separator.
	TokenSemi TokenKind = "semi"
	// TokenAmp is the & background marker.
	TokenAmp TokenKind = "amp"

	// TokenLParen is the opening subshell grouping.
	TokenLParen TokenKind = "lparen"
	// TokenRParen is the closing subshell grouping.
	TokenRParen TokenKind = "rparen"

	// TokenCommandSub is a $(...) command substitution including its braces.
	TokenCommandSub TokenKind = "command_sub"
	// TokenBacktick is a `...` command substitution including its backticks.
	TokenBacktick TokenKind = "backtick"

	// TokenEOF marks the end of the input and is always the final token.
	TokenEOF TokenKind = "eof"
)

// Token is a lexical unit with its byte span into the original input.
// Start and End are UTF-8 byte offsets over the half-open interval [Start, End).
type Token struct {
	Kind  TokenKind
	Text  string
	Start int
	End   int
}
