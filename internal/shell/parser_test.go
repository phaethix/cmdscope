package shell

import "testing"

func parseNode(t *testing.T, input string) Node {
	t.Helper()
	toks, err := Lex(input)
	if err != nil {
		t.Fatalf("Lex(%q) error: %v", input, err)
	}
	node, perr := Parse(toks)
	if perr != nil {
		t.Fatalf("Parse(%q) error: %v", input, perr)
	}
	if node == nil {
		t.Fatalf("Parse(%q) returned nil node", input)
	}
	return node
}

func asSimple(t *testing.T, n Node) *SimpleCommand {
	t.Helper()
	s, ok := n.(*SimpleCommand)
	if !ok {
		t.Fatalf("expected *SimpleCommand, got %T", n)
	}
	return s
}

func wordsOf(t *testing.T, s *SimpleCommand) []string {
	t.Helper()
	out := make([]string, 0, len(s.Words))
	for _, w := range s.Words {
		out = append(out, w.Text)
	}
	return out
}

func TestParserSimpleCommand(t *testing.T) {
	n := parseNode(t, "echo hi")
	s := asSimple(t, n)
	want := []string{"echo", "hi"}
	if got := wordsOf(t, s); !equalStrings(got, want) {
		t.Errorf("Words = %v, want %v", got, want)
	}
	if s.Start != 0 || s.End != 7 {
		t.Errorf("span = [%d,%d), want [0,7)", s.Start, s.End)
	}
}

func TestParserQuotedWordsKeepRawText(t *testing.T) {
	n := parseNode(t, "echo 'a b'")
	s := asSimple(t, n)
	got := wordsOf(t, s)
	want := []string{"echo", "'a b'"}
	if !equalStrings(got, want) {
		t.Errorf("Words = %v, want %v", got, want)
	}
	if s.Words[1].Start != 5 || s.Words[1].End != 10 {
		t.Errorf("quoted word span = [%d,%d), want [5,10)", s.Words[1].Start, s.Words[1].End)
	}
}

func TestParserRedirect(t *testing.T) {
	n := parseNode(t, "echo hi > output.txt")
	s := asSimple(t, n)
	if len(s.Redirects) != 1 {
		t.Fatalf("Redirects = %d, want 1", len(s.Redirects))
	}
	r := s.Redirects[0]
	if r.Operator != ">" {
		t.Errorf("redirect operator = %q, want >", r.Operator)
	}
	if r.Target.Text != "output.txt" {
		t.Errorf("redirect target = %q, want output.txt", r.Target.Text)
	}
	if s.Start != 0 || s.End != 20 {
		t.Errorf("span = [%d,%d), want [0,20)", s.Start, s.End)
	}
}

func TestParserAppendRedirect(t *testing.T) {
	n := parseNode(t, "echo hi >> log.txt")
	s := asSimple(t, n)
	if len(s.Redirects) != 1 || s.Redirects[0].Operator != ">>" {
		t.Fatalf("expected one >> redirect, got %+v", s.Redirects)
	}
}

func TestParserAssignment(t *testing.T) {
	n := parseNode(t, "FOO=bar cmd arg")
	s := asSimple(t, n)
	if len(s.Assignments) != 1 {
		t.Fatalf("Assignments = %d, want 1", len(s.Assignments))
	}
	if s.Assignments[0].Name != "FOO" || s.Assignments[0].Value.Text != "bar" {
		t.Errorf("assignment = %+v, want FOO=bar", s.Assignments[0])
	}
	if got := wordsOf(t, s); !equalStrings(got, []string{"cmd", "arg"}) {
		t.Errorf("Words = %v, want [cmd arg]", got)
	}
}

func TestParserAndOr(t *testing.T) {
	n := parseNode(t, "a && b")
	b, ok := n.(*Binary)
	if !ok || b.Op != "&&" {
		t.Fatalf("expected && Binary, got %T %v", n, n)
	}
}

func TestParserAndOrLeftAssoc(t *testing.T) {
	n := parseNode(t, "a && b && c")
	b, ok := n.(*Binary)
	if !ok || b.Op != "&&" {
		t.Fatalf("top = %T, want && Binary", n)
	}
	inner, ok := b.Left.(*Binary)
	if !ok || inner.Op != "&&" {
		t.Fatalf("left = %T, want nested && Binary (left assoc)", b.Left)
	}
}

func TestParserPipeline(t *testing.T) {
	n := parseNode(t, "a | b | c")
	p, ok := n.(*Pipeline)
	if !ok {
		t.Fatalf("expected *Pipeline, got %T", n)
	}
	if len(p.Commands) != 3 {
		t.Fatalf("pipeline Commands = %d, want 3", len(p.Commands))
	}
}

func TestParserSequence(t *testing.T) {
	n := parseNode(t, "a; b; c")
	seq, ok := n.(*Sequence)
	if !ok {
		t.Fatalf("expected *Sequence, got %T", n)
	}
	if len(seq.Items) != 3 {
		t.Fatalf("Sequence Items = %d, want 3", len(seq.Items))
	}
}

func TestParserMixedPrecedence(t *testing.T) {
	// (§4.4) a && b || c ; d | e
	// ; -> [ (a && b) || c , d | e ]
	n := parseNode(t, "a && b || c ; d | e")
	seq, ok := n.(*Sequence)
	if !ok {
		t.Fatalf("expected top-level *Sequence, got %T", n)
	}
	if len(seq.Items) != 2 {
		t.Fatalf("Sequence Items = %d, want 2", len(seq.Items))
	}
	binary, ok := seq.Items[0].(*Binary)
	if !ok || binary.Op != "||" {
		t.Fatalf("Items[0] = %T, want || Binary", seq.Items[0])
	}
	andand, ok := binary.Left.(*Binary)
	if !ok || andand.Op != "&&" {
		t.Fatalf("||.Left = %T, want && Binary", binary.Left)
	}
	p, ok := seq.Items[1].(*Pipeline)
	if !ok || len(p.Commands) != 2 {
		t.Fatalf("Items[1] = %T, want 2-command pipeline", seq.Items[1])
	}
}

func TestParserSubshell(t *testing.T) {
	n := parseNode(t, "(a; b)")
	sub, ok := n.(*Subshell)
	if !ok {
		t.Fatalf("expected *Subshell, got %T", n)
	}
	body, ok := sub.Body.(*Sequence)
	if !ok {
		t.Fatalf("Subshell.Body = %T, want *Sequence", sub.Body)
	}
	if len(body.Items) != 2 {
		t.Fatalf("subshell body items = %d, want 2", len(body.Items))
	}
}

func TestParserSubshellInAndOr(t *testing.T) {
	n := parseNode(t, "(a) && b")
	b, ok := n.(*Binary)
	if !ok || b.Op != "&&" {
		t.Fatalf("expected && Binary, got %T", n)
	}
	if _, ok := b.Left.(*Subshell); !ok {
		t.Fatalf("left = %T, want *Subshell", b.Left)
	}
}

func TestParserCommandSubstitutionAsWord(t *testing.T) {
	n := parseNode(t, "echo $(pwd)")
	s := asSimple(t, n)
	want := []string{"echo", "$(pwd)"}
	if got := wordsOf(t, s); !equalStrings(got, want) {
		t.Errorf("Words = %v, want %v", got, want)
	}
}

func TestParserDoesNotPanicOnEmpty(t *testing.T) {
	toks, err := Lex("")
	if err != nil {
		t.Fatalf("Lex error: %v", err)
	}
	if _, perr := Parse(toks); perr != nil {
		t.Fatalf("Parse empty should succeed, got %v", perr)
	}
}

// equalStrings is a small non-reflect slice comparator.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
