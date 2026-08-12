# sensitive-dotenv-read

## Simple string guard
- Sees: `cat .env` and may flag `.env` by basename.
- Overlap possible; Runmark still emits structured `sensitive_path` + read touch.

## RunmarkFacts incremental
- `touches.read` includes workspace `.env`; `boundary.sensitive_path` true.

## Notes
- Handwritten baseline; control-ish sensitive marker case.
- Not a verified claim about any specific third-party Guardrail version
  (e.g. cc-safety-net); see docs/research.md for category context only.
