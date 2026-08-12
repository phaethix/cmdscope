# python-c-opaque

## Simple string guard
- Sees: `python3 -c '…'`; may miss or invent file effects from Python source text.
- Risk: false certainty about `open("x").write` without a real Python analysis.

## RunmarkFacts incremental
- `unknown` + `opaque_script`; **no** invented write/delete touches for the `-c` body.

## Notes
- Handwritten baseline for Spike differentiation (honest opacity).
- Not a verified claim about any specific third-party Guardrail version
  (e.g. cc-safety-net); see docs/research.md for category context only.
