# env-prefix-substitution

## Simple string guard
- Sees `rm -rf` plus an opaque `"$OUT"` token; string rules cannot resolve it.
- With caller-supplied env, Runmark substitutes and stays traceable.

## RunmarkFacts incremental
- `delete: logical://workspace/build`; provenance caller_context; no env_missing.

## Notes
- Handwritten baseline for Spike differentiation.
- Not a verified claim about any specific third-party Guardrail version.
