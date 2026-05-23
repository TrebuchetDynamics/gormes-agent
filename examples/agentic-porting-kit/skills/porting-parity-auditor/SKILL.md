---
name: porting-parity-auditor
description: Use when comparing a source implementation and target implementation to find, classify, or prioritize porting gaps.
---

# Porting Parity Auditor

Compare source implementation behavior to target implementation behavior without changing runtime code.

Vocabulary: source implementation; target implementation; `PORTING_PROGRESS_PATH`.

## Audit

1. Inventory user-visible commands, APIs, data formats, config, errors, and tests in the source implementation.
2. Locate matching target implementation surfaces.
3. Classify each finding as covered, partial, planned, blocked, excluded, or owned divergence.
4. For actionable gaps, hand a bounded row packet to `porting-planner`.

## Evidence Standard

Use exact file paths, symbols, fixtures, commands, and observed outputs. Do not treat a broad directory or vague feature name as proof.

## Output

Report mapped coverage, highest-risk gaps, proposed rows, blockers, and validation commands that would prove the next slice.
