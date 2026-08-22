# find-delete

## Simple string guard
- Sees: `find ... -delete`; string guards may flag `-delete` as destructive.
- Overlap possible on the `-delete` token; Runmark adds a structured delete + glob unknown.

## RunmarkFacts incremental
- `delete: logical://workspace` (start point `.`) + `destructive` + `unknown` (glob_runtime_dependent from `'*.txt'`).

## Notes
- Handwritten baseline for Spike differentiation.
- Not a verified claim about any specific third-party Guardrail version.
