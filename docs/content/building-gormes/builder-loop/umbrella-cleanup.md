---
title: "Umbrella Cleanup"
weight: 50
aliases:
  - /building-gormes/umbrella-cleanup/
---

# Umbrella Cleanup

Umbrella rows are inventory or tracking rows, not executable implementation
slices. Split these into smaller rows with contracts, fixtures, trust classes,
and acceptance checks before assigning them to an implementation agent.

<!-- PROGRESS:START kind=umbrella-cleanup -->
| Phase | Umbrella row | Owner | Not ready when | Split into |
|---|---|---|---|---|
| 2 / 2.B.11 | Discord forum media + polish parity | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 2 / 2.F.4 | Notify-to delivery routing | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 2 / 2.F.4 | Channel directory atomic persistence + lookup | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 2 / 2.F.4 | Channel directory refresh + stale-target invalidation | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 2 / 2.F.4 | Manager remember-source hook | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 2 / 2.F.4 | Mirror + sticker cache surfaces | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 4 / 4.A | Bedrock | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 4 / 4.A | Gemini | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 4 / 4.A | OpenRouter | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 4 / 4.A | Google Code Assist | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 4 / 4.A | Codex | `provider` | The row is assigned as one large Codex provider implementation instead of the Responses conversion, auth, and stream-repair slices below. | Codex Responses pure conversion harness, Codex Responses assistant content role types, Codex Responses HTTP client binding, Codex OAuth state + stale-token relogin, Codex stream repair + tool-call leak sanitizer |
| 4 / 4.B | Long session management | `provider` | The row is assigned as one implementation task instead of being split through context engine, token-budget, reference, and compression slices. | ContextEngine interface + status tool contract, Compression token-budget trigger + summary sizing, Manual compression feedback + context references |
| 4 / 4.B | Context compression | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 4 / 4.B | Tool-result pruning + protected head/tail summary | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 4 / 4.B | Manual compression feedback + context references | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 4 / 4.C | System + memory + tools + history assembly | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 4 / 4.C | Model-specific role and tool-use guidance | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 4 / 4.C | Toolset-aware skills prompt snapshot | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 4 / 4.C | Memory and session-search guidance assembly | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 4 / 4.D | Model metadata registry + context limits | `provider` | The row is assigned as one metadata/routing implementation instead of the resolver, pricing/capability, and selector slices below. | Provider-enforced context-length resolver, Model pricing/capability registry fixtures, Routing policy and fallback selector |
| 4 / 4.E | Trajectory writer + redaction gates | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 4 / 4.G | Token vault | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 4 / 4.G | Multi-account auth | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 4 / 4.H | Prompt-cache capability guard | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.A | 61-tool registry port | `tools` | The row is treated as a bulk 61-handler port before descriptor parity and trust classes are frozen. | Tool registry inventory + schema parity harness, Pure core tools first, Stateful tool migration queue |
| 5 / 5.A | Pure core tools first | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.A | Stateful tool migration queue | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.B | Docker | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.B | Modal | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.B | Daytona | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.B | Singularity | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.C | Browser action contract + event transcript | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.C | Chromedp | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.C | Rod | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.C | Browser provider bridge + Firecrawl fallback | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.D | Multimodal in/out | `provider` | The row is assigned directly instead of the smaller image-routing, image-shrink, and image-generation contracts below. | Image input mode router + native content parts, Image-too-large shrink retry helper, Image generation result contract |
| 5 / 5.E | Voice mode port | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.E | Transcription tool contract | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.E | TTS synthesis + voice-mode state | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.F | Skill registries | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.G | MCP client | `tools` | The row is assigned as one all-MCP migration instead of the config, discovery, OAuth, and managed-gateway slices below. | MCP server config/env resolver, MCP stdio transport + tool/list discovery, MCP HTTP transport + tool/list discovery, MCP schema normalization + structured-content adapter, MCP OAuth state store + noninteractive auth errors, Managed tool gateway bridge |
| 5 / 5.I | Third-party extensions | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.J | Dangerous action gating | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.J | Approval mode config normalization | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.J | Cron dangerous-command approval mode | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.J | Tirith, path, URL, and website policy integration | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.L | Atomic checkpoints | `tools` | The row is assigned as one checkpoint/file/patch migration instead of the checkpoint policy, read-dedup guard, and patch/write slices below. | Checkpoint shadow-repo GC policy, File read dedup cache invalidation and wrapper guard |
| 5 / 5.M | Multi-model coordination | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.N | Todo | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.N | Clarify | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.N | Debug helpers | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.N | Cronjob tool API + schedule parser parity | `tools` | The row is assigned as one combined cronjob tool, schedule parser, safety, context chaining, and delivery-port slice instead of the dependency-ordered rows below. | Cron schedule parser + repeat state fixtures, Cron prompt/script safety + pre-run script contract, Cronjob tool action envelope over native store, Cron context_from output chaining, Cron multi-target delivery + media/live-adapter fallback |
| 5 / 5.N | Cron prompt/script safety + pre-run script contract (deprecated umbrella) | `tools` | The row is assigned directly instead of the smaller `Cron prompt/script safety + pre-run script contract` helper row above. | - |
| 5 / 5.O | 49-file CLI tree port | `tools` | The row is assigned as a whole hermes_cli tree migration instead of command-group slices. | Hermes CLI command-tree parity manifest, Deterministic helper-file ports (banner/output/tips/webhook/dump), CLI command registry parity + active-turn busy policy, Config, profile, auth, and setup command surfaces, Gateway, platform, webhook, and cron management CLI, Diagnostics, backup, logs, and status CLI |
| 5 / 5.O | Deterministic helper-file ports (banner/output/tips/webhook/dump) | `tools` | The row is assigned as one combined hermes_cli helper-file migration instead of the four pure-helper slices below. | CLI banner/output formatting helpers, CLI deterministic tip selector, CLI webhook URL normalizer, CLI dump support-summary helper |
| 5 / 5.O | Config, profile, auth, and setup command surfaces | `tools` | The row is assigned as one combined config/profile/auth/setup migration instead of the pure profile, auth-status, setup, and uninstall slices. | CLI profile name validator, CLI profile root resolver, CLI active-profile store, Provider endpoint/API-key root flags + runtime resolution, Gormes config command surface, Hermes config migration dry-run manifest, OpenClaw migration dry-run manifest, CLI auth status read model before provider setup, Setup/uninstall dry-run command contracts |
| 5 / 5.O | CLI profile path and active-profile store (deprecated umbrella) | `tools` | The row is selected at all — execute the three sibling rows above (CLI profile name validator, CLI profile root resolver, CLI active-profile store) instead. | - |
| 5 / 5.O | Gateway, platform, webhook, and cron management CLI | `tools` | The row is assigned as one management-CLI migration instead of separate gateway read-model, cron admin, webhook helper, and platform command slices. | Gateway management CLI read-model closeout, Cron management CLI over native store, Webhook/platform management CLI helpers |
| 5 / 5.O | Diagnostics, backup, logs, and status CLI | `tools` | The row is assigned as one combined diagnostics/backup/logs/status migration instead of log snapshot, status summary, backup manifest, and optional upload slices. | CLI log snapshot reader, CLI status summary over native stores, Backup manifest dry-run contract |
| 5 / 5.P | Unix installer (install.sh) source-backed update flow | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.P | Windows installer (install.ps1 + install.cmd) parity | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.P | Installer site asset/route coverage | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.Q | Deterministic helper-file ports (tool-progress/image/completion-path/personality/platform-event) | `gateway` | The row is assigned as one combined tui_gateway helper migration instead of the two pure-helper slices below. | TUI gateway progress/completion helpers, TUI gateway image/personality/platform-event helpers |
| 5 / 5.R | Execution-mode resolver + config precedence | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.R | Strict-mode CWD + interpreter parity | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.R | Project-mode CWD + active venv detection | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 5 / 5.R | Default mode selection + config cut-over | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 6 / 6.A | Heuristic or LLM-scored signal | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 6 / 6.E | Skill effectiveness scoring | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 6 / 6.F | TUI + Telegram browsing | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 7 / 7.A | Signal transport/bootstrap layer | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 7 / 7.C | Matrix shared-chassis bot seam | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 7 / 7.C | Mattermost shared-chassis bot seam | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 7 / 7.C | Matrix real client/bootstrap layer | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 7 / 7.C | Mattermost REST/WS bootstrap layer | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 7 / 7.E | Feishu transport/bootstrap layer | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 7 / 7.E | Feishu drive-comment rule + pairing seam | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 7 / 7.E | Feishu drive-comment reply workflow | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 7 / 7.E | DingTalk real SDK binding | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
| 7 / 7.E | QQ Bot transport/bootstrap layer | `docs` | The row is inventory-only and must not be selected until a planner pass replaces this note with a builder-ready contract packet. | - |
<!-- PROGRESS:END -->
