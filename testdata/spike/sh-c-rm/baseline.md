# sh-c-rm

## Simple string guard
- Sees: `sh -c 'rm -rf build'` and may catch `rm -rf` in the string.
- Partial overlap: literal guard can work here; Runmark still normalizes the delete touch.

## RunmarkFacts incremental
- Wrapper expansion yields `touches.delete` for `build` with provenance from the `-c` body.

## Notes
- Handwritten baseline; useful as wrapper coverage even when a string guard might also fire.
- Not a verified claim about any specific third-party Guardrail version
  (e.g. cc-safety-net); see docs/research.md for category context only.
