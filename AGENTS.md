# AGENTS.md — AI Collaboration Conventions

This file is read and followed by AI agents working in the cmdscope repository (Claude Code, Knot agents, Cursor, and other agents).

## Comments: explain why, not what

**Mandatory for every code edit.** Full wording: `CONTRIBUTING.md` (Comments).

- Prefer **why**: design intent, constraints, non-obvious invariants, deliberate trade-offs, and what is intentionally *not* handled.
- Do **not** restate what the code already says (no paraphrasing signatures, no “X does Y” that the identifier already conveys, no empty const-group labels).
- **What is allowed** when the code alone is insufficient: wire/JSON contracts, span semantics, omitempty / `[]` vs `null` rules, and other invariants a reader cannot infer from the implementation.
- Do **not** put task/work-item numbers in comments.
- Do **not** cite document section numbers in code comments — no `§4.4`, `architecture §…`, `PRD §…`, or `CONTRIBUTING.md §…`. State the invariant inline; if a doc path is useful, name the file/heading in words without `§`.
- Before finishing an edit that adds comments, re-read them: if deleting a comment loses no intent, delete it.

## Dependencies and tests

**Mandatory.** Full wording: `CONTRIBUTING.md` (Dependencies and Testing).

- **Production code** defaults to the Go standard library. Any third-party import in non-test packages needs explicit maintainer approval and locked `go.mod`/`go.sum`.
- **Test files** (`*_test.go`) **must** use [`stretchr/testify`](https://github.com/stretchr/testify) for assertions. Prefer `require` (fail-fast); use `assert` only for intentional soft multi-checks.
- Do **not** write new verbose `if … { t.Fatalf/Errorf(...) }` assertion ladders. Keep `testing.T`, `t.Run`, table-driven tests, and `t.Helper`.
- Do **not** introduce a second assertion library, and do not use testify `mock`/`suite` unless a maintainer asks for it.

## Go style: idioms and modern stdlib

**Mandatory for every Go edit.** Full wording: `CONTRIBUTING.md` (Go style).

- Prefer **official Go idioms**: small focused types/helpers, composition, table-driven tests, errors as values, clear package boundaries. Do **not** invent ceremony (heavy options frameworks, unnecessary interfaces, mock/suite) when a function or small struct is enough.
- Prefer **current stdlib** available at the module’s `go` version (today `1.26+`), for example:
  - `cmp.Compare` / `cmp.Or` for multi-key ordering
  - `slices.SortStableFunc` / `slices.SortFunc` instead of `sort.Slice`
  - `strings.CutPrefix` / `CutSuffix` / `Cut` instead of `HasPrefix`+`TrimPrefix`
  - `strings.SplitSeq` / `FieldsSeq` when ranging over parts without keeping a slice
  - `encoding/hex` instead of `fmt.Sprintf("%x", …)` for digests
  - `math/rand/v2` in new tests instead of `math/rand`
  - `for i := range n` instead of `for i := 0; i < n; i++` when only the index is needed
- Before finishing a Go change, re-read the diff for older patterns above and replace them when the modern API is a drop-in improvement.
- Stay within the module `go` version in `go.mod`; do not require APIs newer than that floor unless the maintainer bumps it.

## Workspace Draft-Document Convention

### Location
- Any document whose content is **uncertain or under discussion** (drafts, review drafts, design discussions, ad-hoc analyses, etc.) must go into the `.issue` directory.
- Documents under `.issue` are treated as the "workspace discussion area" and are **not** final deliverables.

### File Naming Convention (mandatory)
Draft filenames follow the fixed format:

```
YYYY-MM-DD-HHMM-<name>.md
```

- The date and time are taken from the **actual creation time** of the file, in the format `YYYY-MM-DD-HHMM` (year-month-day, 4-digit hour-minute).
- `<name>` is a short identifier for the item, with words separated by hyphens, e.g. `go-review`.

### Example
- A file created on 2026-08-10 at 15:46 → `2026-08-10-1546-go-review.md`.

### Scope
- This naming rule applies **only** to uncertain/discussion documents (i.e. those placed in `.issue`).
- **Formal, conclusive deliverables** (finalized reviews, schemas, reports, contract documents, etc.) are **not** required to follow this naming convention; they may live in `docs/`, the repository root, or elsewhere unless otherwise agreed.

### Example Flow
1. An AI receives a request such as "analyze X and produce a document", and the result is not yet finalized → produce a draft.
2. Use the **current real time** as the filename time prefix and write to `.issue/`.
3. If `.issue` does not exist, create it automatically.
