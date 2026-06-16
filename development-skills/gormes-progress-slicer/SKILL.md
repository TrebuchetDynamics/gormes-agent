---
name: gormes-progress-slicer
description: Turn broad Gormes plans, PRDs, parity gaps, review findings, or user objectives into thin, verifiable progress-row slices.
---

# Gormes Progress Slicer

Use this to convert broad work into thin, verifiable Gormes backlog rows without creating side queues.

Inspired by `mattpocock/skills` `to-issues` and `triage`; adapted so the progress control plane captures what remains to be built.

## Inputs

- User plan, PRD, issue, review finding, parity audit, TODO/backlog request, missing-feature report, or rough objective.
- Relevant source evidence from Hermes/Honcho/Gormes docs or code.

## Canonical TODO Intake

Gormes TODOs live in the logical progress backlog, not in ad hoc files.
Use this skill when the user asks where TODOs live, asks to add more work, or
pastes a broad missing-feature list that needs to become buildable slices.

- Active backlog: `webpages/docs/content/building-gormes/architecture_plan/progress.json/` via `cmd/progress` / `internal/planning/progress`.
- Parity evidence: `webpages/docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md`; use it for source-backed `missing`/`partial` classifications, not as the queue.
- Historical receipts: `TODO.md` may exist, but do not append new active work there.
- If `go run ./cmd/progress next-work --repo-only` reports no builder-ready rows, the next action is usually row sharpening: split one vague/blocked gap into a repo-scoped builder-ready row with exact validation.

## Slicing Rules

Each slice must be:

- **Vertical:** one user-visible or operator-visible behavior through all needed layers.
- **Verifiable:** has concrete test, command, fixture, or manual smoke evidence.
- **Small:** suitable for one `gormes-builder`/`gormes-tdd-slice` pass.
- **Ordered:** dependencies are explicit.
- **Backlog-safe:** represented as logical progress rows; never as private TODO lists or parity-doc-only work.

## Workflow

1. **Name the parent objective.** Include source refs and why the work matters.
2. **Draft slices.** For each: title, behavior, validation, blockers, HITL/AFK.
3. **Check overlap.** Search existing progress rows with `go run ./cmd/progress list --module <module>` or targeted `rg` over progress docs; consult parity evidence docs only for source-backed classifications.
4. **Convert TODO wording into row fields.** Every proposed slice must name module, source refs, write scope, ready-when, not-ready-when, acceptance/done signal, and validation. If any field is unknown, mark that as the planner work instead of adding a vague row.
5. **Ask if ambiguity changes scope.** Especially for public contracts, persistence, security, or release promises.
6. **Write rows through canonical tooling.** Use `gormes-planner` and `cmd/progress` / `internal/planning/progress` when schema, priorities, or generated docs need edits.

## Output Shape

```text
Objective: <one line>
Source refs: <files/URLs>
Slices:
1. <title> — AFK/HITL
   behavior: <vertical outcome>
   validation: <command/evidence>
   blocked_by: <none/row id>
   progress_row_target: <module/id or new-row-needed>
   write_scope: <repo paths>
   ready_when: <builder-ready condition>
   not_ready_when: <blockers/ambiguity>
```

## Validation

- `go run ./cmd/progress validate`
- `git diff --check`
- Final report names changed progress/docs files and any rows intentionally merged instead of created.
