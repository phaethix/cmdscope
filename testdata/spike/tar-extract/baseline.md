# tar-extract

## Simple string guard
- Sees: `tar -xzf ... -C /dest`; string guards rarely resolve `-C` targets.
- Overlap possible on the `tar` string; Runmark adds a structured write on the `-C` dir.

## RunmarkFacts incremental
- `write: /dest` + `outside_workspace` (absolute path escapes logical root).

## Notes
- Handwritten baseline for Spike differentiation.
- Not a verified claim about any specific third-party Guardrail version.
