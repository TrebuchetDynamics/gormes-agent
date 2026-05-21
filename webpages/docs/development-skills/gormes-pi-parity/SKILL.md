---
name: gormes-pi-parity
description: Use when discovering or applying reusable harness ideas from Pi for Gormes, including extensions, tool middleware, SDK/RPC embedding, TUI components, session trees, compaction, packages, prompt templates, safety gates, or provider hooks, while keeping Hermes parity and OpenClaw-owned enhancements separate.
---

# Gormes Pi Parity

## Repository Branch Rule

For Gormes work, stay on the existing `development` branch. Do not create or
use feature branches, short-lived branches, or git worktrees. If the checkout
is not on `development`, stop before editing and switch safely or report the
blocker.

## Mission

Learn from Pi's best harness techniques without making Pi a product contract.
Pi is a donor for agent-harness architecture, extension seams, UI patterns,
session mechanics, and safety gates. Hermes remains the Gormes compatibility
contract. OpenClaw remains the route for OpenClaw-only Gormes-owned adoption.

## Conflict Boundary

| Source | Role for Gormes | If it conflicts |
|---|---|---|
| Hermes/Honcho | Required user-visible behavior and compatibility contract | Use `gormes-hermes-parity`; preserve Hermes-facing commands, config, outputs, and operator UX. |
| OpenClaw | Optional owned enhancements, migration affordances, and local infra ideas | Use `gormes-openclaw-parity`; classify as adopt/adapt/exclude before rows. |
| Pi | Harness technique donor: extension APIs, tools, TUI, SDK/RPC, sessions, compaction, packages | Adapt the technique only when it does not break Hermes compatibility or OpenClaw-owned boundaries. |

Never label a Pi feature as required parity. Classify it as a Gormes-owned
harness improvement and make the divergence explicit in any progress row.

## Evidence Boundary

Allowed evidence:

- Local Pi package docs, normally
  `/home/xel/.nvm/versions/node/v22.21.1/lib/node_modules/@earendil-works/pi-coding-agent/`.
- A trusted checkout of `https://github.com/earendil-works/pi` or `pi-mono` if
  the local package is unavailable or stale.
- Current Gormes source, tests, docs, generated progress data, and sanitized
  fixtures.

Do not read live private Pi homes such as `~/.pi/agent/auth.json`, real session
stores, package credentials, browser profiles, or user secrets. Use temp
fixtures when testing config/session behavior.

## Source Order

Read the smallest source slice that answers the harness question:

| Need | Pi source |
|---|---|
| Core philosophy and feature map | `README.md` |
| Extension events, tools, commands, UI hooks, providers | `docs/extensions.md`, `examples/extensions/README.md` |
| Custom TUI components and renderers | `docs/tui.md`, targeted examples under `examples/extensions/` |
| Embedding Gormes as a library or process | `docs/sdk.md`, `docs/rpc.md`, `examples/sdk/README.md` |
| Session tree, labels, compaction, branch summaries | `docs/sessions.md`, `docs/session-format.md`, `docs/compaction.md` |
| Skills, prompts, themes, packages | `docs/skills.md`, `docs/prompt-templates.md`, `docs/themes.md`, `docs/packages.md` |
| Provider/model techniques | `docs/models.md`, `docs/custom-provider.md` |
| Plan mode or subagents as optional harness modules | `examples/extensions/plan-mode/README.md`, `examples/extensions/subagent/README.md` |

## Technique Catalog

Prefer these Pi-derived patterns when shaping Gormes harness rows:

- **Extensible core, thin defaults:** keep the runtime small; add tools,
  commands, UI, providers, and policies through typed extension seams.
- **Event middleware:** use before/after hooks for input, context, provider
  requests, tool calls/results, messages, sessions, compaction, and shutdown.
- **Safe tool design:** strict schemas, compatibility shims before validation,
  output truncation with full-output artifact pointers, abort signals, and
  per-file mutation queues for parallel tools.
- **Operator UI seams:** status widgets, custom footers, overlays, editor
  replacement, autocomplete providers, compact tool renderers, and line-width
  safe TUI components.
- **Durable sessions:** JSONL session trees with labels, branch summaries,
  compaction entries, and extension state stored in result details.
- **Embeddable modes:** interactive, print/JSON, RPC, and SDK-style runtime
  APIs with clear event streams and session replacement boundaries.
- **Packageable resources:** skills, prompts, themes, and extensions packaged
  as installable resources with explicit provenance and enable/disable filters.
- **Provider adaptability:** custom provider registration, OAuth, compatibility
  flags, model-level thinking maps, normalized overflow errors, and streaming
  event contracts.

## Workflow

1. Baseline Gormes state:

```sh
git status --short --branch
pwd
git rev-parse --show-toplevel
go run ./cmd/progress validate
```

2. Resolve Pi evidence:

```sh
PI_SRC="$(node -p "require('node:path').dirname(require.resolve('@earendil-works/pi-coding-agent/package.json'))" 2>/dev/null || true)"
test -n "$PI_SRC" && node -p "require('$PI_SRC/package.json').version"
```

3. Bound the sweep to one harness surface: extension seam, tool runtime,
   session/compaction, TUI, SDK/RPC, provider model registry, package/resource
   loading, safety gate, plan mode, or subagent orchestration.
4. Read the Pi source slice and the closest Gormes surface. If the behavior is
   user-visible, read Hermes first. If it is OpenClaw-only, route to
   `gormes-openclaw-parity`.
5. Classify the technique: `adopt`, `adapt`, `covered`, `hermes-conflict`,
   `openclaw-conflict`, `exclude`, or `blocked`.
6. For `adopt`/`adapt`, emit a progress-ready packet or route to
   `gormes-planner`. For implementation, use `gormes-builder` plus
   `gormes-tdd-slice`.

## Classification Rules

- `adopt`: clear harness value, low compatibility risk, small Go write scope,
  and source-backed test shape.
- `adapt`: useful Pi idea but TypeScript/UI/process details must be reshaped
  for Gormes' Go runtime and Hermes-facing UX.
- `covered`: Gormes already has the behavior, tests, docs, or row.
- `hermes-conflict`: Pi behavior would change required Hermes compatibility;
  route to `gormes-hermes-parity` and do not adopt by default.
- `openclaw-conflict`: Pi behavior overlaps OpenClaw-owned migration or fleet
  affordances; route to `gormes-openclaw-parity`.
- `exclude`: not aligned, too package-specific, unsafe, credential-sensitive,
  or not worth the dependency.
- `blocked`: Pi source is missing, stale, or unsafe to inspect.

## Guardrails

- Do not import Pi packages into Gormes runtime without an explicit dependency
  decision and Go-native design. Prefer patterns over code copy.
- Do not create side backlogs. Implementation intent belongs in
  `progress.json` through `cmd/progress` or `internal/progress`.
- Do not weaken Hermes CLI/config/channel compatibility to match Pi's UX.
- Do not treat Pi's lack of built-in subagents, plan mode, MCP, or permission
  popups as a Gormes requirement; those are extensibility lessons.
- Preserve dirty user work and cite exact Pi docs/examples in reports.

## Output Packet

```text
scope:
pi_source:
pi_refs:
gormes_refs:
hermes_boundary:
openclaw_boundary:
classification:
recommendation:
progress_row:
red_test_hint:
validation:
next_skill_chain:
blockers:
```

## Validation

For skill-only or routing edits:

```sh
python3 /home/xel/.codex/skills/.system/skill-creator/scripts/quick_validate.py docs/development-skills/gormes-pi-parity
find -L .agents/skills .claude/skills .codex/skills -maxdepth 2 -path '*/gormes-pi-parity/SKILL.md' -print | sort
git diff --check
```

If progress rows change, also run:

```sh
go run ./cmd/progress write
go run ./cmd/progress validate
go test ./internal/progress -count=1
```
