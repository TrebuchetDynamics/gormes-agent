---
title: "Hermes v0.14 Module Pairings"
description: "Release-level map from Hermes v0.14 user-visible behavior to Gormes modules and progress rows."
date: 2026-05-17
draft: false
aliases:
  - /building-gormes/architecture_plan/hermes-v0.14-module-pairings/
---

# Hermes v0.14 Module Pairings

This ledger pairs the user-visible Hermes v0.14 release surface to the
Gormes module taxonomy. It is not a second backlog: missing or vague work is
owned by rows in `progress.json`, and generated Go from py2many remains
reference evidence only.

The current py2many Go target command is `py2many --go <file.py>`. The older
`--go=1` spelling should not be copied into Gormes docs or automation.

## Pairing Rules

- Pair behavior to the module that owns the Go runtime contract, not to the
  Python source directory name.
- Mark Gormes-owned divergences explicitly; do not hide Hermes parity drift
  behind the Go implementation shape.
- Use source-backed rows for behavior that is not already covered by tests.
- Keep py2many output outside `cmd/`, `internal/`, and `pkg/`; it can expose
  constants, schemas, pure helper boundaries, and manual-rewrite seams.

## Release Surface Map

| Hermes v0.14 surface | Primary upstream refs | Gormes module | Pairing decision |
|---|---|---|---|
| SuperGrok/xAI OAuth, Grok context and entitlement handling | `agent/credential_sources.py`, `agent/model_metadata.py`, `plugins/image_gen/xai`, `plugins/video_gen/xai` | `providers` | Covered or row-backed by xAI/Grok provider, provider registry, OAuth, and model metadata rows. |
| OpenAI-compatible local proxy for OAuth providers | `hermes_cli/proxy/*`, `gateway/platforms/api_server.py` | `gateway` | Covered by gateway proxy mode and OpenAI-compatible API rows; proxy CLI surface remains source-paired through CLI/gateway rows. |
| First-class `x_search` tool | `tools/x_search_tool.py`, `tools/xai_http.py`, `website/docs/user-guide/features/x-search.md` | `tools` | Added planned row `Hermes x_search tool and auth surface`. |
| Microsoft Teams / Graph stack | `plugins/platforms/google_chat`, `gateway/platforms/msgraph_webhook.py`, Teams plugin files | `channels` | Covered or row-backed by MSGraph and Teams channel rows. |
| Lazy dependencies, tiered install, PyPI, supply-chain audit | `pyproject.toml`, `scripts/install.*`, `.github/workflows/supply-chain-audit.yml`, `.github/workflows/upload_to_pypi.yml` | `install`, `release`, `doctor` | Covered or row-backed by installer, release, and doctor section-content rows. |
| Cross-session Claude prompt cache | `agent/prompt_caching.py`, `agent/context_compressor.py` | `providers`, `sessions` | Covered by prompt-cache/retry and compression rows; new compressor drift is tracked by existing session rows. |
| Browser CDP speedup and browser bootstrap for ACP installs | `tools/browser_*.py`, `acp_adapter/bootstrap/*` | `browser`, `tools` | Browser runtime is row-backed; added planned row `ACP setup-browser bootstrap parity`. |
| LINE and SimpleX platform plugins | `plugins/platforms/line/*`, `plugins/platforms/simplex/*` | `channels`, `gateway` | LINE is visible in the platform manifest; added planned row `SimpleX Chat platform plugin parity`. |
| `/handoff`, `/subgoal`, session transfer and recap | `cli.py`, `hermes_cli/session_recap.py`, `run_agent.py` | `sessions`, `cli` | Session transfer is row-backed; added planned row `Hermes session recap command surface`. |
| Profile-scoped gateway processes and profile operations | `hermes_cli/profiles.py`, `gateway/run.py`, `website/docs/reference/cli-commands.md` | `profiles`, `gateway` | Added planned row `Long-term plan: profile fleet supervisor and single control-plane gateway` while preserving Hermes-compatible per-profile isolation. |
| Native clarify buttons and Discord history backfill | `gateway/platforms/telegram.py`, `gateway/platforms/discord.py` | `channels` | Covered or row-backed under channel delivery and Discord/Telegram interaction rows. |
| Native vision pixels, unified video generation, computer-use driver | `tools/vision_tools.py`, `tools/video_generation_tool.py`, `tools/computer_use*` | `tools`, `browser`, `providers` | Vision and computer-use are row-backed; video generation remains under tool/plugin rows. |
| File-mutation verifier and LSP diagnostics on write/patch | `agent/lsp/*`, `tools/file_tools.py`, `tools/patch_parser.py` | `tools` | Added planned row `Hermes LSP write-time semantic diagnostics`. |
| Terminal OSC8/truecolor/Terminal.app hardening | `ui-tui/src/lib/forceTruecolor.ts`, `ui-tui/src/lib/text.ts`, `ui-tui/src/components/textInput.tsx` | `tui` | Added planned row `Native TUI Terminal.app truecolor and ANSI sanitizer parity`. |
| ACP registry / Zed / Copilot ACP | `acp_adapter/*`, `acp_registry/agent.json`, `agent/copilot_acp_client.py` | `tools`, `providers`, `doctor` | ACP server/client/doctor rows exist; browser bootstrap is newly row-backed here. |
| OpenRouter Pareto, NovitaAI, Qwen rename, provider metadata | `hermes_cli/providers.py`, `agent/model_metadata.py`, `model_tools.py` | `providers` | Covered by provider registry, OpenRouter Pareto, NovitaAI, Qwen, and model-picker rows. |
| Optional skills and trusted skill taps | `skills/`, `optional-skills/`, `website/docs/reference/optional-skills-catalog.md` | `skills` | Added planned row `Hermes v0.14 optional skill catalog refresh`. |
| Brave/DDGS search providers | `tools/web_providers/*`, `tools/web_tools.py` | `browser` | Covered by `Brave Search + DDGS web search provider parity`. |
| Security hardening and tool-error sanitization | `tools/approval.py`, `tools/tirith_security.py`, `tests/test_sanitize_tool_error.py` | `tools`, `doctor` | Covered or row-backed by dangerous-command, security audit, and sanitizer rows. |
| Native Windows beta | `scripts/install.ps1`, `hermes_cli/*`, `ui-tui/*` | `install`, `runtime`, `tui`, `gateway` | Covered or row-backed by Windows install/runtime/gateway/TUI rows. |
| `hermes send` payload and TUI resume fixes | `hermes_cli/send_cmd.py`, `hermes_cli/main.py`, `tests/hermes_cli/test_send_cmd.py`, `tests/hermes_cli/test_tui_resume_flow.py` | `cli`, `tui` | Added planned row `Hermes send command stdin/file payload parity`; TUI resume/ANSI hardening is tracked under the TUI row above. |
| Gateway memory monitor | `gateway/memory_monitor.py`, `tests/gateway/test_memory_monitor.py` | `gateway`, `memory` | Added planned row `Gateway memory monitor pressure policy`. |

## Module Additions From This Pass

The generated `building-gormes/modules` pages should now show new planned
rows in `cli`, `channels`, `gateway`, `sessions`, `skills`, `tools`, and
`tui`; a long-term profile-fleet row in `profiles`; plus this completed
`docs` ledger row.
