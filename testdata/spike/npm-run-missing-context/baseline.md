# npm-run-missing-context

## Simple string guard
- Sees: `npm run build` and may allow or deny on the npm token alone.
- Likely misses: whether workspace script content was available; may treat "no matched dangerous string" as safe.

## RunmarkFacts incremental
- `unknown`: true with reason `context_missing`
- `boundary.opaque_script`: true
- Empty touches — does not invent a clean "no impact" result when package.json was not supplied

## Notes
- Handwritten baseline for Spike differentiation.
- Not a verified claim about any specific third-party Guardrail version
  (e.g. cc-safety-net); see docs/research.md for category context only.
