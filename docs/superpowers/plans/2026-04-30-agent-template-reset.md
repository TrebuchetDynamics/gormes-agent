# Agent Template Reset Implementation Plan

**Goal:** Ship `gormes agent reset` so Gormes can seed or reset default agent
persona/context templates for development and runtime homes.

**Reference spec:** `docs/superpowers/specs/2026-04-30-agent-template-reset-design.md`

## Tasks

- [x] Add a progress row under Phase 5.O for `Gormes agent template reset command`.
- [x] Add `internal/agenttemplate` tests for the default file inventory and
  create/skip/force/dry-run behavior.
- [x] Implement `internal/agenttemplate` with fixed relative paths and
  deterministic action reporting.
- [x] Add `cmd/gormes` tests for `agent reset --target`, dry-run, root command
  registration, and CLI parity manifest classification.
- [x] Wire `gormes agent reset` into the root command and parity manifest.
- [x] Update non-historical docs that still identify the active Gormes repo as
  `workspace-mineru/gormes-agent`.
- [x] Run focused Go tests, `go run ./cmd/progress validate`, and
  `git diff --check`.
