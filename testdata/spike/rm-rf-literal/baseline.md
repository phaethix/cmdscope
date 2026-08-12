# rm-rf-literal

## Simple string guard
- Sees: `rm -rf dist` and typically flags destructive delete directly.
- Both sides should notice delete — control group, not differentiation proof.

## RunmarkFacts incremental
- `touches.delete` for `dist`, `destructive` true; confirms harness on a guard-visible case.

## Notes
- Handwritten baseline control case.
- Not a verified claim about any specific third-party Guardrail version
  (e.g. cc-safety-net); see docs/research.md for category context only.
