---
name: gormes-skill-manager
description: Select the right repo-local Gormes skill or skill chain for building gormes-agent and identify when a new Gormes skill is needed. Use when starting any substantial Gormes planning/building task, when unsure whether to use gormes-planner, gormes-builder, gormes-parity-auditor, gormes-tdd-slice, or gormes-interface-designer, or when repeated Gormes work suggests creating another skill.
---

# Gormes Skill Manager

## Mission

Route Gormes work to the smallest effective skill chain. Gormes is finished only when it is Hermes in Go with Goncho as the Honcho-compatible Go port inside Gormes; skill selection should serve that delivery goal, not create process theater.

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
- **Map upstream parity**: use `gormes-parity-auditor`, then `gormes-planner` if rows need edits.
- **Plan roadmap rows**: use `gormes-planner`.
- **Design a Go interface/package boundary**: use `gormes-interface-designer`.
- **Implement one row**: use `gormes-builder`, then `gormes-tdd-slice` for the red-green loop.
- **Fix a failing row/test**: use `gormes-tdd-slice`; escalate to `gormes-builder` if progress/docs need updates.
- **Audit README or public repo messaging**: use `gormes-readme`.
- **Create or improve skills**: use system `skill-creator` plus this manager.

If more than one applies, choose a chain with at most three skills. Do not load every Gormes skill.

Feature-map gaps route to `gormes-parity-auditor` then `gormes-planner`.
Builder-ready rows route to `gormes-builder` and, when tests are required,
`gormes-tdd-slice`. Vague rows route back to `gormes-planner`. Unclear package
boundaries route to `gormes-interface-designer` before implementation.

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

- `gormes-goncho-compat`
- `gormes-provider-parity`
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
