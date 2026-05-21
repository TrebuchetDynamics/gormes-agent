# Codex /goal Research Notes

Research date: 2026-05-08.

Local Codex version: `@openai/codex` 0.128.0, `codex-cli 0.128.0`.

Relevant upstream files inspected from `github.com/openai/codex`:

- `codex-rs/features/src/lib.rs`: feature flag `Goals`, described as persisted thread goals and automatic goal continuation.
- `codex-rs/state/migrations/0029_thread_goals.sql`: `thread_goals` schema.
- `codex-rs/state/src/runtime/goals.rs`: SQLite persistence, replace/insert/update/delete, wall-clock and token accounting, budget limiting.
- `codex-rs/core/src/tools/handlers/goal_spec.rs`: model tools `get_goal`, `create_goal`, `update_goal`.
- `codex-rs/core/templates/goals/continuation.md`: continuation prompt and completion-audit requirement.
- `codex-rs/core/templates/goals/budget_limit.md`: budget-limited prompt.
- `codex-rs/tui/src/app/thread_goal_actions.rs`: TUI slash-command actions.
- `codex-rs/tui/src/chatwidget/tests/slash_commands.rs`: `/goal`, `/goal pause`, `/goal resume`, `/goal clear`, queued-command behavior.
- `codex-rs/tui/src/chatwidget/goal_validation.rs`: objective length validation.
- `codex-rs/protocol/src/protocol.rs`: `MAX_THREAD_GOAL_OBJECTIVE_CHARS = 4000`.

Observed local Codex state during development:

- Observed DB on Joseph's machine: `~/.codex/state_5.sqlite`. This filename is not a stable public API; Codex can change it across versions.
- Table: `thread_goals(thread_id, goal_id, objective, status, token_budget, tokens_used, time_used_seconds, created_at_ms, updated_at_ms)`.
- Status values: `active`, `paused`, `budget_limited`, `complete`.
- On 2026-05-08 this machine had 6 Codex goals: 2 active, 1 paused, 3 complete.

Behavior cloned here:

- Persistent local SQLite state.
- Per-session objective.
- `active`, `paused`, `budget_limited`, `complete`.
- Soft token budget parsing and display.
- Codex-style elapsed formatting.
- Codex-style 4000 character objective limit.
- Bare `/goal` status/menu behavior.
- `/goal pause`, `/goal resume`, `/goal clear`, `/goal complete`.
- Completion-audit prompt before marking complete.
- Objective wrapped in `<objective>` for friendlier public output.

Known difference:

- Claude Code custom skills do not expose a reliable live per-turn token usage API to markdown commands. Token budgets are stored and displayed as soft budgets. Time accounting is local and reliable.
