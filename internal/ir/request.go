package ir

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

// Request and context size limits from the product contract.
const (
	MaxCommandBytes         = 64 * 1024
	MaxContextFileBytes     = 1 * 1024 * 1024
	MaxTotalContextBytes    = 8 * 1024 * 1024
	MaxPlatformShellBytes   = 64
	MaxContextEnvKeyBytes   = 256
	MaxContextEnvValueBytes = 4096
)

// AnalyzeRequest is the core analyzer input.
type AnalyzeRequest struct {
	Command string           `json:"command"`
	Context *AnalysisContext `json:"context,omitempty"`
}

// AnalysisContext carries caller-supplied workspace metadata and explicit files.
type AnalysisContext struct {
	CWD      string            `json:"cwd"`
	Platform string            `json:"platform,omitempty"`
	Shell    string            `json:"shell,omitempty"`
	Files    map[string]string `json:"files,omitempty"`
	Env      map[string]string `json:"env,omitempty"`
}

// ValidateRequest checks request and context invariants before analysis.
func ValidateRequest(req AnalyzeRequest) error {
	if strings.TrimSpace(req.Command) == "" {
		return NewValidationError(ErrCodeEmptyCommand, "command must not be empty")
	}
	if len(req.Command) > MaxCommandBytes {
		return NewValidationError(ErrCodeInputTooLarge, "command exceeds maximum length")
	}
	if req.Context == nil {
		return nil
	}
	if err := validateContext(*req.Context); err != nil {
		return err
	}
	return nil
}

// ParseAnalysisContextJSON decodes JSON supplied by CLI or adapter layers.
func ParseAnalysisContextJSON(data []byte) (*AnalysisContext, error) {
	// Bound the payload before parsing so an oversized input is rejected
	// without being fully unmarshaled and allocated first.
	if len(data) > MaxTotalContextBytes {
		return nil, NewValidationError(ErrCodeContextFileTooLarge, "context JSON exceeds maximum size")
	}
	var ctx AnalysisContext
	if err := json.Unmarshal(data, &ctx); err != nil {
		return nil, NewValidationError(ErrCodeInvalidContextJSON, "context JSON is invalid")
	}
	if err := validateContext(ctx); err != nil {
		return nil, err
	}
	return &ctx, nil
}

func validateContext(ctx AnalysisContext) error {
	if !isValidCWD(ctx.CWD) {
		return NewValidationError(ErrCodeInvalidCWD, "cwd must be an absolute path or logical://<workspace-name>")
	}
	if err := validateShortField("platform", ctx.Platform); err != nil {
		return err
	}
	if err := validateShortField("shell", ctx.Shell); err != nil {
		return err
	}
	var totalBytes int
	for key, content := range ctx.Files {
		if !isValidContextFileKey(key) {
			return NewValidationError(ErrCodeInvalidContextPath, "context file key must be a relative POSIX path without .. or ./ segments")
		}
		size := len(content)
		if size > MaxContextFileBytes {
			return NewValidationError(ErrCodeContextFileTooLarge, "context file exceeds maximum size")
		}
		totalBytes += size
		if totalBytes > MaxTotalContextBytes {
			return NewValidationError(ErrCodeContextFileTooLarge, "total context size exceeds maximum")
		}
	}
	for key, value := range ctx.Env {
		if !isValidEnvKey(key) {
			return NewValidationError(ErrCodeInvalidContextField, "environment key must not be empty")
		}
		if len(key) > MaxContextEnvKeyBytes {
			return NewValidationError(ErrCodeInvalidContextField, "environment key exceeds maximum length")
		}
		if len(value) > MaxContextEnvValueBytes {
			return NewValidationError(ErrCodeInvalidContextField, "environment value exceeds maximum length")
		}
		totalBytes += len(key) + len(value)
		if totalBytes > MaxTotalContextBytes {
			return NewValidationError(ErrCodeContextFileTooLarge, "total context size exceeds maximum")
		}
	}
	return nil
}

// validateShortField bounds platform/shell length and rejects arbitrary overflow.
func validateShortField(name, value string) error {
	if value == "" {
		return nil
	}
	if len(value) > MaxPlatformShellBytes {
		return NewValidationError(ErrCodeInvalidContextField, name+" exceeds maximum length")
	}
	return nil
}

func isValidEnvKey(key string) bool {
	return key != ""
}

func isValidCWD(cwd string) bool {
	if cwd == "" {
		return false
	}
	const logicalPrefix = "logical://"
	if strings.HasPrefix(cwd, logicalPrefix) {
		name := cwd[len(logicalPrefix):]
		// A logical workspace name must not look like a path segment,
		// so ".", "..", and any slash (forward or back) are rejected.
		return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, `/\`)
	}
	return filepath.IsAbs(cwd)
}

func isValidContextFileKey(key string) bool {
	if key == "" {
		return false
	}
	if strings.Contains(key, "\\") {
		return false
	}
	// A drive-letter volume prefix ("C:", "c:") is not a relative POSIX path.
	if len(key) >= 2 && key[1] == ':' {
		if (key[0] >= 'a' && key[0] <= 'z') || (key[0] >= 'A' && key[0] <= 'Z') {
			return false
		}
	}
	if strings.HasPrefix(key, "/") {
		return false
	}
	// A non-canonical key containing "." or ".." segments or collapsed empty
	// segments (e.g. "./foo", "foo//bar") would alias other keys, so reject.
	if strings.Contains(key, "//") {
		return false
	}
	for _, part := range strings.Split(key, "/") {
		if part == "." || part == ".." {
			return false
		}
	}
	return true
}
