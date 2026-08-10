package ir

import (
	"encoding/json"
	"fmt"
)

// Validation error codes returned before analysis produces an ImpactReport.
const (
	ErrCodeEmptyCommand        = "empty_command"
	ErrCodeInputTooLarge       = "input_too_large"
	ErrCodeInvalidCWD          = "invalid_cwd"
	ErrCodeInvalidContextPath  = "invalid_context_path"
	ErrCodeInvalidContextField = "invalid_context_field"
	ErrCodeContextFileTooLarge = "context_file_too_large"
	ErrCodeInvalidContextJSON  = "invalid_context_json"
)

// ValidationError is a stable request validation failure for CLI and core callers.
type ValidationError struct {
	Code    string `json:"error_code"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// MarshalJSON emits the stderr JSON shape required by the product contract.
func (e *ValidationError) MarshalJSON() ([]byte, error) {
	type alias ValidationError
	return json.Marshal(alias(*e))
}

// NewValidationError constructs a validation error with a stable code.
func NewValidationError(code, message string) *ValidationError {
	return &ValidationError{Code: code, Message: message}
}
