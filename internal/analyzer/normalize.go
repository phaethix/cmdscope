package analyzer

import "strings"

// normalizeCommand performs the pure semantics-preserving normalization that
// every pipeline stage relies on (architecture, "Normalize"). Only line
// endings are unified so that the lexer's UTF-8 byte spans are deterministic
// regardless of CRLF/CR/LF input; the rest of the command text is preserved
// verbatim so the reconstructed stage commands and evidence spans point at the
// same bytes the analyzer walks.
//
// It deliberately does not expand globs, substitute unprovided environment
// variables, or drop shell operators: those are later, narrower jobs and must
// never happen here.
func normalizeCommand(command string) string {
	if !strings.ContainsRune(command, '\r') {
		return command
	}
	command = strings.ReplaceAll(command, "\r\n", "\n")
	return strings.ReplaceAll(command, "\r", "\n")
}
