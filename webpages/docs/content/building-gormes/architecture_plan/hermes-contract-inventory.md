# Hermes Contract Inventory

- Hermes SHA: `43e566f77eaf01293086eb7cb99a21e240d60634`
- Generated: `2026-05-18T15:36:56Z`
- Source pairs: `current` (`43e566f77eaf01293086eb7cb99a21e240d60634`)
- Report mode: `report-only`
- Progress source: `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Backlog policy: `progress.json` remains the only backlog; this report classifies evidence and does not create work rows.
- Claim boundary: Gormes may claim all Hermes features and architecture are paired only when every current-SHA inventory gap is classified as `covered`, `partial`, `planned`, `excluded`, or `owned_divergence`; strict mode additionally requires every critical surface to be `covered`, `excluded`, or `owned_divergence` and every upstream source/doc/test file to be mapped or explicitly excluded.

## Headline Counts

- Source files: `2147`
- Docs files: `983`
- Test files: `1220`
- Unmapped upstream source files: `860`
- Unmapped upstream docs files: `959`
- Unmapped upstream test files: `1198`
- Release checkpoints: `16`
- Critical surfaces: `10`
- Surface strict failures: `8`
- Strict failures: `3025`
- `covered`: `2`
- `partial`: `8`

## Critical Surface Blockers

No critical surface blockers in the current classification. `3017` unmapped upstream source/doc/test files still block strict mode.

## Release Checkpoints

| Checkpoint | Present | Path |
|---|---|---|
| Hermes RELEASE_v0.10.0 | `true` | `hermes-agent/RELEASE_v0.10.0.md` |
| Hermes RELEASE_v0.11.0 | `true` | `hermes-agent/RELEASE_v0.11.0.md` |
| Hermes RELEASE_v0.12.0 | `true` | `hermes-agent/RELEASE_v0.12.0.md` |
| Hermes RELEASE_v0.13.0 | `true` | `hermes-agent/RELEASE_v0.13.0.md` |
| Hermes RELEASE_v0.14.0 | `true` | `hermes-agent/RELEASE_v0.14.0.md` |
| Hermes RELEASE_v0.2.0 | `true` | `hermes-agent/RELEASE_v0.2.0.md` |
| Hermes RELEASE_v0.3.0 | `true` | `hermes-agent/RELEASE_v0.3.0.md` |
| Hermes RELEASE_v0.4.0 | `true` | `hermes-agent/RELEASE_v0.4.0.md` |
| Hermes RELEASE_v0.5.0 | `true` | `hermes-agent/RELEASE_v0.5.0.md` |
| Hermes RELEASE_v0.6.0 | `true` | `hermes-agent/RELEASE_v0.6.0.md` |
| Hermes RELEASE_v0.7.0 | `true` | `hermes-agent/RELEASE_v0.7.0.md` |
| Hermes RELEASE_v0.8.0 | `true` | `hermes-agent/RELEASE_v0.8.0.md` |
| Hermes RELEASE_v0.9.0 | `true` | `hermes-agent/RELEASE_v0.9.0.md` |
| Gormes Hermes v0.14 module pairings | `true` | `webpages/docs/content/building-gormes/architecture_plan/hermes-v0.14-module-pairings.md` |
| Gormes Hermes source-pair manifest | `true` | `webpages/docs/content/building-gormes/architecture_plan/hermes-source-pairs.json` |
| Gormes Hermes source-pair report | `true` | `webpages/docs/content/building-gormes/architecture_plan/hermes-source-pairs.md` |

## Per-Module Gap Summary

| Module | Surfaces | Covered | Partial | Planned | Missing | Blocker severity |
|---|---|---:|---:|---:|---:|---|
| `channels` | `gateway_channels` | `0` | `1` | `0` | `0` | `warning` |
| `continuity` | `profiles`, `sessions`, `goncho_memory`, `learning_loop`, `prompt_assembly` | `1` | `4` | `0` | `0` | `warning` |
| `operator` | `provider_auth_setup`, `tui_cli` | `1` | `1` | `0` | `0` | `warning` |
| `runtime` | `tool_runtime` | `0` | `1` | `0` | `0` | `warning` |
| `tools` | `mcp_acp` | `0` | `1` | `0` | `0` | `warning` |

## Continuity Categories

| Category | Status | Severity | Surfaces | Evidence | Reason |
|---|---|---|---|---|---|
| `sessions` | `partial` | `warning` | `sessions` | `../hermes-agent/agent/context_compressor.py@5401a008`, `../hermes-agent/agent/context_compressor.py@94346523:_find_tail_cut_by_tokens`, `../hermes-agent/agent/context_compressor.py@bda2dbc2:_calculate_protect_tail_boundary`, `../hermes-agent/agent/context_compressor.py@cfc8befe:_find_tail_cut_by_tokens`, `+111 more` | Mapped through surfaces: sessions; worst status is sessions=partial. |
| `memory_goncho_honcho_compatibility` | `partial` | `warning` | `goncho_memory` | `../hermes-agent/acp_adapter/`, `../hermes-agent/agent/`, `../hermes-agent/agent/memory_manager.py`, `../hermes-agent/agent/memory_manager.py:178`, `+256 more` | Mapped through surfaces: goncho_memory; worst status is goncho_memory=partial. |
| `workspace_peer_profile_identity_boundaries` | `covered` | `none` | `profiles` | `../hermes-agent/agent/prompt_builder.py`, `../hermes-agent/agent/prompt_builder.py:32-73`, `../hermes-agent/agent/prompt_builder.py:89-127`, `../hermes-agent/agent/prompt_builder.py:951-1118`, `+156 more` | Mapped surfaces are strictly covered: profiles. |
| `context_retrieval_and_prompt_budget` | `partial` | `warning` | `sessions`, `prompt_assembly` | `../agent-zero/agent.py@7c71185f:agent_init,monologue_start,message_loop_start,before_main_llm_call,response_stream_chunk,reasoning_stream_chunk,message_loop_end,monologue_end`, `../agent-zero/helpers/extension.py@7c71185f:extensible,call_extensions_async,call_extensions_sync`, `../hermes-agent/agent/`, `../hermes-agent/agent/context_compressor.py@5401a008`, `+219 more` | Mapped through surfaces: sessions, prompt_assembly; worst status is sessions=partial. |
| `summaries_conclusions_search` | `partial` | `warning` | `sessions` | `../hermes-agent/agent/context_compressor.py@5401a008`, `../hermes-agent/agent/context_compressor.py@94346523:_find_tail_cut_by_tokens`, `../hermes-agent/agent/context_compressor.py@bda2dbc2:_calculate_protect_tail_boundary`, `../hermes-agent/agent/context_compressor.py@cfc8befe:_find_tail_cut_by_tokens`, `+111 more` | Mapped through surfaces: sessions; worst status is sessions=partial. |
| `skill_templates_and_skills_ux` | `partial` | `warning` | `learning_loop`, `prompt_assembly` | `../agent-zero/agent.py@7c71185f:agent_init,monologue_start,message_loop_start,before_main_llm_call,response_stream_chunk,reasoning_stream_chunk,message_loop_end,monologue_end`, `../agent-zero/helpers/extension.py@7c71185f:extensible,call_extensions_async,call_extensions_sync`, `../hermes-agent/agent/`, `../hermes-agent/agent/context_compressor.py@5401a008`, `+218 more` | Mapped through surfaces: learning_loop, prompt_assembly; worst status is learning_loop=partial. |
| `skill_precedence_sync_update_reset` | `partial` | `warning` | `learning_loop`, `prompt_assembly` | `../agent-zero/agent.py@7c71185f:agent_init,monologue_start,message_loop_start,before_main_llm_call,response_stream_chunk,reasoning_stream_chunk,message_loop_end,monologue_end`, `../agent-zero/helpers/extension.py@7c71185f:extensible,call_extensions_async,call_extensions_sync`, `../hermes-agent/agent/`, `../hermes-agent/agent/context_compressor.py@5401a008`, `+218 more` | Mapped through surfaces: learning_loop, prompt_assembly; worst status is learning_loop=partial. |
| `learning_loop_curator_behavior` | `partial` | `warning` | `learning_loop` | `../hermes-agent/hermes_cli/skills_hub.py@e63929d4:_is_valid_installed_skill_name`, `../hermes-agent/hermes_cli/skills_hub.py@e63929d4:do_install,_prompt_for_skill_name,_prompt_for_category,_existing_categories`, `../hermes-agent/skills/creative/claude-design/SKILL.md@8c5d3a99`, `../hermes-agent/skills/creative/design-md/SKILL.md@55be5323`, `+68 more` | Mapped through surfaces: learning_loop; worst status is learning_loop=partial. |
| `candidate_memory_skill_updates` | `partial` | `warning` | `learning_loop`, `goncho_memory` | `../hermes-agent/acp_adapter/`, `../hermes-agent/agent/`, `../hermes-agent/agent/memory_manager.py`, `../hermes-agent/agent/memory_manager.py:178`, `+294 more` | Mapped through surfaces: learning_loop, goncho_memory; worst status is learning_loop=partial. |
| `feedback_outcome_scoring` | `partial` | `warning` | `learning_loop` | `../hermes-agent/hermes_cli/skills_hub.py@e63929d4:_is_valid_installed_skill_name`, `../hermes-agent/hermes_cli/skills_hub.py@e63929d4:do_install,_prompt_for_skill_name,_prompt_for_category,_existing_categories`, `../hermes-agent/skills/creative/claude-design/SKILL.md@8c5d3a99`, `../hermes-agent/skills/creative/design-md/SKILL.md@55be5323`, `+68 more` | Mapped through surfaces: learning_loop; worst status is learning_loop=partial. |
| `audit_trail` | `partial` | `warning` | `sessions`, `tool_runtime`, `learning_loop` | `../agent-zero@7c71185f/agent.py:prepare_prompt,read_prompt,parse_prompt`, `../agent-zero@7c71185f/helpers/files.py:read_prompt_file,process_includes,_get_dirs_after`, `../agent-zero@7c71185f/prompts/ (72 fragment files)`, `../agent-zero@7c71185f/skills/a0-development/SKILL.md:Prompt System`, `+691 more` | Mapped through surfaces: sessions, tool_runtime, learning_loop; worst status is sessions=partial. |
| `mutation_safety` | `partial` | `warning` | `tool_runtime`, `learning_loop`, `goncho_memory` | `../agent-zero@7c71185f/agent.py:prepare_prompt,read_prompt,parse_prompt`, `../agent-zero@7c71185f/helpers/files.py:read_prompt_file,process_includes,_get_dirs_after`, `../agent-zero@7c71185f/prompts/ (72 fragment files)`, `../agent-zero@7c71185f/skills/a0-development/SKILL.md:Prompt System`, `+812 more` | Mapped through surfaces: tool_runtime, learning_loop, goncho_memory; worst status is tool_runtime=partial. |
| `prompt_context_memory_skill_insertion_ordering` | `partial` | `warning` | `prompt_assembly`, `goncho_memory`, `sessions`, `learning_loop` | `../agent-zero/agent.py@7c71185f:agent_init,monologue_start,message_loop_start,before_main_llm_call,response_stream_chunk,reasoning_stream_chunk,message_loop_end,monologue_end`, `../agent-zero/helpers/extension.py@7c71185f:extensible,call_extensions_async,call_extensions_sync`, `../hermes-agent/acp_adapter/`, `../hermes-agent/agent/`, `+447 more` | Mapped through surfaces: prompt_assembly, goncho_memory, sessions, learning_loop; worst status is prompt_assembly=partial. |
| `profile_scoped_isolation` | `partial` | `warning` | `profiles`, `sessions`, `gateway_channels` | `../hermes-agent/agent/account_usage.py:render_account_usage_lines,fetch_account_usage`, `../hermes-agent/agent/bedrock_adapter.py:651:normalize_converse_stream_events`, `../hermes-agent/agent/bedrock_adapter.py:673:stream_converse_with_callbacks`, `../hermes-agent/agent/context_compressor.py@5401a008`, `+1134 more` | Mapped through surfaces: profiles, sessions, gateway_channels; worst status is sessions=partial. |

## Unmapped Upstream Evidence

Unmapped upstream files are strict-mode blockers until each file is joined to a progress row, a source-pair entry, a surface classification, or an explicit exclusion.

| Family | Count | Examples |
|---|---:|---|
| Source | `860` | `acp_adapter/__init__.py`, `acp_adapter/__main__.py`, `acp_adapter/auth.py`, `acp_adapter/bootstrap/__init__.py`, `acp_adapter/events.py`, `acp_adapter/permissions.py`, `acp_adapter/session.py`, `acp_adapter/tools.py`, `agent/__init__.py`, `agent/account_usage.py`, `+850 more` |
| Docs | `959` | `.github/PULL_REQUEST_TEMPLATE.md`, `.plans/openai-api-server.md`, `.plans/streaming-support.md`, `CONTRIBUTING.md`, `README.zh-CN.md`, `SECURITY.md`, `docker/SOUL.md`, `docs/plans/2026-05-02-telegram-dm-user-managed-multisession-topics.md`, `gateway/platforms/ADDING_A_PLATFORM.md`, `hermes-already-has-routines.md`, `+949 more` |
| Tests | `1198` | `plugins/hermes-achievements/tests/test_achievement_engine.py`, `scripts/tests/test-install-ps1-stage-protocol.ps1`, `skills/creative/comfyui/tests/README.md`, `skills/creative/comfyui/tests/conftest.py`, `skills/creative/comfyui/tests/pytest.ini`, `skills/creative/comfyui/tests/test_check_deps.py`, `skills/creative/comfyui/tests/test_cloud_integration.py`, `skills/creative/comfyui/tests/test_common.py`, `skills/creative/comfyui/tests/test_extract_schema.py`, `skills/creative/comfyui/tests/test_run_workflow.py`, `+1188 more` |

## Candidate Inventory

| Candidate family | Count | Examples |
|---|---:|---|
| CLI | `90` | `cli.py`, `hermes_cli/__init__.py`, `hermes_cli/_parser.py`, `hermes_cli/_subprocess_compat.py`, `hermes_cli/auth.py`, `+85 more` |
| Tools | `93` | `tools/__init__.py`, `tools/ansi_strip.py`, `tools/approval.py`, `tools/binary_extensions.py`, `tools/browser_camofox.py`, `+88 more` |
| Providers | `123` | `agent/browser_provider.py`, `agent/image_gen_provider.py`, `agent/memory_provider.py`, `agent/video_gen_provider.py`, `agent/web_search_provider.py`, `+118 more` |
| Channels | `134` | `docs/plans/2026-05-02-telegram-dm-user-managed-multisession-topics.md`, `gateway/__init__.py`, `gateway/builtin_hooks/__init__.py`, `gateway/channel_directory.py`, `gateway/config.py`, `+129 more` |
| Sessions | `58` | `acp_adapter/session.py`, `agent/transports/codex_app_server_session.py`, `docs/plans/2026-05-02-telegram-dm-user-managed-multisession-topics.md`, `gateway/session.py`, `gateway/session_context.py`, `+53 more` |
| Memory | `64` | `agent/memory_manager.py`, `agent/memory_provider.py`, `gateway/memory_monitor.py`, `hermes_cli/memory_setup.py`, `optional-skills/autonomous-ai-agents/honcho/SKILL.md`, `+59 more` |
| Skills | `938` | `agent/skill_commands.py`, `agent/skill_preprocessing.py`, `agent/skill_utils.py`, `hermes_cli/skills_config.py`, `hermes_cli/skills_hub.py`, `+933 more` |
| Learning loop | `20` | `agent/curator.py`, `agent/curator_backup.py`, `hermes_cli/curator.py`, `optional-skills/mlops/nemo-curator/SKILL.md`, `optional-skills/mlops/nemo-curator/references/deduplication.md`, `+15 more` |

## Surface Classification

| Surface | Status | Severity | Confidence | Progress rows | Source pairs | Reason |
|---|---|---|---|---|---|---|
| `profiles` | `covered` | `none` | `high` | `Active Hermes/Sidon profile context root resolver for live turns`, `Agent personalities + enhanced display config`, `Agent personalities + enhanced display config`, `CLI active-profile store`, `+28 more` | `agent/prompt_builder.py`, `hermes_cli/config.py`, `hermes_cli/default_soul.py`, `hermes_cli/profiles.py`, `+1 more` | Covered source-pair evidence joins to a validated complete progress row with test evidence. |
| `sessions` | `partial` | `warning` | `medium` | `Auto-naming sessions`, `Aux compression single-prompt threshold reconciliation`, `Behavioral pattern extraction from session logs`, `Busy command guard for compression and long CLI actions`, `+28 more` | `agent/prompt_builder.py`, `run_agent.py` | Some source-pair or complete-row evidence exists, but the surface is not strictly covered. |
| `goncho_memory` | `partial` | `warning` | `medium` | `2-layer seed selection`, `6 typed memory categories with confidence scoring`, `<memory-context> fence`, `Agent-controlled memory retention with importance scoring`, `+65 more` | `agent/prompt_builder.py` | Some source-pair or complete-row evidence exists, but the surface is not strictly covered. |
| `learning_loop` | `partial` | `warning` | `medium` | `Agent-controlled memory retention with importance scoring`, `Code Cathedral II code-context retrieval fixtures`, `Goncho durable recall trace IR + fused ranking pipeline`, `Goncho recall diagnostics CLI over RecallTrace`, `+13 more` | `tools/skill_manager_tool.py` | Some source-pair or complete-row evidence exists, but the surface is not strictly covered. |
| `prompt_assembly` | `partial` | `warning` | `medium` | `<memory-context> fence`, `Agent lifecycle hooks (agent:start, agent:step, agent:end)`, `Aux compression single-prompt threshold reconciliation`, `Bundled Airtable productivity skill contract`, `+37 more` | `agent/prompt_builder.py`, `run_agent.py`, `tools/skills_tool.py` | Some source-pair or complete-row evidence exists, but the surface is not strictly covered. |
| `provider_auth_setup` | `partial` | `warning` | `medium` | `Anthropic`, `Anthropic OAuth/keychain credential discovery`, `Aux compression provider-aware context cap`, `Azure Anthropic Messages endpoint contract`, `+147 more` | `agent/prompt_builder.py`, `cli.py`, `hermes_cli/auth_commands.py`, `hermes_cli/config.py`, `+1 more` | Some source-pair or complete-row evidence exists, but the surface is not strictly covered. |
| `gateway_channels` | `partial` | `warning` | `medium` | `(platform, chat_id) -> session_id`, `API server cron admin mutating endpoints`, `API server cron admin read-only endpoints`, `API server detailed health endpoint`, `+289 more` | `agent/prompt_builder.py`, `gateway/run.py` | Some source-pair or complete-row evidence exists, but the surface is not strictly covered. |
| `tool_runtime` | `partial` | `warning` | `medium` | `61-tool registry port`, `ACP Client Bridge Mode`, `ACP JSON-RPC stdio session/prompt closeout`, `ACP server side`, `+147 more` | `gateway/run.py`, `run_agent.py`, `tools/skill_manager_tool.py`, `tools/skills_sync.py`, `+1 more` | Some source-pair or complete-row evidence exists, but the surface is not strictly covered. |
| `mcp_acp` | `partial` | `warning` | `medium` | `ACP Client Bridge Mode`, `ACP JSON-RPC stdio session/prompt closeout`, `ACP server side`, `ACP session CWD propagation into prompt runners`, `+24 more` | - | Some source-pair or complete-row evidence exists, but the surface is not strictly covered. |
| `tui_cli` | `covered` | `none` | `high` | `16ms coalescing mailbox`, `Admin TUI: Agents screen wired to the 2.H dynamic registry`, `Admin TUI: Chat tab with keybinding to jump in from any screen`, `Admin TUI: Commands catalog over the root CLI tree`, `+56 more` | `cli.py`, `gateway/run.py`, `hermes_cli/auth_commands.py`, `hermes_cli/commands.py`, `+4 more` | Covered source-pair evidence joins to a validated complete progress row with test evidence. |
