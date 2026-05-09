# Changelog

All notable changes to Gormes-Agent are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
inside the 0.x compatibility window.

## [Unreleased]

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

[Unreleased]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.05...HEAD
[0.1.05]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.04...v0.1.05
[0.1.04]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.03...v0.1.04
[0.1.03]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.02...v0.1.03
[0.1.02]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.01...v0.1.02
[0.1.01]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.0...v0.1.01
[0.1.0]: https://github.com/TrebuchetDynamics/gormes-agent/releases/tag/v0.1.0
[0.2.0-scout]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.0-scout...v0.2.0-scout
[0.1.0-scout]: https://github.com/TrebuchetDynamics/gormes-agent/releases/tag/v0.1.0-scout
