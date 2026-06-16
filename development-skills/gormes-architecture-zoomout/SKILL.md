---
name: gormes-architecture-zoomout
description: Use when an agent is unfamiliar with a Gormes code area, a change crosses package boundaries, or a refactor needs a source-backed module map before edits.
---

# Gormes Architecture Zoom-Out

Use this before changing an unfamiliar subsystem, when a planned change spans packages and callers, or when the user asks to improve codebase architecture.

Inspired by `mattpocock/skills` `zoom-out` and `improve-codebase-architecture`; adapted for Gormes source evidence, progress rows, branch rules, and Hermes/Honcho parity boundaries.

## Trigger Examples

- “I don’t know where this behavior lives.”
- A setup, TUI, provider, gateway, session, tool, or Goncho change crosses package boundaries.
- A refactor proposal lacks caller/data-flow evidence.
- Tests are green but user-visible behavior is still uncertain.
- The user asks for architecture improvement, deeper modules, reduced coupling, AI-navigability, or better test seams.

## Modes

Pick one mode before exploring:

- **Zoom-out map:** answer where behavior lives and what skill should run next.
- **Architecture review:** find 2-4 deepening opportunities in one subsystem.
- **Refactor preflight:** decide whether a proposed extraction is worth doing before code changes.

Default to zoom-out map for vague debugging. Use architecture review when the user says “improve codebase architecture,” “find refactors,” “make this more testable,” “reduce coupling,” or “AI-navigable.”

## Workflow

1. **Protect the worktree first.** Run `git status --short --branch --untracked-files=all`. If unrelated dirty work exists, name it in the report, avoid broad formatters/regenerators, and stage only explicit architecture-skill files.
2. **Read local maps first.** Start with the current repository root's `codemap.md` (`git rev-parse --show-toplevel`); read folder `codemap.md` when present.
3. **Choose one subsystem.** Do not review the whole repository at once. Good first cuts are `cmd/gormes` setup/CLI, `internal/tools`, `internal/gateway`, `internal/channels/<name>`, `internal/goncho`, `internal/memory`, `internal/tui`, or `internal/provider`.
4. **Run targeted discovery.** Use commands like these, scoped to the subsystem:

   ```sh
   find <area> -maxdepth 2 -type f -name '*.go' | sort
   rg -n "type .*interface|func New|func \(.*\)|TODO|panic\(|os\.Getenv|http\.Client|time\.Now|json\.Marshal|AtomicReplace|Status|Evidence" <area>
   rg -n "<public command/tool/error/status text>" cmd internal webpages/docs/content/building-gormes
   go test ./<package> -run '<focused existing test pattern>' -count=1
   ```

5. **Map the subsystem.** List modules, owners, entry points, data flow, persistence, and test surfaces.
6. **Find user-visible contracts.** CLI flags, TUI text, gateway protocol, files on disk, progress rows, and Hermes parity refs.
7. **Evaluate module depth.** Use these terms consistently:
   - **Module:** package/function/type with an interface and implementation.
   - **Interface:** everything callers must know: types, invariants, errors, config, ordering, persistence, and tests.
   - **Implementation:** behavior hidden behind the interface.
   - **Seam:** where alternate behavior can be injected without editing callers.
   - **Adapter:** concrete implementation behind a seam.
   - **Depth:** leverage behind a small interface. Deep modules improve locality; shallow modules move complexity to callers.
8. **Classify architecture smells.** Prefer candidates that match one or more of these Gormes-specific smells:
   - **Scattered evidence shaping:** multiple callers build similar status/degraded/error maps.
   - **Config fan-out:** many packages parse the same env/config defaults or validation rules.
   - **Transport-policy mixing:** channel/provider transport code decides product policy.
   - **Test harness sprawl:** tests duplicate large setup to reach one public behavior.
   - **Persistence ordering leaks:** callers must know lock/order/atomic-write details.
   - **Parity drift risk:** Hermes/Honcho contract knowledge is copied into unrelated packages.
   - **One-off public seam:** exported interface exists for one adapter and mirrors implementation.
9. **Apply architecture tests.** For each candidate:
   - **Deletion test:** if the module disappeared, would complexity vanish or spread across callers?
   - **Two-adapter test:** is this a real seam with multiple adapters, or a hypothetical abstraction with one caller?
   - **Interface-as-test-surface test:** can public behavior be tested at the seam without fragile private-helper tests?
   - **Parity test:** would the change preserve Hermes/Honcho/Gormes public contracts?
10. **Score candidates.** Use the scorecard below; discard candidates below 3 unless the user specifically asks to see speculative ideas.
11. **List anti-candidates.** Name tempting refactors you rejected and why, so future agents do not re-suggest them.
12. **Separate facts from guesses.** Every claim gets a file path, function, test, progress row, or command.
13. **Run the architecture stop/ask gates.** If any gate below trips, stop the review at a recommendation instead of implementation.
14. **Recommend the smallest next skill.** Usually `gormes-tdd-slice`, `gormes-interface-designer`, `gormes-service-layer-refactor`, or `gormes-progress-slicer`.

## Stop / Ask Gates

Stop and ask Juan, or route to planner, when:

- The candidate changes a public CLI/tool/gateway/channel contract and no progress row says to change it.
- The candidate requires moving code across more than one subsystem in the first slice.
- The candidate would remove a feature, fixture, channel, provider, or integration rather than deepen it.
- The candidate depends on external credentials, live providers, browser runtimes, or user data to prove safety.
- The candidate mostly improves aesthetics of package layout without reducing caller burden.

Continue without asking when:

- The first slice preserves public behavior and only moves duplicated mechanics behind a smaller seam.
- A focused characterization test can prove old and new behavior through the same public surface.
- The write scope is one package or one caller family and the rollback is obvious.

## Candidate Scorecard

Score each candidate 0-10:

| Points | Signal |
|---|---|
| +2 | Repeated mechanics exist in two or more callers |
| +2 | Public behavior can be characterized through one seam |
| +1 | Deletion test says complexity would spread across callers |
| +1 | Refactor reduces caller knowledge/config/error handling |
| +1 | Existing tests or fixtures can protect behavior |
| +1 | Change preserves Hermes/Honcho/Gormes user-visible contracts |
| +1 | Candidate removes or centralizes one Gormes-specific smell from the taxonomy |
| +1 | First slice can update one package or one caller family without broad churn |
| -2 | Needs public behavior change without a progress row |
| -2 | Only one adapter/caller and no near-term second adapter |
| -2 | Mainly renames/moves code without improving locality |
| -2 | Requires broad edits across unrelated subsystems |

Recommendation strength:

- `Strong`: 7-10 and first slice is clear.
- `Worth exploring`: 3-6 or needs interface design first.
- `Speculative`: 0-2; usually report as anti-candidate, not a task.

## Score Calibration Examples

Use these examples to keep reviews consistent:

| Example | Likely score | Why |
|---|---:|---|
| Three channel runtimes independently redact the same token/error/status shape | 8-10 | repeated mechanics, clear caller burden, public behavior can be characterized with transcript fixtures |
| `cmd/gormes setup` and `internal/tui/admin` both clamp terminal layout with copied width/height math | 6-8 | repeated mechanics and focused TUI tests, but user-visible layout must be preserved exactly |
| A package exports an interface with one implementation and tests use it only to mock private helpers | 1-3 | likely one-off seam; prefer package-local helper or deletion unless a second adapter is imminent |
| Moving files into a new package because the old package is large | 0-2 | layout aesthetics only unless caller knowledge, tests, or locality improve |
| Replacing provider-specific behavior with one generic provider abstraction | 2-5 | may reduce duplication, but high parity drift risk; route through provider parity/interface design first |

## Evidence Quality Levels

Prefer candidates with `A` or `B` evidence. Mark `C` as speculative unless the user asks for exploration.

- **A — executable evidence:** focused test, fixture, command output, or failing/repeated code path proves the friction.
- **B — source evidence:** two or more file paths/functions show duplicated mechanics or leaked interface burden.
- **C — impression evidence:** “this feels messy” without caller/test/contract proof.

## Output Shape

```text
Area: <subsystem>
Entry points: <files/functions>
Data flow: <source -> transform -> sink>
User-visible contracts: <CLI/TUI/API/files>
Tests/evidence: <commands/files>
Risks: <seam/persistence/security/parity>

Architecture candidates:
1. <candidate title> — <Strong/Worth exploring/Speculative> (<score>/10)
   files: <paths>
   current shape: <modules/callers>
   interface burden: <facts callers/tests must know today>
   evidence_quality: <A executable/B source/C impression>
   friction: <why locality/leverage/testability is weak>
   deepening move: <smaller interface or better seam>
   deletion test: <complexity vanishes/spreads>
   two-adapter test: <real seam/hypothetical seam>
   characterization test: <focused test to write before refactor>
   first safe slice: <TDD/refactor/planner action>
   validation: <focused command + full gate if implemented>

Anti-candidates:
- <tempting refactor rejected> — <why>

Top recommendation: <one candidate + why>
Next skill packet:
  selected_skill: <skill>
  intent: <implementation/design/planning>
  scope: <files/packages>
  behavior_to_preserve: <contract>
  first_test: <command/test name>
  stop_if: <risk or blocker>
  validation_matrix_row: <row from table above>
```

For broad architecture review requests, produce 2-4 candidates, not a grand rewrite plan. Keep the report text-first for Juan. Do not write HTML reports inside this repo; if a visual report is useful, put it under `/tmp` and report the absolute path.

## Validation Matrix

Match the proposed next slice to the smallest proof:

| Next slice type | Minimum proof before implementation | Green gate after implementation |
|---|---|---|
| Package-local helper extraction | Characterization test through existing public caller | focused package test + `git diff --check` |
| New seam or adapter | Interface test with fake and production adapter evidence | focused package test + affected caller test |
| Progress-row refactor plan | `go run ./cmd/progress validate` before and after row edit | `go test ./internal/planning/progress ./webpages/docs -count=1` |
| CLI/tool behavior-preserving refactor | CLI/tool golden or fixture test before moving code | focused command/tool test + full Go gate when feasible |
| Channel/gateway refactor | transcript/status fixture proving no duplicate/leaked output | channel/gateway focused test + progress validate |
| Persistence/config refactor | temp-home/temp-db fixture proving ordering, redaction, and atomicity | package test plus migration/config focused tests |

If no minimum proof exists, the first safe slice is to add the characterization test, not to refactor.

## Review Packet Template

Use this compact packet when handing off implementation:

```text
architecture_review_packet:
  area: <subsystem>
  candidate: <title>
  score: <n>/10
  smell: <taxonomy item>
  evidence_quality: <A/B/C>
  preserve_contracts:
    - <CLI/tool/API/status behavior>
  characterization_test:
    name: <test to add or existing test to extend>
    command: <focused go test command>
  allowed_write_scope:
    - <package/file family>
  forbidden_scope:
    - <public behavior or package not to touch>
  next_skill: <gormes-service-layer-refactor|gormes-interface-designer|gormes-tdd-slice|gormes-progress-slicer>
```

If the packet cannot name a characterization test and allowed write scope, the candidate is not ready for implementation. If evidence quality is `C`, the next skill must be `gormes-architecture-zoomout` or `gormes-interface-designer`, not implementation.

## Rules

- Do not edit production code during the zoom-out pass unless the user explicitly asks.
- Do not create a new backlog; route row work through `gormes-progress-slicer` or `gormes-planner`.
- Do not mark an architecture review as implementation-complete; it only selects the next safe slice.
- Do not propose speculative seams just because code could be abstracted; require repeated mechanics, hard-to-test behavior, or cross-package coupling evidence.
- Keep domain policy at the edge; move only shared mechanics behind deeper modules.
- If the answer changes public behavior, require source-backed Hermes/Gormes evidence before implementation.
- For implementation, require a characterization test before refactoring behavior that is already working.
- Prefer deleting shallow abstractions over adding new ones when the deletion test says complexity does not spread.
