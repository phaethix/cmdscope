# path-escape-dotenv

## Simple string guard
- Sees: `cat ../../.env`; may or may not model workspace containment.
- Likely weaker on logical outside-workspace judgment tied to analysis cwd.

## RunmarkFacts incremental
- `outside_workspace` + `sensitive_path` with read of escaped `.env` path.

## Notes
- Handwritten baseline for Spike differentiation versus a simple command-string guard.
- Not a verified claim about any specific third-party Guardrail version
  (e.g. cc-safety-net); see docs/research.md for category context only.
