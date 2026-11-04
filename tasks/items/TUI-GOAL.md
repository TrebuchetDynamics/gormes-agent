+++
schema = 1
id = "TUI-GOAL"
title = "TUI goal auto-continuation and judge"
short = "TUI goal auto-continue"
status = "open"
goal = "PHASE-5"
section = "Needs decision"
rank = 10
depends_on = []
touches = ["internal/tui/goal/", "internal/tui/", "internal/adapters/tuiadapter/", "internal/platform/cli/gormescli/tuiapp/", "webpages/docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md", "webpages/docs/content/building-gormes/architecture_plan/progress.json"]
resources = []
kind = "implementation"
ownership = "owner"
scope = "Finish the in-flight native TUI goal slice. On each idle frame after a goal turn, increment the turn count once, evaluate an injected goal judge, pause at the budget, and submit the continuation prompt exactly once. Paused, done, cleared, cancelled, queued, and failed turns must not continue. The judge fails open on missing or malformed output."
acceptance = ["TestGoalAfterIdleSubmitsOnce proves one active-goal idle frame submits exactly one continuation prompt and increments TurnsUsed once.", "TestGoalAfterIdleBudgetPauses proves reaching MaxTurns pauses the goal and submits nothing.", "TestGoalAfterIdleJudgeDone proves a done verdict marks the goal done and submits nothing.", "TestGoalAfterIdleSkipsPausedQueuedCancelled proves paused, queued, cancelled, and failed turns do not continue."]
checks = ["go test ./internal/tui/... ./internal/tui/goal -count=1", "go test ./internal/adapters/tuiadapter -count=1", "go run ./cmd/progress validate", "git diff --check"]
exclusions = ["Continuation fires more than once for one idle frame.", "Continuation ignores pause, done, clear, cancel, or queue state.", "The judge is called with live credentials in a fixture."]
+++
- Sources: webpages/docs/content/building-gormes/architecture_plan/completion-plan.md#current-finish-ledger
