# npm-run-cycle

## Simple string guard
- Sees: `npm run a`.
- Likely misses: cyclic script expansion; may treat as a normal npm invocation.

## RunmarkFacts incremental
- `unknown` with expansion-cycle style reason; `opaque_script`; no invented path touches.

## Notes
- Handwritten baseline for Spike differentiation.
- Not a verified claim about any specific third-party Guardrail version
  (e.g. cc-safety-net); see docs/research.md for category context only.
