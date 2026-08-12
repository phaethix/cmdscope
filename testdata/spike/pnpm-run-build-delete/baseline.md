# pnpm-run-build-delete

## Simple string guard
- Sees: `pnpm run build`.
- Likely misses: `scripts.build` = `rm -rf dist` hidden behind the script name.

## RunmarkFacts incremental
- Same class of increment as npm: delete `dist`, destructive, script entry from package.json.

## Notes
- Handwritten baseline for Spike differentiation versus a simple command-string guard.
- Not a verified claim about any specific third-party Guardrail version
  (e.g. cc-safety-net); see docs/research.md for category context only.
