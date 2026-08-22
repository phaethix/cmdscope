# sed-inplace

## Simple string guard
- Sees: `sed -i`; string guards may flag in-place edit as a write.
- Overlap possible on the `-i` flag; Runmark adds read+write on the same file.

## RunmarkFacts incremental
- `read` and `write` both set to `logical://workspace/file.txt` (in-place edit reads then rewrites).

## Notes
- Handwritten baseline for Spike differentiation.
- Not a verified claim about any specific third-party Guardrail version.
