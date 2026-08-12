# echo-redirect-write

## Simple string guard
- Sees: `echo hi > out.txt` and may flag a write to `out.txt` from the redirect token.
- Likely misses: nothing critical on this smoke case; both sides can notice a write.

## RunmarkFacts incremental
- Normalized touch: `write: logical://workspace/out.txt` with command evidence.
- Useful as harness self-check, not as differentiation proof.

## Notes
- Handwritten baseline for Spike differentiation.
- Not a verified claim about any specific third-party Guardrail version
  (e.g. cc-safety-net); see docs/research.md for category context only.
