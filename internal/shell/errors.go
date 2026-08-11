package shell

import (
	"errors"
	"fmt"
)

// ParseErrKind discriminates why a parse failed. A caller maps a
// KindUnsupported error onto an unsupported_syntax unknown on the IR; a
// KindStructural error is a malformed command that cannot be lowered at all.
type ParseErrKind string

const (
	// KindStructural reports a malformed command stream (e.g. a redirect
	// missing its target, an unmatched grouping paren, an empty command).
	KindStructural ParseErrKind = "structural"

	// KindUnsupported reports syntax that is lexically valid but outside the
	// supported L0 surface (background '&', pipe-amp '|&', here-doc '<<').
	// It is a deliberate rejection that must surface as an unknown, never be
	// silently dropped or mis-read as a valid command.
	KindUnsupported ParseErrKind = "unsupported_syntax"
)

// ParseError reports a failed Parse and discriminates why, so a caller never
// has to string-match to decide whether an unsupported construct should
// surface as an unknown. It is a pointer type so the kind and offending span
// travel through any wrapping and resolve with a single type assertion.
type ParseError struct {
	Kind  ParseErrKind
	At    int    // byte offset of the offending token or construct
	Token string // lexical text of the offending token ("" when no token)
	Msg   string // human-readable reason for the caller/message
}

func (e *ParseError) Error() string {
	if e.Token != "" {
		return fmt.Sprintf("[%s] %q at byte %d: %s", e.Kind, e.Token, e.At, e.Msg)
	}
	return fmt.Sprintf("[%s] byte %d: %s", e.Kind, e.At, e.Msg)
}

// IsUnsupportedSyntax reports whether err rejects the input for living outside
// the L0 surface, by walking to the underlying *ParseError. Later stages rely
// on this to surface an unsupported_syntax unknown instead of failing silently.
func IsUnsupportedSyntax(err error) bool {
	pe, ok := errors.AsType[*ParseError](err)
	if !ok {
		return false
	}
	return pe.Kind == KindUnsupported
}
