package logicalpath_test

import (
	"testing"

	"github.com/phaethix/cmdscope/internal/logicalpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathNormalize(t *testing.T) {
	tests := []struct {
		name      string
		raw, cwd  string
		want      string
		wantFlags logicalpath.PathFlags
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
			wantFlags: logicalpath.PathUnprovenDotDot,
		},
		{
			name:      "glob blocks parent collapse",
			raw:       "*/../x",
			cwd:       "/ws",
			want:      "/ws/*/../x",
			wantFlags: logicalpath.PathHasGlob | logicalpath.PathUnprovenDotDot,
		},
		{
			name:      "glob pattern preserved",
			raw:       "dist/*.js",
			cwd:       "/ws",
			want:      "/ws/dist/*.js",
			wantFlags: logicalpath.PathHasGlob,
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
			wantFlags: logicalpath.PathHasGlob,
		},
		{
			name:      "character class is glob",
			raw:       "file[ab].txt",
			cwd:       "/ws",
			want:      "/ws/file[ab].txt",
			wantFlags: logicalpath.PathHasGlob,
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
			wantFlags: logicalpath.PathUnprovenDotDot,
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
			got, flags := logicalpath.NormalizeLogicalPath(tt.raw, tt.cwd)
			require.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantFlags, flags)
		})
	}
}

func TestPathFlagsHas(t *testing.T) {
	f := logicalpath.PathHasGlob | logicalpath.PathUnprovenDotDot
	require.True(t, f.Has(logicalpath.PathHasGlob))
	require.True(t, f.Has(logicalpath.PathUnprovenDotDot))
	require.False(t, logicalpath.PathFlags(0).Has(logicalpath.PathHasGlob))
}
