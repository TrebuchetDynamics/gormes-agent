---
name: gormes-progress-slicer
description: Use when turning a Gormes plan, PRD, parity gap, review finding, or user objective into parity evidence doc atoms.
---

# Gormes Progress Slicer

Use this to convert broad work into thin, verifiable Gormes backlog rows without creating side queues.

Inspired by `mattpocock/skills` `to-issues` and `triage`; adapted so the parity evidence doc captures what remains to be built.

## Inputs

- User plan, PRD, issue, review finding, parity audit, or rough objective.
- Relevant source evidence from Hermes/Honcho/Gormes docs or code.

## Slicing Rules

Each slice must be:

- **Vertical:** one user-visible or operator-visible behavior through all needed layers.
- **Verifiable:** has concrete test, command, fixture, or manual smoke evidence.
- **Small:** suitable for one `gormes-builder`/`gormes-tdd-slice` pass.
- **Ordered:** dependencies are explicit.
- **Backlog-safe:** represented in `docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md` only; never as private TODO lists.

## Workflow

1. **Name the parent objective.** Include source refs and why the work matters.
2. **Draft slices.** For each: title, behavior, validation, blockers, HITL/AFK.
3. **Check overlap.** Search the existing parity evidence doc by `grep -i <topic> docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md`; update/merge instead of duplicating.
4. **Ask if ambiguity changes scope.** Especially for public contracts, persistence, security, or release promises.
5. **Write rows through canonical tooling.** Use `gormes-planner` if schema or priorities need edits.

## Output Shape

```text
Objective: <one line>
Source refs: <files/URLs>
Slices:
1. <title> — AFK/HITL
   behavior: <vertical outcome>
   validation: <command/evidence>
   blocked_by: <none/row id>
```

## Validation

- `go run ./cmd/progress validate`
- `git diff --check`
- Final report names changed progress/docs files and any rows intentionally merged instead of created.
