# sudo-rm-wrapper

## Simple string guard
- A guard keyed on `rm -rf` may fire but miss the elevation; one keyed on
  `sudo` misses the delete.
- Runmark strips the wrapper and reports both facts.

## RunmarkFacts incremental
- `delete: logical://workspace/build`; destructive flag; evidence keeps `sudo`.

## Notes
- Handwritten baseline for Spike differentiation.
- Not a verified claim about any specific third-party Guardrail version.
