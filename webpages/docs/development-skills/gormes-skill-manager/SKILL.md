---
name: gormes-skill-manager
description: Select the right repo-local Gormes skill or skill chain for building gormes-agent and identify when a new Gormes skill is needed. Use when starting any substantial Gormes planning/building task, when unsure whether to use gormes-planner, gormes-builder, gormes-parity-auditor, gormes-tdd-slice, or gormes-interface-designer, or when repeated Gormes work suggests creating another skill.
---

# Gormes Skill Manager

## Repository Branch Rule

For Gormes work, stay on the existing `development` branch. Do not create or
use feature branches, short-lived branches, or git worktrees. If the checkout
is not on `development`, stop before editing and switch safely or report the
blocker.

## Mission

Route Gormes work to the smallest effective skill chain. Gormes is finished only when it is Hermes in Go with Goncho as the Honcho-compatible Go port inside Gormes; skill selection should serve that delivery goal, not create process theater.

Hermes Agent is the Python upstream/father implementation for Gormes. In this
repository, prefer the in-repo reference checkout at `./hermes-agent`; fall
back to `../hermes-agent` only when the in-repo checkout is absent. Treat it as
behavior evidence, not as a Gormes runtime dependency.

Canonical skill source lives under `docs/development-skills/<name>/SKILL.md`.
The `.agents/skills/`, `.claude/skills/`, and `.codex/skills/` directories are
symlink loader views.

## Workflow

### 1. Classify The User Request

Start by locating the work in
`docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md`
and, when implementation intent exists, the matching row in
`docs/content/building-gormes/architecture_plan/progress.json`.

Pick the primary intent:

- **Decide direction**: use `grill-me` and optionally `gormes-planner`.
- **Run recurring or periodic Hermes/Gormes parity progress sweeps**:
  use `gormes-hermes-parity` as the orchestrator. It loads only the needed
  parity reference and manages follow-up subskills such as
  `gormes-parity-auditor`, `gormes-planner`, `gormes-builder`,
  `gormes-tdd-slice`, `gormes-dev-runtime`, and `gormes-openclaw-parity`.
  Use the same route when parity taxonomy, roadmap group names, or
  feature-map sections need source-backed renames or restructures.
- **Discover useful OpenClaw-only behavior absent from Hermes**: use
  `gormes-openclaw-parity` to classify the candidate as adopt, adapt, covered,
  Hermes parity, exclude, or blocked. Route adopt/adapt findings to
  `gormes-planner` for progress rows before implementation.
- **Concrete Gormes-vs-Hermes UX bug** such as ugly TUI chrome, duplicate
  replies, visible hourglass/status messages, hidden tool-progress mismatch,
  Telegram formatting drift, or stale product labels: use
  `gormes-hermes-parity` to pick the active upstream contract, then
  `gormes-tdd-slice` for one observable behavior.
- **Tool iteration limit, bad tool calling, or leaked tool-call text**: start
  with `gormes-hermes-parity` to decide whether the bug is kernel loop,
  provider stream repair, or channel rendering. Route provider malformed
  tool-call payloads to `gormes-provider-parity`; route visible duplicate or
  raw budget errors to `gormes-tdd-slice` with a transcript fixture.
- **Map upstream parity**: use `gormes-parity-auditor`, then `gormes-planner` if rows need edits.
- **Plan roadmap rows**: use `gormes-planner`.
- **Plan Hermes CLI/config/migration parity**: use `gormes-parity-auditor`
  then `gormes-planner`; require the command-tree manifest before handler
  implementation and keep `config migrate` separate from `migrate hermes` /
  `migrate openclaw`. Treat `ooenclaw` as an OpenClaw typo-suggestion path,
  not a second migration command, unless a compatibility row exists.
- **Provider/auth/model/runtime failure or implementation**: use
  `gormes-provider-parity`, then `gormes-builder` + `gormes-tdd-slice` when
  code changes are needed. This route must check Hermes behavior first, then
  inspect local Go references such as GoClaw/Plandex/Nanobot/trpc-agent-go/ADK-Go
  for implementation patterns while preserving Hermes parity as P0.
- **Install/setup reports with live settings** such as Telegram bot token,
  `telegram.allowed_user_ids`, workspace path, Codex provider, or Codex CLI
  auth import: use `gormes-dev-runtime` to prove the active `GORMES_HOME`,
  binary, PATH, config path, and secret redaction; then use
  `gormes-hermes-parity`/`gormes-provider-parity` plus `gormes-tdd-slice` for
  any setup output drift. Require `config check`, `auth status`, `doctor
  --offline`, and `onboard --wizard --non-interactive` before calling the setup
  fixed.
- **Browser automation, Browser Use, browser-harness, CDP, or `/browser connect` parity**:
  use `gormes-browser-harness`, then `gormes-parity-auditor`/`gormes-planner`
  for rows or `gormes-builder` + `gormes-tdd-slice` for a single runtime slice.
- **Local run/install/runtime operations**: use `gormes-dev-runtime` when the
  task involves `go run ./cmd/gormes`, `bin/gormes`, `install.sh`, managed
  source checkouts, PATH shadowing, gateway process ownership, or
  `sessions.db` locks.
- **Workspace identity confusion**: use `gormes-dev-runtime` first when the
  request mixes `workspace-mineru`, `workspace-gormes`, `~/.gormes`,
  installer-managed clones, or `GORMES_HOME`. Then route any remaining parity
  behavior through `gormes-hermes-parity`.
- **Agent persona, SOUL.md defaults, reset-template behavior, or skills tool
  exposure**: use `gormes-hermes-parity` for source-backed Hermes behavior,
  then `gormes-planner` or `gormes-tdd-slice` depending on whether the row is
  missing or builder-ready.
- **Design a Go interface/package boundary**: use `gormes-interface-designer`.
- **Implement one row**: use `gormes-builder`, then `gormes-tdd-slice` for the red-green loop.
- **Fix a failing row/test**: use `gormes-tdd-slice`; escalate to `gormes-builder` if progress/docs need updates.
- **Audit README or public repo messaging**: use `gormes-readme`.
- **Improve landing page content or UI**: use `gormes-landing-web`.
- **Commit, validate, and push the dirty `development` branch**: use
  `gormes-git`.
- **Prepare, publish, or recover a Gormes release**: use `gormes-release`.
  It may route dirty-worktree commit/push work through `gormes-git`, but it
  owns release intent, version/tag checks, artifact verification, and recovery
  stop conditions.
- **Create or improve skills**: use system `skill-creator` plus this manager.
  Fold repeated mistakes into existing class-level skills before creating a
  new one, and keep the update as process guidance rather than a session diary.
- **Learn from past Gormes tasks**: improve the existing skill that should have
  prevented the mistake. Installer, PATH, `go run`, `bin/gormes`,
  `GORMES_HOME`, `workspace-gormes`, or `sessions.db` lessons belong in
  `gormes-dev-runtime`; hidden UX, hourglass, duplicate replies, tool-progress
  formatting, tool-iteration leaks, persona/defaults, and reset-template
  lessons belong in `gormes-hermes-parity`; red-test shape belongs in
  `gormes-tdd-slice`.
- **Pasted channel tool-progress blocks** such as
  `📚 skill_view: "plan"` / `📋 todo: "planning 5 task(s)"` /
  `💻 execute_code: "..."`: use `gormes-hermes-parity` first. The active
  contract is Hermes gateway/channel progress, not only the current Ink TUI.

If more than one applies, choose a chain with at most three skills. Do not load every Gormes skill.

Feature-map gaps route to `gormes-parity-auditor` then `gormes-planner`.
Builder-ready rows route to `gormes-builder` and, when tests are required,
`gormes-tdd-slice`. Vague rows route back to `gormes-planner`. Unclear package
boundaries route to `gormes-interface-designer` before implementation.

When `gormes-hermes-parity` emits a follow-up task packet, treat its
`scope`, `feature_map_area`, `progress_row`, `source_refs`, `write_scope`, and
`validation` fields as the routing input. Return the smallest skill chain that
can finish that packet, and preserve the packet in the handoff so builder or
planner agents do not rediscover the same context.

If a request spans runtime validation plus UX parity, split the routing:
`gormes-dev-runtime` proves which binary/process is running, while
`gormes-hermes-parity` identifies the upstream contract and hands one behavior
to `gormes-tdd-slice`. Do not let installer debugging consume TUI or Telegram
parity work.

If a request says only "tool calling is bad", demand one visible artifact or
run a small local fixture. Then classify it as provider parsing, kernel
iteration, tool descriptor/schema, channel status UX, or stale binary before
choosing the builder skill.

### 2. Prefer Existing Skills

Before creating a new skill, inspect current repo-local skills:

```sh
find docs/development-skills -maxdepth 2 -name SKILL.md -print | sort
find -L .agents/skills .claude/skills .codex/skills -maxdepth 2 -name SKILL.md -print | sort
```

Read `references/skill-routing.md` for the routing table. Reuse or improve an existing skill when the task is only a variant of an existing workflow.

### 3. Decide Whether A New Skill Is Needed

Create or propose a new skill only when at least two are true:

- the task will recur across many Gormes passes;
- the workflow is distinct from planning, parity audit, interface design, or row TDD;
- the task has its own validation gates or fixtures;
- agents repeatedly make the same mistakes without explicit instructions;
- deterministic scripts or reference files would save tokens and reduce failure;
- the task maps to a stable subsystem such as provider parity, Goncho compatibility, channel adapters, docs/web sync, release packaging, or e2e operations.

Do not create a skill for one-off context, a single row, or a vague theme.

### 4. Name And Scope New Skills

Use names like:

- `gormes-provider-parity`
- `gormes-dev-runtime`
- `gormes-hermes-parity`
- `gormes-openclaw-parity`
- `gormes-goncho-compat`
- `gormes-channel-adapter`
- `gormes-docs-web-sync`
- `gormes-e2e-operator`
- `gormes-release-packager`

Each new skill must have:

- clear trigger description;
- one bounded workflow;
- references only when needed;
- validation commands;
- no duplicate doctrine already present in `gormes-planner` or `gormes-builder`.

Create the skill under `docs/development-skills/<name>/`. Then add symlinks:

```sh
ln -s ../../docs/development-skills/<name> .agents/skills/<name>
ln -s ../../docs/development-skills/<name> .claude/skills/<name>
ln -s ../../docs/development-skills/<name> .codex/skills/<name>
```

### 5. Report The Routing Decision

Before doing substantial work, state:

- selected skill or skill chain;
- feature-map area;
- progress row, if any;
- why it fits;
- fallback if the selected skill cannot proceed;
- whether a new skill is needed now.

Keep this short; then execute the chosen workflow.

Use this packet shape:

```text
selected_skill:
feature_map_area:
progress_row:
reason:
fallback:
new_skill_needed:
```

## Guardrails

- Do not let skill management replace delivery.
- Do not create side backlogs. Implementation intent goes into `progress.json`.
- Do not recreate deleted loop commands. Planning/building now happens through bounded skill-driven passes.
- Use Context7 for external library/framework/API docs when required by repo instructions.
- Preserve dirty user work.
