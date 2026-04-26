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
autonomous implementation attempt. Each card carries the execution owner,
slice size, contract, trust class, degraded-mode requirement, fixture target,
write scope, test commands, done signal, acceptance checks, and source
references.

Shared unattended-loop facts live in [Builder Loop Handoff](../builder-loop-handoff/):
the main entrypoint, orchestrator plan, candidate source, generated docs,
tests, and candidate policy. Keep those control-plane facts in
`meta.builder_loop`, and keep row-specific execution facts in `progress.json`.

<!-- PROGRESS:START kind=agent-queue -->
## 1. BlueBubbles iMessage bubble formatting parity

- Phase: 7 / 7.E
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Priority: `P3`
- Contract: BlueBubbles outbound iMessage sends are non-editable, markdown-stripped, paragraph-split bubbles without pagination suffixes
- Trust class: gateway, system
- Ready when: The first-pass BlueBubbles adapter already owns Send, markdown stripping, cached GUID resolution, and home-channel fallback in internal/channels/bluebubbles.
- Not ready when: The slice attempts to add live BlueBubbles HTTP/webhook registration, attachment download, reactions, typing indicators, or edit-message support.
- Degraded mode: BlueBubbles remains a usable first-pass adapter, but long replies may still arrive as one stripped text send until paragraph splitting and suffix-free chunking are fixture-locked.
- Fixture: `internal/channels/bluebubbles/bot_test.go`
- Write scope: `internal/channels/bluebubbles/bot.go`, `internal/channels/bluebubbles/bot_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/channels/bluebubbles -count=1`
- Done signal: BlueBubbles adapter tests prove paragraph-to-bubble sends, suffix-free chunking, and no edit/placeholder capability.
- Acceptance: Send splits blank-line-separated paragraphs into separate SendText calls while preserving existing chat GUID resolution and home-channel fallback., Long paragraph chunks omit `(n/m)` pagination suffixes and concatenate back to the stripped original text., Bot does not implement gateway.MessageEditor or gateway.PlaceholderCapable, preserving non-editable iMessage semantics.
- Source refs: ../hermes-agent/gateway/platforms/bluebubbles.py@f731c2c2, ../hermes-agent/tests/gateway/test_bluebubbles.py@f731c2c2, internal/channels/bluebubbles/bot.go, internal/gateway/channel.go
- Unblocks: BlueBubbles iMessage session-context prompt guidance
- Why now: Unblocks BlueBubbles iMessage session-context prompt guidance.

## 2. TUI TerminalNativeSelectionHelp constant + help-string fixture

- Phase: 5 / 5.Q
- Owner: `gateway`
- Size: `small`
- Status: `planned`
- Contract: internal/tui declares an exported string constant TerminalNativeSelectionHelp = 'Selection: use your terminal's native selection (Shift-drag in most terminals; iTerm Cmd-drag, tmux copy-mode). Gormes does not advertise an in-app copy hotkey.' and a pure helper SelectionHelpLine() that returns it; one fixture asserts the constant exists, mentions 'terminal' but not 'Cmd+C'/'Ctrl+C'/'Ctrl-Shift-C'/'OSC 52'/'clipboard hotkey'/'Ink', and another asserts no advertised copy shortcut leaks anywhere else in the package
- Trust class: operator
- Ready when: internal/tui already exposes Bubble Tea model/view/update files and a mouse tracking config; adding a single new file with one constant compiles cleanly alongside them., phase-5-final-purge.md already documents the terminal-native selection divergence, so this row is mechanical: lift that statement into a typed Go constant and a regression test.
- Not ready when: The slice ports Hermes Ink, calls OSC 52, adds clipboard libraries, modifies internal/tui/update.go input handling, or changes remote TUI transport., The slice introduces a Cobra command flag for copy mode or a configuration key., The slice modifies cmd/gormes/ files.
- Degraded mode: If a future row adds a real Go-native copy mode, it must replace this constant rather than extend it; until then, the help-string fixture prevents accidental advertising of unimplemented Ink shortcuts.
- Fixture: `internal/tui/selection_help_test.go`
- Write scope: `internal/tui/selection_help.go`, `internal/tui/selection_help_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tui -run 'TestTerminalNativeSelectionHelpExists\|TestTerminalNativeSelectionHelpNoFakeShortcuts\|TestTUIPackageDoesNotAdvertiseCopyHotkey' -count=1`, `go test ./internal/tui -count=1`, `go vet ./internal/tui`, `go run ./cmd/builder-loop progress validate`
- Done signal: internal/tui/selection_help.go declares TerminalNativeSelectionHelp and SelectionHelpLine; three named tests pass; no other internal/tui or cmd/gormes file is modified.
- Acceptance: TestTerminalNativeSelectionHelpExists: TerminalNativeSelectionHelp is a non-empty string constant exported from internal/tui, contains the substring 'terminal', and SelectionHelpLine() returns the same value., TestTerminalNativeSelectionHelpNoFakeShortcuts: TerminalNativeSelectionHelp does not contain any of: 'Cmd+C', 'Ctrl+C', 'Ctrl-Shift-C', 'Cmd-Shift-C', 'OSC 52', 'clipboard hotkey', 'Ink' (case-insensitive)., TestTUIPackageDoesNotAdvertiseCopyHotkey: walking internal/tui/*.go files, no string literal in the package contains the same forbidden shortcuts above (test reads the package source via os.ReadFile, not a runtime check)., go vet ./internal/tui passes; no other package is imported by the new file beyond stdlib.
- Source refs: ../hermes-agent/ui-tui/packages/hermes-ink/src/ink/selection.ts@edc78e25, ../hermes-agent/ui-tui/packages/hermes-ink/src/ink/selection.ts@31d7f195, internal/tui/view.go, internal/tui/model.go, internal/tui/mouse_tracking.go, docs/content/building-gormes/architecture_plan/phase-5-final-purge.md
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
