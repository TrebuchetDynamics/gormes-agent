# Hermes Contract Inventory

- Hermes SHA: `43e566f77eaf01293086eb7cb99a21e240d60634`
- Generated: `2026-05-21T15:24:48Z`
- Source pairs: `current` (`43e566f77eaf01293086eb7cb99a21e240d60634`)
- Report mode: `report-only`
- Progress source: `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Backlog policy: `progress.json` remains the only backlog; this report classifies evidence and does not create work rows.
- Claim boundary: Gormes may claim all Hermes features and architecture are paired only when every current-SHA inventory gap is classified as `covered`, `partial`, `planned`, `excluded`, or `owned_divergence`; strict mode additionally requires every critical surface to be `covered`, `excluded`, or `owned_divergence` and every upstream source/doc/test file to be mapped or explicitly excluded.

## Headline Counts

- Source files: `2147`
- Docs files: `983`
- Test files: `1220`
- Unmapped upstream source files: `857`
- Unmapped upstream docs files: `959`
- Unmapped upstream test files: `1164`
- Release checkpoints: `16`
- Critical surfaces: `10`
- Surface strict failures: `4`
- Strict failures: `2984`
- `covered`: `6`
- `partial`: `4`

## Critical Surface Blockers

No critical surface blockers in the current classification. `2980` unmapped upstream source/doc/test files still block strict mode.

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
| `continuity` | `profiles`, `sessions`, `goncho_memory`, `learning_loop`, `prompt_assembly` | `3` | `2` | `0` | `0` | `warning` |
| `operator` | `provider_auth_setup`, `tui_cli` | `2` | `0` | `0` | `0` | `none` |
| `runtime` | `tool_runtime` | `1` | `0` | `0` | `0` | `none` |
| `tools` | `mcp_acp` | `0` | `1` | `0` | `0` | `warning` |

## Continuity Categories

| Category | Status | Severity | Surfaces | Evidence | Reason |
|---|---|---|---|---|---|
| `sessions` | `covered` | `none` | `sessions` | `../hermes-agent/agent/context_compressor.py@5401a008`, `../hermes-agent/agent/context_compressor.py@94346523:_find_tail_cut_by_tokens`, `../hermes-agent/agent/context_compressor.py@bda2dbc2:_calculate_protect_tail_boundary`, `../hermes-agent/agent/context_compressor.py@cfc8befe:_find_tail_cut_by_tokens`, `+142 more` | Mapped surfaces are strictly covered: sessions. |
| `memory_goncho_honcho_compatibility` | `partial` | `warning` | `goncho_memory` | `../hermes-agent/acp_adapter/`, `../hermes-agent/agent/`, `../hermes-agent/agent/memory_manager.py`, `../hermes-agent/agent/memory_manager.py:178`, `+290 more` | Mapped through surfaces: goncho_memory; worst status is goncho_memory=partial. |
| `workspace_peer_profile_identity_boundaries` | `covered` | `none` | `profiles` | `../hermes-agent/agent/lsp/manager.py`, `../hermes-agent/agent/lsp/range_shift.py`, `../hermes-agent/agent/prompt_builder.py`, `../hermes-agent/agent/prompt_builder.py:32-73`, `+182 more` | Mapped surfaces are strictly covered: profiles. |
| `context_retrieval_and_prompt_budget` | `covered` | `none` | `sessions`, `prompt_assembly` | `../agent-zero/agent.py@7c71185f:agent_init,monologue_start,message_loop_start,before_main_llm_call,response_stream_chunk,reasoning_stream_chunk,message_loop_end,monologue_end`, `../agent-zero/helpers/extension.py@7c71185f:extensible,call_extensions_async,call_extensions_sync`, `../hermes-agent/agent/`, `../hermes-agent/agent/context_compressor.py@5401a008`, `+271 more` | Mapped surfaces are strictly covered: sessions, prompt_assembly. |
| `summaries_conclusions_search` | `covered` | `none` | `sessions` | `../hermes-agent/agent/context_compressor.py@5401a008`, `../hermes-agent/agent/context_compressor.py@94346523:_find_tail_cut_by_tokens`, `../hermes-agent/agent/context_compressor.py@bda2dbc2:_calculate_protect_tail_boundary`, `../hermes-agent/agent/context_compressor.py@cfc8befe:_find_tail_cut_by_tokens`, `+142 more` | Mapped surfaces are strictly covered: sessions. |
| `skill_templates_and_skills_ux` | `partial` | `warning` | `learning_loop`, `prompt_assembly` | `../agent-zero/agent.py@7c71185f:agent_init,monologue_start,message_loop_start,before_main_llm_call,response_stream_chunk,reasoning_stream_chunk,message_loop_end,monologue_end`, `../agent-zero/helpers/extension.py@7c71185f:extensible,call_extensions_async,call_extensions_sync`, `../hermes-agent/agent/`, `../hermes-agent/agent/context_compressor.py@5401a008`, `+273 more` | Mapped through surfaces: learning_loop, prompt_assembly; worst status is learning_loop=partial. |
| `skill_precedence_sync_update_reset` | `partial` | `warning` | `learning_loop`, `prompt_assembly` | `../agent-zero/agent.py@7c71185f:agent_init,monologue_start,message_loop_start,before_main_llm_call,response_stream_chunk,reasoning_stream_chunk,message_loop_end,monologue_end`, `../agent-zero/helpers/extension.py@7c71185f:extensible,call_extensions_async,call_extensions_sync`, `../hermes-agent/agent/`, `../hermes-agent/agent/context_compressor.py@5401a008`, `+273 more` | Mapped through surfaces: learning_loop, prompt_assembly; worst status is learning_loop=partial. |
| `learning_loop_curator_behavior` | `partial` | `warning` | `learning_loop` | `../hermes-agent/agent/lsp/manager.py`, `../hermes-agent/agent/lsp/range_shift.py`, `../hermes-agent/hermes_cli/skills_hub.py@e63929d4:_is_valid_installed_skill_name`, `../hermes-agent/hermes_cli/skills_hub.py@e63929d4:do_install,_prompt_for_skill_name,_prompt_for_category,_existing_categories`, `+96 more` | Mapped through surfaces: learning_loop; worst status is learning_loop=partial. |
| `candidate_memory_skill_updates` | `partial` | `warning` | `learning_loop`, `goncho_memory` | `../hermes-agent/acp_adapter/`, `../hermes-agent/agent/`, `../hermes-agent/agent/lsp/manager.py`, `../hermes-agent/agent/lsp/range_shift.py`, `+340 more` | Mapped through surfaces: learning_loop, goncho_memory; worst status is learning_loop=partial. |
| `feedback_outcome_scoring` | `partial` | `warning` | `learning_loop` | `../hermes-agent/agent/lsp/manager.py`, `../hermes-agent/agent/lsp/range_shift.py`, `../hermes-agent/hermes_cli/skills_hub.py@e63929d4:_is_valid_installed_skill_name`, `../hermes-agent/hermes_cli/skills_hub.py@e63929d4:do_install,_prompt_for_skill_name,_prompt_for_category,_existing_categories`, `+96 more` | Mapped through surfaces: learning_loop; worst status is learning_loop=partial. |
| `audit_trail` | `partial` | `warning` | `sessions`, `tool_runtime`, `learning_loop` | `../agent-zero@7c71185f/agent.py:prepare_prompt,read_prompt,parse_prompt`, `../agent-zero@7c71185f/helpers/files.py:read_prompt_file,process_includes,_get_dirs_after`, `../agent-zero@7c71185f/prompts/ (72 fragment files)`, `../agent-zero@7c71185f/skills/a0-development/SKILL.md:Prompt System`, `+789 more` | Mapped through surfaces: sessions, tool_runtime, learning_loop; worst status is learning_loop=partial. |
| `mutation_safety` | `partial` | `warning` | `tool_runtime`, `learning_loop`, `goncho_memory` | `../agent-zero@7c71185f/agent.py:prepare_prompt,read_prompt,parse_prompt`, `../agent-zero@7c71185f/helpers/files.py:read_prompt_file,process_includes,_get_dirs_after`, `../agent-zero@7c71185f/prompts/ (72 fragment files)`, `../agent-zero@7c71185f/skills/a0-development/SKILL.md:Prompt System`, `+926 more` | Mapped through surfaces: tool_runtime, learning_loop, goncho_memory; worst status is learning_loop=partial. |
| `prompt_context_memory_skill_insertion_ordering` | `partial` | `warning` | `prompt_assembly`, `goncho_memory`, `sessions`, `learning_loop` | `../agent-zero/agent.py@7c71185f:agent_init,monologue_start,message_loop_start,before_main_llm_call,response_stream_chunk,reasoning_stream_chunk,message_loop_end,monologue_end`, `../agent-zero/helpers/extension.py@7c71185f:extensible,call_extensions_async,call_extensions_sync`, `../hermes-agent/acp_adapter/`, `../hermes-agent/agent/`, `+543 more` | Mapped through surfaces: prompt_assembly, goncho_memory, sessions, learning_loop; worst status is goncho_memory=partial. |
| `profile_scoped_isolation` | `partial` | `warning` | `profiles`, `sessions`, `gateway_channels` | `../hermes-agent/agent/account_usage.py:render_account_usage_lines,fetch_account_usage`, `../hermes-agent/agent/bedrock_adapter.py:651:normalize_converse_stream_events`, `../hermes-agent/agent/bedrock_adapter.py:673:stream_converse_with_callbacks`, `../hermes-agent/agent/context_compressor.py@5401a008`, `+1189 more` | Mapped through surfaces: profiles, sessions, gateway_channels; worst status is gateway_channels=partial. |

## Unmapped Upstream Evidence

Unmapped upstream files are strict-mode blockers until each file is joined to a progress row, a source-pair entry, a surface classification, or an explicit exclusion.

| Family | Count | Examples |
|---|---:|---|
| Source | `857` | `acp_adapter/__init__.py`, `acp_adapter/__main__.py`, `acp_adapter/auth.py`, `acp_adapter/bootstrap/__init__.py`, `acp_adapter/events.py`, `acp_adapter/permissions.py`, `acp_adapter/session.py`, `acp_adapter/tools.py`, `agent/__init__.py`, `agent/account_usage.py`, `+847 more` |
| Docs | `959` | `.github/PULL_REQUEST_TEMPLATE.md`, `.plans/openai-api-server.md`, `.plans/streaming-support.md`, `CONTRIBUTING.md`, `README.zh-CN.md`, `SECURITY.md`, `docker/SOUL.md`, `docs/plans/2026-05-02-telegram-dm-user-managed-multisession-topics.md`, `gateway/platforms/ADDING_A_PLATFORM.md`, `hermes-already-has-routines.md`, `+949 more` |
| Tests | `1164` | `plugins/hermes-achievements/tests/test_achievement_engine.py`, `scripts/tests/test-install-ps1-stage-protocol.ps1`, `skills/creative/comfyui/tests/README.md`, `skills/creative/comfyui/tests/conftest.py`, `skills/creative/comfyui/tests/pytest.ini`, `skills/creative/comfyui/tests/test_check_deps.py`, `skills/creative/comfyui/tests/test_cloud_integration.py`, `skills/creative/comfyui/tests/test_common.py`, `skills/creative/comfyui/tests/test_extract_schema.py`, `skills/creative/comfyui/tests/test_run_workflow.py`, `+1154 more` |

## Unmapped Test Suite Classification

| Suite | Count | Source under test | Progress rows | Examples |
|---|---:|---|---|---|
| `__init__.py` | `1` | `__init__.py` | - | `tests/__init__.py` |
| `acp` | `12` | `acp` | - | `tests/acp/__init__.py`, `tests/acp/test_approval_isolation.py`, `tests/acp/test_auth.py`, `tests/acp/test_entry.py`, `tests/acp/test_events.py` |
| `acp_adapter` | `2` | `acp_adapter` | `ACP session CWD propagation into prompt runners`, `ACP stdio benign ping/probe suppression` | `tests/acp_adapter/test_acp_commands.py`, `tests/acp_adapter/test_acp_images.py` |
| `agent/__init__.py` | `1` | `agent/__init__.py` | - | `tests/agent/__init__.py` |
| `agent/lsp` | `10` | `agent/lsp` | `Hermes agent runtime strict-fidelity source-pair expansion` | `tests/agent/lsp/__init__.py`, `tests/agent/lsp/_mock_lsp_server.py`, `tests/agent/lsp/test_backend_gate.py`, `tests/agent/lsp/test_broken_set.py`, `tests/agent/lsp/test_client_e2e.py` |
| `agent/test_anthropic_adapter.py` | `1` | `agent/test_anthropic_adapter.py` | - | `tests/agent/test_anthropic_adapter.py` |
| `agent/test_anthropic_keychain.py` | `1` | `agent/test_anthropic_keychain.py` | - | `tests/agent/test_anthropic_keychain.py` |
| `agent/test_anthropic_oauth_pkce.py` | `1` | `agent/test_anthropic_oauth_pkce.py` | - | `tests/agent/test_anthropic_oauth_pkce.py` |
| `agent/test_arcee_trinity_overrides.py` | `1` | `agent/test_arcee_trinity_overrides.py` | - | `tests/agent/test_arcee_trinity_overrides.py` |
| `agent/test_async_utils.py` | `1` | `agent/test_async_utils.py` | - | `tests/agent/test_async_utils.py` |
| `agent/test_auxiliary_client.py` | `1` | `agent/test_auxiliary_client.py` | - | `tests/agent/test_auxiliary_client.py` |
| `agent/test_auxiliary_client_anthropic_custom.py` | `1` | `agent/test_auxiliary_client_anthropic_custom.py` | - | `tests/agent/test_auxiliary_client_anthropic_custom.py` |
| `agent/test_auxiliary_config_bridge.py` | `1` | `agent/test_auxiliary_config_bridge.py` | - | `tests/agent/test_auxiliary_config_bridge.py` |
| `agent/test_auxiliary_main_first.py` | `1` | `agent/test_auxiliary_main_first.py` | - | `tests/agent/test_auxiliary_main_first.py` |
| `agent/test_auxiliary_named_custom_providers.py` | `1` | `agent/test_auxiliary_named_custom_providers.py` | - | `tests/agent/test_auxiliary_named_custom_providers.py` |
| `agent/test_auxiliary_transport_autodetect.py` | `1` | `agent/test_auxiliary_transport_autodetect.py` | - | `tests/agent/test_auxiliary_transport_autodetect.py` |
| `agent/test_bedrock_1m_context.py` | `1` | `agent/test_bedrock_1m_context.py` | - | `tests/agent/test_bedrock_1m_context.py` |
| `agent/test_bedrock_adapter.py` | `1` | `agent/test_bedrock_adapter.py` | - | `tests/agent/test_bedrock_adapter.py` |
| `agent/test_bedrock_integration.py` | `1` | `agent/test_bedrock_integration.py` | - | `tests/agent/test_bedrock_integration.py` |
| `agent/test_codex_cloudflare_headers.py` | `1` | `agent/test_codex_cloudflare_headers.py` | - | `tests/agent/test_codex_cloudflare_headers.py` |
| `agent/test_compress_focus.py` | `1` | `agent/test_compress_focus.py` | - | `tests/agent/test_compress_focus.py` |
| `agent/test_compressor_historical_media.py` | `1` | `agent/test_compressor_historical_media.py` | - | `tests/agent/test_compressor_historical_media.py` |
| `agent/test_compressor_image_tokens.py` | `1` | `agent/test_compressor_image_tokens.py` | - | `tests/agent/test_compressor_image_tokens.py` |
| `agent/test_context_compressor_summary_continuity.py` | `1` | `agent/test_context_compressor_summary_continuity.py` | - | `tests/agent/test_context_compressor_summary_continuity.py` |
| `agent/test_context_references.py` | `1` | `agent/test_context_references.py` | - | `tests/agent/test_context_references.py` |
| `agent/test_copilot_acp_client.py` | `1` | `agent/test_copilot_acp_client.py` | - | `tests/agent/test_copilot_acp_client.py` |
| `agent/test_copilot_acp_deprecation.py` | `1` | `agent/test_copilot_acp_deprecation.py` | - | `tests/agent/test_copilot_acp_deprecation.py` |
| `agent/test_credential_pool.py` | `1` | `agent/test_credential_pool.py` | - | `tests/agent/test_credential_pool.py` |
| `agent/test_credential_pool_routing.py` | `1` | `agent/test_credential_pool_routing.py` | - | `tests/agent/test_credential_pool_routing.py` |
| `agent/test_crossloop_client_cache.py` | `1` | `agent/test_crossloop_client_cache.py` | - | `tests/agent/test_crossloop_client_cache.py` |
| `agent/test_curator.py` | `1` | `agent/test_curator.py` | - | `tests/agent/test_curator.py` |
| `agent/test_curator_activity.py` | `1` | `agent/test_curator_activity.py` | - | `tests/agent/test_curator_activity.py` |
| `agent/test_curator_backup.py` | `1` | `agent/test_curator_backup.py` | - | `tests/agent/test_curator_backup.py` |
| `agent/test_curator_classification.py` | `1` | `agent/test_curator_classification.py` | - | `tests/agent/test_curator_classification.py` |
| `agent/test_curator_reports.py` | `1` | `agent/test_curator_reports.py` | - | `tests/agent/test_curator_reports.py` |
| `agent/test_deepseek_anthropic_thinking.py` | `1` | `agent/test_deepseek_anthropic_thinking.py` | - | `tests/agent/test_deepseek_anthropic_thinking.py` |
| `agent/test_display.py` | `1` | `agent/test_display.py` | - | `tests/agent/test_display.py` |
| `agent/test_display_emoji.py` | `1` | `agent/test_display_emoji.py` | - | `tests/agent/test_display_emoji.py` |
| `agent/test_error_classifier.py` | `1` | `agent/test_error_classifier.py` | - | `tests/agent/test_error_classifier.py` |
| `agent/test_external_skills.py` | `1` | `agent/test_external_skills.py` | - | `tests/agent/test_external_skills.py` |
| `agent/test_external_skills_dirs_cache.py` | `1` | `agent/test_external_skills_dirs_cache.py` | - | `tests/agent/test_external_skills_dirs_cache.py` |
| `agent/test_gemini_cloudcode.py` | `1` | `agent/test_gemini_cloudcode.py` | - | `tests/agent/test_gemini_cloudcode.py` |
| `agent/test_gemini_fast_fallback.py` | `1` | `agent/test_gemini_fast_fallback.py` | - | `tests/agent/test_gemini_fast_fallback.py` |
| `agent/test_gemini_free_tier_gate.py` | `1` | `agent/test_gemini_free_tier_gate.py` | - | `tests/agent/test_gemini_free_tier_gate.py` |
| `agent/test_gemini_native_adapter.py` | `1` | `agent/test_gemini_native_adapter.py` | - | `tests/agent/test_gemini_native_adapter.py` |
| `agent/test_gemini_schema.py` | `1` | `agent/test_gemini_schema.py` | - | `tests/agent/test_gemini_schema.py` |
| `agent/test_image_gen_registry.py` | `1` | `agent/test_image_gen_registry.py` | - | `tests/agent/test_image_gen_registry.py` |
| `agent/test_image_routing.py` | `1` | `agent/test_image_routing.py` | - | `tests/agent/test_image_routing.py` |
| `agent/test_insights.py` | `1` | `agent/test_insights.py` | - | `tests/agent/test_insights.py` |
| `agent/test_kimi_coding_anthropic_thinking.py` | `1` | `agent/test_kimi_coding_anthropic_thinking.py` | - | `tests/agent/test_kimi_coding_anthropic_thinking.py` |
| `agent/test_local_stream_timeout.py` | `1` | `agent/test_local_stream_timeout.py` | - | `tests/agent/test_local_stream_timeout.py` |
| `agent/test_markdown_tables.py` | `1` | `agent/test_markdown_tables.py` | - | `tests/agent/test_markdown_tables.py` |
| `agent/test_memory_provider.py` | `1` | `agent/test_memory_provider.py` | - | `tests/agent/test_memory_provider.py` |
| `agent/test_memory_session_switch.py` | `1` | `agent/test_memory_session_switch.py` | - | `tests/agent/test_memory_session_switch.py` |
| `agent/test_memory_user_id.py` | `1` | `agent/test_memory_user_id.py` | - | `tests/agent/test_memory_user_id.py` |
| `agent/test_minimax_auxiliary_url.py` | `1` | `agent/test_minimax_auxiliary_url.py` | - | `tests/agent/test_minimax_auxiliary_url.py` |
| `agent/test_minimax_provider.py` | `1` | `agent/test_minimax_provider.py` | - | `tests/agent/test_minimax_provider.py` |
| `agent/test_model_metadata.py` | `1` | `agent/test_model_metadata.py` | - | `tests/agent/test_model_metadata.py` |
| `agent/test_model_metadata_local_ctx.py` | `1` | `agent/test_model_metadata_local_ctx.py` | - | `tests/agent/test_model_metadata_local_ctx.py` |
| `agent/test_model_metadata_ssl.py` | `1` | `agent/test_model_metadata_ssl.py` | - | `tests/agent/test_model_metadata_ssl.py` |
| `agent/test_models_dev.py` | `1` | `agent/test_models_dev.py` | - | `tests/agent/test_models_dev.py` |
| `agent/test_moonshot_schema.py` | `1` | `agent/test_moonshot_schema.py` | - | `tests/agent/test_moonshot_schema.py` |
| `agent/test_nous_rate_guard.py` | `1` | `agent/test_nous_rate_guard.py` | - | `tests/agent/test_nous_rate_guard.py` |
| `agent/test_onboarding.py` | `1` | `agent/test_onboarding.py` | - | `tests/agent/test_onboarding.py` |
| `agent/test_openrouter_response_cache.py` | `1` | `agent/test_openrouter_response_cache.py` | - | `tests/agent/test_openrouter_response_cache.py` |
| `agent/test_plugin_llm.py` | `1` | `agent/test_plugin_llm.py` | - | `tests/agent/test_plugin_llm.py` |
| `agent/test_portal_tags.py` | `1` | `agent/test_portal_tags.py` | - | `tests/agent/test_portal_tags.py` |
| `agent/test_prompt_builder.py` | `1` | `agent/test_prompt_builder.py` | - | `tests/agent/test_prompt_builder.py` |
| `agent/test_prompt_caching.py` | `1` | `agent/test_prompt_caching.py` | - | `tests/agent/test_prompt_caching.py` |
| `agent/test_proxy_and_url_validation.py` | `1` | `agent/test_proxy_and_url_validation.py` | - | `tests/agent/test_proxy_and_url_validation.py` |
| `agent/test_rate_limit_tracker.py` | `1` | `agent/test_rate_limit_tracker.py` | - | `tests/agent/test_rate_limit_tracker.py` |
| `agent/test_redact.py` | `1` | `agent/test_redact.py` | - | `tests/agent/test_redact.py` |
| `agent/test_shell_hooks.py` | `1` | `agent/test_shell_hooks.py` | - | `tests/agent/test_shell_hooks.py` |
| `agent/test_shell_hooks_consent.py` | `1` | `agent/test_shell_hooks_consent.py` | - | `tests/agent/test_shell_hooks_consent.py` |
| `agent/test_skill_commands.py` | `1` | `agent/test_skill_commands.py` | - | `tests/agent/test_skill_commands.py` |
| `agent/test_skill_commands_reload.py` | `1` | `agent/test_skill_commands_reload.py` | - | `tests/agent/test_skill_commands_reload.py` |
| `agent/test_skill_utils.py` | `1` | `agent/test_skill_utils.py` | - | `tests/agent/test_skill_utils.py` |
| `agent/test_subagent_progress.py` | `1` | `agent/test_subagent_progress.py` | - | `tests/agent/test_subagent_progress.py` |
| `agent/test_subagent_stop_hook.py` | `1` | `agent/test_subagent_stop_hook.py` | - | `tests/agent/test_subagent_stop_hook.py` |
| `agent/test_subdirectory_hints.py` | `1` | `agent/test_subdirectory_hints.py` | - | `tests/agent/test_subdirectory_hints.py` |
| `agent/test_think_scrubber.py` | `1` | `agent/test_think_scrubber.py` | - | `tests/agent/test_think_scrubber.py` |
| `agent/test_title_generator.py` | `1` | `agent/test_title_generator.py` | - | `tests/agent/test_title_generator.py` |
| `agent/test_usage_pricing.py` | `1` | `agent/test_usage_pricing.py` | - | `tests/agent/test_usage_pricing.py` |
| `agent/test_video_gen_registry.py` | `1` | `agent/test_video_gen_registry.py` | - | `tests/agent/test_video_gen_registry.py` |
| `agent/test_vision_resolved_args.py` | `1` | `agent/test_vision_resolved_args.py` | - | `tests/agent/test_vision_resolved_args.py` |
| `agent/transports` | `5` | `agent/transports` | `Hermes agent runtime strict-fidelity source-pair expansion`, `Hermes fast-mode request override serializer`, `OpenRouter Pareto router request plugin` | `tests/agent/transports/__init__.py`, `tests/agent/transports/test_bedrock_transport.py`, `tests/agent/transports/test_codex_app_server_session.py`, `tests/agent/transports/test_hermes_tools_mcp_server.py`, `tests/agent/transports/test_types.py` |
| `cli` | `61` | `cli` | - | `tests/cli/__init__.py`, `tests/cli/test_branch_command.py`, `tests/cli/test_busy_input_mode_command.py`, `tests/cli/test_cli_approval_ui.py`, `tests/cli/test_cli_background_status_indicator.py` |
| `conftest.py` | `1` | `conftest.py` | - | `tests/conftest.py` |
| `cron` | `13` | `cron` | `Cron deliver=all routing intent expansion`, `Cron env-ref expansion + parallel run state serialization`, `Cron no-agent script-only short-circuit`, `Cron origin delivery isolation from session identity`, `Cron partial legacy job read-model normalization` | `tests/cron/__init__.py`, `tests/cron/test_codex_execution_paths.py`, `tests/cron/test_compute_next_run_last_run_at.py`, `tests/cron/test_cron_context_from.py`, `tests/cron/test_cron_inactivity_timeout.py` |
| `e2e` | `7` | `e2e` | - | `tests/e2e/__init__.py`, `tests/e2e/conftest.py`, `tests/e2e/matrix_xsign_bootstrap/README.md`, `tests/e2e/matrix_xsign_bootstrap/docker-compose.yml`, `tests/e2e/matrix_xsign_bootstrap/test_bootstrap.py` |
| `fakes` | `2` | `fakes` | - | `tests/fakes/__init__.py`, `tests/fakes/fake_ha_server.py` |
| `gateway` | `249` | `gateway` | `Agent lifecycle hooks (agent:start, agent:step, agent:end)`, `Agent turn and tool execution events on bus`, `Discord gateway event-bus adapter`, `Discord native slash/thread command registration parity`, `Email allowlist pre-dispatch loop guard` | `tests/gateway/__init__.py`, `tests/gateway/_plugin_adapter_loader.py`, `tests/gateway/conftest.py`, `tests/gateway/feishu_helpers.py`, `tests/gateway/restart_test_helpers.py` |
| `hermes_cli` | `212` | `hermes_cli` | `Agent personalities + enhanced display config`, `Auth state TOCTOU close + redaction default-on parity`, `CLI setup/onboard/help text fidelity matrix`, `Checkpoints CLI (status/list/prune/clear/clear-legacy)`, `Credential non-ASCII sanitizer + one-shot warning` | `tests/hermes_cli/__init__.py`, `tests/hermes_cli/conftest.py`, `tests/hermes_cli/test_ai_gateway_models.py`, `tests/hermes_cli/test_anthropic_model_flow_stale_oauth.py`, `tests/hermes_cli/test_anthropic_oauth_flow.py` |
| `hermes_state` | `1` | `hermes_state` | - | `tests/hermes_state/test_resolve_resume_session_id.py` |
| `honcho_plugin` | `7` | `honcho_plugin` | - | `tests/honcho_plugin/__init__.py`, `tests/honcho_plugin/test_async_memory.py`, `tests/honcho_plugin/test_cli.py`, `tests/honcho_plugin/test_client.py`, `tests/honcho_plugin/test_empty_profile_hint.py` |
| `integration` | `8` | `integration` | - | `tests/integration/__init__.py`, `tests/integration/test_batch_runner.py`, `tests/integration/test_checkpoint_resumption.py`, `tests/integration/test_daytona_terminal.py`, `tests/integration/test_ha_integration.py` |
| `openviking_plugin` | `1` | `openviking_plugin` | - | `tests/openviking_plugin/test_openviking.py` |
| `plugins` | `30` | `plugins` | `Bundled platform plugin manifest drift guard`, `Google Chat install dependency hint refresh`, `Google Chat relay sender-type self-filter`, `Google Chat shared-chassis platform adapter seam`, `Google Chat standalone cron sender` | `tests/plugins/__init__.py`, `tests/plugins/browser/__init__.py`, `tests/plugins/browser/check_parity_vs_main.py`, `tests/plugins/browser/test_browser_provider_plugins.py`, `tests/plugins/image_gen/__init__.py` |
| `providers` | `6` | `providers` | - | `tests/providers/__init__.py`, `tests/providers/test_e2e_wiring.py`, `tests/providers/test_plugin_discovery.py`, `tests/providers/test_profile_wiring.py`, `tests/providers/test_provider_profiles.py` |
| `run_agent` | `91` | `run_agent` | - | `tests/run_agent/__init__.py`, `tests/run_agent/conftest.py`, `tests/run_agent/test_1630_context_overflow_loop.py`, `tests/run_agent/test_413_compression.py`, `tests/run_agent/test_860_dedup.py` |
| `run_interrupt_test.py` | `1` | `run_interrupt_test.py` | - | `tests/run_interrupt_test.py` |
| `scripts` | `1` | `scripts` | - | `tests/scripts/test_release_acp_registry.py` |
| `skills` | `11` | `skills` | `Hermes skill catalog strict-fidelity classifier`, `Sharp v1.0 differentiator decision` | `tests/skills/test_darwinian_evolver_skill.py`, `tests/skills/test_fetch_transcript.py`, `tests/skills/test_google_oauth_setup.py`, `tests/skills/test_google_workspace_api.py`, `tests/skills/test_google_workspace_credential_files.py` |
| `stress` | `11` | `stress` | - | `tests/stress/README.md`, `tests/stress/_fake_worker.py`, `tests/stress/conftest.py`, `tests/stress/test_atypical_scenarios.py`, `tests/stress/test_benchmarks.py` |
| `test_account_usage.py` | `1` | `test_account_usage.py` | - | `tests/test_account_usage.py` |
| `test_atomic_replace_symlinks.py` | `1` | `test_atomic_replace_symlinks.py` | - | `tests/test_atomic_replace_symlinks.py` |
| `test_base_url_hostname.py` | `1` | `test_base_url_hostname.py` | - | `tests/test_base_url_hostname.py` |
| `test_batch_runner_checkpoint.py` | `1` | `test_batch_runner_checkpoint.py` | - | `tests/test_batch_runner_checkpoint.py` |
| `test_cli_file_drop.py` | `1` | `test_cli_file_drop.py` | - | `tests/test_cli_file_drop.py` |
| `test_cli_manual_compress.py` | `1` | `test_cli_manual_compress.py` | - | `tests/test_cli_manual_compress.py` |
| `test_cli_skin_integration.py` | `1` | `test_cli_skin_integration.py` | - | `tests/test_cli_skin_integration.py` |
| `test_ctx_halving_fix.py` | `1` | `test_ctx_halving_fix.py` | - | `tests/test_ctx_halving_fix.py` |
| `test_empty_model_fallback.py` | `1` | `test_empty_model_fallback.py` | - | `tests/test_empty_model_fallback.py` |
| `test_evidence_store.py` | `1` | `test_evidence_store.py` | - | `tests/test_evidence_store.py` |
| `test_gateway_streaming_nested_config.py` | `1` | `test_gateway_streaming_nested_config.py` | - | `tests/test_gateway_streaming_nested_config.py` |
| `test_get_tool_definitions_cache_isolation.py` | `1` | `test_get_tool_definitions_cache_isolation.py` | - | `tests/test_get_tool_definitions_cache_isolation.py` |
| `test_hermes_bootstrap.py` | `1` | `test_hermes_bootstrap.py` | - | `tests/test_hermes_bootstrap.py` |
| `test_hermes_constants.py` | `1` | `test_hermes_constants.py` | - | `tests/test_hermes_constants.py` |
| `test_hermes_home_profile_warning.py` | `1` | `test_hermes_home_profile_warning.py` | - | `tests/test_hermes_home_profile_warning.py` |
| `test_hermes_logging.py` | `1` | `test_hermes_logging.py` | - | `tests/test_hermes_logging.py` |
| `test_hermes_state.py` | `1` | `test_hermes_state.py` | - | `tests/test_hermes_state.py` |
| `test_hermes_state_wal_fallback.py` | `1` | `test_hermes_state_wal_fallback.py` | - | `tests/test_hermes_state_wal_fallback.py` |
| `test_honcho_client_config.py` | `1` | `test_honcho_client_config.py` | - | `tests/test_honcho_client_config.py` |
| `test_install_sh_browser_install.py` | `1` | `test_install_sh_browser_install.py` | - | `tests/test_install_sh_browser_install.py` |
| `test_install_sh_pythonpath_sanitization.py` | `1` | `test_install_sh_pythonpath_sanitization.py` | - | `tests/test_install_sh_pythonpath_sanitization.py` |
| `test_install_sh_setup_wizard_tty_probe.py` | `1` | `test_install_sh_setup_wizard_tty_probe.py` | - | `tests/test_install_sh_setup_wizard_tty_probe.py` |
| `test_install_sh_symlink_stomp.py` | `1` | `test_install_sh_symlink_stomp.py` | - | `tests/test_install_sh_symlink_stomp.py` |
| `test_install_sh_termux_network_prereqs.py` | `1` | `test_install_sh_termux_network_prereqs.py` | - | `tests/test_install_sh_termux_network_prereqs.py` |
| `test_ipv4_preference.py` | `1` | `test_ipv4_preference.py` | - | `tests/test_ipv4_preference.py` |
| `test_lazy_session_regressions.py` | `1` | `test_lazy_session_regressions.py` | - | `tests/test_lazy_session_regressions.py` |
| `test_lint_config.py` | `1` | `test_lint_config.py` | - | `tests/test_lint_config.py` |
| `test_live_system_guard_self_test.py` | `1` | `test_live_system_guard_self_test.py` | - | `tests/test_live_system_guard_self_test.py` |
| `test_mcp_serve.py` | `1` | `test_mcp_serve.py` | - | `tests/test_mcp_serve.py` |
| `test_mini_swe_runner.py` | `1` | `test_mini_swe_runner.py` | - | `tests/test_mini_swe_runner.py` |
| `test_minimax_model_validation.py` | `1` | `test_minimax_model_validation.py` | - | `tests/test_minimax_model_validation.py` |
| `test_minimax_oauth.py` | `1` | `test_minimax_oauth.py` | - | `tests/test_minimax_oauth.py` |
| `test_minisweagent_path.py` | `1` | `test_minisweagent_path.py` | - | `tests/test_minisweagent_path.py` |
| `test_model_picker_scroll.py` | `1` | `test_model_picker_scroll.py` | - | `tests/test_model_picker_scroll.py` |
| `test_model_tools.py` | `1` | `test_model_tools.py` | - | `tests/test_model_tools.py` |
| `test_model_tools_async_bridge.py` | `1` | `test_model_tools_async_bridge.py` | - | `tests/test_model_tools_async_bridge.py` |
| `test_ollama_num_ctx.py` | `1` | `test_ollama_num_ctx.py` | - | `tests/test_ollama_num_ctx.py` |
| `test_package_json_lazy_deps.py` | `1` | `test_package_json_lazy_deps.py` | - | `tests/test_package_json_lazy_deps.py` |
| `test_packaging_metadata.py` | `1` | `test_packaging_metadata.py` | - | `tests/test_packaging_metadata.py` |
| `test_plugin_skills.py` | `1` | `test_plugin_skills.py` | - | `tests/test_plugin_skills.py` |
| `test_process_loop_event_loop_warning.py` | `1` | `test_process_loop_event_loop_warning.py` | - | `tests/test_process_loop_event_loop_warning.py` |
| `test_project_metadata.py` | `1` | `test_project_metadata.py` | - | `tests/test_project_metadata.py` |
| `test_retry_utils.py` | `1` | `test_retry_utils.py` | - | `tests/test_retry_utils.py` |
| `test_sanitize_tool_error.py` | `1` | `test_sanitize_tool_error.py` | - | `tests/test_sanitize_tool_error.py` |
| `test_sql_injection.py` | `1` | `test_sql_injection.py` | - | `tests/test_sql_injection.py` |
| `test_subprocess_home_isolation.py` | `1` | `test_subprocess_home_isolation.py` | - | `tests/test_subprocess_home_isolation.py` |
| `test_termux_all_extra_compat.py` | `1` | `test_termux_all_extra_compat.py` | - | `tests/test_termux_all_extra_compat.py` |
| `test_timezone.py` | `1` | `test_timezone.py` | - | `tests/test_timezone.py` |
| `test_toolset_distributions.py` | `1` | `test_toolset_distributions.py` | - | `tests/test_toolset_distributions.py` |
| `test_toolsets.py` | `1` | `test_toolsets.py` | - | `tests/test_toolsets.py` |
| `test_trajectory_compressor.py` | `1` | `test_trajectory_compressor.py` | - | `tests/test_trajectory_compressor.py` |
| `test_trajectory_compressor_async.py` | `1` | `test_trajectory_compressor_async.py` | - | `tests/test_trajectory_compressor_async.py` |
| `test_transform_llm_output_hook.py` | `1` | `test_transform_llm_output_hook.py` | - | `tests/test_transform_llm_output_hook.py` |
| `test_transform_tool_result_hook.py` | `1` | `test_transform_tool_result_hook.py` | - | `tests/test_transform_tool_result_hook.py` |
| `test_tui_gateway_server.py` | `1` | `test_tui_gateway_server.py` | - | `tests/test_tui_gateway_server.py` |
| `test_utils_truthy_values.py` | `1` | `test_utils_truthy_values.py` | - | `tests/test_utils_truthy_values.py` |
| `test_yuanbao_integration.py` | `1` | `test_yuanbao_integration.py` | - | `tests/test_yuanbao_integration.py` |
| `test_yuanbao_markdown.py` | `1` | `test_yuanbao_markdown.py` | - | `tests/test_yuanbao_markdown.py` |
| `test_yuanbao_pipeline.py` | `1` | `test_yuanbao_pipeline.py` | - | `tests/test_yuanbao_pipeline.py` |
| `test_yuanbao_proto.py` | `1` | `test_yuanbao_proto.py` | - | `tests/test_yuanbao_proto.py` |
| `tools` | `194` | `tools` | `ACP session CWD propagation into prompt runners`, `Audio preprocessing and chunking pipeline`, `Block-anchor fuzzy replace for native patch tool`, `Brave Search + DDGS web search provider parity`, `Browser console expression CDP result shaping` | `tests/tools/__init__.py`, `tests/tools/test_accretion_caps.py`, `tests/tools/test_ansi_strip.py`, `tests/tools/test_approval.py`, `tests/tools/test_approval_heartbeat.py` |
| `tui_gateway` | `7` | `tui_gateway` | `Hermes fast-mode request override serializer`, `Hermes gateway platform strict-fidelity source-pair expansion`, `TUI gateway config health null-section probe`, `TUI websocket attach transport`, `Voice record-key config binding for native TUI` | `tests/tui_gateway/__init__.py`, `tests/tui_gateway/test_entry_sys_path.py`, `tests/tui_gateway/test_goal_command.py`, `tests/tui_gateway/test_make_agent_provider.py`, `tests/tui_gateway/test_protocol.py` |
| `ui-tui` | `50` | `ui-tui` | `CLI setup/onboard/help text fidelity matrix`, `Gormes-owned chat TUI divergence ratification`, `Gormes-owned semantic chat style system`, `Gormes-owned session-aware welcome panel`, `Hermes ui-tui strict-fidelity action matrix` | `ui-tui/src/__tests__/asCommandDispatch.test.ts`, `ui-tui/src/__tests__/clipboard.test.ts`, `ui-tui/src/__tests__/constants.test.ts`, `ui-tui/src/__tests__/createGatewayEventHandler.test.ts`, `ui-tui/src/__tests__/createSlashHandler.test.ts` |
| `website` | `2` | `website` | `Google Chat install dependency hint refresh`, `Gormes profile distribution metadata readout`, `Gormes setup/channel/provider docs webpage parity gate`, `Hermes config migration dry-run manifest`, `Hermes website docs mirror coverage gate` | `tests/website/__init__.py`, `tests/website/test_generate_skill_docs.py` |

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
| `profiles` | `covered` | `none` | `high` | `Active Hermes/Sidon profile context root resolver for live turns`, `Agent personalities + enhanced display config`, `Agent personalities + enhanced display config`, `CLI active-profile store`, `+35 more` | `agent/lsp/manager.py`, `agent/prompt_builder.py`, `hermes_cli/config.py`, `hermes_cli/default_soul.py`, `+2 more` | Covered source-pair evidence joins to a validated complete progress row with test evidence. |
| `sessions` | `covered` | `none` | `high` | `Auto-naming sessions`, `Aux compression single-prompt threshold reconciliation`, `Behavioral pattern extraction from session logs`, `Busy command guard for compression and long CLI actions`, `+31 more` | `agent/context_engine.py`, `agent/prompt_builder.py`, `agent/transports/chat_completions.py`, `agent/transports/codex.py`, `+2 more` | Covered source-pair evidence joins to a validated complete progress row with test evidence. |
| `goncho_memory` | `partial` | `warning` | `medium` | `2-layer seed selection`, `6 typed memory categories with confidence scoring`, `<memory-context> fence`, `Agent-controlled memory retention with importance scoring`, `+74 more` | `agent/prompt_builder.py` | Some source-pair or complete-row evidence exists, but the surface is not strictly covered. |
| `learning_loop` | `partial` | `warning` | `medium` | `Agent-controlled memory retention with importance scoring`, `Code Cathedral II code-context retrieval fixtures`, `Goncho durable recall trace IR + fused ranking pipeline`, `Goncho golden transcript e2e harness`, `+19 more` | `agent/lsp/manager.py`, `tools/skill_manager_tool.py` | Some source-pair or complete-row evidence exists, but the surface is not strictly covered. |
| `prompt_assembly` | `covered` | `none` | `high` | `<memory-context> fence`, `Agent lifecycle hooks (agent:start, agent:step, agent:end)`, `Aux compression single-prompt threshold reconciliation`, `Bundled Airtable productivity skill contract`, `+39 more` | `agent/context_engine.py`, `agent/conversation_loop.py`, `agent/prompt_builder.py`, `agent/tool_executor.py`, `+3 more` | Covered source-pair evidence joins to a validated complete progress row with test evidence. |
| `provider_auth_setup` | `covered` | `none` | `high` | `Anthropic`, `Anthropic OAuth/keychain credential discovery`, `Aux compression provider-aware context cap`, `Azure Anthropic Messages endpoint contract`, `+155 more` | `agent/context_engine.py`, `agent/prompt_builder.py`, `agent/transports/chat_completions.py`, `agent/transports/codex.py`, `+9 more` | Covered source-pair evidence joins to a validated complete progress row with test evidence. |
| `gateway_channels` | `partial` | `warning` | `medium` | `(platform, chat_id) -> session_id`, `API server cron admin mutating endpoints`, `API server cron admin read-only endpoints`, `API server detailed health endpoint`, `+291 more` | `agent/prompt_builder.py`, `gateway/run.py` | Some source-pair or complete-row evidence exists, but the surface is not strictly covered. |
| `tool_runtime` | `covered` | `none` | `high` | `61-tool registry port`, `ACP Client Bridge Mode`, `ACP JSON-RPC stdio session/prompt closeout`, `ACP server side`, `+150 more` | `agent/conversation_loop.py`, `agent/lsp/manager.py`, `agent/tool_executor.py`, `agent/transports/chat_completions.py`, `+13 more` | Covered source-pair evidence joins to a validated complete progress row with test evidence. |
| `mcp_acp` | `partial` | `warning` | `medium` | `ACP Client Bridge Mode`, `ACP JSON-RPC stdio session/prompt closeout`, `ACP server side`, `ACP session CWD propagation into prompt runners`, `+26 more` | `tools/web_tools.py` | Some source-pair or complete-row evidence exists, but the surface is not strictly covered. |
| `tui_cli` | `covered` | `none` | `high` | `16ms coalescing mailbox`, `Admin TUI: Agents screen wired to the 2.H dynamic registry`, `Admin TUI: Chat tab with keybinding to jump in from any screen`, `Admin TUI: Commands catalog over the root CLI tree`, `+59 more` | `agent/transports/chat_completions.py`, `agent/transports/codex.py`, `cli.py`, `gateway/run.py`, `+7 more` | Covered source-pair evidence joins to a validated complete progress row with test evidence. |
