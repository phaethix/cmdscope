package analyzer_test

import (
	"testing"

	"github.com/phaethix/cmdscope/internal/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathNormalize(t *testing.T) {
	tests := []struct {
		name      string
		raw, cwd  string
		want      string
		wantFlags analyzer.PathFlags
	}{
		{
			name: "relative joins cwd",
			raw:  "output.txt",
			cwd:  "/workspace",
			want: "/workspace/output.txt",
		},
		{
			name: "backslash to slash",
			raw:  `foo\bar`,
			cwd:  "/ws",
			want: "/ws/foo/bar",
		},
		{
			name: "collapse dot segments",
			raw:  "foo/./bar",
			cwd:  "/ws",
			want: "/ws/foo/bar",
		},
		{
			name: "provable parent against concrete segment",
			raw:  "foo/../bar",
			cwd:  "/ws",
			want: "/ws/bar",
		},
		{
			name: "provable escape of single cwd segment",
			raw:  "../x",
			cwd:  "/ws",
			want: "/x",
		},
		{
			name:      "unproven escape past absolute root",
			raw:       "../../x",
			cwd:       "/ws",
			want:      "/../x",
			wantFlags: analyzer.PathUnprovenDotDot,
		},
		{
			name:      "glob blocks parent collapse",
			raw:       "*/../x",
			cwd:       "/ws",
			want:      "/ws/*/../x",
			wantFlags: analyzer.PathHasGlob | analyzer.PathUnprovenDotDot,
		},
		{
			name:      "glob pattern preserved",
			raw:       "dist/*.js",
			cwd:       "/ws",
			want:      "/ws/dist/*.js",
			wantFlags: analyzer.PathHasGlob,
		},
		{
			name: "absolute raw ignores cwd for join",
			raw:  "/abs/./a",
			cwd:  "/ws",
			want: "/abs/a",
		},
		{
			name: "logical cwd prefix",
			raw:  "a",
			cwd:  "logical://workspace",
			want: "logical://workspace/a",
		},
		{
			name: "empty raw",
			raw:  "",
			cwd:  "/ws",
			want: "",
		},
		{
			name: "cwd trailing slash",
			raw:  "a",
			cwd:  "/ws/",
			want: "/ws/a",
		},
		{
			name:      "question mark is glob",
			raw:       "file?.txt",
			cwd:       "/ws",
			want:      "/ws/file?.txt",
			wantFlags: analyzer.PathHasGlob,
		},
		{
			name:      "character class is glob",
			raw:       "file[ab].txt",
			cwd:       "/ws",
			want:      "/ws/file[ab].txt",
			wantFlags: analyzer.PathHasGlob,
		},
		{
			name: "windows cwd separators",
			raw:  "a",
			cwd:  `\ws\proj`,
			want: "/ws/proj/a",
		},
		{
			name: "relative without cwd stays relative",
			raw:  "foo/../bar",
			cwd:  "",
			want: "bar",
		},
		{
			name:      "relative unproven without cwd",
			raw:       "../x",
			cwd:       "",
			want:      "../x",
			wantFlags: analyzer.PathUnprovenDotDot,
		},
		{
			name: "collapse empty segments",
			raw:  "foo//bar",
			cwd:  "/ws",
			want: "/ws/foo/bar",
		},
		{
			name: "absolute root alone",
			raw:  "/",
			cwd:  "/ws",
			want: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, flags := analyzer.NormalizeLogicalPath(tt.raw, tt.cwd)
			require.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantFlags, flags)
		})
	}
}

func TestPathFlagsHas(t *testing.T) {
	f := analyzer.PathHasGlob | analyzer.PathUnprovenDotDot
	require.True(t, f.Has(analyzer.PathHasGlob))
	require.True(t, f.Has(analyzer.PathUnprovenDotDot))
	require.False(t, analyzer.PathFlags(0).Has(analyzer.PathHasGlob))
}
