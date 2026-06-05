---
name: gormes-interface-designer
description: Design Gormes package/API boundaries. Use for provider, gateway, tool, session, memory, Goncho, plugin, API, or TUI seams, unclear write_scope, or design comparison.
---

# Gormes Interface Designer

## Repository Branch Rule

For Gormes work, stay on the existing `development` branch. Do not create or
use feature branches, short-lived branches, or git worktrees. If the checkout
is not on `development`, stop before editing and switch safely or report the
blocker.

## Mission

Choose Gormes interfaces that make Hermes-in-Go easier to finish: small caller-facing surfaces, strong compatibility, hermetic tests, and implementation complexity hidden behind useful modules.

## Workflow

### 1. Gather Requirements From Code

Identify:

- callers;
- upstream Hermes/Honcho behavior;
- existing Gormes package patterns;
- trust class and degraded mode;
- tests/fixtures that should survive refactors.

If the answer is in code, inspect code instead of asking.

### 2. Design It Twice

Produce at least two materially different interface shapes:

- minimal/deep interface;
- compatibility-first interface;
- common-case optimized interface;
- adapter-oriented interface when two real adapters exist.

Do not implement yet.

### 3. Compare

Compare in prose:

- caller simplicity;
- compatibility with Hermes/Honcho contracts;
- depth and locality;
- testability through public behavior;
- migration cost from current Gormes code;
- risk of shallow pass-through abstractions.

### 4. Pick Or Produce A Row

If one design is clearly best, recommend it and name the progress row it enables. If the design needs implementation, create/refine a builder-ready row rather than editing runtime code.

## Rules

- Avoid interfaces with only one hypothetical adapter unless they hide real complexity.
- Keep external Honcho-compatible names where public contracts require them; keep internal Gormes package naming as Goncho.
- Prefer fixtures over live provider/platform dependencies.
- Preserve existing repo patterns unless they are the problem being fixed.

## Final Report

Report the considered interfaces, recommended design, rejected alternatives, tests that would prove it, and the row/write_scope to implement it.
