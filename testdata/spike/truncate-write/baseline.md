# truncate-write

## Simple string guard
- Sees: `-s 0` and `f.txt` as loose tokens.
- Runmark resolves that the named file is truncated to zero.

## RunmarkFacts incremental
- `write: logical://workspace/f.txt`.

## Notes
- Handwritten baseline for Spike differentiation.
- Not a verified claim about any specific third-party Guardrail version.
