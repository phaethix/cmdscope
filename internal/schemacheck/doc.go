// Package schemacheck is reserved for JSON report schema validation and the
// gold-corpus fixture check. That validation is not implemented here: the
// repo policy forbids third-party JSON Schema libraries in production code, so
// the round-trip check against the on-disk schema lives in
// internal/ir's schema_roundtrip_test.go instead. This package only pins a
// compile-time dependency on the ir contract today.
package schemacheck

import "github.com/phaethix/runmark/internal/ir"

// Placeholder exists so schemacheck can pin this package in the import graph.
const Placeholder = ir.Placeholder
