---
name: gormes-skill-manager
description: Use when starting any substantial Gormes planning or building task, when unsure whether to use gormes-planner, gormes-builder, gormes-parity-auditor, gormes-tdd-slice, or gormes-interface-designer, or when repeated Gormes work suggests creating or improving a repo-local skill.
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

Canonical skill source lives under `development-skills/<name>/SKILL.md`.
The `.agents/skills/`, `.claude/skills/`, and `.codex/skills/` directories are
symlink loader views.

## Workflow

### 1. Classify The User Request

Start by locating the work in
`docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md`
and, when implementation intent exists, the matching row in
`docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md`.

Before picking a skill, detect the user's intent from verbs, artifacts, and
risk. Do not route only by subsystem name.

#### Intent Detection Ladder

Use this order when several skills could apply:

1. **Safety/repo state intent** — commit, push, release, PR, Greptile, CI,
   dirty tree, branch, install, running binary, gateway process, or local
   operator state. Route to `gormes-git`, `gormes-release`,
   `gormes-pr-check`, `gormes-greptile-loop`, `gormes-review-loop`,
   `gormes-review-scorecard`, `gormes-dev-runtime`, or `gormes-install` before
   planning feature work.
2. **Evidence-gathering intent** — "what is missing", "compare", "audit",
   "map parity", "actually implemented", "already implemented", "why does
   Hermes do X", "source refs", Hermes release notes, `docs/hermes-releases/`,
   `docs/hermes-releases/FEATURE-MATRIX.md`, `hermes-knowledge-graph.json`,
   external API docs, or a freshly tagged external dependency/release. Route to
   `gormes-hermes-parity`, `gormes-parity-auditor`, `gormes-openclaw-parity`,
   `gormes-pi-parity`, `gormes-context-sourcing`, or
   `gormes-architecture-zoomout` before builder work. For "actually
   implemented" sweeps, use `gormes-hermes-parity` plus `gormes-planner` to
   correct stale `missing` atoms before builder selection.
3. **Backlog-shaping intent** — plan, split, rows, roadmap, progress,
   acceptance, PRD, review finding becomes work. Route to
   `gormes-progress-slicer` or `gormes-planner`.
4. **Implementation intent** — build, fix, implement, make test pass, port one
   behavior, execute a row. Route to `gormes-builder` plus
   `gormes-tdd-slice`; add subsystem skills such as `gormes-provider-parity`,
   `gormes-browser-harness`, or `navivox-telegram-ui` only when they supply the
   needed contract.
5. **Design/refactor intent** — interface, package boundary, repeated logic,
   `cmd/gormes` domain extraction, module shape, cleanup after working behavior.
   Route to `gormes-interface-designer`, `cmd-internal-refactor`,
   `gormes-service-layer-refactor`, or `gormes-prototype-spike`.
6. **Communication/content intent** — README, landing page, dashboard screenshot,
   image-based dashboard asset, public claims, handoff-like status, or
   user-facing messaging. Route dashboard visuals to `dashboard-image-design`;
   route public copy to `gormes-readme` or `gormes-landing-web` unless the
   request is actually release prep.

#### Fast Signal Table

| User words or artifact | Usually means | First skill |
|---|---|---|
| `/goal`, development-goal iteration, DEV_GOAL markers, keep going, finish Gormes | persistent objective or runner contract | `gormes-goal` |
| `PR`, mergeable, checks, comments | PR readiness | `gormes-pr-check` |
| Greptile, 5/5, unresolved review | automated review loop | `gormes-greptile-loop` |
| score this, production-ready, no Greptile | local quality score | `gormes-review-scorecard` |
| CI failed, review feedback, make green | bounded fix loop | `gormes-review-loop` |
| release, tag, publish, version | release lane | `gormes-release` |
| commit everything, push development | git delivery | `gormes-git` |
| install.sh, setup, PATH, binary, gateway, sessions.db | local runtime | `gormes-dev-runtime` or `gormes-install` |
| provider, auth, model, streaming, rate limit | provider contract | `gormes-provider-parity` |
| browser, CDP, Browser Use, `/browser connect` | browser parity | `gormes-browser-harness` |
| Navivox, Telegram-like, Flutter chat/contact | mobile UI | `navivox-telegram-ui` |
| what is missing, compare Hermes/Honcho | parity discovery | `gormes-parity-auditor` or `gormes-hermes-parity` |
| Hermes release notes, `docs/hermes-releases/FEATURE-MATRIX.md`, or `hermes-knowledge-graph.json` | release/topology-seeded parity routing | `gormes-hermes-parity` then `gormes-planner` only if atoms/rows need edits |
| actually implemented, already implemented, stale missing atoms | evidence reconciliation before build | `gormes-hermes-parity` then `gormes-planner` |
| OpenClaw-only behavior | owned enhancement triage | `gormes-openclaw-parity` |
| Pi, pi.dev, pi-coding-agent, extension API, SDK/RPC harness, TUI components | harness technique donor triage | `gormes-pi-parity` |
| external API/library docs | source context | `gormes-context-sourcing` |
| tagged external repo release, `go get`, GitHub release, use module from release | dependency integration evidence | `gormes-context-sourcing` then `gormes-tdd-slice` |
| `goscrapling v0.1.0`, tagged sibling repo, scrape dependency, use from GitHub release | released-module E2E integration | `gormes-context-sourcing` then `gormes-tdd-slice` |
| architecture/planner/parity/builder loop, delivery loop extension | bounded delivery orchestration | `gormes-delivery-loop` |
| plan, roadmap, progress row | planner pass | `gormes-planner` |
| split plan/PRD/review into rows | vertical slicing | `gormes-progress-slicer` |
| implement row, build slice | builder pass | `gormes-builder` + `gormes-tdd-slice` |
| failing test, bug fix, TDD | red-green loop | `gormes-tdd-slice` |
| unfamiliar package, crosses modules | architecture map | `gormes-architecture-zoomout` |
| improve architecture, deep modules, reduce coupling, AI-navigable | architecture candidates | `gormes-architecture-zoomout` |
| interface, API boundary | design boundary | `gormes-interface-designer` |
| cmd-internal refactor, thin cmd/gormes, move command behavior to internal/app | bounded command-domain extraction | `cmd-internal-refactor` |
| duplicate mechanics, cleanup | refactor mechanics | `gormes-service-layer-refactor` |
| try designs, prototype | throwaway experiment | `gormes-prototype-spike` |
| README | public repo messaging | `gormes-readme` |
| landing, homepage, www.gormes.ai | website copy/UI | `gormes-landing-web` |
| dashboard screenshot, hero image, social card, image-based dashboard | visual asset design | `dashboard-image-design` |
| create/update skills | skill management | `gormes-skill-manager` |

Pick the primary intent:

- **Decide direction**: use the global `grill-me` skill and optionally `gormes-planner`; do not create a repo-local `grill-me` shadow skill.
- **Persistent long-running objective, `/goal` command, or development-goal runner prompt**: use `gormes-goal` first to set, inspect, pause, resume, clear, complete, or preserve the runner's final marker contract. Route concrete work through the smallest applicable Gormes skill chain, but keep development-goal `DEV_GOAL_*` markers as the final response tail.
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
- **Learn from Pi harness techniques** such as extension APIs, tool
  middleware, SDK/RPC embedding, TUI components, session trees, compaction,
  packages, prompt templates, safety gates, or provider hooks: use
  `gormes-pi-parity`. Pi is a donor for harness design, not a Hermes or
  OpenClaw parity contract.
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
- **Flutter Navivox Telegram-like chat/contact UI**: use `navivox-telegram-ui`, then `gormes-tdd-slice` for widget-test implementation. Use this for chat-thread chrome, bottom navigation removal, bubbles, composer, profile contacts, action sheets, voice affordances, or Telegram/Flutter clone reference research.
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
- **External library/framework/upstream source context** before planning or implementation: use `gormes-context-sourcing`, then route to the smallest parity/planner/builder skill.
- **Use a freshly tagged external Go module release** such as a sibling repo tag pushed to GitHub: use `gormes-context-sourcing` first to verify the module path, tag, release commit, public availability, and absence of local `replace` assumptions. Then use `gormes-tdd-slice` for a failing E2E/import test that proves Gormes consumes the released module, not a local checkout. If the integration changes public behavior or lacks a progress row, insert `gormes-progress-slicer`/`gormes-planner` before implementation.
  - For Juan's `goscrapling v0.1.0` handoff, preserve the supplied evidence (`/home/xel/git/sages-openclaw/workspace-mineru/goscrapling`, branch `main`, tag commit `ca1f046aa942c0739a73cb0715b67aec608b8e39`, tag `v0.1.0`, pre-tag validation) as context, but do not import from that sibling checkout. Verify the public GitHub module/tag with Go tooling, update Gormes via the release version, and write a failing E2E/TDD proof before implementation.
- **Architecture zoom-out, codebase architecture improvement, or unfamiliar cross-package work**: use `gormes-architecture-zoomout` before implementation; route unclear package boundaries to `gormes-interface-designer` and repeated mechanics to `gormes-service-layer-refactor`.
- **Bounded architecture -> planner -> parity -> builder cycles**: use `gormes-delivery-loop` when Juan explicitly asks for the chain or an extension that repeats it. Keep each iteration budgeted, progress-row-backed, test-validated, committed, and pushed before continuing.
- **Broad plan, PRD, parity gap, or review finding that needs progress rows**: use `gormes-progress-slicer`, then `gormes-planner` to update canonical progress surfaces.
- **Throwaway design, state-machine, protocol, or UI experiment**: use `gormes-prototype-spike`; route validated production work to `gormes-tdd-slice`.
- **Repeated runtime mechanics or service-layer cleanup** after a feature works: use `gormes-service-layer-refactor`; route unclear package boundaries to `gormes-interface-designer` first.
- **PR readiness audit before external review or merge**: use `gormes-pr-check`; route actionable findings to `gormes-review-loop` or `gormes-greptile-loop`.
- **Greptile review feedback, unresolved Greptile comments, or sub-5/5 confidence**: use `gormes-greptile-loop`; route behavior fixes through `gormes-tdd-slice` and stop only on fetched 5/5 evidence or documented blocker.
- **Local 1-5 production-readiness score when Greptile is unavailable**: use `gormes-review-scorecard`; it cannot replace CI or fetched Greptile evidence.
- **PR feedback, CI failures, review comments, or bounded review-to-green iteration**: use `gormes-review-loop`; route code behavior fixes through `gormes-tdd-slice`.
- **Design a Go interface/package boundary**: use `gormes-interface-designer`.
- **Refactor one `cmd/gormes` command domain into `internal/app/<domain>` without behavior changes**: use `cmd-internal-refactor`; keep CLI compatibility characterized and do not mix domains.
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
- **Create or improve skills**: use global `writing-skills` for the skill-doc
  red/green loop, use system `skill-creator` validation when available, plus
  this manager for Gormes routing. Fold repeated mistakes into existing
  class-level skills before creating a new one, and keep the update as
  process guidance rather than a session diary.
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

If more than one applies, choose a chain with at most three skills. Prefer
`evidence -> row shaping -> TDD` or `runtime proof -> parity contract -> TDD`.
Do not load every Gormes skill.

Use these composition rules:

- If the user asks to **do work now**, avoid a pure planner answer unless the
  row is missing or vague.
- If the user says to **use a release/tag from another repo**, never rely on a
  sibling checkout or local `replace` by default. Verify the upstream module tag
  first, then write an E2E/TDD proof that imports or exercises the released
  artifact. Treat supplied local release notes as evidence to verify, not as a
  substitute for public module/tag resolution.
- If the user gives **a concrete failure artifact**, reproduce or inspect that
  artifact before broad parity audits.
- If the user asks to **make it better** without an artifact, use the ladder:
  runtime/state preflight, then scorecard or architecture zoom-out, then one
  bounded fix.
- If the user asks for **multiple independent domains**, split the report into
  separate skill chains and ask before editing multiple domains unless the
  request clearly says fleet-wide or broad.
- If a route would exceed three skills, stop at the first handoff boundary and
  emit the next skill packet.

Feature-map gaps route to `gormes-parity-auditor` then `gormes-planner`.
Hermes release notes, `docs/hermes-releases/FEATURE-MATRIX.md`, and
`hermes-knowledge-graph.json` route to `gormes-hermes-parity` first because
they are study/navigation aids, not executable queues; hand off to
`gormes-planner` only after the active upstream source contract and matching
behavior atom are identified. Builder-ready rows route to `gormes-builder` and,
when tests are required, `gormes-tdd-slice`. Vague rows route back to
`gormes-planner`. Unclear package boundaries route to
`gormes-interface-designer` before implementation.

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
find development-skills -maxdepth 2 -name SKILL.md -print | sort
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
- `gormes-progress-slicer`
- `gormes-prototype-spike`
- `gormes-architecture-zoomout`
- `gormes-greptile-loop`
- `gormes-pr-check`
- `gormes-review-scorecard`
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

Create the skill under `development-skills/<name>/`. Then add symlinks:

```sh
ln -s ../../development-skills/<name> .agents/skills/<name>
ln -s ../../development-skills/<name> .claude/skills/<name>
ln -s ../../development-skills/<name> .codex/skills/<name>
```

### 5. Report The Routing Decision

Before doing substantial work, state:

- selected skill or skill chain;
- detected intent, including the user words/artifact that triggered it;
- feature-map area;
- progress row, if any;
- why it fits;
- fallback if the selected skill cannot proceed;
- whether a new skill is needed now.

If confidence is low, ask one clarifying question only when the answer would
change the first skill. Otherwise state the assumption and proceed with the
lowest-risk evidence-gathering skill.

Keep this short; then execute the chosen workflow.

Use this packet shape:

```text
selected_skill:
detected_intent:
trigger:
feature_map_area:
progress_row:
reason:
fallback:
new_skill_needed:
```

## Guardrails

- Do not let skill management replace delivery.
- Do not create side backlogs. Implementation intent goes into `docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md`, the single canonical parity inventory. Update it by editing the file directly (atoms are plain markdown, not a JSON schema).
- Loop/orchestrator commands require explicit user intent, progress-row-backed scope, validation gates, and operator controls; otherwise prefer bounded skill-driven passes.
- Use Context7 for external library/framework/API docs when required by repo instructions.
- Preserve dirty user work.
