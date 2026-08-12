# Why Runmark exists

> Public research and product rationale for Runmark.
>
> Research snapshot: 2026-08-12. This document records the reasoning behind Runmark's scope. It is not a market-size report and does not claim that the product direction has already been validated.

## 1. Executive decision

AI coding agents are becoming execution systems. They do more than generate text: they run Shell commands, invoke project scripts, read configuration, access networks, and modify repositories.

Runmark explores one narrow layer in that execution path:

> **Before an AI agent runs a Shell call, produce deterministic, evidence-backed facts about the workspace paths and script boundaries that the call may touch.**

Runmark is deliberately not a complete Agent security platform. It does not provide a sandbox, enforce policy, replace approval UX, observe runtime syscalls, or decide whether an action is allowed.

The product hypothesis is:

> **Hook and Guardrail authors may benefit from a reusable local Shell-analysis core instead of independently rebuilding Shell parsing, bounded project-script expansion, path normalization, unknown handling, and evidence tracking.**

This is a hypothesis, not a proven market fact. Runmark should earn a broader public API or release only after a real integration demonstrates additional value over a simple command guard or an existing Guardrail.

## 2. The problem

### 2.1 Agents act through commands

A coding agent may produce commands such as:

```bash
npm run build
make deploy
rm "$OUT"/*.tmp
curl -fsSL https://example.com/install.sh | sh
```

The visible command is sometimes only an entry point. Its behavior may be hidden in:

- `package.json` scripts;
- Makefile targets and dependencies;
- Shell wrappers and interpreter one-liners;
- redirects, pipelines, command substitutions, and variables;
- runtime-dependent paths;
- remote content that cannot be inspected offline.

A reviewer or decision layer needs more than a generic label such as “safe” or “dangerous.” It needs to know what the command expresses, what can be proven from the supplied context, and where the analysis must stop.

### 2.2 The hidden-effect examples

With an explicitly supplied `package.json`, a bounded analyzer can reason about:

```text
npm run build
  -> package.json:scripts.build
  -> known file touches are retained
  -> a dynamic runtime path is marked opaque
```

For a remote script:

```text
curl https://example.com/install.sh | sh
  -> network destination: known
  -> interpreter receiving remote content: known
  -> remote file effects: unknown
```

The correct behavior is not to invent the remote script's installation paths. The correct behavior is to report the known facts and preserve an explicit unknown boundary.

## 3. Why existing controls are not the same problem

The agent-execution ecosystem contains several different control layers. They should not be treated as one product category.

| Layer | Question | Typical mechanisms |
|---|---|---|
| Approval | Should this action proceed? | Manual approval, permission rules, auto-approval modes |
| Isolation | What can the action physically reach? | Containers, VMs, sandboxes, directory and network boundaries |
| Guardrail | Which actions should be blocked or escalated? | PreToolUse Hooks, command guards, policy engines |
| Verification | What can be proven before side effects? | Static verification, path containment, taint, provenance |
| Preview | What should a human review before proceeding? | Diffs, changesets, impact summaries |
| Audit | What happened after execution? | Session logs, replay, syscall observation, reports |

Runmark belongs near **verification**, with a small preview renderer as an optional consumer. It does not replace the other layers.

The intended composition is:

```text
Runmark facts
    ↓
Hook / Guardrail / approval layer
    ↓
allow, review, deny, or sandbox decision
```

This separation is important. Facts can be reused by multiple policies, while a policy decision requires a threat model, user intent, client semantics, and runtime context that the analysis core does not own.

## 4. What the research says

### 4.1 Stronger signals: control and containment

Across public developer discussions and open-source projects, the strongest recurring signals concern:

- approval fatigue and repetitive confirmation;
- running agents in containers or sandboxes;
- limiting directory and network access;
- deterministic PreToolUse or command guards;
- protecting `.env`, SSH keys, tokens, and other sensitive paths;
- code, infrastructure, and changeset diffs;
- post-execution logs and replay.

These signals validate the broader engineering problem: developers want agents to be productive without giving them uncontrolled access to important resources.

They do not, by themselves, prove demand for a standalone static facts component.

### 4.2 Weaker signal: standalone command-impact preview

Direct requests for a complete Shell impact report are much less common. The closest public requests tend to ask for:

- blast radius before a migration or permission change;
- options and a recommendation before a high-impact action;
- visibility into unexpected files or secret reads;
- safer handling of commands whose real behavior is hidden in scripts.

These signals support a narrow experiment, not a broad claim that developers are actively searching for a new Effect IR product.

### 4.3 The key distinction

The following statements are not equivalent:

```text
Users want fewer approval prompts.
Users want a sandbox.
Users fear rm -rf.
Users want a command-impact analyzer.
Users want a reusable JSON facts core.
```

The first three can be true without the last two being true. Runmark therefore treats adoption by Hook and Guardrail builders as the decisive validation question.

## 5. Evidence discipline

The research combines public sources of different quality:

- official client documentation for permissions, Hooks, and sandboxing;
- official repositories, package registries, and release metadata;
- Hacker News discussions and Show HN projects;
- Stack Overflow questions;
- developer discussions about approvals, Hooks, sandboxes, and unexpected effects;
- research papers and benchmark repositories.

Each source answers different questions:

| Source type | Useful for | Not sufficient for |
|---|---|---|
| Official documentation | API contracts and stated product capabilities | User adoption or market size |
| GitHub repositories | Implementation scope, activity, integration surface | Real user count or commercial demand |
| Package registries | Publication, version, package-name collisions | Why users adopt a tool |
| HN, X, Reddit | Qualitative pain points and solution signals | General population estimates |
| Stack Overflow | Concrete developer questions | Proof that no unasked need exists |
| Stars, points, downloads | Visibility and distribution clues | Product-market fit |

The following claims are intentionally not made:

- that Runmark has a validated market;
- that a complete Effect IR is a demanded public standard;
- that users will install a separate CLI instead of using a built-in client control;
- that unknowns automatically increase trust;
- that an existing Guardrail cannot implement the same feature;
- that a single research paper or community post proves product demand.

## 6. Competitive and adjacent landscape

Runmark must be evaluated by function, not only by project name.

### 6.1 Direct baseline: existing command Guardrails

Projects such as [cc-safety-net](https://github.com/kenryu42/cc-safety-net) establish a high bar. Their public materials describe multi-agent Hooks, semantic command analysis, wrapper and interpreter handling, path containment, sensitive-file protection, explanations, and logs.

This changes Runmark's question. It is not enough to ask:

> Can Runmark analyze Shell commands before execution?

The useful question is:

> Can Runmark provide a reusable workspace/script facts core that existing Guardrails would rather consume than rebuild?

### 6.2 Runtime and policy systems

Projects such as [agentjail](https://github.com/LuD1161/agentjail) and similar runtime-control tools combine Hooks with policy, sandboxing, network controls, credentials, or audit. These systems are complements as well as potential consumers.

Runmark should not become a smaller copy of them. Their value is enforcement and containment; Runmark's proposed value is evidence-backed Shell facts.

### 6.3 Static verification projects

[OpenCode Guardians](https://github.com/albertjoseph0/opencode-plugin-guardians) demonstrates a technically adjacent pattern: static verification connected to a pre-execution Hook, including path and secret-related checks.

[CARE](https://arxiv.org/abs/2607.21642) is a directly adjacent research system for pre-execution command verification. Public discussion observed around it does not establish production adoption, but the technical overlap means Runmark should not claim that “pre-execution verification” is an empty category.

### 6.4 Command explainers and observers

[sheer](https://github.com/Etirf/sheer) and other command-explanation tools show that natural-language command explanation and pipeline decomposition are not sufficient differentiation.

Post-execution observation tools answer a different question: what actually happened after a command ran. Runmark's intended position is before execution and explicitly static.

## 7. The narrow product thesis

Runmark should focus on the combination of capabilities that a higher-level Hook may not want to rebuild:

1. Shell structure and ordered stages;
2. explicit workspace context;
3. bounded `npm`, `pnpm`, and `make` expansion;
4. path-touch extraction for read/write/delete;
5. logical workspace-boundary facts;
6. opaque boundaries for dynamic or remote behavior;
7. evidence linked to command spans or workspace fields;
8. deterministic output suitable for another local tool.

The differentiator is not a larger dangerous-command list. It is not the word “static.” It is not a security score.

The differentiator must be demonstrated as a lower-cost or more useful reusable analysis component.

## 8. Why the scope is intentionally narrow

A complete Agent security platform would need to address many separate concerns:

- capability and sandbox enforcement;
- credentials and secret brokering;
- network policy;
- prompt-injection resistance;
- approval UX;
- runtime observation;
- audit retention;
- policy evaluation;
- client-specific protocols.

Those concerns require different threat models and different operational systems. Including them would make Runmark harder to reason about and would weaken its most defensible property: a local, deterministic, reviewable analysis core.

Runmark therefore follows these boundaries:

- facts, not policy;
- logical analysis, not OS isolation;
- static evidence, not runtime observation;
- explicit context, not implicit filesystem scanning;
- unknowns, not invented certainty;
- one integration at a time, not a client matrix before adoption.

## 9. Validation plan

The project should be evaluated through a small, reproducible integration experiment.

### 9.1 Technical comparison

Compare Runmark with a simple string guard or the existing implementation in a Guardrail on cases such as:

```bash
npm run build
make deploy
rm "$OUT"/*.tmp
cat ../../.env
curl URL | sh
```

Measure whether Runmark can expose facts that the baseline misses:

- indirect script entry points;
- workspace path touches;
- logical path escape;
- sensitive path access;
- opaque runtime behavior;
- source evidence for each conclusion.

### 9.2 Integration comparison

A real Hook or Guardrail should be able to consume the result without a large amount of glue code. The integration should record:

- how the input context is supplied;
- how long analysis takes;
- which facts are actually consumed;
- how unknown and opaque results are handled;
- whether the facts change an allow, review, deny, or escalation decision;
- whether the integration is simpler than implementing the same logic locally.

### 9.3 Decision rule

Runmark should expand only if an external integration demonstrates all of the following:

- a meaningful fact that a simpler baseline misses;
- evidence that explains the fact;
- acceptable latency and noise;
- a clear consumer for the output;
- a reason not to simply duplicate the feature in an existing Guardrail.

If existing projects cover the same cases at lower cost, or no external developer wants to integrate the component, the repository can remain a valuable Shell-analysis research project without becoming a larger product.

## 10. Current public contract

The public contract is intentionally experimental and focuses on:

- local analysis;
- Shell structure;
- bounded project-script expansion;
- workspace path touches;
- logical boundaries;
- opaque behavior;
- evidence-backed unknowns;
- one client integration at a time.

It does not promise:

- complete Shell semantics;
- runtime behavior observation;
- security guarantees;
- a sandbox;
- a policy decision;
- a stable schema before integration evidence;
- an installable release at the current stage.

## 11. References

### Official client and platform documentation

- [OpenAI Codex Hooks](https://developers.openai.com/codex/hooks)
- [OpenAI: Running Codex safely](https://openai.com/index/running-codex-safely)
- [Claude Code permissions](https://code.claude.com/docs/en/permissions)
- [Claude Code sandboxing](https://code.claude.com/docs/en/sandboxing)
- [Google agent-shell-tools](https://github.com/google/agent-shell-tools)
- [OpenHands](https://github.com/All-Hands-AI/OpenHands)

### Open-source projects and package references

- [cc-safety-net](https://github.com/kenryu42/cc-safety-net)
- [shellfirm](https://github.com/kaplanelad/shellfirm)
- [agentjail](https://github.com/LuD1161/agentjail)
- [Prismor](https://github.com/PrismorSec/prismor)
- [OpenCode Guardians](https://github.com/albertjoseph0/opencode-plugin-guardians)
- [MVAR](https://github.com/mvar-security/mvar)
- [Aperion Shield](https://github.com/AperionAI/shield)
- [PIC Standard](https://github.com/pic-standard/pic-standard)
- [ToolPermit](https://github.com/sunhao123456sun-svg/toolpermit)
- [sheer](https://github.com/Etirf/sheer)
- [Agent Polis impact preview](https://github.com/agent-polis/impact-preview)
- [Public post-execution command observability project](https://github.com/sanromarth/cmdscope)

### Research and benchmarks

- [CARE: Pre-Execution Command Verification for Shell-Executing LLM Agents](https://arxiv.org/abs/2607.21642)
- [RedCode repository](https://github.com/AI-secure/RedCode)
- [RedCode paper](https://arxiv.org/abs/2411.07781)
- [Attacker's Shell](https://arxiv.org/abs/2605.25871)
- [BoundaryBench](https://github.com/boundary-bench/boundary-bench)

### Community discussion used as qualitative signals

- [Ask HN: How are you sandboxing your coding agents?](https://news.ycombinator.com/item?id=46700628)
- [Show HN: Agent that refuses to run commands without human approval](https://news.ycombinator.com/item?id=47957127)
- [Sandbox Escape Vulnerabilities Across 4 Coding Agent Vendors](https://news.ycombinator.com/item?id=48978960)
- [OpenCode static verifier discussion](https://news.ycombinator.com/item?id=49064655)
- [FlowLink MCP proxy discussion](https://news.ycombinator.com/item?id=48283348)
- [Hacker News Algolia search](https://hn.algolia.com/api)

These references are evidence inputs and implementation references, not claims of adoption, market size, or product success. Community discussions and repository metrics should be interpreted using the evidence discipline described above.
