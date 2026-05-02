# AGENTS.md — gormes-agent

This file briefs every agent (codexu, claudeu, claude-code, opencode, or
any future backend) that runs against this repository. Read it before
touching code or docs in `cmd/`, `internal/`, `docs/content/building-gormes/`,
or `progress.json`.

## Branch and CI safety rule

Main must always stay green. Treat this as a hard repository rule for every
agent in this workspace, including workspace-mineru / workspace-mimeru agents.

- `main` is protected release-trunk state. Do not do feature, docs, roadmap, or
  repair work directly on `main` after the bootstrap that created this rule.
- Work directly on the existing `development` branch only. Do not create or use
  short-lived branches, feature branches, or git worktrees for Gormes work.
- Before editing, confirm the current branch is `development`. If it is not,
  stop before changing files and switch safely to `development` or report the
  blocker; never create another branch or worktree as the workaround.
- Changes reach `main` only through a GitHub pull request into `main`.
- Before opening or updating a PR to `main`, run the same gate as CI:
  `go test ./... -count=1`, `go run ./cmd/progress validate`, and
  `git diff --check`.
- If `main` is red, stop normal feature work and repair through the
  `development` branch and PR path. Do not check out `main` for edits and do
  not branch new work from a red `main`.
- This branch rule overrides any generic agent or skill workflow that suggests
  temporary branches or git worktrees.
- GitHub rules for `main` must require pull requests and the required CI status
  check. Do not bypass them unless the user explicitly asks for emergency
  repository recovery.

## Mandatory repo-local skill routing

Before doing any substantive work in this repository, every agent must select
and use at least one repo-local skill. The canonical skill source is
`docs/development-skills/<name>/SKILL.md`; `.agents/skills/`,
`.claude/skills/`, and `.codex/skills/` are symlink loader views for different
agents. If the right skill is not obvious, start with `gormes-skill-manager`
and let it route the task. Do not proceed "skill-less" on planning, building,
parity analysis, interface design, TDD implementation, or skill maintenance.

Use these skills as the default routing surface:

| Work type | Required skill |
|---|---|
| Unsure which workflow applies, or deciding whether a new skill is needed | `gormes-skill-manager` |
| Running a recurring full Hermes-in-Go parity sweep or recording periodic parity progress | `gormes-hermes-parity` |
| Mapping Hermes/Honcho/GBrain parity gaps | `gormes-parity-auditor` |
| Fixing provider/auth/client/model-routing/usage/rate-limit parity bugs | `gormes-provider-parity` |
| Browser automation parity, Browser Use, browser-harness, CDP, or `/browser connect` work | `gormes-browser-harness` |
| Local run/install/runtime work: `go run ./cmd/gormes`, `bin/gormes`, `install.sh`, managed source checkouts, PATH shadowing, gateway process ownership, or `sessions.db` locks | `gormes-dev-runtime` |
| Updating roadmap rows, phases, dependencies, or planning docs | `gormes-planner` |
| Implementing one `progress.json` row | `gormes-builder` |
| Red-green-refactor delivery of one behavior | `gormes-tdd-slice` |
| Designing Go package/API boundaries before implementation | `gormes-interface-designer` |
| Stuck on a Go implementation shape; want a donor file from `references/go-agent-os/` before writing code | `gormes-references` |
| Auditing or periodically refreshing README/public repository messaging | `gormes-readme` |
| Improving `www.gormes.ai` landing page content or UI | `gormes-landing-web` |
| Committing all dirty work, making `development` green, and pushing it | `gormes-git` |
| Stress-testing a plan or decision tree with the user | `grill-me` |

If none of these skills fits repeated Gormes work, use
`gormes-skill-manager` plus the system `skill-creator` workflow to create or
refine a repo-local skill under `docs/development-skills/`. Keep the new skill
bounded, validate it, and do not use skill creation as a substitute for
shipping Gormes. Recreate symlinks into `.agents/skills/`, `.claude/skills/`,
and `.codex/skills/` instead of copying skill files.

`gormes-planner` and `gormes-builder` are manual skill-routed workflows. The
old autonomous command binaries were intentionally removed: do not recreate
`cmd/planner-loop` or `cmd/builder-loop` as part of Gormes delivery. If a
future orchestrator is needed, plan it as a fresh subsystem with new names,
interfaces, and progress rows.

## Skill-Driven Delivery Architecture

Gormes' self-development now runs through repo-local skills plus one shared
progress representation. Agents do bounded passes:

```text
gormes-skill-manager -> gormes-planner / gormes-builder / gormes-tdd-slice
                     -> progress.json evidence -> tests -> handoff
```

The rule is simple: planning edits roadmap rows and docs; building implements
one row with tests; both use `progress.json` as the only backlog. Skills
replace the deleted autonomous loop executables.

### Shared Progress Representation

All planner and builder skills talk through these files. **Do not bypass them.**

- `docs/content/building-gormes/architecture_plan/progress.json` — canonical
  prioritized trajectory. Planner skills write; builder skills read to select
  one row. Schema lives at `internal/progress/`; rendered surfaces live under
  `docs/content/building-gormes/`.
- `cmd/progress` — focused command for validating `progress.json` and
  regenerating progress-driven docs.
- `cmd/repoctl` — focused command for repo metadata updates such as benchmark
  and README refreshes.
- Historical `.codex/builder-loop/` and `.codex/planner-loop/` ledgers may
  exist as evidence, but they are no longer active control-plane queues.

## Standing directive for any agent working here

1. **Preserve the contract.** New progress fields must round-trip through the
   typed structs in `internal/progress/`.
2. **Use the right skill.** Roadmap shape, row priorities, source references,
   ready-when / not-ready-when conditions, and trajectory go through
   `gormes-planner`. Runtime implementation goes through `gormes-builder` and
   `gormes-tdd-slice`.
3. **Keep passes bounded.** One planner pass sharpens a lane or row group. One
   builder pass ships one row. Stop with validation and handoff evidence.
4. **Don't introduce a parallel queue.** Side-channel TODO files,
   private prompt instructions, or hand-curated row lists outside
   `progress.json` are explicitly out of bounds. Fix the canonical row
   instead.

## Where to look first

| If you're … | Read this first |
|---|---|
| Unsure which workflow applies | `docs/development-skills/gormes-skill-manager/SKILL.md` |
| Running a recurring Hermes/Gormes parity sweep or checking periodic parity progress | `docs/development-skills/gormes-hermes-parity/SKILL.md` |
| Planning phases, dependencies, or roadmap rows | `docs/development-skills/gormes-planner/SKILL.md` |
| Fixing provider/auth/client/model-routing/usage/rate-limit parity bugs | `docs/development-skills/gormes-provider-parity/SKILL.md` |
| Browser automation parity, Browser Use, browser-harness, CDP, or `/browser connect` work | `docs/development-skills/gormes-browser-harness/SKILL.md` |
| Local run/install/runtime work, binary refresh, gateway ownership, or session DB locks | `docs/development-skills/gormes-dev-runtime/SKILL.md` |
| Stuck on a Go implementation shape and want a donor file before writing code | `docs/development-skills/gormes-references/SKILL.md` |
| Refreshing README.md or public repository claims from current evidence | `docs/development-skills/gormes-readme/SKILL.md` |
| Improving the public landing page content or UI | `docs/development-skills/gormes-landing-web/SKILL.md` |
| Committing all dirty work, making `development` green, and pushing it | `docs/development-skills/gormes-git/SKILL.md` |
| Implementing one row | `docs/development-skills/gormes-builder/SKILL.md` |
| Driving red-green-refactor | `docs/development-skills/gormes-tdd-slice/SKILL.md` |
| Changing the row schema or rendered docs | `internal/progress/` and the schema doc rendered at `docs/content/building-gormes/builder-loop/progress-schema.md` |
| Onboarding to the architecture with no prior context | this file, then `docs/content/building-gormes/_index.md` |
