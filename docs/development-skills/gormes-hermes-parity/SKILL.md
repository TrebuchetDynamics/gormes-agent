---
name: gormes-hermes-parity
description: Use when running a recurring or periodic Gormes-vs-Hermes parity sweep, checking overall Hermes-in-Go progress, refreshing parity definitions, safely renaming or restructuring parity taxonomy, delegating follow-up tasks through gormes-skill-manager, or recording source-backed parity evidence into progress.json.
---

# Gormes Hermes Parity

## Mission

Run a bounded recurring parity sweep that keeps Gormes pointed at the real
finish line: Hermes in Go, with Goncho as the Honcho-compatible Go port. This
skill coordinates audit and planning work. It records progress in the canonical
roadmap and hands implementation to builder skills; it does not create a second
queue.

Use this when the user says `gormes-hermes-parity`, asks for a periodic parity
check, asks "what is left for full parity?", or wants ambiguous parity goals
turned into source-backed progress rows. It may also reshape parity taxonomy,
feature-map sections, progress row structure, and public progress wording when
the current names or grouping hide the real Hermes contract.

## Skill Chain

Default chain:

```text
gormes-hermes-parity
  -> gormes-skill-manager for routing follow-up tasks
  -> gormes-parity-auditor
  -> gormes-planner
  -> gormes-builder / gormes-tdd-slice for selected implementation rows
```

Keep each run bounded. If the user says "everything", produce a subsystem map
and the next three concrete passes instead of trying to audit the whole repo in
one turn.

When the user reports a concrete parity bug and says "continue working", this
skill may hand the current bounded behavior directly to `gormes-tdd-slice`
after the upstream contract is identified. Still keep the slice narrow, prove a
RED test first, and update the canonical row evidence after GREEN.

## Repeated Operator Lessons

Fold repeated live-testing failures into every parity pass. These are not
side preferences; they are Hermes compatibility evidence until Gormes has a
source-backed reason to diverge.

| Report shape | Treat it as | First action |
|---|---|---|
| `gormes` asks for `hermes gateway start`, mentions Hermes `api_server`, or depends on `~/.hermes` | runtime/install parity bug | Route through `gormes-dev-runtime`; prove binary path, `GORMES_HOME`, and installed source before changing UX code. |
| `go run`, `./bin/gormes`, and installed `gormes` behave differently | surface-selection bug | Test all three only with explicit paths/homes; never infer installed behavior from dirty source. |
| `sessions.db` locked while switching binaries | process/home ownership issue | Use gateway status/stop or isolated `GORMES_HOME`; do not delete the database. |
| extra `⏳`, duplicate assistant reply, visible `tool iteration limit exceeded`, leaked tool-call text, or stale `Hermes` label | user-visible parity bug | Preserve the exact transcript and build a channel/TUI fixture; a correct final answer is not enough. |
| "Gormes bot is you; Mineru bot is Hermes" or persona/reset mismatch | prompt/template parity bug | Inspect Hermes default soul/prompt sources and Gormes `internal/agenttemplate` before planning or coding. |
| `install.sh` behavior is confusing | installer contract issue | Remember final-user installs clone/build a branch, while development validation uses `go run ./cmd/gormes` or `./bin/gormes`. |

Do not tell the operator they must push `main` to validate local development
changes. Pushing matters only when validating the installer-managed branch
path; dirty-checkout behavior belongs to `go run` or a locally rebuilt binary.

## Delegation Through Skill Manager

Use `gormes-skill-manager` whenever the sweep discovers work that is bigger
than the current bounded pass, crosses subsystem boundaries, or requires a
different delivery mode.

Delegate by producing small task packets, not loose TODOs:

```text
task:
scope:
feature_map_area:
progress_row:
recommended_skill_chain:
source_refs:
observed_transcript_or_terminal_output:
visible_artifacts:
write_scope:
red_test_hint:
validation:
blocked_by:
```

Route typical follow-ups this way:

| Follow-up | Ask skill manager to route to |
|---|---|
| Need more upstream comparison | `gormes-parity-auditor` |
| Need progress rows, taxonomy, or docs reshaped | `gormes-planner` |
| Need one row implemented | `gormes-builder` |
| Need tests-first runtime behavior | `gormes-tdd-slice` |
| Need package/API boundary design | `gormes-interface-designer` |
| Need provider/auth/model behavior | `gormes-provider-parity` |
| Need browser automation parity | `gormes-browser-harness` |
| Need install/build/run validation, PATH shadowing, source clones, or session DB locks | `gormes-dev-runtime` |
| Need donor Go implementation shape | `gormes-references` |

When the agent runtime supports parallel workers and the user has authorized
delegation, independent task packets may run in parallel. Keep write scopes
disjoint, tell workers they are not alone in the codebase, and make each worker
report changed files and validation. Otherwise, record the packets as the next
builder-ready work and stop.

## Default Periodic Prompt

Use this as the general-purpose recurring prompt:

```text
Use $gormes-hermes-parity to run a bounded Hermes/Gormes parity sweep. If I do
not name a scope, choose the highest-risk or stalest incomplete surface from
progress.json and recent upstream evidence. Compare ../hermes-agent, ../honcho,
and ../gbrain against current Gormes, classify each behavior, update canonical
progress evidence and rows when needed, run validation, and finish with the next
builder-ready rows. If the existing taxonomy is misleading, safely rename or
restructure it with source-wide reference checks and compatibility notes. Use
gormes-skill-manager to route or delegate follow-up task packets.
```

## Parity Definitions

Use one of these labels for every behavior in scope:

| Label | Meaning |
|---|---|
| `strict` | Gormes must match upstream names, inputs, outputs, errors, side effects, and registration exactly. |
| `functional` | Gormes preserves the user/operator contract, but the Go internals or provider shape may differ. This is the default target. |
| `owned` | Gormes intentionally diverges or extends Hermes. The row must explain why and how compatibility is preserved or why it is not required. |
| `excluded` | Upstream behavior is intentionally not part of Gormes. This needs explicit source-backed rationale and user-visible risk noted. |

Use these progress classifications during the sweep:

| Classification | Record it as |
|---|---|
| `covered` | Implemented, tested, and source-backed. Mark complete only with repository evidence. |
| `planned` | Represented by a builder-ready row with acceptance and tests. |
| `vague` | A row exists but is too broad, ambiguous, missing tests, or missing source refs. Refine or split it. |
| `missing` | No useful Gormes code or progress row exists. Add the smallest source-backed row. |
| `stale-upstream` | Existing evidence points at old upstream behavior. Refresh refs and acceptance. |
| `blocked` | Cannot proceed because a dependency, source checkout, credential, or interface decision is absent. Record the blocker explicitly. |

When strict and functional parity conflict, prefer functional parity only when
the difference is documented as `owned` or the public Hermes contract is still
preserved.

## Periodic Workflow

### 1. Bound The Sweep

Pick one surface unless the user named several independent scopes:

- web/tools and native tool descriptors;
- provider/auth/model routing and usage;
- CLI/config/migration command tree;
- gateway, channels, and operator flows;
- sessions, memory, and Goncho/Honcho compatibility;
- prompt/context/runtime loop behavior;
- plugins, skills, browser automation, packaging, docs, or public progress.

If no scope is named, choose the highest-value surface by checking incomplete
rows, stale source refs, recent upstream changes, and user-visible risk.

### 2. Establish Baseline

Run lightweight discovery before editing:

```sh
git status --short --branch
pwd
git rev-parse --show-toplevel
which -a gormes || true
go run ./cmd/progress validate
git rev-parse --short HEAD
git -C ../hermes-agent rev-parse --short HEAD || true
git -C ../honcho rev-parse --short HEAD || true
git -C ../gbrain rev-parse --short HEAD || true
```

Then inspect the relevant feature-map section, upstream coverage ledger,
matching `progress.json` rows, and current Gormes packages. Use `rg`, `find`,
and `jq`; do not infer parity from file names alone.

Before editing, identify unrelated dirty work. Treat new or modified files that
are outside the selected slice as user/parallel-agent work: do not revert them,
do not fold them into the parity claim, and call out any validation impact.

Keep four paths separate in every parity note: source checkout, Gormes runtime
home (`GORMES_HOME` / default `~/.gormes`), installer-managed checkout, and
upstream Hermes checkout. In the Sages workspace family, an agent may edit from
`workspace-mineru` while runtime/dev config belongs in a `workspace-gormes` or
`GORMES_HOME` location; discover that relationship instead of hard-coding an
operator-specific path into code, tests, rows, or skills.

### 3. Inventory Upstream Behavior

For the scoped surface, list exact upstream files, symbols, commands, tests,
docs, request/response contracts, fixtures, and registration points from:

- `../hermes-agent`
- `../honcho`
- `../gbrain`

If a sibling checkout is missing, record the missing source as a blocker. Do
not replace upstream evidence with memory or guesses.

### 3a. Pick The Active Upstream Contract

Hermes has multiple historical implementations. Pick the active user-visible
contract before comparing:

| Surface | Prefer these upstream refs first | Legacy refs are useful for |
|---|---|---|
| Full-screen TUI / visual UX | `../hermes-agent/ui-tui/src/components/appLayout.tsx`, `appChrome.tsx`, `messageLine.tsx`, `thinking.tsx`, related `ui-tui/src/__tests__` | Older `cli.py` prompt_toolkit details only when current Ink does not cover the behavior |
| Classic CLI prompts/status | `../hermes-agent/cli.py` and `../hermes-agent/tests/cli` | Current Ink only for shared semantics |
| Telegram/channel-visible behavior | channel adapters, gateway event handlers, tool progress renderers, Telegram tests | TUI-only renderers only as shape hints |
| Install/runtime behavior | `install.sh`, packaging docs, command startup paths, current Gormes dev-runtime skill | Historical installer notes only as migration context |
| Prompt identity, memory, and defaults | `../hermes-agent/hermes_cli/default_soul.py`, `agent/prompt_builder.py`, `tools/memory_tool.py`, `agent/memory_manager.py`, gateway reset tests | Old prompt snippets only as migration context |
| Skills and template expansion | `../hermes-agent/agent/skill_commands.py`, `skill_preprocessing.py`, `skill_utils.py`, `tools/skills_tool.py`, `tools/skills_sync.py` | Website skill docs only as catalog evidence |
| Streaming/tool-call/channel UX | `../hermes-agent/tests/gateway/test_stream_consumer.py`, `test_update_streaming.py`, channel adapter tests, tool progress renderers | Model-loop code only when the UX is emitted there |
| Tool loop and iteration budget | `../hermes-agent/run_agent.py:_handle_max_iterations`, `run_conversation max_iterations`, tool-loop tests | Provider adapters only when the provider malformed tool calls |

If sources disagree, classify the old behavior as `stale-upstream` unless the
user explicitly wants legacy parity. Update row refs and acceptance so future
agents do not keep re-implementing retired chrome or command behavior.

### 4. Compare To Gormes

For each upstream behavior:

1. Identify the closest Gormes package, command, tool, fixture, or missing area.
2. Classify the parity state as `covered`, `planned`, `vague`, `missing`,
   `stale-upstream`, `blocked`, or `owned`.
3. Assign the parity definition: `strict`, `functional`, `owned`, or `excluded`.
4. Link exact source refs and current Gormes evidence.
5. Decide whether to update the feature map, coverage ledger, or progress row.

Use `gormes-parity-auditor` for the detailed source comparison when the surface
is not already clear. Use `gormes-planner` before editing rows or planning docs.

### 4a. Preserve Operator Evidence

For live parity reports, capture the artifact before reducing it to a row:

- channel or surface: Telegram, TUI, CLI, installer, gateway log, etc.;
- exact user input and visible output, including duplicate messages;
- transient status artifacts such as hourglass, typing, edits, deletes, and
  "still working" messages;
- whether the final content was correct but surrounding UX was wrong;
- which binary/home/source surface was running.

This evidence belongs in the task packet or progress note. Do not paraphrase a
duplicate-message, tool-loop, or hourglass bug into "tool calling failed"; the
message sequence is the contract future tests need.

When the observed artifact is a channel-visible tool-progress block like:

```text
📚 skill_view: "plan"
📋 todo: "planning 5 task(s)"
📖 read_file: "/path/..."
🐍 execute_code: "from hermes_tools import terminal, se..."
```

pin the contract to Hermes gateway progress, not only the TUI. Start from
`../hermes-agent/gateway/run.py:progress_callback`,
`agent.display.build_tool_preview`, and `agent.display.get_tool_emoji`.
The short channel form is `emoji tool_name: "preview"` or
`emoji tool_name...`; `all/new` mode truncates previews to
`display.tool_preview_length` or 40 characters, `new` suppresses consecutive
same-tool names, and consecutive identical rendered lines collapse to
`(×N)`. `verbose` uses `emoji tool_name([arg_keys])` plus JSON args.
Compare Gormes against `internal/kernel/toolexec.go:toolCallPreview` and
`internal/gateway/render.go:FormatToolProgressPlain`. Common drift points are
`todo merge=true` needing `updating N task(s)`, `execute_code` accepting the
Hermes Python-only `code` schema with `hermes_tools` wrappers, and model-visible
`read_file` / `search_files` implementations rather than renderer-only labels.

### 5. Record Progress

Use only canonical progress surfaces. Do not create side TODO files, private
ledgers, or prompt-only backlog lists.

For each gap that needs work, update or create a row with:

- source refs from upstream and Gormes;
- parity definition and classification;
- observable contract;
- acceptance and done signal;
- focused test commands;
- write scope;
- dependencies and blockers;
- `ready_when` and `not_ready_when`.

If a row is complete, only mark it complete when code, tests, and docs evidence
prove the behavior. If a behavior is `owned`, document the divergence and the
compatibility boundary in the row.

If parity docs or coordinator briefs still describe a row as `regressed`,
`planner refinement`, or a top-P0 blocker after `progress.json` and source/tests
show the row is already complete, do a docs/progress reconciliation slice instead
of creating new runtime work. Verify the completed row's focused tests first,
update the parity matrix/detail sections and coordinator next-slice ordering to
remove stale blocker language, run the progress/docs gates, and commit only the
docs/progress surfaces.

When a progress command fails on an unrelated row while recording evidence,
fix only schema-level metadata that blocks validation and is objectively wrong
(for example invalid `trust_class` values). Do not silently rewrite unrelated
contracts or claim that blocker as part of the selected parity slice.

### 6. Safe Taxonomy And Restructure Mode

Use this mode when parity labels, feature-map headings, phase/subphase names,
row names, public progress wording, or package-level terminology no longer
match the upstream contract.

Allowed in the same parity sweep:

- rename or split parity labels and classifications;
- rename feature-map headings or coverage-ledger categories;
- split, merge, or regroup progress rows and subphases;
- update generated progress docs and `www.gormes.ai` progress data;
- update skill-routing docs when the workflow taxonomy changes.

For runtime Go identifiers, commands, tool names, config keys, database fields,
or public APIs, first decide whether the rename is internal, compatibility
preserving, or breaking. Internal renames can be handled as refactors. Public
renames need aliases, migration notes, or a builder-ready compatibility row.

Follow this safety loop:

1. Write a mapping table before editing: `old name -> new name`, scope,
   source refs, and compatibility decision.
2. Use `rg -n` to find every current reference across `cmd`, `internal`,
   `docs`, `www.gormes.ai`, skills, tests, and generated data.
3. Update structured data with structured tools when practical. Preserve
   `progress.json` schema fields and row identity unless the row split/merge is
   the point of the refactor.
4. Keep user-facing aliases for any public name unless a source-backed decision
   says the compatibility break is intentional.
5. Regenerate derived docs with `go run ./cmd/progress write`.
6. Re-run `rg -n` for old terms. Remaining references must be intentional
   history, migration, or compatibility notes.
7. Run the full validation set for the touched surfaces.

Large restructures should land as one no-behavior-change taxonomy migration,
then separate builder rows for runtime behavior. Do not mix a broad rename with
new feature implementation unless the user explicitly asks for that combined
slice and the tests prove both.

### 7. Validate

After progress or docs edits, run:

```sh
go run ./cmd/progress write
go run ./cmd/progress validate
go test ./internal/progress -count=1
go test ./docs -count=1
git diff --check
```

If only this skill or routing docs changed, validate the skill shape and run
`git diff --check`.

If upstream coverage claims changed, also run:

```sh
go test ./docs -run TestUpstreamCoverageLedgerMatchesSourceClasses -count=1
```

If `www.gormes.ai` data changed, also run:

```sh
(cd www.gormes.ai && go test ./... -count=1)
```

If runtime Go identifiers, commands, tools, config, persistence, or public APIs
changed as part of a taxonomy refactor, also run the focused package tests and
then:

```sh
go test ./... -count=1
```

### 8. Report The Sweep

Finish with this compact report:

```text
scope:
upstream_refs:
gormes_refs:
parity_definition:
classification_summary:
taxonomy_changes:
rows_changed:
compatibility_notes:
delegated_task_packets:
validation:
next_builder_rows:
blockers:
```

Include exact file paths and commands. Do not claim "full parity" unless the
feature map, coverage ledger, progress rows, and validation all support it.

## Guardrails

- Do not implement runtime code in a parity sweep. Create or refine builder-ready
  rows, then use `gormes-builder` and `gormes-tdd-slice` for implementation.
  Exception: when the user asks to continue a concrete parity bug in the same
  turn, use `gormes-tdd-slice` for one focused behavior after source comparison.
- Do not treat a passing unit test as parity unless upstream behavior was
  compared and source refs are recorded.
- When a progress row's focused test command passes with Go's `[no tests to run]`
  or an equivalent empty selector, do not count that as validation. Treat the
  row's fixture/test-command evidence as stale, inspect current tests and
  implementation, then do a planner/progress correction before builder handoff
  or parity claims.
- Do not mark vague umbrella rows complete.
- Do not silently accept owned divergences. Name them and explain the boundary.
- Do not perform broad taxonomy renames without an old-to-new mapping, `rg`
  reference checks before and after, generated-doc refresh, and validation.
- Preserve dirty user work and unrelated pending changes.
- Record the baseline `git rev-parse --short HEAD` during preflight, then recheck
  `git status --short --branch` and HEAD immediately before committing or
  pushing. If HEAD advanced or new commits appeared during the run, treat that
  as concurrent work: do not push a mixed slice or layer progress edits on top
  of it unless you re-audit the new HEAD, restate ownership of only your files,
  and rerun focused validation.
- Do not use old Hermes `cli.py` prompt_toolkit TUI refs as the source of truth
  for current full-screen TUI UX when `ui-tui` Ink files cover the behavior.
- Do not stop at "it works"; parity bugs often hide in visible details:
  duplicate replies, hourglass/status messages, boxed composer chrome, stale
  product labels, tool-progress exposure, hidden retries, and platform-specific
  formatting.

## Recent Failure Patterns To Check

- **TUI looks bad:** inspect current Hermes Ink first. Common Gormes regressions
  are rounded input cards, repeated prompt glyph rows, idle `phase:` debug
  chrome, standalone full-width input rules, stale `⚕ model │ ctx` status
  footers, assistant rows labeled as Hermes, and duplicate assistant output
  from history plus live draft.
- **Installer/dev runtime confusion:** route through `gormes-dev-runtime`.
  Validate whether the user is running `go run ./cmd/gormes`, `bin/gormes`, or
  installed `~/.gormes/bin/gormes`; keep Gormes independent from Hermes
  api_server startup; handle `sessions.db` locks with explicit owner/status or
  in-memory fallback evidence.
- **Workspace identity drift:** do not let `workspace-mineru`,
  `workspace-gormes`, `~/.gormes`, `$HOME/.gormes/gormes-agent`, and
  `../hermes-agent` collapse into one concept. If a test needs a real-ish home,
  set `GORMES_HOME` or use temp fixtures. If code needs a default, use config
  APIs and discovery, never a developer's absolute path.
- **Agent reset/default template drift:** check the existing
  `Gormes agent template reset command` progress row plus
  `internal/agenttemplate` and `cmd/gormes/agent_reset_test.go` before opening
  new work. Follow-ups should refine that behavior, not create duplicate reset
  rows.
- **Skills/tool parity:** if platform-visible behavior depends on skills, check
  both tool registration (`skills_list`, `skill_view`) and the agent persona /
  template defaults. Hidden channel UX matters as much as CLI output.
- **Telegram/channel transcript bugs:** preserve the user's before/after
  transcript as evidence. Count outbound messages, edits, deletes, transient
  status messages, and final answer text; duplicate final replies and visible
  hourglass/status messages are regressions even when the final content is
  correct.
- **Tool iteration / bad tool-calling reports:** first check the completed
  `Kernel tool loop` row, `internal/kernel/kernel.go`, and
  `internal/kernel/tools_test.go`. Hermes defaults to a 90-turn budget and asks
  for one toolless summary on exhaustion. A raw
  `tool iteration limit exceeded (10)` reaching Telegram/TUI is either stale
  binary/runtime evidence or a channel finalization bug unless the kernel
  fixtures fail.
- **Progress drift:** `progress.json` is canonical. After editing it, run
  `go run ./cmd/progress write`; if generated docs change, include them and
  validate. Never create side queues.
