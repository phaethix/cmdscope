# npm-run-build-delete

## Simple string guard
- Sees: literal `npm run build`.
- Likely misses: that `scripts.build` expands to `rm -rf dist`, so delete/destructive impact is hidden behind the script name.

## RunmarkFacts incremental
- `touches.delete`: `logical://workspace/dist`
- `boundary.destructive`: true
- `scripts`: npm / build / package.json
- Evidence includes `workspace_file` `scripts.build` snippet `rm -rf dist`

## Notes
- Handwritten baseline for Spike differentiation versus a simple command-string guard.
- Not a verified claim about any specific third-party Guardrail version
  (e.g. cc-safety-net); see docs/research.md for category context only.
