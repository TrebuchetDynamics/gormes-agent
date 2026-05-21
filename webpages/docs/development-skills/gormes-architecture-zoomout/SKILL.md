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

1. **Read local maps first.** Start with `/home/xel/git/sages-openclaw/workspace-mineru/gormes-agent/codemap.md`; read folder `codemap.md` when present.
2. **Map the subsystem.** List modules, owners, entry points, data flow, persistence, and test surfaces.
3. **Find user-visible contracts.** CLI flags, TUI text, gateway protocol, files on disk, progress rows, and Hermes parity refs.
4. **Evaluate module depth.** Use these terms consistently:
   - **Module:** package/function/type with an interface and implementation.
   - **Interface:** everything callers must know: types, invariants, errors, config, ordering, persistence, and tests.
   - **Implementation:** behavior hidden behind the interface.
   - **Seam:** where alternate behavior can be injected without editing callers.
   - **Adapter:** concrete implementation behind a seam.
   - **Depth:** leverage behind a small interface. Deep modules improve locality; shallow modules move complexity to callers.
5. **Apply architecture tests.** For each candidate:
   - **Deletion test:** if the module disappeared, would complexity vanish or spread across callers?
   - **Two-adapter test:** is this a real seam with multiple adapters, or a hypothetical abstraction with one caller?
   - **Interface-as-test-surface test:** can public behavior be tested at the seam without fragile private-helper tests?
   - **Parity test:** would the change preserve Hermes/Honcho/Gormes public contracts?
6. **Score candidates.** Use the scorecard below; discard candidates below 3 unless the user specifically asks to see speculative ideas.
7. **List anti-candidates.** Name tempting refactors you rejected and why, so future agents do not re-suggest them.
8. **Separate facts from guesses.** Every claim gets a file path, function, test, progress row, or command.
9. **Recommend the smallest next skill.** Usually `gormes-tdd-slice`, `gormes-interface-designer`, `gormes-service-layer-refactor`, or `gormes-progress-slicer`.

## Candidate Scorecard

Score each candidate 0-8:

| Points | Signal |
|---|---|
| +2 | Repeated mechanics exist in two or more callers |
| +2 | Public behavior can be characterized through one seam |
| +1 | Deletion test says complexity would spread across callers |
| +1 | Refactor reduces caller knowledge/config/error handling |
| +1 | Existing tests or fixtures can protect behavior |
| +1 | Change preserves Hermes/Honcho/Gormes user-visible contracts |
| -2 | Needs public behavior change without a progress row |
| -2 | Only one adapter/caller and no near-term second adapter |
| -2 | Mainly renames/moves code without improving locality |
| -2 | Requires broad edits across unrelated subsystems |

Recommendation strength:

- `Strong`: 6-8 and first slice is clear.
- `Worth exploring`: 3-5 or needs interface design first.
- `Speculative`: 0-2; usually report as anti-candidate, not a task.

## Output Shape

```text
Area: <subsystem>
Entry points: <files/functions>
Data flow: <source -> transform -> sink>
User-visible contracts: <CLI/TUI/API/files>
Tests/evidence: <commands/files>
Risks: <seam/persistence/security/parity>

Architecture candidates:
1. <candidate title> — <Strong/Worth exploring/Speculative> (<score>/8)
   files: <paths>
   current shape: <modules/callers>
   interface burden: <facts callers/tests must know today>
   friction: <why locality/leverage/testability is weak>
   deepening move: <smaller interface or better seam>
   deletion test: <complexity vanishes/spreads>
   two-adapter test: <real seam/hypothetical seam>
   characterization test: <focused test to write before refactor>
   first safe slice: <TDD/refactor/planner action>

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
```

For broad architecture review requests, produce 2-4 candidates, not a grand rewrite plan. Keep the report text-first for Juan. Do not write HTML reports inside this repo; if a visual report is useful, put it under `/tmp` and report the absolute path.

## Rules

- Do not edit production code during the zoom-out pass unless the user explicitly asks.
- Do not create a new backlog; route row work through `gormes-progress-slicer` or `gormes-planner`.
- Do not mark an architecture review as implementation-complete; it only selects the next safe slice.
- Do not propose speculative seams just because code could be abstracted; require repeated mechanics, hard-to-test behavior, or cross-package coupling evidence.
- Keep domain policy at the edge; move only shared mechanics behind deeper modules.
- If the answer changes public behavior, require source-backed Hermes/Gormes evidence before implementation.
- For implementation, require a characterization test before refactoring behavior that is already working.
