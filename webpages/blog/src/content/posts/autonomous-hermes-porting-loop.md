---
title: "Autonomous Hermes-porting loop"
date: "2026-05-13"
summary: "How TrebuchetDynamics turns Gormes progress rows into small, test-proven Hermes-parity slices."
---

# How an autonomous loop ships Hermes-parity slices

Gormes is useful as a Go-native agent runtime, but it is also a public receipt
for a deeper claim: large Python systems can be ported by a validation-gated
agentic engineering loop without giving up test discipline.

The loop starts from one canonical backlog, `progress.json`. A planner sharpens
the row contract, a builder selects one ready row, and the TDD slice writes the
smallest failing test that proves the behavior is missing. The implementation
does not land until the focused test, the package test, `go test ./... -count=1`,
`go run ./cmd/progress validate`, and `git diff --check` pass.

That shape matters more than raw automation speed. The useful artifact is not
"an agent wrote code"; it is that every slice has a public contract, source
evidence, acceptance criteria, generated docs, and CI evidence. Humans can read
the row, inspect the commit, and replay the gate.

Hermes remains the parity oracle for behavior, but Gormes is allowed to become
opinionated where the Go-native runtime has a sharper operator story: one
binary, local SQLite memory, provider turns, gateway routing, and admin TUI
surfaces that do not require a Python environment.

This blog will publish the loop in the open: which rows shipped, which tests
caught regressions, which upstream assumptions changed, and where the porting
method has to improve before it becomes a reusable `agentic-porting-kit`.
