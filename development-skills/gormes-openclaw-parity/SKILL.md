---
name: gormes-openclaw-parity
description: Discover useful OpenClaw-only behavior absent from Hermes. Use when classifying owned enhancements, adoption risk, or source-backed progress tasks.
---

# Gormes OpenClaw Parity

## Repository Branch Rule

For Gormes work, stay on the existing `development` branch. Do not create or
use feature branches, short-lived branches, or git worktrees. If the checkout
is not on `development`, stop before editing and switch safely or report the
blocker.

## Mission

Find useful OpenClaw behavior that is not part of Hermes, classify whether it
belongs in Gormes, and turn adoptable items into source-backed progress rows.

Hermes remains the primary compatibility contract for Gormes. OpenClaw is
evidence for optional Gormes-owned improvements, migration affordances, and Go
implementation ideas. Do not label OpenClaw-only behavior as required Hermes
parity unless Hermes source also proves that contract.

## Evidence Boundary

Allowed evidence:

- OpenClaw source at `./openclaw`, falling back to `../openclaw` or
  `references/openclaw`.
- Hermes source at `./hermes-agent`, falling back to `../hermes-agent` or
  `references/hermes-agent`.
- Current Gormes source, tests, docs, generated progress data, and sanitized
  user-provided transcripts.
- Temp fixtures and temp `GORMES_HOME` directories created for validation.

Do not read live private config, memory, credentials, session stores, or home
directories such as `~/.openclaw`, `~/.hermes`, `~/.gormes`, `~/.claude`,
`~/.codex`, or `~/.agents` unless the selected work is explicitly a migration
or runtime-home row with sanitized fixtures.

## Invocation Modes

| Invocation | Behavior |
|---|---|
| `gormes-openclaw-parity` | Auto-select one OpenClaw-only candidate by operator value, implementation size, testability, and compatibility risk. |
| `gormes-openclaw-parity <topic>` | Use `<topic>` as the seed; search OpenClaw, Hermes, Gormes, docs, tests, and the parity evidence doc, then classify the smallest coherent candidate. |

## Baseline

Run these before editing rows or code:

```sh
git status --short --branch
pwd
git rev-parse --show-toplevel
test -d ./openclaw && git -C ./openclaw rev-parse --short HEAD || true
test -d ../openclaw && git -C ../openclaw rev-parse --short HEAD || true
test -d ./hermes-agent && git -C ./hermes-agent rev-parse --short HEAD || true
test -d ../hermes-agent && git -C ../hermes-agent rev-parse --short HEAD || true
go run ./cmd/progress validate
```

Resolve source roots for citations:

```sh
OPENCLAW_SRC="$(for p in ./openclaw ../openclaw references/openclaw; do [ -d "$p" ] && { printf '%s\n' "$p"; break; }; done)"
HERMES_SRC="$(for p in ./hermes-agent ../hermes-agent references/hermes-agent; do [ -d "$p" ] && { printf '%s\n' "$p"; break; }; done)"
```

Stop if `$OPENCLAW_SRC` is empty. If `$HERMES_SRC` is empty, continue only as
an OpenClaw discovery pass and mark the Hermes absence check as blocked.

## Workflow

### 1. Bound The Sweep

Pick one surface: CLI/config/migration, provider behavior, tool UX, gateway or
channel handling, memory/session behavior, installer/runtime, browser
automation, security, observability, docs/onboarding, or packaging.

If the user asks for "everything", produce a short subsystem map and choose the
next three bounded OpenClaw-only audits. Do not copy a whole OpenClaw subsystem
into Gormes as one task.

### 2. Prove It Is OpenClaw-Only

For each candidate, record:

- the exact OpenClaw files, tests, docs, command names, or symbols that prove
  the behavior exists;
- the Hermes search performed and the closest Hermes equivalent, if any;
- the current Gormes implementation, tests, docs, or progress rows that already
  cover or reject it.

Use `rg`, `find`, and `jq`. Do not infer uniqueness from package names alone.
A candidate is OpenClaw-only only when Hermes lacks the behavior or implements a
weaker/meaningfully different version.

### 3. Classify The Candidate

Use one classification:

| Classification | Meaning |
|---|---|
| `adopt` | Useful Gormes-owned enhancement that fits without breaking Hermes compatibility. |
| `adapt` | Useful idea, but must be reshaped to preserve Gormes/Hermes contracts. |
| `covered` | Already implemented, tested, or row-backed in Gormes. |
| `hermes-parity` | Actually exists in Hermes; route to the `gormes-hermes-parity` orchestrator. |
| `exclude` | Not aligned, unsafe, too coupled to OpenClaw internals, licensing-risky, or not worth the dependency. |
| `blocked` | Source evidence is missing or unsafe to inspect. |

Prefer `exclude` for OpenClaw branding, home-directory coupling, telemetry,
private config import outside migration rows, broad dependency imports, or UX
that would confuse Hermes-compatible users.

### 4. Score Adoption

Recommend `adopt` or `adapt` only when the feature has:

- clear operator or user value;
- low compatibility risk against Hermes-facing contracts;
- a focused test shape;
- a small write scope;
- source refs strong enough for future agents to re-check;
- no credential, privacy, or supply-chain hazard.

If the idea needs a new package boundary, route first to
`gormes-interface-designer`. If it needs progress rows or docs reshaped, route
to `gormes-planner`. If a builder-ready row already exists and the user asked
for implementation, route to `gormes-builder` and `gormes-tdd-slice`.

### 5. Produce Progress-Ready Work

For `adopt` or `adapt`, add or refine an atom in the parity evidence doc only when the user
asked this skill to apply changes. Otherwise, emit a task packet for
`gormes-planner`.

Rows should make the divergence explicit as Gormes-owned. Required row inputs:

- observable behavior, not a theme;
- OpenClaw source refs and Hermes absence refs;
- current Gormes refs and any existing progress row;
- target Go package and public contract;
- exact write scope;
- focused red-test hint;
- acceptance criteria and validation command;
- compatibility notes saying why Hermes users are not broken.

## Output Packet

```text
scope:
openclaw_source:
openclaw_ref:
hermes_ref_check:
gormes_refs:
classification:
recommendation:
progress_row:
source_refs:
adoption_risk:
red_test_hint:
validation:
next_skill_chain:
blockers:
```

## Guardrails

- Do not treat OpenClaw as the primary upstream for Gormes; Hermes remains the
  compatibility contract.
- Do not read live OpenClaw or Hermes private state as evidence.
- Do not create side backlogs. Implementation intent goes into
  `progress.json`.
- Do not implement runtime code during discovery unless the user explicitly
  asks to continue one builder-ready row in the same turn.
- Do not import OpenClaw runtime paths, config homes, or command branding into
  Gormes without a migration or compatibility row.
- Preserve dirty user work and exact source citations.

## Validation

If editing only this skill or routing docs:

```sh
python3 /home/xel/.codex/skills/.system/skill-creator/scripts/quick_validate.py development-skills/gormes-openclaw-parity
find -L .agents/skills .claude/skills .codex/skills -maxdepth 2 -name SKILL.md -print | sort
git diff --check
```

If editing progress rows:

```sh
go run ./cmd/progress write
go run ./cmd/progress validate
go test ./internal/planning/progress -count=1
git diff --check
```
