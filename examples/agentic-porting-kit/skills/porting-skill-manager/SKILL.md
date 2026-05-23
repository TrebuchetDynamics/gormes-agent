---
name: porting-skill-manager
description: Use when starting substantial source-to-target porting work or deciding whether planning, auditing, building, TDD, or reference-sourcing applies.
---

# Porting Skill Manager

Use this as the routing surface for a validation-gated port from a source implementation to a target implementation.

Vocabulary: source implementation; target implementation; `PORTING_PROGRESS_PATH`.

## Route

1. If the request asks what is missing, compare source and target first with `porting-parity-auditor`.
2. If the request changes backlog rows, use `porting-planner`.
3. If the request ships one row, use `porting-builder` and `porting-tdd-slice`.
4. If implementation shape is unclear, inspect references with `porting-references` before coding.

## Rules

- Treat `PORTING_PROGRESS_PATH` as the only backlog; default it to `progress.json` in the target repository.
- Do not create side queues, private TODO files, or prompt-only task lists.
- Keep host-repository branch, commit, and release rules in that repository's instructions.
- Prefer one vertical slice with validation evidence over broad horizontal rewrites.
