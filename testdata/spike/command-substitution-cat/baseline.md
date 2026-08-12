# command-substitution-cat

## Simple string guard
- Sees: `echo $(cat secret.txt)`; may flag `cat`/`secret.txt` substrings or miss substitution opacity.

## RunmarkFacts incremental
- Current Spike: non-blocking `command_substitution` unknowns do **not** set `facts.unknown`; touches stay empty.
- This case documents honest current projection, not a polished marketing claim.

## Notes
- Handwritten baseline documenting a known facts-projection gap (non-blocking unknowns).
- Not a verified claim about any specific third-party Guardrail version
  (e.g. cc-safety-net); see docs/research.md for category context only.
