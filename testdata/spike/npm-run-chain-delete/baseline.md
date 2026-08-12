# npm-run-chain-delete

## Simple string guard
- Sees: `npm run parent`.
- Likely misses: that `parent` → `child` → `rm -rf dist` via package.json scripts.

## RunmarkFacts incremental
- Chained expansion yields `touches.delete` for `dist`, `destructive`, and script entries.

## Notes
- Handwritten baseline for Spike differentiation versus a simple command-string guard.
- Not a verified claim about any specific third-party Guardrail version
  (e.g. cc-safety-net); see docs/research.md for category context only.
