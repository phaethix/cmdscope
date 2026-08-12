<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/hero-dark.svg">
    <source media="(prefers-color-scheme: light)" srcset="./assets/hero-light.svg">
    <img alt="Runmark turns an AI agent's shell call into deterministic workspace and script facts: path touches, workspace boundary, script entry, and the opaque boundary where static analysis stops — without executing anything." src="./assets/hero-light.svg" width="880">
  </picture>
</p>

**Runmark — mark the impact before an AI agent runs.**

A local, deterministic, workspace-aware facts layer for AI-agent shell calls.

Runmark analyzes what an agent's shell call may touch before execution. It does not execute commands, enforce policy, provide a sandbox, or make allow / ask / deny decisions.

<p align="center">
  <a href="https://github.com/phaethix/runmark/actions/workflows/ci.yml"><img src="https://github.com/phaethix/runmark/actions/workflows/ci.yml/badge.svg?branch=master" alt="CI status" /></a>
  <a href="./LICENSE"><img src="https://img.shields.io/badge/license-Apache--2.0-green" alt="Apache-2.0 License" /></a>
  <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.26%2B-blue" alt="Go 1.26" /></a>
</p>

## Why

AI coding agents run commands like:

```bash
npm run build
make deploy
rm "$OUT"/*.tmp
curl -fsSL https://example.com/install.sh | sh
```

The raw command is not always enough for a Hook or Guardrail to determine the command's workspace and script effects reliably. Runmark combines the command with an explicitly supplied workspace snapshot and produces deterministic facts a decision layer can act on:

- which logical paths are read, written, or deleted;
- whether a target can escape the workspace;
- whether a sensitive path is touched;
- which package script or Make target the command enters;
- where static analysis stops, and why (the opaque boundary);
- what each fact is evidenced by.

Runmark never executes the command and never decides allow / ask / deny. It produces the facts; the Hook or Guardrail makes the call.

## Name clarification

Runmark is an AI-agent shell analysis project. It is not the HTTP workflow runner published as `@exit-zero-labs/runmark`, which focuses on tracked HTTP workflows, request execution, and MCP-based workflow operations.

Runmark focuses on pre-execution facts for AI-agent shell calls:

- workspace path touches;
- script entry expansion;
- opaque analysis boundaries;
- deterministic evidence.

## Experimental output

`runmark analyze` will project an experimental facts document — path touches, boundary flags, script entries, unknowns, and evidence:

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

This example is illustrative. The current CLI does not yet implement `analyze`; it describes the target Spike contract, not a stable public API.

## How it works

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/flow-dark.svg">
    <source media="(prefers-color-scheme: light)" srcset="./assets/flow-light.svg">
    <img alt="Runmark analysis pipeline: parse the command into a stage graph, expand npm/make scripts in a bounded way, extract per-stage effects, attach evidence and certainty, then project stable facts with unknowns." src="./assets/flow-light.svg" width="880">
  </picture>
</p>

Runmark is designed as a two-layer system:

```text
Shell call
    ↓
Internal ImpactReport
    ↓
Experimental RunmarkFacts projection
    ↓
CLI / Hook context
```

It parses the command into an ordered stage graph, expands `npm run` / `pnpm run` / `make` from caller-supplied project files (bounded, never executed), then extracts effects with evidence and certainty. Anything it cannot determine becomes an `unknown` — never a guess. The internal `ImpactReport` stays rich; the experimental projection exposes only the facts currently needed by Hook integrations.

## What it is not

- **Not an executor** — it never runs the analyzed command or a child process.
- **Not a guardrail** — no allow/ask/deny, no policy engine, no risk score.
- **Not a sandbox** — `outside_workspace` is a logical, static judgment, not an OS-level containment guarantee.
- **Not an audit system** — it does not observe or record post-execution behavior.
- **Not an LLM guesser** — facts come from deterministic parsing and rules.
- **Not a complete Shell interpreter** — unsupported or dynamic behavior is reported as unknown.

## Status

Runmark is currently an early **Conditional-Go Spike**.

The repository already contains the analysis building blocks:

- Shell lexer and parser;
- stage splitting;
- file and command effect rules;
- bounded npm/pnpm/make/script expansion;
- unknown and evidence primitives.

The product loop is not closed yet:

- analyzer orchestration is incomplete;
- `analyze` is not implemented;
- the experimental facts projection is not implemented;
- no external Hook integration has been validated.

There is no stable public API or installable release yet.

## Who this is for

Runmark is intended for developers building:

- Agent PreToolUse Hooks;
- coding-agent Guardrails;
- approval or review layers;
- local agent infrastructure;
- shell-aware policy and verification tools.

Runmark is not primarily intended to be a standalone natural-language command explainer.

## Usage target (pre-1.0)

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
