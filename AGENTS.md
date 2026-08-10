# AGENTS.md — AI Collaboration Conventions

This file is read and followed by AI agents working in the cmdscope repository (Claude Code, Knot agents, Cursor, and other agents).

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