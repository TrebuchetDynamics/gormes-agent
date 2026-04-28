# CLAUDE.md — gormes-agent

Claude, claudeu, and Claude Code agents must follow `AGENTS.md` as the
canonical repository contract.

## Mandatory skill routing

Before doing any substantive work in this repository, select and use at least
one repo-local skill. Canonical skill files live under
`docs/development-skills/<name>/SKILL.md`; `.claude/skills/`,
`.agents/skills/`, and `.codex/skills/` are symlink loader views. If the right
skill is unclear, start with `gormes-skill-manager`.

Default routing:

| Work type | Required skill |
|---|---|
| Unsure which workflow applies, or deciding whether a new skill is needed | `gormes-skill-manager` |
| Mapping Hermes/Honcho/GBrain parity gaps | `gormes-parity-auditor` |
| Updating roadmap rows, phases, dependencies, or planning docs | `gormes-planner` |
| Implementing one `progress.json` row | `gormes-builder` |
| Red-green-refactor delivery of one behavior | `gormes-tdd-slice` |
| Designing Go package/API boundaries before implementation | `gormes-interface-designer` |
| Auditing or periodically refreshing README/public repository messaging | `gormes-research` |
| Stress-testing a plan or decision tree with the user | `grill-me` |

If none fits repeated Gormes work, use `gormes-skill-manager` with a skill
creation workflow to create or refine a repo-local skill under
`docs/development-skills/`, then symlink it into `.claude/skills/`,
`.agents/skills/`, and `.codex/skills/`. Do not proceed skill-less on
planning, building, parity analysis, interface design, TDD implementation, or
skill maintenance.

`gormes-planner` and `gormes-builder` mean manual skill-guided work. The old
autonomous command binaries were intentionally removed; do not recreate
`cmd/planner-loop` or `cmd/builder-loop` as part of Gormes delivery. If a
future orchestrator is needed, plan it as a fresh subsystem with new names,
interfaces, and progress rows.

## Core rule

Gormes is complete only when it is Hermes in Go, with Goncho as the
Honcho-compatible Go port inside Gormes. Keep all implementation intent in
`docs/content/building-gormes/architecture_plan/progress.json`; do not create
parallel queues.
