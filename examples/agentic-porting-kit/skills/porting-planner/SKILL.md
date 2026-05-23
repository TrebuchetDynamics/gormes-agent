---
name: porting-planner
description: Use when mapping source implementation behavior into target implementation progress rows, refining acceptance, or splitting broad porting work.
---

# Porting Planner

Plan source implementation behavior into executable target implementation rows.

Vocabulary: source implementation; target implementation; `PORTING_PROGRESS_PATH`.

## Workflow

1. Resolve `PORTING_PROGRESS_PATH`; default to `progress.json` in the target repository.
2. Read the source implementation evidence before editing a row.
3. Read the closest target implementation code or docs before naming write scope.
4. Split broad work into vertical rows with source refs, write scope, tests, acceptance, and done signal.
5. Mark rows blocked when credentials, external repositories, live services, or owner decisions are required.

## Row Contract

Every executable row should state:

- behavior to port;
- source refs;
- target files allowed to change;
- test command or explicit no-test reason;
- acceptance checks;
- blocker conditions;
- final evidence expected from the builder.
