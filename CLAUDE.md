# CLAUDE.md — gormes-agent

Claude, claudeu, and Claude Code agents must follow `AGENTS.md` as the
canonical repository contract.

## Branch and CI safety rule

Main must always stay green. This is mandatory memory for Claude, claudeu,
Claude Code, and workspace-mineru / workspace-mimeru agents.

- `main` is protected release-trunk state. Do not do feature, docs, roadmap, or
  repair work directly on `main` after the bootstrap that created this rule.
- Develop only on the `development` branch, or on a short-lived branch created
  from `development`.
- Changes reach `main` only through a GitHub pull request into `main`.
- Before opening or updating a PR to `main`, run the same gate as CI:
  `go test ./... -count=1`, `go run ./cmd/progress validate`, and
  `git diff --check`.
- If `main` is red, stop normal feature work and fix `main` first. Do not
  branch new work from a red `main`.
- GitHub rules for `main` must require pull requests and the required CI status
  check. Do not bypass them unless the user explicitly asks for emergency
  repository recovery.

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
| Fixing provider/auth/client/model-routing/usage/rate-limit parity bugs | `gormes-provider-parity` |
| Updating roadmap rows, phases, dependencies, or planning docs | `gormes-planner` |
| Implementing one `progress.json` row | `gormes-builder` |
| Red-green-refactor delivery of one behavior | `gormes-tdd-slice` |
| Designing Go package/API boundaries before implementation | `gormes-interface-designer` |
| Stuck on a Go implementation shape; want a donor file from `references/go-agent-os/` before writing code | `gormes-references` |
| Auditing or periodically refreshing README/public repository messaging | `gormes-readme` |
| Improving `www.gormes.ai` landing page content or UI | `gormes-landing-web` |
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
