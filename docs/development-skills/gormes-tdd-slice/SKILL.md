---
name: gormes-tdd-slice
description: Implement one Gormes progress.json row using tracer-bullet test-driven development. Use when building or fixing a Gormes feature with tests, when a row has test_commands, when the user asks for TDD/red-green-refactor, or when implementing Goncho/Hermes parity behavior one vertical slice at a time.
---

# Gormes TDD Slice

## Mission

Ship one narrow Gormes behavior with a red-green-refactor loop. Tests must verify public behavior, not implementation details.

## Workflow

### 1. Select One Behavior

Use the selected `progress.json` row or choose one builder-ready row. State:

- public interface under test;
- feature-map target or upstream concept;
- behavior to prove;
- row-local `test_commands`;
- allowed write scope.

If the row is too broad, split/refine it before coding.

### 2. RED

Write one failing test for one observable behavior. Prefer:

- command output or exit behavior for CLI;
- request/response behavior for API/gateway;
- public Go package behavior for provider/tools/memory/session;
- compatibility request/response for Goncho/Honcho;
- hermetic fixtures over live network/provider calls.

The failing test should prove the feature-map behavior through a public
contract. Avoid private helper tests unless the progress row explicitly makes
that helper the exported contract for the slice.

Run the focused test and confirm it fails for the right reason.

### 3. GREEN

Write the smallest implementation that passes this test. Stay inside write scope. Do not add speculative future behavior.

### 4. Repeat Vertically

Add the next behavior only after the prior test is green. Do not write a batch of imagined tests first.

### 5. Refactor While Green

Only refactor after tests pass. Prefer deep modules: small interface, substantial hidden implementation, clear locality.

### 6. Verify

Run row `test_commands`, focused package tests, `go run ./cmd/progress
validate`, and the gates in `references/gates.md`.

## Final Report

Report red-green cycles, feature-map target, behavior shipped, tests run, and
any progress row updates needed.
