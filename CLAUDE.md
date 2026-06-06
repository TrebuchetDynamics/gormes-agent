# CLAUDE.md — gormes-agent

Claude, claudeu, and Claude Code agents must follow `AGENTS.md` as the
canonical repository contract.

## Branch and CI safety rule

Main must always stay green. This is mandatory memory for Claude, claudeu,
Claude Code, and workspace-mineru / workspace-mimeru agents.

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

## Mandatory skill routing

Before doing any substantive work in this repository, select and use at least
one repo-local skill. Canonical skill files live under
`development-skills/<name>/SKILL.md`; `.claude/skills/`,
`.agents/skills/`, and `.codex/skills/` are symlink loader views. If the right
skill is unclear, start with `gormes-skill-manager`.

Default routing:

| Work type | Required skill |
|---|---|
| Unsure which workflow applies, or deciding whether a new skill is needed | `gormes-skill-manager` |
| Mapping Hermes/Honcho parity gaps | `gormes-parity-auditor` |
| Fixing provider/auth/client/model-routing/usage/rate-limit parity bugs | `gormes-provider-parity` |
| Browser automation parity, Browser Use, browser-harness, CDP, or `/browser connect` work | `gormes-browser-harness` |
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
`development-skills/`, then symlink it into `.claude/skills/`,
`.agents/skills/`, and `.codex/skills/`. Do not proceed skill-less on
planning, building, parity analysis, interface design, TDD implementation, or
skill maintenance.

`gormes-planner` and `gormes-builder` mean manual skill-guided work. The old
autonomous command binaries were intentionally removed; do not recreate
`cmd/planner-loop` or `cmd/builder-loop` as part of Gormes delivery. If a
future orchestrator is needed, plan it as a fresh subsystem with new names,
interfaces, and progress rows.

## Core rule

Gormes is complete only when its runtime, protocol, and
tool/provider/gateway/session behavior are Hermes in Go, with Goncho as the
Honcho-compatible Go port inside Gormes. The interactive `gormes chat` TUI
presentation (welcome screen, chrome, styling, streaming affordances) is a
Gormes-owned surface that may diverge from Hermes `ui-tui` by design;
presentation parity is not a completion gate. This owned divergence is tracked
in `progress.json` (Phase 8.D, "Gormes-owned chat TUI" rows) per the
methodology-first strategy. Keep all implementation intent in
`webpages/docs/content/building-gormes/architecture_plan/progress.json`; do not create
parallel queues.

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
