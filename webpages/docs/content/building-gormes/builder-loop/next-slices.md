---
title: "Next Slices"
weight: 30
aliases:
  - /building-gormes/next-slices/
---

# Next Slices

This page is generated from the canonical progress file and lists the highest
leverage contract-bearing roadmap rows to execute next.

The ordering is:

1. unblocked `P0` handoffs;
2. active `in_progress` rows;
3. `fixture_ready` rows;
4. unblocked rows that unblock other slices;
5. remaining `draft` contract rows.

Use this page when choosing implementation work. If a row is too broad, split
the row in `progress.json` before assigning it.

If no slices are listed, the next correct action is planner work: choose one
planned row from `progress.json` or a phase page and add enough contract detail
for it to appear here. Do not infer that an empty generated list means the
roadmap is complete.

<!-- PROGRESS:START kind=next-slices -->
| Phase | Slice | Contract | Trust class | Fixture | Why now |
|---|---|---|---|---|---|
| 5 / 5.U | Sandbox isolation depth selection | Operator can select sandbox isolation depth: process-level (fast, weaker isolation), container-level (Docker/gVisor, balanced), or VM-level (Firecracker, strongest isolation). Default is process-level with transactional rollback. | operator | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 6 / 6.K | Behavioral pattern extraction from session logs | Mine session logs and tool execution audits for behavioral patterns: which tool sequences succeed vs fail, which reasoning patterns precede good outcomes, which response styles correlate with user satisfaction. Patterns feed into the self-evolution loop as candidate mutations. | operator | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 6 / 6.L | Skill code execution runtime | Skills are not just markdown instructions — they contain executable code that can be run in a sandboxed environment. This mirrors Voyager's code-as-action pattern: skills are validated, sandboxed, and can be composed by the agent at runtime. | operator, system | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 6 / 6.L | Skill dependency resolution and composition | Skills can declare dependencies on other skills. The runtime resolves the dependency graph before execution. The agent can compose skills by chaining: output of Skill A feeds into input of Skill B. Dependencies are validated at load time. | operator | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
