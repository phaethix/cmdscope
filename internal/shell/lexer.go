package shell

import "errors"

// isWordByte reports whether a byte can occur inside a word token. Bytes at or
// above 0x80 are the leading bytes of multi-byte UTF-8 runes and belong to a
// word, which keeps Start/End as UTF-8 byte offsets rather than rune counts.
func isWordByte(c byte) bool {
	if c >= 0x80 {
		return true
	}
	switch c {
	case ' ', '\t', '\n', '\r',
		'|', '&', '>', '<', ';',
		'(', ')', '\'', '"', '\\', '`':
		return false
	}
	return true
}

// Lex tokenizes a shell command into a slice of Token values carrying byte
// spans over the half-open interval [Start, End). The final token is always
// TokenEOF. It never panics: an unterminated quote, command substitution or
// backtick (and a trailing escape) returns the tokens collected so far along
// with an error. The input is never executed.
func Lex(input string) ([]Token, error) {
	toks := make([]Token, 0, 16)
	i, n := 0, len(input)
	for i < n {
		c := input[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '$' && i+1 < n && input[i+1] == '(':
			tok, end, err := scanCommandSub(input, i)
			if err != nil {
				return appendEOF(toks, n), err
			}
			toks = append(toks, tok)
			i = end
		case isWordByte(c):
			start := i
			for i < n && isWordByte(input[i]) {
				i++
			}
			toks = append(toks, Token{Kind: TokenWord, Text: input[start:i], Start: start, End: i})
		default:
			switch c {
			case '\'':
				tok, end, err := scanQuoted(input, i, '\'', false)
				if err != nil {
					return appendEOF(toks, n), err
				}
				toks = append(toks, tok)
				i = end
			case '"':
				tok, end, err := scanQuoted(input, i, '"', true)
				if err != nil {
					return appendEOF(toks, n), err
				}
				toks = append(toks, tok)
				i = end
			case '\\':
				if i+1 >= n {
					return appendEOF(toks, n), errors.New("unterminated escape")
				}
				toks = append(toks, Token{Kind: TokenEscape, Text: input[i : i+2], Start: i, End: i + 2})
				i += 2
			case '`':
				tok, end, err := scanBacktick(input, i)
				if err != nil {
					return appendEOF(toks, n), err
				}
				toks = append(toks, tok)
				i = end
			case '|':
				switch {
				case i+1 < n && input[i+1] == '|':
					toks = append(toks, Token{Kind: TokenOrOr, Text: "||", Start: i, End: i + 2})
					i += 2
				case i+1 < n && input[i+1] == '&':
					toks = append(toks, Token{Kind: TokenPipeAmp, Text: "|&", Start: i, End: i + 2})
					i += 2
				default:
					toks = append(toks, Token{Kind: TokenPipe, Text: "|", Start: i, End: i + 1})
					i++
				}
			case '&':
				if i+1 < n && input[i+1] == '&' {
					toks = append(toks, Token{Kind: TokenAndAnd, Text: "&&", Start: i, End: i + 2})
					i += 2
				} else {
					toks = append(toks, Token{Kind: TokenAmp, Text: "&", Start: i, End: i + 1})
					i++
				}
			case '>':
				if i+1 < n && input[i+1] == '>' {
					toks = append(toks, Token{Kind: TokenGTGT, Text: ">>", Start: i, End: i + 2})
					i += 2
				} else {
					toks = append(toks, Token{Kind: TokenGT, Text: ">", Start: i, End: i + 1})
					i++
				}
			case '<':
				if i+1 < n && input[i+1] == '<' {
					toks = append(toks, Token{Kind: TokenLTLT, Text: "<<", Start: i, End: i + 2})
					i += 2
				} else {
					toks = append(toks, Token{Kind: TokenLT, Text: "<", Start: i, End: i + 1})
					i++
				}
			case ';':
				toks = append(toks, Token{Kind: TokenSemi, Text: ";", Start: i, End: i + 1})
				i++
			case '(':
				toks = append(toks, Token{Kind: TokenLParen, Text: "(", Start: i, End: i + 1})
				i++
			case ')':
				toks = append(toks, Token{Kind: TokenRParen, Text: ")", Start: i, End: i + 1})
				i++
			default:
				// Unreachable for bytes that are neither whitespace, word
				// bytes nor one of the operators above. Guard defensively by
				// consuming a single byte so the loop always progresses.
				toks = append(toks, Token{Kind: TokenWord, Text: input[i : i+1], Start: i, End: i + 1})
				i++
			}
		}
	}
	toks = append(toks, Token{Kind: TokenEOF, Text: "", Start: n, End: n})
	return toks, nil
}

// scanQuoted scans a quoted run starting at the opening quote byte. When
// escapes is true (double quotes) a backslash escapes the following byte;
// otherwise (single quotes) every byte up to the closing quote is literal.
func scanQuoted(input string, start int, quote byte, escapes bool) (Token, int, error) {
	i := start + 1
	n := len(input)
	for i < n {
		if escapes && input[i] == '\\' {
			i += 2
			continue
		}
		if input[i] == quote {
			return Token{
				Kind:  tokenKindForQuote(quote),
				Text:  input[start : i+1],
				Start: start,
				End:   i + 1,
			}, i + 1, nil
		}
		i++
	}
	return Token{}, n, errors.New("unterminated " + quoteName(quote))
}

func tokenKindForQuote(quote byte) TokenKind {
	if quote == '\'' {
		return TokenSingleQuote
	}
	return TokenDoubleQuote
}

func quoteName(quote byte) string {
	if quote == '\'' {
		return "single quote"
	}
	return "double quote"
}

// scanBacktick returns the backtick command substitution token. The next
// unescaped backtick closes it.
func scanBacktick(input string, start int) (Token, int, error) {
	i := start + 1
	n := len(input)
	for i < n {
		if input[i] == '`' {
			return Token{Kind: TokenBacktick, Text: input[start : i+1], Start: start, End: i + 1}, i + 1, nil
		}
		i++
	}
	return Token{}, n, errors.New("unterminated backtick")
}

// scanCommandSub returns the $(...) command substitution token, matching the
// outermost closing parenthesis and respecting nested parentheses.
func scanCommandSub(input string, start int) (Token, int, error) {
	depth := 1
	i := start + 2
	n := len(input)
	for i < n {
		switch input[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return Token{Kind: TokenCommandSub, Text: input[start : i+1], Start: start, End: i + 1}, i + 1, nil
			}
		}
		i++
	}
	return Token{}, n, errors.New("unterminated command substitution")
}

// appendEOF guarantees the token stream always terminates with an EOF token,
// even when an error is reported. The EOF span points at the end of the input.
func appendEOF(toks []Token, end int) []Token {
	return append(toks, Token{Kind: TokenEOF, Text: "", Start: end, End: end})
}
