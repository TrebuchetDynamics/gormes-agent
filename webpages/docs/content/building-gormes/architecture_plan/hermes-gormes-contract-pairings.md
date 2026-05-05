---
title: "Hermes/Gormes Contract Pairings"
description: "Canonical vocabulary for pairing upstream Hermes contracts with their Go-native Gormes adapters."
date: 2026-04-30
draft: false
---

# Hermes/Gormes Contract Pairings

This page names the pairings between upstream Hermes behavior and the Go
surfaces that implement or adapt that behavior in Gormes. Use it when a
Gormes file name hides the upstream contract it exists to preserve.

The goal is not to rename every `progress.json` row at once. Row names are
referenced by generated docs, `blocked_by`, handoff pages, and tests. Instead,
new and reshaped rows should use the vocabulary below, and older rows should
be renamed only during bounded planner passes with source-wide reference
checks.

## Vocabulary

| Field | Meaning | Example |
|---|---|---|
| `surface` | Operator-visible capability or model-visible tool family. | `gateway.tool_progress`, `tools.skill_view`, `provider.transport` |
| `contract_layer` | The part of the surface being preserved. | `schema`, `preview`, `render`, `transport`, `lifecycle`, `persistence`, `error`, `safety` |
| `upstream_contract` | Exact Hermes/Honcho files or symbols that define the behavior. | `../hermes-agent/gateway/run.py:progress_callback` |
| `gormes_adapter` | Exact Go package, file, type, or function that implements the behavior. | `internal/gateway/render.go:FormatToolProgressPlain` |
| `parity_mode` | `strict`, `functional`, `owned`, or `excluded`. | `functional` |
| `progress_classification` | `covered`, `planned`, `vague`, `missing`, `stale-upstream`, or `blocked`. | `covered` |
| `contract_status` | `missing`, `draft`, `fixture_ready`, or `validated`, matching the `progress.json` schema. | `validated` |

Preferred row naming form:

```text
<surface>.<contract_layer> <short human phrase>
```

Examples:

- `gateway.tool_progress.preview-render Hermes transcript lines`
- `tools.skill_view.schema-result read-only skill loading`
- `provider.transport.stream Anthropic event conversion`
- `cli.auth.lifecycle provider credential pool commands`

Avoid naming rows after implementation files alone. `render.go` is not a
contract; `gateway.tool_progress.preview-render` is the contract.

## Pairing Matrix

| Surface + layer | Upstream contract | Gormes adapter | Parity mode | Progress | Contract status |
|---|---|---|---|---|---|
| `gateway.tool_progress.preview` | `../hermes-agent/agent/display.py:build_tool_preview`; `../hermes-agent/run_agent.py:tool_progress_callback` | `internal/kernel/toolexec.go:toolCallSoulText`; `internal/kernel/toolexec.go:toolCallPreview` | functional | covered | validated |
| `gateway.tool_progress.render` | `../hermes-agent/gateway/run.py:progress_callback`; `../hermes-agent/gateway/run.py:send_progress_messages`; `../hermes-agent/agent/display.py:get_tool_emoji` | `internal/tooltrace`; `internal/gateway/render.go:FormatToolProgressPlain`; `internal/gateway/render.go:FormatToolProgressTelegram`; `internal/gateway/manager.go:dispatchToolProgress`; `internal/tui`; `internal/slack`; `internal/discord` | functional | covered | validated |
| `gateway.stream.final_delivery` | `../hermes-agent/gateway/stream_consumer.py`; `../hermes-agent/gateway/run.py` platform send/edit flow | `internal/gateway/manager.go:dispatchFrame`; `internal/gateway/render.go:FormatStreamPlain`; `internal/gateway/render.go:FormatFinalPlain` | functional | covered | validated |
| `gateway.display_config.tool_progress` | `../hermes-agent/gateway/display_config.py`; `../hermes-agent/gateway/run.py:/verbose`; `../hermes-agent/tests/gateway/test_verbose_command.py` | `internal/config/config.go`; `internal/config/hermes_display_writer.go`; `cmd/gormes/gateway.go`; `internal/gateway/manager.go` | functional | covered | validated for config load, gateway wiring, `/verbose` gate/cycle, and config.yaml per-platform persistence |
| `tools.skill_view.schema_result` | `../hermes-agent/tools/skills_tool.py:SKILL_VIEW_SCHEMA`; `../hermes-agent/tools/skills_tool.py:skill_view` | `internal/tools/skills_tools.go:SkillViewTool`; `internal/skills` store/preprocess helpers | functional | covered | validated |
| `tools.skills_list.schema_result` | `../hermes-agent/tools/skills_tool.py:SKILLS_LIST_SCHEMA`; `../hermes-agent/tools/skills_tool.py:skills_list` | `internal/tools/skills_tools.go:SkillsListTool`; `cmd/gormes/registry.go` | functional | covered | validated |
| `tools.todo.schema_lifecycle` | `../hermes-agent/tools/todo_tool.py:TODO_SCHEMA`; `../hermes-agent/tools/todo_tool.py:TodoStore` | `internal/tools/todo_tool.go:TodoTool`; `internal/kernel/toolexec.go` todo preview | functional | covered | validated |
| `tools.execute_code.schema_runtime` | `../hermes-agent/tools/code_execution_tool.py:build_execute_code_schema`; `../hermes-agent/tools/code_execution_tool.py:execute_code`; generated `hermes_tools.py` RPC helper | `internal/tools/execute_code.go:ExecuteCodeTool`; planned project/strict mode and `hermes_tools` rows | functional | planned | draft |
| `tools.file_read.schema_safety` | `../hermes-agent/tools/file_tools.py:READ_FILE_SCHEMA`; `../hermes-agent/tools/file_tools.py:read_file_tool`; `../hermes-agent/tools/file_operations.py` | `internal/tools/file_task_tools.go:ReadFileTool`; `internal/tools/file_read_guard.go`; `cmd/gormes/registry.go` | functional | covered | validated |
| `tools.search_files.schema_result` | `../hermes-agent/tools/file_tools.py:SEARCH_FILES_SCHEMA`; `../hermes-agent/tools/file_tools.py:search_tool` | `internal/tools/file_task_tools.go:SearchFilesTool`; `cmd/gormes/registry.go` | strict for schema, functional for implementation | covered | validated first pass |
| `tools.write_file.schema_safety` | `../hermes-agent/tools/file_tools.py:WRITE_FILE_SCHEMA`; `../hermes-agent/tools/file_tools.py:write_file_tool`; `../hermes-agent/tools/file_operations.py:ShellFileOperations.write_file` | `internal/tools/file_task_tools.go:WriteFileTool`; `internal/tools/file_read_guard.go`; `cmd/gormes/registry.go` | functional | covered | validated first pass |
| `tools.patch.schema_safety` | `../hermes-agent/tools/file_tools.py:PATCH_SCHEMA`; `../hermes-agent/tools/file_tools.py:patch_tool`; `../hermes-agent/tools/patch_parser.py` | `internal/tools/file_task_tools.go:PatchTool`; future fuzzy/V4A/checkpoint rows | functional | partial | validated first pass |
| `tools.terminal.foreground_runtime` | `../hermes-agent/tools/terminal_tool.py:TERMINAL_SCHEMA`; `../hermes-agent/tools/terminal_tool.py:terminal_tool`; `../hermes-agent/tools/approval.py` | `internal/tools/terminal_tool.go:TerminalTool`; `internal/tools/dangerous_command.go`; `cmd/gormes/registry.go`; future process-registry/PTY/backend rows | functional | partial | validated first pass |
| `tools.registry.descriptor` | `../hermes-agent/tools/registry.py`; `../hermes-agent/model_tools.py`; `../hermes-agent/toolsets.py` | `internal/tools`; `internal/tools/testdata/upstream_tool_parity_manifest.json`; `cmd/gormes/registry.go` | strict for names/schema, functional for Go execution | partial | validated for shipped descriptors |
| `provider.transport.request_stream` | `../hermes-agent/agent/transports/*`; provider adapters under `../hermes-agent/agent/*_adapter.py` | `internal/hermes.ProviderTransport`; provider transcript fixtures | functional | planned | draft |
| `provider.auth.lifecycle` | `../hermes-agent/hermes_cli/auth.py`; `../hermes-agent/hermes_cli/auth_commands.py`; `../hermes-agent/agent/credential_pool.py` | `cmd/gormes/auth*.go`; `internal/config`; `internal/cli` | functional | planned | draft |
| `cli.command_tree.parser` | `../hermes-agent/hermes_cli/main.py`; `../hermes-agent/hermes_cli/commands.py`; `../hermes-agent/cli.py` slash commands | `cmd/gormes/hermes_cli_parity.go`; `cmd/gormes`; `internal/gateway/commands.go` | strict for spelling/aliases, functional for implementation | planned | draft |
| `gateway.platforms.adapter_lifecycle` | `../hermes-agent/gateway/config.py:Platform`; `../hermes-agent/gateway/platforms/*.py` | `internal/channels/*`; `internal/gateway`; `cmd/gormes/gateway.go` | functional | planned | draft |
| `cron.job_tool.lifecycle` | `../hermes-agent/cron/*`; `../hermes-agent/tools/cronjob_tools.py` | `internal/cron`; `internal/tools`; `internal/gateway` | functional | planned | draft |
| `memory.honcho.compatibility` | `../honcho/src/*`; `../hermes-agent/plugins/memory/honcho`; Honcho SDK docs | `internal/goncho`; `internal/gonchotools`; `internal/memory`; `internal/session` | functional with Honcho-compatible external names | planned | draft |
| `prompt.context.assembly` | `../hermes-agent/agent/prompt_builder.py`; `../hermes-agent/run_agent.py`; gateway session context files | `internal/hermes`; `internal/kernel`; `internal/gateway` prompt/context builders | functional | planned | draft |
| `tui.operator_chrome.render` | `../hermes-agent/ui-tui`; legacy `../hermes-agent/cli.py` only when current TUI lacks coverage | `internal/tui`; `internal/tuigateway`; `cmd/gormes` | functional | planned | draft |

## Rename Policy

Renaming rows is useful when the old name hides the upstream contract, but it
must be done deliberately:

1. Add or confirm the pairing here first.
2. Search for the row name across `progress.json`, generated docs, tests,
   skills, and handoff pages.
3. Rename only one bounded surface at a time.
4. Keep `blocked_by`, `unblocks`, `source_refs`, generated docs, and site
   data synchronized.
5. Run `go run ./cmd/progress write`, `go run ./cmd/progress validate`,
   `go test ./internal/progress -count=1`, and `go test ./docs -count=1`.

When the choice is between a risky rename and a clearer pairing note, prefer
the pairing note. The taxonomy is meant to make parity work easier, not to
create a second naming migration project.

## Progress Row Template

Use this shape when creating or reshaping rows:

```json
{
  "name": "gateway.tool_progress.preview-render Hermes transcript lines",
  "contract": "Gormes emits and renders Hermes-compatible tool progress lines from started tool calls, including preview extraction, emoji mapping, duplicate collapse, and final-answer separation.",
  "source_refs": [
    "../hermes-agent/run_agent.py:tool_progress_callback",
    "../hermes-agent/agent/display.py:build_tool_preview",
    "../hermes-agent/gateway/run.py:progress_callback",
    "internal/kernel/toolexec.go",
    "internal/gateway/render.go",
    "internal/gateway/manager.go"
  ],
  "acceptance": [
    "Tool progress is a separate editable channel message, not assistant final text.",
    "Known Hermes tools render with the expected emoji and compact quoted preview.",
    "Adjacent identical progress lines collapse as `(×N)` using the Hermes-visible counter shape.",
    "Unknown tool names degrade to bounded generic evidence."
  ]
}
```

Note: the user-visible transcript uses the multiplication sign in `(×N)` style
rendering. Tests should assert the exact glyph used by the current renderer.
