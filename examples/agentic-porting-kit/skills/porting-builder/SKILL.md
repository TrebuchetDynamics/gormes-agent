---
name: porting-builder
description: Use when implementing one builder-ready progress row that ports source implementation behavior into a target implementation.
---

# Porting Builder

Build one row from `PORTING_PROGRESS_PATH` with tests and evidence.

Vocabulary: source implementation; target implementation; `PORTING_PROGRESS_PATH`.

## Before Editing

1. Select one unblocked row with write scope and validation commands.
2. Summarize the source implementation behavior and the target implementation surface.
3. Confirm the row is not an umbrella, placeholder, or external-access task.
4. If the row is vague, return to `porting-planner` instead of guessing.

## Build Loop

1. Use `porting-tdd-slice` to create or confirm a failing behavior test.
2. Implement the smallest target change that passes.
3. Update row evidence only for the selected slice.
4. Run row-local validation plus any repository-required checks.

## Report

Name the row, changed files, validation commands, blocked work, and whether the row remains partial or is complete.
