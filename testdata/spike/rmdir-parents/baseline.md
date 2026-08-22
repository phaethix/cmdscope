# rmdir-parents

## Simple string guard
- String guards may match `rmdir` but miss the option semantics.
- Runmark keeps the delete target even with `--parents` present.

## RunmarkFacts incremental
- `delete: logical://workspace/old/nested`; `destructive=true`.

## Notes
- Handwritten baseline for Spike differentiation.
- Not a verified claim about any specific third-party Guardrail version.
