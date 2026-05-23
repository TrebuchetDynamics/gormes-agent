---
name: porting-references
description: Use when target implementation shape is unclear and external or local reference implementations may provide patterns before coding.
---

# Porting References

Use references as pattern donors, not as replacement contracts.

Vocabulary: source implementation; target implementation; `PORTING_PROGRESS_PATH`.

## Process

1. Start from the source implementation behavior and the selected row.
2. Inspect reference implementations only for reusable patterns, package shape, tests, or edge-case handling.
3. Record provenance when a pattern influences the target implementation.
4. Return to `porting-builder` with a narrow implementation plan.

## Guardrails

- Do not import reference code blindly.
- Do not replace the source implementation as the parity oracle.
- Do not add new dependencies unless the selected row and host repository allow them.
- Keep follow-up work in `PORTING_PROGRESS_PATH`.
