# make-dynamic-include

## Simple string guard
- Sees: `make deploy`.
- Likely misses / overclaims: may ignore include dynamism or assume the visible recipe is complete.

## RunmarkFacts incremental
- Reports unknown (unsupported/dynamic Makefile); does **not** invent delete touches from the recipe text when include makes the Makefile non-static.

## Notes
- Handwritten baseline for Spike differentiation.
- Not a verified claim about any specific third-party Guardrail version
  (e.g. cc-safety-net); see docs/research.md for category context only.
