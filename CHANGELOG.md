# Changelog

All notable changes to Gormes-Agent are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
inside the 0.x compatibility window.

## [Unreleased]

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

[Unreleased]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.05...HEAD
[0.1.05]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.04...v0.1.05
[0.1.04]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.03...v0.1.04
[0.1.03]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.02...v0.1.03
[0.1.02]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.01...v0.1.02
[0.1.01]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.0...v0.1.01
[0.1.0]: https://github.com/TrebuchetDynamics/gormes-agent/releases/tag/v0.1.0
[0.2.0-scout]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.0-scout...v0.2.0-scout
[0.1.0-scout]: https://github.com/TrebuchetDynamics/gormes-agent/releases/tag/v0.1.0-scout
