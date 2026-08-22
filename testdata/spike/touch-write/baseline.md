# touch-write

## Simple string guard
- Sees: `f.txt`; a filename string match says little about the action.
- Runmark resolves the action: touch creates or updates the file.

## RunmarkFacts incremental
- `write: logical://workspace/f.txt`; no destructive or network flag.

## Notes
- Handwritten baseline for Spike differentiation.
- Not a verified claim about any specific third-party Guardrail version.
