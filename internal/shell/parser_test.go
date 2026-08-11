package shell

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseNode(t *testing.T, input string) Node {
	t.Helper()
	toks, err := Lex(input)
	require.NoError(t, err, "Lex(%q)", input)
	node, perr := Parse(toks)
	require.NoError(t, perr, "Parse(%q)", input)
	require.NotNil(t, node, "Parse(%q) returned nil node", input)
	return node
}

func asSimple(t *testing.T, n Node) *SimpleCommand {
	t.Helper()
	s, ok := n.(*SimpleCommand)
	require.True(t, ok, "expected *SimpleCommand, got %T", n)
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
	assert.Equal(t, []string{"echo", "hi"}, wordsOf(t, s))
	assert.Equal(t, 0, s.Start)
	assert.Equal(t, 7, s.End)
}

func TestParserQuotedWordsKeepRawText(t *testing.T) {
	n := parseNode(t, "echo 'a b'")
	s := asSimple(t, n)
	assert.Equal(t, []string{"echo", "'a b'"}, wordsOf(t, s))
	assert.Equal(t, 5, s.Words[1].Start)
	assert.Equal(t, 10, s.Words[1].End)
}

func TestParserRedirect(t *testing.T) {
	n := parseNode(t, "echo hi > output.txt")
	s := asSimple(t, n)
	require.Len(t, s.Redirects, 1)
	r := s.Redirects[0]
	assert.Equal(t, ">", r.Operator)
	assert.Equal(t, "output.txt", r.Target.Text)
	assert.Equal(t, 0, s.Start)
	assert.Equal(t, 20, s.End)
}

func TestParserAppendRedirect(t *testing.T) {
	n := parseNode(t, "echo hi >> log.txt")
	s := asSimple(t, n)
	require.Len(t, s.Redirects, 1)
	require.Equal(t, ">>", s.Redirects[0].Operator)
}

func TestParserAssignment(t *testing.T) {
	n := parseNode(t, "FOO=bar cmd arg")
	s := asSimple(t, n)
	require.Len(t, s.Assignments, 1)
	assert.Equal(t, "FOO", s.Assignments[0].Name)
	assert.Equal(t, "bar", s.Assignments[0].Value.Text)
	assert.Equal(t, []string{"cmd", "arg"}, wordsOf(t, s))
}

func TestParserAndOr(t *testing.T) {
	n := parseNode(t, "a && b")
	b, ok := n.(*Binary)
	require.True(t, ok, "expected && Binary, got %T %v", n, n)
	require.Equal(t, "&&", b.Op)
}

func TestParserAndOrLeftAssoc(t *testing.T) {
	n := parseNode(t, "a && b && c")
	b, ok := n.(*Binary)
	require.True(t, ok, "top = %T, want && Binary", n)
	require.Equal(t, "&&", b.Op)
	inner, ok := b.Left.(*Binary)
	require.True(t, ok, "left = %T, want nested && Binary (left assoc)", b.Left)
	require.Equal(t, "&&", inner.Op)
}

func TestParserPipeline(t *testing.T) {
	n := parseNode(t, "a | b | c")
	p, ok := n.(*Pipeline)
	require.True(t, ok, "expected *Pipeline, got %T", n)
	require.Len(t, p.Commands, 3)
}

func TestParserSequence(t *testing.T) {
	n := parseNode(t, "a; b; c")
	seq, ok := n.(*Sequence)
	require.True(t, ok, "expected *Sequence, got %T", n)
	require.Len(t, seq.Items, 3)
}

func TestParserMixedPrecedence(t *testing.T) {
	// a && b || c ; d | e  →  Sequence[ (a && b) || c , d | e ]
	n := parseNode(t, "a && b || c ; d | e")
	seq, ok := n.(*Sequence)
	require.True(t, ok, "expected top-level *Sequence, got %T", n)
	require.Len(t, seq.Items, 2)
	binary, ok := seq.Items[0].(*Binary)
	require.True(t, ok, "Items[0] = %T, want || Binary", seq.Items[0])
	require.Equal(t, "||", binary.Op)
	andand, ok := binary.Left.(*Binary)
	require.True(t, ok, "||.Left = %T, want && Binary", binary.Left)
	require.Equal(t, "&&", andand.Op)
	p, ok := seq.Items[1].(*Pipeline)
	require.True(t, ok, "Items[1] = %T, want 2-command pipeline", seq.Items[1])
	require.Len(t, p.Commands, 2)
}

func TestParserSubshell(t *testing.T) {
	n := parseNode(t, "(a; b)")
	sub, ok := n.(*Subshell)
	require.True(t, ok, "expected *Subshell, got %T", n)
	body, ok := sub.Body.(*Sequence)
	require.True(t, ok, "Subshell.Body = %T, want *Sequence", sub.Body)
	require.Len(t, body.Items, 2)
}

func TestParserSubshellInAndOr(t *testing.T) {
	n := parseNode(t, "(a) && b")
	b, ok := n.(*Binary)
	require.True(t, ok, "expected && Binary, got %T", n)
	require.Equal(t, "&&", b.Op)
	_, ok = b.Left.(*Subshell)
	require.True(t, ok, "left = %T, want *Subshell", b.Left)
}

func TestParserCommandSubstitutionAsWord(t *testing.T) {
	n := parseNode(t, "echo $(pwd)")
	s := asSimple(t, n)
	assert.Equal(t, []string{"echo", "$(pwd)"}, wordsOf(t, s))
}

func TestParserDoesNotPanicOnEmpty(t *testing.T) {
	toks, err := Lex("")
	require.NoError(t, err)
	_, perr := Parse(toks)
	require.NoError(t, perr)
}
