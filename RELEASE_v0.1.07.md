# Gormes Agent v0.1.07 (v2026.5.7)

**Release Date:** May 7, 2026
**Date Alias:** `v2026.5.7` (same operating day as v0.1.06)
**Since v0.1.06:** ~735 commits on `development` (autonomous loop + operator-driven install/update parity slices)

> **The Operator-First Release** — `curl https://gormes.ai/install.sh | bash` now completes in ~2 seconds via signed binary fetch instead of 2-5 minutes via Go toolchain download + clone + build. The setup wizard adopts Hermes-style framed headers with screen clearing and color, and `gormes update` ships 5 of 6 audit-batch parity rows: structured ⚕/→/✓ progress UX, pre-update backup policy, bundled-skill profile sync, web UI rebuild, and config schema migration prompt.

---

## ✨ Highlights

- **`install.sh` defaults to release-binary fetch** — Fresh `curl … | bash` installs go from ~2-5 minutes (Go toolchain download + git clone + go build) to ~2 seconds (signed tarball fetch + SHA-256 verify + extract). Source-build remains available via `--from-source` / `GORMES_INSTALL_FROM_SOURCE=1`, and it auto-fallbacks for non-default branches, unsupported platforms, or any binary-fetch failure (network, missing asset, hash mismatch). Plan emitter exposes the decision via `install_method: binary-fetch|source-build (<reason>)`.

- **Setup wizard prettified** — Hermes-style framed `╭──╮│ title │╰──╯` headers, ANSI screen clear between sections, color/bold for active selection, dim for inactive. The 47-provider menu now scans top-down with a yellow `*` for the default, bright-cyan number + bold label for that row, and dim numbers + auth-type tag for the rest. Fully TTY-gated: piped output and `NO_COLOR=1` strip every escape; `GORMES_NO_CLEAR_SCREEN=1` keeps colors but preserves scrollback.

- **`gormes update` Hermes parity — 5 of 6 audit-batch rows shipped** — Each row reuses the same structured-progress vocabulary so the transcript reads as a single coherent UI:
  - **Structured progress UX**: `⚕ Updating Gormes Agent...` banner + per-evidence outcome glyphs (`✓ ✗ ⚠ ℹ ◆`) + `✓ Update complete!` / `✗ Update failed` summary line. Existing `update_*` evidence kinds stay verbatim in every line for parser compatibility.
  - **Pre-update backup policy**: `--backup` / `--no-backup` flags wired through `ResolveBackupPolicy`. Silent default; explicit choices emit `update_pre_backup_skipped` / `update_pre_backup_requested` (with `◆` glyph mirroring Hermes' `◆ Creating pre-update backup...`).
  - **Bundled-skill profile sync**: calls `internal/skills.SyncBundledSkillsToProfiles` after pull, renders `default: +5 new, 12 unchanged, 1 user-modified (kept)` counts. Best-effort — failures emit `update_skill_sync_failed` but never fail the update.
  - **Web UI rebuild step**: runs `npm install --silent` + `npm run build` in `<checkout>/web` when `package.json` exists. `--skip-web` opt-out. 4 typed outcomes: `update_web_build_{completed,skipped,unavailable,failed}`. Soft-failure (matches Hermes' `_build_web_ui` contract).
  - **Config schema migration prompt**: checks on-disk version after web build. With `--yes` auto-applies via `MigrateConfigFile`; without `--yes` emits advisory `⚠ update_config_migrate_needed` pointing the operator at `gormes config migrate`.

- **Install isolation hardening** — Two more boundaries closed via the `gormes-install` skill's sandboxed test methodology:
  - **iso-shellrc-leak** (P2): `install.sh`'s `ensure_path_in_shell_config()` early-returns when `sandbox_bin_dir_set()` is true. Sandbox installs no longer leave dangling `export PATH=/tmp/...` lines in `~/.bashrc` / `~/.profile`.
  - **iso-systemd-hijack** (P0): `install.sh`'s `print_service_instructions()` early-returns when the sandbox bin dir is set, leaving the production `~/.config/systemd/user/gormes-gateway.service` byte-identical. Same fix protects macOS launchd plists.

  All three install-isolation surfaces (binary, shellrc, systemd) now share the same `sandbox_bin_dir_set` helper. Plan emitter exposes every decision: `update_active_path_command`, `edit_shell_rc_files`, `install_system_service`.

- **Repaired red `Deploy gormes.ai` workflow** — The post-build asserts on the v0.1.06 release commit had been failing because the workflow was checking pre-pivot hero text. Updated positive asserts to match the methodology-first landing (`AI agent runtime in one Go binary.`, `An autonomous engineering loop ports Hermes to Go, every day.`, `HOW IT IS BUILT`, `Validated rows shipped`) and moved stale strings to negative-assert block. Main is fully green again.

- **Loop $/iteration cost metric** — `scripts/codexu-gormes-builder-loop.sh --cost-report` parses opencode JSONL log files and prints 7-day + 30-day spend rollups. New `internal/loopcost` package handles parsing + windowing.

- **`hermes-agent` submodule bumped** to upstream `main` (current sha pinned by the latest parity sweep).

---

## 🛠 Install Path

### Default (binary fetch — fast)

```sh
curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh | bash
```

Plan output now leads with:

```
install plan
  branch: main
  install_method: binary-fetch (linux-amd64 from latest release (no Go toolchain or git clone needed))
  source: github releases (no git clone, no Go toolchain)
  ...
```

The binary-fetch path:

1. GET `https://api.github.com/repos/TrebuchetDynamics/gormes-agent/releases/latest` → parse `tag_name`.
2. Download `gormes-<version>-<os>-<arch>.tar.gz` + `.sha256` (curl/wget, retry 3).
3. Verify SHA-256 (sha256sum or shasum -a 256).
4. Extract → install at `<managed_bin_dir>/gormes`.
5. Falls back to source-build on ANY failure (network unreachable, asset missing, hash mismatch, missing tools).

### Opt-out (source-build)

```sh
curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh | bash -s -- --from-source
# or
GORMES_INSTALL_FROM_SOURCE=1 sh install.sh
```

Source-build is also auto-selected for `--branch <non-main>`, unsupported platforms (release matrix is linux/darwin/windows × amd64/arm64), and runtime fetch failures.

---

## 🔧 Setup Wizard

The setup wizard now matches the polish operators see in `hermes setup`:

```
╭──────────────────────────────────────╮
│ How would you like to set up Gormes? │
╰──────────────────────────────────────╯

  No existing Gormes configuration was found.

 → (●) Quick setup - provider, model, and messaging
   (○) Full setup - configure everything
```

Each section clears the screen and re-renders its framed header, so the operator sees one focused decision at a time instead of a stack of menus. Color helpers (`Bold`, `Cyan`, `BrightCyan`, `Yellow`, `Green`, `Dim`) are TTY-gated: non-TTY writers and `NO_COLOR=1` get plain text. `GORMES_NO_CLEAR_SCREEN=1` keeps colors but preserves scrollback.

The 47-provider list:

```
╭──────────────────────────────╮
│ Choose a provider            │
╰──────────────────────────────╯

  1) alibaba (api_key)
  2) alibaba-coding-plan (api_key)
* 3) anthropic (api_key)        ← yellow * + bright-cyan number + bold label
  4) arcee (api_key)
  ...
```

---

## 🔁 `gormes update` Structured Progress UX

Before:

```
update branch: main
update_check	checked update readiness for main
```

After:

```
⚕ Updating Gormes Agent...

update branch: main
ℹ update_check    checked update readiness for main

✓ Update complete!
```

A real successful update now reads top-down through the phases:

```
⚕ Updating Gormes Agent...

update branch: main
◆ update_pre_backup_requested    backup_forced; backup writer not yet implemented (planned next slice)
✓ update_autostash_created       stashed local changes
✓ update_autostash_restored      restored stash refs/stash@{0}
✓ update_skill_sync_completed    default: +5 new, 12 unchanged
✓ update_web_build_completed     web UI built in /home/op/.gormes/gormes-agent/web
⚠ update_config_migrate_needed   config schema v11 → v19 available; run `gormes config migrate` or rerun with --yes
✓ update_gateway_restarted       gateway service reported active after restart

✓ Update complete!
```

### New flags

| Flag | Behavior |
|---|---|
| `--backup` | Opt-in to a single-run pre-update backup (writer is a follow-up slice; emits `◆ update_pre_backup_requested`). |
| `--no-backup` | Force-skip pre-update backup; beats `--backup` and config opt-in. |
| `--skip-web` | Skip the web UI rebuild step after pulling source. |
| `--yes` (already existed) | Now also auto-applies config migrations via `MigrateConfigFile`. |

### New evidence kinds

13 new `UpdateEvidenceKind` constants surface the structured-progress phases:

`update_pre_backup_skipped`, `update_pre_backup_requested`,
`update_skill_sync_completed`, `update_skill_sync_failed`,
`update_web_build_completed`, `update_web_build_skipped`,
`update_web_build_unavailable`, `update_web_build_failed`,
`update_config_migrate_needed`, `update_config_migrate_completed`,
`update_config_migrate_failed`.

All map to glyphs via `updateGlyphAndColor`:

- `*_failed`, `*_error` → `✗` (bold)
- `*_unavailable`, `*_timeout`, `*_needed` → `⚠` (yellow)
- `*_skipped`, `update_check`, `*_log_mirrored`, `update_not_managed_checkout` → `ℹ` (dim)
- `*_requested` → `◆` (bright cyan, mirrors Hermes' `◆ Creating pre-update backup...`)
- default → `✓` (green)

### Owned divergence vs Hermes' update

- **Multi-profile fan-out**: only the `default` profile is synced today. Named-profile discovery is a follow-up slice.
- **"↑ updated" / "− removed from manifest" counts**: `SyncBundledSkillsToProfiles` does not yet distinguish those. Counts surface as Added / Unchanged / Conflicts only.
- **Goncho profile sync**: `internal/goncho` does not yet expose a profile-sync surface. Audit row #4 deferred until that prerequisite ships.
- **Interactive y/n config-migrate prompt**: `gormes update --yes` auto-applies; without `--yes` the lifecycle emits an advisory pointing operators at the existing `gormes config migrate` interactive command (which already handles API-key prompts).
- **Pre-update backup writer**: this release ships the policy/decision surface; the actual zip writer is a follow-up slice that will wire behind the existing `update_pre_backup_requested` trigger.

---

## 🛡 Install Isolation

Two more `sandbox_bin_dir_set` boundaries closed via the gormes-install skill's sandboxed test methodology. Combined with `iso-bin-hijack` from v0.1.06, install.sh now respects three protected production surfaces when `GORMES_BIN_DIR` or `GORMES_PREFIX` is set:

| Surface | Plan output | Risk class |
|---|---|---|
| Active PATH command (`~/.local/bin/gormes`) | `update_active_path_command: skipped|yes` | iso-bin-hijack (P1) |
| Shell rc files (`~/.bashrc`, `~/.profile`, `~/.zshrc`, fish config) | `edit_shell_rc_files: skipped|yes` | iso-shellrc-leak (P2) |
| User systemd unit + macOS launchd plist | `install_system_service: skipped|yes` | iso-systemd-hijack (P0) |

All three are now regression-tested via fast `--dry-run` plan inspection in `internal/installtest/` (15 tests across 3 test files). Real-install verification on Linux/amd64 confirmed production binary symlink, shell rc files, and systemd unit are all byte-identical pre and post a sandbox install pass.

---

## 🌐 CI / Release Hygiene

The v0.1.06 release commit (`1e9c0a026`) had a red `Deploy gormes.ai` workflow because its post-build verify step was asserting the pre-pivot hero text from before the methodology-first landing rewrite. The fix:

- Replaced stale positive asserts with current strings:
  - `"AI agent runtime in one Go binary."` (hero h1)
  - `"An autonomous engineering loop ports Hermes to Go, every day."` (methodology h2)
- Added 2 new positive asserts that lock in the 8.B pivot:
  - `"HOW IT IS BUILT"` (methodology kicker)
  - `"Validated rows shipped"` (methodology metrics block)
- Moved stale hero strings to the negative-assert block so a regression to the old hero gets caught.

Main has been fully green since the v0.1.07 PR merge.

---

## 🔁 Loop Infrastructure

`scripts/codexu-gormes-builder-loop.sh --cost-report` now prints:

```
7-day spend: $XX.XX | 30-day spend: $YY.YY | runs: N
```

Backed by the new `internal/loopcost` package which parses opencode JSONL output (one JSON object per line, `usage.cost` field) and computes per-window aggregations. Useful for tracking the autonomous builder's $/iteration as the methodology-first North Star evolves.

---

## 📚 Hermes Submodule

`hermes-agent/` was bumped to upstream `NousResearch/hermes-agent` `main` (sha pinned by the latest parity sweep). The `gormes-hermes-parity` skill refreshes the submodule to upstream `main` HEAD at the start of every parity sweep, so classifications run against current upstream truth.

---

## ⚖️ Versioning Taxonomy

This release continues the Hermes-style dual-naming convention introduced in v0.1.06:

- **Canonical git tag:** `v0.1.07` (semver — gates with `cmd/gormes/version.go` and the release workflow's tag-vs-source check).
- **Date alias:** `v2026.5.7` (Hermes-style `vYYYY.M.D`, surfaced in this release-notes filename header and the GitHub release body).

Same operating day as v0.1.06 — the date alias is shared. Future releases on different days will get unique date aliases.

---

## ✅ Validation Evidence

Local gates (run on the `development` branch before opening the merge PR):

```sh
go test ./... -count=1                                  # ok (full suite)
go run ./cmd/progress validate                          # validated 8 phases
git diff --check                                        # ok
(cd webpages/landing && npm run build)                  # 1 page built
```

Remote gates: GitHub Actions `Release Binaries` workflow (validate → build matrix on linux/amd64+arm64, darwin/amd64+arm64, windows/amd64+arm64 → publish with checksums, attestations, and SBOM).

---

**Full changelog:** [CHANGELOG.md#0.1.07](https://github.com/TrebuchetDynamics/gormes-agent/blob/main/CHANGELOG.md#0107---2026-05-07)
**Compare:** [v0.1.06...v0.1.07](https://github.com/TrebuchetDynamics/gormes-agent/compare/v0.1.06...v0.1.07)
**Strategy doc:** [success-plan.md](https://github.com/TrebuchetDynamics/gormes-agent/blob/main/webpages/docs/content/building-gormes/strategy/success-plan.md)
