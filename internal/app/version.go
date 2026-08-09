// Package app hosts the cmdscope CLI use cases and wiring.
package app

import "io"

// Version is the cmdscope CLI semantic version.
// Keep it aligned with the schema_version release policy (0.x pre-release).
const Version = "0.1.0"

// PrintVersion writes the fixed version string to w.
func PrintVersion(w io.Writer) error {
	_, err := io.WriteString(w, "cmdscope "+Version+"\n")
	return err
}
