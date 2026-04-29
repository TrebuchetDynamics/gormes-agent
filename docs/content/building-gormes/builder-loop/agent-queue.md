---
title: "Agent Queue"
weight: 20
aliases:
  - /building-gormes/agent-queue/
---

# Agent Queue

This page is generated from the canonical progress file:
`docs/content/building-gormes/architecture_plan/progress.json`.

It lists unblocked, non-umbrella contract rows that are ready for a focused
skill-driven implementation attempt. Each card carries the execution owner,
slice size, contract, trust class, degraded-mode requirement, fixture target,
write scope, test commands, done signal, acceptance checks, and source
references.

Shared skill handoff facts live in [Skill Builder Handoff](../builder-loop-handoff/):
the main skill entrypoint, plan, candidate source, generated docs, tests, and
candidate policy. Keep those control-plane facts in `meta.builder_loop`, and
keep row-specific execution facts in `progress.json`.

<!-- PROGRESS:START kind=agent-queue -->
## 1. Context-file discovery + injection scan

- Phase: 4 / 4.C
- Owner: `provider`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Native prompt assembly exposes a pure context-file discovery helper that mirrors Hermes project-context precedence: load SOUL.md from the Hermes/Gormes profile unless skipped, then load exactly one project context source in order .hermes.md/HERMES.md walking up to git root, AGENTS.md/agents.md in cwd, CLAUDE.md/claude.md in cwd, then .cursorrules plus sorted .cursor/rules/*.mdc in cwd. Each loaded source is scanned for Hermes-compatible injection/invisible-character patterns and head/tail truncated to the context-file budget before being rendered into a deterministic prompt block.
- Trust class: system, operator
- Ready when: The builder restates the Hermes parity contract and confirms the helper is pure Go and does not call, import, launch, or shell out to hermes-agent runtime services., Tests can use temp profile/project directories and fixture files only; no live ~/.hermes profile, provider credentials, channel SDK, network call, or model call is required., The row stays a prompt-context helper slice and does not replace the entire normal-turn prompt builder, provider role selection, skill snapshotting, memory recall, or tool schema rendering.
- Not ready when: The implementation reads Juan's live Hermes profile or repository context directly in tests instead of using temp fixtures., The slice changes provider adapters, kernel turn execution, tool registry formatting, Goncho memory recall, session-search behavior, or channel/gateway routing., The implementation loads multiple project context types at once when Hermes precedence says the first project context source wins., Blocked context content is injected into the prompt instead of replaced with a safe blocked marker.
- Degraded mode: If profile or project context files are absent, unreadable, blocked by the injection scan, or larger than the budget, the helper returns deterministic empty/blocked/truncated evidence instead of reading live secrets, widening filesystem scope, or falling back to Python Hermes runtime services.
- Fixture: `internal/hermes/context_files_test.go::TestContextFiles*`
- Write scope: `internal/hermes/context_files.go`, `internal/hermes/context_files_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/hermes -run '^TestContextFiles' -count=1`, `go test ./internal/hermes -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Context-file fixtures prove profile SOUL loading/skip behavior, project-context first-match precedence, .hermes.md frontmatter stripping, injection/invisible-character blocking, deterministic head/tail truncation, and no dependency on Python Hermes runtime services.
- Acceptance: TestContextFilesLoadsSoulUnlessSkipped proves profile SOUL.md is independent from project context and skip_soul prevents duplicate identity injection., TestContextFilesProjectPrecedence proves .hermes.md/HERMES.md wins while walking to git root, and AGENTS.md, CLAUDE.md, and .cursorrules are considered only when higher-priority sources are absent., TestContextFilesStripsFrontmatterScansAndBlocksInjection proves YAML frontmatter removal for .hermes.md/HERMES.md plus invisible-character and prompt-injection markers produce a blocked prompt-safe placeholder., TestContextFilesTruncatesHeadTail proves over-budget content keeps deterministic head/tail text and a middle marker naming the source and original length., The rendered block starts with the Hermes-compatible Project Context heading and remains deterministic for prompt snapshots.
- Source refs: ../hermes-agent/agent/prompt_builder.py:32-73, ../hermes-agent/agent/prompt_builder.py:89-127, ../hermes-agent/agent/prompt_builder.py:951-1118, docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md:104-114, internal/hermes/self_help_guidance.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
