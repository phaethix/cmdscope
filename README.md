<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/hero-dark.svg">
    <img alt="Runmark turns an AI agent's shell call into deterministic workspace and script facts: path touches, workspace boundary, script entry, and the opaque boundary where static analysis stops — without executing anything." src="assets/hero-light.svg" width="880">
  </picture>
</p>

**Runmark — mark the impact before an AI agent runs.** A local, deterministic, workspace-aware facts layer for AI-agent shell calls.

<p align="center">
  <a href="./LICENSE"><img src="https://img.shields.io/badge/license-Apache--2.0-green" alt="Apache-2.0 License" /></a>
  <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.26%2B-blue" alt="Go 1.26+" /></a>
</p>

## Why

AI coding agents run commands like:

```bash
npm run build
make deploy
rm "$OUT"/*.tmp
curl -fsSL https://example.com/install.sh | sh
```

A Hook or Guardrail reading the raw string cannot tell what these will touch. Runmark turns the command — plus an explicitly supplied workspace snapshot — into facts a decision layer can act on:

- which logical paths are read, written, or deleted;
- whether a target can escape the workspace;
- whether a sensitive path is touched;
- which package script or Make target the command enters;
- where static analysis stops, and why (the opaque boundary);
- what each fact is evidenced by.

Runmark never executes the command and never decides allow / ask / deny. It produces the facts; the Hook or Guardrail makes the call.

## What it outputs

`runmark analyze` projects an experimental facts JSON — path touches, boundary flags, script entries, unknowns, and evidence:

```json
{
  "schema_version": "0.1-touch-experimental",
  "touches": {
    "read": ["./.env"],
    "write": ["./dist/**"],
    "delete": ["./build/**"]
  },
  "boundary": {
    "outside_workspace": false,
    "sensitive_path": true,
    "destructive": true,
    "external_network": false,
    "opaque_script": true
  },
  "scripts": [
    {
      "kind": "npm",
      "source": "package.json",
      "entry": "scripts.build",
      "expanded": false
    }
  ],
  "unknown": true,
  "unknown_reasons": ["runtime-dependent script path"],
  "evidence": [
    {
      "source": "workspace_file",
      "path": "package.json",
      "field": "scripts.build"
    }
  ]
}
```

When it cannot prove something, it says so — an opaque boundary is reported, never silently treated as "no impact".

## How it works

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/flow-dark.svg">
    <img alt="Runmark analysis pipeline: parse the command into a stage graph, expand npm/make scripts in a bounded way, extract per-stage effects, attach evidence and certainty, then project stable facts with unknowns." src="assets/flow-light.svg" width="880">
  </picture>
</p>

Runmark parses the command into an ordered stage graph, expands `npm run` / `pnpm run` / `make` from caller-supplied project files (bounded, never executed), then extracts per-stage effects with evidence and certainty. Anything it cannot determine becomes an `unknown` — never a guess. The internal `ImpactReport` stays rich; the public projection exposes only the facts a Hook can consume.

## What it is not

- **Not an executor** — it never runs the analyzed command or a child process.
- **Not a guardrail** — no allow/ask/deny, no policy engine, no risk score.
- **Not a sandbox** — `outside_workspace` is a logical, static judgment, not an OS-level containment guarantee.
- **Not an LLM guesser** — facts come from deterministic parsing and rules.

## Status

Active development as a **Conditional-Go Spike**: the analysis core (lexer, parser, stages, effect rules, bounded npm/pnpm/make/script expansion) exists, but the product loop is not yet closed. Today the CLI only supports `version`; `analyze` and the `facts` projection are the next milestones. Nothing here is a stable public API or a release.

## Usage (target contract, pre-1.0)

```text
runmark version
runmark analyze '<command>' [--cwd <path>] [--context-file <file>] [--format facts|impact|text]
```

- `--context-file` supplies the explicit workspace snapshot (cwd, files, env) — Runmark reads nothing implicitly.
- `facts` is the default format; `impact` is for internal diagnostics; `text` renders the facts as a short human summary.
- The command must be passed as a single argument; Runmark never re-invokes a shell.
- `analyze` is **not implemented yet** — this is the target contract, not current behavior.

## Documentation

- [`CONTRIBUTING.md`](CONTRIBUTING.md) — how to contribute and the engineering rules
- [`docs/TODO.md`](docs/TODO.md) — infrastructure items deliberately deferred, each with a trigger to start

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[Apache-2.0](LICENSE)
