# Changelog

All notable changes to Gormes-Agent are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
inside the 0.x compatibility window.

## [Unreleased]

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

- **`internal/hermes/http_client.go` `openAICompatibleURL`** now
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
  `Kind: "photo"` now materialize into `hermes.MessageContentPart{Type:
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

[Unreleased]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.05...HEAD
[0.1.05]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.04...v0.1.05
[0.1.04]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.03...v0.1.04
[0.1.03]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.02...v0.1.03
[0.1.02]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.01...v0.1.02
[0.1.01]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.0...v0.1.01
[0.1.0]: https://github.com/TrebuchetDynamics/gormes-agent/releases/tag/v0.1.0
[0.2.0-scout]: https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.0-scout...v0.2.0-scout
[0.1.0-scout]: https://github.com/TrebuchetDynamics/gormes-agent/releases/tag/v0.1.0-scout
