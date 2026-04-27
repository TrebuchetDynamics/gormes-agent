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

## 2. Cron schedule parser + repeat state fixtures

- Phase: 5 / 5.N
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: internal/cron adds a pure Hermes-compatible schedule parser and repeat-state read model before any public cronjob tool handler is exposed. ParseCronSchedule(input string, now time.Time) returns a typed ParsedSchedule for one-shot durations (`30m`, `2h`, `1d`), recurring intervals (`every 30m`, `every 2h`), 5-field cron expressions, and ISO timestamps. CronNextRunDecision(parsed, lastRunUnix, repeatCompleted, now) reports whether a one-shot is still recoverable inside the 120s grace window, whether recurring jobs should fast-forward stale next-run times, and whether finite repeat counts are exhausted.
- Trust class: operator, system
- Ready when: internal/cron/job.go currently validates only robfig/cron standard expressions; the new parser can be introduced as a sibling pure helper without changing Scheduler.Start in this slice., Tests can inject a fixed time anchor `now := time.Date(2026,4,26,12,0,0,0,time.UTC)`; no live clock, goroutine, bbolt store, or kernel is required.
- Not ready when: The slice exposes a public cronjob tool, edits cmd/gormes, starts scheduler goroutines, or writes bbolt/SQLite rows., The slice imports Hermes Python, croniter, or any non-Go runtime instead of using Go parsing and deterministic fixtures.
- Degraded mode: Invalid schedules and exhausted repeat counters return typed unavailable evidence; the scheduler keeps skipping only the bad job instead of stopping the whole cron loop.
- Fixture: `internal/cron/schedule_parser_test.go`
- Write scope: `internal/cron/schedule_parser.go`, `internal/cron/schedule_parser_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/cron -run 'TestParseCronSchedule_\|TestCronNextRunDecision_' -count=1`, `go test ./internal/cron -count=1`, `go run ./cmd/builder-loop progress validate`
- Done signal: internal/cron/schedule_parser_test.go proves one-shot, interval, cron-expression, ISO timestamp, one-shot grace, finite repeat, and stale-recurring fast-forward behavior under an injected clock.
- Acceptance: TestParseCronSchedule_OneShotDuration parses `30m`, `2h`, and `1d` as Kind=once with RunAt=now+duration and Display preserving the operator input., TestParseCronSchedule_RecurringInterval parses `every 30m` and `every 2h` as Kind=interval with Minutes=30 and 120., TestParseCronSchedule_CronExpression accepts `0 9 * * *`, rejects too-few fields and out-of-range minutes, and returns a typed error containing `invalid schedule`., TestParseCronSchedule_ISOTimestamp accepts timezone-aware and naive ISO timestamps; naive values are interpreted through the injected location, not time.Local., TestCronNextRunDecision_OneShotGraceAllowsLateTick includes a one-shot that is 119s late and excludes one that is 121s late., TestCronNextRunDecision_FiniteRepeatExhaustion marks repeat=1 completed=1 as exhausted while repeat=3 completed=2 remains runnable., TestCronNextRunDecision_RecurringFastForward returns a next run after now when the stored next-run time is stale by multiple intervals.
- Source refs: ../hermes-agent/cron/jobs.py@755a2804:parse_duration, ../hermes-agent/cron/jobs.py@755a2804:parse_schedule, ../hermes-agent/cron/jobs.py@755a2804:_recoverable_oneshot_run_at, ../hermes-agent/cron/jobs.py@755a2804:_compute_grace_seconds, internal/cron/job.go:ValidateSchedule, internal/cron/scheduler.go:Start
- Unblocks: Cron prompt/script safety + pre-run script contract, Cronjob tool action envelope over native store
- Why now: Unblocks Cron prompt/script safety + pre-run script contract, Cronjob tool action envelope over native store.

<!-- PROGRESS:END -->
