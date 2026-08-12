// Package app hosts the runmark CLI use cases and wiring.
package app

import "io"

// Version is the runmark CLI semantic version.
// Keep it aligned with the schema_version release policy (0.x pre-release).
const Version = "0.1.0"

// PrintVersion emits the human-readable CLI line (not JSON) so hooks and
// scripts can parse a stable "runmark <semver>" prefix.
func PrintVersion(w io.Writer) error {
	_, err := io.WriteString(w, "runmark "+Version+"\n")
	return err
}
