# Contributing to cmdscope

Thanks for your interest. cmdscope is a policy-neutral, evidence-aware static analyzer for AI coding agent shell commands: it previews file, network, process, and privilege effects before a command runs, and reports what it cannot know. It never executes the command and never makes allow/deny decisions.

Contributions are licensed under the same terms as the project — [Apache-2.0](LICENSE).

## Quick start

Prerequisites: Go 1.26+ and [git](https://git-scm.com/).

```bash
git clone git@github.com:phaethix/cmdscope.git
cd cmdscope
go test ./...          # run all tests
make check-schema      # validate schema, examples, and gold corpus (once CI assets exist)
```

## Reporting issues

- **Bugs** — use the [bug report template](.github/ISSUE_TEMPLATE/bug_report.md). Include the command analyzed, the context you supplied (cwd/platform/env/files), what you expected versus what you got, and whether an existing gold case regressed. Only include sanitized context — no secrets.
- **Feature ideas** — use the [feature request template](.github/ISSUE_TEMPLATE/feature_request.md). The proposal must fit the product boundary: static preview only, no command execution, no allow/deny, no auditing, no LLM-generated facts.

## Commit conventions

Use [Conventional Commits](https://www.conventionalcommits.org/):

```text
<type>(<scope>): <imperative summary>
```

Allowed types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `build`, `ci`, `perf`, `style`. Keep each commit to one logical change.

**Scope semantics.** The `scope` must name the **code module / domain** the change touches (`ir`, `app`, `schemachcheck`, `internal`, …) — never a task/work-item number. A change spanning the whole module layout but not one package may use `internal`; a change to documentation or build tooling that has no single code module should **omit** the scope (`docs:`, `build:`, `chore:`).

> Do **not** use `task-NN` (e.g. `feat(task-03)`) as a scope: task numbers are a roadmap/work-item concern, not a code-boundary one. Record the task reference in the commit body (for example a `Refs: task-03` footer) instead of the subject line.

## Development workflow

1. Branch from `master`; one focused change per pull request.
2. If the change touches core behavior, write or update tests in the same pull request.
3. Run locally before pushing:
   ```bash
   go test ./...
   go vet ./...
   test -z "$(gofmt -l .)"
   ```
4. Keep the change small. **No unrelated refactoring or scope creep** — if a bigger idea exists, open an issue first.

## Comments: explain why, not what

Comments must explain **why** — the design intent, constraint, or non-obvious invariant — and must **not** restate what the code already says. The code itself is the source of truth for *what*; repetition is noise that rots as the code changes.

Applies to all code, documentation, and commit messages in this repository, and is expected of every contributor and AI agent:

- Explain *why* (design reasoning, contract edge cases, trade-offs) and *what is intentionally not handled*.
- Do **not** paraphrase the implementation (e.g. a doc comment that re-spells the function signature).
- Do **not** reference task/work-item numbers in comments (see the `Refs:` footer note under commit conventions).
- Do **not** cite document section numbers in code comments (no `§4.4`, `architecture §…`, `PRD §…`, `CONTRIBUTING.md §…`). Write the invariant in the comment; if a doc pointer is needed, name the file or heading in words without `§`.
- Do **not** use comments to *explain what*; use them to make the non-obvious obvious.
- Before finishing, re-read every new/changed comment in the diff; delete or rewrite any that only narrate the next line of code.

## Dependencies

cmdscope does **not** treat “zero third-party modules” as a hard value. The rule is layered:

| Layer | Policy |
|---|---|
| Production (`*.go` excluding tests) | Default to the Go **standard library**. Any third-party import needs maintainer approval and must be locked in `go.mod` / `go.sum`. |
| Tests (`*_test.go`) | **Required:** [`github.com/stretchr/testify`](https://github.com/stretchr/testify) for assertions (already approved). |
| Forbidden | Unapproved production dependencies; adding a second assertion library. |

## Testing

All new and updated tests must use testify:

- Prefer `require` (fail-fast) for preconditions and primary expectations.
- Use `assert` only when continuing after a soft check is intentional.
- Keep `testing.T`, `t.Run`, table-driven structure, and `t.Helper`.
- Do not write verbose `if got != want { t.Fatalf(...) }` ladders for ordinary equality/error checks.
- Do not adopt testify `mock` or `suite` unless a maintainer explicitly asks.

## Go style

Write idiomatic Go that matches current standard-library guidance for the module’s `go` version (`go.mod`, currently **1.26+**). Agents and contributors are expected to use modern stdlib APIs when they are a clear improvement — not to pile on patterns for their own sake.

**Prefer**

- Small focused functions / unexported helper types (composition) over large procedural blobs
- Table-driven tests; stable deterministic helpers for ordering and IDs
- `cmp.Compare` and `cmp.Or` for multi-key comparisons
- `slices.SortStableFunc` / `slices.SortFunc` instead of `sort.Slice`
- `strings.CutPrefix` / `CutSuffix` / `Cut` instead of `HasPrefix`+`TrimPrefix` (or `HasSuffix`+`TrimSuffix`)
- `strings.SplitSeq` / `FieldsSeq` when iterating parts without needing a slice
- `encoding/hex` for digest hex encoding instead of `fmt.Sprintf("%x", …)`
- `math/rand/v2` in **new** tests instead of `math/rand`
- `range` over integers (`for i := range n`) where it replaces `for i := 0; i < n; i++`

**Avoid**

- Unnecessary interfaces, option-bag frameworks, or testify `mock`/`suite` without a concrete need
- Re-implementing stdlib (`filepath.Clean` for *logical* paths is wrong for this project; hand-rolled sorts when `slices` fits; dual `HasPrefix`+`TrimPrefix`)
- Using language/stdlib APIs newer than the `go` version in `go.mod` unless that floor is deliberately bumped

**Self-check before merge:** skim the diff for the “Avoid” list and replace drop-in cases with the “Prefer” APIs.

## Adding a command rule

New command rules are valuable and follow a fixed order (do not skip steps):

1. **Define the supported boundary** — which syntax and arguments your rule covers, and which it intentionally does not.
2. **Define the unknown rules** — how the rule behaves when input is dynamic, context is missing, or the command is unsupported. Unsupported commands must still produce a structured `unsupported_command` unknown, never an empty report.
3. **Add gold cases first** — see below. They document expected behavior and prevent regressions.
4. **Implement the extractor** — then make the new gold cases pass without breaking existing ones.
5. **Verify determinism** — the same input must produce byte-for-byte identical output on repeated runs.

## Contributing gold corpus cases

Gold cases are the project's most valuable contribution: every new command rule, edge case, and honest-unknown case improves everyone.

Each case lives in one directory:

```text
testdata/gold/<name>/
├── request.json      # command + analysis context
├── context.json      # caller-provided workspace files (package.json, Makefile, ...)
└── expected.json     # expected Impact Report
```

Particularly welcome:

- Commands whose raw string hides their impact (the "demo value" cases);
- Unsupported commands and syntax that must produce structured unknowns;
- **False-certainty candidates** — cases where an analyzer might claim too much certainty about an unknown branch;
- Platform-specific behavior (`darwin` vs `linux`);
- Boundary and regression cases.

In the pull request, say in one sentence why the case is included.

## Changing the JSON contract (schema)

- The schema is versioned (`schema_version`, currently `0.1`). During `0.x`:
  - **Adding an optional field** — allowed without a version bump;
  - **Adding a required field, removing a field, changing an enum, or changing semantics** — must bump the minor version and keep the old schema for existing reports.
- The report's own `schema_version` selects which schema validates it; never silently validate an old report against a new schema.
- Every contract change must update, in the same pull request: the Go IR, the schema file, `schema/examples/`, all affected gold `expected.json`, and pass `make check-schema` (which also runs `ValidateReport` at runtime, not just JSON Schema).

## Hard requirements (red lines)

The following are enforced by review and by CI:

- **Never execute the analyzed command.** No `os/exec`, shell, `npm`, `make`, or subprocess invocation anywhere in `internal/analyzer`, `internal/expand`, or `internal/effect`.
- **Never access the network** from core analysis.
- **Never read files that the caller did not explicitly provide** through the analysis context.
- **Never use an LLM to generate facts.** Contributors may only summarize; deterministic parsing and rules produce the facts.
- **Never convert a known unknown into an empty result**, and never raise certainty for the sake of a demo.
- **No allow/deny decisions in core.**
- **Deterministic, stable output** for identical inputs.

## Review process

Pull requests are reviewed by a project maintainer (see `CODEOWNERS` for path ownership). CI must pass, including full test suite and schema checks. Feedback may ask you to add or adjust gold cases — that is expected, not optional.

## Workspace draft-document convention

Any document whose content is **uncertain or under discussion** (review drafts, design discussions, ad-hoc analyses, etc.) goes into the repository's `.issue` directory, treated as the workspace discussion area — not a final deliverable.

**File naming (mandatory)**: `<date>-<file-creation-time>-<name>.md` (i.e. `YYYY-MM-DD-HHMM-<name>.md`), where the date-time is the file's actual creation time in `YYYY-MM-DD-HHMM` format. Example: `2026-08-10-1546-go-review-2026.md` (created 2026-08-10 15:46; name `go-review-2026`).

**Formal, conclusive deliverables** (finalized schemas, reports, etc.) are not bound by this naming rule. This convention mirrors the instruction given to AI agents in the root `AGENTS.md`.

## Getting help

Open a discussion or issue for questions. Detailed product, architecture, and development-plan documents are published together with the first release.