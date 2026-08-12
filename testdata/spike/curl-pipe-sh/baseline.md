# curl-pipe-sh

## Simple string guard
- Sees: `curl` and `| sh`; often flags remote pipe patterns.
- Overlap possible on the string; Runmark adds structured remote/network/opaque facts.

## RunmarkFacts incremental
- `unknown` (remote content class), `external_network`, `opaque_script`; no invented local path touches.

## Notes
- Handwritten baseline for Spike differentiation.
- Not a verified claim about any specific third-party Guardrail version
  (e.g. cc-safety-net); see docs/research.md for category context only.
