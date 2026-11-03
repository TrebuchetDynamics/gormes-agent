---
title: "Agent Queue"
weight: 20
aliases:
  - /building-gormes/agent-queue/
---

# Agent Queue

This page is generated from the canonical progress file:
`webpages/docs/content/building-gormes/architecture_plan/progress.json`.

It lists unblocked, non-umbrella contract rows that are ready for a focused
skill-driven implementation attempt. Each card carries the execution owner,
slice size, contract, trust class, degraded-mode requirement, fixture target,
write scope, test commands, done signal, acceptance checks, and source
references.

Shared skill handoff facts live in [Skill Builder Handoff](../builder-loop-handoff/):
the main skill entrypoint, plan, candidate source, generated docs, tests, and
candidate policy. Keep those control-plane facts in `meta.builder_loop`, and
keep row-specific execution facts in `progress.json`.

If the generated list is empty, do not switch to an ad hoc TODO list. Route
through `gormes-planner`, repair one planned/draft row until it satisfies the
handoff contract, validate `progress.json`, and then return to builder
selection.

<!-- PROGRESS:START kind=agent-queue -->
## 1. Terminal background process registry

- Phase: 5 / 5.L
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P2`
- Contract: Add a bounded in-process registry for background terminal executions. Start with terminal background=true, return a stable process id, list/poll running status and bounded output, terminate by id, and clean up on session end. Reuse the terminal tool's approval and command-safety path; no PTY, WSL-specific, or remote process ownership behavior in this slice.
- Trust class: operator, system
- Ready when: The terminal tool keeps its current approval and command-safety path for foreground commands., The registry owns process handles, start time, bounded output, and a stable id., Cleanup is tied to the tool/session lifecycle and is safe to call twice.
- Not ready when: Background execution bypasses the existing approval or dangerous-command checks., The registry stores unbounded output or leaks process handles after session end., PTY, WSL process ownership, or remote process behavior is mixed into this slice.
- Degraded mode: When the registry is unavailable, background=true returns unsupported with stable guidance. Unknown ids return not_found without mutation. Output is capped and redacted through the existing tool result path.
- Fixture: `internal/tools/processregistry/registry_test.go; internal/tools/terminal_tool_test.go`
- Write scope: `internal/tools/processregistry/`, `internal/tools/terminal_tool.go`, `internal/tools/terminal_tool_test.go`, `webpages/docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools/processregistry -count=1`, `go test ./internal/tools -run 'Background\|ProcessRegistry' -count=1`, `go run ./cmd/progress validate`
- Done signal: Background terminal starts, lists, polls bounded output, and terminates through the registry with approval and cleanup evidence.
- Acceptance: TestTerminalToolBackgroundStartsAndLists proves background=true starts a process, returns a stable id, and list shows it running., TestProcessRegistryPollsBoundedOutput proves polling returns bounded, redacted output and a terminal status after exit., TestProcessRegistryTerminate proves terminate stops the process and a second terminate/list is safe., TestProcessRegistryUnknownID proves unknown ids return not_found without mutation., TestProcessRegistryCleanup proves session-end cleanup terminates and removes owned handles.
- Source refs: ./hermes-agent/tools/code_execution_tool.py@197f16a12b098f1ea541bbd0a5ddd7d66f2d2dee, ./hermes-agent/tools/process_registry_ownership.py@197f16a12b098f1ea541bbd0a5ddd7d66f2d2dee, ./hermes-agent/tools/process_registry_termination.py@197f16a12b098f1ea541bbd0a5ddd7d66f2d2dee, ./hermes-agent/tools/process_registry_output_log.py@197f16a12b098f1ea541bbd0a5ddd7d66f2d2dee, ./hermes-agent/tools/process_registry_wait_policy.py@197f16a12b098f1ea541bbd0a5ddd7d66f2d2dee, internal/tools/terminal_tool.go, internal/tools/terminal_tool_test.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 2. TUI goal auto-continuation and judge

- Phase: 5 / 5.Q
- Owner: `tui`
- Size: `medium`
- Status: `planned`
- Priority: `P4`
- Contract: Finish the in-flight native TUI goal slice. On each idle frame after a goal turn, increment the turn count once, evaluate an injected goal judge, pause at the budget, and submit the continuation prompt exactly once. Paused, done, cleared, cancelled, queued, and failed turns must not continue. The judge fails open on missing or malformed output.
- Trust class: operator, system
- Ready when: The in-flight TUI goal control and bar slice is present., The TUI idle frame path exposes a single post-turn hook., The judge and continuation prompt are injected interfaces.
- Not ready when: Continuation fires more than once for one idle frame., Continuation ignores pause, done, clear, cancel, or queue state., The judge is called with live credentials in a fixture.
- Degraded mode: A missing judge continues with continue evidence. A missing goal store disables continuation and keeps the bar empty. A session without an ID does not continue.
- Fixture: `internal/tui/goal/goal_test.go; internal/tui/goal_continuation_test.go; internal/tui/slash_goal_test.go`
- Write scope: `internal/tui/goal/`, `internal/tui/`, `internal/adapters/tuiadapter/`, `internal/platform/cli/gormescli/tuiapp/`, `webpages/docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tui/... ./internal/tui/goal -count=1`, `go test ./internal/adapters/tuiadapter -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: The TUI continues an active goal until the judge says done or the budget pauses it, with one submission per idle frame.
- Acceptance: TestGoalAfterIdleSubmitsOnce proves one active-goal idle frame submits exactly one continuation prompt and increments TurnsUsed once., TestGoalAfterIdleBudgetPauses proves reaching MaxTurns pauses the goal and submits nothing., TestGoalAfterIdleJudgeDone proves a done verdict marks the goal done and submits nothing., TestGoalAfterIdleSkipsPausedQueuedCancelled proves paused, queued, cancelled, and failed turns do not continue.
- Source refs: ./hermes-agent/hermes_cli/goals.py@fbf6c6c91d, ./hermes-agent/ui-tui/src/app/goalStatus.ts@fbf6c6c91d, ./hermes-agent/ui-tui/src/components/goalBar.tsx@fbf6c6c91d, internal/tui/goal/goal.go, internal/tui/slash_goal.go, internal/tui/update.go, internal/persistence/session/goal_state.go, internal/gateway/goalloop/goal.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
