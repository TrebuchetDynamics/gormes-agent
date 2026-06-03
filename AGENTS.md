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
`development-skills/<name>/SKILL.md`; `.agents/skills/`,
`.claude/skills/`, and `.codex/skills/` are symlink loader views for different
agents. If the right skill is not obvious, start with `gormes-skill-manager`
and let it route the task. Do not proceed "skill-less" on planning, building,
parity analysis, interface design, TDD implementation, or skill maintenance.

Use these skills as the default routing surface:

| Work type | Required skill |
|---|---|
| Unsure which workflow applies, or deciding whether a new skill is needed | `gormes-skill-manager` |
| Running a recurring full Hermes-in-Go parity sweep or recording periodic parity progress | `gormes-hermes-parity` as the subskill orchestrator |
| Discovering useful OpenClaw-only behavior absent from Hermes for possible Gormes-owned adoption | `gormes-openclaw-parity` |
| Discovering reusable Pi harness techniques without making Pi a parity contract | `gormes-pi-parity` |
| Mapping Hermes/Honcho parity gaps | `gormes-parity-auditor` |
| Fixing provider/auth/client/model-routing/usage/rate-limit parity bugs | `gormes-provider-parity` |
| Browser automation parity, Browser Use, browser-harness, CDP, or `/browser connect` work | `gormes-browser-harness` |
| Local run/install/runtime work: `go run ./cmd/gormes`, `bin/gormes`, `install.sh`, managed source checkouts, PATH shadowing, gateway process ownership, or `sessions.db` locks | `gormes-dev-runtime` |
| Updating roadmap rows, phases, dependencies, or planning docs | `gormes-planner` |
| Running a bounded architecture -> planner -> parity -> builder delivery cycle | `gormes-delivery-loop` |
| Implementing one `progress.json` row | `gormes-builder` |
| Red-green-refactor delivery of one behavior | `gormes-tdd-slice` |
| Designing Go package/API boundaries before implementation | `gormes-interface-designer` |
| Refactoring one `cmd/gormes` command domain into `internal/app/<domain>` without behavior changes | `cmd-internal-refactor` |
| Stuck on a Go implementation shape; want a donor file from `references/go-agent-os/` before writing code | `gormes-references` |
| External library/framework/upstream source context before planning or implementation | `gormes-context-sourcing` |
| Repeated runtime mechanics or service-layer cleanup after a feature works | `gormes-service-layer-refactor` |
| PR feedback, CI failures, or bounded review-to-green iteration | `gormes-review-loop` |
| Auditing or periodically refreshing README/public repository messaging | `gormes-readme` |
| Improving `www.gormes.ai` landing page content or UI | `gormes-landing-web` |
| Designing, critiquing, or polishing dashboard screenshots, hero images, social cards, or image-based dashboard assets | `dashboard-image-design` |
| Committing all dirty work, making `development` green, and pushing it | `gormes-git` |
| Preparing, PR-merging, tagging, and verifying a Gormes release | `gormes-release` |
| Stress-testing a plan or decision tree with the user | global `grill-me`; pair with repo-local `gormes-skill-manager` when Gormes routing is needed |

The global `/home/xel/.agents/skills/grill-me` skill is canonical; do not add a
repo-local `grill-me` shadow skill. For Gormes-specific safety, keep branch
rules in this `AGENTS.md` and load `gormes-skill-manager` alongside global
`grill-me` when needed.

If none of these skills fits repeated Gormes work, use
`gormes-skill-manager` plus the system `skill-creator` workflow to create or
refine a repo-local skill under `development-skills/`. Keep the new skill
bounded, validate it, and do not use skill creation as a substitute for
shipping Gormes. Recreate symlinks into `.agents/skills/`, `.claude/skills/`,
and `.codex/skills/` instead of copying skill files.

`gormes-planner` and `gormes-builder` are skill-routed workflows. Delivery
orchestrators are allowed when explicitly requested by Juan, but they must be
planned as first-class subsystems with clear names, interfaces, progress rows,
validation gates, and operator controls.

## Skill-Driven Delivery Architecture

Gormes' self-development now runs through repo-local skills plus one shared
progress representation. Agents do bounded passes:

```text
gormes-skill-manager -> gormes-planner / gormes-builder / gormes-tdd-slice
                     -> progress.json evidence -> tests -> handoff
```

The rule is simple: planning edits roadmap rows and docs; building implements
one row with tests; both use `progress.json` as the only backlog. "The only
backlog" is a single *logical* backlog accessed through `internal/progress`
(`Load`/`SaveProgress`) and `cmd/progress` — never by hand-parsing files. The
loader transparently supports either the single monolithic `progress.json`
*or* a split/per-module directory layout; the canonical on-disk form today is
the monolith, and the split layout, while fully supported by the tooling, is
not yet materialized (pending the module-split umbrella's final child). Skills
replace the deleted autonomous loop executables.

### Shared Progress Representation

All planner and builder skills talk through these files. **Do not bypass them.**

- `docs/content/building-gormes/architecture_plan/progress.json` — canonical
  prioritized trajectory and single logical backlog. Planner skills write;
  builder skills read to select one row. Always go through
  `internal/progress.Load`/`SaveProgress` (or `cmd/progress`), which
  transparently read and write **either** this monolithic file **or** a
  split/per-module directory layout (one logical backlog, physically one or
  many files); never hand-parse member files. Schema lives at
  `internal/progress/`; rendered surfaces live under
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
| Unsure which workflow applies | `development-skills/gormes-skill-manager/SKILL.md` |
| Running a recurring Hermes/Gormes parity sweep or checking periodic parity progress | `development-skills/gormes-hermes-parity/SKILL.md` |
| Discovering useful OpenClaw-only behavior that Hermes lacks | `development-skills/gormes-openclaw-parity/SKILL.md` |
| Learning reusable Pi harness techniques without conflicting with Hermes/OpenClaw parity | `development-skills/gormes-pi-parity/SKILL.md` |
| Planning phases, dependencies, or roadmap rows | `development-skills/gormes-planner/SKILL.md` |
| Fixing provider/auth/client/model-routing/usage/rate-limit parity bugs | `development-skills/gormes-provider-parity/SKILL.md` |
| Browser automation parity, Browser Use, browser-harness, CDP, or `/browser connect` work | `development-skills/gormes-browser-harness/SKILL.md` |
| Local run/install/runtime work, binary refresh, gateway ownership, or session DB locks | `development-skills/gormes-dev-runtime/SKILL.md` |
| Stuck on a Go implementation shape and want a donor file before writing code | `development-skills/gormes-references/SKILL.md` |
| External library/framework/upstream source context before planning or implementation | `development-skills/gormes-context-sourcing/SKILL.md` |
| Repeated runtime mechanics or service-layer cleanup after a feature works | `development-skills/gormes-service-layer-refactor/SKILL.md` |
| PR feedback, CI failures, or bounded review-to-green iteration | `development-skills/gormes-review-loop/SKILL.md` |
| Refreshing README.md or public repository claims from current evidence | `development-skills/gormes-readme/SKILL.md` |
| Improving the public landing page content or UI | `development-skills/gormes-landing-web/SKILL.md` |
| Committing all dirty work, making `development` green, and pushing it | `development-skills/gormes-git/SKILL.md` |
| Preparing, PR-merging, tagging, and verifying a Gormes release | `development-skills/gormes-release/SKILL.md` |
| Implementing one row | `development-skills/gormes-builder/SKILL.md` |
| Driving red-green-refactor | `development-skills/gormes-tdd-slice/SKILL.md` |
| Refactoring one `cmd/gormes` command domain into `internal/app/<domain>` | `development-skills/cmd-internal-refactor/SKILL.md` |
| Changing the row schema or rendered docs | `internal/progress/` and the schema doc rendered at `docs/content/building-gormes/builder-loop/progress-schema.md` |
| Onboarding to the architecture with no prior context | this file, then `docs/content/building-gormes/_index.md` |

## Repository Map

A root codemap is available at `codemap.md`.

Before working on any task, read `codemap.md` to understand:
- Project architecture and entry points
- Directory responsibilities and design patterns
- Data flow and integration points between modules

For deep work on a specific folder, also read that folder's `codemap.md`.

<!-- karpathy-guidelines:start -->
## Karpathy-Inspired Agent Guardrails

Source: https://github.com/forrestchang/andrej-karpathy-skills at commit `2c60614`.

These guardrails supplement the local instructions above. Local project, safety, and user-specific rules win on conflict.

Tradeoff: they bias toward caution over speed for non-trivial work; use judgment for obvious one-line fixes.

### Think Before Coding

- State assumptions before implementing; ask when uncertainty would change the solution.
- Surface multiple interpretations and tradeoffs instead of silently picking one.
- Push back when a simpler approach meets the goal.

### Simplicity First

- Build the minimum code that solves the requested problem.
- Avoid speculative features, single-use abstractions, and unnecessary configurability.
- If the solution is growing large, stop and simplify before continuing.

### Surgical Changes

- Touch only files and lines required by the request.
- Preserve existing style, comments, and nearby code unless the task requires changing them.
- Clean up only dead code introduced by your own change; mention unrelated dead code instead of deleting it.

### Goal-Driven Execution

- Convert the request into verifiable success criteria before editing.
- For multi-step work, state a short plan with a verification check for each step.
- Loop until the relevant tests, builds, or manual checks prove the goal is met.
<!-- karpathy-guidelines:end -->

<!-- karpathy-project-adjustment:start -->
## Project-Specific Karpathy Adjustment

This section localizes the Karpathy guardrails for `workspace-mineru/gormes-agent`. Source inspiration: https://github.com/forrestchang/andrej-karpathy-skills at commit `2c60614`.

- Project family: Gormes Go-native Hermes-compatible agent runtime.
- Local focus: Go-native agent runtime, TUI, gateway, tools, sessions, local memory, installer, docs, and progress.json delivery tracking.
- Stack cues: Go.
- Evidence to prefer: go test output, CLI smoke checks, doctor/onboard output, exact branch, progress.json updates, and compatibility notes against Hermes behavior.
- Surgical boundary: work from development rules in the repo; avoid Python/venv assumptions and avoid changing public CLI/runtime contracts without tests.
- Stop and ask when: a change affects agent behavior, persistence, gateway routing, provider config, install flow, or compatibility promises.
<!-- karpathy-project-adjustment:end -->
