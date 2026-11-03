# Hermes Contract Inventory

- Hermes SHA: `197f16a12b098f1ea541bbd0a5ddd7d66f2d2dee`
- Generated: `2026-11-02T16:16:44Z`
- Source pairs: `current` (`197f16a12b098f1ea541bbd0a5ddd7d66f2d2dee`)
- Report mode: `report-only`
- Progress source: `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Backlog policy: `progress.json` remains the only backlog; this report classifies evidence and does not create work rows.
- Claim boundary: Gormes may claim all Hermes features and architecture are paired only when every current-SHA inventory gap is classified as `covered`, `partial`, `planned`, `excluded`, or `owned_divergence`; strict mode additionally requires every critical surface to be `covered`, `excluded`, or `owned_divergence` and every upstream source/doc/test file to be mapped or explicitly excluded.

## Headline Counts

- Source files: `15958`
- Docs files: `1569`
- Test files: `9995`
- Unmapped upstream source files: `5772`
- Unmapped upstream docs files: `1554`
- Unmapped upstream test files: `9873`
- Release checkpoints: `3`
- Critical surfaces: `10`
- Surface strict failures: `10`
- Strict failures: `17209`
- `partial`: `10`

## Critical Surface Blockers

No critical surface blockers in the current classification. `17199` unmapped upstream source/doc/test files still block strict mode.

## Release Checkpoints

| Checkpoint | Present | Path |
|---|---|---|
| Gormes Hermes v0.14 module pairings | `true` | `webpages/docs/content/building-gormes/architecture_plan/hermes-v0.14-module-pairings.md` |
| Gormes Hermes source-pair manifest | `true` | `webpages/docs/content/building-gormes/architecture_plan/hermes-source-pairs.json` |
| Gormes Hermes source-pair report | `true` | `webpages/docs/content/building-gormes/architecture_plan/hermes-source-pairs.md` |

## Per-Module Gap Summary

| Module | Surfaces | Covered | Partial | Planned | Missing | Blocker severity |
|---|---|---:|---:|---:|---:|---|
| `channels` | `gateway_channels` | `0` | `1` | `0` | `0` | `warning` |
| `continuity` | `profiles`, `sessions`, `goncho_memory`, `learning_loop`, `prompt_assembly` | `0` | `5` | `0` | `0` | `warning` |
| `operator` | `provider_auth_setup`, `tui_cli` | `0` | `2` | `0` | `0` | `warning` |
| `runtime` | `tool_runtime` | `0` | `1` | `0` | `0` | `warning` |
| `tools` | `mcp_acp` | `0` | `1` | `0` | `0` | `warning` |

## Continuity Categories

| Category | Status | Severity | Surfaces | Evidence | Reason |
|---|---|---|---|---|---|
| `sessions` | `partial` | `warning` | `sessions` | `../hermes-agent/agent/context_compressor.py@5401a008`, `../hermes-agent/agent/context_compressor.py@94346523:_find_tail_cut_by_tokens`, `../hermes-agent/agent/context_compressor.py@bda2dbc2:_calculate_protect_tail_boundary`, `../hermes-agent/agent/context_compressor.py@cfc8befe:_find_tail_cut_by_tokens`, `+156 more` | Mapped through surfaces: sessions; worst status is sessions=partial. |
| `memory_goncho_honcho_compatibility` | `partial` | `warning` | `goncho_memory` | `../hermes-agent/acp_adapter/`, `../hermes-agent/agent/`, `../hermes-agent/agent/memory_manager.py`, `../hermes-agent/agent/memory_manager.py:178`, `+289 more` | Mapped through surfaces: goncho_memory; worst status is goncho_memory=partial. |
| `workspace_peer_profile_identity_boundaries` | `partial` | `warning` | `profiles` | `../hermes-agent/agent/lsp/manager.py`, `../hermes-agent/agent/lsp/range_shift.py`, `../hermes-agent/agent/prompt_builder.py`, `../hermes-agent/agent/prompt_builder.py:32-73`, `+228 more` | Mapped through surfaces: profiles; worst status is profiles=partial. |
| `context_retrieval_and_prompt_budget` | `partial` | `warning` | `sessions`, `prompt_assembly` | `../agent-zero/agent.py@7c71185f:agent_init,monologue_start,message_loop_start,before_main_llm_call,response_stream_chunk,reasoning_stream_chunk,message_loop_end,monologue_end`, `../agent-zero/helpers/extension.py@7c71185f:extensible,call_extensions_async,call_extensions_sync`, `../hermes-agent/agent/`, `../hermes-agent/agent/context_compressor.py@5401a008`, `+312 more` | Mapped through surfaces: sessions, prompt_assembly; worst status is sessions=partial. |
| `summaries_conclusions_search` | `partial` | `warning` | `sessions` | `../hermes-agent/agent/context_compressor.py@5401a008`, `../hermes-agent/agent/context_compressor.py@94346523:_find_tail_cut_by_tokens`, `../hermes-agent/agent/context_compressor.py@bda2dbc2:_calculate_protect_tail_boundary`, `../hermes-agent/agent/context_compressor.py@cfc8befe:_find_tail_cut_by_tokens`, `+156 more` | Mapped through surfaces: sessions; worst status is sessions=partial. |
| `skill_templates_and_skills_ux` | `partial` | `warning` | `learning_loop`, `prompt_assembly` | `../agent-zero/agent.py@7c71185f:agent_init,monologue_start,message_loop_start,before_main_llm_call,response_stream_chunk,reasoning_stream_chunk,message_loop_end,monologue_end`, `../agent-zero/helpers/extension.py@7c71185f:extensible,call_extensions_async,call_extensions_sync`, `../hermes-agent/agent/`, `../hermes-agent/agent/context_compressor.py@5401a008`, `+323 more` | Mapped through surfaces: learning_loop, prompt_assembly; worst status is learning_loop=partial. |
| `skill_precedence_sync_update_reset` | `partial` | `warning` | `learning_loop`, `prompt_assembly` | `../agent-zero/agent.py@7c71185f:agent_init,monologue_start,message_loop_start,before_main_llm_call,response_stream_chunk,reasoning_stream_chunk,message_loop_end,monologue_end`, `../agent-zero/helpers/extension.py@7c71185f:extensible,call_extensions_async,call_extensions_sync`, `../hermes-agent/agent/`, `../hermes-agent/agent/context_compressor.py@5401a008`, `+323 more` | Mapped through surfaces: learning_loop, prompt_assembly; worst status is learning_loop=partial. |
| `learning_loop_curator_behavior` | `partial` | `warning` | `learning_loop` | `../hermes-agent/agent/lsp/manager.py`, `../hermes-agent/agent/lsp/range_shift.py`, `../hermes-agent/hermes_cli/skills_hub.py@e63929d4:_is_valid_installed_skill_name`, `../hermes-agent/hermes_cli/skills_hub.py@e63929d4:do_install,_prompt_for_skill_name,_prompt_for_category,_existing_categories`, `+120 more` | Mapped through surfaces: learning_loop; worst status is learning_loop=partial. |
| `candidate_memory_skill_updates` | `partial` | `warning` | `learning_loop`, `goncho_memory` | `../hermes-agent/acp_adapter/`, `../hermes-agent/agent/`, `../hermes-agent/agent/lsp/manager.py`, `../hermes-agent/agent/lsp/range_shift.py`, `+363 more` | Mapped through surfaces: learning_loop, goncho_memory; worst status is learning_loop=partial. |
| `feedback_outcome_scoring` | `partial` | `warning` | `learning_loop` | `../hermes-agent/agent/lsp/manager.py`, `../hermes-agent/agent/lsp/range_shift.py`, `../hermes-agent/hermes_cli/skills_hub.py@e63929d4:_is_valid_installed_skill_name`, `../hermes-agent/hermes_cli/skills_hub.py@e63929d4:do_install,_prompt_for_skill_name,_prompt_for_category,_existing_categories`, `+120 more` | Mapped through surfaces: learning_loop; worst status is learning_loop=partial. |
| `audit_trail` | `partial` | `warning` | `sessions`, `tool_runtime`, `learning_loop` | `../agent-zero@7c71185f/agent.py:prepare_prompt,read_prompt,parse_prompt`, `../agent-zero@7c71185f/helpers/files.py:read_prompt_file,process_includes,_get_dirs_after`, `../agent-zero@7c71185f/prompts/ (72 fragment files)`, `../agent-zero@7c71185f/skills/a0-development/SKILL.md:Prompt System`, `+948 more` | Mapped through surfaces: sessions, tool_runtime, learning_loop; worst status is sessions=partial. |
| `mutation_safety` | `partial` | `warning` | `tool_runtime`, `learning_loop`, `goncho_memory` | `../agent-zero@7c71185f/agent.py:prepare_prompt,read_prompt,parse_prompt`, `../agent-zero@7c71185f/helpers/files.py:read_prompt_file,process_includes,_get_dirs_after`, `../agent-zero@7c71185f/prompts/ (72 fragment files)`, `../agent-zero@7c71185f/skills/a0-development/SKILL.md:Prompt System`, `+1078 more` | Mapped through surfaces: tool_runtime, learning_loop, goncho_memory; worst status is tool_runtime=partial. |
| `prompt_context_memory_skill_insertion_ordering` | `partial` | `warning` | `prompt_assembly`, `goncho_memory`, `sessions`, `learning_loop` | `../agent-zero/agent.py@7c71185f:agent_init,monologue_start,message_loop_start,before_main_llm_call,response_stream_chunk,reasoning_stream_chunk,message_loop_end,monologue_end`, `../agent-zero/helpers/extension.py@7c71185f:extensible,call_extensions_async,call_extensions_sync`, `../hermes-agent/acp_adapter/`, `../hermes-agent/agent/`, `+600 more` | Mapped through surfaces: prompt_assembly, goncho_memory, sessions, learning_loop; worst status is prompt_assembly=partial. |
| `profile_scoped_isolation` | `partial` | `warning` | `profiles`, `sessions`, `gateway_channels` | `../hermes-agent/agent/account_usage.py:render_account_usage_lines,fetch_account_usage`, `../hermes-agent/agent/bedrock_adapter.py:651:normalize_converse_stream_events`, `../hermes-agent/agent/bedrock_adapter.py:673:stream_converse_with_callbacks`, `../hermes-agent/agent/context_compressor.py@5401a008`, `+1408 more` | Mapped through surfaces: profiles, sessions, gateway_channels; worst status is profiles=partial. |

## Unmapped Upstream Evidence

Unmapped upstream files are strict-mode blockers until each file is joined to a progress row, a source-pair entry, a surface classification, or an explicit exclusion.

| Family | Count | Examples |
|---|---:|---|
| Source | `5772` | `acp_adapter/__init__.py`, `acp_adapter/__main__.py`, `acp_adapter/auth.py`, `acp_adapter/catalog.py`, `acp_adapter/commands.py`, `acp_adapter/content.py`, `acp_adapter/edit_approval.py`, `acp_adapter/elicitation.py`, `acp_adapter/events.py`, `acp_adapter/gateway_commands.py`, `+5762 more` |
| Docs | `1554` | `.github/PULL_REQUEST_TEMPLATE.md`, `.github/actions/setup-pm/README.md`, `CONTRIBUTING.es.md`, `CONTRIBUTING.md`, `README.es.md`, `README.ur-pk.md`, `README.zh-CN.md`, `SECURITY.es.md`, `SECURITY.md`, `SOUL.md`, `+1544 more` |
| Tests | `9873` | `apps/desktop/e2e/at-rest-connection-token.spec.ts`, `apps/desktop/e2e/batch-clarify.spec.ts`, `apps/desktop/e2e/boot-failure.spec.ts`, `apps/desktop/e2e/boot.spec.ts`, `apps/desktop/e2e/bot-screen-stale-status-reply.spec.ts`, `apps/desktop/e2e/chat.spec.ts`, `apps/desktop/e2e/completed-reply-refresh.spec.ts`, `apps/desktop/e2e/connection-version-label.spec.ts`, `apps/desktop/e2e/core/behind-window-send.spec.ts`, `apps/desktop/e2e/core/boot-lifecycle.spec.ts`, `+9863 more` |

## Unmapped Test Suite Classification

| Suite | Count | Source under test | Progress rows | Examples |
|---|---:|---|---|---|
| `AGENTS.md` | `1` | `AGENTS.md` | - | `tests/AGENTS.md` |
| `__init__.py` | `1` | `__init__.py` | - | `tests/__init__.py` |
| `_fixtures` | `5` | `_fixtures` | - | `tests/_fixtures/__init__.py`, `tests/_fixtures/env_filter.py`, `tests/_fixtures/live_system_guard.py`, `tests/_fixtures/model_peer.py`, `tests/_fixtures/platform_gating.py` |
| `acp_adapter` | `92` | `acp_adapter` | `ACP session CWD propagation into prompt runners`, `ACP stdio benign ping/probe suppression` | `tests/acp_adapter/__init__.py`, `tests/acp_adapter/conftest.py`, `tests/acp_adapter/test_acp_commands.py`, `tests/acp_adapter/test_acp_dashboard_model_switch_validation.py`, `tests/acp_adapter/test_acp_discard_review.py` |
| `agent/__init__.py` | `1` | `agent/__init__.py` | - | `tests/agent/__init__.py` |
| `agent/conftest.py` | `1` | `agent/conftest.py` | - | `tests/agent/conftest.py` |
| `agent/lsp` | `28` | `agent/lsp` | `Hermes agent runtime strict-fidelity source-pair expansion` | `tests/agent/lsp/__init__.py`, `tests/agent/lsp/_mock_lsp_server.py`, `tests/agent/lsp/test_backend_gate.py`, `tests/agent/lsp/test_broken_set.py`, `tests/agent/lsp/test_client_e2e.py` |
| `agent/metadata_transport.py` | `1` | `agent/metadata_transport.py` | - | `tests/agent/metadata_transport.py` |
| `agent/repro_image_shrink_brick.py` | `1` | `agent/repro_image_shrink_brick.py` | - | `tests/agent/repro_image_shrink_brick.py` |
| `agent/test_413_compression.py` | `1` | `agent/test_413_compression.py` | - | `tests/agent/test_413_compression.py` |
| `agent/test_413_image_payload_recovery.py` | `1` | `agent/test_413_image_payload_recovery.py` | - | `tests/agent/test_413_image_payload_recovery.py` |
| `agent/test_860_dedup.py` | `1` | `agent/test_860_dedup.py` | - | `tests/agent/test_860_dedup.py` |
| `agent/test_account_policy_block.py` | `1` | `agent/test_account_policy_block.py` | - | `tests/agent/test_account_policy_block.py` |
| `agent/test_account_usage.py` | `1` | `agent/test_account_usage.py` | - | `tests/agent/test_account_usage.py` |
| `agent/test_account_usage_fetch.py` | `1` | `agent/test_account_usage_fetch.py` | - | `tests/agent/test_account_usage_fetch.py` |
| `agent/test_account_usage_pool_fetch.py` | `1` | `agent/test_account_usage_pool_fetch.py` | - | `tests/agent/test_account_usage_pool_fetch.py` |
| `agent/test_acp_client.py` | `1` | `agent/test_acp_client.py` | - | `tests/agent/test_acp_client.py` |
| `agent/test_acp_client_invariants.py` | `1` | `agent/test_acp_client_invariants.py` | - | `tests/agent/test_acp_client_invariants.py` |
| `agent/test_acp_openai_bridge.py` | `1` | `agent/test_acp_openai_bridge.py` | - | `tests/agent/test_acp_openai_bridge.py` |
| `agent/test_acp_provider_rails.py` | `1` | `agent/test_acp_provider_rails.py` | - | `tests/agent/test_acp_provider_rails.py` |
| `agent/test_activity_persist_off_relay_loop.py` | `1` | `agent/test_activity_persist_off_relay_loop.py` | - | `tests/agent/test_activity_persist_off_relay_loop.py` |
| `agent/test_actual_auxiliary_routing.py` | `1` | `agent/test_actual_auxiliary_routing.py` | - | `tests/agent/test_actual_auxiliary_routing.py` |
| `agent/test_affordable_402_main_loop.py` | `1` | `agent/test_affordable_402_main_loop.py` | - | `tests/agent/test_affordable_402_main_loop.py` |
| `agent/test_agent_guardrails.py` | `1` | `agent/test_agent_guardrails.py` | - | `tests/agent/test_agent_guardrails.py` |
| `agent/test_anthropic_adapter.py` | `1` | `agent/test_anthropic_adapter.py` | - | `tests/agent/test_anthropic_adapter.py` |
| `agent/test_anthropic_adapter_extra_headers.py` | `1` | `agent/test_anthropic_adapter_extra_headers.py` | - | `tests/agent/test_anthropic_adapter_extra_headers.py` |
| `agent/test_anthropic_billing_guidance.py` | `1` | `agent/test_anthropic_billing_guidance.py` | - | `tests/agent/test_anthropic_billing_guidance.py` |
| `agent/test_anthropic_borrowed_row_authority.py` | `1` | `agent/test_anthropic_borrowed_row_authority.py` | - | `tests/agent/test_anthropic_borrowed_row_authority.py` |
| `agent/test_anthropic_client_auth.py` | `1` | `agent/test_anthropic_client_auth.py` | - | `tests/agent/test_anthropic_client_auth.py` |
| `agent/test_anthropic_credential_persist_failure.py` | `1` | `agent/test_anthropic_credential_persist_failure.py` | - | `tests/agent/test_anthropic_credential_persist_failure.py` |
| `agent/test_anthropic_dead_grant_hints.py` | `1` | `agent/test_anthropic_dead_grant_hints.py` | - | `tests/agent/test_anthropic_dead_grant_hints.py` |
| `agent/test_anthropic_external_login_optout.py` | `1` | `agent/test_anthropic_external_login_optout.py` | - | `tests/agent/test_anthropic_external_login_optout.py` |
| `agent/test_anthropic_keychain.py` | `1` | `agent/test_anthropic_keychain.py` | - | `tests/agent/test_anthropic_keychain.py` |
| `agent/test_anthropic_kimi_signed_thinking_replay.py` | `1` | `agent/test_anthropic_kimi_signed_thinking_replay.py` | - | `tests/agent/test_anthropic_kimi_signed_thinking_replay.py` |
| `agent/test_anthropic_kwargs_sanitize.py` | `1` | `agent/test_anthropic_kwargs_sanitize.py` | - | `tests/agent/test_anthropic_kwargs_sanitize.py` |
| `agent/test_anthropic_mcp_prefix_strip.py` | `1` | `agent/test_anthropic_mcp_prefix_strip.py` | - | `tests/agent/test_anthropic_mcp_prefix_strip.py` |
| `agent/test_anthropic_message_stop.py` | `1` | `agent/test_anthropic_message_stop.py` | - | `tests/agent/test_anthropic_message_stop.py` |
| `agent/test_anthropic_mid_tool_call_drop.py` | `1` | `agent/test_anthropic_mid_tool_call_drop.py` | - | `tests/agent/test_anthropic_mid_tool_call_drop.py` |
| `agent/test_anthropic_model_cooldowns.py` | `1` | `agent/test_anthropic_model_cooldowns.py` | - | `tests/agent/test_anthropic_model_cooldowns.py` |
| `agent/test_anthropic_oauth_pkce.py` | `1` | `agent/test_anthropic_oauth_pkce.py` | - | `tests/agent/test_anthropic_oauth_pkce.py` |
| `agent/test_anthropic_oauth_stress.py` | `1` | `agent/test_anthropic_oauth_stress.py` | - | `tests/agent/test_anthropic_oauth_stress.py` |
| `agent/test_anthropic_oauth_ua_prefix.py` | `1` | `agent/test_anthropic_oauth_ua_prefix.py` | - | `tests/agent/test_anthropic_oauth_ua_prefix.py` |
| `agent/test_anthropic_output_field_leak.py` | `1` | `agent/test_anthropic_output_field_leak.py` | - | `tests/agent/test_anthropic_output_field_leak.py` |
| `agent/test_anthropic_owned_routing.py` | `1` | `agent/test_anthropic_owned_routing.py` | - | `tests/agent/test_anthropic_owned_routing.py` |
| `agent/test_anthropic_preserved_thinking_replay.py` | `1` | `agent/test_anthropic_preserved_thinking_replay.py` | - | `tests/agent/test_anthropic_preserved_thinking_replay.py` |
| `agent/test_anthropic_prompt_cache_policy.py` | `1` | `agent/test_anthropic_prompt_cache_policy.py` | - | `tests/agent/test_anthropic_prompt_cache_policy.py` |
| `agent/test_anthropic_request_blank_block_guard.py` | `1` | `agent/test_anthropic_request_blank_block_guard.py` | - | `tests/agent/test_anthropic_request_blank_block_guard.py` |
| `agent/test_anthropic_request_client_reuse.py` | `1` | `agent/test_anthropic_request_client_reuse.py` | - | `tests/agent/test_anthropic_request_client_reuse.py` |
| `agent/test_anthropic_route_oauth_identity.py` | `1` | `agent/test_anthropic_route_oauth_identity.py` | - | `tests/agent/test_anthropic_route_oauth_identity.py` |
| `agent/test_anthropic_signed_platform_replay.py` | `1` | `agent/test_anthropic_signed_platform_replay.py` | - | `tests/agent/test_anthropic_signed_platform_replay.py` |
| `agent/test_anthropic_spent_rotation_verdict.py` | `1` | `agent/test_anthropic_spent_rotation_verdict.py` | - | `tests/agent/test_anthropic_spent_rotation_verdict.py` |
| `agent/test_anthropic_stream_fallbacks.py` | `1` | `agent/test_anthropic_stream_fallbacks.py` | - | `tests/agent/test_anthropic_stream_fallbacks.py` |
| `agent/test_anthropic_structured_output.py` | `1` | `agent/test_anthropic_structured_output.py` | - | `tests/agent/test_anthropic_structured_output.py` |
| `agent/test_anthropic_thinking_block_order.py` | `1` | `agent/test_anthropic_thinking_block_order.py` | - | `tests/agent/test_anthropic_thinking_block_order.py` |
| `agent/test_anthropic_thinking_disable.py` | `1` | `agent/test_anthropic_thinking_disable.py` | - | `tests/agent/test_anthropic_thinking_disable.py` |
| `agent/test_anthropic_third_party_oauth_guard.py` | `1` | `agent/test_anthropic_third_party_oauth_guard.py` | - | `tests/agent/test_anthropic_third_party_oauth_guard.py` |
| `agent/test_anthropic_token_scope_isolation.py` | `1` | `agent/test_anthropic_token_scope_isolation.py` | - | `tests/agent/test_anthropic_token_scope_isolation.py` |
| `agent/test_anthropic_truncation_continuation.py` | `1` | `agent/test_anthropic_truncation_continuation.py` | - | `tests/agent/test_anthropic_truncation_continuation.py` |
| `agent/test_anthropic_whitespace_text_blocks.py` | `1` | `agent/test_anthropic_whitespace_text_blocks.py` | - | `tests/agent/test_anthropic_whitespace_text_blocks.py` |
| `agent/test_api_content_row_addressed_backfill.py` | `1` | `agent/test_api_content_row_addressed_backfill.py` | - | `tests/agent/test_api_content_row_addressed_backfill.py` |
| `agent/test_api_content_sidecar.py` | `1` | `agent/test_api_content_sidecar.py` | - | `tests/agent/test_api_content_sidecar.py` |
| `agent/test_api_max_retries_config.py` | `1` | `agent/test_api_max_retries_config.py` | - | `tests/agent/test_api_max_retries_config.py` |
| `agent/test_approval_display_sanitize.py` | `1` | `agent/test_approval_display_sanitize.py` | - | `tests/agent/test_approval_display_sanitize.py` |
| `agent/test_arcee_trinity_overrides.py` | `1` | `agent/test_arcee_trinity_overrides.py` | - | `tests/agent/test_arcee_trinity_overrides.py` |
| `agent/test_astra_baseline_runtime.py` | `1` | `agent/test_astra_baseline_runtime.py` | - | `tests/agent/test_astra_baseline_runtime.py` |
| `agent/test_astra_oauth_native_compaction.py` | `1` | `agent/test_astra_oauth_native_compaction.py` | - | `tests/agent/test_astra_oauth_native_compaction.py` |
| `agent/test_async_httpx_del_neuter.py` | `1` | `agent/test_async_httpx_del_neuter.py` | - | `tests/agent/test_async_httpx_del_neuter.py` |
| `agent/test_async_token_accounting.py` | `1` | `agent/test_async_token_accounting.py` | - | `tests/agent/test_async_token_accounting.py` |
| `agent/test_async_utils.py` | `1` | `agent/test_async_utils.py` | - | `tests/agent/test_async_utils.py` |
| `agent/test_auth_provider_failover.py` | `1` | `agent/test_auth_provider_failover.py` | - | `tests/agent/test_auth_provider_failover.py` |
| `agent/test_authorization_gate.py` | `1` | `agent/test_authorization_gate.py` | - | `tests/agent/test_authorization_gate.py` |
| `agent/test_aux_affordable_402_retry.py` | `1` | `agent/test_aux_affordable_402_retry.py` | - | `tests/agent/test_aux_affordable_402_retry.py` |
| `agent/test_aux_alias_home_scope.py` | `1` | `agent/test_aux_alias_home_scope.py` | - | `tests/agent/test_aux_alias_home_scope.py` |
| `agent/test_aux_custom_endpoint_headers.py` | `1` | `agent/test_aux_custom_endpoint_headers.py` | - | `tests/agent/test_aux_custom_endpoint_headers.py` |
| `agent/test_aux_progress_streaming.py` | `1` | `agent/test_aux_progress_streaming.py` | - | `tests/agent/test_aux_progress_streaming.py` |
| `agent/test_aux_relay_progress_seam.py` | `1` | `agent/test_aux_relay_progress_seam.py` | - | `tests/agent/test_aux_relay_progress_seam.py` |
| `agent/test_aux_runtime_main_context_only.py` | `1` | `agent/test_aux_runtime_main_context_only.py` | - | `tests/agent/test_aux_runtime_main_context_only.py` |
| `agent/test_aux_session_endpoint_affinity.py` | `1` | `agent/test_aux_session_endpoint_affinity.py` | - | `tests/agent/test_aux_session_endpoint_affinity.py` |
| `agent/test_aux_stream_host_deadline.py` | `1` | `agent/test_aux_stream_host_deadline.py` | - | `tests/agent/test_aux_stream_host_deadline.py` |
| `agent/test_aux_stream_host_deadline_sibling_wires.py` | `1` | `agent/test_aux_stream_host_deadline_sibling_wires.py` | - | `tests/agent/test_aux_stream_host_deadline_sibling_wires.py` |
| `agent/test_aux_summary_stream_hard_cancel.py` | `1` | `agent/test_aux_summary_stream_hard_cancel.py` | - | `tests/agent/test_aux_summary_stream_hard_cancel.py` |
| `agent/test_auxiliary_anthropic_pool_fallback_regression.py` | `1` | `agent/test_auxiliary_anthropic_pool_fallback_regression.py` | - | `tests/agent/test_auxiliary_anthropic_pool_fallback_regression.py` |
| `agent/test_auxiliary_auth_rung_fallthrough.py` | `1` | `agent/test_auxiliary_auth_rung_fallthrough.py` | - | `tests/agent/test_auxiliary_auth_rung_fallthrough.py` |
| `agent/test_auxiliary_auto_never_guesses_provider.py` | `1` | `agent/test_auxiliary_auto_never_guesses_provider.py` | - | `tests/agent/test_auxiliary_auto_never_guesses_provider.py` |
| `agent/test_auxiliary_auto_route_xai_oauth_refresh.py` | `1` | `agent/test_auxiliary_auto_route_xai_oauth_refresh.py` | - | `tests/agent/test_auxiliary_auto_route_xai_oauth_refresh.py` |
| `agent/test_auxiliary_client.py` | `1` | `agent/test_auxiliary_client.py` | - | `tests/agent/test_auxiliary_client.py` |
| `agent/test_auxiliary_client_anthropic_custom.py` | `1` | `agent/test_auxiliary_client_anthropic_custom.py` | - | `tests/agent/test_auxiliary_client_anthropic_custom.py` |
| `agent/test_auxiliary_client_azure_foundry.py` | `1` | `agent/test_auxiliary_client_azure_foundry.py` | - | `tests/agent/test_auxiliary_client_azure_foundry.py` |
| `agent/test_auxiliary_client_base_url_host_validation.py` | `1` | `agent/test_auxiliary_client_base_url_host_validation.py` | - | `tests/agent/test_auxiliary_client_base_url_host_validation.py` |
| `agent/test_auxiliary_client_bootstrap_skew.py` | `1` | `agent/test_auxiliary_client_bootstrap_skew.py` | - | `tests/agent/test_auxiliary_client_bootstrap_skew.py` |
| `agent/test_auxiliary_client_cache_eviction.py` | `1` | `agent/test_auxiliary_client_cache_eviction.py` | - | `tests/agent/test_auxiliary_client_cache_eviction.py` |
| `agent/test_auxiliary_client_cache_follows_credential.py` | `1` | `agent/test_auxiliary_client_cache_follows_credential.py` | - | `tests/agent/test_auxiliary_client_cache_follows_credential.py` |
| `agent/test_auxiliary_client_minimax_oauth.py` | `1` | `agent/test_auxiliary_client_minimax_oauth.py` | - | `tests/agent/test_auxiliary_client_minimax_oauth.py` |
| `agent/test_auxiliary_client_nous_401_cache_key.py` | `1` | `agent/test_auxiliary_client_nous_401_cache_key.py` | - | `tests/agent/test_auxiliary_client_nous_401_cache_key.py` |
| `agent/test_auxiliary_client_proxy_env.py` | `1` | `agent/test_auxiliary_client_proxy_env.py` | - | `tests/agent/test_auxiliary_client_proxy_env.py` |
| `agent/test_auxiliary_client_resolve_dedup.py` | `1` | `agent/test_auxiliary_client_resolve_dedup.py` | - | `tests/agent/test_auxiliary_client_resolve_dedup.py` |
| `agent/test_auxiliary_client_ssl_verify.py` | `1` | `agent/test_auxiliary_client_ssl_verify.py` | - | `tests/agent/test_auxiliary_client_ssl_verify.py` |
| `agent/test_auxiliary_client_xai_oauth_recovery.py` | `1` | `agent/test_auxiliary_client_xai_oauth_recovery.py` | - | `tests/agent/test_auxiliary_client_xai_oauth_recovery.py` |
| `agent/test_auxiliary_codex_completion_status.py` | `1` | `agent/test_auxiliary_codex_completion_status.py` | - | `tests/agent/test_auxiliary_codex_completion_status.py` |
| `agent/test_auxiliary_compression_timeout_floor.py` | `1` | `agent/test_auxiliary_compression_timeout_floor.py` | - | `tests/agent/test_auxiliary_compression_timeout_floor.py` |
| `agent/test_auxiliary_concurrency.py` | `1` | `agent/test_auxiliary_concurrency.py` | - | `tests/agent/test_auxiliary_concurrency.py` |
| `agent/test_auxiliary_error_classification.py` | `1` | `agent/test_auxiliary_error_classification.py` | - | `tests/agent/test_auxiliary_error_classification.py` |
| `agent/test_auxiliary_explicit_base_anthropic.py` | `1` | `agent/test_auxiliary_explicit_base_anthropic.py` | - | `tests/agent/test_auxiliary_explicit_base_anthropic.py` |
| `agent/test_auxiliary_explicit_cancellation.py` | `1` | `agent/test_auxiliary_explicit_cancellation.py` | - | `tests/agent/test_auxiliary_explicit_cancellation.py` |
| `agent/test_auxiliary_fallback_multihop.py` | `1` | `agent/test_auxiliary_fallback_multihop.py` | - | `tests/agent/test_auxiliary_fallback_multihop.py` |
| `agent/test_auxiliary_hooks.py` | `1` | `agent/test_auxiliary_hooks.py` | - | `tests/agent/test_auxiliary_hooks.py` |
| `agent/test_auxiliary_main_first.py` | `1` | `agent/test_auxiliary_main_first.py` | - | `tests/agent/test_auxiliary_main_first.py` |
| `agent/test_auxiliary_named_custom_providers.py` | `1` | `agent/test_auxiliary_named_custom_providers.py` | - | `tests/agent/test_auxiliary_named_custom_providers.py` |
| `agent/test_auxiliary_ollama_local_provider.py` | `1` | `agent/test_auxiliary_ollama_local_provider.py` | - | `tests/agent/test_auxiliary_ollama_local_provider.py` |
| `agent/test_auxiliary_opencode_routing.py` | `1` | `agent/test_auxiliary_opencode_routing.py` | - | `tests/agent/test_auxiliary_opencode_routing.py` |
| `agent/test_auxiliary_openrouter_max_tokens.py` | `1` | `agent/test_auxiliary_openrouter_max_tokens.py` | - | `tests/agent/test_auxiliary_openrouter_max_tokens.py` |
| `agent/test_auxiliary_parameter_rung_chaining.py` | `1` | `agent/test_auxiliary_parameter_rung_chaining.py` | - | `tests/agent/test_auxiliary_parameter_rung_chaining.py` |
| `agent/test_auxiliary_provider_supplied_client.py` | `1` | `agent/test_auxiliary_provider_supplied_client.py` | - | `tests/agent/test_auxiliary_provider_supplied_client.py` |
| `agent/test_auxiliary_reasoning_catalog_floor.py` | `1` | `agent/test_auxiliary_reasoning_catalog_floor.py` | - | `tests/agent/test_auxiliary_reasoning_catalog_floor.py` |
| `agent/test_auxiliary_reasoning_field_rejection_retry.py` | `1` | `agent/test_auxiliary_reasoning_field_rejection_retry.py` | - | `tests/agent/test_auxiliary_reasoning_field_rejection_retry.py` |
| `agent/test_auxiliary_reasoning_floor.py` | `1` | `agent/test_auxiliary_reasoning_floor.py` | - | `tests/agent/test_auxiliary_reasoning_floor.py` |
| `agent/test_auxiliary_relay.py` | `1` | `agent/test_auxiliary_relay.py` | - | `tests/agent/test_auxiliary_relay.py` |
| `agent/test_auxiliary_runtime_cache_key.py` | `1` | `agent/test_auxiliary_runtime_cache_key.py` | - | `tests/agent/test_auxiliary_runtime_cache_key.py` |
| `agent/test_auxiliary_structured_output.py` | `1` | `agent/test_auxiliary_structured_output.py` | - | `tests/agent/test_auxiliary_structured_output.py` |
| `agent/test_auxiliary_title_timeout_bound.py` | `1` | `agent/test_auxiliary_title_timeout_bound.py` | - | `tests/agent/test_auxiliary_title_timeout_bound.py` |
| `agent/test_auxiliary_transient_retry.py` | `1` | `agent/test_auxiliary_transient_retry.py` | - | `tests/agent/test_auxiliary_transient_retry.py` |
| `agent/test_auxiliary_transport_autodetect.py` | `1` | `agent/test_auxiliary_transport_autodetect.py` | - | `tests/agent/test_auxiliary_transport_autodetect.py` |
| `agent/test_auxiliary_unavailable_nous_reason.py` | `1` | `agent/test_auxiliary_unavailable_nous_reason.py` | - | `tests/agent/test_auxiliary_unavailable_nous_reason.py` |
| `agent/test_auxiliary_user_default_headers.py` | `1` | `agent/test_auxiliary_user_default_headers.py` | - | `tests/agent/test_auxiliary_user_default_headers.py` |
| `agent/test_auxiliary_wire_messages.py` | `1` | `agent/test_auxiliary_wire_messages.py` | - | `tests/agent/test_auxiliary_wire_messages.py` |
| `agent/test_azure_identity_adapter.py` | `1` | `agent/test_azure_identity_adapter.py` | - | `tests/agent/test_azure_identity_adapter.py` |
| `agent/test_backend_identity.py` | `1` | `agent/test_backend_identity.py` | - | `tests/agent/test_backend_identity.py` |
| `agent/test_background_review.py` | `1` | `agent/test_background_review.py` | - | `tests/agent/test_background_review.py` |
| `agent/test_background_review_cache_parity.py` | `1` | `agent/test_background_review_cache_parity.py` | - | `tests/agent/test_background_review_cache_parity.py` |
| `agent/test_background_review_cost_controls.py` | `1` | `agent/test_background_review_cost_controls.py` | - | `tests/agent/test_background_review_cost_controls.py` |
| `agent/test_background_review_input_budget.py` | `1` | `agent/test_background_review_input_budget.py` | - | `tests/agent/test_background_review_input_budget.py` |
| `agent/test_background_review_list_shapes.py` | `1` | `agent/test_background_review_list_shapes.py` | - | `tests/agent/test_background_review_list_shapes.py` |
| `agent/test_background_review_memory_scope.py` | `1` | `agent/test_background_review_memory_scope.py` | - | `tests/agent/test_background_review_memory_scope.py` |
| `agent/test_background_review_routed_effort.py` | `1` | `agent/test_background_review_routed_effort.py` | - | `tests/agent/test_background_review_routed_effort.py` |
| `agent/test_background_review_summary.py` | `1` | `agent/test_background_review_summary.py` | - | `tests/agent/test_background_review_summary.py` |
| `agent/test_background_review_tool_call_guard.py` | `1` | `agent/test_background_review_tool_call_guard.py` | - | `tests/agent/test_background_review_tool_call_guard.py` |
| `agent/test_background_review_toolset_restriction.py` | `1` | `agent/test_background_review_toolset_restriction.py` | - | `tests/agent/test_background_review_toolset_restriction.py` |
| `agent/test_background_review_usage.py` | `1` | `agent/test_background_review_usage.py` | - | `tests/agent/test_background_review_usage.py` |
| `agent/test_battery.py` | `1` | `agent/test_battery.py` | - | `tests/agent/test_battery.py` |
| `agent/test_bedrock_1m_context.py` | `1` | `agent/test_bedrock_1m_context.py` | - | `tests/agent/test_bedrock_1m_context.py` |
| `agent/test_bedrock_adapter.py` | `1` | `agent/test_bedrock_adapter.py` | - | `tests/agent/test_bedrock_adapter.py` |
| `agent/test_bedrock_anthropic_foreign_reasoning.py` | `1` | `agent/test_bedrock_anthropic_foreign_reasoning.py` | - | `tests/agent/test_bedrock_anthropic_foreign_reasoning.py` |
| `agent/test_bedrock_context_cache.py` | `1` | `agent/test_bedrock_context_cache.py` | - | `tests/agent/test_bedrock_context_cache.py` |
| `agent/test_bedrock_converse_reasoning_replay.py` | `1` | `agent/test_bedrock_converse_reasoning_replay.py` | - | `tests/agent/test_bedrock_converse_reasoning_replay.py` |
| `agent/test_bedrock_credential_probe.py` | `1` | `agent/test_bedrock_credential_probe.py` | - | `tests/agent/test_bedrock_credential_probe.py` |
| `agent/test_bedrock_empty_text_blocks.py` | `1` | `agent/test_bedrock_empty_text_blocks.py` | - | `tests/agent/test_bedrock_empty_text_blocks.py` |
| `agent/test_bedrock_guardrail_claude_route.py` | `1` | `agent/test_bedrock_guardrail_claude_route.py` | - | `tests/agent/test_bedrock_guardrail_claude_route.py` |
| `agent/test_bedrock_integration.py` | `1` | `agent/test_bedrock_integration.py` | - | `tests/agent/test_bedrock_integration.py` |
| `agent/test_bedrock_interrupt_post_worker.py` | `1` | `agent/test_bedrock_interrupt_post_worker.py` | - | `tests/agent/test_bedrock_interrupt_post_worker.py` |
| `agent/test_bedrock_no_sdk_retry.py` | `1` | `agent/test_bedrock_no_sdk_retry.py` | - | `tests/agent/test_bedrock_no_sdk_retry.py` |
| `agent/test_bedrock_request_timeout.py` | `1` | `agent/test_bedrock_request_timeout.py` | - | `tests/agent/test_bedrock_request_timeout.py` |
| `agent/test_bedrock_stream_completion.py` | `1` | `agent/test_bedrock_stream_completion.py` | - | `tests/agent/test_bedrock_stream_completion.py` |
| `agent/test_bedrock_transport_replay.py` | `1` | `agent/test_bedrock_transport_replay.py` | - | `tests/agent/test_bedrock_transport_replay.py` |
| `agent/test_billing_links.py` | `1` | `agent/test_billing_links.py` | - | `tests/agent/test_billing_links.py` |
| `agent/test_billing_unverified_carrythrough.py` | `1` | `agent/test_billing_unverified_carrythrough.py` | - | `tests/agent/test_billing_unverified_carrythrough.py` |
| `agent/test_billing_usage.py` | `1` | `agent/test_billing_usage.py` | - | `tests/agent/test_billing_usage.py` |
| `agent/test_billing_view.py` | `1` | `agent/test_billing_view.py` | - | `tests/agent/test_billing_view.py` |
| `agent/test_bitwarden_secrets.py` | `1` | `agent/test_bitwarden_secrets.py` | - | `tests/agent/test_bitwarden_secrets.py` |
| `agent/test_bot_chat_toolset_refresh.py` | `1` | `agent/test_bot_chat_toolset_refresh.py` | - | `tests/agent/test_bot_chat_toolset_refresh.py` |
| `agent/test_bot_profile_prompt_isolation.py` | `1` | `agent/test_bot_profile_prompt_isolation.py` | - | `tests/agent/test_bot_profile_prompt_isolation.py` |
| `agent/test_bounded_response.py` | `1` | `agent/test_bounded_response.py` | - | `tests/agent/test_bounded_response.py` |
| `agent/test_budget_reasoning_details_exclusion.py` | `1` | `agent/test_budget_reasoning_details_exclusion.py` | - | `tests/agent/test_budget_reasoning_details_exclusion.py` |
| `agent/test_builtin_memory_disabled_surface.py` | `1` | `agent/test_builtin_memory_disabled_surface.py` | - | `tests/agent/test_builtin_memory_disabled_surface.py` |
| `agent/test_cache_disabled_on_stubs.py` | `1` | `agent/test_cache_disabled_on_stubs.py` | - | `tests/agent/test_cache_disabled_on_stubs.py` |
| `agent/test_callable_api_key.py` | `1` | `agent/test_callable_api_key.py` | - | `tests/agent/test_callable_api_key.py` |
| `agent/test_canon_args_memo_parity.py` | `1` | `agent/test_canon_args_memo_parity.py` | - | `tests/agent/test_canon_args_memo_parity.py` |
| `agent/test_capability_probe_credentials.py` | `1` | `agent/test_capability_probe_credentials.py` | - | `tests/agent/test_capability_probe_credentials.py` |
| `agent/test_cascading_interrupt.py` | `1` | `agent/test_cascading_interrupt.py` | - | `tests/agent/test_cascading_interrupt.py` |
| `agent/test_chat_completion_helpers_provider_sort.py` | `1` | `agent/test_chat_completion_helpers_provider_sort.py` | - | `tests/agent/test_chat_completion_helpers_provider_sort.py` |
| `agent/test_chat_completion_helpers_tool_calls.py` | `1` | `agent/test_chat_completion_helpers_tool_calls.py` | - | `tests/agent/test_chat_completion_helpers_tool_calls.py` |
| `agent/test_chat_sdk_transform_bypass.py` | `1` | `agent/test_chat_sdk_transform_bypass.py` | - | `tests/agent/test_chat_sdk_transform_bypass.py` |
| `agent/test_cjk_token_estimation.py` | `1` | `agent/test_cjk_token_estimation.py` | - | `tests/agent/test_cjk_token_estimation.py` |
| `agent/test_claude_code_version_path.py` | `1` | `agent/test_claude_code_version_path.py` | - | `tests/agent/test_claude_code_version_path.py` |
| `agent/test_cleanup_task_resources_headed.py` | `1` | `agent/test_cleanup_task_resources_headed.py` | - | `tests/agent/test_cleanup_task_resources_headed.py` |
| `agent/test_close_interrupted_tool_sequence.py` | `1` | `agent/test_close_interrupted_tool_sequence.py` | - | `tests/agent/test_close_interrupted_tool_sequence.py` |
| `agent/test_close_memory_trim.py` | `1` | `agent/test_close_memory_trim.py` | - | `tests/agent/test_close_memory_trim.py` |
| `agent/test_codex_app_server_compaction.py` | `1` | `agent/test_codex_app_server_compaction.py` | - | `tests/agent/test_codex_app_server_compaction.py` |
| `agent/test_codex_app_server_event_bridge.py` | `1` | `agent/test_codex_app_server_event_bridge.py` | - | `tests/agent/test_codex_app_server_event_bridge.py` |
| `agent/test_codex_app_server_integration.py` | `1` | `agent/test_codex_app_server_integration.py` | - | `tests/agent/test_codex_app_server_integration.py` |
| `agent/test_codex_app_server_lifecycle.py` | `1` | `agent/test_codex_app_server_lifecycle.py` | - | `tests/agent/test_codex_app_server_lifecycle.py` |
| `agent/test_codex_app_server_persist.py` | `1` | `agent/test_codex_app_server_persist.py` | - | `tests/agent/test_codex_app_server_persist.py` |
| `agent/test_codex_app_server_thread_resume.py` | `1` | `agent/test_codex_app_server_thread_resume.py` | - | `tests/agent/test_codex_app_server_thread_resume.py` |
| `agent/test_codex_app_server_transform.py` | `1` | `agent/test_codex_app_server_transform.py` | - | `tests/agent/test_codex_app_server_transform.py` |
| `agent/test_codex_aux_no_progress_timeout.py` | `1` | `agent/test_codex_aux_no_progress_timeout.py` | - | `tests/agent/test_codex_aux_no_progress_timeout.py` |
| `agent/test_codex_aux_timeout_fd_ownership.py` | `1` | `agent/test_codex_aux_timeout_fd_ownership.py` | - | `tests/agent/test_codex_aux_timeout_fd_ownership.py` |
| `agent/test_codex_cloudflare_headers.py` | `1` | `agent/test_codex_cloudflare_headers.py` | - | `tests/agent/test_codex_cloudflare_headers.py` |
| `agent/test_codex_echo_ownership.py` | `1` | `agent/test_codex_echo_ownership.py` | - | `tests/agent/test_codex_echo_ownership.py` |
| `agent/test_codex_first_event_timing.py` | `1` | `agent/test_codex_first_event_timing.py` | - | `tests/agent/test_codex_first_event_timing.py` |
| `agent/test_codex_gpt55_autoraise_notice.py` | `1` | `agent/test_codex_gpt55_autoraise_notice.py` | - | `tests/agent/test_codex_gpt55_autoraise_notice.py` |
| `agent/test_codex_happy_eyeballs.py` | `1` | `agent/test_codex_happy_eyeballs.py` | - | `tests/agent/test_codex_happy_eyeballs.py` |
| `agent/test_codex_incomplete_budget_escalation.py` | `1` | `agent/test_codex_incomplete_budget_escalation.py` | - | `tests/agent/test_codex_incomplete_budget_escalation.py` |
| `agent/test_codex_model_entitlement_rotation.py` | `1` | `agent/test_codex_model_entitlement_rotation.py` | - | `tests/agent/test_codex_model_entitlement_rotation.py` |
| `agent/test_codex_multimodal_tool_result.py` | `1` | `agent/test_codex_multimodal_tool_result.py` | - | `tests/agent/test_codex_multimodal_tool_result.py` |
| `agent/test_codex_no_tools_nonetype.py` | `1` | `agent/test_codex_no_tools_nonetype.py` | - | `tests/agent/test_codex_no_tools_nonetype.py` |
| `agent/test_codex_pending_tool_text.py` | `1` | `agent/test_codex_pending_tool_text.py` | - | `tests/agent/test_codex_pending_tool_text.py` |
| `agent/test_codex_reasoning_only_streak.py` | `1` | `agent/test_codex_reasoning_only_streak.py` | - | `tests/agent/test_codex_reasoning_only_streak.py` |
| `agent/test_codex_request_compression.py` | `1` | `agent/test_codex_request_compression.py` | - | `tests/agent/test_codex_request_compression.py` |
| `agent/test_codex_request_transport_diagnostics.py` | `1` | `agent/test_codex_request_transport_diagnostics.py` | - | `tests/agent/test_codex_request_transport_diagnostics.py` |
| `agent/test_codex_responses_adapter.py` | `1` | `agent/test_codex_responses_adapter.py` | - | `tests/agent/test_codex_responses_adapter.py` |
| `agent/test_codex_responses_settle_pending_tool_calls.py` | `1` | `agent/test_codex_responses_settle_pending_tool_calls.py` | - | `tests/agent/test_codex_responses_settle_pending_tool_calls.py` |
| `agent/test_codex_runtime_history_seed.py` | `1` | `agent/test_codex_runtime_history_seed.py` | - | `tests/agent/test_codex_runtime_history_seed.py` |
| `agent/test_codex_runtime_live_events.py` | `1` | `agent/test_codex_runtime_live_events.py` | - | `tests/agent/test_codex_runtime_live_events.py` |
| `agent/test_codex_runtime_prompt_handoff.py` | `1` | `agent/test_codex_runtime_prompt_handoff.py` | - | `tests/agent/test_codex_runtime_prompt_handoff.py` |
| `agent/test_codex_sdk_transform_bypass.py` | `1` | `agent/test_codex_sdk_transform_bypass.py` | - | `tests/agent/test_codex_sdk_transform_bypass.py` |
| `agent/test_codex_service_tiers.py` | `1` | `agent/test_codex_service_tiers.py` | - | `tests/agent/test_codex_service_tiers.py` |
| `agent/test_codex_soft_failure_pool_rotation.py` | `1` | `agent/test_codex_soft_failure_pool_rotation.py` | - | `tests/agent/test_codex_soft_failure_pool_rotation.py` |
| `agent/test_codex_stream_rotating_item_ids.py` | `1` | `agent/test_codex_stream_rotating_item_ids.py` | - | `tests/agent/test_codex_stream_rotating_item_ids.py` |
| `agent/test_codex_stream_supersession.py` | `1` | `agent/test_codex_stream_supersession.py` | - | `tests/agent/test_codex_stream_supersession.py` |
| `agent/test_codex_token_expired_replay_recovery.py` | `1` | `agent/test_codex_token_expired_replay_recovery.py` | - | `tests/agent/test_codex_token_expired_replay_recovery.py` |
| `agent/test_codex_ttfb_watchdog.py` | `1` | `agent/test_codex_ttfb_watchdog.py` | - | `tests/agent/test_codex_ttfb_watchdog.py` |
| `agent/test_codex_usage_attribution.py` | `1` | `agent/test_codex_usage_attribution.py` | - | `tests/agent/test_codex_usage_attribution.py` |
| `agent/test_codex_watchdog_reasoning_effort.py` | `1` | `agent/test_codex_watchdog_reasoning_effort.py` | - | `tests/agent/test_codex_watchdog_reasoning_effort.py` |
| `agent/test_codex_xai_oauth_recovery.py` | `1` | `agent/test_codex_xai_oauth_recovery.py` | - | `tests/agent/test_codex_xai_oauth_recovery.py` |
| `agent/test_coding_context.py` | `1` | `agent/test_coding_context.py` | - | `tests/agent/test_coding_context.py` |
| `agent/test_coding_worktree_note.py` | `1` | `agent/test_coding_worktree_note.py` | - | `tests/agent/test_coding_worktree_note.py` |
| `agent/test_command_secret_source.py` | `1` | `agent/test_command_secret_source.py` | - | `tests/agent/test_command_secret_source.py` |
| `agent/test_command_token_source.py` | `1` | `agent/test_command_token_source.py` | - | `tests/agent/test_command_token_source.py` |
| `agent/test_commit_memory_session_context_engine.py` | `1` | `agent/test_commit_memory_session_context_engine.py` | - | `tests/agent/test_commit_memory_session_context_engine.py` |
| `agent/test_compaction_anti_thrash.py` | `1` | `agent/test_compaction_anti_thrash.py` | - | `tests/agent/test_compaction_anti_thrash.py` |
| `agent/test_compaction_events.py` | `1` | `agent/test_compaction_events.py` | - | `tests/agent/test_compaction_events.py` |
| `agent/test_compaction_ops_notification_anchor.py` | `1` | `agent/test_compaction_ops_notification_anchor.py` | - | `tests/agent/test_compaction_ops_notification_anchor.py` |
| `agent/test_compaction_prompt_rebuild.py` | `1` | `agent/test_compaction_prompt_rebuild.py` | - | `tests/agent/test_compaction_prompt_rebuild.py` |
| `agent/test_compaction_redaction_boundaries.py` | `1` | `agent/test_compaction_redaction_boundaries.py` | - | `tests/agent/test_compaction_redaction_boundaries.py` |
| `agent/test_compress_context_progress_timeout.py` | `1` | `agent/test_compress_context_progress_timeout.py` | - | `tests/agent/test_compress_context_progress_timeout.py` |
| `agent/test_compress_focus.py` | `1` | `agent/test_compress_focus.py` | - | `tests/agent/test_compress_focus.py` |
| `agent/test_compress_signal_leak.py` | `1` | `agent/test_compress_signal_leak.py` | - | `tests/agent/test_compress_signal_leak.py` |
| `agent/test_compressed_summary_metadata.py` | `1` | `agent/test_compressed_summary_metadata.py` | - | `tests/agent/test_compressed_summary_metadata.py` |
| `agent/test_compression_abort_state_reset.py` | `1` | `agent/test_compression_abort_state_reset.py` | - | `tests/agent/test_compression_abort_state_reset.py` |
| `agent/test_compression_adoption_preserves_live_tail.py` | `1` | `agent/test_compression_adoption_preserves_live_tail.py` | - | `tests/agent/test_compression_adoption_preserves_live_tail.py` |
| `agent/test_compression_anti_thrash_persistence.py` | `1` | `agent/test_compression_anti_thrash_persistence.py` | - | `tests/agent/test_compression_anti_thrash_persistence.py` |
| `agent/test_compression_anti_thrash_recovery.py` | `1` | `agent/test_compression_anti_thrash_recovery.py` | - | `tests/agent/test_compression_anti_thrash_recovery.py` |
| `agent/test_compression_attempt_effect.py` | `1` | `agent/test_compression_attempt_effect.py` | - | `tests/agent/test_compression_attempt_effect.py` |
| `agent/test_compression_attempt_lifecycle.py` | `1` | `agent/test_compression_attempt_lifecycle.py` | - | `tests/agent/test_compression_attempt_lifecycle.py` |
| `agent/test_compression_attempt_ownership.py` | `1` | `agent/test_compression_attempt_ownership.py` | - | `tests/agent/test_compression_attempt_ownership.py` |
| `agent/test_compression_attempt_telemetry.py` | `1` | `agent/test_compression_attempt_telemetry.py` | - | `tests/agent/test_compression_attempt_telemetry.py` |
| `agent/test_compression_auto_summary_fallback.py` | `1` | `agent/test_compression_auto_summary_fallback.py` | - | `tests/agent/test_compression_auto_summary_fallback.py` |
| `agent/test_compression_aux_fallback_authority.py` | `1` | `agent/test_compression_aux_fallback_authority.py` | - | `tests/agent/test_compression_aux_fallback_authority.py` |
| `agent/test_compression_aux_fallback_route.py` | `1` | `agent/test_compression_aux_fallback_route.py` | - | `tests/agent/test_compression_aux_fallback_route.py` |
| `agent/test_compression_boundary.py` | `1` | `agent/test_compression_boundary.py` | - | `tests/agent/test_compression_boundary.py` |
| `agent/test_compression_boundary_hook.py` | `1` | `agent/test_compression_boundary_hook.py` | - | `tests/agent/test_compression_boundary_hook.py` |
| `agent/test_compression_budget_rearm.py` | `1` | `agent/test_compression_budget_rearm.py` | - | `tests/agent/test_compression_budget_rearm.py` |
| `agent/test_compression_budget_refund.py` | `1` | `agent/test_compression_budget_refund.py` | - | `tests/agent/test_compression_budget_refund.py` |
| `agent/test_compression_busy_steer_anchor.py` | `1` | `agent/test_compression_busy_steer_anchor.py` | - | `tests/agent/test_compression_busy_steer_anchor.py` |
| `agent/test_compression_call_llm_fallback_reported.py` | `1` | `agent/test_compression_call_llm_fallback_reported.py` | - | `tests/agent/test_compression_call_llm_fallback_reported.py` |
| `agent/test_compression_child_source.py` | `1` | `agent/test_compression_child_source.py` | - | `tests/agent/test_compression_child_source.py` |
| `agent/test_compression_closed_adoption.py` | `1` | `agent/test_compression_closed_adoption.py` | - | `tests/agent/test_compression_closed_adoption.py` |
| `agent/test_compression_commit_fence_race.py` | `1` | `agent/test_compression_commit_fence_race.py` | - | `tests/agent/test_compression_commit_fence_race.py` |
| `agent/test_compression_concurrent_fork.py` | `1` | `agent/test_compression_concurrent_fork.py` | - | `tests/agent/test_compression_concurrent_fork.py` |
| `agent/test_compression_config_defaults.py` | `1` | `agent/test_compression_config_defaults.py` | - | `tests/agent/test_compression_config_defaults.py` |
| `agent/test_compression_count_warning.py` | `1` | `agent/test_compression_count_warning.py` | - | `tests/agent/test_compression_count_warning.py` |
| `agent/test_compression_fallback_budget.py` | `1` | `agent/test_compression_fallback_budget.py` | - | `tests/agent/test_compression_fallback_budget.py` |
| `agent/test_compression_feasibility.py` | `1` | `agent/test_compression_feasibility.py` | - | `tests/agent/test_compression_feasibility.py` |
| `agent/test_compression_interrupt_protection.py` | `1` | `agent/test_compression_interrupt_protection.py` | - | `tests/agent/test_compression_interrupt_protection.py` |
| `agent/test_compression_last_assistant_anchor.py` | `1` | `agent/test_compression_last_assistant_anchor.py` | - | `tests/agent/test_compression_last_assistant_anchor.py` |
| `agent/test_compression_lock_defer.py` | `1` | `agent/test_compression_lock_defer.py` | - | `tests/agent/test_compression_lock_defer.py` |
| `agent/test_compression_logging_session_context.py` | `1` | `agent/test_compression_logging_session_context.py` | - | `tests/agent/test_compression_logging_session_context.py` |
| `agent/test_compression_marker.py` | `1` | `agent/test_compression_marker.py` | - | `tests/agent/test_compression_marker.py` |
| `agent/test_compression_max_attempts_config.py` | `1` | `agent/test_compression_max_attempts_config.py` | - | `tests/agent/test_compression_max_attempts_config.py` |
| `agent/test_compression_orphan_recovery.py` | `1` | `agent/test_compression_orphan_recovery.py` | - | `tests/agent/test_compression_orphan_recovery.py` |
| `agent/test_compression_persistence.py` | `1` | `agent/test_compression_persistence.py` | - | `tests/agent/test_compression_persistence.py` |
| `agent/test_compression_progress.py` | `1` | `agent/test_compression_progress.py` | - | `tests/agent/test_compression_progress.py` |
| `agent/test_compression_prompt_reading.py` | `1` | `agent/test_compression_prompt_reading.py` | - | `tests/agent/test_compression_prompt_reading.py` |
| `agent/test_compression_refusal_restores_transcript.py` | `1` | `agent/test_compression_refusal_restores_transcript.py` | - | `tests/agent/test_compression_refusal_restores_transcript.py` |
| `agent/test_compression_review.py` | `1` | `agent/test_compression_review.py` | - | `tests/agent/test_compression_review.py` |
| `agent/test_compression_rotation_state.py` | `1` | `agent/test_compression_rotation_state.py` | - | `tests/agent/test_compression_rotation_state.py` |
| `agent/test_compression_skill_safety_single_copy.py` | `1` | `agent/test_compression_skill_safety_single_copy.py` | - | `tests/agent/test_compression_skill_safety_single_copy.py` |
| `agent/test_compression_small_ctx_threshold_floor.py` | `1` | `agent/test_compression_small_ctx_threshold_floor.py` | - | `tests/agent/test_compression_small_ctx_threshold_floor.py` |
| `agent/test_compression_stall_deterministic_fallback.py` | `1` | `agent/test_compression_stall_deterministic_fallback.py` | - | `tests/agent/test_compression_stall_deterministic_fallback.py` |
| `agent/test_compression_stall_fallback.py` | `1` | `agent/test_compression_stall_fallback.py` | - | `tests/agent/test_compression_stall_fallback.py` |
| `agent/test_compression_stall_interrupt.py` | `1` | `agent/test_compression_stall_interrupt.py` | - | `tests/agent/test_compression_stall_interrupt.py` |
| `agent/test_compression_supersession_split.py` | `1` | `agent/test_compression_supersession_split.py` | - | `tests/agent/test_compression_supersession_split.py` |
| `agent/test_compression_tail_threshold_share.py` | `1` | `agent/test_compression_tail_threshold_share.py` | - | `tests/agent/test_compression_tail_threshold_share.py` |
| `agent/test_compression_terminal_activity_persist.py` | `1` | `agent/test_compression_terminal_activity_persist.py` | - | `tests/agent/test_compression_terminal_activity_persist.py` |
| `agent/test_compression_timeout_terminal.py` | `1` | `agent/test_compression_timeout_terminal.py` | - | `tests/agent/test_compression_timeout_terminal.py` |
| `agent/test_compression_trigger_labels.py` | `1` | `agent/test_compression_trigger_labels.py` | - | `tests/agent/test_compression_trigger_labels.py` |
| `agent/test_compression_worker_isolation.py` | `1` | `agent/test_compression_worker_isolation.py` | - | `tests/agent/test_compression_worker_isolation.py` |
| `agent/test_compressor_actionable_tail_anchor.py` | `1` | `agent/test_compressor_actionable_tail_anchor.py` | - | `tests/agent/test_compressor_actionable_tail_anchor.py` |
| `agent/test_compressor_assistant_tail_anchor.py` | `1` | `agent/test_compressor_assistant_tail_anchor.py` | - | `tests/agent/test_compressor_assistant_tail_anchor.py` |
| `agent/test_compressor_fallback_update.py` | `1` | `agent/test_compressor_fallback_update.py` | - | `tests/agent/test_compressor_fallback_update.py` |
| `agent/test_compressor_handoff_tail_price.py` | `1` | `agent/test_compressor_handoff_tail_price.py` | - | `tests/agent/test_compressor_handoff_tail_price.py` |
| `agent/test_compressor_handoff_tool_group_floor.py` | `1` | `agent/test_compressor_handoff_tool_group_floor.py` | - | `tests/agent/test_compressor_handoff_tool_group_floor.py` |
| `agent/test_compressor_historical_media.py` | `1` | `agent/test_compressor_historical_media.py` | - | `tests/agent/test_compressor_historical_media.py` |
| `agent/test_compressor_image_tokens.py` | `1` | `agent/test_compressor_image_tokens.py` | - | `tests/agent/test_compressor_image_tokens.py` |
| `agent/test_compressor_media_stripping.py` | `1` | `agent/test_compressor_media_stripping.py` | - | `tests/agent/test_compressor_media_stripping.py` |
| `agent/test_compressor_stale_tool_images.py` | `1` | `agent/test_compressor_stale_tool_images.py` | - | `tests/agent/test_compressor_stale_tool_images.py` |
| `agent/test_compressor_tail_cut_oob_fix.py` | `1` | `agent/test_compressor_tail_cut_oob_fix.py` | - | `tests/agent/test_compressor_tail_cut_oob_fix.py` |
| `agent/test_compressor_tail_cut_tool_pair_floor.py` | `1` | `agent/test_compressor_tail_cut_tool_pair_floor.py` | - | `tests/agent/test_compressor_tail_cut_tool_pair_floor.py` |
| `agent/test_compressor_tool_call_budget.py` | `1` | `agent/test_compressor_tool_call_budget.py` | - | `tests/agent/test_compressor_tool_call_budget.py` |
| `agent/test_compressor_truncated_summary_guard.py` | `1` | `agent/test_compressor_truncated_summary_guard.py` | - | `tests/agent/test_compressor_truncated_summary_guard.py` |
| `agent/test_compressor_zero_user_guard.py` | `1` | `agent/test_compressor_zero_user_guard.py` | - | `tests/agent/test_compressor_zero_user_guard.py` |
| `agent/test_concurrent_interrupt.py` | `1` | `agent/test_concurrent_interrupt.py` | - | `tests/agent/test_concurrent_interrupt.py` |
| `agent/test_config_load_warning.py` | `1` | `agent/test_config_load_warning.py` | - | `tests/agent/test_config_load_warning.py` |
| `agent/test_content_policy_blocked.py` | `1` | `agent/test_content_policy_blocked.py` | - | `tests/agent/test_content_policy_blocked.py` |
| `agent/test_content_policy_refusal_stop_details.py` | `1` | `agent/test_content_policy_refusal_stop_details.py` | - | `tests/agent/test_content_policy_refusal_stop_details.py` |
| `agent/test_context_breakdown.py` | `1` | `agent/test_context_breakdown.py` | - | `tests/agent/test_context_breakdown.py` |
| `agent/test_context_compressor_bench_probe.py` | `1` | `agent/test_context_compressor_bench_probe.py` | - | `tests/agent/test_context_compressor_bench_probe.py` |
| `agent/test_context_compressor_cross_session_guard.py` | `1` | `agent/test_context_compressor_cross_session_guard.py` | - | `tests/agent/test_context_compressor_cross_session_guard.py` |
| `agent/test_context_compressor_custom_providers.py` | `1` | `agent/test_context_compressor_custom_providers.py` | - | `tests/agent/test_context_compressor_custom_providers.py` |
| `agent/test_context_compressor_overload.py` | `1` | `agent/test_context_compressor_overload.py` | - | `tests/agent/test_context_compressor_overload.py` |
| `agent/test_context_compressor_reasoning_fallback.py` | `1` | `agent/test_context_compressor_reasoning_fallback.py` | - | `tests/agent/test_context_compressor_reasoning_fallback.py` |
| `agent/test_context_compressor_session_end_clears_state.py` | `1` | `agent/test_context_compressor_session_end_clears_state.py` | - | `tests/agent/test_context_compressor_session_end_clears_state.py` |
| `agent/test_context_compressor_structural_backoff.py` | `1` | `agent/test_context_compressor_structural_backoff.py` | - | `tests/agent/test_context_compressor_structural_backoff.py` |
| `agent/test_context_compressor_summary_continuity.py` | `1` | `agent/test_context_compressor_summary_continuity.py` | - | `tests/agent/test_context_compressor_summary_continuity.py` |
| `agent/test_context_compressor_summary_hook_compat.py` | `1` | `agent/test_context_compressor_summary_hook_compat.py` | - | `tests/agent/test_context_compressor_summary_hook_compat.py` |
| `agent/test_context_compressor_tail.py` | `1` | `agent/test_context_compressor_tail.py` | - | `tests/agent/test_context_compressor_tail.py` |
| `agent/test_context_compressor_task_heading.py` | `1` | `agent/test_context_compressor_task_heading.py` | - | `tests/agent/test_context_compressor_task_heading.py` |
| `agent/test_context_compressor_temporal_anchoring.py` | `1` | `agent/test_context_compressor_temporal_anchoring.py` | - | `tests/agent/test_context_compressor_temporal_anchoring.py` |
| `agent/test_context_compressor_zero_user_provenance.py` | `1` | `agent/test_context_compressor_zero_user_provenance.py` | - | `tests/agent/test_context_compressor_zero_user_provenance.py` |
| `agent/test_context_display_provenance.py` | `1` | `agent/test_context_display_provenance.py` | - | `tests/agent/test_context_display_provenance.py` |
| `agent/test_context_engine_host_contract.py` | `1` | `agent/test_context_engine_host_contract.py` | - | `tests/agent/test_context_engine_host_contract.py` |
| `agent/test_context_engine_message_uid.py` | `1` | `agent/test_context_engine_message_uid.py` | - | `tests/agent/test_context_engine_message_uid.py` |
| `agent/test_context_engine_on_turn_complete_usage.py` | `1` | `agent/test_context_engine_on_turn_complete_usage.py` | - | `tests/agent/test_context_engine_on_turn_complete_usage.py` |
| `agent/test_context_engine_select_context.py` | `1` | `agent/test_context_engine_select_context.py` | - | `tests/agent/test_context_engine_select_context.py` |
| `agent/test_context_estimator_multimodal.py` | `1` | `agent/test_context_estimator_multimodal.py` | - | `tests/agent/test_context_estimator_multimodal.py` |
| `agent/test_context_file_sources.py` | `1` | `agent/test_context_file_sources.py` | - | `tests/agent/test_context_file_sources.py` |
| `agent/test_context_pin.py` | `1` | `agent/test_context_pin.py` | - | `tests/agent/test_context_pin.py` |
| `agent/test_context_references.py` | `1` | `agent/test_context_references.py` | - | `tests/agent/test_context_references.py` |
| `agent/test_context_refs_concurrent.py` | `1` | `agent/test_context_refs_concurrent.py` | - | `tests/agent/test_context_refs_concurrent.py` |
| `agent/test_context_route_mismatch.py` | `1` | `agent/test_context_route_mismatch.py` | - | `tests/agent/test_context_route_mismatch.py` |
| `agent/test_context_token_tracking.py` | `1` | `agent/test_context_token_tracking.py` | - | `tests/agent/test_context_token_tracking.py` |
| `agent/test_continuation_ceiling_wedge.py` | `1` | `agent/test_continuation_ceiling_wedge.py` | - | `tests/agent/test_continuation_ceiling_wedge.py` |
| `agent/test_continuation_repetition_guard.py` | `1` | `agent/test_continuation_repetition_guard.py` | - | `tests/agent/test_continuation_repetition_guard.py` |
| `agent/test_conversation_compression_manual.py` | `1` | `agent/test_conversation_compression_manual.py` | - | `tests/agent/test_conversation_compression_manual.py` |
| `agent/test_conversation_fallback_state.py` | `1` | `agent/test_conversation_fallback_state.py` | - | `tests/agent/test_conversation_fallback_state.py` |
| `agent/test_conversation_loop_interpreter_shutdown.py` | `1` | `agent/test_conversation_loop_interpreter_shutdown.py` | - | `tests/agent/test_conversation_loop_interpreter_shutdown.py` |
| `agent/test_copilot_acp_alias_compat.py` | `1` | `agent/test_copilot_acp_alias_compat.py` | - | `tests/agent/test_copilot_acp_alias_compat.py` |
| `agent/test_corruption_recovery_guidance.py` | `1` | `agent/test_corruption_recovery_guidance.py` | - | `tests/agent/test_corruption_recovery_guidance.py` |
| `agent/test_create_openai_client_disables_sdk_retries.py` | `1` | `agent/test_create_openai_client_disables_sdk_retries.py` | - | `tests/agent/test_create_openai_client_disables_sdk_retries.py` |
| `agent/test_create_openai_client_kwargs_isolation.py` | `1` | `agent/test_create_openai_client_kwargs_isolation.py` | - | `tests/agent/test_create_openai_client_kwargs_isolation.py` |
| `agent/test_create_openai_client_proxy_env.py` | `1` | `agent/test_create_openai_client_proxy_env.py` | - | `tests/agent/test_create_openai_client_proxy_env.py` |
| `agent/test_create_openai_client_reuse.py` | `1` | `agent/test_create_openai_client_reuse.py` | - | `tests/agent/test_create_openai_client_reuse.py` |
| `agent/test_create_openai_client_ssl_verify.py` | `1` | `agent/test_create_openai_client_ssl_verify.py` | - | `tests/agent/test_create_openai_client_ssl_verify.py` |
| `agent/test_credential_pool.py` | `1` | `agent/test_credential_pool.py` | - | `tests/agent/test_credential_pool.py` |
| `agent/test_credential_pool_anthropic_refresh_race.py` | `1` | `agent/test_credential_pool_anthropic_refresh_race.py` | - | `tests/agent/test_credential_pool_anthropic_refresh_race.py` |
| `agent/test_credential_pool_auto_detect.py` | `1` | `agent/test_credential_pool_auto_detect.py` | - | `tests/agent/test_credential_pool_auto_detect.py` |
| `agent/test_credential_pool_codex_singleton_isolation.py` | `1` | `agent/test_credential_pool_codex_singleton_isolation.py` | - | `tests/agent/test_credential_pool_codex_singleton_isolation.py` |
| `agent/test_credential_pool_connectors_guest.py` | `1` | `agent/test_credential_pool_connectors_guest.py` | - | `tests/agent/test_credential_pool_connectors_guest.py` |
| `agent/test_credential_pool_deferred_refresh.py` | `1` | `agent/test_credential_pool_deferred_refresh.py` | - | `tests/agent/test_credential_pool_deferred_refresh.py` |
| `agent/test_credential_pool_interrupt.py` | `1` | `agent/test_credential_pool_interrupt.py` | - | `tests/agent/test_credential_pool_interrupt.py` |
| `agent/test_credential_pool_key_rotation.py` | `1` | `agent/test_credential_pool_key_rotation.py` | - | `tests/agent/test_credential_pool_key_rotation.py` |
| `agent/test_credential_pool_lease_refresh_reselect.py` | `1` | `agent/test_credential_pool_lease_refresh_reselect.py` | - | `tests/agent/test_credential_pool_lease_refresh_reselect.py` |
| `agent/test_credential_pool_no_entries_log_throttle.py` | `1` | `agent/test_credential_pool_no_entries_log_throttle.py` | - | `tests/agent/test_credential_pool_no_entries_log_throttle.py` |
| `agent/test_credential_pool_nous_refresh_stampede.py` | `1` | `agent/test_credential_pool_nous_refresh_stampede.py` | - | `tests/agent/test_credential_pool_nous_refresh_stampede.py` |
| `agent/test_credential_pool_nous_transient_401.py` | `1` | `agent/test_credential_pool_nous_transient_401.py` | - | `tests/agent/test_credential_pool_nous_transient_401.py` |
| `agent/test_credential_pool_oat_authtype.py` | `1` | `agent/test_credential_pool_oat_authtype.py` | - | `tests/agent/test_credential_pool_oat_authtype.py` |
| `agent/test_credential_pool_oauth_writethrough.py` | `1` | `agent/test_credential_pool_oauth_writethrough.py` | - | `tests/agent/test_credential_pool_oauth_writethrough.py` |
| `agent/test_credential_pool_operations.py` | `1` | `agent/test_credential_pool_operations.py` | - | `tests/agent/test_credential_pool_operations.py` |
| `agent/test_credential_pool_plugin_seam.py` | `1` | `agent/test_credential_pool_plugin_seam.py` | - | `tests/agent/test_credential_pool_plugin_seam.py` |
| `agent/test_credential_pool_profile_oauth_fork.py` | `1` | `agent/test_credential_pool_profile_oauth_fork.py` | - | `tests/agent/test_credential_pool_profile_oauth_fork.py` |
| `agent/test_credential_pool_provider_boundary.py` | `1` | `agent/test_credential_pool_provider_boundary.py` | - | `tests/agent/test_credential_pool_provider_boundary.py` |
| `agent/test_credential_pool_refresh_sweep_spacing.py` | `1` | `agent/test_credential_pool_refresh_sweep_spacing.py` | - | `tests/agent/test_credential_pool_refresh_sweep_spacing.py` |
| `agent/test_credential_pool_routing.py` | `1` | `agent/test_credential_pool_routing.py` | - | `tests/agent/test_credential_pool_routing.py` |
| `agent/test_credential_pool_seed_existing_env_sources.py` | `1` | `agent/test_credential_pool_seed_existing_env_sources.py` | - | `tests/agent/test_credential_pool_seed_existing_env_sources.py` |
| `agent/test_credential_pool_sole_cooldown.py` | `1` | `agent/test_credential_pool_sole_cooldown.py` | - | `tests/agent/test_credential_pool_sole_cooldown.py` |
| `agent/test_credential_pool_sole_rotate_recovery.py` | `1` | `agent/test_credential_pool_sole_rotate_recovery.py` | - | `tests/agent/test_credential_pool_sole_rotate_recovery.py` |
| `agent/test_credential_pool_terminal_refresh_visibility.py` | `1` | `agent/test_credential_pool_terminal_refresh_visibility.py` | - | `tests/agent/test_credential_pool_terminal_refresh_visibility.py` |
| `agent/test_credential_pool_unmatched_rotation_bound.py` | `1` | `agent/test_credential_pool_unmatched_rotation_bound.py` | - | `tests/agent/test_credential_pool_unmatched_rotation_bound.py` |
| `agent/test_credential_rotation_route_settings.py` | `1` | `agent/test_credential_rotation_route_settings.py` | - | `tests/agent/test_credential_rotation_route_settings.py` |
| `agent/test_credential_source_labels.py` | `1` | `agent/test_credential_source_labels.py` | - | `tests/agent/test_credential_source_labels.py` |
| `agent/test_credit_404_fallback.py` | `1` | `agent/test_credit_404_fallback.py` | - | `tests/agent/test_credit_404_fallback.py` |
| `agent/test_credits_cold_start.py` | `1` | `agent/test_credits_cold_start.py` | - | `tests/agent/test_credits_cold_start.py` |
| `agent/test_credits_notices_toggle.py` | `1` | `agent/test_credits_notices_toggle.py` | - | `tests/agent/test_credits_notices_toggle.py` |
| `agent/test_credits_policy.py` | `1` | `agent/test_credits_policy.py` | - | `tests/agent/test_credits_policy.py` |
| `agent/test_credits_tracker.py` | `1` | `agent/test_credits_tracker.py` | - | `tests/agent/test_credits_tracker.py` |
| `agent/test_credits_usage_window.py` | `1` | `agent/test_credits_usage_window.py` | - | `tests/agent/test_credits_usage_window.py` |
| `agent/test_credits_view.py` | `1` | `agent/test_credits_view.py` | - | `tests/agent/test_credits_view.py` |
| `agent/test_cron_inflight_prompt_reappend.py` | `1` | `agent/test_cron_inflight_prompt_reappend.py` | - | `tests/agent/test_cron_inflight_prompt_reappend.py` |
| `agent/test_cron_inline_api_call.py` | `1` | `agent/test_cron_inline_api_call.py` | - | `tests/agent/test_cron_inline_api_call.py` |
| `agent/test_cross_process_turn_lease.py` | `1` | `agent/test_cross_process_turn_lease.py` | - | `tests/agent/test_cross_process_turn_lease.py` |
| `agent/test_cross_turn_reasoning_continuity.py` | `1` | `agent/test_cross_turn_reasoning_continuity.py` | - | `tests/agent/test_cross_turn_reasoning_continuity.py` |
| `agent/test_crossloop_client_cache.py` | `1` | `agent/test_crossloop_client_cache.py` | - | `tests/agent/test_crossloop_client_cache.py` |
| `agent/test_ctx_halving_fix.py` | `1` | `agent/test_ctx_halving_fix.py` | - | `tests/agent/test_ctx_halving_fix.py` |
| `agent/test_curator.py` | `1` | `agent/test_curator.py` | - | `tests/agent/test_curator.py` |
| `agent/test_curator_activity.py` | `1` | `agent/test_curator_activity.py` | - | `tests/agent/test_curator_activity.py` |
| `agent/test_curator_backup.py` | `1` | `agent/test_curator_backup.py` | - | `tests/agent/test_curator_backup.py` |
| `agent/test_curator_classification.py` | `1` | `agent/test_curator_classification.py` | - | `tests/agent/test_curator_classification.py` |
| `agent/test_curator_reports.py` | `1` | `agent/test_curator_reports.py` | - | `tests/agent/test_curator_reports.py` |
| `agent/test_curator_rollback_staging.py` | `1` | `agent/test_curator_rollback_staging.py` | - | `tests/agent/test_curator_rollback_staging.py` |
| `agent/test_curator_run_hygiene.py` | `1` | `agent/test_curator_run_hygiene.py` | - | `tests/agent/test_curator_run_hygiene.py` |
| `agent/test_cursor_optimizations_parity.py` | `1` | `agent/test_cursor_optimizations_parity.py` | - | `tests/agent/test_cursor_optimizations_parity.py` |
| `agent/test_custom_pool_mismatch_guard.py` | `1` | `agent/test_custom_pool_mismatch_guard.py` | - | `tests/agent/test_custom_pool_mismatch_guard.py` |
| `agent/test_custom_provider_ca_probes.py` | `1` | `agent/test_custom_provider_ca_probes.py` | - | `tests/agent/test_custom_provider_ca_probes.py` |
| `agent/test_custom_provider_extra_body.py` | `1` | `agent/test_custom_provider_extra_body.py` | - | `tests/agent/test_custom_provider_extra_body.py` |
| `agent/test_custom_provider_extra_body_matching.py` | `1` | `agent/test_custom_provider_extra_body_matching.py` | - | `tests/agent/test_custom_provider_extra_body_matching.py` |
| `agent/test_custom_provider_extra_headers_client.py` | `1` | `agent/test_custom_provider_extra_headers_client.py` | - | `tests/agent/test_custom_provider_extra_headers_client.py` |
| `agent/test_custom_providers_vision.py` | `1` | `agent/test_custom_providers_vision.py` | - | `tests/agent/test_custom_providers_vision.py` |
| `agent/test_dead_connection_cleanup.py` | `1` | `agent/test_dead_connection_cleanup.py` | - | `tests/agent/test_dead_connection_cleanup.py` |
| `agent/test_deadline.py` | `1` | `agent/test_deadline.py` | - | `tests/agent/test_deadline.py` |
| `agent/test_deadline_fork_race.py` | `1` | `agent/test_deadline_fork_race.py` | - | `tests/agent/test_deadline_fork_race.py` |
| `agent/test_declared_conversation_scope.py` | `1` | `agent/test_declared_conversation_scope.py` | - | `tests/agent/test_declared_conversation_scope.py` |
| `agent/test_deepseek_anthropic_thinking.py` | `1` | `agent/test_deepseek_anthropic_thinking.py` | - | `tests/agent/test_deepseek_anthropic_thinking.py` |
| `agent/test_deepseek_reasoning_content_echo.py` | `1` | `agent/test_deepseek_reasoning_content_echo.py` | - | `tests/agent/test_deepseek_reasoning_content_echo.py` |
| `agent/test_degenerate_final_recovery.py` | `1` | `agent/test_degenerate_final_recovery.py` | - | `tests/agent/test_degenerate_final_recovery.py` |
| `agent/test_delegation_switch_next_session.py` | `1` | `agent/test_delegation_switch_next_session.py` | - | `tests/agent/test_delegation_switch_next_session.py` |
| `agent/test_detached_fork_lifecycle_hooks.py` | `1` | `agent/test_detached_fork_lifecycle_hooks.py` | - | `tests/agent/test_detached_fork_lifecycle_hooks.py` |
| `agent/test_dict_tool_call_args.py` | `1` | `agent/test_dict_tool_call_args.py` | - | `tests/agent/test_dict_tool_call_args.py` |
| `agent/test_direct_contexts_stream_inline.py` | `1` | `agent/test_direct_contexts_stream_inline.py` | - | `tests/agent/test_direct_contexts_stream_inline.py` |
| `agent/test_display.py` | `1` | `agent/test_display.py` | - | `tests/agent/test_display.py` |
| `agent/test_display_emoji.py` | `1` | `agent/test_display_emoji.py` | - | `tests/agent/test_display_emoji.py` |
| `agent/test_display_todo_progress.py` | `1` | `agent/test_display_todo_progress.py` | - | `tests/agent/test_display_todo_progress.py` |
| `agent/test_display_tool_failure.py` | `1` | `agent/test_display_tool_failure.py` | - | `tests/agent/test_display_tool_failure.py` |
| `agent/test_dropped_tool_call_recovery.py` | `1` | `agent/test_dropped_tool_call_recovery.py` | - | `tests/agent/test_dropped_tool_call_recovery.py` |
| `agent/test_empty_response_fallback_refund.py` | `1` | `agent/test_empty_response_fallback_refund.py` | - | `tests/agent/test_empty_response_fallback_refund.py` |
| `agent/test_empty_response_guard.py` | `1` | `agent/test_empty_response_guard.py` | - | `tests/agent/test_empty_response_guard.py` |
| `agent/test_empty_response_recovery_persistence.py` | `1` | `agent/test_empty_response_recovery_persistence.py` | - | `tests/agent/test_empty_response_recovery_persistence.py` |
| `agent/test_empty_terminal_reasoning_surface.py` | `1` | `agent/test_empty_terminal_reasoning_surface.py` | - | `tests/agent/test_empty_terminal_reasoning_surface.py` |
| `agent/test_empty_tool_name_loop_dampening.py` | `1` | `agent/test_empty_tool_name_loop_dampening.py` | - | `tests/agent/test_empty_tool_name_loop_dampening.py` |
| `agent/test_endpoint_blackhole.py` | `1` | `agent/test_endpoint_blackhole.py` | - | `tests/agent/test_endpoint_blackhole.py` |
| `agent/test_endpoint_profile_seams.py` | `1` | `agent/test_endpoint_profile_seams.py` | - | `tests/agent/test_endpoint_profile_seams.py` |
| `agent/test_engine_preflight_wire.py` | `1` | `agent/test_engine_preflight_wire.py` | - | `tests/agent/test_engine_preflight_wire.py` |
| `agent/test_entitlement_fail_closed.py` | `1` | `agent/test_entitlement_fail_closed.py` | - | `tests/agent/test_entitlement_fail_closed.py` |
| `agent/test_env_credential_turn_refresh.py` | `1` | `agent/test_env_credential_turn_refresh.py` | - | `tests/agent/test_env_credential_turn_refresh.py` |
| `agent/test_env_loader_applied_homes.py` | `1` | `agent/test_env_loader_applied_homes.py` | - | `tests/agent/test_env_loader_applied_homes.py` |
| `agent/test_env_loader_reload_restore.py` | `1` | `agent/test_env_loader_reload_restore.py` | - | `tests/agent/test_env_loader_reload_restore.py` |
| `agent/test_env_loader_secret_sources.py` | `1` | `agent/test_env_loader_secret_sources.py` | - | `tests/agent/test_env_loader_secret_sources.py` |
| `agent/test_error_classifier.py` | `1` | `agent/test_error_classifier.py` | - | `tests/agent/test_error_classifier.py` |
| `agent/test_error_classifier_content_policy.py` | `1` | `agent/test_error_classifier_content_policy.py` | - | `tests/agent/test_error_classifier_content_policy.py` |
| `agent/test_error_classifier_data_policy.py` | `1` | `agent/test_error_classifier_data_policy.py` | - | `tests/agent/test_error_classifier_data_policy.py` |
| `agent/test_error_classifier_observed.py` | `1` | `agent/test_error_classifier_observed.py` | - | `tests/agent/test_error_classifier_observed.py` |
| `agent/test_error_classifier_reasoning_level.py` | `1` | `agent/test_error_classifier_reasoning_level.py` | - | `tests/agent/test_error_classifier_reasoning_level.py` |
| `agent/test_error_classifier_upstream_blocked.py` | `1` | `agent/test_error_classifier_upstream_blocked.py` | - | `tests/agent/test_error_classifier_upstream_blocked.py` |
| `agent/test_error_surface.py` | `1` | `agent/test_error_surface.py` | - | `tests/agent/test_error_surface.py` |
| `agent/test_estimator_parity.py` | `1` | `agent/test_estimator_parity.py` | - | `tests/agent/test_estimator_parity.py` |
| `agent/test_exit_cleanup_interrupt.py` | `1` | `agent/test_exit_cleanup_interrupt.py` | - | `tests/agent/test_exit_cleanup_interrupt.py` |
| `agent/test_export_current_turn_boundary.py` | `1` | `agent/test_export_current_turn_boundary.py` | - | `tests/agent/test_export_current_turn_boundary.py` |
| `agent/test_external_process_provider_init.py` | `1` | `agent/test_external_process_provider_init.py` | - | `tests/agent/test_external_process_provider_init.py` |
| `agent/test_external_process_streaming.py` | `1` | `agent/test_external_process_streaming.py` | - | `tests/agent/test_external_process_streaming.py` |
| `agent/test_external_skills.py` | `1` | `agent/test_external_skills.py` | - | `tests/agent/test_external_skills.py` |
| `agent/test_external_skills_dirs_cache.py` | `1` | `agent/test_external_skills_dirs_cache.py` | - | `tests/agent/test_external_skills_dirs_cache.py` |
| `agent/test_failed_turn_chat_copy.py` | `1` | `agent/test_failed_turn_chat_copy.py` | - | `tests/agent/test_failed_turn_chat_copy.py` |
| `agent/test_failed_turn_site_codes.py` | `1` | `agent/test_failed_turn_site_codes.py` | - | `tests/agent/test_failed_turn_site_codes.py` |
| `agent/test_failover_identity.py` | `1` | `agent/test_failover_identity.py` | - | `tests/agent/test_failover_identity.py` |
| `agent/test_fallback_429_after_timeout.py` | `1` | `agent/test_fallback_429_after_timeout.py` | - | `tests/agent/test_fallback_429_after_timeout.py` |
| `agent/test_fallback_api_mode_preservation.py` | `1` | `agent/test_fallback_api_mode_preservation.py` | - | `tests/agent/test_fallback_api_mode_preservation.py` |
| `agent/test_fallback_credential_isolation.py` | `1` | `agent/test_fallback_credential_isolation.py` | - | `tests/agent/test_fallback_credential_isolation.py` |
| `agent/test_fallback_dynamic_credential.py` | `1` | `agent/test_fallback_dynamic_credential.py` | - | `tests/agent/test_fallback_dynamic_credential.py` |
| `agent/test_fallback_entry_base_url.py` | `1` | `agent/test_fallback_entry_base_url.py` | - | `tests/agent/test_fallback_entry_base_url.py` |
| `agent/test_fallback_exhaustion_cooldown.py` | `1` | `agent/test_fallback_exhaustion_cooldown.py` | - | `tests/agent/test_fallback_exhaustion_cooldown.py` |
| `agent/test_fallback_reasoning_override.py` | `1` | `agent/test_fallback_reasoning_override.py` | - | `tests/agent/test_fallback_reasoning_override.py` |
| `agent/test_fallback_restore_codex_early_reopen.py` | `1` | `agent/test_fallback_restore_codex_early_reopen.py` | - | `tests/agent/test_fallback_restore_codex_early_reopen.py` |
| `agent/test_fast_compression_lane.py` | `1` | `agent/test_fast_compression_lane.py` | - | `tests/agent/test_fast_compression_lane.py` |
| `agent/test_fast_mode_auto.py` | `1` | `agent/test_fast_mode_auto.py` | - | `tests/agent/test_fast_mode_auto.py` |
| `agent/test_file_mutation_verifier.py` | `1` | `agent/test_file_mutation_verifier.py` | - | `tests/agent/test_file_mutation_verifier.py` |
| `agent/test_file_safety.py` | `1` | `agent/test_file_safety.py` | - | `tests/agent/test_file_safety.py` |
| `agent/test_file_safety_container_mirror.py` | `1` | `agent/test_file_safety_container_mirror.py` | - | `tests/agent/test_file_safety_container_mirror.py` |
| `agent/test_file_safety_credentials.py` | `1` | `agent/test_file_safety_credentials.py` | - | `tests/agent/test_file_safety_credentials.py` |
| `agent/test_file_safety_cross_profile.py` | `1` | `agent/test_file_safety_cross_profile.py` | - | `tests/agent/test_file_safety_cross_profile.py` |
| `agent/test_file_safety_sandbox_mirror.py` | `1` | `agent/test_file_safety_sandbox_mirror.py` | - | `tests/agent/test_file_safety_sandbox_mirror.py` |
| `agent/test_file_safety_session_state.py` | `1` | `agent/test_file_safety_session_state.py` | - | `tests/agent/test_file_safety_session_state.py` |
| `agent/test_file_safety_write_credentials.py` | `1` | `agent/test_file_safety_write_credentials.py` | - | `tests/agent/test_file_safety_write_credentials.py` |
| `agent/test_first_chunk_at_hook.py` | `1` | `agent/test_first_chunk_at_hook.py` | - | `tests/agent/test_first_chunk_at_hook.py` |
| `agent/test_flush_diverts_on_corrupt_state_db.py` | `1` | `agent/test_flush_diverts_on_corrupt_state_db.py` | - | `tests/agent/test_flush_diverts_on_corrupt_state_db.py` |
| `agent/test_flush_uid_reinsert_guard.py` | `1` | `agent/test_flush_uid_reinsert_guard.py` | - | `tests/agent/test_flush_uid_reinsert_guard.py` |
| `agent/test_free_tier_rate_limit_class.py` | `1` | `agent/test_free_tier_rate_limit_class.py` | - | `tests/agent/test_free_tier_rate_limit_class.py` |
| `agent/test_future_timeout_worker_death.py` | `1` | `agent/test_future_timeout_worker_death.py` | - | `tests/agent/test_future_timeout_worker_death.py` |
| `agent/test_gateway_turn_sidecar.py` | `1` | `agent/test_gateway_turn_sidecar.py` | - | `tests/agent/test_gateway_turn_sidecar.py` |
| `agent/test_gemini_alias_native_route.py` | `1` | `agent/test_gemini_alias_native_route.py` | - | `tests/agent/test_gemini_alias_native_route.py` |
| `agent/test_gemini_fast_fallback.py` | `1` | `agent/test_gemini_fast_fallback.py` | - | `tests/agent/test_gemini_fast_fallback.py` |
| `agent/test_gemini_free_tier_gate.py` | `1` | `agent/test_gemini_free_tier_gate.py` | - | `tests/agent/test_gemini_free_tier_gate.py` |
| `agent/test_gemini_key_surface_guidance.py` | `1` | `agent/test_gemini_key_surface_guidance.py` | - | `tests/agent/test_gemini_key_surface_guidance.py` |
| `agent/test_gemini_long_decode.py` | `1` | `agent/test_gemini_long_decode.py` | - | `tests/agent/test_gemini_long_decode.py` |
| `agent/test_gemini_native_adapter.py` | `1` | `agent/test_gemini_native_adapter.py` | - | `tests/agent/test_gemini_native_adapter.py` |
| `agent/test_gemini_native_reservation.py` | `1` | `agent/test_gemini_native_reservation.py` | - | `tests/agent/test_gemini_native_reservation.py` |
| `agent/test_gemini_schema.py` | `1` | `agent/test_gemini_schema.py` | - | `tests/agent/test_gemini_schema.py` |
| `agent/test_gemini_standard_key_guidance.py` | `1` | `agent/test_gemini_standard_key_guidance.py` | - | `tests/agent/test_gemini_standard_key_guidance.py` |
| `agent/test_gemini_thinking_config.py` | `1` | `agent/test_gemini_thinking_config.py` | - | `tests/agent/test_gemini_thinking_config.py` |
| `agent/test_gemini_type_arrays.py` | `1` | `agent/test_gemini_type_arrays.py` | - | `tests/agent/test_gemini_type_arrays.py` |
| `agent/test_generated_media_durable.py` | `1` | `agent/test_generated_media_durable.py` | - | `tests/agent/test_generated_media_durable.py` |
| `agent/test_ghost_skill_pruning.py` | `1` | `agent/test_ghost_skill_pruning.py` | - | `tests/agent/test_ghost_skill_pruning.py` |
| `agent/test_glm_context_window_resolution.py` | `1` | `agent/test_glm_context_window_resolution.py` | - | `tests/agent/test_glm_context_window_resolution.py` |
| `agent/test_grid_sweep_cli_tui_stream_diagnostics.py` | `1` | `agent/test_grid_sweep_cli_tui_stream_diagnostics.py` | - | `tests/agent/test_grid_sweep_cli_tui_stream_diagnostics.py` |
| `agent/test_hermetic_side_effect_guards.py` | `1` | `agent/test_hermetic_side_effect_guards.py` | - | `tests/agent/test_hermetic_side_effect_guards.py` |
| `agent/test_hold_stream_agent.py` | `1` | `agent/test_hold_stream_agent.py` | - | `tests/agent/test_hold_stream_agent.py` |
| `agent/test_hold_stream_edges.py` | `1` | `agent/test_hold_stream_edges.py` | - | `tests/agent/test_hold_stream_edges.py` |
| `agent/test_hold_stream_exits.py` | `1` | `agent/test_hold_stream_exits.py` | - | `tests/agent/test_hold_stream_exits.py` |
| `agent/test_hygiene_timeout_cooldown_isolation.py` | `1` | `agent/test_hygiene_timeout_cooldown_isolation.py` | - | `tests/agent/test_hygiene_timeout_cooldown_isolation.py` |
| `agent/test_i18n_languages.py` | `1` | `agent/test_i18n_languages.py` | - | `tests/agent/test_i18n_languages.py` |
| `agent/test_i18n_layers.py` | `1` | `agent/test_i18n_layers.py` | - | `tests/agent/test_i18n_layers.py` |
| `agent/test_i18n_tui_packs.py` | `1` | `agent/test_i18n_tui_packs.py` | - | `tests/agent/test_i18n_tui_packs.py` |
| `agent/test_identity_flush.py` | `1` | `agent/test_identity_flush.py` | - | `tests/agent/test_identity_flush.py` |
| `agent/test_idle_compaction.py` | `1` | `agent/test_idle_compaction.py` | - | `tests/agent/test_idle_compaction.py` |
| `agent/test_idle_compaction_lock_and_guards.py` | `1` | `agent/test_idle_compaction_lock_and_guards.py` | - | `tests/agent/test_idle_compaction_lock_and_guards.py` |
| `agent/test_image_corrupt_recovery.py` | `1` | `agent/test_image_corrupt_recovery.py` | - | `tests/agent/test_image_corrupt_recovery.py` |
| `agent/test_image_eviction_policy.py` | `1` | `agent/test_image_eviction_policy.py` | - | `tests/agent/test_image_eviction_policy.py` |
| `agent/test_image_gen_registry.py` | `1` | `agent/test_image_gen_registry.py` | - | `tests/agent/test_image_gen_registry.py` |
| `agent/test_image_generate_parallel.py` | `1` | `agent/test_image_generate_parallel.py` | - | `tests/agent/test_image_generate_parallel.py` |
| `agent/test_image_rejection_fallback.py` | `1` | `agent/test_image_rejection_fallback.py` | - | `tests/agent/test_image_rejection_fallback.py` |
| `agent/test_image_replay.py` | `1` | `agent/test_image_replay.py` | - | `tests/agent/test_image_replay.py` |
| `agent/test_image_routing.py` | `1` | `agent/test_image_routing.py` | - | `tests/agent/test_image_routing.py` |
| `agent/test_image_shrink_recovery.py` | `1` | `agent/test_image_shrink_recovery.py` | - | `tests/agent/test_image_shrink_recovery.py` |
| `agent/test_image_token_cost.py` | `1` | `agent/test_image_token_cost.py` | - | `tests/agent/test_image_token_cost.py` |
| `agent/test_in_place_commit_rollback.py` | `1` | `agent/test_in_place_commit_rollback.py` | - | `tests/agent/test_in_place_commit_rollback.py` |
| `agent/test_in_place_compaction.py` | `1` | `agent/test_in_place_compaction.py` | - | `tests/agent/test_in_place_compaction.py` |
| `agent/test_in_place_merged_user_rows.py` | `1` | `agent/test_in_place_merged_user_rows.py` | - | `tests/agent/test_in_place_merged_user_rows.py` |
| `agent/test_in_place_persist_marker.py` | `1` | `agent/test_in_place_persist_marker.py` | - | `tests/agent/test_in_place_persist_marker.py` |
| `agent/test_in_place_preflight_rewind.py` | `1` | `agent/test_in_place_preflight_rewind.py` | - | `tests/agent/test_in_place_preflight_rewind.py` |
| `agent/test_infinite_compaction_loop.py` | `1` | `agent/test_infinite_compaction_loop.py` | - | `tests/agent/test_infinite_compaction_loop.py` |
| `agent/test_init_exhausted_pool_billing_error.py` | `1` | `agent/test_init_exhausted_pool_billing_error.py` | - | `tests/agent/test_init_exhausted_pool_billing_error.py` |
| `agent/test_init_fallback_on_exhausted_pool.py` | `1` | `agent/test_init_fallback_on_exhausted_pool.py` | - | `tests/agent/test_init_fallback_on_exhausted_pool.py` |
| `agent/test_init_fallback_refusal_warning.py` | `1` | `agent/test_init_fallback_refusal_warning.py` | - | `tests/agent/test_init_fallback_refusal_warning.py` |
| `agent/test_injected_param_strip_retry_registry.py` | `1` | `agent/test_injected_param_strip_retry_registry.py` | - | `tests/agent/test_injected_param_strip_retry_registry.py` |
| `agent/test_inline_edit_persistence.py` | `1` | `agent/test_inline_edit_persistence.py` | - | `tests/agent/test_inline_edit_persistence.py` |
| `agent/test_inline_session_search_profile.py` | `1` | `agent/test_inline_session_search_profile.py` | - | `tests/agent/test_inline_session_search_profile.py` |
| `agent/test_inline_tool_executors_memory_args.py` | `1` | `agent/test_inline_tool_executors_memory_args.py` | - | `tests/agent/test_inline_tool_executors_memory_args.py` |
| `agent/test_insights.py` | `1` | `agent/test_insights.py` | - | `tests/agent/test_insights.py` |
| `agent/test_intent_ack_continuation.py` | `1` | `agent/test_intent_ack_continuation.py` | - | `tests/agent/test_intent_ack_continuation.py` |
| `agent/test_interpreter_shutdown_turn_exit.py` | `1` | `agent/test_interpreter_shutdown_turn_exit.py` | - | `tests/agent/test_interpreter_shutdown_turn_exit.py` |
| `agent/test_interrupt_compat.py` | `1` | `agent/test_interrupt_compat.py` | - | `tests/agent/test_interrupt_compat.py` |
| `agent/test_interrupt_issuer_attribution.py` | `1` | `agent/test_interrupt_issuer_attribution.py` | - | `tests/agent/test_interrupt_issuer_attribution.py` |
| `agent/test_interrupt_propagation.py` | `1` | `agent/test_interrupt_propagation.py` | - | `tests/agent/test_interrupt_propagation.py` |
| `agent/test_invalid_context_length_warning.py` | `1` | `agent/test_invalid_context_length_warning.py` | - | `tests/agent/test_invalid_context_length_warning.py` |
| `agent/test_iron_proxy.py` | `1` | `agent/test_iron_proxy.py` | - | `tests/agent/test_iron_proxy.py` |
| `agent/test_iron_proxy_e2e.py` | `1` | `agent/test_iron_proxy_e2e.py` | - | `tests/agent/test_iron_proxy_e2e.py` |
| `agent/test_iteration_budget_race.py` | `1` | `agent/test_iteration_budget_race.py` | - | `tests/agent/test_iteration_budget_race.py` |
| `agent/test_iteration_budget_warning.py` | `1` | `agent/test_iteration_budget_warning.py` | - | `tests/agent/test_iteration_budget_warning.py` |
| `agent/test_iteration_summary_reasoning_details.py` | `1` | `agent/test_iteration_summary_reasoning_details.py` | - | `tests/agent/test_iteration_summary_reasoning_details.py` |
| `agent/test_jiter_preload.py` | `1` | `agent/test_jiter_preload.py` | - | `tests/agent/test_jiter_preload.py` |
| `agent/test_kanban_budget_checkpoint.py` | `1` | `agent/test_kanban_budget_checkpoint.py` | - | `tests/agent/test_kanban_budget_checkpoint.py` |
| `agent/test_kanban_stop.py` | `1` | `agent/test_kanban_stop.py` | - | `tests/agent/test_kanban_stop.py` |
| `agent/test_kanban_turn_recovery.py` | `1` | `agent/test_kanban_turn_recovery.py` | - | `tests/agent/test_kanban_turn_recovery.py` |
| `agent/test_kimi_coding_anthropic_thinking.py` | `1` | `agent/test_kimi_coding_anthropic_thinking.py` | - | `tests/agent/test_kimi_coding_anthropic_thinking.py` |
| `agent/test_kimi_coding_openai_base_url.py` | `1` | `agent/test_kimi_coding_openai_base_url.py` | - | `tests/agent/test_kimi_coding_openai_base_url.py` |
| `agent/test_last_total_tokens.py` | `1` | `agent/test_last_total_tokens.py` | - | `tests/agent/test_last_total_tokens.py` |
| `agent/test_lean_single_aux_call.py` | `1` | `agent/test_lean_single_aux_call.py` | - | `tests/agent/test_lean_single_aux_call.py` |
| `agent/test_learn_prompt.py` | `1` | `agent/test_learn_prompt.py` | - | `tests/agent/test_learn_prompt.py` |
| `agent/test_learning_graph.py` | `1` | `agent/test_learning_graph.py` | - | `tests/agent/test_learning_graph.py` |
| `agent/test_learning_graph_render.py` | `1` | `agent/test_learning_graph_render.py` | - | `tests/agent/test_learning_graph_render.py` |
| `agent/test_learning_mutations.py` | `1` | `agent/test_learning_mutations.py` | - | `tests/agent/test_learning_mutations.py` |
| `agent/test_left_core_kept_notice_delivery.py` | `1` | `agent/test_left_core_kept_notice_delivery.py` | - | `tests/agent/test_left_core_kept_notice_delivery.py` |
| `agent/test_legacy_cli.py` | `1` | `agent/test_legacy_cli.py` | - | `tests/agent/test_legacy_cli.py` |
| `agent/test_length_continuation_thinking_exhaustion.py` | `1` | `agent/test_length_continuation_thinking_exhaustion.py` | - | `tests/agent/test_length_continuation_thinking_exhaustion.py` |
| `agent/test_length_continuation_window_headroom.py` | `1` | `agent/test_length_continuation_window_headroom.py` | - | `tests/agent/test_length_continuation_window_headroom.py` |
| `agent/test_litellm_tool_part_cache_markers.py` | `1` | `agent/test_litellm_tool_part_cache_markers.py` | - | `tests/agent/test_litellm_tool_part_cache_markers.py` |
| `agent/test_live_db_preview_locks.py` | `1` | `agent/test_live_db_preview_locks.py` | - | `tests/agent/test_live_db_preview_locks.py` |
| `agent/test_llamacpp_alias_managed_endpoint.py` | `1` | `agent/test_llamacpp_alias_managed_endpoint.py` | - | `tests/agent/test_llamacpp_alias_managed_endpoint.py` |
| `agent/test_lmstudio_load_mode.py` | `1` | `agent/test_lmstudio_load_mode.py` | - | `tests/agent/test_lmstudio_load_mode.py` |
| `agent/test_lmstudio_reasoning.py` | `1` | `agent/test_lmstudio_reasoning.py` | - | `tests/agent/test_lmstudio_reasoning.py` |
| `agent/test_load_time_durability_stamp.py` | `1` | `agent/test_load_time_durability_stamp.py` | - | `tests/agent/test_load_time_durability_stamp.py` |
| `agent/test_local_probe_disk_cache.py` | `1` | `agent/test_local_probe_disk_cache.py` | - | `tests/agent/test_local_probe_disk_cache.py` |
| `agent/test_local_prompt_progress.py` | `1` | `agent/test_local_prompt_progress.py` | - | `tests/agent/test_local_prompt_progress.py` |
| `agent/test_local_stream_timeout.py` | `1` | `agent/test_local_stream_timeout.py` | - | `tests/agent/test_local_stream_timeout.py` |
| `agent/test_local_tool_batches.py` | `1` | `agent/test_local_tool_batches.py` | - | `tests/agent/test_local_tool_batches.py` |
| `agent/test_lock_fallback_base_semantics.py` | `1` | `agent/test_lock_fallback_base_semantics.py` | - | `tests/agent/test_lock_fallback_base_semantics.py` |
| `agent/test_long_lived_cache_bounds.py` | `1` | `agent/test_long_lived_cache_bounds.py` | - | `tests/agent/test_long_lived_cache_bounds.py` |
| `agent/test_malformed_tool_args_no_fallback.py` | `1` | `agent/test_malformed_tool_args_no_fallback.py` | - | `tests/agent/test_malformed_tool_args_no_fallback.py` |
| `agent/test_malformed_tool_arguments.py` | `1` | `agent/test_malformed_tool_arguments.py` | - | `tests/agent/test_malformed_tool_arguments.py` |
| `agent/test_manual_compression_feedback.py` | `1` | `agent/test_manual_compression_feedback.py` | - | `tests/agent/test_manual_compression_feedback.py` |
| `agent/test_manual_compression_refusal_feedback.py` | `1` | `agent/test_manual_compression_refusal_feedback.py` | - | `tests/agent/test_manual_compression_refusal_feedback.py` |
| `agent/test_markdown_tables.py` | `1` | `agent/test_markdown_tables.py` | - | `tests/agent/test_markdown_tables.py` |
| `agent/test_materialize_data_url_cleanup.py` | `1` | `agent/test_materialize_data_url_cleanup.py` | - | `tests/agent/test_materialize_data_url_cleanup.py` |
| `agent/test_max_iteration_handoff.py` | `1` | `agent/test_max_iteration_handoff.py` | - | `tests/agent/test_max_iteration_handoff.py` |
| `agent/test_mcp_execution_deadline.py` | `1` | `agent/test_mcp_execution_deadline.py` | - | `tests/agent/test_mcp_execution_deadline.py` |
| `agent/test_memory_agent_context.py` | `1` | `agent/test_memory_agent_context.py` | - | `tests/agent/test_memory_agent_context.py` |
| `agent/test_memory_async_sync.py` | `1` | `agent/test_memory_async_sync.py` | - | `tests/agent/test_memory_async_sync.py` |
| `agent/test_memory_boundary_commit.py` | `1` | `agent/test_memory_boundary_commit.py` | - | `tests/agent/test_memory_boundary_commit.py` |
| `agent/test_memory_context_dedupe.py` | `1` | `agent/test_memory_context_dedupe.py` | - | `tests/agent/test_memory_context_dedupe.py` |
| `agent/test_memory_migration_notifications.py` | `1` | `agent/test_memory_migration_notifications.py` | - | `tests/agent/test_memory_migration_notifications.py` |
| `agent/test_memory_provider.py` | `1` | `agent/test_memory_provider.py` | - | `tests/agent/test_memory_provider.py` |
| `agent/test_memory_provider_caller_profile.py` | `1` | `agent/test_memory_provider_caller_profile.py` | - | `tests/agent/test_memory_provider_caller_profile.py` |
| `agent/test_memory_provider_init.py` | `1` | `agent/test_memory_provider_init.py` | - | `tests/agent/test_memory_provider_init.py` |
| `agent/test_memory_provider_unavailable_warning.py` | `1` | `agent/test_memory_provider_unavailable_warning.py` | - | `tests/agent/test_memory_provider_unavailable_warning.py` |
| `agent/test_memory_recall_indicator.py` | `1` | `agent/test_memory_recall_indicator.py` | - | `tests/agent/test_memory_recall_indicator.py` |
| `agent/test_memory_review_grounding.py` | `1` | `agent/test_memory_review_grounding.py` | - | `tests/agent/test_memory_review_grounding.py` |
| `agent/test_memory_session_switch.py` | `1` | `agent/test_memory_session_switch.py` | - | `tests/agent/test_memory_session_switch.py` |
| `agent/test_memory_skill_scaffolding.py` | `1` | `agent/test_memory_skill_scaffolding.py` | - | `tests/agent/test_memory_skill_scaffolding.py` |
| `agent/test_memory_stalled_background_call.py` | `1` | `agent/test_memory_stalled_background_call.py` | - | `tests/agent/test_memory_stalled_background_call.py` |
| `agent/test_memory_sync_interrupted.py` | `1` | `agent/test_memory_sync_interrupted.py` | - | `tests/agent/test_memory_sync_interrupted.py` |
| `agent/test_memory_sync_redaction.py` | `1` | `agent/test_memory_sync_redaction.py` | - | `tests/agent/test_memory_sync_redaction.py` |
| `agent/test_memory_user_id.py` | `1` | `agent/test_memory_user_id.py` | - | `tests/agent/test_memory_user_id.py` |
| `agent/test_memory_write_bridge.py` | `1` | `agent/test_memory_write_bridge.py` | - | `tests/agent/test_memory_write_bridge.py` |
| `agent/test_merged_user_override.py` | `1` | `agent/test_merged_user_override.py` | - | `tests/agent/test_merged_user_override.py` |
| `agent/test_message_content.py` | `1` | `agent/test_message_content.py` | - | `tests/agent/test_message_content.py` |
| `agent/test_message_metadata.py` | `1` | `agent/test_message_metadata.py` | - | `tests/agent/test_message_metadata.py` |
| `agent/test_message_reactions.py` | `1` | `agent/test_message_reactions.py` | - | `tests/agent/test_message_reactions.py` |
| `agent/test_message_sanitization_policy.py` | `1` | `agent/test_message_sanitization_policy.py` | - | `tests/agent/test_message_sanitization_policy.py` |
| `agent/test_message_sequence_repair.py` | `1` | `agent/test_message_sequence_repair.py` | - | `tests/agent/test_message_sequence_repair.py` |
| `agent/test_meta_agent_init.py` | `1` | `agent/test_meta_agent_init.py` | - | `tests/agent/test_meta_agent_init.py` |
| `agent/test_meta_usage_cache_reporting.py` | `1` | `agent/test_meta_usage_cache_reporting.py` | - | `tests/agent/test_meta_usage_cache_reporting.py` |
| `agent/test_micro_compaction.py` | `1` | `agent/test_micro_compaction.py` | - | `tests/agent/test_micro_compaction.py` |
| `agent/test_mid_turn_compaction_turn_boundary.py` | `1` | `agent/test_mid_turn_compaction_turn_boundary.py` | - | `tests/agent/test_mid_turn_compaction_turn_boundary.py` |
| `agent/test_minimax_auxiliary_url.py` | `1` | `agent/test_minimax_auxiliary_url.py` | - | `tests/agent/test_minimax_auxiliary_url.py` |
| `agent/test_minimax_provider.py` | `1` | `agent/test_minimax_provider.py` | - | `tests/agent/test_minimax_provider.py` |
| `agent/test_missing_provider_credentials_hint.py` | `1` | `agent/test_missing_provider_credentials_hint.py` | - | `tests/agent/test_missing_provider_credentials_hint.py` |
| `agent/test_mistral_tool_ids.py` | `1` | `agent/test_mistral_tool_ids.py` | - | `tests/agent/test_mistral_tool_ids.py` |
| `agent/test_moa_aggregator_cache_control.py` | `1` | `agent/test_moa_aggregator_cache_control.py` | - | `tests/agent/test_moa_aggregator_cache_control.py` |
| `agent/test_moa_aggregator_cost_slot.py` | `1` | `agent/test_moa_aggregator_cost_slot.py` | - | `tests/agent/test_moa_aggregator_cost_slot.py` |
| `agent/test_moa_aggregator_prefix_stability.py` | `1` | `agent/test_moa_aggregator_prefix_stability.py` | - | `tests/agent/test_moa_aggregator_prefix_stability.py` |
| `agent/test_moa_alternation_recovery.py` | `1` | `agent/test_moa_alternation_recovery.py` | - | `tests/agent/test_moa_alternation_recovery.py` |
| `agent/test_moa_cold_start_cache.py` | `1` | `agent/test_moa_cold_start_cache.py` | - | `tests/agent/test_moa_cold_start_cache.py` |
| `agent/test_moa_context_max_tokens.py` | `1` | `agent/test_moa_context_max_tokens.py` | - | `tests/agent/test_moa_context_max_tokens.py` |
| `agent/test_moa_fanout_cadence.py` | `1` | `agent/test_moa_fanout_cadence.py` | - | `tests/agent/test_moa_fanout_cadence.py` |
| `agent/test_moa_fanout_liveness.py` | `1` | `agent/test_moa_fanout_liveness.py` | - | `tests/agent/test_moa_fanout_liveness.py` |
| `agent/test_moa_loop_mode.py` | `1` | `agent/test_moa_loop_mode.py` | - | `tests/agent/test_moa_loop_mode.py` |
| `agent/test_moa_observability_bridge.py` | `1` | `agent/test_moa_observability_bridge.py` | - | `tests/agent/test_moa_observability_bridge.py` |
| `agent/test_moa_prepared_request_client_swap.py` | `1` | `agent/test_moa_prepared_request_client_swap.py` | - | `tests/agent/test_moa_prepared_request_client_swap.py` |
| `agent/test_moa_prepared_request_leak.py` | `1` | `agent/test_moa_prepared_request_leak.py` | - | `tests/agent/test_moa_prepared_request_leak.py` |
| `agent/test_moa_privacy_filter.py` | `1` | `agent/test_moa_privacy_filter.py` | - | `tests/agent/test_moa_privacy_filter.py` |
| `agent/test_moa_progress.py` | `1` | `agent/test_moa_progress.py` | - | `tests/agent/test_moa_progress.py` |
| `agent/test_moa_quiet_reference_output.py` | `1` | `agent/test_moa_quiet_reference_output.py` | - | `tests/agent/test_moa_quiet_reference_output.py` |
| `agent/test_moa_reasoning_effort.py` | `1` | `agent/test_moa_reasoning_effort.py` | - | `tests/agent/test_moa_reasoning_effort.py` |
| `agent/test_moa_slot_api_mode.py` | `1` | `agent/test_moa_slot_api_mode.py` | - | `tests/agent/test_moa_slot_api_mode.py` |
| `agent/test_moa_slot_max_tokens.py` | `1` | `agent/test_moa_slot_max_tokens.py` | - | `tests/agent/test_moa_slot_max_tokens.py` |
| `agent/test_moa_slot_request_headers.py` | `1` | `agent/test_moa_slot_request_headers.py` | - | `tests/agent/test_moa_slot_request_headers.py` |
| `agent/test_moa_stale_guidance_note.py` | `1` | `agent/test_moa_stale_guidance_note.py` | - | `tests/agent/test_moa_stale_guidance_note.py` |
| `agent/test_moa_streaming.py` | `1` | `agent/test_moa_streaming.py` | - | `tests/agent/test_moa_streaming.py` |
| `agent/test_moa_switch_api_mode.py` | `1` | `agent/test_moa_switch_api_mode.py` | - | `tests/agent/test_moa_switch_api_mode.py` |
| `agent/test_moa_trace_streamed_capture.py` | `1` | `agent/test_moa_trace_streamed_capture.py` | - | `tests/agent/test_moa_trace_streamed_capture.py` |
| `agent/test_model_extra_type_guard.py` | `1` | `agent/test_model_extra_type_guard.py` | - | `tests/agent/test_model_extra_type_guard.py` |
| `agent/test_model_metadata.py` | `1` | `agent/test_model_metadata.py` | - | `tests/agent/test_model_metadata.py` |
| `agent/test_model_metadata_cache.py` | `1` | `agent/test_model_metadata_cache.py` | - | `tests/agent/test_model_metadata_cache.py` |
| `agent/test_model_metadata_http.py` | `1` | `agent/test_model_metadata_http.py` | - | `tests/agent/test_model_metadata_http.py` |
| `agent/test_model_metadata_local_ctx.py` | `1` | `agent/test_model_metadata_local_ctx.py` | - | `tests/agent/test_model_metadata_local_ctx.py` |
| `agent/test_model_metadata_probe_deadline.py` | `1` | `agent/test_model_metadata_probe_deadline.py` | - | `tests/agent/test_model_metadata_probe_deadline.py` |
| `agent/test_model_streaming_config.py` | `1` | `agent/test_model_streaming_config.py` | - | `tests/agent/test_model_streaming_config.py` |
| `agent/test_models_dev.py` | `1` | `agent/test_models_dev.py` | - | `tests/agent/test_models_dev.py` |
| `agent/test_models_dev_fetch_deadline.py` | `1` | `agent/test_models_dev_fetch_deadline.py` | - | `tests/agent/test_models_dev_fetch_deadline.py` |
| `agent/test_models_dev_provider_capabilities.py` | `1` | `agent/test_models_dev_provider_capabilities.py` | - | `tests/agent/test_models_dev_provider_capabilities.py` |
| `agent/test_moonshot_schema.py` | `1` | `agent/test_moonshot_schema.py` | - | `tests/agent/test_moonshot_schema.py` |
| `agent/test_multimodal_interim.py` | `1` | `agent/test_multimodal_interim.py` | - | `tests/agent/test_multimodal_interim.py` |
| `agent/test_multimodal_tool_content_recovery.py` | `1` | `agent/test_multimodal_tool_content_recovery.py` | - | `tests/agent/test_multimodal_tool_content_recovery.py` |
| `agent/test_multimodal_tool_result_spill.py` | `1` | `agent/test_multimodal_tool_result_spill.py` | - | `tests/agent/test_multimodal_tool_result_spill.py` |
| `agent/test_multiplex_base_url_scope.py` | `1` | `agent/test_multiplex_base_url_scope.py` | - | `tests/agent/test_multiplex_base_url_scope.py` |
| `agent/test_multiplex_cloud_credential_clients.py` | `1` | `agent/test_multiplex_cloud_credential_clients.py` | - | `tests/agent/test_multiplex_cloud_credential_clients.py` |
| `agent/test_native_compaction.py` | `1` | `agent/test_native_compaction.py` | - | `tests/agent/test_native_compaction.py` |
| `agent/test_native_compaction_nontext_retention.py` | `1` | `agent/test_native_compaction_nontext_retention.py` | - | `tests/agent/test_native_compaction_nontext_retention.py` |
| `agent/test_native_compaction_summary_retention.py` | `1` | `agent/test_native_compaction_summary_retention.py` | - | `tests/agent/test_native_compaction_summary_retention.py` |
| `agent/test_native_compaction_switch_capabilities.py` | `1` | `agent/test_native_compaction_switch_capabilities.py` | - | `tests/agent/test_native_compaction_switch_capabilities.py` |
| `agent/test_native_compaction_typed_user_retention.py` | `1` | `agent/test_native_compaction_typed_user_retention.py` | - | `tests/agent/test_native_compaction_typed_user_retention.py` |
| `agent/test_native_preflight_estimate.py` | `1` | `agent/test_native_preflight_estimate.py` | - | `tests/agent/test_native_preflight_estimate.py` |
| `agent/test_non_chat_primary_restore.py` | `1` | `agent/test_non_chat_primary_restore.py` | - | `tests/agent/test_non_chat_primary_restore.py` |
| `agent/test_non_stream_stale_timeout.py` | `1` | `agent/test_non_stream_stale_timeout.py` | - | `tests/agent/test_non_stream_stale_timeout.py` |
| `agent/test_none_deref_guards.py` | `1` | `agent/test_none_deref_guards.py` | - | `tests/agent/test_none_deref_guards.py` |
| `agent/test_nonretryable_error_html_summary.py` | `1` | `agent/test_nonretryable_error_html_summary.py` | - | `tests/agent/test_nonretryable_error_html_summary.py` |
| `agent/test_nonretryable_result_carries_verdict.py` | `1` | `agent/test_nonretryable_result_carries_verdict.py` | - | `tests/agent/test_nonretryable_result_carries_verdict.py` |
| `agent/test_nonstream_wait_notice.py` | `1` | `agent/test_nonstream_wait_notice.py` | - | `tests/agent/test_nonstream_wait_notice.py` |
| `agent/test_notice_spine.py` | `1` | `agent/test_notice_spine.py` | - | `tests/agent/test_notice_spine.py` |
| `agent/test_notification_persistence.py` | `1` | `agent/test_notification_persistence.py` | - | `tests/agent/test_notification_persistence.py` |
| `agent/test_nous_credits_gauge.py` | `1` | `agent/test_nous_credits_gauge.py` | - | `tests/agent/test_nous_credits_gauge.py` |
| `agent/test_nous_credits_snapshot.py` | `1` | `agent/test_nous_credits_snapshot.py` | - | `tests/agent/test_nous_credits_snapshot.py` |
| `agent/test_nous_fallback_unavailable.py` | `1` | `agent/test_nous_fallback_unavailable.py` | - | `tests/agent/test_nous_fallback_unavailable.py` |
| `agent/test_nous_key_pre_expiry_adoption.py` | `1` | `agent/test_nous_key_pre_expiry_adoption.py` | - | `tests/agent/test_nous_key_pre_expiry_adoption.py` |
| `agent/test_nous_portal_anthropic_wire.py` | `1` | `agent/test_nous_portal_anthropic_wire.py` | - | `tests/agent/test_nous_portal_anthropic_wire.py` |
| `agent/test_nous_rate_guard.py` | `1` | `agent/test_nous_rate_guard.py` | - | `tests/agent/test_nous_rate_guard.py` |
| `agent/test_nous_transient_401_interrupt.py` | `1` | `agent/test_nous_transient_401_interrupt.py` | - | `tests/agent/test_nous_transient_401_interrupt.py` |
| `agent/test_nous_usage_profile_context.py` | `1` | `agent/test_nous_usage_profile_context.py` | - | `tests/agent/test_nous_usage_profile_context.py` |
| `agent/test_nous_welcome_client_contract.py` | `1` | `agent/test_nous_welcome_client_contract.py` | - | `tests/agent/test_nous_welcome_client_contract.py` |
| `agent/test_nous_wire_auto.py` | `1` | `agent/test_nous_wire_auto.py` | - | `tests/agent/test_nous_wire_auto.py` |
| `agent/test_nt_namespace_guard.py` | `1` | `agent/test_nt_namespace_guard.py` | - | `tests/agent/test_nt_namespace_guard.py` |
| `agent/test_ollama_num_ctx.py` | `1` | `agent/test_ollama_num_ctx.py` | - | `tests/agent/test_ollama_num_ctx.py` |
| `agent/test_onboarding.py` | `1` | `agent/test_onboarding.py` | - | `tests/agent/test_onboarding.py` |
| `agent/test_onboarding_first_tasks.py` | `1` | `agent/test_onboarding_first_tasks.py` | - | `tests/agent/test_onboarding_first_tasks.py` |
| `agent/test_onboarding_flow.py` | `1` | `agent/test_onboarding_flow.py` | - | `tests/agent/test_onboarding_flow.py` |
| `agent/test_onepassword_secrets.py` | `1` | `agent/test_onepassword_secrets.py` | - | `tests/agent/test_onepassword_secrets.py` |
| `agent/test_oneshot.py` | `1` | `agent/test_oneshot.py` | - | `tests/agent/test_oneshot.py` |
| `agent/test_oneshot_footprint.py` | `1` | `agent/test_oneshot_footprint.py` | - | `tests/agent/test_oneshot_footprint.py` |
| `agent/test_oneshot_interactive_surface.py` | `1` | `agent/test_oneshot_interactive_surface.py` | - | `tests/agent/test_oneshot_interactive_surface.py` |
| `agent/test_openai_client_lifecycle.py` | `1` | `agent/test_openai_client_lifecycle.py` | - | `tests/agent/test_openai_client_lifecycle.py` |
| `agent/test_openai_ultrafast.py` | `1` | `agent/test_openai_ultrafast.py` | - | `tests/agent/test_openai_ultrafast.py` |
| `agent/test_opencode_session_affinity.py` | `1` | `agent/test_opencode_session_affinity.py` | - | `tests/agent/test_opencode_session_affinity.py` |
| `agent/test_openrouter_response_cache.py` | `1` | `agent/test_openrouter_response_cache.py` | - | `tests/agent/test_openrouter_response_cache.py` |
| `agent/test_openrouter_silent_default_purge.py` | `1` | `agent/test_openrouter_silent_default_purge.py` | - | `tests/agent/test_openrouter_silent_default_purge.py` |
| `agent/test_outbound_stale_vision.py` | `1` | `agent/test_outbound_stale_vision.py` | - | `tests/agent/test_outbound_stale_vision.py` |
| `agent/test_outbound_webhooks.py` | `1` | `agent/test_outbound_webhooks.py` | - | `tests/agent/test_outbound_webhooks.py` |
| `agent/test_outer_error_retry_bound.py` | `1` | `agent/test_outer_error_retry_bound.py` | - | `tests/agent/test_outer_error_retry_bound.py` |
| `agent/test_output_cap_parsing.py` | `1` | `agent/test_output_cap_parsing.py` | - | `tests/agent/test_output_cap_parsing.py` |
| `agent/test_over_window_compression_fail_closed.py` | `1` | `agent/test_over_window_compression_fail_closed.py` | - | `tests/agent/test_over_window_compression_fail_closed.py` |
| `agent/test_overflow_overhead_aware_tokens.py` | `1` | `agent/test_overflow_overhead_aware_tokens.py` | - | `tests/agent/test_overflow_overhead_aware_tokens.py` |
| `agent/test_overflow_partial_terminal.py` | `1` | `agent/test_overflow_partial_terminal.py` | - | `tests/agent/test_overflow_partial_terminal.py` |
| `agent/test_owned_process_cleanup.py` | `1` | `agent/test_owned_process_cleanup.py` | - | `tests/agent/test_owned_process_cleanup.py` |
| `agent/test_partial_stream_continuation_prompt.py` | `1` | `agent/test_partial_stream_continuation_prompt.py` | - | `tests/agent/test_partial_stream_continuation_prompt.py` |
| `agent/test_partial_stream_finish_reason.py` | `1` | `agent/test_partial_stream_finish_reason.py` | - | `tests/agent/test_partial_stream_finish_reason.py` |
| `agent/test_per_model_compression_threshold.py` | `1` | `agent/test_per_model_compression_threshold.py` | - | `tests/agent/test_per_model_compression_threshold.py` |
| `agent/test_per_model_provider_routing.py` | `1` | `agent/test_per_model_provider_routing.py` | - | `tests/agent/test_per_model_provider_routing.py` |
| `agent/test_per_model_threshold_init_ordering.py` | `1` | `agent/test_per_model_threshold_init_ordering.py` | - | `tests/agent/test_per_model_threshold_init_ordering.py` |
| `agent/test_percentage_clamp.py` | `1` | `agent/test_percentage_clamp.py` | - | `tests/agent/test_percentage_clamp.py` |
| `agent/test_periodic_scheduler.py` | `1` | `agent/test_periodic_scheduler.py` | - | `tests/agent/test_periodic_scheduler.py` |
| `agent/test_periodic_scheduler_context.py` | `1` | `agent/test_periodic_scheduler_context.py` | - | `tests/agent/test_periodic_scheduler_context.py` |
| `agent/test_pet_engine.py` | `1` | `agent/test_pet_engine.py` | - | `tests/agent/test_pet_engine.py` |
| `agent/test_phantom_tool_references.py` | `1` | `agent/test_phantom_tool_references.py` | - | `tests/agent/test_phantom_tool_references.py` |
| `agent/test_plan_limits.py` | `1` | `agent/test_plan_limits.py` | - | `tests/agent/test_plan_limits.py` |
| `agent/test_plan_prompt.py` | `1` | `agent/test_plan_prompt.py` | - | `tests/agent/test_plan_prompt.py` |
| `agent/test_plan_snapshot_sync.py` | `1` | `agent/test_plan_snapshot_sync.py` | - | `tests/agent/test_plan_snapshot_sync.py` |
| `agent/test_platform_hint_cron_delivery.py` | `1` | `agent/test_platform_hint_cron_delivery.py` | - | `tests/agent/test_platform_hint_cron_delivery.py` |
| `agent/test_platform_hint_desktop.py` | `1` | `agent/test_platform_hint_desktop.py` | - | `tests/agent/test_platform_hint_desktop.py` |
| `agent/test_platform_hint_overrides.py` | `1` | `agent/test_platform_hint_overrides.py` | - | `tests/agent/test_platform_hint_overrides.py` |
| `agent/test_plugin_context_engine_clone.py` | `1` | `agent/test_plugin_context_engine_clone.py` | - | `tests/agent/test_plugin_context_engine_clone.py` |
| `agent/test_plugin_context_engine_init.py` | `1` | `agent/test_plugin_context_engine_init.py` | - | `tests/agent/test_plugin_context_engine_init.py` |
| `agent/test_plugin_context_references.py` | `1` | `agent/test_plugin_context_references.py` | - | `tests/agent/test_plugin_context_references.py` |
| `agent/test_plugin_live_notes_delivery.py` | `1` | `agent/test_plugin_live_notes_delivery.py` | - | `tests/agent/test_plugin_live_notes_delivery.py` |
| `agent/test_plugin_llm.py` | `1` | `agent/test_plugin_llm.py` | - | `tests/agent/test_plugin_llm.py` |
| `agent/test_plugin_llm_task_routing.py` | `1` | `agent/test_plugin_llm_task_routing.py` | - | `tests/agent/test_plugin_llm_task_routing.py` |
| `agent/test_plugin_prompt_sections.py` | `1` | `agent/test_plugin_prompt_sections.py` | - | `tests/agent/test_plugin_prompt_sections.py` |
| `agent/test_plugin_stream_hooks.py` | `1` | `agent/test_plugin_stream_hooks.py` | - | `tests/agent/test_plugin_stream_hooks.py` |
| `agent/test_pool_revert_after_cooldown.py` | `1` | `agent/test_pool_revert_after_cooldown.py` | - | `tests/agent/test_pool_revert_after_cooldown.py` |
| `agent/test_pool_rotation_endpoint_veto.py` | `1` | `agent/test_pool_rotation_endpoint_veto.py` | - | `tests/agent/test_pool_rotation_endpoint_veto.py` |
| `agent/test_portal_tags.py` | `1` | `agent/test_portal_tags.py` | - | `tests/agent/test_portal_tags.py` |
| `agent/test_post_compaction_hook.py` | `1` | `agent/test_post_compaction_hook.py` | - | `tests/agent/test_post_compaction_hook.py` |
| `agent/test_post_compression_trim.py` | `1` | `agent/test_post_compression_trim.py` | - | `tests/agent/test_post_compression_trim.py` |
| `agent/test_post_tool_compression_attempt_cap.py` | `1` | `agent/test_post_tool_compression_attempt_cap.py` | - | `tests/agent/test_post_tool_compression_attempt_cap.py` |
| `agent/test_pre_compress_checkpoint_contract.py` | `1` | `agent/test_pre_compress_checkpoint_contract.py` | - | `tests/agent/test_pre_compress_checkpoint_contract.py` |
| `agent/test_pre_compress_memory_context.py` | `1` | `agent/test_pre_compress_memory_context.py` | - | `tests/agent/test_pre_compress_memory_context.py` |
| `agent/test_pre_compress_memory_context_handoff.py` | `1` | `agent/test_pre_compress_memory_context_handoff.py` | - | `tests/agent/test_pre_compress_memory_context_handoff.py` |
| `agent/test_pre_llm_block_hook.py` | `1` | `agent/test_pre_llm_block_hook.py` | - | `tests/agent/test_pre_llm_block_hook.py` |
| `agent/test_preflight_compression_cap_e2e.py` | `1` | `agent/test_preflight_compression_cap_e2e.py` | - | `tests/agent/test_preflight_compression_cap_e2e.py` |
| `agent/test_preflight_compression_gate.py` | `1` | `agent/test_preflight_compression_gate.py` | - | `tests/agent/test_preflight_compression_gate.py` |
| `agent/test_preflight_compression_timeout_fail_closed.py` | `1` | `agent/test_preflight_compression_timeout_fail_closed.py` | - | `tests/agent/test_preflight_compression_timeout_fail_closed.py` |
| `agent/test_preflight_lock_defer.py` | `1` | `agent/test_preflight_lock_defer.py` | - | `tests/agent/test_preflight_lock_defer.py` |
| `agent/test_preflight_timeout_typed_result.py` | `1` | `agent/test_preflight_timeout_typed_result.py` | - | `tests/agent/test_preflight_timeout_typed_result.py` |
| `agent/test_prepared_compaction.py` | `1` | `agent/test_prepared_compaction.py` | - | `tests/agent/test_prepared_compaction.py` |
| `agent/test_primary_runtime_restore.py` | `1` | `agent/test_primary_runtime_restore.py` | - | `tests/agent/test_primary_runtime_restore.py` |
| `agent/test_private_credential_writers.py` | `1` | `agent/test_private_credential_writers.py` | - | `tests/agent/test_private_credential_writers.py` |
| `agent/test_proactive_prune_config.py` | `1` | `agent/test_proactive_prune_config.py` | - | `tests/agent/test_proactive_prune_config.py` |
| `agent/test_proactive_prune_loop_wiring.py` | `1` | `agent/test_proactive_prune_loop_wiring.py` | - | `tests/agent/test_proactive_prune_loop_wiring.py` |
| `agent/test_proactive_prune_rearm_threshold.py` | `1` | `agent/test_proactive_prune_rearm_threshold.py` | - | `tests/agent/test_proactive_prune_rearm_threshold.py` |
| `agent/test_proactive_prune_restart_safety.py` | `1` | `agent/test_proactive_prune_restart_safety.py` | - | `tests/agent/test_proactive_prune_restart_safety.py` |
| `agent/test_proactive_tool_result_pruning.py` | `1` | `agent/test_proactive_tool_result_pruning.py` | - | `tests/agent/test_proactive_tool_result_pruning.py` |
| `agent/test_probe_cache_followups.py` | `1` | `agent/test_probe_cache_followups.py` | - | `tests/agent/test_probe_cache_followups.py` |
| `agent/test_profile_declared_core_sets.py` | `1` | `agent/test_profile_declared_core_sets.py` | - | `tests/agent/test_profile_declared_core_sets.py` |
| `agent/test_profile_home_override_precedence.py` | `1` | `agent/test_profile_home_override_precedence.py` | - | `tests/agent/test_profile_home_override_precedence.py` |
| `agent/test_project_skills.py` | `1` | `agent/test_project_skills.py` | - | `tests/agent/test_project_skills.py` |
| `agent/test_prompt_builder.py` | `1` | `agent/test_prompt_builder.py` | - | `tests/agent/test_prompt_builder.py` |
| `agent/test_prompt_cache_boundary.py` | `1` | `agent/test_prompt_cache_boundary.py` | - | `tests/agent/test_prompt_cache_boundary.py` |
| `agent/test_prompt_cache_prefix_contract.py` | `1` | `agent/test_prompt_cache_prefix_contract.py` | - | `tests/agent/test_prompt_cache_prefix_contract.py` |
| `agent/test_prompt_cache_scope.py` | `1` | `agent/test_prompt_cache_scope.py` | - | `tests/agent/test_prompt_cache_scope.py` |
| `agent/test_prompt_cache_ttl_auto.py` | `1` | `agent/test_prompt_cache_ttl_auto.py` | - | `tests/agent/test_prompt_cache_ttl_auto.py` |
| `agent/test_prompt_cache_ttl_propagation.py` | `1` | `agent/test_prompt_cache_ttl_propagation.py` | - | `tests/agent/test_prompt_cache_ttl_propagation.py` |
| `agent/test_prompt_caching.py` | `1` | `agent/test_prompt_caching.py` | - | `tests/agent/test_prompt_caching.py` |
| `agent/test_protected_tail_pressure.py` | `1` | `agent/test_protected_tail_pressure.py` | - | `tests/agent/test_protected_tail_pressure.py` |
| `agent/test_provider_attribution_headers.py` | `1` | `agent/test_provider_attribution_headers.py` | - | `tests/agent/test_provider_attribution_headers.py` |
| `agent/test_provider_client_cancel.py` | `1` | `agent/test_provider_client_cancel.py` | - | `tests/agent/test_provider_client_cancel.py` |
| `agent/test_provider_client_seam.py` | `1` | `agent/test_provider_client_seam.py` | - | `tests/agent/test_provider_client_seam.py` |
| `agent/test_provider_credential_seams.py` | `1` | `agent/test_provider_credential_seams.py` | - | `tests/agent/test_provider_credential_seams.py` |
| `agent/test_provider_fallback.py` | `1` | `agent/test_provider_fallback.py` | - | `tests/agent/test_provider_fallback.py` |
| `agent/test_provider_history_parity.py` | `1` | `agent/test_provider_history_parity.py` | - | `tests/agent/test_provider_history_parity.py` |
| `agent/test_provider_hook_policy.py` | `1` | `agent/test_provider_hook_policy.py` | - | `tests/agent/test_provider_hook_policy.py` |
| `agent/test_provider_id_mint_normalization.py` | `1` | `agent/test_provider_id_mint_normalization.py` | - | `tests/agent/test_provider_id_mint_normalization.py` |
| `agent/test_provider_media.py` | `1` | `agent/test_provider_media.py` | - | `tests/agent/test_provider_media.py` |
| `agent/test_provider_parity.py` | `1` | `agent/test_provider_parity.py` | - | `tests/agent/test_provider_parity.py` |
| `agent/test_provider_profile_recovery_seam.py` | `1` | `agent/test_provider_profile_recovery_seam.py` | - | `tests/agent/test_provider_profile_recovery_seam.py` |
| `agent/test_provider_projection.py` | `1` | `agent/test_provider_projection.py` | - | `tests/agent/test_provider_projection.py` |
| `agent/test_provider_refresh_hook_failures.py` | `1` | `agent/test_provider_refresh_hook_failures.py` | - | `tests/agent/test_provider_refresh_hook_failures.py` |
| `agent/test_provider_reset_cooldown.py` | `1` | `agent/test_provider_reset_cooldown.py` | - | `tests/agent/test_provider_reset_cooldown.py` |
| `agent/test_provider_scope_keys.py` | `1` | `agent/test_provider_scope_keys.py` | - | `tests/agent/test_provider_scope_keys.py` |
| `agent/test_provider_wire_seams.py` | `1` | `agent/test_provider_wire_seams.py` | - | `tests/agent/test_provider_wire_seams.py` |
| `agent/test_proxy_and_url_validation.py` | `1` | `agent/test_proxy_and_url_validation.py` | - | `tests/agent/test_proxy_and_url_validation.py` |
| `agent/test_proxy_bypass_shared.py` | `1` | `agent/test_proxy_bypass_shared.py` | - | `tests/agent/test_proxy_bypass_shared.py` |
| `agent/test_pydantic_dump_warning_leak.py` | `1` | `agent/test_pydantic_dump_warning_leak.py` | - | `tests/agent/test_pydantic_dump_warning_leak.py` |
| `agent/test_rate_limit_reset_hint.py` | `1` | `agent/test_rate_limit_reset_hint.py` | - | `tests/agent/test_rate_limit_reset_hint.py` |
| `agent/test_rate_limit_tracker.py` | `1` | `agent/test_rate_limit_tracker.py` | - | `tests/agent/test_rate_limit_tracker.py` |
| `agent/test_reactions.py` | `1` | `agent/test_reactions.py` | - | `tests/agent/test_reactions.py` |
| `agent/test_read_dedup_rewrite_boundaries.py` | `1` | `agent/test_read_dedup_rewrite_boundaries.py` | - | `tests/agent/test_read_dedup_rewrite_boundaries.py` |
| `agent/test_reasoning_carriers.py` | `1` | `agent/test_reasoning_carriers.py` | - | `tests/agent/test_reasoning_carriers.py` |
| `agent/test_reasoning_disable_recovery.py` | `1` | `agent/test_reasoning_disable_recovery.py` | - | `tests/agent/test_reasoning_disable_recovery.py` |
| `agent/test_reasoning_echo_resolver_e2e.py` | `1` | `agent/test_reasoning_echo_resolver_e2e.py` | - | `tests/agent/test_reasoning_echo_resolver_e2e.py` |
| `agent/test_reasoning_effort_labels.py` | `1` | `agent/test_reasoning_effort_labels.py` | - | `tests/agent/test_reasoning_effort_labels.py` |
| `agent/test_reasoning_effort_module.py` | `1` | `agent/test_reasoning_effort_module.py` | - | `tests/agent/test_reasoning_effort_module.py` |
| `agent/test_reasoning_effort_wire_translation.py` | `1` | `agent/test_reasoning_effort_wire_translation.py` | - | `tests/agent/test_reasoning_effort_wire_translation.py` |
| `agent/test_reasoning_gate_claimed_hosts.py` | `1` | `agent/test_reasoning_gate_claimed_hosts.py` | - | `tests/agent/test_reasoning_gate_claimed_hosts.py` |
| `agent/test_reasoning_only_stop_persistence.py` | `1` | `agent/test_reasoning_only_stop_persistence.py` | - | `tests/agent/test_reasoning_only_stop_persistence.py` |
| `agent/test_reasoning_replay_routes.py` | `1` | `agent/test_reasoning_replay_routes.py` | - | `tests/agent/test_reasoning_replay_routes.py` |
| `agent/test_reasoning_shape_boundaries.py` | `1` | `agent/test_reasoning_shape_boundaries.py` | - | `tests/agent/test_reasoning_shape_boundaries.py` |
| `agent/test_reasoning_stale_timeout_floor.py` | `1` | `agent/test_reasoning_stale_timeout_floor.py` | - | `tests/agent/test_reasoning_stale_timeout_floor.py` |
| `agent/test_reasoning_summaries.py` | `1` | `agent/test_reasoning_summaries.py` | - | `tests/agent/test_reasoning_summaries.py` |
| `agent/test_recovery_diagnostic_producers.py` | `1` | `agent/test_recovery_diagnostic_producers.py` | - | `tests/agent/test_recovery_diagnostic_producers.py` |
| `agent/test_redact.py` | `1` | `agent/test_redact.py` | - | `tests/agent/test_redact.py` |
| `agent/test_redaction_registry.py` | `1` | `agent/test_redaction_registry.py` | - | `tests/agent/test_redaction_registry.py` |
| `agent/test_reference_handoff_active_turn.py` | `1` | `agent/test_reference_handoff_active_turn.py` | - | `tests/agent/test_reference_handoff_active_turn.py` |
| `agent/test_refine_focus.py` | `1` | `agent/test_refine_focus.py` | - | `tests/agent/test_refine_focus.py` |
| `agent/test_refine_snapshot_isolation.py` | `1` | `agent/test_refine_snapshot_isolation.py` | - | `tests/agent/test_refine_snapshot_isolation.py` |
| `agent/test_relay_atof_cwd.py` | `1` | `agent/test_relay_atof_cwd.py` | - | `tests/agent/test_relay_atof_cwd.py` |
| `agent/test_relay_compaction.py` | `1` | `agent/test_relay_compaction.py` | - | `tests/agent/test_relay_compaction.py` |
| `agent/test_relay_compaction_atof.py` | `1` | `agent/test_relay_compaction_atof.py` | - | `tests/agent/test_relay_compaction_atof.py` |
| `agent/test_relay_cwd.py` | `1` | `agent/test_relay_cwd.py` | - | `tests/agent/test_relay_cwd.py` |
| `agent/test_relay_llm.py` | `1` | `agent/test_relay_llm.py` | - | `tests/agent/test_relay_llm.py` |
| `agent/test_relay_nested_execution.py` | `1` | `agent/test_relay_nested_execution.py` | - | `tests/agent/test_relay_nested_execution.py` |
| `agent/test_relay_runtime_bounded_scope_ops.py` | `1` | `agent/test_relay_runtime_bounded_scope_ops.py` | - | `tests/agent/test_relay_runtime_bounded_scope_ops.py` |
| `agent/test_relay_runtime_plugins.py` | `1` | `agent/test_relay_runtime_plugins.py` | - | `tests/agent/test_relay_runtime_plugins.py` |
| `agent/test_relay_scope_pop_metadata.py` | `1` | `agent/test_relay_scope_pop_metadata.py` | - | `tests/agent/test_relay_scope_pop_metadata.py` |
| `agent/test_relay_session_segments.py` | `1` | `agent/test_relay_session_segments.py` | - | `tests/agent/test_relay_session_segments.py` |
| `agent/test_relay_tools.py` | `1` | `agent/test_relay_tools.py` | - | `tests/agent/test_relay_tools.py` |
| `agent/test_remote_fetch_ssrf_guard.py` | `1` | `agent/test_remote_fetch_ssrf_guard.py` | - | `tests/agent/test_remote_fetch_ssrf_guard.py` |
| `agent/test_repair_reanchors_current_turn_user_idx.py` | `1` | `agent/test_repair_reanchors_current_turn_user_idx.py` | - | `tests/agent/test_repair_reanchors_current_turn_user_idx.py` |
| `agent/test_repair_tool_call_arguments.py` | `1` | `agent/test_repair_tool_call_arguments.py` | - | `tests/agent/test_repair_tool_call_arguments.py` |
| `agent/test_repair_tool_call_name.py` | `1` | `agent/test_repair_tool_call_name.py` | - | `tests/agent/test_repair_tool_call_name.py` |
| `agent/test_repeated_dropped_tool_args.py` | `1` | `agent/test_repeated_dropped_tool_args.py` | - | `tests/agent/test_repeated_dropped_tool_args.py` |
| `agent/test_repetition_guard.py` | `1` | `agent/test_repetition_guard.py` | - | `tests/agent/test_repetition_guard.py` |
| `agent/test_replay_budget_accounting.py` | `1` | `agent/test_replay_budget_accounting.py` | - | `tests/agent/test_replay_budget_accounting.py` |
| `agent/test_replay_cleanup.py` | `1` | `agent/test_replay_cleanup.py` | - | `tests/agent/test_replay_cleanup.py` |
| `agent/test_replay_echo_retirement.py` | `1` | `agent/test_replay_echo_retirement.py` | - | `tests/agent/test_replay_echo_retirement.py` |
| `agent/test_reply_line_before_silence.py` | `1` | `agent/test_reply_line_before_silence.py` | - | `tests/agent/test_reply_line_before_silence.py` |
| `agent/test_request_client_reuse.py` | `1` | `agent/test_request_client_reuse.py` | - | `tests/agent/test_request_client_reuse.py` |
| `agent/test_request_client_reuse_abort_races.py` | `1` | `agent/test_request_client_reuse_abort_races.py` | - | `tests/agent/test_request_client_reuse_abort_races.py` |
| `agent/test_reset_aware_primary_restore.py` | `1` | `agent/test_reset_aware_primary_restore.py` | - | `tests/agent/test_reset_aware_primary_restore.py` |
| `agent/test_responses_empty_tool_replay.py` | `1` | `agent/test_responses_empty_tool_replay.py` | - | `tests/agent/test_responses_empty_tool_replay.py` |
| `agent/test_responses_watchdog_accommodations.py` | `1` | `agent/test_responses_watchdog_accommodations.py` | - | `tests/agent/test_responses_watchdog_accommodations.py` |
| `agent/test_restore_primary_pool_reselect.py` | `1` | `agent/test_restore_primary_pool_reselect.py` | - | `tests/agent/test_restore_primary_pool_reselect.py` |
| `agent/test_resume_stale_active_task.py` | `1` | `agent/test_resume_stale_active_task.py` | - | `tests/agent/test_resume_stale_active_task.py` |
| `agent/test_retry_delay_parsers_shared.py` | `1` | `agent/test_retry_delay_parsers_shared.py` | - | `tests/agent/test_retry_delay_parsers_shared.py` |
| `agent/test_retry_exhaustion_partial_retention.py` | `1` | `agent/test_retry_exhaustion_partial_retention.py` | - | `tests/agent/test_retry_exhaustion_partial_retention.py` |
| `agent/test_retry_status_buffer.py` | `1` | `agent/test_retry_status_buffer.py` | - | `tests/agent/test_retry_status_buffer.py` |
| `agent/test_retry_utils.py` | `1` | `agent/test_retry_utils.py` | - | `tests/agent/test_retry_utils.py` |
| `agent/test_review_engine.py` | `1` | `agent/test_review_engine.py` | - | `tests/agent/test_review_engine.py` |
| `agent/test_review_fork_cache_scope.py` | `1` | `agent/test_review_fork_cache_scope.py` | - | `tests/agent/test_review_fork_cache_scope.py` |
| `agent/test_review_idle_queue.py` | `1` | `agent/test_review_idle_queue.py` | - | `tests/agent/test_review_idle_queue.py` |
| `agent/test_rotation_flush_persisted_boundary.py` | `1` | `agent/test_rotation_flush_persisted_boundary.py` | - | `tests/agent/test_rotation_flush_persisted_boundary.py` |
| `agent/test_router_timeout_shim.py` | `1` | `agent/test_router_timeout_shim.py` | - | `tests/agent/test_router_timeout_shim.py` |
| `agent/test_run_agent.py` | `1` | `agent/test_run_agent.py` | - | `tests/agent/test_run_agent.py` |
| `agent/test_run_agent_codex_responses.py` | `1` | `agent/test_run_agent_codex_responses.py` | - | `tests/agent/test_run_agent_codex_responses.py` |
| `agent/test_run_agent_interrupt_hook.py` | `1` | `agent/test_run_agent_interrupt_hook.py` | - | `tests/agent/test_run_agent_interrupt_hook.py` |
| `agent/test_run_agent_multimodal_prologue.py` | `1` | `agent/test_run_agent_multimodal_prologue.py` | - | `tests/agent/test_run_agent_multimodal_prologue.py` |
| `agent/test_run_budget.py` | `1` | `agent/test_run_budget.py` | - | `tests/agent/test_run_budget.py` |
| `agent/test_runtime_cwd.py` | `1` | `agent/test_runtime_cwd.py` | - | `tests/agent/test_runtime_cwd.py` |
| `agent/test_runtime_self_protection.py` | `1` | `agent/test_runtime_self_protection.py` | - | `tests/agent/test_runtime_self_protection.py` |
| `agent/test_runtime_session_store.py` | `1` | `agent/test_runtime_session_store.py` | - | `tests/agent/test_runtime_session_store.py` |
| `agent/test_runtime_session_store_lease_refresh.py` | `1` | `agent/test_runtime_session_store_lease_refresh.py` | - | `tests/agent/test_runtime_session_store_lease_refresh.py` |
| `agent/test_runtime_session_store_proxy.py` | `1` | `agent/test_runtime_session_store_proxy.py` | - | `tests/agent/test_runtime_session_store_proxy.py` |
| `agent/test_runtime_session_store_surface_parity.py` | `1` | `agent/test_runtime_session_store_surface_parity.py` | - | `tests/agent/test_runtime_session_store_surface_parity.py` |
| `agent/test_safe_worker_policy.py` | `1` | `agent/test_safe_worker_policy.py` | - | `tests/agent/test_safe_worker_policy.py` |
| `agent/test_salvage_grown_transcript.py` | `1` | `agent/test_salvage_grown_transcript.py` | - | `tests/agent/test_salvage_grown_transcript.py` |
| `agent/test_sanitiser_escalation.py` | `1` | `agent/test_sanitiser_escalation.py` | - | `tests/agent/test_sanitiser_escalation.py` |
| `agent/test_save_url_image.py` | `1` | `agent/test_save_url_image.py` | - | `tests/agent/test_save_url_image.py` |
| `agent/test_secret_cache_location.py` | `1` | `agent/test_secret_cache_location.py` | - | `tests/agent/test_secret_cache_location.py` |
| `agent/test_secret_scope.py` | `1` | `agent/test_secret_scope.py` | - | `tests/agent/test_secret_scope.py` |
| `agent/test_secret_scope_env_cache.py` | `1` | `agent/test_secret_scope_env_cache.py` | - | `tests/agent/test_secret_scope_env_cache.py` |
| `agent/test_secret_scope_pinned_launch_home.py` | `1` | `agent/test_secret_scope_pinned_launch_home.py` | - | `tests/agent/test_secret_scope_pinned_launch_home.py` |
| `agent/test_secret_scope_tier1_migration.py` | `1` | `agent/test_secret_scope_tier1_migration.py` | - | `tests/agent/test_secret_scope_tier1_migration.py` |
| `agent/test_secret_scope_user_message.py` | `1` | `agent/test_secret_scope_user_message.py` | - | `tests/agent/test_secret_scope_user_message.py` |
| `agent/test_send_path_history_isolation.py` | `1` | `agent/test_send_path_history_isolation.py` | - | `tests/agent/test_send_path_history_isolation.py` |
| `agent/test_sequential_deadline_delegate_exempt.py` | `1` | `agent/test_sequential_deadline_delegate_exempt.py` | - | `tests/agent/test_sequential_deadline_delegate_exempt.py` |
| `agent/test_sequential_tool_interrupt.py` | `1` | `agent/test_sequential_tool_interrupt.py` | - | `tests/agent/test_sequential_tool_interrupt.py` |
| `agent/test_sequential_tool_timeout.py` | `1` | `agent/test_sequential_tool_timeout.py` | - | `tests/agent/test_sequential_tool_timeout.py` |
| `agent/test_served_model.py` | `1` | `agent/test_served_model.py` | - | `tests/agent/test_served_model.py` |
| `agent/test_serves_routed_profile_pin.py` | `1` | `agent/test_serves_routed_profile_pin.py` | - | `tests/agent/test_serves_routed_profile_pin.py` |
| `agent/test_session_activity.py` | `1` | `agent/test_session_activity.py` | - | `tests/agent/test_session_activity.py` |
| `agent/test_session_activity_persist.py` | `1` | `agent/test_session_activity_persist.py` | - | `tests/agent/test_session_activity_persist.py` |
| `agent/test_session_affinity_header.py` | `1` | `agent/test_session_affinity_header.py` | - | `tests/agent/test_session_affinity_header.py` |
| `agent/test_session_id_env.py` | `1` | `agent/test_session_id_env.py` | - | `tests/agent/test_session_id_env.py` |
| `agent/test_session_meta_filtering.py` | `1` | `agent/test_session_meta_filtering.py` | - | `tests/agent/test_session_meta_filtering.py` |
| `agent/test_session_prompt_refresh.py` | `1` | `agent/test_session_prompt_refresh.py` | - | `tests/agent/test_session_prompt_refresh.py` |
| `agent/test_session_reset_fix.py` | `1` | `agent/test_session_reset_fix.py` | - | `tests/agent/test_session_reset_fix.py` |
| `agent/test_session_rotation_flush_cold_resume.py` | `1` | `agent/test_session_rotation_flush_cold_resume.py` | - | `tests/agent/test_session_rotation_flush_cold_resume.py` |
| `agent/test_session_row_under_live_agent_persist.py` | `1` | `agent/test_session_row_under_live_agent_persist.py` | - | `tests/agent/test_session_row_under_live_agent_persist.py` |
| `agent/test_session_snapshot_removal.py` | `1` | `agent/test_session_snapshot_removal.py` | - | `tests/agent/test_session_snapshot_removal.py` |
| `agent/test_session_source.py` | `1` | `agent/test_session_source.py` | - | `tests/agent/test_session_source.py` |
| `agent/test_set_runtime_main_custom_provider.py` | `1` | `agent/test_set_runtime_main_custom_provider.py` | - | `tests/agent/test_set_runtime_main_custom_provider.py` |
| `agent/test_setup_beat_card_round.py` | `1` | `agent/test_setup_beat_card_round.py` | - | `tests/agent/test_setup_beat_card_round.py` |
| `agent/test_setup_logging_profile_scope.py` | `1` | `agent/test_setup_logging_profile_scope.py` | - | `tests/agent/test_setup_logging_profile_scope.py` |
| `agent/test_shared_client_fd_corruption.py` | `1` | `agent/test_shared_client_fd_corruption.py` | - | `tests/agent/test_shared_client_fd_corruption.py` |
| `agent/test_shared_http_transport.py` | `1` | `agent/test_shared_http_transport.py` | - | `tests/agent/test_shared_http_transport.py` |
| `agent/test_shell_hooks.py` | `1` | `agent/test_shell_hooks.py` | - | `tests/agent/test_shell_hooks.py` |
| `agent/test_shell_hooks_consent.py` | `1` | `agent/test_shell_hooks_consent.py` | - | `tests/agent/test_shell_hooks_consent.py` |
| `agent/test_shell_hooks_tree_kill.py` | `1` | `agent/test_shell_hooks_tree_kill.py` | - | `tests/agent/test_shell_hooks_tree_kill.py` |
| `agent/test_short_housekeeping_reply.py` | `1` | `agent/test_short_housekeeping_reply.py` | - | `tests/agent/test_short_housekeeping_reply.py` |
| `agent/test_side_question.py` | `1` | `agent/test_side_question.py` | - | `tests/agent/test_side_question.py` |
| `agent/test_side_question_lease.py` | `1` | `agent/test_side_question_lease.py` | - | `tests/agent/test_side_question_lease.py` |
| `agent/test_side_question_request.py` | `1` | `agent/test_side_question_request.py` | - | `tests/agent/test_side_question_request.py` |
| `agent/test_skill_app_snapshot.py` | `1` | `agent/test_skill_app_snapshot.py` | - | `tests/agent/test_skill_app_snapshot.py` |
| `agent/test_skill_applied_notifications.py` | `1` | `agent/test_skill_applied_notifications.py` | - | `tests/agent/test_skill_applied_notifications.py` |
| `agent/test_skill_bundles.py` | `1` | `agent/test_skill_bundles.py` | - | `tests/agent/test_skill_bundles.py` |
| `agent/test_skill_cache_thrash.py` | `1` | `agent/test_skill_cache_thrash.py` | - | `tests/agent/test_skill_cache_thrash.py` |
| `agent/test_skill_commands.py` | `1` | `agent/test_skill_commands.py` | - | `tests/agent/test_skill_commands.py` |
| `agent/test_skill_commands_freshness.py` | `1` | `agent/test_skill_commands_freshness.py` | - | `tests/agent/test_skill_commands_freshness.py` |
| `agent/test_skill_commands_reload.py` | `1` | `agent/test_skill_commands_reload.py` | - | `tests/agent/test_skill_commands_reload.py` |
| `agent/test_skill_inline_bash_windows.py` | `1` | `agent/test_skill_inline_bash_windows.py` | - | `tests/agent/test_skill_inline_bash_windows.py` |
| `agent/test_skill_inline_shell_output.py` | `1` | `agent/test_skill_inline_shell_output.py` | - | `tests/agent/test_skill_inline_shell_output.py` |
| `agent/test_skill_inline_shell_trust.py` | `1` | `agent/test_skill_inline_shell_trust.py` | - | `tests/agent/test_skill_inline_shell_trust.py` |
| `agent/test_skill_invocation_description.py` | `1` | `agent/test_skill_invocation_description.py` | - | `tests/agent/test_skill_invocation_description.py` |
| `agent/test_skill_session_platform_gate.py` | `1` | `agent/test_skill_session_platform_gate.py` | - | `tests/agent/test_skill_session_platform_gate.py` |
| `agent/test_skill_todo_retention_parity.py` | `1` | `agent/test_skill_todo_retention_parity.py` | - | `tests/agent/test_skill_todo_retention_parity.py` |
| `agent/test_skill_utils.py` | `1` | `agent/test_skill_utils.py` | - | `tests/agent/test_skill_utils.py` |
| `agent/test_skills_auto_load.py` | `1` | `agent/test_skills_auto_load.py` | - | `tests/agent/test_skills_auto_load.py` |
| `agent/test_skills_guidance_content_filter.py` | `1` | `agent/test_skills_guidance_content_filter.py` | - | `tests/agent/test_skills_guidance_content_filter.py` |
| `agent/test_skills_index_delta.py` | `1` | `agent/test_skills_index_delta.py` | - | `tests/agent/test_skills_index_delta.py` |
| `agent/test_skills_index_format.py` | `1` | `agent/test_skills_index_format.py` | - | `tests/agent/test_skills_index_format.py` |
| `agent/test_skip_background_review.py` | `1` | `agent/test_skip_background_review.py` | - | `tests/agent/test_skip_background_review.py` |
| `agent/test_skip_memory_store.py` | `1` | `agent/test_skip_memory_store.py` | - | `tests/agent/test_skip_memory_store.py` |
| `agent/test_soul_legacy_bot_protocol.py` | `1` | `agent/test_soul_legacy_bot_protocol.py` | - | `tests/agent/test_soul_legacy_bot_protocol.py` |
| `agent/test_soul_md_profile_isolation.py` | `1` | `agent/test_soul_md_profile_isolation.py` | - | `tests/agent/test_soul_md_profile_isolation.py` |
| `agent/test_split_turn_compaction.py` | `1` | `agent/test_split_turn_compaction.py` | - | `tests/agent/test_split_turn_compaction.py` |
| `agent/test_ssl_transport_contracts.py` | `1` | `agent/test_ssl_transport_contracts.py` | - | `tests/agent/test_ssl_transport_contracts.py` |
| `agent/test_ssl_verify.py` | `1` | `agent/test_ssl_verify.py` | - | `tests/agent/test_ssl_verify.py` |
| `agent/test_stale_replay_prune.py` | `1` | `agent/test_stale_replay_prune.py` | - | `tests/agent/test_stale_replay_prune.py` |
| `agent/test_stall_guards.py` | `1` | `agent/test_stall_guards.py` | - | `tests/agent/test_stall_guards.py` |
| `agent/test_start_order_gate.py` | `1` | `agent/test_start_order_gate.py` | - | `tests/agent/test_start_order_gate.py` |
| `agent/test_steer.py` | `1` | `agent/test_steer.py` | - | `tests/agent/test_steer.py` |
| `agent/test_steer_compaction_persistence.py` | `1` | `agent/test_steer_compaction_persistence.py` | - | `tests/agent/test_steer_compaction_persistence.py` |
| `agent/test_stream_chunk_byte_estimate.py` | `1` | `agent/test_stream_chunk_byte_estimate.py` | - | `tests/agent/test_stream_chunk_byte_estimate.py` |
| `agent/test_stream_drop_logging.py` | `1` | `agent/test_stream_drop_logging.py` | - | `tests/agent/test_stream_drop_logging.py` |
| `agent/test_stream_interrupt_abort_socket.py` | `1` | `agent/test_stream_interrupt_abort_socket.py` | - | `tests/agent/test_stream_interrupt_abort_socket.py` |
| `agent/test_stream_interrupt_retry.py` | `1` | `agent/test_stream_interrupt_retry.py` | - | `tests/agent/test_stream_interrupt_retry.py` |
| `agent/test_stream_max_lifetime.py` | `1` | `agent/test_stream_max_lifetime.py` | - | `tests/agent/test_stream_max_lifetime.py` |
| `agent/test_stream_reasoning_details.py` | `1` | `agent/test_stream_reasoning_details.py` | - | `tests/agent/test_stream_reasoning_details.py` |
| `agent/test_stream_retry_backoff.py` | `1` | `agent/test_stream_retry_backoff.py` | - | `tests/agent/test_stream_retry_backoff.py` |
| `agent/test_stream_retry_route_switch.py` | `1` | `agent/test_stream_retry_route_switch.py` | - | `tests/agent/test_stream_retry_route_switch.py` |
| `agent/test_stream_runaway_repetition.py` | `1` | `agent/test_stream_runaway_repetition.py` | - | `tests/agent/test_stream_runaway_repetition.py` |
| `agent/test_stream_serving_provider.py` | `1` | `agent/test_stream_serving_provider.py` | - | `tests/agent/test_stream_serving_provider.py` |
| `agent/test_stream_single_writer.py` | `1` | `agent/test_stream_single_writer.py` | - | `tests/agent/test_stream_single_writer.py` |
| `agent/test_stream_single_writer_guard.py` | `1` | `agent/test_stream_single_writer_guard.py` | - | `tests/agent/test_stream_single_writer_guard.py` |
| `agent/test_stream_stale_abandon.py` | `1` | `agent/test_stream_stale_abandon.py` | - | `tests/agent/test_stream_stale_abandon.py` |
| `agent/test_stream_stale_breaker_reset.py` | `1` | `agent/test_stream_stale_breaker_reset.py` | - | `tests/agent/test_stream_stale_breaker_reset.py` |
| `agent/test_stream_stale_circuit_breaker.py` | `1` | `agent/test_stream_stale_circuit_breaker.py` | - | `tests/agent/test_stream_stale_circuit_breaker.py` |
| `agent/test_stream_stale_kill_unwedge.py` | `1` | `agent/test_stream_stale_kill_unwedge.py` | - | `tests/agent/test_stream_stale_kill_unwedge.py` |
| `agent/test_stream_unmask_5xx.py` | `1` | `agent/test_stream_unmask_5xx.py` | - | `tests/agent/test_stream_unmask_5xx.py` |
| `agent/test_stream_wait_notice.py` | `1` | `agent/test_stream_wait_notice.py` | - | `tests/agent/test_stream_wait_notice.py` |
| `agent/test_streamed_reasoning_details.py` | `1` | `agent/test_streamed_reasoning_details.py` | - | `tests/agent/test_streamed_reasoning_details.py` |
| `agent/test_streamed_text_accumulation.py` | `1` | `agent/test_streamed_text_accumulation.py` | - | `tests/agent/test_streamed_text_accumulation.py` |
| `agent/test_streaming.py` | `1` | `agent/test_streaming.py` | - | `tests/agent/test_streaming.py` |
| `agent/test_streaming_tool_call_repair.py` | `1` | `agent/test_streaming_tool_call_repair.py` | - | `tests/agent/test_streaming_tool_call_repair.py` |
| `agent/test_strict_api_validation.py` | `1` | `agent/test_strict_api_validation.py` | - | `tests/agent/test_strict_api_validation.py` |
| `agent/test_strip_reasoning_tags_cli.py` | `1` | `agent/test_strip_reasoning_tags_cli.py` | - | `tests/agent/test_strip_reasoning_tags_cli.py` |
| `agent/test_structured_output_rejection_retry.py` | `1` | `agent/test_structured_output_rejection_retry.py` | - | `tests/agent/test_structured_output_rejection_retry.py` |
| `agent/test_studio_bridge_affinity.py` | `1` | `agent/test_studio_bridge_affinity.py` | - | `tests/agent/test_studio_bridge_affinity.py` |
| `agent/test_subagent_lifecycle.py` | `1` | `agent/test_subagent_lifecycle.py` | - | `tests/agent/test_subagent_lifecycle.py` |
| `agent/test_subagent_progress.py` | `1` | `agent/test_subagent_progress.py` | - | `tests/agent/test_subagent_progress.py` |
| `agent/test_subagent_stop_hook.py` | `1` | `agent/test_subagent_stop_hook.py` | - | `tests/agent/test_subagent_stop_hook.py` |
| `agent/test_subdirectory_hints.py` | `1` | `agent/test_subdirectory_hints.py` | - | `tests/agent/test_subdirectory_hints.py` |
| `agent/test_subdirectory_hints_tilde.py` | `1` | `agent/test_subdirectory_hints_tilde.py` | - | `tests/agent/test_subdirectory_hints_tilde.py` |
| `agent/test_subprocess_env_guard.py` | `1` | `agent/test_subprocess_env_guard.py` | - | `tests/agent/test_subprocess_env_guard.py` |
| `agent/test_subscription_view.py` | `1` | `agent/test_subscription_view.py` | - | `tests/agent/test_subscription_view.py` |
| `agent/test_summarize_api_error.py` | `1` | `agent/test_summarize_api_error.py` | - | `tests/agent/test_summarize_api_error.py` |
| `agent/test_summarize_tool_result_type_safety.py` | `1` | `agent/test_summarize_tool_result_type_safety.py` | - | `tests/agent/test_summarize_tool_result_type_safety.py` |
| `agent/test_summary_prefix_semantics.py` | `1` | `agent/test_summary_prefix_semantics.py` | - | `tests/agent/test_summary_prefix_semantics.py` |
| `agent/test_summary_prefix_tool_use.py` | `1` | `agent/test_summary_prefix_tool_use.py` | - | `tests/agent/test_summary_prefix_tool_use.py` |
| `agent/test_summary_role_template_alternation.py` | `1` | `agent/test_summary_role_template_alternation.py` | - | `tests/agent/test_summary_role_template_alternation.py` |
| `agent/test_surrogate_chokepoints.py` | `1` | `agent/test_surrogate_chokepoints.py` | - | `tests/agent/test_surrogate_chokepoints.py` |
| `agent/test_switch_model_bedrock_sigv4.py` | `1` | `agent/test_switch_model_bedrock_sigv4.py` | - | `tests/agent/test_switch_model_bedrock_sigv4.py` |
| `agent/test_switch_model_context.py` | `1` | `agent/test_switch_model_context.py` | - | `tests/agent/test_switch_model_context.py` |
| `agent/test_switch_model_fallback_prune.py` | `1` | `agent/test_switch_model_fallback_prune.py` | - | `tests/agent/test_switch_model_fallback_prune.py` |
| `agent/test_switch_model_pool_reload.py` | `1` | `agent/test_switch_model_pool_reload.py` | - | `tests/agent/test_switch_model_pool_reload.py` |
| `agent/test_switch_model_reapplies_config_context_pin.py` | `1` | `agent/test_switch_model_reapplies_config_context_pin.py` | - | `tests/agent/test_switch_model_reapplies_config_context_pin.py` |
| `agent/test_switch_model_reapplies_headers.py` | `1` | `agent/test_switch_model_reapplies_headers.py` | - | `tests/agent/test_switch_model_reapplies_headers.py` |
| `agent/test_switch_model_reasoning_override.py` | `1` | `agent/test_switch_model_reasoning_override.py` | - | `tests/agent/test_switch_model_reasoning_override.py` |
| `agent/test_switch_model_request_overrides.py` | `1` | `agent/test_switch_model_request_overrides.py` | - | `tests/agent/test_switch_model_request_overrides.py` |
| `agent/test_switch_model_rollback.py` | `1` | `agent/test_switch_model_rollback.py` | - | `tests/agent/test_switch_model_rollback.py` |
| `agent/test_switch_model_stale_base_url.py` | `1` | `agent/test_switch_model_stale_base_url.py` | - | `tests/agent/test_switch_model_stale_base_url.py` |
| `agent/test_synthetic_turn_display_kind.py` | `1` | `agent/test_synthetic_turn_display_kind.py` | - | `tests/agent/test_synthetic_turn_display_kind.py` |
| `agent/test_system_prompt.py` | `1` | `agent/test_system_prompt.py` | - | `tests/agent/test_system_prompt.py` |
| `agent/test_system_prompt_restore.py` | `1` | `agent/test_system_prompt_restore.py` | - | `tests/agent/test_system_prompt_restore.py` |
| `agent/test_system_prompt_switch_invalidation.py` | `1` | `agent/test_system_prompt_switch_invalidation.py` | - | `tests/agent/test_system_prompt_switch_invalidation.py` |
| `agent/test_terminal_approval_batch.py` | `1` | `agent/test_terminal_approval_batch.py` | - | `tests/agent/test_terminal_approval_batch.py` |
| `agent/test_terminal_env_registry.py` | `1` | `agent/test_terminal_env_registry.py` | - | `tests/agent/test_terminal_env_registry.py` |
| `agent/test_text_turn_incremental_persistence.py` | `1` | `agent/test_text_turn_incremental_persistence.py` | - | `tests/agent/test_text_turn_incremental_persistence.py` |
| `agent/test_think_scrubber.py` | `1` | `agent/test_think_scrubber.py` | - | `tests/agent/test_think_scrubber.py` |
| `agent/test_thinking_only_sanitizer.py` | `1` | `agent/test_thinking_only_sanitizer.py` | - | `tests/agent/test_thinking_only_sanitizer.py` |
| `agent/test_thinking_prefill_trailing_turn.py` | `1` | `agent/test_thinking_prefill_trailing_turn.py` | - | `tests/agent/test_thinking_prefill_trailing_turn.py` |
| `agent/test_thinking_timeout_guidance.py` | `1` | `agent/test_thinking_timeout_guidance.py` | - | `tests/agent/test_thinking_timeout_guidance.py` |
| `agent/test_thread_scoped_output.py` | `1` | `agent/test_thread_scoped_output.py` | - | `tests/agent/test_thread_scoped_output.py` |
| `agent/test_timeout_error_summary.py` | `1` | `agent/test_timeout_error_summary.py` | - | `tests/agent/test_timeout_error_summary.py` |
| `agent/test_title_generator.py` | `1` | `agent/test_title_generator.py` | - | `tests/agent/test_title_generator.py` |
| `agent/test_title_input.py` | `1` | `agent/test_title_input.py` | - | `tests/agent/test_title_input.py` |
| `agent/test_tls_fd_recycle_corruption.py` | `1` | `agent/test_tls_fd_recycle_corruption.py` | - | `tests/agent/test_tls_fd_recycle_corruption.py` |
| `agent/test_token_estimator_shared.py` | `1` | `agent/test_token_estimator_shared.py` | - | `tests/agent/test_token_estimator_shared.py` |
| `agent/test_token_guard_session_source.py` | `1` | `agent/test_token_guard_session_source.py` | - | `tests/agent/test_token_guard_session_source.py` |
| `agent/test_token_memo_lru_budget.py` | `1` | `agent/test_token_memo_lru_budget.py` | - | `tests/agent/test_token_memo_lru_budget.py` |
| `agent/test_token_memo_release_on_close.py` | `1` | `agent/test_token_memo_release_on_close.py` | - | `tests/agent/test_token_memo_release_on_close.py` |
| `agent/test_token_persistence_non_cli.py` | `1` | `agent/test_token_persistence_non_cli.py` | - | `tests/agent/test_token_persistence_non_cli.py` |
| `agent/test_token_read_barrier_on_loop.py` | `1` | `agent/test_token_read_barrier_on_loop.py` | - | `tests/agent/test_token_read_barrier_on_loop.py` |
| `agent/test_tool_activity_heartbeat.py` | `1` | `agent/test_tool_activity_heartbeat.py` | - | `tests/agent/test_tool_activity_heartbeat.py` |
| `agent/test_tool_arg_coercion.py` | `1` | `agent/test_tool_arg_coercion.py` | - | `tests/agent/test_tool_arg_coercion.py` |
| `agent/test_tool_batch_segmentation.py` | `1` | `agent/test_tool_batch_segmentation.py` | - | `tests/agent/test_tool_batch_segmentation.py` |
| `agent/test_tool_call_arg_no_redaction.py` | `1` | `agent/test_tool_call_arg_no_redaction.py` | - | `tests/agent/test_tool_call_arg_no_redaction.py` |
| `agent/test_tool_call_args_sanitizer.py` | `1` | `agent/test_tool_call_args_sanitizer.py` | - | `tests/agent/test_tool_call_args_sanitizer.py` |
| `agent/test_tool_call_guardrail_runtime.py` | `1` | `agent/test_tool_call_guardrail_runtime.py` | - | `tests/agent/test_tool_call_guardrail_runtime.py` |
| `agent/test_tool_call_incremental_persistence.py` | `1` | `agent/test_tool_call_incremental_persistence.py` | - | `tests/agent/test_tool_call_incremental_persistence.py` |
| `agent/test_tool_dispatch_helpers.py` | `1` | `agent/test_tool_dispatch_helpers.py` | - | `tests/agent/test_tool_dispatch_helpers.py` |
| `agent/test_tool_executor_checkpoint_paths.py` | `1` | `agent/test_tool_executor_checkpoint_paths.py` | - | `tests/agent/test_tool_executor_checkpoint_paths.py` |
| `agent/test_tool_name_db_persistence.py` | `1` | `agent/test_tool_name_db_persistence.py` | - | `tests/agent/test_tool_name_db_persistence.py` |
| `agent/test_tool_result_memory_trim.py` | `1` | `agent/test_tool_result_memory_trim.py` | - | `tests/agent/test_tool_result_memory_trim.py` |
| `agent/test_trace_upload.py` | `1` | `agent/test_trace_upload.py` | - | `tests/agent/test_trace_upload.py` |
| `agent/test_trajectory_file_locking.py` | `1` | `agent/test_trajectory_file_locking.py` | - | `tests/agent/test_trajectory_file_locking.py` |
| `agent/test_transcript_decoration_idempotence.py` | `1` | `agent/test_transcript_decoration_idempotence.py` | - | `tests/agent/test_transcript_decoration_idempotence.py` |
| `agent/test_transcription_registry.py` | `1` | `agent/test_transcription_registry.py` | - | `tests/agent/test_transcription_registry.py` |
| `agent/test_transform_llm_output_persistence.py` | `1` | `agent/test_transform_llm_output_persistence.py` | - | `tests/agent/test_transform_llm_output_persistence.py` |
| `agent/test_treekill_consolidation.py` | `1` | `agent/test_treekill_consolidation.py` | - | `tests/agent/test_treekill_consolidation.py` |
| `agent/test_truncated_tool_call_boost.py` | `1` | `agent/test_truncated_tool_call_boost.py` | - | `tests/agent/test_truncated_tool_call_boost.py` |
| `agent/test_tts_registry.py` | `1` | `agent/test_tts_registry.py` | - | `tests/agent/test_tts_registry.py` |
| `agent/test_turn_api_call_interrupt.py` | `1` | `agent/test_turn_api_call_interrupt.py` | - | `tests/agent/test_turn_api_call_interrupt.py` |
| `agent/test_turn_api_error_stream_parse.py` | `1` | `agent/test_turn_api_error_stream_parse.py` | - | `tests/agent/test_turn_api_error_stream_parse.py` |
| `agent/test_turn_author.py` | `1` | `agent/test_turn_author.py` | - | `tests/agent/test_turn_author.py` |
| `agent/test_turn_base_display_anchor.py` | `1` | `agent/test_turn_base_display_anchor.py` | - | `tests/agent/test_turn_base_display_anchor.py` |
| `agent/test_turn_budget_durable_rows.py` | `1` | `agent/test_turn_budget_durable_rows.py` | - | `tests/agent/test_turn_budget_durable_rows.py` |
| `agent/test_turn_completion_explainer.py` | `1` | `agent/test_turn_completion_explainer.py` | - | `tests/agent/test_turn_completion_explainer.py` |
| `agent/test_turn_context.py` | `1` | `agent/test_turn_context.py` | - | `tests/agent/test_turn_context.py` |
| `agent/test_turn_context_compaction.py` | `1` | `agent/test_turn_context_compaction.py` | - | `tests/agent/test_turn_context_compaction.py` |
| `agent/test_turn_context_overflow_warning.py` | `1` | `agent/test_turn_context_overflow_warning.py` | - | `tests/agent/test_turn_context_overflow_warning.py` |
| `agent/test_turn_end_stall_diagnostics.py` | `1` | `agent/test_turn_end_stall_diagnostics.py` | - | `tests/agent/test_turn_end_stall_diagnostics.py` |
| `agent/test_turn_facade_lease.py` | `1` | `agent/test_turn_facade_lease.py` | - | `tests/agent/test_turn_facade_lease.py` |
| `agent/test_turn_finalizer_cleanup_guard.py` | `1` | `agent/test_turn_finalizer_cleanup_guard.py` | - | `tests/agent/test_turn_finalizer_cleanup_guard.py` |
| `agent/test_turn_finalizer_final_response_persistence.py` | `1` | `agent/test_turn_finalizer_final_response_persistence.py` | - | `tests/agent/test_turn_finalizer_final_response_persistence.py` |
| `agent/test_turn_finalizer_interrupt_alternation.py` | `1` | `agent/test_turn_finalizer_interrupt_alternation.py` | - | `tests/agent/test_turn_finalizer_interrupt_alternation.py` |
| `agent/test_turn_finalizer_iteration_limit_exit.py` | `1` | `agent/test_turn_finalizer_iteration_limit_exit.py` | - | `tests/agent/test_turn_finalizer_iteration_limit_exit.py` |
| `agent/test_turn_iteration_prep.py` | `1` | `agent/test_turn_iteration_prep.py` | - | `tests/agent/test_turn_iteration_prep.py` |
| `agent/test_turn_lease_frees_by_refcount.py` | `1` | `agent/test_turn_lease_frees_by_refcount.py` | - | `tests/agent/test_turn_lease_frees_by_refcount.py` |
| `agent/test_turn_liveness.py` | `1` | `agent/test_turn_liveness.py` | - | `tests/agent/test_turn_liveness.py` |
| `agent/test_turn_liveness_watchdog.py` | `1` | `agent/test_turn_liveness_watchdog.py` | - | `tests/agent/test_turn_liveness_watchdog.py` |
| `agent/test_turn_overlap_tripwire.py` | `1` | `agent/test_turn_overlap_tripwire.py` | - | `tests/agent/test_turn_overlap_tripwire.py` |
| `agent/test_turn_recovery_autorecover.py` | `1` | `agent/test_turn_recovery_autorecover.py` | - | `tests/agent/test_turn_recovery_autorecover.py` |
| `agent/test_turn_response_intake_relay.py` | `1` | `agent/test_turn_response_intake_relay.py` | - | `tests/agent/test_turn_response_intake_relay.py` |
| `agent/test_turn_retry_state.py` | `1` | `agent/test_turn_retry_state.py` | - | `tests/agent/test_turn_retry_state.py` |
| `agent/test_turn_scripted_prelude.py` | `1` | `agent/test_turn_scripted_prelude.py` | - | `tests/agent/test_turn_scripted_prelude.py` |
| `agent/test_turn_summary.py` | `1` | `agent/test_turn_summary.py` | - | `tests/agent/test_turn_summary.py` |
| `agent/test_turn_usage_log_line.py` | `1` | `agent/test_turn_usage_log_line.py` | - | `tests/agent/test_turn_usage_log_line.py` |
| `agent/test_turn_user_intervened.py` | `1` | `agent/test_turn_user_intervened.py` | - | `tests/agent/test_turn_user_intervened.py` |
| `agent/test_uncompressed_context_guardrail.py` | `1` | `agent/test_uncompressed_context_guardrail.py` | - | `tests/agent/test_uncompressed_context_guardrail.py` |
| `agent/test_unicode_ascii_codec.py` | `1` | `agent/test_unicode_ascii_codec.py` | - | `tests/agent/test_unicode_ascii_codec.py` |
| `agent/test_unset_reasoning_effort_default.py` | `1` | `agent/test_unset_reasoning_effort_default.py` | - | `tests/agent/test_unset_reasoning_effort_default.py` |
| `agent/test_usage_anchor.py` | `1` | `agent/test_usage_anchor.py` | - | `tests/agent/test_usage_anchor.py` |
| `agent/test_usage_pricing.py` | `1` | `agent/test_usage_pricing.py` | - | `tests/agent/test_usage_pricing.py` |
| `agent/test_vault_backend_registry.py` | `1` | `agent/test_vault_backend_registry.py` | - | `tests/agent/test_vault_backend_registry.py` |
| `agent/test_vault_backends.py` | `1` | `agent/test_vault_backends.py` | - | `tests/agent/test_vault_backends.py` |
| `agent/test_vault_key_storage.py` | `1` | `agent/test_vault_key_storage.py` | - | `tests/agent/test_vault_key_storage.py` |
| `agent/test_vault_keychain_native.py` | `1` | `agent/test_vault_keychain_native.py` | - | `tests/agent/test_vault_keychain_native.py` |
| `agent/test_vault_store.py` | `1` | `agent/test_vault_store.py` | - | `tests/agent/test_vault_store.py` |
| `agent/test_vault_unlock_tokens.py` | `1` | `agent/test_vault_unlock_tokens.py` | - | `tests/agent/test_vault_unlock_tokens.py` |
| `agent/test_verification_continuation_budget.py` | `1` | `agent/test_verification_continuation_budget.py` | - | `tests/agent/test_verification_continuation_budget.py` |
| `agent/test_verification_evidence.py` | `1` | `agent/test_verification_evidence.py` | - | `tests/agent/test_verification_evidence.py` |
| `agent/test_verification_evidence_disabled.py` | `1` | `agent/test_verification_evidence_disabled.py` | - | `tests/agent/test_verification_evidence_disabled.py` |
| `agent/test_verification_evidence_fd_leak.py` | `1` | `agent/test_verification_evidence_fd_leak.py` | - | `tests/agent/test_verification_evidence_fd_leak.py` |
| `agent/test_verification_stop.py` | `1` | `agent/test_verification_stop.py` | - | `tests/agent/test_verification_stop.py` |
| `agent/test_verification_stop_caching.py` | `1` | `agent/test_verification_stop_caching.py` | - | `tests/agent/test_verification_stop_caching.py` |
| `agent/test_verify_hooks.py` | `1` | `agent/test_verify_hooks.py` | - | `tests/agent/test_verify_hooks.py` |
| `agent/test_vertex_adapter.py` | `1` | `agent/test_vertex_adapter.py` | - | `tests/agent/test_vertex_adapter.py` |
| `agent/test_video_gen_registry.py` | `1` | `agent/test_video_gen_registry.py` | - | `tests/agent/test_video_gen_registry.py` |
| `agent/test_vision_aware_preprocessing.py` | `1` | `agent/test_vision_aware_preprocessing.py` | - | `tests/agent/test_vision_aware_preprocessing.py` |
| `agent/test_vision_resolved_args.py` | `1` | `agent/test_vision_resolved_args.py` | - | `tests/agent/test_vision_resolved_args.py` |
| `agent/test_vision_routing.py` | `1` | `agent/test_vision_routing.py` | - | `tests/agent/test_vision_routing.py` |
| `agent/test_vision_tool_messages.py` | `1` | `agent/test_vision_tool_messages.py` | - | `tests/agent/test_vision_tool_messages.py` |
| `agent/test_voice_turn_route.py` | `1` | `agent/test_voice_turn_route.py` | - | `tests/agent/test_voice_turn_route.py` |
| `agent/test_wait_notice_cadence.py` | `1` | `agent/test_wait_notice_cadence.py` | - | `tests/agent/test_wait_notice_cadence.py` |
| `agent/test_wait_state_visibility.py` | `1` | `agent/test_wait_state_visibility.py` | - | `tests/agent/test_wait_state_visibility.py` |
| `agent/test_warning_presentation.py` | `1` | `agent/test_warning_presentation.py` | - | `tests/agent/test_warning_presentation.py` |
| `agent/test_welcome_error_identity.py` | `1` | `agent/test_welcome_error_identity.py` | - | `tests/agent/test_welcome_error_identity.py` |
| `agent/test_welcome_tier_recovery.py` | `1` | `agent/test_welcome_tier_recovery.py` | - | `tests/agent/test_welcome_tier_recovery.py` |
| `agent/test_worker_outbox_adoption.py` | `1` | `agent/test_worker_outbox_adoption.py` | - | `tests/agent/test_worker_outbox_adoption.py` |
| `agent/test_worker_outbox_reopen.py` | `1` | `agent/test_worker_outbox_reopen.py` | - | `tests/agent/test_worker_outbox_reopen.py` |
| `agent/test_worker_rpc_connection_reuse.py` | `1` | `agent/test_worker_rpc_connection_reuse.py` | - | `tests/agent/test_worker_rpc_connection_reuse.py` |
| `agent/test_worker_rpc_loop_diagnostic.py` | `1` | `agent/test_worker_rpc_loop_diagnostic.py` | - | `tests/agent/test_worker_rpc_loop_diagnostic.py` |
| `agent/test_worker_rpc_proxy.py` | `1` | `agent/test_worker_rpc_proxy.py` | - | `tests/agent/test_worker_rpc_proxy.py` |
| `agent/test_would_grow_refusal_runway.py` | `1` | `agent/test_would_grow_refusal_runway.py` | - | `tests/agent/test_would_grow_refusal_runway.py` |
| `agent/transports` | `19` | `agent/transports` | `Hermes agent runtime strict-fidelity source-pair expansion`, `Hermes fast-mode request override serializer`, `OpenRouter Pareto router request plugin` | `tests/agent/transports/__init__.py`, `tests/agent/transports/test_bedrock_transport.py`, `tests/agent/transports/test_chat_completions_empty_tool_calls.py`, `tests/agent/transports/test_chat_completions_reasoning_details_replay.py`, `tests/agent/transports/test_chat_completions_xai_tool_search_alias.py` |
| `ci` | `41` | `ci` | - | `tests/ci/_gha_expr.py`, `tests/ci/desktop_release_roles.py`, `tests/ci/test_assemble_review_comment.py`, `tests/ci/test_cache_report.py`, `tests/ci/test_cache_report_collection.py` |
| `compat` | `4` | `compat` | - | `tests/compat/README.md`, `tests/compat/old_updater_dependencies.py`, `tests/compat/old_updater_support.py`, `tests/compat/old_updater_surface.json` |
| `computer_use` | `12` | `computer_use` | - | `tests/computer_use/driver_fixture.py`, `tests/computer_use/live_cua_0_9_smoke.py`, `tests/computer_use/test_cua_atexit_teardown.py`, `tests/computer_use/test_cua_no_overlay.py`, `tests/computer_use/test_cua_perf_knobs.py` |
| `conformance` | `14` | `conformance` | - | `tests/conformance/__init__.py`, `tests/conformance/persistence/README.md`, `tests/conformance/persistence/__init__.py`, `tests/conformance/persistence/_harness.py`, `tests/conformance/persistence/test_cell1_prefix_durability.py` |
| `conftest.py` | `1` | `conftest.py` | - | `tests/conftest.py` |
| `cron` | `176` | `cron` | `Cron deliver=all routing intent expansion`, `Cron env-ref expansion + parallel run state serialization`, `Cron no-agent script-only short-circuit`, `Cron origin delivery isolation from session identity`, `Cron partial legacy job read-model normalization` | `tests/cron/__init__.py`, `tests/cron/conftest.py`, `tests/cron/test_agent_scheduling_gate.py`, `tests/cron/test_atomic_paused_creation.py`, `tests/cron/test_blueprint_catalog.py` |
| `dashboard` | `1` | `dashboard` | - | `tests/dashboard/test_ws_client_host.py` |
| `desktop` | `2` | `desktop` | - | `tests/desktop/test_appinstaller_check_script.py`, `tests/desktop/test_check_store_update.py` |
| `docker` | `33` | `docker` | - | `tests/docker/__init__.py`, `tests/docker/conftest.py`, `tests/docker/test_config_migration.py`, `tests/docker/test_container_restart.py`, `tests/docker/test_dashboard.py` |
| `e2e` | `258` | `e2e` | - | `tests/e2e/__init__.py`, `tests/e2e/conftest.py`, `tests/e2e/core/__init__.py`, `tests/e2e/core/_pending_fixes.py`, `tests/e2e/core/_pm_dependencies.py` |
| `evals` | `4` | `evals` | - | `tests/evals/__init__.py`, `tests/evals/test_codebase_navigability.py`, `tests/evals/test_compaction_jev_arm.py`, `tests/evals/test_core_tool_deferral_sandbox.py` |
| `fakes` | `22` | `fakes` | - | `tests/fakes/cron_owner.py`, `tests/fakes/fake_llm_provider.py`, `tests/fakes/platforms/__init__.py`, `tests/fakes/platforms/_standin.py`, `tests/fakes/platforms/discord_shim/sitecustomize.py` |
| `file_write_observer.py` | `1` | `file_write_observer.py` | - | `tests/file_write_observer.py` |
| `fixtures` | `48` | `fixtures` | - | `tests/fixtures/cua_driver_0_9_tools_list.json`, `tests/fixtures/identifier_speech_corpus.json`, `tests/fixtures/legacy_desktop/update-marker-be3fd671.mts`, `tests/fixtures/plugin_component_detection.json`, `tests/fixtures/plugins/example-dashboard/dashboard/manifest.json` |
| `gateway` | `1804` | `gateway` | `Advance Hermes checkout and source pairs to latest upstream`, `Agent lifecycle hooks (agent:start, agent:step, agent:end)`, `Agent turn and tool execution events on bus`, `Discord gateway event-bus adapter`, `Discord native slash/thread command registration parity` | `tests/gateway/__init__.py`, `tests/gateway/_plugin_adapter_loader.py`, `tests/gateway/conftest.py`, `tests/gateway/feishu_helpers.py`, `tests/gateway/fixtures/agent_persistence_worker.py` |
| `git_safety.py` | `1` | `git_safety.py` | - | `tests/git_safety.py` |
| `hermes_cli` | `1868` | `hermes_cli` | `Advance Hermes checkout and source pairs to latest upstream`, `Agent personalities + enhanced display config`, `Auth state TOCTOU close + redaction default-on parity`, `Bitwarden disk cache parity`, `Bitwarden managed bws install + checksum verification` | `tests/hermes_cli/__init__.py`, `tests/hermes_cli/anon_portal.py`, `tests/hermes_cli/benchmark_session_timeline.py`, `tests/hermes_cli/conftest.py`, `tests/hermes_cli/conftest_dashboard_auth.py` |
| `hermes_platform` | `9` | `hermes_platform` | - | `tests/hermes_platform/__init__.py`, `tests/hermes_platform/test_declaration.py`, `tests/hermes_platform/test_facts.py`, `tests/hermes_platform/test_gpu_adapters.py`, `tests/hermes_platform/test_host_summary.py` |
| `hermes_state` | `214` | `hermes_state` | - | `tests/hermes_state/__init__.py`, `tests/hermes_state/_wal_generation_harness.py`, `tests/hermes_state/conftest.py`, `tests/hermes_state/test_append_message_after_close.py`, `tests/hermes_state/test_append_messages_batch.py` |
| `home_io_guard.py` | `1` | `home_io_guard.py` | - | `tests/home_io_guard.py` |
| `install` | `74` | `install` | - | `tests/install/BUNDLED_UPDATES.md`, `tests/install/KNOWN_FAILURES.md`, `tests/install/README.md`, `tests/install/e2e-assets/bundle-inputs.mjs`, `tests/install/e2e-assets/bundle-manifest.cjs` |
| `installation` | `2` | `installation` | - | `tests/installation/__init__.py`, `tests/installation/test_installer_common.py` |
| `installation_launcher_fixture.py` | `1` | `installation_launcher_fixture.py` | - | `tests/installation_launcher_fixture.py` |
| `integration` | `1` | `integration` | - | `tests/integration/test_local_runtime_pm_download.py` |
| `live_process_fixtures.py` | `1` | `live_process_fixtures.py` | - | `tests/live_process_fixtures.py` |
| `lock_contention.py` | `1` | `lock_contention.py` | - | `tests/lock_contention.py` |
| `manual` | `2` | `manual` | - | `tests/manual/cron_inchannel_dm_e2e.py`, `tests/manual/cron_inchannel_e2e.py` |
| `module_clock.py` | `1` | `module_clock.py` | - | `tests/module_clock.py` |
| `monitoring` | `6` | `monitoring` | - | `tests/monitoring/__init__.py`, `tests/monitoring/test_cron_health_export.py`, `tests/monitoring/test_emitter.py`, `tests/monitoring/test_export_redaction.py`, `tests/monitoring/test_gateway_health_export.py` |
| `perf_guards` | `1` | `perf_guards` | - | `tests/perf_guards/test_pattern_b_scaling.py` |
| `plugins` | `160` | `plugins` | `Advance Hermes checkout and source pairs to latest upstream`, `Bundled platform plugin manifest drift guard`, `Google Chat install dependency hint refresh`, `Google Chat relay sender-type self-filter`, `Google Chat shared-chassis platform adapter seam` | `tests/plugins/__init__.py`, `tests/plugins/browser/__init__.py`, `tests/plugins/browser/check_parity_vs_main.py`, `tests/plugins/browser/test_browser_provider_plugins.py`, `tests/plugins/config_backends/__init__.py` |
| `pm` | `142` | `pm` | - | `tests/pm/__init__.py`, `tests/pm/_fixtures.py`, `tests/pm/_range_server.py`, `tests/pm/activation_support.py`, `tests/pm/conftest.py` |
| `posix_lock_probe.py` | `1` | `posix_lock_probe.py` | - | `tests/posix_lock_probe.py` |
| `providers` | `22` | `providers` | - | `tests/providers/__init__.py`, `tests/providers/fake_pkce_idp.py`, `tests/providers/test_auth_registry_import_order.py`, `tests/providers/test_auth_registry_lazy_mirror.py`, `tests/providers/test_auth_registry_mid_discovery.py` |
| `run_interrupt_test.py` | `1` | `run_interrupt_test.py` | - | `tests/run_interrupt_test.py` |
| `scripts` | `178` | `scripts` | - | `tests/scripts/desktop_update/legacy_desktop_reader.py`, `tests/scripts/desktop_update/lineage_rule_cases.py`, `tests/scripts/desktop_update/test_desktop_update_gateway_flag.py`, `tests/scripts/desktop_update/test_desktop_update_linux_gate.py`, `tests/scripts/desktop_update/test_desktop_update_linux_gate_arm.py` |
| `secret_sources` | `5` | `secret_sources` | - | `tests/secret_sources/__init__.py`, `tests/secret_sources/conformance.py`, `tests/secret_sources/test_error_remediation.py`, `tests/secret_sources/test_profile_secrets.py`, `tests/secret_sources/test_secret_source_registry.py` |
| `security` | `1` | `security` | - | `tests/security/test_gitspawn_config_injection.py` |
| `skills` | `32` | `skills` | `Advance Hermes checkout and source pairs to latest upstream`, `Hermes integrations claim audit + source-backed plugin/skill parity map`, `Hermes skill catalog strict-fidelity classifier`, `Sharp v1.0 differentiator decision` | `tests/skills/test_authoring_standards.py`, `tests/skills/test_bundled_skill_doc_placeholders.py`, `tests/skills/test_cloudflare_temporary_deploy_skill.py`, `tests/skills/test_comfyui_skill.py`, `tests/skills/test_document_env_blocks.py` |
| `termux_fixtures.py` | `1` | `termux_fixtures.py` | - | `tests/termux_fixtures.py` |
| `test_agent_close_lifecycle.py` | `1` | `test_agent_close_lifecycle.py` | - | `tests/test_agent_close_lifecycle.py` |
| `test_atomic_json_writers_unified.py` | `1` | `test_atomic_json_writers_unified.py` | - | `tests/test_atomic_json_writers_unified.py` |
| `test_atomic_replace_symlinks.py` | `1` | `test_atomic_replace_symlinks.py` | - | `tests/test_atomic_replace_symlinks.py` |
| `test_atomic_write_fsync_dir_target.py` | `1` | `test_atomic_write_fsync_dir_target.py` | - | `tests/test_atomic_write_fsync_dir_target.py` |
| `test_atomic_write_text_metadata.py` | `1` | `test_atomic_write_text_metadata.py` | - | `tests/test_atomic_write_text_metadata.py` |
| `test_audit_old_updater_imports.py` | `1` | `test_audit_old_updater_imports.py` | - | `tests/test_audit_old_updater_imports.py` |
| `test_base_url_hostname.py` | `1` | `test_base_url_hostname.py` | - | `tests/test_base_url_hostname.py` |
| `test_basetemp_isolation.py` | `1` | `test_basetemp_isolation.py` | - | `tests/test_basetemp_isolation.py` |
| `test_batch_runner_checkpoint.py` | `1` | `test_batch_runner_checkpoint.py` | - | `tests/test_batch_runner_checkpoint.py` |
| `test_batch_runner_discard_resume.py` | `1` | `test_batch_runner_discard_resume.py` | - | `tests/test_batch_runner_discard_resume.py` |
| `test_batch_runner_durability.py` | `1` | `test_batch_runner_durability.py` | - | `tests/test_batch_runner_durability.py` |
| `test_credential_file_permissions.py` | `1` | `test_credential_file_permissions.py` | - | `tests/test_credential_file_permissions.py` |
| `test_dispatch_session_id.py` | `1` | `test_dispatch_session_id.py` | - | `tests/test_dispatch_session_id.py` |
| `test_display_hermes_home.py` | `1` | `test_display_hermes_home.py` | - | `tests/test_display_hermes_home.py` |
| `test_docker_poll_stalled_exec.py` | `1` | `test_docker_poll_stalled_exec.py` | - | `tests/test_docker_poll_stalled_exec.py` |
| `test_engines_satisfiable.py` | `1` | `test_engines_satisfiable.py` | - | `tests/test_engines_satisfiable.py` |
| `test_fast_safe_load.py` | `1` | `test_fast_safe_load.py` | - | `tests/test_fast_safe_load.py` |
| `test_get_tool_definitions_cache_isolation.py` | `1` | `test_get_tool_definitions_cache_isolation.py` | - | `tests/test_get_tool_definitions_cache_isolation.py` |
| `test_git_safety_boundary.py` | `1` | `test_git_safety_boundary.py` | - | `tests/test_git_safety_boundary.py` |
| `test_hermes_bootstrap.py` | `1` | `test_hermes_bootstrap.py` | - | `tests/test_hermes_bootstrap.py` |
| `test_hermes_constants.py` | `1` | `test_hermes_constants.py` | - | `tests/test_hermes_constants.py` |
| `test_hermes_dir_acl.py` | `1` | `test_hermes_dir_acl.py` | - | `tests/test_hermes_dir_acl.py` |
| `test_hermes_dir_acl_windows.py` | `1` | `test_hermes_dir_acl_windows.py` | - | `tests/test_hermes_dir_acl_windows.py` |
| `test_hermes_home_key_cache.py` | `1` | `test_hermes_home_key_cache.py` | - | `tests/test_hermes_home_key_cache.py` |
| `test_hermes_home_profile_warning.py` | `1` | `test_hermes_home_profile_warning.py` | - | `tests/test_hermes_home_profile_warning.py` |
| `test_hermes_logging.py` | `1` | `test_hermes_logging.py` | - | `tests/test_hermes_logging.py` |
| `test_hermes_yaml.py` | `1` | `test_hermes_yaml.py` | - | `tests/test_hermes_yaml.py` |
| `test_ipv4_preference.py` | `1` | `test_ipv4_preference.py` | - | `tests/test_ipv4_preference.py` |
| `test_live_nous_services_fence.py` | `1` | `test_live_nous_services_fence.py` | - | `tests/test_live_nous_services_fence.py` |
| `test_live_system_guard.py` | `1` | `test_live_system_guard.py` | - | `tests/test_live_system_guard.py` |
| `test_live_system_guard_self_test.py` | `1` | `test_live_system_guard_self_test.py` | - | `tests/test_live_system_guard_self_test.py` |
| `test_load_yaml_file_readonly.py` | `1` | `test_load_yaml_file_readonly.py` | - | `tests/test_load_yaml_file_readonly.py` |
| `test_log_isolation.py` | `1` | `test_log_isolation.py` | - | `tests/test_log_isolation.py` |
| `test_managed_runtime_resolution.py` | `1` | `test_managed_runtime_resolution.py` | - | `tests/test_managed_runtime_resolution.py` |
| `test_mini_swe_runner.py` | `1` | `test_mini_swe_runner.py` | - | `tests/test_mini_swe_runner.py` |
| `test_model_forces_max_completion_tokens.py` | `1` | `test_model_forces_max_completion_tokens.py` | - | `tests/test_model_forces_max_completion_tokens.py` |
| `test_model_tools_async_bridge.py` | `1` | `test_model_tools_async_bridge.py` | - | `tests/test_model_tools_async_bridge.py` |
| `test_no_shadowed_test_definitions.py` | `1` | `test_no_shadowed_test_definitions.py` | - | `tests/test_no_shadowed_test_definitions.py` |
| `test_old_updater_compat_surface.py` | `1` | `test_old_updater_compat_surface.py` | - | `tests/test_old_updater_compat_surface.py` |
| `test_os_marker_gating.py` | `1` | `test_os_marker_gating.py` | - | `tests/test_os_marker_gating.py` |
| `test_packaging_build_guard.py` | `1` | `test_packaging_build_guard.py` | - | `tests/test_packaging_build_guard.py` |
| `test_packaging_metadata.py` | `1` | `test_packaging_metadata.py` | - | `tests/test_packaging_metadata.py` |
| `test_packaging_py_modules.py` | `1` | `test_packaging_py_modules.py` | - | `tests/test_packaging_py_modules.py` |
| `test_platforms_marker.py` | `1` | `test_platforms_marker.py` | - | `tests/test_platforms_marker.py` |
| `test_project_metadata.py` | `1` | `test_project_metadata.py` | - | `tests/test_project_metadata.py` |
| `test_sanitize_tool_error.py` | `1` | `test_sanitize_tool_error.py` | - | `tests/test_sanitize_tool_error.py` |
| `test_scratch_dir.py` | `1` | `test_scratch_dir.py` | - | `tests/test_scratch_dir.py` |
| `test_scratch_prune_undeletable.py` | `1` | `test_scratch_prune_undeletable.py` | - | `tests/test_scratch_prune_undeletable.py` |
| `test_scratch_registered_clock_step.py` | `1` | `test_scratch_registered_clock_step.py` | - | `tests/test_scratch_registered_clock_step.py` |
| `test_scratch_scan_failures.py` | `1` | `test_scratch_scan_failures.py` | - | `tests/test_scratch_scan_failures.py` |
| `test_search_cjk_prefix_wildcard.py` | `1` | `test_search_cjk_prefix_wildcard.py` | - | `tests/test_search_cjk_prefix_wildcard.py` |
| `test_session_daemon_reaper.py` | `1` | `test_session_daemon_reaper.py` | - | `tests/test_session_daemon_reaper.py` |
| `test_session_spillover_cleanup.py` | `1` | `test_session_spillover_cleanup.py` | - | `tests/test_session_spillover_cleanup.py` |
| `test_state_config_contention.py` | `1` | `test_state_config_contention.py` | - | `tests/test_state_config_contention.py` |
| `test_tests_tree_layout.py` | `1` | `test_tests_tree_layout.py` | - | `tests/test_tests_tree_layout.py` |
| `test_trajectory_compressor.py` | `1` | `test_trajectory_compressor.py` | - | `tests/test_trajectory_compressor.py` |
| `test_trajectory_compressor_async.py` | `1` | `test_trajectory_compressor_async.py` | - | `tests/test_trajectory_compressor_async.py` |
| `test_utils_atomic_roundtrip_yaml_save.py` | `1` | `test_utils_atomic_roundtrip_yaml_save.py` | - | `tests/test_utils_atomic_roundtrip_yaml_save.py` |
| `test_utils_atomic_writers_deleted_profile.py` | `1` | `test_utils_atomic_writers_deleted_profile.py` | - | `tests/test_utils_atomic_writers_deleted_profile.py` |
| `test_utils_no_proxy_sanitize.py` | `1` | `test_utils_no_proxy_sanitize.py` | - | `tests/test_utils_no_proxy_sanitize.py` |
| `test_utils_rmtree_readonly.py` | `1` | `test_utils_rmtree_readonly.py` | - | `tests/test_utils_rmtree_readonly.py` |
| `test_utils_truthy_values.py` | `1` | `test_utils_truthy_values.py` | - | `tests/test_utils_truthy_values.py` |
| `test_utils_unlink_files_older_than.py` | `1` | `test_utils_unlink_files_older_than.py` | - | `tests/test_utils_unlink_files_older_than.py` |
| `test_wal_reset_repair_hint.py` | `1` | `test_wal_reset_repair_hint.py` | - | `tests/test_wal_reset_repair_hint.py` |
| `test_windows_subprocess_no_window_flags.py` | `1` | `test_windows_subprocess_no_window_flags.py` | - | `tests/test_windows_subprocess_no_window_flags.py` |
| `test_yaml_indent_consistency.py` | `1` | `test_yaml_indent_consistency.py` | - | `tests/test_yaml_indent_consistency.py` |
| `tools` | `844` | `tools` | `ACP session CWD propagation into prompt runners`, `Advance Hermes checkout and source pairs to latest upstream`, `Audio preprocessing and chunking pipeline`, `Block-anchor fuzzy replace for native patch tool`, `Brave Search + DDGS web search provider parity` | `tests/tools/__init__.py`, `tests/tools/_child_env_fixtures.py`, `tests/tools/browser_launcher_fakes.py`, `tests/tools/conftest.py`, `tests/tools/environments/__init__.py` |
| `tui_gateway` | `365` | `tui_gateway` | `Hermes fast-mode request override serializer`, `Hermes gateway platform strict-fidelity source-pair expansion`, `TUI + Telegram browsing`, `TUI gateway config health null-section probe`, `TUI websocket attach transport` | `tests/tui_gateway/__init__.py`, `tests/tui_gateway/_compute_host_frames.py`, `tests/tui_gateway/_turn_marker_backend.py`, `tests/tui_gateway/conftest.py`, `tests/tui_gateway/contracts/__init__.py` |
| `ui-tui` | `148` | `ui-tui` | `Advance Hermes checkout and source pairs to latest upstream`, `CLI setup/onboard/help text fidelity matrix`, `Gateway/TUI /queue explicit FIFO slash parity`, `Gormes-owned chat TUI divergence ratification`, `Gormes-owned semantic chat style system` | `ui-tui/src/__tests__/activeSessionSwitcher.test.ts`, `ui-tui/src/__tests__/agentControls.test.ts`, `ui-tui/src/__tests__/agentRoster.test.ts`, `ui-tui/src/__tests__/agentsCompact.test.tsx`, `ui-tui/src/__tests__/agentsDock.test.tsx` |
| `usage_window_fixtures.py` | `1` | `usage_window_fixtures.py` | - | `tests/usage_window_fixtures.py` |
| `verify` | `4` | `verify` | - | `tests/verify/test_environment_and_runner.py`, `tests/verify/test_ledger_and_nudge_integration.py`, `tests/verify/test_recipes.py`, `tests/verify/test_verify_cmd.py` |
| `waiting.py` | `1` | `waiting.py` | - | `tests/waiting.py` |
| `website` | `8` | `website` | `Google Chat install dependency hint refresh`, `Gormes profile distribution metadata readout`, `Gormes setup/channel/provider docs webpage parity gate`, `Hermes config migration dry-run manifest`, `Hermes website docs mirror coverage gate` | `tests/website/__init__.py`, `tests/website/test_catalog_rules_mirror.py`, `tests/website/test_check_doc_links.py`, `tests/website/test_extract_plugins.py`, `tests/website/test_extract_skills.py` |

## Plugin Catalog Classification

| Family | Status | Count | Progress rows | Source pairs | Examples | Reason |
|---|---|---:|---|---|---|---|
| `browser_web_search` | `partial` | `47` | `ACP Client Bridge Mode`, `ACP JSON-RPC stdio session/prompt closeout`, `ACP bridge doctor/status evidence`, `ACP server side`, `+1065 more` | `gateway/run.py`, `tools/url_safety.py`, `tools/web_tools.py`, `tools/website_policy.py` | `plugins/browser/_common.py`, `plugins/browser/browser_use/__init__.py`, `plugins/browser/browser_use/plugin.yaml`, `plugins/browser/browser_use/provider.py`, `plugins/browser/browserbase/__init__.py`, `+42 more` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `dashboard_observability` | `partial` | `6` | `16ms coalescing mailbox`, `API server detailed health endpoint`, `API server detailed health snapshot contract`, `API server legacy jobs routes + default toolset`, `+146 more` | - | `plugins/hermes-achievements/README.md`, `plugins/hermes-achievements/dashboard/manifest.json`, `plugins/hermes-achievements/dashboard/plugin_api.py`, `plugins/hermes-achievements/tests/test_achievement_engine.py`, `plugins/kanban/dashboard/manifest.json`, `+1 more` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `google_meet` | `partial` | `18` | `Block-anchor fuzzy replace for native patch tool`, `Context-aware fuzzy replace for native patch tool`, `First-party Google Meet plugin metadata fixture`, `Hermes plugin catalog strict-fidelity classifier`, `+3 more` | - | `plugins/google_meet/README.md`, `plugins/google_meet/SKILL.md`, `plugins/google_meet/__init__.py`, `plugins/google_meet/_jsonfile.py`, `plugins/google_meet/audio_bridge.py`, `+13 more` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `image_video_generation` | `partial` | `25` | `6 typed memory categories with confidence scoring`, `ACP Client Bridge Mode`, `ACP JSON-RPC stdio session/prompt closeout`, `ACP server side`, `+535 more` | `agent/prompt_builder.py`, `hermes_cli/default_soul.py`, `tools/image_generation_tool.py`, `tools/transcription_tools.py`, `+1 more` | `plugins/image_gen/_common.py`, `plugins/image_gen/deepinfra/__init__.py`, `plugins/image_gen/deepinfra/plugin.yaml`, `plugins/image_gen/fal/__init__.py`, `plugins/image_gen/fal/plugin.yaml`, `+20 more` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `memory_providers` | `partial` | `9` | `2-layer seed selection`, `6 typed memory categories with confidence scoring`, `<memory-context> fence`, `ACP Client Bridge Mode`, `+805 more` | `agent/context_engine.py`, `agent/prompt_builder.py`, `agent/transports/chat_completions.py`, `agent/transports/codex.py`, `+8 more` | `plugins/memory/__init__.py`, `plugins/memory/byterover/README.md`, `plugins/memory/byterover/__init__.py`, `plugins/memory/byterover/plugin.yaml`, `plugins/memory/config_schema.py`, `+4 more` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `model_providers` | `partial` | `81` | `6 typed memory categories with confidence scoring`, `ACP Client Bridge Mode`, `ACP JSON-RPC stdio session/prompt closeout`, `ACP bridge doctor/status evidence`, `+838 more` | `agent/context_engine.py`, `agent/prompt_builder.py`, `agent/transports/chat_completions.py`, `agent/transports/codex.py`, `+9 more` | `plugins/model-providers/README.md`, `plugins/model-providers/actual/__init__.py`, `plugins/model-providers/actual/plugin.yaml`, `plugins/model-providers/ai-gateway/__init__.py`, `plugins/model-providers/ai-gateway/plugin.yaml`, `+76 more` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `platform_adapters` | `partial` | `188` | `(platform, chat_id) -> session_id`, `61-tool registry port`, `ACP Client Bridge Mode`, `ACP JSON-RPC stdio session/prompt closeout`, `+698 more` | `agent/prompt_builder.py`, `gateway/run.py`, `tools/web_tools.py` | `plugins/platforms/a2a/DESIGN.md`, `plugins/platforms/a2a/README.md`, `plugins/platforms/a2a/__init__.py`, `plugins/platforms/a2a/adapter.py`, `plugins/platforms/a2a/plugin.yaml`, `+183 more` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `teams_pipeline` | `partial` | `9` | `Agent middleware chain framework`, `Agentic-porting-kit public repo scaffold`, `Audio preprocessing and chunking pipeline`, `Backlog split C5d: migrate gormes-* skill discovery commands off raw jq of the canonical progress.json`, `+55 more` | - | `plugins/teams_pipeline/__init__.py`, `plugins/teams_pipeline/cli.py`, `plugins/teams_pipeline/meetings.py`, `plugins/teams_pipeline/models.py`, `plugins/teams_pipeline/pipeline.py`, `+4 more` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |

## Skill Catalog Classification

| Family | Status | Count | Progress rows | Source pairs | Examples | Reason |
|---|---|---:|---|---|---|---|
| `bundled_catalog_metadata` | `partial` | `45` | `Advance Hermes checkout and source pairs to latest upstream`, `Bundled Airtable productivity skill contract`, `Bundled TouchDesigner MCP skill catalog contract`, `Cron GitHub auth-header scanner parity`, `+37 more` | `tools/skills_sync.py`, `tools/skills_tool.py` | `skills/apple/apple-notes/SKILL.md`, `skills/apple/apple-reminders/SKILL.md`, `skills/apple/findmy/SKILL.md`, `skills/apple/imessage/SKILL.md`, `skills/autonomous-ai-agents/claude-code/SKILL.md`, `+40 more` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `category_descriptions` | `partial` | `26` | `Agentic-porting-kit local porting skill skeletons`, `Bundled Airtable productivity skill contract`, `Bundled TouchDesigner MCP skill catalog contract`, `Discord native slash/thread command registration parity`, `+22 more` | - | `optional-skills/DESCRIPTION.md`, `optional-skills/autonomous-ai-agents/DESCRIPTION.md`, `optional-skills/communication/DESCRIPTION.md`, `optional-skills/data-science/DESCRIPTION.md`, `optional-skills/dogfood/DESCRIPTION.md`, `+21 more` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `optional_catalog_metadata` | `partial` | `149` | `Advance Hermes checkout and source pairs to latest upstream`, `Backlog split C5: single atomic operator-gated flip to the module-keyed split directory`, `Backlog split C5a: optional per-row module key + deterministic derivation + backfill`, `Bundled Airtable productivity skill contract`, `+37 more` | - | `optional-skills/autonomous-ai-agents/agent-merge-conflict-arbiter/SKILL.md`, `optional-skills/autonomous-ai-agents/antigravity-cli/SKILL.md`, `optional-skills/autonomous-ai-agents/blackbox/SKILL.md`, `optional-skills/autonomous-ai-agents/dynamic-workflow/SKILL.md`, `optional-skills/autonomous-ai-agents/grok/SKILL.md`, `+144 more` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `prerequisites_readiness_metadata` | `partial` | `31` | `Agentic-porting-kit local public layout assembly gate`, `Agentic-porting-kit local standalone fixture`, `Agentic-porting-kit public repo scaffold`, `Backlog split C5a: optional per-row module key + deterministic derivation + backfill`, `+114 more` | - | `optional-skills/autonomous-ai-agents/honcho/SKILL.md`, `optional-skills/creative/comfyui/SKILL.md`, `optional-skills/creative/hyperframes/SKILL.md`, `optional-skills/email/agentmail/SKILL.md`, `optional-skills/health/fitness-nutrition/SKILL.md`, `+26 more` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `python_script_examples` | `partial` | `172` | `Advance Hermes checkout and source pairs to latest upstream`, `Agentic-porting-kit extraction spec`, `Agentic-porting-kit local README and license fixture`, `Agentic-porting-kit local porting skill skeletons`, `+42 more` | - | `optional-skills/blockchain/evm/scripts/evm_client.py`, `optional-skills/blockchain/hyperliquid/scripts/hyperliquid_client.py`, `optional-skills/blockchain/solana/scripts/solana_client.py`, `optional-skills/creative/ai-presenter-video/scripts/finalize_delivery.sh`, `optional-skills/creative/ai-presenter-video/scripts/init_job.py`, `+167 more` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `support_assets` | `partial` | `580` | `Advance Hermes checkout and source pairs to latest upstream`, `Agent Hooks Registry`, `Agent personalities + enhanced display config`, `Agentic-porting-kit extraction spec`, `+104 more` | `tools/skill_manager_tool.py` | `optional-skills/autonomous-ai-agents/antigravity-cli/references/cli-docs.md`, `optional-skills/creative/ai-presenter-video/assets/job.template.json`, `optional-skills/creative/ai-presenter-video/references/editing.md`, `optional-skills/creative/ai-presenter-video/references/generation.md`, `optional-skills/creative/ai-presenter-video/references/qa-recovery.md`, `+575 more` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `sync_reset_boundaries` | `partial` | `10` | `Advance Hermes checkout and source pairs to latest upstream`, `Bundled Airtable productivity skill contract`, `Dashboard page-scoped plugin slot inventory`, `Dashboard theme/plugin extension status contract`, `+61 more` | `tools/skills_sync.py` | `optional-skills/creative/baoyu-article-illustrator/references/style-presets.md`, `optional-skills/creative/baoyu-comic/references/presets/concept-story.md`, `optional-skills/creative/baoyu-comic/references/presets/four-panel.md`, `optional-skills/creative/baoyu-comic/references/presets/ohmsha.md`, `optional-skills/creative/baoyu-comic/references/presets/shoujo.md`, `+5 more` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `triggers_tags_related_skills` | `partial` | `194` | `Agent Hooks Registry`, `Backlog split C5: single atomic operator-gated flip to the module-keyed split directory`, `Backlog split C5a: optional per-row module key + deterministic derivation + backfill`, `Blocker Policy Integration`, `+35 more` | - | `optional-skills/autonomous-ai-agents/agent-merge-conflict-arbiter/SKILL.md`, `optional-skills/autonomous-ai-agents/antigravity-cli/SKILL.md`, `optional-skills/autonomous-ai-agents/blackbox/SKILL.md`, `optional-skills/autonomous-ai-agents/dynamic-workflow/SKILL.md`, `optional-skills/autonomous-ai-agents/grok/SKILL.md`, `+189 more` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |

## Gateway Platform Classification

| Family | Status | Count | Progress rows | Source pairs | Examples | Reason |
|---|---|---:|---|---|---|---|
| `api_server_surface` | `partial` | `1` | `16ms coalescing mailbox`, `API server cron admin mutating endpoints`, `API server cron admin read-only endpoints`, `API server detailed health endpoint`, `+363 more` | `agent/transports/chat_completions.py` | `gateway/platforms/api_server.py` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `builtin_platform_connectors` | `partial` | `38` | `61-tool registry port`, `ACP Client Bridge Mode`, `ACP JSON-RPC stdio session/prompt closeout`, `ACP server side`, `+478 more` | `gateway/run.py`, `tools/web_tools.py` | `gateway/platforms/__init__.py`, `gateway/platforms/_shared.py`, `gateway/platforms/access_policy_mixin.py`, `gateway/platforms/api_server_authority_runs.py`, `gateway/platforms/api_server_bot_chat.py`, `+33 more` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `bundled_platform_plugins` | `partial` | `188` | `ACP bridge doctor/status evidence`, `ACP stdio benign ping/probe suppression`, `Admin TUI: Agents screen wired to the 2.H dynamic registry`, `Admin TUI: Chat tab with keybinding to jump in from any screen`, `+363 more` | - | `plugins/platforms/a2a/DESIGN.md`, `plugins/platforms/a2a/README.md`, `plugins/platforms/a2a/__init__.py`, `plugins/platforms/a2a/adapter.py`, `plugins/platforms/a2a/plugin.yaml`, `+183 more` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `gateway_runtime_lifecycle` | `partial` | `1` | `ACP JSON-RPC stdio session/prompt closeout`, `ACP bridge doctor/status evidence`, `ACP server side`, `ACP stdio benign ping/probe suppression`, `+598 more` | `agent/lsp/manager.py`, `gateway/run.py`, `hermes_cli/auth_commands.py` | `gateway/run.py` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `platform_docs` | `partial` | `1` | `Backlog split C5e: make non-Go raw progress.json consumers (fleet scripts + CI path globs) split-directory-safe`, `Bundled TouchDesigner MCP skill catalog contract`, `CI and installer Go toolchain floor sync`, `Cron prompt/script safety + pre-run script contract (deprecated umbrella)`, `+33 more` | - | `gateway/platforms/ADDING_A_PLATFORM.md` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `platform_enum_config` | `partial` | `1` | `Agent Hooks Registry`, `Approval mode config normalization`, `Bundled platform plugin manifest drift guard`, `Gateway /kanban shared command-runner binding`, `+25 more` | - | `gateway/config.py` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `platform_helpers` | `partial` | `3` | `API server detailed health snapshot contract`, `Admin TUI: Chat tab with keybinding to jump in from any screen`, `Anthropic OAuth/keychain credential discovery`, `Atomic file write helper with temp+rename pattern`, `+352 more` | `tools/url_safety.py` | `gateway/platforms/_http_client_limits.py`, `gateway/platforms/base.py`, `gateway/platforms/helpers.py` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `tui_gateway_bridge` | `partial` | `124` | `ACP Client Bridge Mode`, `ACP JSON-RPC stdio session/prompt closeout`, `ACP bridge doctor/status evidence`, `ACP server side`, `+407 more` | `tools/skills_tool.py` | `tui_gateway/AGENTS.md`, `tui_gateway/__init__.py`, `tui_gateway/_env.py`, `tui_gateway/_stdin_recovery.py`, `tui_gateway/agent_callbacks.py`, `+119 more` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |

## Web Dashboard Classification

| Family | Status | Count | Progress rows | Source pairs | Examples | Reason |
|---|---|---:|---|---|---|---|
| `cron_admin_jobs` | `partial` | `1` | `API server cron admin mutating endpoints`, `API server cron admin read-only endpoints`, `API server detailed health endpoint`, `API server detailed health snapshot contract`, `+106 more` | `gateway/run.py` | `web/src/pages/CronPage.tsx` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `gateway_client_events` | `partial` | `2` | `61-tool registry port`, `ACP JSON-RPC stdio session/prompt closeout`, `ACP server side`, `ACP session CWD propagation into prompt runners`, `+519 more` | `agent/context_engine.py`, `agent/conversation_loop.py`, `agent/prompt_builder.py`, `run_agent.py` | `web/src/lib/api.ts`, `web/src/lib/gatewayClient.ts` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `i18n_catalog` | `partial` | `22` | `ACP JSON-RPC stdio session/prompt closeout`, `Block-anchor fuzzy replace for native patch tool`, `Gateway channel adapters publish to event bus`, `Gormes setup/channel/provider docs webpage parity gate`, `+10 more` | - | `web/src/components/LanguageSwitcher.tsx`, `web/src/i18n/af.ts`, `web/src/i18n/ar.ts`, `web/src/i18n/context.tsx`, `web/src/i18n/de.ts`, `+17 more` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `model_picker` | `partial` | `3` | `Custom provider model-switch credential preservation`, `Custom provider model-switch key_env write guard`, `Dashboard API client contract`, `Discord interaction authorization + mention safety guards`, `+24 more` | - | `web/src/components/ModelInfoCard.tsx`, `web/src/components/ModelPickerDialog.tsx`, `web/src/pages/ModelsPage.tsx` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `oauth_provider_panels` | `partial` | `2` | `ACP JSON-RPC stdio session/prompt closeout`, `API server detailed health endpoint`, `Anthropic OAuth/keychain credential discovery`, `Auth state TOCTOU close + redaction default-on parity`, `+120 more` | `hermes_cli/auth_commands.py`, `tools/x_search_tool.py` | `web/src/components/OAuthLoginModal.tsx`, `web/src/components/OAuthProvidersCard.tsx` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `plugin_pages_slots` | `partial` | `11` | `Agent Hooks Registry`, `Dashboard page-scoped plugin slot inventory`, `Dashboard theme/plugin extension status contract`, `Effective toolset picker dedupes bundled plugin keys`, `+12 more` | - | `web/src/pages/PluginsPage.tsx`, `web/src/plugins/PluginPage.tsx`, `web/src/plugins/index.ts`, `web/src/plugins/registry.test.ts`, `web/src/plugins/registry.ts`, `+6 more` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `profiles_config` | `partial` | `3` | `Active Hermes/Sidon profile context root resolver for live turns`, `Admin TUI: Setup health screen with missing-config callouts`, `Agent personalities + enhanced display config`, `Agent personalities + enhanced display config`, `+78 more` | `hermes_cli/profiles.py` | `web/src/components/PlatformsCard.tsx`, `web/src/pages/ConfigPage.tsx`, `web/src/pages/ProfilesPage.tsx` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `sessions_page` | `partial` | `1` | `Dashboard API client contract`, `Ephemeral prefill messages file injection`, `Gateway auto-resume on restart`, `Hermes background review fork lifecycle`, `+2 more` | - | `web/src/pages/SessionsPage.tsx` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `terminal_chat_pty` | `partial` | `3` | `ACP session CWD propagation into prompt runners`, `API server multimodal content preservation`, `Admin TUI: Agents screen wired to the 2.H dynamic registry`, `Admin TUI: Chat tab with keybinding to jump in from any screen`, `+310 more` | - | `web/src/components/ChatSidebar.tsx`, `web/src/components/Markdown.tsx`, `web/src/pages/ChatPage.tsx` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |
| `theme_system` | `partial` | `7` | `Dashboard page-scoped plugin slot inventory`, `Dashboard theme/plugin extension status contract`, `Gormes MCP custom HTTP config add + redacted list`, `Gormes setup full-wizard shell and branded summary`, `+9 more` | - | `web/src/components/ThemeSwitcher.tsx`, `web/src/themes/context.tsx`, `web/src/themes/fonts.ts`, `web/src/themes/index.ts`, `web/src/themes/presets.test.ts`, `+2 more` | Some source-pair or complete-row evidence exists, but this plugin family is not strictly covered. |

## Candidate Inventory

| Candidate family | Count | Examples |
|---|---:|---|
| CLI | `814` | `cli.py`, `hermes_cli/AGENTS.md`, `hermes_cli/__init__.py`, `hermes_cli/_early_recovery.py`, `hermes_cli/_early_recovery_zip.py`, `+809 more` |
| Tools | `369` | `tools/AGENTS.md`, `tools/__init__.py`, `tools/annotate_preview_tool.py`, `tools/ansi_strip.py`, `tools/approval.py`, `+364 more` |
| Providers | `461` | `agent/agent_init_memory_provider.py`, `agent/browser_provider.py`, `agent/delegation_provider.py`, `agent/image_gen_provider.py`, `agent/memory_provider.py`, `+456 more` |
| Channels | `661` | `apps/desktop/src/app/messaging/telegram-qr-setup.tsx`, `evals/slack_stream_wire_contract.py`, `gateway/AGENTS.md`, `gateway/__init__.py`, `gateway/active_lease_io.py`, `+656 more` |
| Sessions | `1191` | `acp_adapter/gateway_session_meta.py`, `acp_adapter/session.py`, `agent/onboarding_session.py`, `agent/runtime_session_child.py`, `agent/runtime_session_compression.py`, `+1186 more` |
| Memory | `134` | `agent/agent_init_memory_provider.py`, `agent/memory_caller_scope.py`, `agent/memory_context.py`, `agent/memory_manager.py`, `agent/memory_manager_tools.py`, `+129 more` |
| Skills | `1449` | `agent/skill_bundles.py`, `agent/skill_commands.py`, `agent/skill_preprocessing.py`, `agent/skill_utils.py`, `agent/skills_index_delta.py`, `+1444 more` |
| Learning loop | `37` | `agent/curator.py`, `agent/curator_backup.py`, `agent/learning_graph.py`, `agent/learning_graph_render.py`, `agent/learning_mutations.py`, `+32 more` |

## Surface Classification

| Surface | Status | Severity | Confidence | Progress rows | Source pairs | Reason |
|---|---|---|---|---|---|---|
| `profiles` | `partial` | `warning` | `medium` | `Active Hermes/Sidon profile context root resolver for live turns`, `Agent personalities + enhanced display config`, `Agent personalities + enhanced display config`, `CLI active-profile store`, `+35 more` | `agent/lsp/manager.py`, `agent/prompt_builder.py`, `hermes_cli/config.py`, `hermes_cli/default_soul.py`, `+2 more` | Some source-pair or complete-row evidence exists, but the surface is not strictly covered. |
| `sessions` | `partial` | `warning` | `medium` | `Auto-naming sessions`, `Aux compression single-prompt threshold reconciliation`, `Behavioral pattern extraction from session logs`, `Busy command guard for compression and long CLI actions`, `+33 more` | `agent/context_engine.py`, `agent/prompt_builder.py`, `agent/transports/chat_completions.py`, `agent/transports/codex.py`, `+2 more` | Some source-pair or complete-row evidence exists, but the surface is not strictly covered. |
| `goncho_memory` | `partial` | `warning` | `medium` | `2-layer seed selection`, `6 typed memory categories with confidence scoring`, `<memory-context> fence`, `Agent-controlled memory retention with importance scoring`, `+74 more` | `agent/prompt_builder.py` | Some source-pair or complete-row evidence exists, but the surface is not strictly covered. |
| `learning_loop` | `partial` | `warning` | `medium` | `Agent-controlled memory retention with importance scoring`, `Code Cathedral II code-context retrieval fixtures`, `Goncho durable recall trace IR + fused ranking pipeline`, `Goncho golden transcript e2e harness`, `+22 more` | `agent/lsp/manager.py`, `tools/skill_manager_tool.py` | Some source-pair or complete-row evidence exists, but the surface is not strictly covered. |
| `prompt_assembly` | `partial` | `warning` | `medium` | `<memory-context> fence`, `Agent lifecycle hooks (agent:start, agent:step, agent:end)`, `Aux compression single-prompt threshold reconciliation`, `Bundled Airtable productivity skill contract`, `+45 more` | `agent/context_engine.py`, `agent/conversation_loop.py`, `agent/prompt_builder.py`, `agent/tool_executor.py`, `+3 more` | Some source-pair or complete-row evidence exists, but the surface is not strictly covered. |
| `provider_auth_setup` | `partial` | `warning` | `medium` | `Anthropic`, `Anthropic OAuth/keychain credential discovery`, `Aux compression provider-aware context cap`, `Azure Anthropic Messages endpoint contract`, `+168 more` | `agent/context_engine.py`, `agent/prompt_builder.py`, `agent/transports/chat_completions.py`, `agent/transports/codex.py`, `+10 more` | Some source-pair or complete-row evidence exists, but the surface is not strictly covered. |
| `gateway_channels` | `partial` | `warning` | `medium` | `(platform, chat_id) -> session_id`, `API server cron admin mutating endpoints`, `API server cron admin read-only endpoints`, `API server detailed health endpoint`, `+303 more` | `agent/prompt_builder.py`, `gateway/run.py`, `ui-tui/package.json` | Some source-pair or complete-row evidence exists, but the surface is not strictly covered. |
| `tool_runtime` | `partial` | `warning` | `medium` | `61-tool registry port`, `ACP Client Bridge Mode`, `ACP JSON-RPC stdio session/prompt closeout`, `ACP server side`, `+168 more` | `agent/conversation_loop.py`, `agent/lsp/manager.py`, `agent/tool_executor.py`, `agent/transports/chat_completions.py`, `+13 more` | Some source-pair or complete-row evidence exists, but the surface is not strictly covered. |
| `mcp_acp` | `partial` | `warning` | `medium` | `ACP Client Bridge Mode`, `ACP JSON-RPC stdio session/prompt closeout`, `ACP server side`, `ACP session CWD propagation into prompt runners`, `+41 more` | `tools/web_tools.py` | Some source-pair or complete-row evidence exists, but the surface is not strictly covered. |
| `tui_cli` | `partial` | `warning` | `medium` | `16ms coalescing mailbox`, `Admin TUI: Agents screen wired to the 2.H dynamic registry`, `Admin TUI: Chat tab with keybinding to jump in from any screen`, `Admin TUI: Commands catalog over the root CLI tree`, `+86 more` | `agent/transports/chat_completions.py`, `agent/transports/codex.py`, `cli.py`, `gateway/run.py`, `+8 more` | Some source-pair or complete-row evidence exists, but the surface is not strictly covered. |
