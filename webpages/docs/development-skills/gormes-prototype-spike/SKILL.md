---
name: gormes-prototype-spike
description: Use when exploring a Gormes UI, TUI, protocol, state-machine, or runtime design with throwaway code before committing to production implementation.
---

# Gormes Prototype Spike

Use this when a small disposable experiment will answer a design question faster than editing production code.

Inspired by `mattpocock/skills` `prototype`; adapted for Gormes branch, progress, and parity rules.

## Use When

- A TUI, Navivox, provider, gateway, Goncho, or setup flow has multiple plausible designs.
- A state machine or protocol boundary is hard to reason about on paper.
- The user asks to try variants, mock something up, or sanity-check behavior.

Do not use this for normal progress-row implementation; use `gormes-tdd-slice` instead.

## Rules

1. **State the question first.** One sentence: what uncertainty will the prototype resolve?
2. **Keep it disposable.** Put files under a clearly named scratch path such as `tmp/`, `artifacts/`, or a `prototype_*.go`/`*_prototype.dart` file excluded from production paths.
3. **No persistent state by default.** Use in-memory state or temp files only.
4. **One command to run.** Print the exact command and expected observation.
5. **No parallel backlog.** If the result implies real work, update `progress.json` through `cmd/progress` or the planner skill; do not leave private TODOs.
6. **Delete or absorb.** Before completion, either remove prototype files or explicitly mark why they remain as an artifact.

## Validation

- `git status --short` shows prototype files are removed, isolated, or clearly staged as artifacts.
- Production tests are unchanged unless the prototype became a real TDD slice.
- Final report includes: question answered, result, and next production action.
