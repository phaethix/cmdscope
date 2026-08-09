# Contributing to cmdscope

Thanks for your interest. cmdscope is a policy-neutral, evidence-aware static analyzer for AI coding agent shell commands: it previews file, network, process, and privilege effects before a command runs, and reports what it cannot know. It never executes the command and never makes allow/deny decisions.

Contributions are licensed under the same terms as the project — [Apache-2.0](LICENSE).

## Quick start

Prerequisites: Go 1.22+ and [git](https://git-scm.com/).

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

Allowed types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `build`, `ci`, `perf`, `style`. Keep each commit to one logical change. Maintainers may use `task-NN` as the scope for roadmap-driven work (for example `build(task-01): initialize go module and cli entrypoint`).

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

## Getting help

Open a discussion or issue for questions. Detailed product, architecture, and development-plan documents are published together with the first release.