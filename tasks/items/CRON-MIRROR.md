+++
schema = 1
id = "CRON-MIRROR"
title = "Deterministic CRON.md mirror crash test cleanup"
short = "Fix flaky CRON mirror test"
status = "open"
goal = "PHASE-2"
section = "Now"
rank = 10
depends_on = []
touches = ["internal/automation/cron/mirror_test.go", "internal/automation/cron/mirror.go"]
resources = []
kind = "implementation"
ownership = "worker"
scope = "Make `TestMirror_AtomicWrite_NoPartialReadOnCrash` deterministic so the required `ci` check stops flaking. The test starts `Mirror.Run` in a goroutine and relies on `defer cancel()` plus a fixed sleep; the loop can still write into `t.TempDir()` when the test ends, and CI on development at bd8ca3e20 failed with `TempDir RemoveAll cleanup: unlinkat ...: directory not empty`. Start Run with a done channel, cancel, wait for Run to return before asserting leftovers and before the test returns, and remove the sleep race. If `Mirror.Run` lacks a bounded stop, add the minimal stop/wait seam it needs."
acceptance = ["TestMirror_AtomicWrite_NoPartialReadOnCrash passes in repeated runs with no TempDir cleanup error.", "The test waits for Mirror.Run to exit before it returns.", "The no-partial-read assertion and the leftover .tmp check stay intact."]
checks = ["go test ./internal/automation/cron -run 'TestMirror' -count=50", "go test -race ./internal/automation/cron -count=1"]
exclusions = ["The fix would change CRON.md output or mirror scheduling behavior.", "The test could only pass by removing the crash-write assertion."]
+++
- Sources: webpages/docs/content/building-gormes/architecture_plan/completion-plan.md#current-finish-ledger
