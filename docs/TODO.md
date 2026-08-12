# Infrastructure TODO — Deferred Second-Batch Items

> Status: planned / deferred
> Created: 2026-08-09
> Owner: maintainer (phaethix)
> Related: [`docs/research.md`](research.md), [`docs/architecture.md`](architecture.md), and the first-batch repo scaffold design (community-facing files: `CONTRIBUTING.md`, PR/issue templates, `SECURITY.md`, `CODEOWNERS`, CI, `Makefile`, `.editorconfig`).

This file records project-infrastructure items deliberately **deferred** from the first batch (the "minimal necessary set" that keeps the project aligned with the architecture red lines). Each item stays open until its trigger condition is met. Do **not** implement these just because they are listed — each must earn its place as the project/community matures.

---

## Open items

### 1. `CODE_OF_CONDUCT.md`
- **What:** Contributor Covenant CoC, surfaced by GitHub in the repo header.
- **Why deferred:** Single-maintainer phase; no community heavy enough yet to need explicit behavioral policy.
- **Trigger:** First external contributor / active community forms, or repo goes public with real traffic.
- **Notes:** Add a link to it from `CONTRIBUTING.md` the day it lands.

### 2. `SUPPORT.md`
- **What:** Official support channels statement.
- **Why deferred:** No support channels or SLAs exist yet; writing one now would be speculative.
- **Trigger:** Once a `#`-channel / Discussions / issue-response turnaround policy is actually ready to commit to.
- **Notes:** Small file; combine with `README` "Support" section when written.

### 3. GitHub labels taxonomy
- **What:** A curated label set (e.g. `good first issue`, `help wanted`, `gold-case`, `schema`, `adapter/codex`, `documentation`, `priority:p0..`) + `config.yml` for the issue chooser.
- **Why deferred:** Labels are only useful when there is a steady flow of issues/PRs; also, the exact categories should follow the first real triage experience, not guesswork.
- **Trigger:** First external issue traffic, or when the gold-corpus contribution path opens up to outsiders.
- **Notes:** PRD §7.3 item 6 makes "contributing gold cases" a first-class goal — `gold-case` label will likely be one of the first to add.

### 4. Dependabot
- **What:** Automated updates for GitHub Actions (`actions/*`) and `go.mod`/`go.sum`.
- **Why deferred:** Actions usage is still minimal; Dependabot can wait until the Actions set is stable. `go.mod` already locks testify (test-only) and will grow with approved production deps.
- **Trigger:** After CI Actions usage is non-trivial, or the first production third-party dependency lands.
- **Notes:** Architecture §7.5 requires locked dependency versions committed to `go.mod`/`go.sum`; Dependabot must respect that (no spontaneous unpinned bumps).

### 5. GoReleaser (release pipeline)
- **What:** Cross-platform binaries (linux/darwin), Homebrew tap, semver tags + release notes.
- **Why deferred:** No CLI binary that users can install yet (that's roadmap Phase 8+); a release pipeline before a shippable artifact is premature.
- **Trigger:** Roadmap Gate E+/F passes, or the first "installable" milestone (the first CLI exists and the README demo is reproducible).
- **Notes:** Follow semantic versioning; keep `0.x` policy consistent with `schema_version` bump rules.

### 6. `CHANGELOG.md`
- **What:** Keep-a-Changelog style changelog.
- **Why deferred:** No releases yet; a changelog with zero entries is dead weight, and it would churn against roadmap-numbered commits.
- **Trigger:** First versioned release or first breaking `schema_version` change.
- **Notes:** Could be auto-generated from release notes later instead of hand-maintained.

### 7. ADR directory (`docs/adr/`)
- **What:** Architecture Decision Records for major choices (schema version policy, platform handling, adapter selection, path normalization, etc.).
- **Why deferred:** The public architecture guide in [`docs/architecture.md`](architecture.md) already captures the current design; ADRs would currently duplicate it.
- **Trigger:** First decision that *changes* the architecture doc, or the first contested design discussion with contributors.
- **Notes:** When started, make each ADR one paragraph summary + decision + status; do not backfill historical decisions.

### 8. pre-commit / commit-msg hooks
- **What:** Local hooks (gofmt, goimports, lint, commitlint) so the commit standards are enforced by tooling, not by memory.
- **Why deferred:** Minimal batch keeps the barrier to cloning/contributing low; CI already enforces the gates, and hooks can annoy contributors before a community exists.
- **Trigger:** After the first external contributor PR, or once the CI gate set feels too slow for fast feedback.
- **Notes:** If adopted, make hooks optional (documented setup) — never required to merge.

### 9. CLI library decision (third-party CLI framework)
- **What:** Decide whether to adopt a Go CLI library — `spf13/cobra` (+`pflag`), `urfave/cli`, or `alecthomas/kong` — versus keeping the current **standard-library `flag`** per architecture §2.1 ("CLI args prefer stdlib flag; avoid introducing large frameworks early").
- **Why deferred:** The CLI surface is tiny (`version`, `analyze`/`validate` + ~6 flags), production code defaults to stdlib, and a mature CLI library is not yet justified by user need. Third-party **production** dependencies need explicit maintainer approval (test-only deps such as testify are already allowed).
- **Trigger:** First concrete user need (help UX, shell completion, flags-after-positional ordering) that `stdlib flag` cannot satisfy reasonably, or the first installable binary demo milestone, whichever comes first.
- **Notes:** When closing, attach the shortlisted data (popularity/maintenance/interspersed-flag tradeoffs) to a new ADR, e.g. `docs/adr/0009-cli-library.md`. Priority ordering to weigh at that time: `kong` (declarative, lightweight) ≈ `pflag` (POSIX, minimal) over `cobra` unless subcommand-tree + completions earn the weight.

---

## Rules for closing items

1. Move the item to the repo's normal backlog (or its own design doc) the moment you decide to start it — this file only records the decision to defer.
2. Each item must reference the issue/task or design doc it was started in, and be removed from this list when done.
3. Re-evaluate quarterly, or whenever the project's community/scale status changes.