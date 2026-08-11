package ir_test

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/phaethix/cmdscope/internal/ir"
	"github.com/stretchr/testify/require"
)

func TestRequestValidateAcceptsMinimalCommand(t *testing.T) {
	req := ir.AnalyzeRequest{Command: "echo hi"}
	require.NoError(t, ir.ValidateRequest(req))
}

func TestRequestValidateRejectsEmptyCommand(t *testing.T) {
	req := ir.AnalyzeRequest{Command: ""}
	err := ir.ValidateRequest(req)
	assertValidationCode(t, err, ir.ErrCodeEmptyCommand)
}

func TestRequestValidateRejectsWhitespaceOnlyCommand(t *testing.T) {
	req := ir.AnalyzeRequest{Command: "   \t\n  "}
	err := ir.ValidateRequest(req)
	assertValidationCode(t, err, ir.ErrCodeEmptyCommand)
}

func TestRequestValidateRejectsOversizedCommand(t *testing.T) {
	req := ir.AnalyzeRequest{Command: strings.Repeat("a", ir.MaxCommandBytes+1)}
	err := ir.ValidateRequest(req)
	assertValidationCode(t, err, ir.ErrCodeInputTooLarge)
}

func TestRequestValidateAcceptsMaxSizedCommand(t *testing.T) {
	req := ir.AnalyzeRequest{Command: strings.Repeat("a", ir.MaxCommandBytes)}
	require.NoError(t, ir.ValidateRequest(req))
}

func TestContextValidateRejectsInvalidCWD(t *testing.T) {
	cases := []struct {
		name string
		cwd  string
	}{
		{name: "empty", cwd: ""},
		{name: "relative", cwd: "workspace"},
		{name: "windows drive backslash", cwd: `C:\work`},
		{name: "windows drive slash", cwd: "C:/work"},
		{name: "logical missing name", cwd: "logical://"},
		{name: "logical with slash", cwd: "logical://foo/bar"},
		{name: "logical dot", cwd: "logical://."},
		{name: "logical dotdot", cwd: "logical://.."},
		{name: "logical backslash", cwd: "logical://foo\\bar"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := ir.AnalyzeRequest{
				Command: "echo hi",
				Context: &ir.AnalysisContext{CWD: tc.cwd},
			}
			err := ir.ValidateRequest(req)
			assertValidationCode(t, err, ir.ErrCodeInvalidCWD)
		})
	}
}

func TestContextValidateAcceptsValidCWD(t *testing.T) {
	cases := []string{"/workspace", "logical://unknown", "logical://my-repo"}
	for _, cwd := range cases {
		t.Run(cwd, func(t *testing.T) {
			req := ir.AnalyzeRequest{
				Command: "echo hi",
				Context: &ir.AnalysisContext{CWD: cwd},
			}
			require.NoError(t, ir.ValidateRequest(req))
		})
	}
}

func TestContextValidateRejectsInvalidFileKeys(t *testing.T) {
	cases := []struct {
		name string
		key  string
	}{
		{name: "empty", key: ""},
		{name: "absolute", key: "/package.json"},
		{name: "parent traversal", key: "../secret"},
		{name: "embedded traversal", key: "foo/../bar"},
		{name: "backslash", key: "foo\\bar"},
		{name: "volume prefix uppercase", key: "C:/foo"},
		{name: "volume prefix lowercase", key: "c:foo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := ir.AnalyzeRequest{
				Command: "echo hi",
				Context: &ir.AnalysisContext{
					CWD:   "/workspace",
					Files: map[string]string{tc.key: "content"},
				},
			}
			err := ir.ValidateRequest(req)
			assertValidationCode(t, err, ir.ErrCodeInvalidContextPath)
		})
	}
}

func TestContextValidateAcceptsValidFileKeys(t *testing.T) {
	req := ir.AnalyzeRequest{
		Command: "echo hi",
		Context: &ir.AnalysisContext{
			CWD: "/workspace",
			Files: map[string]string{
				"package.json":      "{}",
				"scripts/deploy.sh": "#!/bin/sh",
			},
		},
	}
	require.NoError(t, ir.ValidateRequest(req))
}

func TestContextValidateRejectsOversizedSingleFile(t *testing.T) {
	req := ir.AnalyzeRequest{
		Command: "echo hi",
		Context: &ir.AnalysisContext{
			CWD:   "/workspace",
			Files: map[string]string{"large.txt": strings.Repeat("x", ir.MaxContextFileBytes+1)},
		},
	}
	err := ir.ValidateRequest(req)
	assertValidationCode(t, err, ir.ErrCodeContextFileTooLarge)
}

func TestContextValidateRejectsOversizedTotalContext(t *testing.T) {
	chunk := strings.Repeat("x", ir.MaxContextFileBytes)
	req := ir.AnalyzeRequest{
		Command: "echo hi",
		Context: &ir.AnalysisContext{
			CWD: "/workspace",
			Files: map[string]string{
				"a.txt": chunk,
				"b.txt": chunk,
				"c.txt": chunk,
				"d.txt": chunk,
				"e.txt": chunk,
				"f.txt": chunk,
				"g.txt": chunk,
				"h.txt": chunk,
				"i.txt": "x",
			},
		},
	}
	err := ir.ValidateRequest(req)
	assertValidationCode(t, err, ir.ErrCodeContextFileTooLarge)
}

func TestContextParseRejectsInvalidJSON(t *testing.T) {
	_, err := ir.ParseAnalysisContextJSON([]byte(`{"cwd":`))
	assertValidationCode(t, err, ir.ErrCodeInvalidContextJSON)
}

func TestContextParseRejectsOversizedJSON(t *testing.T) {
	// 8 MiB of raw JSON must be rejected before any full unmarshal/alloc.
	data := []byte(`{"cwd":"/workspace","pad":"` + strings.Repeat("x", ir.MaxTotalContextBytes) + `"}`)
	_, err := ir.ParseAnalysisContextJSON(data)
	assertValidationCode(t, err, ir.ErrCodeContextFileTooLarge)
}

func TestContextValidateAcceptsValidJSON(t *testing.T) {
	ctx, err := ir.ParseAnalysisContextJSON([]byte(`{"cwd":"/workspace","files":{"package.json":"{}"}}`))
	require.NoError(t, err)
	require.Equal(t, "/workspace", ctx.CWD)
	require.Equal(t, "{}", ctx.Files["package.json"])
}

func TestContextRejectsNonCanonicalFileKeys(t *testing.T) {
	cases := []struct {
		name string
		key  string
	}{
		{name: "current dir prefix", key: "./foo"},
		{name: "embedded current dir", key: "foo/./bar"},
		{name: "collapsed double slash", key: "foo//bar"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := ir.AnalyzeRequest{
				Command: "echo hi",
				Context: &ir.AnalysisContext{
					CWD:   "/workspace",
					Files: map[string]string{tc.key: "content"},
				},
			}
			err := ir.ValidateRequest(req)
			assertValidationCode(t, err, ir.ErrCodeInvalidContextPath)
		})
	}
}

func TestContextAcceptsEnvAndPlatformShell(t *testing.T) {
	req := ir.AnalyzeRequest{
		Command: "echo hi",
		Context: &ir.AnalysisContext{
			CWD:      "/workspace",
			Platform: "darwin",
			Shell:    "zsh",
			Env:      map[string]string{"FOO": "bar"},
		},
	}
	require.NoError(t, ir.ValidateRequest(req))
}

func TestContextRejectsEmptyEnvKey(t *testing.T) {
	req := ir.AnalyzeRequest{
		Command: "echo hi",
		Context: &ir.AnalysisContext{
			CWD: "/workspace",
			Env: map[string]string{"": "bar"},
		},
	}
	err := ir.ValidateRequest(req)
	require.Error(t, err, "empty env key must be rejected")
	ve, ok := err.(*ir.ValidationError)
	require.True(t, ok, "error type = %T, want *ir.ValidationError", err)
	require.Equal(t, ir.ErrCodeInvalidContextField, ve.Code)
}

func TestContextRejectsOversizedPlatformShell(t *testing.T) {
	long := strings.Repeat("d", ir.MaxPlatformShellBytes+1)
	for _, name := range []string{"platform", "shell"} {
		t.Run(name, func(t *testing.T) {
			ctx := ir.AnalysisContext{CWD: "/workspace"}
			if name == "platform" {
				ctx.Platform = long
			} else {
				ctx.Shell = long
			}
			req := ir.AnalyzeRequest{Command: "echo hi", Context: &ctx}
			assertValidationCode(t, ir.ValidateRequest(req), ir.ErrCodeInvalidContextField)
		})
	}
}

func TestContextEnvCountsTowardTotalBytes(t *testing.T) {
	// Many small env entries whose summed size alone exceeds the total cap must
	// be rejected, proving Env participates in the total-bytes limit alongside
	// (or independently of) Files.
	env := make(map[string]string, 2200)
	for i := range 2200 {
		env[strconv.Itoa(i)] = strings.Repeat("y", ir.MaxContextEnvValueBytes)
	}
	req := ir.AnalyzeRequest{
		Command: "echo hi",
		Context: &ir.AnalysisContext{CWD: "/workspace", Env: env},
	}
	assertValidationCode(t, ir.ValidateRequest(req), ir.ErrCodeContextFileTooLarge)
}

func TestValidationErrorJSONShape(t *testing.T) {
	err := ir.NewValidationError(ir.ErrCodeEmptyCommand, "command must not be empty")
	data, marshalErr := json.Marshal(err)
	require.NoError(t, marshalErr)
	require.JSONEq(t, `{"error_code":"empty_command","message":"command must not be empty"}`, string(data))
}

func assertValidationCode(t *testing.T, err error, want string) {
	t.Helper()
	require.Error(t, err)
	ve, ok := err.(*ir.ValidationError)
	require.True(t, ok, "error type = %T, want *ir.ValidationError", err)
	require.Equal(t, want, ve.Code)
}
