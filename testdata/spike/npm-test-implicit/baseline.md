# npm-test-implicit

## Simple string guard
- Sees: `npm test`; string guards rarely expand npm lifecycle scripts.
- Overlap possible on the `npm` string; Runmark expands the lifecycle script from package.json.

## RunmarkFacts incremental
- Expands `test` → `rm -rf dist`: `delete: logical://workspace/dist` + `destructive`; script entry recorded with source `package.json`.

## Notes
- Handwritten baseline for Spike differentiation.
- Not a verified claim about any specific third-party Guardrail version.
