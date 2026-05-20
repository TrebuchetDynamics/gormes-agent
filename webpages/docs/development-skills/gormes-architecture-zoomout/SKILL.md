---
name: gormes-architecture-zoomout
description: Use when an agent is unfamiliar with a Gormes code area, a change crosses package boundaries, or a refactor needs a source-backed module map before edits.
---

# Gormes Architecture Zoom-Out

Use this before changing an unfamiliar subsystem or when a planned change spans packages and callers.

Inspired by `mattpocock/skills` `zoom-out` and architecture-improvement guidance; adapted for Gormes source evidence and parity boundaries.

## Trigger Examples

- “I don’t know where this behavior lives.”
- A setup, TUI, provider, gateway, session, tool, or Goncho change crosses package boundaries.
- A refactor proposal lacks caller/data-flow evidence.
- Tests are green but user-visible behavior is still uncertain.

## Workflow

1. **Read local maps first.** Start with `/home/xel/git/sages-openclaw/workspace-mineru/gormes-agent/codemap.md`; read folder `codemap.md` when present.
2. **Map the subsystem.** List modules, owners, entry points, data flow, persistence, and test surfaces.
3. **Find user-visible contracts.** CLI flags, TUI text, gateway protocol, files on disk, progress rows, and Hermes parity refs.
4. **Separate facts from guesses.** Every claim gets a file path, function, test, or command.
5. **Recommend the smallest next skill.** Usually `gormes-tdd-slice`, `gormes-interface-designer`, `gormes-service-layer-refactor`, or `gormes-progress-slicer`.

## Output Shape

```text
Area: <subsystem>
Entry points: <files/functions>
Data flow: <source -> transform -> sink>
User-visible contracts: <CLI/TUI/API/files>
Tests/evidence: <commands/files>
Risks: <boundary/persistence/security/parity>
Next skill: <one skill + why>
```

## Rules

- Do not edit production code during the zoom-out pass unless the user explicitly asks.
- Do not create a new backlog; route row work through `gormes-progress-slicer` or `gormes-planner`.
- If the answer changes public behavior, require source-backed Hermes/Gormes evidence before implementation.
