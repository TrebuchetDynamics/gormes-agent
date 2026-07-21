# Specialized Planner Lanes

Load only the section matching the current request. The main planner workflow
and progress-row contract still apply.

## CLI, Config, And Migration Parity

When planning Hermes command or config parity:

- Treat `Hermes CLI command-tree parity manifest` as the gate before claiming
  broad command parity or assigning handler ports.
- Separate native config lifecycle from cross-product imports:
  `gormes config migrate` updates Gormes' own schema, while
  `gormes migrate hermes` and `gormes migrate openclaw` import external state.
- Classify Gormes-owned additions such as `goncho`, `--offline`, `--remote`,
  and XDG/TOML config as `owned` with source-backed rationale. Hermes-owned
  `-z/--oneshot` is removed-command guidance in Gormes; `gormes chat -q` is
  the canonical scripted-chat surface.
- Do not accept broad globs (`hermes_cli/**`, `gateway/**`, `_handle_*`) as
  sole evidence. Pair them with exact commands, symbols, fixtures, or tests.
- Preserve the public command spelling `openclaw`; typo-like requests such as
  `ooenclaw` should be deterministic suggestions to
  `gormes migrate openclaw`, not silent aliases, unless a dedicated
  compatibility row explicitly changes that API policy.

## Persona, Templates, Skills, And Reset Defaults

When planning agent-default or "Gormes bot persona" parity, inspect the
upstream sources that seed and inject identity:
`$HERMES_SRC/hermes_cli/default_soul.py`,
`$HERMES_SRC/hermes_cli/config.py`,
`$HERMES_SRC/hermes_cli/profiles.py`,
`$HERMES_SRC/agent/prompt_builder.py`, and
`$HERMES_SRC/docker/SOUL.md` when container defaults matter. Rows must say
whether Gormes ports Hermes' default `SOUL.md`, intentionally replaces brand
text with Gormes identity, or preserves user-owned local customization.

For skills parity, use exact sources such as
`$HERMES_SRC/agent/skill_commands.py`,
`$HERMES_SRC/agent/skill_preprocessing.py`,
`$HERMES_SRC/agent/skill_utils.py`,
`$HERMES_SRC/tools/skills_tool.py`,
`$HERMES_SRC/tools/skill_manager_tool.py`, and
`$HERMES_SRC/tools/skills_sync.py`. Plan user-visible load order, template
files, linked references, enabled/disabled/platform filtering, reset/sync
commands, and model-visible skill tool calls.

For reset behavior, separate session reset from development-environment reset.
Hermes session reset evidence lives in
`$HERMES_SRC/tests/gateway/test_session_boundary_hooks.py`,
`$HERMES_SRC/tests/gateway/test_session_reset_notify.py`,
`$HERMES_SRC/tests/gateway/test_session_model_reset.py`, and
`$HERMES_SRC/tests/run_agent/test_session_reset_fix.py`. A Gormes development
reset follow-up must inspect the completed `Gormes agent template reset
command` row, `internal/agenttemplate`, and
`cmd/gormes/agent_reset_test.go`. Extend that surface with explicit write
scope, fixture state, and acceptance; do not hide reset behavior in installer
or generic runtime rows or create a duplicate reset row.

Workspace-specific paths such as `workspace-gormes` and `workspace-mineru` are
planning evidence, not product defaults. Rows may require fixture-backed
behavior around those layouts, but must reject hard-coded operator paths.

## External Review Feedback Ingestion

This lane shapes exact Greptile/Grep-style PR review, GitHub comments, CI
annotations, static-analysis findings, or pasted local review logs. It does not
fix code.

1. Preserve reviewer text, PR/check URL when available, file path,
   line/symbol, command output, and commit SHA. If feedback is summarized, ask
   for the exact text.
2. Classify each finding as existing-row refinement, new builder-ready row,
   duplicate, out-of-scope noise, or blocker needing Juan/system input.
3. Prefer refining `ready_when`, `not_ready_when`, `acceptance`, `source_refs`,
   `write_scope`, or `test_commands` on an existing row.
4. Create at most one new row per bounded review theme, with exact review
   evidence in `source_refs`, `fixture`, or provenance note.
5. Reject TODO files, issue lists, private review ledgers, and prompt-only
   queues.
6. If the finding is already builder-ready, route to `gormes-review-loop` plus
   `gormes-tdd-slice` instead of editing planning surfaces.
