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
6. **Separate facts from guesses.** Every claim gets a file path, function, test, progress row, or command.
7. **Recommend the smallest next skill.** Usually `gormes-tdd-slice`, `gormes-interface-designer`, `gormes-service-layer-refactor`, or `gormes-progress-slicer`.

## Output Shape

```text
Area: <subsystem>
Entry points: <files/functions>
Data flow: <source -> transform -> sink>
User-visible contracts: <CLI/TUI/API/files>
Tests/evidence: <commands/files>
Risks: <seam/persistence/security/parity>

Architecture candidates:
1. <candidate title> — <Strong/Worth exploring/Speculative>
   files: <paths>
   current shape: <modules/callers>
   friction: <why locality/leverage/testability is weak>
   deepening move: <smaller interface or better seam>
   deletion test: <complexity vanishes/spreads>
   two-adapter test: <real seam/hypothetical seam>
   first safe slice: <TDD/refactor/planner action>

Top recommendation: <one candidate + why>
Next skill: <one skill + why>
```

For broad architecture review requests, produce 2-4 candidates, not a grand rewrite plan. Do not write HTML reports inside this repo; if a visual report is useful, put it under `/tmp` and report the absolute path.

## Rules

- Do not edit production code during the zoom-out pass unless the user explicitly asks.
- Do not create a new backlog; route row work through `gormes-progress-slicer` or `gormes-planner`.
- Do not propose speculative seams just because code could be abstracted; require repeated mechanics, hard-to-test behavior, or cross-package coupling evidence.
- Keep domain policy at the edge; move only shared mechanics behind deeper modules.
- If the answer changes public behavior, require source-backed Hermes/Gormes evidence before implementation.
- For implementation, require a characterization test before refactoring behavior that is already working.
