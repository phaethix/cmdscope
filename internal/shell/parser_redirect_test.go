package shell

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserFdRedirect(t *testing.T) {
	cases := []struct {
		name       string
		input      string
		wantWords  []string
		wantOp     string
		wantTarget string
	}{
		{
			name:       "fd prefix 2>",
			input:      "echo hi 2> err.txt",
			wantWords:  []string{"echo", "hi"},
			wantOp:     ">",
			wantTarget: "err.txt",
		},
		{
			name:       "fd prefix 2>>",
			input:      "make 2>> build.log",
			wantWords:  []string{"make"},
			wantOp:     ">>",
			wantTarget: "build.log",
		},
		{
			name:       "fd dup 2>&1",
			input:      "echo x 2>&1",
			wantWords:  []string{"echo", "x"},
			wantOp:     ">&",
			wantTarget: "&1",
		},
		{
			name:       "both streams &>",
			input:      "echo x &> all.log",
			wantWords:  []string{"echo", "x"},
			wantOp:     "&>",
			wantTarget: "all.log",
		},
		{
			name:       "both streams append &>>",
			input:      "echo x &>> all.log",
			wantWords:  []string{"echo", "x"},
			wantOp:     "&>>",
			wantTarget: "all.log",
		},
		{
			name:       "2 with space is real argv",
			input:      "echo 2 > file",
			wantWords:  []string{"echo", "2"},
			wantOp:     ">",
			wantTarget: "file",
		},
		{
			name:       "12> with multi-digit fd",
			input:      "cmd 12> out.txt",
			wantWords:  []string{"cmd"},
			wantOp:     ">",
			wantTarget: "out.txt",
		},
		{
			name:       "1< input fd",
			input:      "cat 1< input.txt",
			wantWords:  []string{"cat"},
			wantOp:     "<",
			wantTarget: "input.txt",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := asSimpleCmd(t, tc.input)
			words := wordTexts(s.Words)
			assert.Equal(t, tc.wantWords, words)
			require.Len(t, s.Redirects, 1)
			r := s.Redirects[0]
			assert.Equal(t, tc.wantOp, r.Operator)
			assert.Equal(t, tc.wantTarget, r.Target.Text)
		})
	}
}

func TestParserFdRedirectMultiple(t *testing.T) {
	s := asSimpleCmd(t, "cmd 2> err.txt 1> out.txt")
	words := wordTexts(s.Words)
	assert.Equal(t, []string{"cmd"}, words)
	require.Len(t, s.Redirects, 2)
	assert.Equal(t, ">", s.Redirects[0].Operator)
	assert.Equal(t, "err.txt", s.Redirects[0].Target.Text)
	assert.Equal(t, ">", s.Redirects[1].Operator)
	assert.Equal(t, "out.txt", s.Redirects[1].Target.Text)
}

func asSimpleCmd(t *testing.T, input string) *SimpleCommand {
	t.Helper()
	toks, err := Lex(input)
	require.NoError(t, err)
	n, err := Parse(toks)
	require.NoError(t, err)
	s, ok := n.(*SimpleCommand)
	require.True(t, ok, "want *SimpleCommand, got %T", n)
	return s
}

func wordTexts(words []Word) []string {
	out := make([]string, len(words))
	for i, w := range words {
		out[i] = w.Text
	}
	return out
}
