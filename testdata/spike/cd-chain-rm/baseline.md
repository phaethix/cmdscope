# cd-chain-rm

## Simple string guard
- Sees `cd sub` and `rm -rf .`; the `.` looks like the workspace root.
- Runmark resolves `.` against the post-cd directory: only `sub/` is at risk.

## RunmarkFacts incremental
- `delete: logical://workspace/sub`, not the workspace root; `destructive=true`.

## Notes
- Handwritten baseline for Spike differentiation.
- Not a verified claim about any specific third-party Guardrail version.
