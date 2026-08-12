# make-deploy-static

## Simple string guard
- Sees: `make deploy`.
- Likely misses: static Makefile recipe `rm -rf build`.

## RunmarkFacts incremental
- `touches.delete` for `build`, destructive, `scripts` tool=make from Makefile evidence.

## Notes
- Handwritten baseline for Spike differentiation versus a simple command-string guard.
- Not a verified claim about any specific third-party Guardrail version
  (e.g. cc-safety-net); see docs/research.md for category context only.
