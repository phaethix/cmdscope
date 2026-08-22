# git-rm

## Simple string guard
- Sees: `git rm file.txt`; string guards may flag `rm` as destructive delete.
- Overlap possible on the `rm` token; Runmark adds a structured delete on the path.

## RunmarkFacts incremental
- `delete: logical://workspace/file.txt` + `destructive`; no network (plain rm).

## Notes
- Handwritten baseline for Spike differentiation.
- Not a verified claim about any specific third-party Guardrail version.
