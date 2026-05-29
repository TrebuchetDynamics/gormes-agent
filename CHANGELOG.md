# Changelog

All notable changes to Gormes-Agent are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
inside the 0.x compatibility window.

## [Unreleased]

## [0.2.23] - 2026-05-25

Date alias: `v2026.5.25`.

> **Local Router, embeddable RPC mode, safer native file tools, and Navivox capability hardening.**

### Added
- Local OpenAI-compatible Router setup, status, health, model listing, non-streaming and SSE chat-completions surfaces with redacted recursion diagnostics and safe fallback behavior.
- Gormes-owned stdin/stdout JSONL RPC mode for embedders, including LF-only framing, correlated responses, fake-runtime fixtures, structured errors, and deterministic lifecycle event streaming.
- Native session tree navigation with labels/bookmarks, lineage rendering, branch/resume adapters, and safe prompt restore support.
- Progress projection read models and write dry-run support for safer generated-roadmap maintenance.

### Changed
- Native write/edit/patch tools now serialize mutations per canonical file path while preserving independent-file concurrency and symlink alias safety.
- Navivox capability documents now describe only callable/current client gates; setup handoff stays on status/connect surfaces and unavailable upload behavior is expressed as an explicit exclusion rather than a reserved endpoint.
- Progress, README, docs, and landing mirrors were refreshed from the current validated backlog evidence.

### Fixed
- Installed `SOUL.md` stays lean by keeping project workflow rules in repository guidance instead of user-facing agent templates.
- Blocked progress rendering and generated artifact planning now use shared workitem/projection helpers instead of duplicated row selection logic.
- Navivox profile capability wording now avoids stale compatibility promises, raw local path hints, and secret-bearing capability output.

## [0.2.22] - 2026-05-23

Date alias: `v2026.5.23`.

> **Native TUI slash parity, Navivox profile/voice APIs, and porting-kit publication prep.**

### Added
- Native TUI slash handlers and adapter seams for local `/tools`, `/voice`, `/skin`, session, logs, status, details, history, usage, and title workflows.
- Navivox backend APIs for redacted voice run records, natural-language profile seed drafts, safe config administration, and per-profile voice-provider validation.
- Agentic porting-kit example scaffolding, schema validation, local/public layout checks, and engineering writeup evidence packets.

### Changed
- Progress, docs, landing, and benchmark mirrors were refreshed from the split backlog and current release evidence.
- Gateway/profile setup surfaces now expose clearer profile, fleet, SimpleX, and Navivox readiness state for operator workflows.

### Fixed
- Local release hygiene now ignores `.env.backup.*` and `.pi/development-goal/` so credential backups and run logs cannot be staged accidentally.
- Update/install/docs coverage now checks release freshness, installer asset sync, and public surface drift around the release lane.

## [0.2.21] - 2026-05-22

Date alias: `v2026.5.22`.

> **Termux installer recovery, WhatsApp profile setup readiness, and Navivox pairing handoff.**

### Added
- WhatsApp profile-channel setup readiness now reports access-policy state, pairing status, degraded pairing readouts, duplicate token-hash conflicts, group/direct chat counts, and canonicalized user JID forms without leaking tokens, user IDs, or chat IDs.
- Navivox setup now writes an owner-only QR PNG pairing handoff containing the HTTP/WebSocket descriptor and REST token while terminal output stays token-redacted.
- Navivox profile routing and F-Droid preparation docs now carry local gateway, first-run, screenshot, metadata, artifact, privacy, and data-safety evidence.
- `gormes setup providers` now works as a plural alias for `gormes setup provider`.

### Changed
- Setup provider non-interactive mode now honors `GORMES_INFERENCE_PROVIDER` and `GORMES_INFERENCE_MODEL` fallbacks alongside the existing endpoint/API-key flow.
- Memory, Goncho, learning-loop, provider, Navivox, and release progress surfaces were refreshed from the split backlog.

### Fixed
- Termux latest-release installer recovery now normalizes `/data/data` versus `/data/user/0` executable-path aliases before Cobra parsing and falls back from Termux binary-fetch publish verification failures to one source-build retry.
- Navivox Bubble Tea gateway selection now routes through the native Navivox setup path instead of the generic row-backed fallback.

## [0.2.20] - 2026-05-21

Date alias: `v2026.5.21`.

> **Provider credential fallbacks, native TUI slash parity, and Navivox channel hardening.**

### Added
- Native TUI `/help`, `/clear`/`/new`, and `/compact` slash handlers with regression coverage for local dispatch, busy-time visibility, reset behavior, and compact transcript rendering.
- Provider credential-pool and model-catalog coverage for setup/auth flows, including fallback-provider documentation refreshes.
- Navivox gateway channel authentication, streaming, turn handling, and memory overview helpers with focused tests.
- Dashboard image design skill routing for future screenshot/visual asset work.

### Changed
- Slash-command ownership, gateway recognition, and generated progress surfaces now track the newly ported TUI handlers.
- Provider setup docs and fallback-provider operations guidance now reflect the credential-pool/provider readiness path.
- Development-loop evidence is captured in the repo-local `.pi/development-loop/logs.jsonl` log.

### Fixed
- Native TUI reset and compact commands no longer fall through to recognized-unavailable evidence or model submission.
- Provider setup/auth tests now cover manual API-key persistence and credential-pool authorization selection.
- Navivox channel code is split into focused auth, stream, turn, and memory-overview helpers for clearer gateway behavior.

## [0.2.19] - 2026-05-20

Date alias: `v2026.5.20`.

> **Navivox VPN setup, deploy hardening, and release freshness gates.**

### Added
- Navivox connect-info now reports WebSocket stream URLs and brackets IPv6 hosts correctly in generated connection metadata.
- Navivox setup now supports WireGuard and generic VPN exposure modes with detected bind defaults.

### Changed
- Docs and landing deploy smoke checks now assert the current generated page copy instead of stale pre-redesign strings.
- `gormes-release` now requires post-merge main CI/CD checks and README/public-status freshness before tagging.
- README public release status now tracks the current public release and benchmark-derived binary size.

### Fixed
- Restored green `Deploy docs.gormes.ai` and `Deploy gormes.ai` paths after stale smoke-check assertions blocked CD.

## [0.2.18] - 2026-05-20

Date alias: `v2026.5.20`.

> **Public Goncho/Goscrapling integration, TUI hardening, and release-path cleanup.**

### Added
- Public Goncho module integration with Gormes tool registration and a module-graph guard that rejects local `replace` dependencies.
- Public Goscrapling release guard coverage proving Gormes consumes the tagged GitHub module rather than a sibling checkout.
- Hard E2E and TUI regression coverage for complex chat/setup flows, cramped terminal rendering, setup profiles, admin health/fix screens, and full `Model.View()` wrapping.
- Repo-local workflow, review-gate, architecture-review, refactor, and Navivox Telegram UI skills for safer bounded delivery.

### Changed
- Navivox gateway and protocol work now focus on the Go gateway/runtime path; the tracked Flutter prototype was removed from the release branch.
- Release and git skills now document the mandatory `development` -> PR -> `main` -> tag path and stronger dirty-work/CI safety rules.
- Module release guard tests now share a reusable `internal/support/testutil/modassert` helper.

### Fixed
- Bubble Tea setup/admin/profile/chat views now preserve prompts, selected rows, help text, status lines, and resize guidance in cramped terminals.
- CI no longer runs obsolete Navivox Flutter steps after the prototype removal.
- Architecture-review guidance now requires evidence quality, score calibration, stop gates, and validation-matrix proof before refactor implementation.

## [0.2.17] - 2026-05-18

Date alias: `v2026.5.18`.

> **Local voice transcription and voice tool-progress hardening.**

### Fixed
- Local Telegram Whisper discovery now checks `~/.local/bin/whisper` when
  service-managed PATH values omit user-local binaries.
- Local STT now converts OGG and other supported audio inputs into temporary
  WAV chunks before WASI Whisper transcription, improving Telegram voice-note
  handling.
- Gateway and channel tool-progress previews now recognize
  `transcribe_audio` and `text_to_speech` as voice tools while suppressing raw
  audio paths and speech text from compact progress messages.

## [0.2.16] - 2026-05-18

Date alias: `v2026.5.18`.

> **Profile config v2 groundwork, memory provenance inventory, and Telegram audio hardening.**

### Added
- Root `config.toml` v2 profile service schema with `config_version = 2`,
  `[profiles.main]` seed data, all-enabled profile enumeration, per-profile
  provider/channel references, and a global SecretRef-backed credential
  registry.
- Memory provenance inventory in `gormes memory status`, separating Goncho
  state, durable Markdown memory, legacy Hermes memory, context files, and
  session transcript evidence.

### Changed
- Setup, config edit/check/migrate, and doctor schema-fix flows now write and
  report canonical `config_version` v2 data while keeping legacy
  `_config_version` files readable.
- Profile control center planning and generated progress docs now reflect the
  single-root config design with no global active/default profile key.

### Fixed
- Telegram inbound audio/transcription handling now preserves source-backed
  audio evidence for STT/TTS workflows.
- Profile-session audit bundles now capture recent agent response issues more
  reliably for memory, learning-loop, tool, and response-quality review.

## [0.2.15] - 2026-05-18

Date alias: `v2026.5.18`.

> **Goncho recall proof, Hermes fidelity gates, durable run reports, and compact tool output.**

### Added
- Goncho recall diagnostics, replay traces, benchmark corpus coverage, proof
  matrix reporting, TOON prompt context encoding, and end-to-end memory-turn
  proof coverage.
- Hermes fidelity evidence gates, contract inventory validation, and refreshed
  source-pair reporting for parity work.
- Durable operator run reports for scheduled runs.
- Browser extraction, Navivox continuous voice/profile-contact planning, gateway
  memory-pressure policy, and profile workspace/subprocess-home parity coverage.

### Changed
- Built-in `execute_code` and `terminal` tool output now compact noisy build
  and Go test output by default while preserving `full_output` escape hatches.
- Gormes command construction was split into long-term CLI modules, with the
  command refactor plan closed and release/action runtimes refreshed.
- Benchmark and progress mirrors were refreshed for the current development
  line.

### Fixed
- Azure Anthropic auth now sends bearer credentials correctly.
- Multiline tool-output compaction preserves build diagnostics instead of
  dropping failing-test evidence.
- Source-backed gateway installs restart against the intended source root and
  detect stale-code conditions more reliably.
- Telegram audio handling, Goncho transient message locks, contract inventory
  timestamp no-op behavior, and send-command output sanitization were hardened.

## [0.2.14] - 2026-05-17

Date alias: `v2026.5.17`.

> **Provider setup repairs, module-split progress storage, and gateway audio replies.**

### Added
- Gateway text-to-speech reply synthesis for audio-requested turns and `/tts on`
  sessions, including usable default Edge TTS selection when audio mode is
  enabled.
- Split progress backlog storage materialized by module while preserving the
  single logical backlog contract through `internal/progress.Load` and
  `SaveProgress`.

### Changed
- OpenAI Codex setup now routes through the OAuth-style account flow instead
  of incorrectly asking for an API key.
- OpenRouter setup/model selection now fetches the live model catalog instead
  of showing only a static subset.

### Fixed
- Discord gateway smoke coverage now accepts final sends that arrive after
  partial responses, matching observed manager behavior in CI.

## [0.2.13] - 2026-05-16

Date alias: `v2026.5.16`.

> **Native update flow hardening, module-split backlog infrastructure, and Navivox channel polish.**

### Added
- Native `gormes update` release-binary, release-assets, release-plan, service-coordination, and skill update-sync paths for Hermes-style operator upgrades without Python.
- Module-keyed progress backlog infrastructure behind the existing `internal/progress.Load` and `SaveProgress` contracts, including split-preserving writes, `progress emit`, completed-note compaction, and split-safe docs/fleet/skill readers.
- Navivox connect-and-talk refinements: profile contacts, gateway channel routing, Chrome e2e coverage, and structured tool-progress updates for native clients.
- Doctor and setup coverage for profiles, directory structure, skills hub, auth providers, security advisories, profile workspaces, and channel lists.

### Changed
- Release validation now mirrors the CI engineering-blog dependency install/test gate before tag publication.
- Progress-backed docs and landing mirrors were refreshed around the single-logical-backlog split plan; the destructive C5 flip remains blocked on explicit quiet-fleet operator approval.
- Setup wizard, model picker, provider defaults, and TUI error/readiness copy were tightened for operator-facing flows.

### Fixed
- Termux and gateway install/runtime paths, including notification bridging and release-publish behavior that avoids the v0.2.12 Termux noexec symlink regression.
- Doctor `--fix` wording, setup model-picker wiring, update release-plan handling, and CI path coverage for slim progress mirrors.

## [0.2.12] - 2026-05-15

Date alias: `v2026.5.15`.

> **Gateway platform resilience, native model-switch UX, and refreshed provider/parity evidence.**

### Added
- Gateway per-platform circuit breaker: a platform that fails the configurable threshold of consecutive retryable connects (`HERMES_/GORMES_GATEWAY_PAUSE_AFTER_FAILURES`, default 10) is paused with a reason and skipped by the reconnect watcher so one failing platform no longer keeps the gateway from running healthy channels.
- `/platform <list|pause|resume> [name]` operator slash command to surface and manually control failed/paused gateway platforms.
- Kernel in-session model-switch seam plus native TUI `/model` slash binding over the existing model picker.
- Bubble Tea provider picker in `gormes setup` (line-input fallback for piped/non-terminal stdin).
- Offline-doctor peak-RSS runtime benchmark recorded in `benchmarks.json` and surfaced on the landing page.
- PicoClaw-derived channel media/identity regression matrix and a session ledger read model.

### Changed
- Provider catalog consolidated into a dedicated `internal/modelcatalog` package.
- Setup and model provider taxonomy aligned across the CLI and picker surfaces.
- Progress, benchmark, and upstream Hermes parity evidence refreshed (xAI Grok OAuth, SimpleX Chat, subscription-proxy mirrors; `hermes-agent` reference advanced).

### Removed
- SSH stdio Navivox transport: `gormes navivox serve|pair|setup-host` CLI subcommands, the wire-protocol Go package (`internal/channels/navivox/{protocol,server,status}.go`), the Flutter `Dartssh2ByteTransport` + `SshNavivoxChannel` clients, and the `dartssh2` Dart dependency. The HTTP/WS gateway channel (`internal/channels/navivox/channel.go` and `flutter-navivox/app/lib/core/gateway/*`) is now the only supported Navivox transport. Operators using `[navivox]` config must run with `auth_mode = "static_token"` or `"pairing_token"` and connect over HTTP/WS.

## [0.2.11] - 2026-05-14

Date alias: `v2026.5.14`.

> **First-run CLI/TUI setup, safer channel onboarding, and refreshed Hermes parity evidence.**

### Added
- Shared first-run readiness planner used by root launch, setup, onboard, and doctor surfaces.
- Target-first `gormes setup --quick` flow for terminal, Telegram, WhatsApp, Discord, Slack, and Navivox.
- Minimal Telegram, Discord, and Slack channel setup prompts with secret-safe config writes.
- Planner-backed `gormes onboard` and `gormes doctor --target` text/JSON readiness reports.
- Fresh-install e2e coverage for root no-TTY guidance, quick setup non-interactive mode, onboard JSON, and doctor target JSON.
- Hermes parity refresh automation, source-pair evidence, and default template/guidance fixtures.
- Novita provider parity classification and local install CLI smoke coverage.

### Changed
- Fresh `gormes` launches now show setup guidance or route into setup instead of opening an unusable TUI.
- WhatsApp setup guidance now uses the plan-only `gormes whatsapp --plan` path unless the operator explicitly runs the live command.
- `setup --quick --target <channel>` skips channel setup when that channel is already configured.
- `gormes doctor --offline` now stays local-only for provider, gateway, CDP, and GitHub auth checks.
- Gateway background command guards and quick setup entry-mode isolation were tightened.
- Progress, benchmark, upstream Hermes mirror, and cross-project parity evidence were refreshed.

### Fixed
- Browser skip installer flags are accepted by the source-build installer path.
- Existing `.env` permissions are tightened without clobbering operator secrets.
- SecretRef-backed provider auth is counted correctly in terminal/onboard readiness without leaking resolved values.
- Discord and Slack numeric channel IDs remain string-safe in nested config paths.
- Kanban board pinning and fresh-install kanban environment isolation no longer leak across tests.

## [0.2.10] - 2026-05-14

Date alias: `v2026.5.14`.

> **Native Navivox gateway channel over private HTTP/WebSocket transport.**

### Added
- Native `navivox` gateway channel owned by `gormes gateway`, with HTTP status
  routes, WebSocket streaming, typed JSON messages, bearer-token auth, safe
  metadata logging, and no raw shell endpoint.
- Setup support for enabling the Navivox channel, choosing local/Tailscale/public
  exposure, bind host, port, auth mode, pairing token generation, and explicit
  firewall intent without silently changing firewall rules.
- Flutter Navivox gateway client, active fake/gateway channel switcher, setup
  screen connection fields, and chat streaming over the native gateway.
- Navivox gateway runbook covering trust boundaries, setup, Tailscale mode,
  firewall examples and rollback, endpoints, and smoke tests.

### Security
- Navivox is disabled by default and binds to loopback by default.
- Public exposure requires both `navivox.exposure_mode = "public"` and
  `navivox.public_confirmed = true`.
- `navivox.token` is routed to `GORMES_NAVIVOX_TOKEN` instead of committed TOML.

## [0.2.9] - 2026-05-13

Date alias: `v2026.5.13`.

> **Unified admin TUI, agent-per-thread controls, and hardened install/release evidence.**

### Added
- Unified Bubble Tea admin TUI with Setup, Chat, Agents, and Commands tabs.
- Admin command catalog over the live Cobra command tree, with safe inline
  execution for read-only commands such as `gormes kanban list`.
- Agent spawn/list/bind/unbind/inspect CLI for channel-native agent routing.
- Dynamic Goncho agent registry for runtime-spawned agent personas.
- Hermes-parity command stubs and profile lifecycle aliases for the wider CLI
  surface.

### Changed
- `gormes login --provider openai-codex` and admin command discovery now route
  through the Gormes command tree instead of falling back to Hermes-owned
  instructions.
- Docs and operator release skills now require the full development green gate,
  PR-to-main merge, tag, release workflow, artifact verification, and
  post-release development sync.
- Install and release tests now cover no-curl/no-wget fallback, hash mismatch,
  missing Go toolchain, `--branch`, `--local`, `--from-source`,
  `--uninstall --dry-run`, no-systemd skips, and Termux detection.

### Fixed
- Gateway and Navivox command parsing now rejects unknown subcommands and
  protects TTY-only setup paths in non-interactive contexts.
- Telegram client shutdown is guarded against double-close panics.
- Corrupt SQLite databases for Goncho memory, sessions, and memory.db can
  self-heal instead of blocking startup.

## [0.2.8] - 2026-05-11

Date alias: `v2026.5.11`.

> **Install URLs switched to GitHub for trust. Sandbox provider, DingTalk channel, interactive menus.**

### Added
- SandboxProvider interface with LocalSandboxProvider and VirtualPathResolver
- DingTalk StreamClient with SDK event channel and lifecycle
- Arrow-key navigation in setup menu and provider selection
- Migration options (Hermes/OpenClaw) in setup menu
- Guard composer for Tirith + allowlist + URL policy

### Changed
- All install URLs point to `raw.githubusercontent.com` instead of `gormes.ai`
- `gormes migrate hermes --yes` now auto-discovers source (same as `--dry-run`)
- Deploy workflow verifies landing page doesn't advertise curl-pipe-sh install

## [0.2.7] - 2026-05-11

Date alias: `v2026.5.11`.

> **Cross-channel formatting, Hermes config parity, cron in gateway mode.**
> Discord and Slack now get rich markdown formatting. Agent personalities,
> session auto-reset, STT TOML config, and enhanced display settings reach
> Hermes config parity. Cron jobs fire in gateway mode with Telegram delivery.

### Added

- **Cross-channel message formatting** (`Phase 9.A`): Shared `FormatFinalMarkdown`
  renders headings, lists, code blocks, bold, italic, and links for Telegram
  (MarkdownV2), Discord (Discord markdown), and Slack (mrkdwn).
- **Middleware chain framework** (`internal/core/agent/`): `Middleware` interface,
  `MiddlewareChain` with deterministic ordering, `RuntimeFeatures` with
  `FeatureFlag` toggle, 5 built-in middlewares, kernel integration.
- **Cron in gateway mode**: Scheduler now starts in `gormes gateway` with
  Telegram delivery sink. `gormes cron list/status/remove` CLI commands.
- **STT wiring**: `transcribe_audio` tool registered by default with local
  WASI whisper provider. No API key required.
- **Hermes config parity**: 12 built-in personalities, agent runtime settings,
  enhanced display config (show_reasoning, streaming, bell_on_complete, etc.),
  session auto-reset (inactivity/daily/both/none policies), STT TOML config.
- **Tool progress plain text fix**: No more MarkdownV2 escaping in URLs and paths.
- **Kanban notification delivery parity**: `ThrottledNotifySender` for rate-limited
  Telegram notifications.
- **Landing page redesign**: Updated copy, proof strip, and navigation.

### Changed

- Binary reduced from 42.2 MB to **40.1 MB** (stripped Go 1.26 build).
- install.sh `--build` flag with release-quality `-trimpath -ldflags="-s -w"`.
- Snowflake test → gormes-browser-harness migration.

## [0.2.6] - 2026-05-11

Date alias: `v2026.5.11`.

> **Declarative middleware chain framework + STT wiring with local whisper.**
> Phase 9 ships two owned architecture improvements from DeerFlow pattern
> analysis: an ordered, inspectable middleware chain with RuntimeFeatures
> toggle flags, and the first step toward a sandbox provider abstraction.
> Speech-to-text now works by default via WASI whisper (no API key required).

### Added

- **Middleware chain framework** (`internal/core/agent/`): `Middleware` interface
  with Before/After lifecycle hooks, `MiddlewareChain` with deterministic
  ordering and `Dump()` inspectability, `RuntimeFeatures` with
  `FeatureFlag` (Enabled/Disabled) and `CustomMiddleware` overrides.
  `AssembleFromFeatures()` factory builds ordered chains from declarative
  feature structs. Includes 5 built-in middlewares: thread_data, tool_error,
  loop_detector (wraps existing LoopDetector), memory, subagent.
  (`b5d2b7f1f`)
- **Kernel integration**: `AgentMiddleware` field on `kernel.Config` with
  Before/After hooks called in `runTurn()` turn lifecycle.
- **STT tool wiring** (`Phase 9.C`): `transcribe_audio` tool registered in
  default tool registry. `LocalSTTProvider` wraps WASI whisper runtime with
  auto-downloading tiny.en model (~77MB from HuggingFace). Cloud providers
  (OpenAI, Groq, Mistral, XAI) activate when API keys are present.
  (`3b44ec379`, `532d583ce`, `35eb1f769`)
- **Phase 9 planning**: Design & Security Hardening phase with builder-ready
  rows for middleware chain, sandbox provider, and STT wiring.
  (`2b1bbcf2b`)

### Developer

- `go test ./internal/tools -run TestLocalSTTProvider_Transcribe_JFKFixture`
  proves the full STT pipeline: model download → WASM runtime → WAV decode →
  transcription, asserting JFK speech excerpts.
- `go test ./internal/agent -count=1` covers middleware chain assembly,
  ordering, lifecycle, abort-on-error, feature toggling, and dump output.

## [0.2.5] - 2026-05-10

Date alias: `v2026.5.10` (second same-day patch over v0.2.4; shared
alias follows the v0.1.06 / v0.1.07 and v0.2.1 / v0.2.2 / v0.2.3 / v0.2.4
precedent for back-to-back same-day releases).

> **Critical install.sh data-loss fix + OpenRouter URL fix + OpenAI STT
> companion fix.** Operators who ran `install.sh --uninstall` from a
> sandbox in v0.2.4 or earlier could permanently delete their REAL
> `~/.gormes/` (provider keys in `.env`, Goncho `memory.db`, `config.toml`,
> binaries) — two independent bugs lined up to make this maximally
> destructive. Operators who copy-pasted OpenRouter's documented base URL
> `https://openrouter.ai/api/v1` got the cryptic
> `"Not Found: provider returned HTML error body"` error instead of a
> working chat completion. Both classes are fixed. Plus the OpenAI STT
> provider had the same `response_format=text` + JSON-decode mismatch the
> v0.2.4 Groq fix already addressed for one provider.

### Critical — install.sh `--uninstall` no longer destroys operators' real `~/.gormes`

- **Scope leak fixed.** `install.sh`'s `run_uninstall()` now exports
  `GORMES_HOME="$(managed_home_dir)"` before invoking the gormes
  `uninstall` subcommand. Without this, an operator running
  `GORMES_INSTALL_HOME=/tmp/sandbox install.sh --uninstall` saw the
  gormes process inherit the operator's default $HOME-derived
  `~/.gormes` path and delete the WRONG tree. Live regression
  2026-05-10 confirmed unrecoverable data loss of `.env`, `memory.db`,
  `config.toml`, and PATH symlinks during a bug-hunting sandbox
  uninstall test.

- **Permanent delete → recoverable trash by default.**
  `cmd/gormes/uninstall.go` now uses `gio trash` when available
  (freedesktop trash, recoverable from any file manager's trash UI)
  instead of `os.RemoveAll`. Operators who legitimately need permanent
  deletion (CI cleanup, container teardown, secure wipe) can opt in
  via `GORMES_UNINSTALL_FORCE_PURGE=1`. On hosts without `gio` the
  fallback is permanent delete with an explicit label that names the
  missing dependency (`"gio not available; install glib2-tools for
  recoverable trash"`).

- **Removal mode is surfaced.** Stdout now prints
  `removal mode: <label>` before the per-group report, and
  `--json` output includes `removal_mode: "..."` so fleet automation
  can verify which mode fired.

- Four new tests pin the contract:
  `TestPickArtifactMover_PrefersGioTrashWhenAvailable`,
  `TestPickArtifactMover_ForcePurgeOptsIntoPermanentDelete`,
  `TestPickArtifactMover_ForcePurgeAcceptsTrueAlias`,
  `TestPickArtifactMover_DefaultOnHostWithoutGio`.

### Fixed — OpenRouter (and other OpenAI-compatible) base URL with `/v1` no longer 404s

- **`internal/llm/http_client.go` `openAICompatibleURL`** now
  collapses a `/v1` prefix when both basePath and endpointPath carry
  it. Previously, `endpoint = "https://openrouter.ai/api/v1"` (the
  documented base URL) joined with `/v1/chat/completions` produced
  `https://openrouter.ai/api/v1/v1/chat/completions` (double /v1) and
  the upstream service returned a 404 HTML page. The cryptic
  `"Not Found: provider returned HTML error body"` Go error gave the
  operator no clue that the URL was double-prefixed.
- Both `endpoint = ".../api"` and `endpoint = ".../api/v1"` now
  resolve to the same correct URL across every OpenAI-compatible
  provider (OpenAI itself, OpenRouter, Together, Groq chat, DeepInfra).
- Eight-case `TestOpenAICompatibleURL_CollapsesDoubleV1Prefix` pins
  the contract, including non-collapsible cases (Anthropic
  `/v1/messages`, Azure `/responses`, middle-`/v1` proxy URLs).

### Fixed — OpenAI STT response parsing (companion to the v0.2.4 Groq fix)

- `TranscriptionOpenAIProvider.Transcribe` was sending
  `response_format=text` in the multipart form but attempting
  `json.NewDecoder(...).Decode(...)` on the response body. Same
  defect class the v0.2.4 Groq fix addressed for one provider; this
  was a copy-paste bug in the OpenAI implementation. Now reads the
  body as plain text matching the requested format. Existing OpenAI
  STT test updated to serve plain text (matches real OpenAI behavior
  under `response_format=text`) and asserts `response_format=text`
  is in the outgoing request.

### Added — Pure-Go WASI Whisper transcriber wired into Telegram STT

- `internal/wasi/whisper/transcriber.go` — pure-Go Whisper transcriber
  via wazero + whisper.cpp WASM, no CGO, no Python.
- `internal/wasi/whisper/audio/` — audio preprocessing package
  (16 kHz mono PCM resampling).
- Telegram channel STT resolver now tries the WASI Whisper path
  before HTTP fallbacks (Groq, OpenAI), so operators with the
  bundled wasm + model file get fully-local transcription without
  paying a cloud STT provider.
- WASI Whisper benchmark evidence recorded for tiny.en/base.en/small
  performance comparison.

### Improved — V4A patch tool resilience

- Block-anchor patch matching for hunks that don't match the literal
  source bytes.
- Fuzzy patch replace strategies (whitespace-insensitive, unicode
  normalization).
- Context-aware patch replacement when multiple matches exist.
- Failed v4a patch applies now roll back partial changes.
- Patch no-match errors surface recovery hints.

### Improved — Kanban orchestrator hardening

- `feat: add Kanban orchestrator board tools` — chat tools can pin a
  kanban board and operate on it implicitly.
- `feat: add kanban tail event follower` — operators can stream
  kanban worker events from the CLI.
- `fix: stabilize kanban tool list order`,
  `fix: harden kanban worker spawn resolution`,
  `fix: harden kanban corrupt timestamp reads`.

### Improved — Provider/runtime polish

- `feat(provider): add stream drop upstream diagnostics` — kernel
  provider-stream-drop frames now carry upstream diagnostic
  fingerprints for fleet triage.
- `fix: cover Hermes bundled platform plugins` — bundled-plugin
  loader regression closed.
- `fix: shape browser console CDP results` — browser tools now
  return CDP-shaped results consistently.
- `fix: guard curator prompt against transient failures` — curator
  background skill now retries transient failures instead of
  aborting the prompt build.
- `fix: bind release date alias to GitHub releases` — release.json
  date_alias is now correctly bound to the GitHub release URL.
- `feat: expand Hermes i18n locale parity` — additional locale
  files mirrored from upstream Hermes.

### CI

- `ci: smoke release binary metadata` — release workflow now smokes
  binary metadata (version/git_commit/build_date provenance) before
  publishing the GitHub release.
- `test(landing): avoid e2e port collisions` — landing e2e
  Playwright config honors `LANDING_E2E_PORT` env override (built on
  in v0.2.4, hardened here).

## [0.2.4] - 2026-05-10

Date alias: `v2026.5.10`.

> **Telegram voice transcription via Groq Whisper actually works.**
> The provider was sending `response_format=text` to Groq Whisper but
> attempting `json.NewDecoder(...).Decode(...)` on the response body —
> Groq honored the format and returned the raw transcript starting
> with whatever character the user said, the JSON decoder choked on
> the first non-`{` character, and Telegram voice messages came back
> as `"audio transcription provider failed"`. Reading the body as
> plain text matches the format we already requested.

### Fixed — Groq Whisper STT response parsing

- **`internal/tools/transcription_providers.go`**:
  `TranscriptionGroqProvider.Transcribe` now reads the response body as
  plain text (`io.ReadAll(resp.Body)` + `strings.TrimSpace`) instead of
  attempting to JSON-decode it, matching the `response_format=text`
  field already in the outgoing multipart form.
- **Regression test**: `TestTranscriptionGroqProviderTranscribe` server
  now returns plain text (matching real Groq behavior under
  `response_format=text`) and asserts the outgoing request carries
  `response_format=text`. The test fails with the old JSON-decode
  approach and passes with the read-as-text fix.

### Improved — Telegram STT diagnostics (built on 0.2.3 plumbing)

- **`telegramAudioErrorDiagnostic`** now distinguishes the seven Groq
  STT failure sites (`stt_groq_network_failure`,
  `stt_groq_file_open_failure`, `stt_groq_request_build_failure`,
  `stt_groq_parse_failure`, `stt_groq_copy_failure`,
  `stt_groq_writer_close_failure`, `stt_groq_form_failure`) where the
  previous taxonomy collapsed them all to `stt_groq_local_failure`.
  The newly granular `stt_groq_parse_failure` is what pinpointed the
  response-format/JSON-decode mismatch above in one round trip.
- **New `telegramAudioErrorRedactedDetail`** returns a 256-char-bounded,
  redacted substring of `err.Error()` suitable for the WARN log. It
  strips Telegram bot-token-shaped substrings and Telegram getFile
  direct URLs so log forwarders cannot leak credentials; provider-side
  errors (Groq HTTP body, dial errors, TLS messages) pass through.
- Both `bot.go` WARN call sites (voice + audio) now log
  `diagnostic=<token> detail=<redacted-truncated-err>` alongside the
  existing sanitized `err=` field, so operators can group by token or
  read the underlying error without a rebuild cycle.

## [0.2.3] - 2026-05-09

Date alias: `v2026.5.9` (third same-day patch; shared alias follows the
v0.1.06 / v0.1.07 and same-day v0.2.1 / v0.2.2 precedent for back-to-back
releases).

> **Methodology-first landing + multimodal vision + provider-fallback polish.**
> Landing copy at gormes.ai now leads with TrebuchetDynamics' autonomous-porting
> methodology and the ratified v1 differentiator (30 Hermes skills, 1 Go binary,
> 3 hard targets) instead of Hermes-compatibility, which becomes supporting
> evidence per `docs/content/building-gormes/strategy/success-plan.md`. Telegram
> photo attachments now reach vision-capable providers as `image_url` content
> parts; vision-rejected turns auto-retry text-only.

### Public site — methodology-first positioning

- **Landing hero rewritten.** Page title, headline, hero paragraphs, navigation,
  and secondary CTA lead with the autonomous-porting methodology and the
  ratified v1 differentiator (30 most-used Hermes skills · single Go binary ·
  Termux + Windows-without-Python + locked-down corp Linux). Hermes parity
  demoted from the lede to supporting evidence; methodology section renamed
  `THE METHODOLOGY / How the receipt is produced.`; methodology pillars
  reordered so "Reusable porting toolkit" precedes "Hermes is the parity
  oracle, not the contract".
- **Differentiator chip variant.** Proof-strip gains a `.proof-item-pop`
  yellow-emphasized chip variant for the three v1 differentiator items.
- **Deploy guard alignment.** `deploy-gormes-www.yml` content guards repointed
  to the new copy; old hero strings added to the negative-grep list so future
  drift back to parity-first copy fails the deploy.

### Multimodal vision passthrough

- **Telegram photo → image_url content parts.** Channel attachments with
  `Kind: "photo"` now materialize into `llm.MessageContentPart{Type:
  "image_url", ImageURL: "data:<mime>;base64,..."}` on the kernel turn
  message, so vision-capable providers (gpt-5.5 via openai-codex,
  Anthropic, OpenAI multipart) receive the image instead of just a text
  marker. Pure-Go path; zero Python dependency.
- **Admission gate exempts image bytes.** `validateTurnAdmission` no longer
  counts `image_url` data URI payloads against the text `MaxBytes` limit.
  Image-only turns (no caption text) pass the empty-input check. Image
  payload size is governed separately by the provider-side
  `image_shrink_retry` path.
- **Native image input mode wired.** `DecideImageInputMode` and the
  associated path hints (previously dead code) are now wired into the
  channel→kernel turn submission so the auto/native/text mode decision
  actually runs.
- **Vision-unsupported retry.** When a provider returns a vision-rejection
  phrase, the turn now retries text-only automatically instead of failing
  the user's request — Hermes parity for `agent/run_agent.py` retry
  behavior.

### Voice STT

- **Telegram voice STT HTTP fallback.** Channel resolver now wires the
  Groq HTTP transcription provider as the priority cloud fallback ahead
  of paid OpenAI when local `whisper`/`whisper-cli` binary is absent;
  configured by the `GROQ_API_KEY` environment variable.
- **Pure-Go STT exploration started.** Filed Phase 5.E exploration row
  selecting the WASI productionization path (wazero + whisper.cpp WASM)
  and spawned the first two builder rows: `wazero WASI smoke harness`
  and `whisper.cpp WASI module discovery fixture`. Preserves the
  static-binary + zero-CGO promise.

### Provider / auth / runtime polish

- `fix: preserve fallback credential aliases` — provider fallback now
  preserves credential-pool alias mapping when the primary provider
  routes through a fallback.
- `fix: preserve config comments on set` — `gormes config set` now
  round-trips TOML comments through the writer.
- `fix(provider): classify generic timeout messages` — provider router
  now correctly classifies generic timeout messages as transient/retry,
  not permanent.
- `fix: preserve proxy replay metadata` — proxy replay no longer drops
  metadata on round-trip.
- `fix: add shell lint evidence to file tools` — file-task tools now
  surface shell linter (node, npx, go, rustfmt) evidence per file in
  patch and write results.
- `feat: add python lint evidence to file tools` and
  `feat: add structured lint evidence to file tools` — same shape for
  Python and structured-evidence cases.
- `feat: bind FAL image generation queue API` — new image-generation
  provider binding.
- `feat: add OpenRouter Pareto request plugin` — new request-routing
  plugin for OpenRouter.
- `feat: add shell completion command` — bash/zsh/fish completion
  generation.
- `feat: silence Telegram placeholders by default` — Telegram channel
  no longer sends placeholder/typing-indicator messages by default;
  opt-in via `telegram.notifications`.
- `feat: surface curator rename summaries` — curator now reports rename
  summaries in operator output.

### Kanban

- `feat: pin kanban board for chat tools` — chat tools can now pin a
  kanban board so subsequent operations target it implicitly.
- `feat: add kanban notify delivery engine` — kanban notifications now
  deliver through a typed engine with structured retry semantics.
- `feat: add kanban worker log command` — operators can stream kanban
  worker logs from the CLI.

## [0.2.2] - 2026-05-09

Date alias: `v2026.5.9` (same-day patch over v0.2.1; shared alias
follows the v0.1.06 / v0.1.07 precedent for back-to-back same-day
releases).

### Fixed — `--json` conformance fence (parent without subcommand)

- **13 parent commands** that previously printed Help text on stdout
  when invoked with `--json` and no subcommand now emit a structured
  `{build, action: "subcommand_required", parent, available, error}`
  document instead. Affects `config`, `kanban`, `session`, `auth`,
  `mcp`, `memory`, `goncho`, `profile`, `curator`, `system`,
  `security`, `channels`, `agent`. Fleet automation can now discover
  every parent's subcommand surface programmatically without scraping
  Help text. The recursive
  `installParentUnknownSubcommandGuards` helper registers a hidden
  `--json` flag on each guarded parent; `auth` and `mcp` parents
  (which have their own RunE so the guard skips them) wire the same
  helper explicitly.

Three new subtests in the `TestFreshInstallE2E_InvalidInputJSONEmitsStructuredError`
battery pin the contract for representative cases (`config`,
`kanban`, `agent`) plus the explicit-RunE path (`auth`, `mcp`).

## [0.2.1] - 2026-05-09

Date alias: `v2026.5.9`.

> **Closes 13 v0.2.0 fresh-install probe issues.** A nuke + reinstall +
> probe sweep against v0.2.0 surfaced one critical, two cleanup-completeness,
> one release-distribution, and nine `--json` conformance-fence escapes.
> All thirteen are fixed and verified end-to-end against a dev binary
> overlay; the conformance fence now covers eight batteries with no
> escape paths in the surface that was probed.

### Critical
- **`gormes update` mutates the install's managed checkout, not cwd.**
  Previously, running `gormes update` from inside the gormes-agent dev tree
  walked up cwd to find a git checkout, switched its branch from
  `development` to `main`, and ran a web build there — mutating the
  user's source instead of the install. New `resolveManagedCheckoutDir()`
  mirrors install.sh's `managed_checkout_dir()`: `GORMES_INSTALL_DIR` →
  `$GORMES_INSTALL_HOME/gormes-agent` → `$HOME/.gormes/gormes-agent`. Never
  falls back to cwd.

### Uninstall lifecycle
- **Removes the install.sh-published PATH symlink.** `gormes uninstall --yes`
  used to nuke `~/.gormes/bin/gormes` but leave `~/.local/bin/gormes`
  dangling. New `published-binary` artifact group enumerates symlinks
  pointing into the gormes home (only — never touches a real binary at
  the same path).
- **`gormes-home` group surfaces the home-tree wildcard honestly.** The
  preview previously listed `<home>/` under "logs" because
  `config.CrashLogDir()` returns `GormesHome()`. Renamed split: "logs"
  enumerates explicit log files only; "gormes-home" lists the home
  wildcard so operators see the wholesale-removal scope unambiguously.
- **`install.sh --uninstall` applies by default.** Without flags it now
  runs `gormes uninstall --yes --dry-run=false`, matching `install.sh`'s
  default-to-apply UX. `install.sh --uninstall --dry-run` still previews
  (caller intent wins).

### Release distribution
- **`install.sh` and `install.ps1` are first-class GitHub release assets.**
  The natural URL pattern
  `https://github.com/.../releases/download/<tag>/install.sh` returned 404
  in v0.2.0 — the publish step uploaded only platform tarballs + checksums
  + SBOMs. Release notes now also document the canonical curl/irm install
  one-liners.

### `--json` conformance fence — 9 new structured-error paths
Five commands paired with `--json` previously emitted cobra errors to
stderr with empty stdout, breaking the wire-shape contract fleet
automation depends on. New shared helper `emitJSONInputError` ships
`{build, action, error}` documents on stdout, exits 1, with `action`
discriminating the failure kind:

  - `auth status --json` (missing required `<provider>` arg) → `missing_argument`
  - `logs --json` (no log file exists yet) → `no_logs`
  - `secrets audit --json` (missing required `--plan` flag) → `missing_flag`
  - `restore --json` (missing `--list`/`--latest`/`--path`) → `missing_argument`
  - `<parent> <typo> --json` (every parent: mcp, config, kanban, session,
    agent, root) → `unknown_subcommand`
  - `<parent> <typo-with-suggestion> --json` (cobra's "did you mean" path
    short-circuited Find() before the parent guard fired) → `unknown_subcommand`
  - `gateway xyz --json` (gateway parent has its own RunE, so cobra
    rejected `--json` as "unknown flag" before subcommand routing) →
    `unknown_subcommand`

A seventh fresh-install conformance battery
(`TestFreshInstallE2E_InvalidInputJSONEmitsStructuredError`) pins all
nine cases.

## [0.2.0] - 2026-05-08

Date alias: `v2026.5.8`.

> **Convention enforcement via fresh-install E2E batteries.** The autonomous
> engineering loop now ships its own conformance fence: a six-battery suite
> runs every probed `--json` surface from a freshly nuked
> `GORMES_HOME`/`XDG`/`HERMES_HOME`/`CODEX_HOME` and asserts the wire shape
> fleet automation depends on. Minor bump (0.1 → 0.2) marks the move from
> case-by-case bug fixing to a CI-guarded contract.

### Added — Fresh-install E2E conformance fence

- **Six-battery `cmd/gormes/fresh_install_e2e_test.go`** asserts the
  `--json` arc behaves uniformly across every probed surface, run from a
  hermetic `freshInstallE2EHome(t)` (temp `GORMES_HOME`, `XDG_DATA_HOME`,
  `XDG_CONFIG_HOME`, `HERMES_HOME`, `CODEX_HOME`; provider env zeroed):
  - `TestFreshInstallE2E_NoNullArrayFieldsInJSON` — 21 commands; empty
    arrays and maps emit `[]`/`{}`, never `null`. Fleet automation can
    `len(items)` without nil-checking.
  - `TestFreshInstallE2E_TypoSuggestionsAcrossParents` — 9 parents wire
    cobra `SuggestionsMinimumDistance`/`SuggestionsFor` so typo hints
    surface across the whole command tree, not just root.
  - `TestFreshInstallE2E_FreshInstallReadOnlyCommandsExitZero` — 8
    read-only commands return exit 0 from a never-initialized home (no
    "DB not found" failures masquerading as user errors).
  - `TestFreshInstallE2E_BuildProvenancePresentInJSON` — 18 commands
    prepend `{build: {version, git_commit}}` so any captured snapshot
    can be attributed to a specific binary.
  - `TestFreshInstallE2E_JSONIsParseable` — 20 commands; stdout under
    `--json` is always a single parseable JSON document.
  - `TestFreshInstallE2E_NotFoundJSONEmitsStructuredDocument` —
    `kanban show|complete|claim <missing> --json` now emit
    `{build, action: "not_found", id, error}` instead of empty stdout
    on cobra-rendered stderr. Same convention as
    `session delete --json` and the mcp login JSON path. Fleet
    automation can distinguish "task missing" from "command crashed."

### Added — Native Windows installer

- **`scripts/install.ps1`** ships native Windows install support
  alongside `install.sh` (Linux/macOS/WSL2). Source-backed managed
  install, no Python/Node/Docker required. README + install docs
  document the path; landing page advertises it as a separate
  post-release slice (held back to keep this release scoped to the
  conformance fence).

### Added — Structured `--json` surfaces

- **`gormes config get --json`** emits
  `{build, key, value, secret_redacted, set}`. Secret keys
  (`hermes.api_key`, `hermes.api_token`, etc.) emit a redacted
  placeholder instead of the raw value, even when the env-overlay path
  has populated the key. Test asserts the placeholder leak-protection
  contract end-to-end.

### Fixed — Fresh-install UX

- Legacy `~/.hermes` XDG migration runs cleanly when `XDG_DATA_HOME`
  has not yet been seeded by another command.
- Read commands no longer fail when the kanban DB does not exist
  (lazy-init under cover).
- Parent-command typo suggestions now surface across the entire
  command tree (cobra `SuggestionsMinimumDistance` set on every
  parent in `newRootCommand`).
- `t.Setenv` cleanup vs. dotenv overlay leak — config-set tests
  use explicit `os.LookupEnv`/`t.Cleanup` capture/restore so
  `internal/config/dotenv.go`'s `os.Setenv` overlays do not bleed
  into later tests in the suite.

## [0.1.07] - 2026-05-07

Date alias: `v2026.5.7` (same operating day as v0.1.06).

### Added — Install Path
- **`install.sh` defaults to release-binary fetch** (~7x faster fresh installs):
  resolves the latest GitHub release tag, downloads
  `gormes-<v>-<os>-<arch>.tar.gz` + `.sha256`, verifies, extracts, and
  publishes — all in ~2 seconds vs. ~2-5 min for the previous source-build
  default. `--from-source` / `GORMES_INSTALL_FROM_SOURCE=1` opt-out, plus
  graceful fallback on any binary-fetch failure (network, missing asset,
  hash mismatch). Plan emitter exposes the decision via
  `install_method: binary-fetch|source-build (<reason>)`.

### Added — Setup Wizard UX
- **Prettified setup wizard** — Hermes-style framed headers, screen
  clearing between sections, color/bold for active selection. New
  `internal/cli/wizard_ui.go` helpers (`ClearScreen`, `Bold`, `Dim`,
  `Cyan`, `BrightCyan`, `Yellow`, `Green`, `PrintHeader`,
  `PrintSectionDivider`) are TTY-gated: piped output and `NO_COLOR=1`
  strip all escapes; `GORMES_NO_CLEAR_SCREEN=1` keeps colors but
  preserves scrollback.
- 47-provider list now leads with framed `Choose a provider` header,
  yellow `*` for default, bright-cyan number + bold label for default,
  dim numbers + auth-type tag for the rest.

### Added — `gormes update` Hermes Parity (5 of 6 audit-batch rows)
- **Structured progress UX**: `⚕ Updating Gormes Agent...` banner +
  per-evidence outcome glyphs (`✓ ✗ ⚠ ℹ ◆`) + `✓ Update complete!` /
  `✗ Update failed` summary. Existing UpdateEvidenceKind strings stay in
  every line for parser compatibility.
- **Pre-update backup policy**: `--backup` / `--no-backup` flags +
  silent-default + `update_pre_backup_skipped` /
  `update_pre_backup_requested` evidence. Backup writer is a follow-up
  slice; this surface ships the policy decision.
- **Bundled-skill profile sync**: calls the existing
  `internal/skills.SyncBundledSkillsToProfiles` after pull, renders
  `default: +N new, K unchanged, M user-modified (kept)` counts.
  Best-effort — failures emit `update_skill_sync_failed` but never fail
  the update.
- **Web UI rebuild step**: runs `npm install` + `npm run build` in
  `<checkout>/web` when `package.json` exists. `--skip-web` flag for
  opt-out. 4 typed outcomes:
  `update_web_build_{completed,skipped,unavailable,failed}`. Soft-failure
  contract — never fails the update.
- **Config schema migration prompt**: checks on-disk schema version
  after web build. With `--yes` auto-applies via
  `internal/config.MigrateConfigFile`; without `--yes` emits advisory
  `⚠ update_config_migrate_needed` pointing the operator at
  `gormes config migrate`.

### Added — Install Isolation
- **`iso-shellrc-leak` closed**: `install.sh`'s
  `ensure_path_in_shell_config()` now early-returns when
  `sandbox_bin_dir_set()`. Plan: `edit_shell_rc_files: skipped|yes`.
- **`iso-systemd-hijack` closed (P0)**:
  `install.sh`'s `print_service_instructions()` early-returns when the
  sandbox bin dir is set, leaving the production
  `~/.config/systemd/user/gormes-gateway.service` and macOS launchd plist
  untouched. Plan: `install_system_service: skipped|yes`.

### Added — CI / Release Hygiene
- **Repaired red `Deploy gormes.ai` workflow** on main: post-build
  verification asserts updated to match the methodology-first landing
  rewrite from v0.1.06. New positive asserts lock in the pivot
  (`HOW IT IS BUILT`, `Validated rows shipped`); negative asserts catch
  regressions to the old hero text.

### Added — Loop Infrastructure
- **Loop $/iteration cost metric** via
  `scripts/codexu-gormes-builder-loop.sh --cost-report`. New
  `internal/loopcost` package parses opencode JSONL and computes 7-day
  + 30-day spend rollups.

### Changed — Hermes Submodule
- `hermes-agent` submodule bumped to upstream `main` (current sha
  pinned by parity sweep).

### Notes
- 6 of 6 `gormes update` parity rows were planned by the audit batch;
  5 ship in v0.1.07. The deferred row #4 (goncho profile sync) waits on
  `internal/goncho` gaining a profile-sync surface.

## [0.1.06] - 2026-05-07

Date alias: `v2026.5.7` (adopting the Hermes-style `vYYYY.M.D` taxonomy in
release notes and `release.json`; the canonical git tag remains `v0.1.06`
until the release workflow accepts date-based tags as a separate concern).

### Added
- Phase 8 (Reputation & Publication) in `progress.json` with seven subphases
  (8.A–8.G) and ten gormes-owned builder-ready rows: TD blog scaffold, social
  presence, README rewrite, landing-page positioning audit, engineering
  writeup #1, sharp v1.0 differentiator decision, single-binary release
  pipeline, agentic-porting-kit extraction, loop $/iteration cost telemetry,
  and built-with-Gormes page.
- `docs/content/building-gormes/strategy/success-plan.md` — 12-month strategy
  doc capturing the methodology-first North Star, quarterly roadmap, 30-day
  action sprint, reputation-metrics scoreboard, and risk register.
- Six builder-ready Hermes-parity rows from the v2026.4.30 + v2026.5.7
  upstream sweep: provider client lazy-init for cold-start, plugin
  `transform_llm_output` lifecycle hook, native `video_analyze` tool
  contract, Kanban worker heartbeat/reclaim/zombie detection, Google Chat
  shared-chassis adapter seam, and auth-state TOCTOU close + redaction
  default-on parity (P0 security row).
- Methodology section on the landing page (`#methodology`) with live
  loop-output metrics, four pillar cards, and a link into the architecture
  plan.

### Changed
- Repositioned `webpages/landing/` hero, kicker, sub-lines, secondary CTA,
  proof strip, and top nav from "Run AI agents from one Go binary" to
  "AI agent runtime in one Go binary" with the autonomous-porting loop as
  the trust play. Outcome-first per the landing-quality rubric; methodology
  is the differentiator.
- Refreshed `gormes-hermes-parity` skill workflow to refresh the
  `hermes-agent` git submodule to upstream `main` HEAD before classifying
  behavior atoms, with a fetch+ff-only fallback for legacy clones.
- README dropped the gitignore entry for `hermes-agent/` now that the
  upstream reference checkout is tracked as a submodule.

### Fixed
- `progress.json` schema slips in the v0.12.0 + v0.13.0 parity rows
  corrected: `provenance` now uses the struct shape
  `{origin_type: "upstream"|"gormes"}`, and `trust_class: "plugin"` was
  replaced with the valid `system` value.

### Infrastructure
- `hermes-agent` is now tracked as a git submodule pinned to
  `NousResearch/hermes-agent` `main` (sha `7e2af0c2e`), replacing the
  previous gitignored side-clone. Mirrors 7 missing upstream-hermes docs
  for mirrored-coverage parity.
- `scripts/codexu-gormes-builder-cron.sh` and the loop wrapper gained an
  `opencode/deepseek-v4-pro` backend alongside the codexu path, switchable
  via `GORMES_BUILDER_BACKEND`. The cron loop now runs on opencode.
- Phase 6.L code executor and dependency resolver rows marked complete with
  passing tests.

## [0.1.05] - 2026-05-06

### Added
- Navivox framed transport, OpenSSH and Termius key import parsing, and tool
  lifecycle frames for SSH-backed mobile sessions.
- Bundled-skill profile sync support plus `gormes skills sync` command seams
  for active and named profiles.
- Model catalog cache helpers, Kanban worker tools, and the codexu builder loop
  wrapper for recurring progress delivery.

### Changed
- Streamlined README install guidance and refreshed public release metadata for
  `v0.1.05`.
- Moved the public site source under `webpages/landing` only; the root
  `www.gormes.ai` path is no longer part of the repo surface.

### Fixed
- Hardened installer prompts, download-plan output, and PATH setup behavior.
- Prevented codexu builder loop lock file-descriptor leaks, including dirty
  worktree runs and CI stress coverage.
- Made the codexu loop test runner work in environments without a configured
  `codexu` binary.

## [0.1.04] - 2026-05-06

### Added
- Kanban worker process lifecycle binding so Gormes can spawn, track, and
  cleanly supervise worker processes.
- Telegram event-bus adapter and tool execution event publishing so channel
  integrations can observe runtime activity through the shared bus.
- Configurable `voice.record_key` handling plus voice-mode environment
  detection for native and remote TUI sessions.
- ACP bridge status reporting in `gormes doctor` and expanded curator, update,
  and plugin runtime surfaces.

### Changed
- Kept the builder cron and release lane tied to green CI evidence before
  publishing release work.
- Refreshed release metadata for the README and landing site to point at
  `v0.1.04`.

### Fixed
- Cron environment/state parity tracking and closed the related progress row
  with validation evidence.
- Doctor auth detection for GitHub CLI credentials.
- Telegram mention boundaries, email sender allowlist enforcement, gateway
  stale-code checks, and Discord forum test isolation.

## [0.1.03] - 2026-05-06

### Added
- Home Assistant tool handlers and progress evidence for the completed parity
  row.
- Discord interaction authorization helpers for user, role, channel, ignored
  channel, thread-parent, wildcard, and autocomplete visibility policies.
- Execute-code mode resolution, trajectory writer redaction boundaries,
  Telegram text batching, and gateway media-provider goal loop support.

### Changed
- Discord outbound sends now use safe `allowed_mentions` defaults, with
  operator opt-in environment knobs for everyone, role, user, and replied-user
  mention behavior.
- Public docs, landing progress mirrors, benchmarks, and parity roadmap slices
  were refreshed from current source evidence.

### Fixed
- Suppressed duplicate final TUI transcript output.
- Stabilized cron validation gates, setup entry-mode behavior, Slack mention
  gating, Telegram startup lifecycle, docs Astro builds, and non-ASCII
  credential sanitization.

## [0.1.02] - 2026-05-05

### Added
- Expanded Go-native Hermes parity across CLI, TUI, provider, gateway, tool,
  memory, cron, migration, and dashboard surfaces.
- Operator workflows for setup and onboarding, provider defaults, security audit
  SecretRef findings, source-backed installation, discovery probes, system
  events, ACP bridge mode, WhatsApp setup, kanban dispatch, and Goncho memory
  tools.
- Repo-local parity and delivery skills for Hermes/OpenClaw discovery,
  provider/browser/runtime/build/release/git/readme/landing workflows, plus
  skill validation and dependency tooling.

### Changed
- Relocated web surfaces under `webpages/` and refreshed landing, docs, README,
  progress, and benchmark outputs from current source evidence.
- Hardened release and installer flows, Windows builds, config isolation,
  gateway secrets, terminal settings, and branch safety guardrails.

### Fixed
- Provider and gateway failure handling now sanitizes provider errors,
  authenticates health checks, redacts runtime errors and media paths, and
  stabilizes provider failure admission.
- Telegram markdown, audio, voice, typing, live-turn, and browser-artifact
  behavior; TUI session DB lock fallback and Hermes chrome alignment; dashboard
  static assets; setup cancellation and noninteractive paths; and release SBOM
  upload behavior.

## [0.1.01] - 2026-05-04

### Added
- `gormes secrets` runtime controls for SecretRef plan apply, audit,
  configure, and atomic reload flows with redacted evidence.
- Repo-local `gormes-release` skill for the development-to-main release lane.

### Fixed
- First-run setup model picker cancellation now exits cleanly without leaking
  the internal cancellation sentinel.

## [0.1.0] - 2026-05-01

### Added
- Hermes-in-Go parity wave: 11 P0/P1 rows implemented
  - Agent prompt assembly (identity loader, memory guidance, session search, full prompt builder)
  - xAI Grok adapter with error classification taxonomy
  - Permission-hardened tool execution (trust class, shell blocklist, filesystem scoping, approval UX)
  - Loop detector with 5 loop types and configurable thresholds
  - Structured memory types with confidence scoring and conflict resolution
  - LM Studio provider adapter
  - Browser harness doctor
- Cross-project feature mapping (8 projects) + synthesis plan
- Progress.json now has zero incomplete P0/P1 rows
- Release automation with changelog, version bump, and artifact attestation
- Source-backed Unix and Windows installers with dry-run/local modes, install
  locking, publish rollback, managed Go checksum enforcement, JSONL install
  ledgers, and live gateway restart policy controls.
- `gormes gateway status --json` for stable installer and operator parsing.
- Safe bounded audit logging for tool arguments and errors.

### Changed
- Improved CI workflow with build isolation tests for kernel package
- Merged main branch updates into development, resolving 52 file conflicts
- README and public release metadata now describe a conservative 0.1.0
  source-first release path instead of overstating connector or TTS parity.

### Fixed
- apiserver SSE fields missing after merge (sseMu, sseClients)
- Kernel/goncho integration import cycle resolved
- Build isolation violations (kernel no longer transitively imports session/memory/sqlite3)
- Telegram message dedupe now falls back to `MsgID` when a channel surface does
  not populate `MessageID`.
- Tool preview and audit logging preserve the right edge of truncated URLs and
  command-like fields.

## [0.2.0-scout] - 2025-01-15

### Added
- Native TUI with Bubble Tea
- Offline doctor diagnostics (`gormes doctor --offline`)
- Provider-backed one-shot turns
- Telegram, Discord, and Slack gateway runtime
- Goncho in-binary SQLite memory
- Tool registry with approval callbacks
- Progress-driven architecture docs

## [0.1.0-scout] - 2024-11-01

### Added
- Initial static binary runtime
- Basic CLI structure
- Gateway event routing
- SQLite session store

[Unreleased]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.2.23...HEAD
[0.2.23]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.2.22...v0.2.23
[0.2.22]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.2.21...v0.2.22
[0.2.21]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.2.20...v0.2.21
[0.2.20]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.2.19...v0.2.20
[0.2.19]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.2.18...v0.2.19
[0.2.18]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.2.17...v0.2.18
[0.2.17]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.2.16...v0.2.17
[0.2.16]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.2.15...v0.2.16
[0.2.15]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.2.14...v0.2.15
[0.2.14]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.2.13...v0.2.14
[0.2.13]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.2.12...v0.2.13
[0.2.12]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.2.11...v0.2.12
[0.2.11]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.2.10...v0.2.11
[0.2.10]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.2.9...v0.2.10
[0.2.9]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.2.8...v0.2.9
[0.2.8]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.2.7...v0.2.8
[0.2.7]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.2.6...v0.2.7
[0.2.6]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.2.5...v0.2.6
[0.1.05]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.04...v0.1.05
[0.1.04]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.03...v0.1.04
[0.1.03]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.02...v0.1.03
[0.1.02]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.01...v0.1.02
[0.1.01]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.0...v0.1.01
[0.1.0]: https://github.com/TrebuchetDynamics/gormes-agent/releases/tag/v0.1.0
[0.2.0-scout]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.0-scout...v0.2.0-scout
[0.1.0-scout]: https://github.com/TrebuchetDynamics/gormes-agent/releases/tag/v0.1.0-scout
