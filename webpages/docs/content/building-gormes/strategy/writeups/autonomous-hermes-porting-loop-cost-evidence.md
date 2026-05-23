---
title: "Autonomous Hermes-porting loop cost evidence"
weight: 30
---

# Autonomous Hermes-porting loop cost evidence

Status: local evidence packet only. This file does not publish Engineering
Writeup #1, choose Juan's publication platform/date, run social automation, or
turn missing cost data into a zero-dollar claim.

## Source artifacts

- Parent writeup row: `Engineering writeup #1: autonomous Hermes-porting loop` in Phase `8.C`.
- Cost mechanism row: `Loop $/iteration cost metric in status file` in Phase `8.F`.
- Local review packet: `webpages/docs/content/building-gormes/strategy/writeups/autonomous-hermes-porting-loop-review-packet.md`.
- Draft post: `webpages/blog/src/content/posts/autonomous-hermes-porting-loop.md`.
- Strategy source: `webpages/docs/content/building-gormes/strategy/success-plan.md#publication-stack`.
- Cost script: `scripts/gormes-builder-loop.sh`.
- Cost parser package: `internal/loopcost/cost.go` and `internal/loopcost/cost_test.go`.
- Development-goal log: `.pi/development-goal/logs.jsonl`.

## Cost-report command evidence

Current default builder state command:

```bash
bash scripts/gormes-builder-loop.sh cost-report
```

Output captured on 2026-05-23 during development-goal iteration 55:

```text
cost_status=unknown_no_cost_data
cost_7d_usd=unknown
cost_7d_runs=0
cost_30d_usd=unknown
cost_30d_runs=0
cost_report=unknown_no_cost_data
```

Isolated empty-state command:

```bash
GORMES_CODEXU_STATE_DIR=$(mktemp -d) bash scripts/gormes-builder-loop.sh cost-report
```

Output captured on 2026-05-23 during development-goal iteration 55:

```text
cost_status=unknown_no_cost_data
cost_7d_usd=unknown
cost_7d_runs=0
cost_30d_usd=unknown
cost_30d_runs=0
cost_report=unknown_no_cost_data
```

Interpretation at iteration 55: local evidence was `unknown_no_cost_data`, not
`$0.00`. The public writeup must not claim cost-per-feature numbers from this
output.

### Iteration 56 adapter update

Iteration 56 found that the local OpenCode JSONL schema records spend at
`part.cost` with nested `part.tokens`, while the cost-report parser only read
`usage.cost`. After adding the local `OpenCode part-cost telemetry adapter for
builder loop` row and fixture-backed parser support, the same default builder
state command reports priced 30-day telemetry:

```text
cost_status=ok
cost_7d_usd=unknown
cost_7d_runs=0
cost_30d_usd=29.3266
cost_30d_runs=6703
7-day spend: unknown | 30-day spend: $29.33 | runs: 0 (7d) / 6703 (30d)
```

The isolated empty-state command still reports explicit unknown-cost evidence:

```text
cost_status=unknown_no_cost_data
cost_7d_usd=unknown
cost_7d_runs=0
cost_30d_usd=unknown
cost_30d_runs=0
cost_report=unknown_no_cost_data
```

Interpretation after iteration 56: the cost-source adapter no longer blocks
local 30-day spend evidence, but the public writeup still cannot claim a
complete one-week measured window because the 7-day report has 0 priced runs.

## Development-goal log coverage

The active development-goal log exists and can be summarized without provider,
social, or release credentials.

- Path: `.pi/development-goal/logs.jsonl`.
- Total parsed JSONL events: 249.
- First event timestamp: `2026-05-23T06:23:10.982000+00:00`.
- Last event timestamp at capture time: `2026-05-23T16:52:58.762000+00:00`.
- Elapsed captured window: 10.5 hours.
- `iteration_prompt_sent` events: 55.
- `iteration_result` events: 54.
- Latest completed iteration in the log: 54.
- Latest completed decision: `continue`.
- `user_steering` events: 1.

Event counts at capture time:

```text
compaction_before_next_iteration=30
compaction_continue_queued_iteration=29
compaction_failed_before_next_iteration=1
compaction_resume_queued=7
compaction_resume_sent=7
compaction_started=37
empty_agent_response_waiting_for_compaction=3
iteration_prompt_sent=55
iteration_queued=24
iteration_result=54
loop_started=1
user_steering=1
```

Interpretation: the log proves an active 10.5-hour development-goal window, not
the parent row's required one week of measured cost telemetry.

## Builder state observations

Default builder state path resolved by `scripts/gormes-builder-loop.sh`:

```text
/home/xel/.local/state/gormes-agent/builder
```

Observed files include older `.opencode.jsonl` logs and a current
`loop-health.env`. Iteration 55 sampled only root-level usage fields and found:

```text
usage/cost probe sample: [null,null,null,null]
```

Iteration 56 then walked nested keys and found current OpenCode cost evidence in
`part.cost` and `part.tokens`:

```text
part.cost entries: 6703
part.tokens entries: 6703
```

Interpretation: the original mechanism was present but the adapter seam was too
shallow. The parser now hides both `usage.cost` and `part.cost` schemas behind
the same cost-report interface.

## One-week publication requirement

Engineering Writeup #1 requires at least one week of measured loop-cost
telemetry before it can publish cost-per-feature claims.

Current evidence status:

- Requirement: 168 hours of measured telemetry.
- Captured development-goal window: 10.5 hours.
- 7-day cost report: `unknown`, 0 priced runs.
- 30-day cost report after the iteration 56 adapter fix: `29.3266`, 6703 priced runs.
- Requirement satisfied: no.

The parent row must remain blocked on measured cost telemetry and Juan's
publication decision.

## Non-fabricated next collection step

To make the parent writeup publishable without fabricating numbers:

1. Run the builder/development loop for at least seven calendar days with cost
   logging enabled.
2. Confirm `scripts/gormes-builder-loop.sh cost-report` returns `cost_status=ok`
   with nonzero 7-day priced run counts before claiming one-week spend. The
   30-day parser path is now proven locally; the remaining blocker is elapsed
   recent telemetry, not the OpenCode cost-source adapter.
3. Recompute cost-per-feature from receipts: development-goal iteration results,
   completed row count, and measured 7-day/30-day spend.
4. Only then update the public draft's cost section and ask Juan for publication
   date/platform review.

## Validation checklist for this packet

The row-local validation commands are:

```bash
GORMES_CODEXU_STATE_DIR=$(mktemp -d) bash scripts/gormes-builder-loop.sh cost-report
bash scripts/gormes-builder-loop.sh cost-report
go test ./internal/loopcost -count=1
go test ./internal -run 'TestCodexuBuilderLoop.*Cost' -count=1
go test ./webpages/docs -count=1
go run ./cmd/progress validate
git diff --check
```

No live network, social credential, GitHub release, or external publication step
is required for this packet.
