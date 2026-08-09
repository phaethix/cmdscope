package ir_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/phaethix/cmdscope/internal/ir"
)

func TestRequestValidateAcceptsMinimalCommand(t *testing.T) {
	req := ir.AnalyzeRequest{Command: "echo hi"}
	if err := ir.ValidateRequest(req); err != nil {
		t.Fatalf("ValidateRequest() = %v, want nil", err)
	}
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
	if err := ir.ValidateRequest(req); err != nil {
		t.Fatalf("ValidateRequest() = %v, want nil", err)
	}
}

func TestContextValidateRejectsInvalidCWD(t *testing.T) {
	cases := []struct {
		name string
		cwd  string
	}{
		{name: "empty", cwd: ""},
		{name: "relative", cwd: "workspace"},
		{name: "logical missing name", cwd: "logical://"},
		{name: "logical with slash", cwd: "logical://foo/bar"},
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
			if err := ir.ValidateRequest(req); err != nil {
				t.Fatalf("ValidateRequest() = %v, want nil", err)
			}
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
	if err := ir.ValidateRequest(req); err != nil {
		t.Fatalf("ValidateRequest() = %v, want nil", err)
	}
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

func TestContextParseAcceptsValidJSON(t *testing.T) {
	ctx, err := ir.ParseAnalysisContextJSON([]byte(`{"cwd":"/workspace","files":{"package.json":"{}"}}`))
	if err != nil {
		t.Fatalf("ParseAnalysisContextJSON() = %v, want nil", err)
	}
	if ctx.CWD != "/workspace" {
		t.Fatalf("CWD = %q, want /workspace", ctx.CWD)
	}
	if ctx.Files["package.json"] != "{}" {
		t.Fatalf("Files[package.json] = %q, want {}", ctx.Files["package.json"])
	}
}

func TestValidationErrorJSONShape(t *testing.T) {
	err := ir.NewValidationError(ir.ErrCodeEmptyCommand, "command must not be empty")
	data, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("json.Marshal() = %v", marshalErr)
	}
	const want = `{"error_code":"empty_command","message":"command must not be empty"}`
	if string(data) != want {
		t.Fatalf("json = %s, want %s", data, want)
	}
}

func assertValidationCode(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want %q", want)
	}
	ve, ok := err.(*ir.ValidationError)
	if !ok {
		t.Fatalf("error type = %T, want *ir.ValidationError", err)
	}
	if ve.Code != want {
		t.Fatalf("error code = %q, want %q", ve.Code, want)
	}
}
