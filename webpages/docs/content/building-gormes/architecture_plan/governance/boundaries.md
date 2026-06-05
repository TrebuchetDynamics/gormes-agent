---
title: "Project Boundaries"
weight: 110
aliases:
  - /building-gormes/architecture_plan/boundaries/
---

## 5. Project Boundaries

Hard rule: no Python file in this repository is modified. Gormes is now the repository root: Go runtime code lives under `cmd/`, `internal/`, and `pkg/`; operator and contributor docs live under `docs/`; site code lives under `www.gormes.ai/`.

The bridge is allowed to exist. The bridge is not allowed to become the destination.

## Hermes Compatibility Namespace Boundary

`internal/llm` is a parity staging namespace. It is acceptable while Gormes
is proving Hermes-compatible provider, prompt, context, model, and tool-call
contracts in Go. It is not the intended long-term name for Gormes-owned runtime
architecture.

The long-term split is:

- `internal/compat/hermes` keeps upstream-facing compatibility evidence:
  Hermes config import/migration, command-surface manifests, provider/platform
  drift inventories, fixture provenance, and any explicit deprecated shim.
- Gormes-owned runtime packages keep product architecture: provider transports
  and routing under a provider package, model metadata under a model registry
  package, raw tool-call parsing/repair under a tool-call package, and prompt /
  context / compression under runtime or context packages.
- `internal/llm` becomes a temporary import shim only during migration and is
  removed once callers and fixtures prove the split.

Two alternatives are intentionally rejected. Renaming everything immediately
would churn tests before the compatibility manifests are strong enough. Keeping
`internal/llm` forever would make the donor name the permanent architecture
for Gormes-owned runtime code.

## Upstream Contract Boundary

Gormes studies Hermes and Gormes-owned as donors, but only ports contracts that make
the Go runtime better:

- provider-neutral stream and tool-call events;
- stable prompt assembly rules;
- gateway command/session semantics;
- operation and tool descriptors;
- memory/context provider lifecycle;
- durable job and subagent ledgers;
- graph provenance and retrieval evaluation.

It does not port upstream file shape. `run_agent.py`, `gateway/run.py`, and
external large operation and queue files are evidence, not templates.

## Trust-Class Boundary

Every operation should be classified before handler code runs:

| Trust class | Caller | Default posture |
|---|---|---|
| `operator` | local CLI/TUI/admin process | broadest access, still audited |
| `gateway` | Telegram/Discord/Slack/API user input | no local-operator tools without explicit allowlist |
| `child-agent` | delegated subagent | bounded tools, depth, timeout, and workspace scope |
| `system` | cron, boot hooks, maintenance jobs | deterministic payloads, audit required |

The executor should reject disallowed trust classes centrally. Handler-local
checks are still useful, but they are defense in depth rather than the primary
boundary.

## Provider Boundary

Provider quirks stay out of the kernel. Anthropic Messages, OpenAI Responses,
Bedrock Converse, OpenRouter, Gemini, Codex, and custom OpenAI-compatible
servers currently collapse into the shared Hermes-compatible event contract
staged under `internal/llm`:

- text and reasoning deltas;
- final finish reason;
- assistant tool calls;
- tool-result continuation payloads;
- token usage;
- classified retry/auth/rate/context errors.

Adapters own request shaping and protocol oddities. The kernel owns turn state,
cancellation, retry orchestration, tool execution, and finalization.

After the compatibility manifests and normal-turn fixtures are stable, the
event contract should move to a Gormes-owned provider/runtime package while
Hermes-specific import, drift, and deprecation evidence stays in
`internal/compat/hermes`.
