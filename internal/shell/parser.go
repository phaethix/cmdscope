package shell

import (
	"errors"
	"fmt"
)

// Parse turns a lexer token stream (which must end with a TokenEOF) into an
// AST rooted at a Node. It follows the architecture precedence rules:
// subshell > pipeline '|' > '&&'/'||' > ';'. Nodes carry UTF-8 byte spans.
//
// Parse never panics. A structurally invalid stream, or an unsupported L0-out-
// of-scope construct (background '&', '|&', here-doc '<<'), is reported as an
// error rather than a classified node; recognising those as unsupported
// unknowns is the responsibility of a later stage.
func Parse(tokens []Token) (Node, error) {
	if tokens == nil {
		return nil, errors.New("empty token stream")
	}
	p := &parser{toks: tokens}
	node, err := p.parseSequence()
	if err != nil {
		return node, err
	}
	if !p.done() {
		return node, fmt.Errorf("unexpected token %q at byte %d", p.cur().Text, p.cur().Start)
	}
	return node, nil
}

// parser is a cursor over the token stream.
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

// parseSequence parses ';'-separated lists at one nesting level.
func (p *parser) parseSequence() (Node, error) {
	var items []Node
	for !p.done() && !p.at(TokenRParen) {
		if p.at(TokenSemi) {
			// Skip empty statements between consecutive ';'.
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

// parseAndOr parses the left-associative '&&' / '||' operators.
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

// parsePipeline parses '|'-separated commands belonging to one stage.
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

// parseCommand parses a single simple command or a subshell.
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
		return nil, errors.New("cannot parse command at end of input")
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
			p.advance()
			target, ok := p.redirectTarget()
			if !ok {
				return nil, fmt.Errorf("redirect %q missing target at byte %d", op, t.Start)
			}
			s.Redirects = append(s.Redirects, Redirect{
				Operator: op,
				Target:   target,
				Start:    t.Start,
				End:      target.End,
			})
		default:
			// Unsupported L0-out-of-scope construct; surface it, never panic.
			return nil, fmt.Errorf("unsupported syntax %q at byte %d", t.Text, t.Start)
		}
	}

	if len(s.Assignments) == 0 && len(s.Words) == 0 && len(s.Redirects) == 0 {
		return nil, fmt.Errorf("empty command at byte %d", s.Start)
	}
	s.End = simpleCommandEnd(s)
	return s, nil
}

// redirectTarget consumes the word that follows a redirect operator.
func (p *parser) redirectTarget() (Word, bool) {
	if p.done() || isDelimiter(p.cur().Kind) || isRedirectOp(p.cur().Kind) {
		return Word{}, false
	}
	t := p.cur()
	p.advance()
	return Word{Text: t.Text, Start: t.Start, End: t.End}, true
}

// parseSubshell parses a parenthesised grouping.
func (p *parser) parseSubshell() (Node, error) {
	open := p.cur()
	p.advance()
	body, err := p.parseSequence()
	if err != nil {
		return nil, err
	}
	if !p.at(TokenRParen) {
		return nil, fmt.Errorf("missing closing ')' for grouping at byte %d", open.Start)
	}
	closeT := p.cur()
	p.advance()
	return &Subshell{Body: body, Start: open.Start, End: closeT.End}, nil
}

// makeAssignment reports whether a leading word is a simple name=value
// assignment, and if so builds the Assignment node with a byte-accurate span.
func makeAssignment(t Token) (Assignment, bool) {
	eq := indexEquals(t.Text)
	if eq < 0 || !isValidIdent(t.Text[:eq]) {
		return Assignment{}, false
	}
	name := t.Text[:eq]
	valueText := t.Text[eq+1:]
	a := Assignment{
		Name:  name,
		Value: Word{Text: valueText, Start: t.Start + eq + 1, End: t.End},
		Start: t.Start,
		End:   t.End,
	}
	return a, true
}

// indexEquals returns the byte index of the first '=' in s, or -1.
func indexEquals(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return i
		}
	}
	return -1
}

// isValidIdent reports whether name is a valid shell variable identifier.
func isValidIdent(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
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

// nodeStart / nodeEnd read back the byte span of a node.
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

// simpleCommandEnd is the byte offset at which the simple command ends.
func simpleCommandEnd(s *SimpleCommand) int {
	end := s.Start
	if len(s.Words) > 0 {
		end = s.Words[len(s.Words)-1].End
	}
	if len(s.Redirects) > 0 {
		if re := s.Redirects[len(s.Redirects)-1].End; re > end {
			end = re
		}
	}
	return end
}

// isRedirectOp reports whether the token kind is a redirection operator.
func isRedirectOp(k TokenKind) bool {
	return k == TokenGT || k == TokenGTGT || k == TokenLT
}

// isDelimiter reports whether the token ends a simple command.
func isDelimiter(t TokenKind) bool {
	return t == TokenEOF || t == TokenSemi || t == TokenAndAnd || t == TokenOrOr ||
		t == TokenPipe || t == TokenRParen || t == TokenLParen
}
