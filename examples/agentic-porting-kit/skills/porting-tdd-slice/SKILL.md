---
name: porting-tdd-slice
description: Use when writing or fixing target implementation behavior with a red-green-refactor loop against source implementation evidence.
---

# Porting TDD Slice

Ship one observable target implementation behavior at a time.

Vocabulary: source implementation; target implementation; `PORTING_PROGRESS_PATH`.

## Red

Write the smallest test that fails because the target implementation does not yet match the source implementation behavior. Prefer hermetic fixtures over live services.

## Green

Implement only what is needed for that test. Stay inside the row write scope from `PORTING_PROGRESS_PATH`.

## Refactor

Clean up while tests stay green. Do not expand scope or invent extra behavior.

## Verify

Run the focused test, the row validation command, and the repository-required diff/build checks before claiming completion.
