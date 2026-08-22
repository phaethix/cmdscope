# tee-pipe

## Simple string guard
- Sees: `tee out.txt`; string guards rarely model stdin pipelines precisely.
- Overlap possible on the filename string; Runmark adds a structured write fact.

## RunmarkFacts incremental
- `write: logical://workspace/out.txt`; no destructive or network flag.

## Notes
- Handwritten baseline for Spike differentiation.
- Not a verified claim about any specific third-party Guardrail version.
