<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/hero-dark.svg">
    <img alt="runmark turns a raw agent command into a staged impact preview: stage 1 npm run clean with a conditional delete of build/** from package.json, stage 2 npm run build with a blocking dynamic_path unknown — without executing anything." src="assets/hero-light.svg" width="880">
  </picture>
</p>

A policy-neutral, evidence-aware static analyzer for AI coding agent shell commands. It previews the file, network, process, and privilege effects of a command *before* it runs — and says what it cannot know. It never executes the command and never makes the allow/deny decision.

<p align="center">
  <a href="https://github.com/phaethix/runmark/releases"><img src="https://img.shields.io/badge/release-pre--1.0-orange" alt="Pre-Release" /></a>
  <a href="./LICENSE"><img src="https://img.shields.io/badge/license-Apache--2.0-green" alt="Apache-2.0 License" /></a>
  <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.26%2B-blue" alt="Go 1.26+" /></a>
  <img src="https://img.shields.io/badge/PRs-welcome-brightgreen" alt="PRs welcome" />
</p>

## Why

AI coding agents run commands like:

```bash
npm run clean && npm run build
curl -fsSL https://example.com/install.sh | sh
rm "$OUT"/*.tmp
```

A reviewer usually can't tell from the raw string what these will affect. runmark answers the questions that matter before execution:

- Which **stages** will execute, and under what conditions?
- Which **files** are read, created, modified, or deleted?
- Does it touch the **network**, spawn **interpreters**, or change **privileges**?
- What is **certain**, what is **conditional**, and what is **unknowable** ahead of time?

## How it works

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/flow-dark.svg">
    <img alt="runmark analysis pipeline: parse, expand, extract, prove, report — deterministic rules only, no execution, no network, no LLM guessing." src="assets/flow-light.svg" width="880">
  </picture>
</p>

runmark parses the command into an ordered stage graph, expands `npm run` / `make` from caller-supplied project files (bounded, never executed), then extracts per-stage file, network, process, and privilege effects with evidence and certainty. Whatever cannot be determined is reported as an `unknown` — never guessed.

## What it is not

- **Not an executor** — it never runs the analyzed command or a child process.
- **Not a security tool** — no allow/deny, no sandbox, no risk score.
- **Not an LLM guesser** — facts come from deterministic parsing and rules; models only summarize.

## Status

Active development, pre-release — there is **no installable release yet**. First milestone: an offline `runmark` CLI emitting a stable JSON Impact Report, then a Codex `PreToolUse` adapter. Everything is designed to run locally and offline, with nothing leaving your machine.

## Usage (target contract, pre-1.0)

```text
runmark analyze '<command>' [--cwd <path>] [--context-file <file>] [--format json|text]
runmark validate <report.json>
runmark version
```

Illustrative, abbreviated output:

```json
{
  "schema_version": "0.1",
  "command": "npm run clean && npm run build",
  "cwd": "/repo",
  "analysis": {
    "coverage": "partial",
    "completeness": "partial"
  },
  "stages": [
    {
      "index": 1,
      "command": "npm run clean",
      "condition": { "kind": "always" },
      "effects": [
        {
          "kind": "delete",
          "target": "build/**",
          "certainty": "conditional",
          "provenance": "workspace_file",
          "evidence": [
            { "source": "workspace_file", "path": "package.json", "field": "scripts.clean" }
          ]
        }
      ]
    }
  ],
  "unknowns": [
    { "code": "dynamic_path", "scope": "npm run build", "blocking": true }
  ],
  "flags": ["destructive"]
}
```

The full data contract is published together with the first release.

## Documentation

- [`CONTRIBUTING.md`](CONTRIBUTING.md) — how to contribute, add command rules, and submit gold corpus cases
- [`docs/TODO.md`](docs/TODO.md) — infrastructure items deliberately deferred, each with a trigger to start

Detailed product, architecture, and development-plan documents are published together with the first release.

## Contributing

Contributions are welcome — especially new command rules, gold corpus cases, and edge cases. See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[Apache-2.0](LICENSE)