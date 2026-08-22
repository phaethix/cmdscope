package shell

import "strings"

// Parse turns a lexer token stream (which must end with a TokenEOF) into an
// AST rooted at a Node. Precedence: subshell > pipeline '|' > '&&'/'||' > ';'.
// Nodes carry UTF-8 byte spans. && / || are left-associative.
//
// Parse never panics. A structurally invalid stream, or an unsupported L0-out-
// of-scope construct (background '&', '|&', here-doc '<<'), is reported as an
// error rather than a classified node; recognising those as unsupported
// unknowns is the responsibility of a later stage.
func Parse(tokens []Token) (Node, error) {
	if tokens == nil {
		return nil, &ParseError{Kind: KindStructural, Msg: "empty token stream"}
	}
	p := &parser{toks: tokens}
	node, err := p.parseSequence()
	if err != nil {
		return node, err
	}
	if !p.done() {
		t := p.cur()
		return node, &ParseError{Kind: KindStructural, At: t.Start, Token: t.Text, Msg: "unexpected token"}
	}
	return node, nil
}

// parser holds a token slice (not an io.Reader) so Parse can lookahead and
// attach byte spans without re-lexing.
type parser struct {
	toks []Token
	pos  int
}

func (p *parser) cur() Token          { return p.toks[p.pos] }
func (p *parser) done() bool          { return p.toks[p.pos].Kind == TokenEOF }
func (p *parser) at(k TokenKind) bool { return p.toks[p.pos].Kind == k }

func (p *parser) advance() {
	if p.pos < len(p.toks)-1 {
		p.pos++
	}
}

func (p *parser) parseSequence() (Node, error) {
	var items []Node
	for !p.done() && !p.at(TokenRParen) {
		if p.at(TokenSemi) {
			// Allow a;;b — consecutive ';' are empty statements, as in common shells.
			p.advance()
			continue
		}
		item, err := p.parseAndOr()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
		if p.done() || p.at(TokenRParen) {
			break
		}
		if !p.at(TokenSemi) {
			break
		}
		p.advance()
	}
	if len(items) == 0 {
		return nil, nil
	}
	if len(items) == 1 {
		return items[0], nil
	}
	return &Sequence{
		Items: items,
		Start: nodeStart(items[0]),
		End:   nodeEnd(items[len(items)-1]),
	}, nil
}

func (p *parser) parseAndOr() (Node, error) {
	left, err := p.parsePipeline()
	if err != nil {
		return nil, err
	}
	for p.at(TokenAndAnd) || p.at(TokenOrOr) {
		op := p.cur().Text
		p.advance()
		right, err := p.parsePipeline()
		if err != nil {
			return nil, err
		}
		left = &Binary{
			Op:    op,
			Left:  left,
			Right: right,
			Start: nodeStart(left),
			End:   nodeEnd(right),
		}
	}
	return left, nil
}

func (p *parser) parsePipeline() (Node, error) {
	first, err := p.parseCommand()
	if err != nil {
		return nil, err
	}
	if !p.at(TokenPipe) {
		return first, nil
	}
	cmds := []Node{first}
	for p.at(TokenPipe) {
		p.advance()
		cmd, err := p.parseCommand()
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, cmd)
	}
	return &Pipeline{
		Commands: cmds,
		Start:    nodeStart(cmds[0]),
		End:      nodeEnd(cmds[len(cmds)-1]),
	}, nil
}

func (p *parser) parseCommand() (Node, error) {
	if p.at(TokenLParen) {
		return p.parseSubshell()
	}

	s := &SimpleCommand{
		Assignments: []Assignment{},
		Words:       []Word{},
		Redirects:   []Redirect{},
		Start:       p.cur().Start,
	}
	if p.done() {
		return nil, &ParseError{Kind: KindStructural, At: s.Start, Msg: "cannot parse command at end of input"}
	}

	for {
		if p.done() || p.at(TokenSemi) || p.at(TokenAndAnd) || p.at(TokenOrOr) ||
			p.at(TokenPipe) || p.at(TokenRParen) {
			break
		}
		t := p.cur()
		switch t.Kind {
		case TokenWord, TokenSingleQuote, TokenDoubleQuote, TokenEscape,
			TokenCommandSub, TokenBacktick:
			if len(s.Assignments) == 0 && len(s.Words) == 0 && len(s.Redirects) == 0 {
				if a, ok := makeAssignment(t); ok {
					s.Assignments = append(s.Assignments, a)
					p.advance()
					continue
				}
			}
			s.Words = append(s.Words, Word{Text: t.Text, Start: t.Start, End: t.End})
			p.advance()
		case TokenGT, TokenGTGT, TokenLT:
			op := t.Text
			// A bare-digit word glued to the operator (`2>`, `12>>`) is an fd
			// prefix, not an argument; only byte-adjacency distinguishes it
			// from `echo 2 > file`, where 2 is real argv.
			if last := len(s.Words) - 1; last >= 0 && isDigits(s.Words[last].Text) && s.Words[last].End == t.Start {
				s.Words = s.Words[:last]
			}
			p.advance()
			target, fdDup, ok := p.redirectTarget()
			if !ok {
				return nil, &ParseError{Kind: KindStructural, At: t.Start, Token: op, Msg: "redirect missing target"}
			}
			if fdDup {
				// 2>&1 duplicates a descriptor: the target names an fd, not a
				// file, so no write effect may be derived from it.
				op = ">&"
			}
			s.Redirects = append(s.Redirects, Redirect{
				Operator: op,
				Target:   target,
				Start:    t.Start,
				End:      target.End,
			})
		case TokenAmp:
			// Bash's &>file / &>>file redirects both streams to a file; the
			// & must be glued to the operator or it is a background marker,
			// which stays unsupported.
			next := p.toks[min(p.pos+1, len(p.toks)-1)]
			if (next.Kind == TokenGT || next.Kind == TokenGTGT) && t.End == next.Start {
				op := "&" + next.Text
				p.advance()
				t = p.cur()
				p.advance()
				target, fdDup, ok := p.redirectTarget()
				if !ok {
					return nil, &ParseError{Kind: KindStructural, At: t.Start, Token: op, Msg: "redirect missing target"}
				}
				if fdDup {
					op = ">&"
				}
				s.Redirects = append(s.Redirects, Redirect{
					Operator: op,
					Target:   target,
					Start:    t.Start,
					End:      target.End,
				})
				continue
			}
			return nil, &ParseError{Kind: KindUnsupported, At: t.Start, Token: t.Text, Msg: "unsupported syntax"}
		default:
			return nil, &ParseError{Kind: KindUnsupported, At: t.Start, Token: t.Text, Msg: "unsupported syntax"}
		}
	}

	if len(s.Assignments) == 0 && len(s.Words) == 0 && len(s.Redirects) == 0 {
		return nil, &ParseError{Kind: KindStructural, At: s.Start, Msg: "empty command"}
	}
	s.End = simpleCommandEnd(s)
	return s, nil
}

// redirectTarget consumes the word after a redirect operator. fdDup is true
// for the `&N` duplication form (2>&1), where the word names a descriptor
// rather than a file.
func (p *parser) redirectTarget() (Word, bool, bool) {
	if p.done() || isDelimiter(p.cur().Kind) || isRedirectOp(p.cur().Kind) {
		return Word{}, false, false
	}
	t := p.cur()
	if t.Kind == TokenAmp {
		next := p.toks[min(p.pos+1, len(p.toks)-1)]
		if isDigitWord(next) && t.End == next.Start {
			p.advance()
			dup := p.cur()
			p.advance()
			return Word{Text: "&" + dup.Text, Start: t.Start, End: dup.End}, true, true
		}
		return Word{}, false, false
	}
	p.advance()
	return Word{Text: t.Text, Start: t.Start, End: t.End}, false, true
}

func isDigits(text string) bool {
	if text == "" {
		return false
	}
	for i := range len(text) {
		if text[i] < '0' || text[i] > '9' {
			return false
		}
	}
	return true
}

func isDigitWord(t Token) bool {
	return (t.Kind == TokenWord || t.Kind == TokenEscape) && isDigits(t.Text)
}

func (p *parser) parseSubshell() (Node, error) {
	open := p.cur()
	p.advance()
	body, err := p.parseSequence()
	if err != nil {
		return nil, err
	}
	if !p.at(TokenRParen) {
		return nil, &ParseError{Kind: KindStructural, At: open.Start, Token: open.Text, Msg: "missing closing ')' for grouping"}
	}
	closeT := p.cur()
	p.advance()
	return &Subshell{Body: body, Start: open.Start, End: closeT.End}, nil
}

// makeAssignment splits a leading name=value word using POSIX-ish identifier
// rules so FOO=bar becomes an Assignment with a byte-accurate value span.
func makeAssignment(t Token) (Assignment, bool) {
	name, valueText, ok := strings.Cut(t.Text, "=")
	if !ok || !isValidIdent(name) {
		return Assignment{}, false
	}
	a := Assignment{
		Name:  name,
		Value: Word{Text: valueText, Start: t.Start + len(name) + 1, End: t.End},
		Start: t.Start,
		End:   t.End,
	}
	return a, true
}

func isValidIdent(name string) bool {
	if name == "" {
		return false
	}
	for i := range len(name) {
		c := name[i]
		isLetter := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
		isDigit := c >= '0' && c <= '9'
		if i == 0 {
			if !isLetter {
				return false
			}
			continue
		}
		if !isLetter && !isDigit {
			return false
		}
	}
	return true
}

func nodeStart(n Node) int {
	switch v := n.(type) {
	case *Word:
		return v.Start
	case *Sequence:
		return v.Start
	case *Binary:
		return v.Start
	case *Pipeline:
		return v.Start
	case *SimpleCommand:
		return v.Start
	case *Subshell:
		return v.Start
	case *CommandSubstitution:
		return v.Start
	}
	return 0
}

func nodeEnd(n Node) int {
	switch v := n.(type) {
	case *Word:
		return v.End
	case *Sequence:
		return v.End
	case *Binary:
		return v.End
	case *Pipeline:
		return v.End
	case *SimpleCommand:
		return v.End
	case *Subshell:
		return v.End
	case *CommandSubstitution:
		return v.End
	}
	return 0
}

func simpleCommandEnd(s *SimpleCommand) int {
	end := s.Start
	if len(s.Words) > 0 {
		end = s.Words[len(s.Words)-1].End
	}
	if len(s.Redirects) > 0 {
		end = max(end, s.Redirects[len(s.Redirects)-1].End)
	}
	return end
}

func isRedirectOp(k TokenKind) bool {
	return k == TokenGT || k == TokenGTGT || k == TokenLT
}

func isDelimiter(t TokenKind) bool {
	return t == TokenEOF || t == TokenSemi || t == TokenAndAnd || t == TokenOrOr ||
		t == TokenPipe || t == TokenRParen || t == TokenLParen
}
