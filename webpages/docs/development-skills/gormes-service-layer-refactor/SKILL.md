---
name: gormes-service-layer-refactor
description: Use when a Gormes change creates or reveals duplicated runtime mechanics, repeated channel/provider/setup logic, or feature code that should be reusable before more work builds on it.
---

# Gormes Service Layer Refactor

## Overview

After a feature works, make the reusable mechanics obvious. Gormes should keep domain policy at the edge and shared runtime behavior behind deep modules: small caller-facing interfaces with substantial hidden implementation and strong locality.

## When to Use

Use when you see:
- similar provider, channel, setup, gateway, tool, or store code in multiple files;
- a new feature that copied an existing flow instead of reusing it;
- tests that need the same setup repeated across packages;
- future agents likely to rewrite the same helper because no service seam exists.

Do not use for speculative architecture. If only one caller exists and no duplication is present, leave it alone.

## Workflow

1. Verify behavior is already green or write a failing characterization test first.
2. List the repeated mechanics and the domain policy that must stay at the caller.
3. Apply the deletion test: if the helper vanished, would complexity spread across callers? If not, do not extract it.
4. Apply the two-adapter test: a seam with one caller/adapter is provisional and should usually stay package-local.
5. Extract the smallest reusable seam: function, method, adapter, or package-local helper.
6. Update callers one at a time; preserve public CLI/runtime contracts.
7. Run focused tests, then the repo gate when appropriate.

## Quick Reference

| Smell | Preferred move |
|---|---|
| Same retry/auth/parsing logic twice | Extract helper with tests |
| Channel-specific policy mixed with transport | Keep policy at caller, move mechanics down |
| Provider behavior drift | Route through `gormes-provider-parity` first |
| Unclear package boundary | Route through `gormes-interface-designer` first |
| No test around old behavior | Add characterization test before refactor |
| Pass-through helper with one caller | Do not extract; keep locality |
| Interface mirrors implementation complexity | Deepen or delete the seam |

## Common Mistakes

- Refactoring before the feature works or before behavior is characterized.
- Moving domain decisions into generic services.
- Creating a large abstraction when a small helper would remove the duplicate.
- Creating a public seam for a one-adapter implementation without a near-term second adapter.
- Cleaning unrelated old code in the same slice.
