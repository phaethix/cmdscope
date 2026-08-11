package shell

// TokenKind classifies the lexical unit emitted by the lexer.
type TokenKind string

// Token kinds for the L0 surface. Lex also emits |&, <<, and & so later
// stages can classify them as unsupported unknowns rather than silently
// dropping them.
const (
	TokenWord        TokenKind = "word"
	TokenSingleQuote TokenKind = "single_quote"
	TokenDoubleQuote TokenKind = "double_quote"
	TokenEscape      TokenKind = "escape"

	TokenPipe    TokenKind = "pipe"
	TokenPipeAmp TokenKind = "pipe_amp"
	TokenGT      TokenKind = "gt"
	TokenGTGT    TokenKind = "gtgt"
	TokenLT      TokenKind = "lt"
	TokenLTLT    TokenKind = "ltlt"

	TokenAndAnd TokenKind = "and_and"
	TokenOrOr   TokenKind = "or_or"
	TokenSemi   TokenKind = "semi"
	TokenAmp    TokenKind = "amp"

	TokenLParen TokenKind = "lparen"
	TokenRParen TokenKind = "rparen"

	TokenCommandSub TokenKind = "command_sub"
	TokenBacktick   TokenKind = "backtick"

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
