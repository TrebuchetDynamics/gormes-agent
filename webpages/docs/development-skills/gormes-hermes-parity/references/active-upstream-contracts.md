# Active Upstream Contracts

Hermes has historical and current implementations. Pick the active
user-visible contract before comparing Gormes.

| Surface | Prefer these upstream refs first | Legacy refs are useful for |
|---|---|---|
| Full-screen TUI / visual UX | `$HERMES_SRC/ui-tui/src/components/appLayout.tsx`, `appChrome.tsx`, `messageLine.tsx`, `thinking.tsx`, and related `ui-tui/src/__tests__` | Older `cli.py` prompt-toolkit details only when current Ink does not cover the behavior |
| Classic CLI prompts/status | `$HERMES_SRC/cli.py` and `$HERMES_SRC/tests/cli` | Current Ink only for shared semantics |
| Telegram/channel-visible behavior | channel adapters, gateway event handlers, tool progress renderers, Telegram tests | TUI-only renderers only as shape hints |
| Install/runtime behavior | `install.sh`, packaging docs, command startup paths, `gormes-dev-runtime` evidence | Historical installer notes only as migration context |
| Prompt identity, memory, defaults | `$HERMES_SRC/hermes_cli/default_soul.py`, `agent/prompt_builder.py`, `tools/memory_tool.py`, `agent/memory_manager.py`, gateway reset tests | Old prompt snippets only as migration context |
| Skills and template expansion | `$HERMES_SRC/agent/skill_commands.py`, `skill_preprocessing.py`, `skill_utils.py`, `tools/skills_tool.py`, `tools/skills_sync.py` | Website skill docs only as catalog evidence |
| Streaming/tool-call/channel UX | `$HERMES_SRC/tests/gateway/test_stream_consumer.py`, `test_update_streaming.py`, channel adapter tests, tool progress renderers | Model-loop code only when the UX is emitted there |
| Tool loop and iteration budget | `$HERMES_SRC/run_agent.py:_handle_max_iterations`, `run_conversation max_iterations`, tool-loop tests | Provider adapters only when provider malformed tool calls are in scope |

If sources disagree, classify the old behavior as `stale-upstream` unless the
user explicitly wants legacy parity. Refresh progress row refs and acceptance
before implementation handoff.
