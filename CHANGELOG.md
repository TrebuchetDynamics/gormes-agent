# Changelog

All notable changes to Gormes-Agent are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
inside the 0.x compatibility window.

## [Unreleased]

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

[Unreleased]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.01...HEAD
[0.1.01]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.0...v0.1.01
[0.1.0]: https://github.com/TrebuchetDynamics/gormes-agent/releases/tag/v0.1.0
[0.2.0-scout]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.0-scout...v0.2.0-scout
[0.1.0-scout]: https://github.com/TrebuchetDynamics/gormes-agent/releases/tag/v0.1.0-scout
