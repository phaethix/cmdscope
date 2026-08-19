# command-substitution-cat

## Simple string guard
- Sees: `echo $(cat secret.txt)`; may flag `cat`/`secret.txt` substrings or miss substitution opacity.

## RunmarkFacts incremental
- `unknown`: true with reason `command_substitution`; touches stay empty because the substitution body is not expanded.
- This is the honest projection: the command's effects are not statically knowable, so it reports unknown rather than a clean "no impact".

## Notes
- Handwritten baseline documenting that a non-blocking `command_substitution` unknown still marks the facts as undetermined.
- Not a verified claim about any specific third-party Guardrail version
  (e.g. cc-safety-net); see docs/research.md for category context only.
