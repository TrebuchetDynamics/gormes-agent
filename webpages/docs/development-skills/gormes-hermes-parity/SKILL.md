---
name: gormes-hermes-parity
description: Use when checking Gormes-vs-Hermes/Honcho parity, automatically discovering behavioral fidelity gaps across user-visible surfaces, handling stale upstream evidence, restructuring parity taxonomy, or turning drift reports into source-backed progress rows.
---

# Gormes Hermes Parity

## Repository Branch Rule

For Gormes work, stay on the existing `development` branch. Do not create or
use feature branches, short-lived branches, or git worktrees. If the checkout
is not on `development`, stop before editing and switch safely or report the
blocker.

## Role

This skill is the parity orchestrator. Keep it as the entry point for
Hermes/Gormes sweeps, but route detailed work to the smallest useful subskill
or reference. Do not delete or replace `gormes-hermes-parity`; do not collapse
`gormes-openclaw-parity` into it. OpenClaw-only discoveries route there.

Mission: run bounded, source-backed parity passes that keep Gormes pointed at
the finish line: Hermes in Go, with Goncho as the Honcho-compatible Go port.
The output is progress-row-ready evidence, focused validation, and routed next
work. Runtime implementation belongs to builder/TDD skills, not to a broad
parity sweep.

## Evidence Boundary

Allowed evidence: upstream source checkouts (`./hermes-agent`, `./honcho`,
falling back to siblings or `references/`), checked-in Gormes files, tests,
docs, generated progress data, temp fixtures, temp `GORMES_HOME`, and
sanitized user-provided transcripts.

Do not read live private config, memory, credentials, session stores, or home
directories such as `~/.hermes`, `~/.openclaw`, `~/.gormes`, `~/.claude`,
`~/.codex`, or `~/.agents` unless the selected row is explicitly migration or
runtime-home validation with sanitized fixtures.

## Invocation

| Invocation | Behavior |
|---|---|
| `gormes-hermes-parity` | Auto-select one behavior-fidelity pair from progress risk, stale refs, unmapped upstream classes, recent user-visible regressions, or narrow builder-ready gaps. |
| `gormes-hermes-parity <topic>` | Treat `<topic>` as a discovery seed, expand to adjacent behavior atoms, then pick the smallest coherent parity surface. |

A behavior-fidelity pair is one upstream Hermes/Honcho behavior family plus
the closest Gormes surface or missing row that should preserve it.

## Hard And Soft Dependencies

Hard dependencies block claims of parity:

- `development` branch and preserved dirty user work.
- Resolved `$HERMES_SRC` for Hermes parity claims.
- Resolved `$HONCHO_SRC` for Goncho/Honcho parity claims.
- Exact source refs for every `covered`, `missing`, `stale-upstream`, `owned`,
  or `excluded` classification.
- Focused validation that exercises real behavior; empty test selectors do not
  count.
- Canonical `progress.json` for implementation intent.

Soft dependencies sharpen output but do not always block:

- Live user transcripts when source/tests already prove the surface.
- Installed-binary checks when the pass is source-only and not runtime/install
  parity.
- OpenClaw checkout when the behavior exists in Hermes. OpenClaw-only ideas
  route to `gormes-openclaw-parity`.
- `./honcho` when the selected scope is unrelated to Goncho.
- Understand Anything graphs when present under
  `$HERMES_SRC/.understand-anything/knowledge-graph.json`; these accelerate
  source navigation but never replace exact upstream source refs.

## Understand Graph Accelerator

When a scoped Hermes graph exists, use it before broad `rg` sweeps to choose
entry files, layers, and tour anchors. Treat graph output as an index, not as a
parity oracle: every `covered`, `missing`, `stale-upstream`, `owned`, or
`excluded` claim still needs file/line source refs from `$HERMES_SRC`.

Current useful Hermes graph anchors, when available:

| Parity topic | Graph layer or tour anchor | Source files to verify |
|---|---|---|
| CLI command surface | `layer:cli-and-entrypoints`, tour "CLI Command Surface" | `hermes_cli/_parser.py`, `hermes_cli/commands.py`, `hermes_cli/plugins.py` |
| Agent runtime / skills | `layer:agent-runtime`, tour "Agent Runtime Core" | `agent/`, `skills/`, `agent/plugin_loader.py` |
| Provider/model behavior | `layer:providers-and-models`, tour "Provider and Model Integration" | `providers/`, `plugins/model-providers/**/plugin.yaml` |
| Gateway/channels | `layer:gateway-and-channels`, tour "Gateway and Messaging" | `gateway/`, `gateway/platforms/`, channel docs |
| TUI/web UI | `layer:tui-and-web-ui`, tour "TUI and Frontend Surfaces" | `ui-tui/src/app.tsx`, `web/src/`, `hermes_cli/` TUI files |
| Browser automation | `layer:browser-automation` | `plugins/browser/`, browser docs, browser tests |
| Plugins / ACP | `layer:plugins-acp-and-extensions`, tour "ACP Integration" | `plugins/registry.py`, `acp_adapter/`, `acp_registry/` |
| Cron/background work | `layer:cron-and-background-jobs`, tour "Cron and Background Work" | `cron/manager.py`, `cron/scheduler.py` |
| Validation evidence | `layer:tests-and-evaluation`, tour "Validation and Tests" | `tests/`, evaluation and benchmark files |

Graph query recipe:

```sh
GRAPH="$HERMES_SRC/.understand-anything/knowledge-graph.json"
test -f "$GRAPH" && node - "$GRAPH" <<'NODE'
const fs = require('fs');
const graph = JSON.parse(fs.readFileSync(process.argv[2], 'utf8'));
console.log('graph_commit=' + (graph.project?.gitCommitHash || 'unknown'));
for (const layer of graph.layers || []) {
  console.log(`${layer.id}\t${layer.name}\t${(layer.nodeIds || []).length}`);
}
for (const step of graph.tour || []) {
  console.log(`tour:${step.order}\t${step.title}\t${(step.nodeIds || []).join(',')}`);
}
NODE
```

For a topic-specific lookup, filter layer node IDs to candidate paths, then
read the files themselves:

```sh
node - "$GRAPH" 'browser|provider|gateway|tui|acp|cron|plugin' <<'NODE'
const fs = require('fs');
const graph = JSON.parse(fs.readFileSync(process.argv[2], 'utf8'));
const re = new RegExp(process.argv[3], 'i');
const nodes = new Map((graph.nodes || []).map(n => [n.id, n]));
for (const layer of graph.layers || []) {
  const hits = (layer.nodeIds || []).map(id => nodes.get(id)).filter(n => n?.filePath && re.test(n.filePath));
  if (hits.length) console.log(`\n${layer.name}\n` + hits.slice(0, 30).map(n => `- ${n.filePath}`).join('\n'));
}
NODE
```

Graph caveats to preserve in reports:

- Record the graph's `project.gitCommitHash` next to the actual
  `git -C "$HERMES_SRC" rev-parse --short HEAD` value.
- A graph generated by deterministic extraction may have sparse `imports`
  edges; absence of an import edge is not evidence that a behavior is absent.
- Orphan-node warnings are common for docs/config/infra; use them as coverage
  hints, not parity failures.
- Do not edit or commit generated graph artifacts as part of normal parity
  work unless the user explicitly asks for Understand graph maintenance.

## Subskill Routing

`gormes-hermes-parity` manages the subskills below. Use the smallest chain that
finishes the current behavior packet.

| Situation | Route to |
|---|---|
| Unsure how to continue or multiple subsystems are involved | `gormes-skill-manager` |
| Need deeper Hermes/Honcho source comparison | `gormes-parity-auditor` |
| Need progress rows, taxonomy, feature-map, or docs reshaped | `gormes-planner` |
| Need one existing row implemented | `gormes-builder` |
| Need tests-first runtime behavior | `gormes-tdd-slice` |
| Need package/API boundary design before a row is buildable | `gormes-interface-designer` |
| Provider/auth/model/rate-limit/streaming behavior | `gormes-provider-parity` |
| Browser automation, Browser Use, CDP, or `/browser connect` | `gormes-browser-harness` |
| Local run/install/runtime, `bin/gormes`, PATH, gateway ownership, locks | `gormes-dev-runtime` |
| Need Go donor shape before coding | `gormes-references` |
| Useful OpenClaw-only behavior absent from Hermes | `gormes-openclaw-parity` |

## Provider Auth And Setup Incidents

Treat install, setup, config writing, doctor/onboard output, Telegram settings,
provider auth, and Codex auth import reports as user-visible parity surfaces,
not as incidental local-machine issues.

When a report touches these surfaces:

- Identify the active Hermes contract first, then route implementation through
  `gormes-provider-parity`, `gormes-dev-runtime`, and `gormes-tdd-slice` as
  needed.
- Use isolated `GORMES_HOME`, `HERMES_HOME`, and `CODEX_HOME` fixtures. Never
  read real private homes or print user secrets; use synthetic tokens/JWTs.
- For Codex auth, Hermes can find `CODEX_HOME/auth.json` or
  `~/.codex/auth.json`, import fresh Codex CLI tokens into Hermes-owned auth
  state, and fall back to device-code when the file is missing, stale, or bad.
  Gormes parity means fresh Codex CLI auth imports into Gormes' own credential
  pool, stale imports do not block device-code login, and all evidence is
  redacted.
- Setup regressions need TDD coverage for the exact broken surface:
  `config set`, nested config loading, `onboard`, `doctor --offline`,
  `gateway status --json`, and source/binary/install smoke when PATH or
  published binaries are in scope.
- Do not infer provider readiness from `hermes.api_key` alone. Credential-pool
  providers such as `openai-codex` must be visible as configured in
  `auth status`, `doctor --offline`, and `onboard`; failures here are parity
  bugs because operators see them as broken setup.
- Config writer regressions must round-trip through `config.Load`; list-shaped
  fields such as `telegram.allowed_user_ids` must be written as TOML arrays,
  while secrets such as Telegram bot tokens stay in dotenv and out of reports.

When delegating or handing off, emit a task packet:

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

## Progressive References

Load only the reference needed for the current pass:

| Need | Read |
|---|---|
| Canonical terms and concise parity vocabulary | `references/domain-language.md` |
| Behavior atom matrix, inventory searches, source buckets | `references/behavior-atoms.md` |
| Which upstream file is authoritative for a surface | `references/active-upstream-contracts.md` |
| Live transcripts, repeated operator failures, channel-visible artifacts | `references/operator-evidence.md` |
| Owned/excluded divergence, ADR guidance, taxonomy refactors | `references/taxonomy-and-owned-divergence.md` |
| Feedback loops, vertical slices, validation gates, report shape | `references/validation-and-feedback-loops.md` |

## Workflow

1. Establish baseline. Refresh the in-repo `hermes-agent` submodule to upstream
   `main` before reading parity refs so the sweep classifies against current
   upstream truth, not a stale checkout:

```sh
git status --short --branch
pwd
git rev-parse --show-toplevel
go run ./cmd/progress validate
git rev-parse --short HEAD
# Refresh hermes-agent (now a git submodule pinned to NousResearch/hermes-agent
# main) to upstream HEAD before classifying behavior atoms. Falls back to a
# plain fetch+ff-only if the path is a legacy clone, or no-op if absent.
git submodule update --init --remote -- hermes-agent 2>/dev/null \
  || git -C hermes-agent fetch --quiet origin main && git -C hermes-agent merge --ff-only origin/main 2>/dev/null \
  || true
HERMES_SRC="$(for p in ./hermes-agent ../hermes-agent references/hermes-agent; do [ -d "$p" ] && [ -f "$p/hermes_cli/main.py" ] && { printf '%s\n' "$p"; break; }; done)"
HONCHO_SRC="$(for p in ./honcho ../honcho references/honcho; do [ -d "$p" ] && { printf '%s\n' "$p"; break; }; done)"
git -C "$HERMES_SRC" rev-parse --short HEAD 2>/dev/null || true
test -z "$HONCHO_SRC" || git -C "$HONCHO_SRC" rev-parse --short HEAD 2>/dev/null || true
```

If `git submodule update --remote` advances the pinned submodule sha and the
parity sweep ends up landing builder rows, treat the bumped sha as part of the
parity-driven change set: stage `hermes-agent` in the parent repo and let
`gormes-git` commit it alongside the row work. Do not commit a submodule bump
on its own without parity evidence; record the upstream sha in
`source_shas:` of the final report regardless.

2. Choose one bounded surface. If the user asks for everything, produce a
   subsystem map and the next three concrete passes.
3. If `$HERMES_SRC/.understand-anything/knowledge-graph.json` exists, query its
   layers/tour first to pick likely source files; then read those files before
   making parity claims.
4. Load only the reference file needed for the chosen surface.
5. Inventory upstream behavior as behavior atoms, then identify the closest
   Gormes code, tests, docs, or progress row.
6. Pick the active upstream contract before comparing historical refs.
7. Classify each atom as `covered`, `planned`, `vague`, `missing`,
   `stale-upstream`, `blocked`, `owned`, or `excluded`.
8. Choose the feedback loop before claiming coverage or handing off.
9. Route to the smallest subskill chain. One behavior atom should become one
   vertical implementation slice.
10. Record implementation intent only in `progress.json` — the single logical
    backlog, accessed via `internal/progress.Load`/`cmd/progress` (monolithic
    file or split/per-module layout, transparently; never hand-parse members).
    Do not create side queues.

## Guardrails

- Do not implement runtime code in a broad parity sweep. Use builder/TDD for a
  selected row after the upstream contract is identified.
- Treat named examples as discovery seeds, not as the whole scope.
- Do not mark vague umbrella rows complete.
- Do not silently accept owned divergence; name the compatibility boundary.
- Do not use old Hermes `cli.py` prompt-toolkit refs as full-screen TUI truth
  when current `ui-tui` Ink files cover the behavior.
- Do not infer absence from missing Understand graph edges; graph data is a
  navigation aid and source reads remain authoritative.
- Keep source checkout, Gormes runtime home, installer-managed checkout, and
  upstream Hermes checkout separate in every note.
- Preserve dirty user work. If HEAD advances during a run, re-audit before
  committing or pushing.

## Validation

If only this skill or references changed:

```sh
python3 /home/xel/.codex/skills/.system/skill-creator/scripts/quick_validate.py docs/development-skills/gormes-hermes-parity
find -L .agents/skills .claude/skills .codex/skills -maxdepth 2 -name SKILL.md -print | sort
go run ./cmd/progress validate
git diff --check
```

If progress/docs rows changed, also run:

```sh
go run ./cmd/progress write
go run ./cmd/progress validate
go test ./internal/progress -count=1
go test ./webpages/docs -count=1
git diff --check
```

Finish with:

```text
scope:
source_shas:
graph_refs:
upstream_refs:
gormes_refs:
evidence_boundary:
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
