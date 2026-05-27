---
title: "Gormes Home Contract"
description: "Who reads, writes, and manages every file and directory inside `.gormes` (`$GORMES_HOME`). Source-backed ownership map and governing rules for operators, agents, and gateway code."
---

# Gormes Home Contract

Every file and directory inside `$GORMES_HOME` (default `~/.gormes`) is owned
by a specific subsystem. This document enumerates every runtime entry with its
owner, governing rules, and lifecycle.

## Governing root rules

| Rule | Source |
|---|---|
| **Runtime home** is `GORMES_HOME` if set, else `~/.gormes`. | `internal/config/config.go` |
| **Install home** is `GORMES_INSTALL_HOME` if set, else `~/.gormes`; may differ from runtime home. | `install.sh`, `cmd/gormes/update.go` |
| **Config load order**: built-in defaults → `config.toml` / `config.yaml` fallback → `.env` → environment vars → CLI flags. Shell env beats `.env`. | `internal/config/config.go` |
| **Secrets**: raw API keys, bot tokens, OAuth refresh tokens never inline in `config.toml`. Use `.env` (mode `0600`) or `auth.json`. | `internal/config/writer.go` |
| **Config v2**: one canonical `config.toml`. Profile directories hold runtime data (sessions, memory, logs) but not per-profile config files. | `docs/building-gormes/architecture_plan/profile-config-v2.md` |
| **Profile startup**: base home owns `active_profile` + `profiles/`. `--profile` or sticky `active_profile` selects the profile, then `GORMES_HOME` is set to the profile root. Profile names must match `[a-z0-9][a-z0-9_-]{0,63}` and not collide with reserved subcommands. | `cmd/gormes/main.go`, `internal/cli/profile_name.go` |
| Profile startup: base home owns `active_profile` + `profiles/`. `--profile` or sticky `active_profile` selects the profile, then `GORMES_HOME` is set to the profile root. Profile names must match `[a-z0-9][a-z0-9_-]{0,63}` and not collide with reserved subcommands. | `cmd/gormes/main.go`, `internal/cli/profile_name.go` |
| **Config sections**: closed set in `allowedTOMLSections`; unknown sections rejected on write. | `internal/config/writer.go` |
| **Runtime state** lives under `runtime/` (`gateway_state.json`, `gateway.pid`, `gateway-locks/`, `gateway.log`). | `internal/config/config.go` |
| **Lifecycle artifacts** live under `lifecycle/` (`update.log`, `install.log.jsonl`, `backups/`). | `cmd/gormes/update.go`, `install.sh` |
| **Transient pairing QR** lives under `cache/navivox/` (not root). | `cmd/gormes/navivox_pair.go`, `cmd/gormes/setup_navivox.go` |

## Top-level file/directory ownership

### `.env`
| Attribute | Value |
|---|---|
| **Readers** | `internal/config/dotenv.go` on `config.Load()`; tools look up env vars |
| **Writers** | `internal/config/writer.go`/`WriteEnvValue`; `gormes config set —secret` |
| **Permissions** | dir parent `0700`, file `0600` |
| **Rules** | Shell env beats `.env`. Malformed lines silently skipped. Secrets only. `#`-comment lines, blank lines skipped. Double-quoted values with escape expansion; single-quoted literal. |

### `config.toml`
| Attribute | Value |
|---|---|
| **Readers** | `internal/config/config.go`/`Load`/`loadFile`; `gormes config check/validate` |
| **Writers** | `internal/config/writer.go`/`WriteTOMLValue`; `gormes config set`; migration tools |
| **Permissions** | dir parent `0700`, file `0600` |
| **Rules** | TOML preferred. YAML fallback for Hermes migrants (config.yaml). Atomic temp+rename writes never replace with partial file. Closed section set. When `config_version = 2`, `[profiles.<id>]` schema applies. |

### `auth.json`
| Attribute | Value |
|---|---|
| **Readers** | credential pool (`internal/config/credential_pool.go`), Codex OAuth |
| **Writers** | credential pool save, auth import/export/register; `gormes auth` commands |
| **Permissions** | dir parent `0700`, all writes `0600` |
| **Rules** | Atomic `.auth-*.json` → `Rename`. Missing = empty map. All credential evidence redacted in status output. |

### `active_profile`
| Attribute | Value |
|---|---|
| **Readers** | `internal/cli/active_profile.go` on startup |
| **Writers** | `gormes profile use/create` |
| **Permissions** | Written via `active_profile.tmp` + `Rename`; `0600` |
| **Rules** | Missing = default profile. Content trimmed of whitespace. Valid profile name checked before write. `gormes profile`/`gormes config` subcommands skip sticky read. |

### `bin/gormes`
| Attribute | Value |
|---|---|
| **Readers** | shell PATH, `gormes update` |
| **Writers** | `install.sh`, `gormes update` built binary |
| **Permissions** | Executable (typically `0755` after go build) |
| **Rules** | `bin/gormes.build-tag` stores the source commit; rebuild triggered on changed/dirty source. Published symlink at `~/.local/bin/gormes` points here on default non-root. |

### `bin/gormes.build-tag`
| Attribute | Value |
|---|---|
| **Readers** | `install.sh`, `gormes update` |
| **Writers** | `install.sh`, `gormes update` |
| **Rules** | Stores git short HEAD hash. If current source commit matches, skips rebuild. If dirty or changed, rebuilds. |

### `channel_directory_sources.json`
| Attribute | Value |
|---|---|
| **Readers** | `internal/gateway/channel_directory_source.go` |
| **Writers** | Gateway Manager on session sources |
| **Permissions** | `0600`, atomic temp + rename |
| **Rules** | Separate ledger from `channel_directory.json`. Missing = silent (no sources yet). Invalids produce degraded evidence. Remembered inbound sources for future directory refresh. |

### `pairing.json`
| Attribute | Value |
|---|---|
| **Readers** | `internal/gateway/pairing_store.go`; gateway status |
| **Writers** | Gateway pairing flow (token request/approval) |
| **Permissions** | `0600`, atomic temp + rename |
| **Rules** | 1h code TTL, 10min rate limit, max 3 pending codes/platform, lockout after 5 approval failures. |

### `runtime/gateway_state.json`
| Attribute | Value |
|---|
| **Readers** | `internal/gateway/status.go`/`RuntimeStatusStore`; `gormes gateway status/stop` |
| **Writers** | Gateway lifecycle (start, drain, stop, config reload) |
| **Permissions** | `0600`, atomic `.gateway_state-*.tmp` + Rename |
| **Rules** | Runtime read model — PID, start time, generation, gateway state, platform states, active agents, token locks, drain evidence, restart markers. Surrendered when file missing. Validates PID against `runtime/gateway.pid` and `/proc` for stale/PID-reused/stopped detection. |

### `runtime/gateway.pid`
| Attribute | Value |
|---|---|
| **Readers** | `internal/gateway/status.go` process validation |
| **Writers** | Gateway runtime status store on status writes |
| **Rules** | Slim `RuntimeStatus` record (PID, start time, generation, command) alongside main state file. Written atomically every time state is written. Read for cross‑validation: if state PID ≠ pid record PID (or mismatched start time/gen) the status is stale. |

### `runtime/gateway-locks/*.lock`
| Attribute | Value |
|---|---|
| **Readers** | `internal/gateway/token_lock.go`/`TokenLockStore` |
| **Writers** | Gateway token lock acquire/release |
| **Permissions** | `0600` |
| **Rules** | One lock per platform+credential hash. Stores non‑reversible hash, PID, start time, command. Acquire checks PID/start‑time; stale lock cleared only when owning process is proven gone/reused/stopped. Platform name sanitised to safe filename characters. |

### `runtime/gateway.log`
| Attribute | Value |
|---|---|
| **Readers** | `gormes logs` (gateway endpoint, fallback to file read) |
| **Writers** | Detached gateway restart, default configured gateway process |
| **Rules** | Append target for detached gateway stdout/stderr. Created when `gormes gateway` is started via restart. |

### `lifecycle/install.log.jsonl`
| Attribute | Value |
|---|---|
| **Readers** | Operator/human; `gormes update` appends |
| **Writers** | `install.sh`, `gormes update` |
| **Permissions** | Append-only, `0600` |
| **Rules** | JSONL install/update/release ledger. `update --check`/`dry-run` should not write. |

### `lifecycle/update.log`
| Attribute | Value |
|---|---|
| **Readers** | Operator/human |
| **Writers** | `gormes update` output mirror |
| **Permissions** | Append, `0600` |
| **Rules** | Mirrors update output for SIGHUP protection. `update --check`/`dry-run` should not write. |

### `secrets-runtime.json`
| Attribute | Value |
|---|---|
| **Readers** | `gormes secrets` commands; gateway startup |
| **Writers** | `gormes secrets apply` |
| **Permissions** | `0600`, atomic temp + rename |
| **Rules** | Redacted snapshot of resolved credentials from secret providers. Generation-tracked. |

### `mcp_oauth.json`
| Attribute | Value |
|---|---|
| **Readers** | MCP OAuth handlers |
| **Writers** | MCP OAuth flow |
| **Rules** | Tracked by `gormes uninstall` as explicit group. |

### `slack-manifest.json`
| Attribute | Value |
|---|---|
| **Readers** | `gormes slack` command |
| **Writers** | `gormes slack manifest generate` |
| **Rules** | Written to `$GORMES_HOME` only when a relative path target is used. |

## Database files

### `sessions.db`
| Attribute | Value |
|---|---|
| **Readers** | `internal/session` via bbolt; gateway, telegram, TUI, cron, send, chat, discover |
| **Writers** | Session map, session metadata, cron jobs bucket |
| **Permissions** | dir parent `0700`, file `0600` |
| **Rules** | bbolt; 100ms lock timeout → `ErrDBLocked`; corrupt DB detected by `ErrDBCorrupt` → quarantined as `.corrupt-<timestamp>` and recreated. Single bbolt file shared by session map + cron jobs (separate buckets). |

### `memory.db`
| Attribute | Value |
|---|---|
| **Readers** | `internal/memory` via SQLite; gateway, telegram, Goncho, cron runs, transcript export |
| **Writers** | Background worker goroutine; fire-and-forget command queue |
| **Permissions** | dir parent `0700`, file -- (default sqlite) |
| **Rules** | SQLite WAL mode, synchronous NORMAL, busy timeout 2000ms, single writer connection. Corrupt detected → quarantined as `.corrupt-<timestamp>`, recreated. Schema includes goncho turns, entities, relationships, cron runs. Queue‑full = drop + log (never block kernel). |

### `memory.db-wal`, `memory.db-shm`, `memory.db-journal`
| Attribute | Value |
|---|---|
| **Readers** | SQLite engine, backup writer |
| **Writers** | SQLite engine |
| **Rules** | Transient SQLite sidecars. Excluded from backup (`IsExcludedFromBackup`). Relocated alongside `memory.db` during corruption quarantine. |

### `kanban.db`
| Attribute | Value |
|---|---|
| **Readers** | Kanban board runtime; kanban tools/commands |
| **Writers** | Kanban board runtime; kanban commands |
| **Rules** | Optional. `GORMES_KANBAN_DB` or `GORMES_KANBAN_HOME` overrides location. |

## Directory trees

### `sessions/`
| Entry | Owner | Rules |
|---|---|---|
| `index.yaml` | `internal/session/index_mirror.go` | Read‑only mirror of `sessions.db` for operator audit. Updated periodically on a timer; content‑hash skip avoids redundant writes. |
| `exports/` | `/save` slash handler, `gormes session export` | Transcript `ExportMarkdown` output. Created on first export. Collision-safe with incrementing suffix. |

### `profiles/<name>/`
| Entry | Owner | Rules |
|---|---|---|
| `runtime/gateway_state.json`, `runtime/gateway.pid` | Fleet supervisor, profile gateway workers | Per-profile runtime status read from here. |
| `sessions/`, `sessions/index.yaml` | Session subsystem (per-profile) | Profile-scoped index mirror when active profile is a named profile. |
| (other runtime state) | Profile-specific data in runtime dir | Homogeneous `baseHome/profiles/<name>` layout; old "default"/"main" falls back to base home until materialised as a profile dir. |

### `memory/`
| Entry | Owner | Rules |
|---|---|---|
| `USER.md` | Memory tool (source of truth); optional mirror (`USER.md` overwritten) | Durable user facts. Memory tool writes with `§` delimiter. If mirror enabled (`mirror_enabled`), SQLite is source of truth and this file may be overwritten. |
| `MEMORY.md` | Memory tool | Assistant/environment notes. Same tool contract. |
| `*.lock` | Memory tool | File lock for Hermes-compatible memory mutation serialisation. |
| `GONCHO_MEMORY.md` | Goncho v1 | Legacy markdown-based Goncho store. Read-only for migration. |

### `skills/`
| Entry | Owner | Rules |
|---|---|---|
| `active/<name>/SKILL.md` | Skill runtime / curator | Validated frontmatter; injected into live context on selection. |
| `candidates/<id>/` | Subagent delegation, curator | Candidate skills drafted, then promoted to `active/`; runtime never reads candidates. |
| `.usage.json` | Skill curator | Telemetry sidecar (use count, pin state, created-by). Never inside authored SKILL.md. |
| `usage.jsonl` | Skill usage logger | Append-only usage records with skill name and timestamp. |
| `.hub/` | Skills hub (optional) | Remote skill discovery and sync. |

### `tools/`
| Entry | Owner | Rules |
|---|---|---|
| `audit.jsonl` | Audit JSONL writer | Append-only; one JSON record per tool execution. Redacts secrets, home paths, bounds args/error lengths. |

### `subagents/`
| Entry | Owner | Rules |
|---|---|---|
| `runs.jsonl` | Delegation run logger | Append-only; `id`, `parent_id`, `goal`, `status`, `exit_reason`, `duration_ms`, `error`. |

### `hooks/`
| Entry | Owner | Rules |
|---|---|---|
| `*` | Gateway hook loader | `HOOK.yaml` hook scripts loaded by gateway on config reload. |

### `cache/`
| Entry | Owner | Rules |
|---|---|---|
| `audio/` (dir) | TTS tool | Audio output cache. Configurable `OutputDir`. Validates extensions before write. |
| `whisper/` (dir) | Local STT tool, Whisper model cache | Whisper model artifact cache. Override via `GORMES_STT_CACHE_DIR`. Downloads and verifies model SHA-256/size before caching. |

### `prompts/`
| Entry | Owner | Rules |
|---|---|---|
| `*.md` | Prompt template loader | Discovered and loaded as prompt templates for TUI. Not the canonical source files. |

### `plugins/`
| Entry | Owner | Rules |
|---|---|---|
| `*` | Plugin loader | User-installed plugins under `$GORMES_HOME/plugins/`. |

### `lifecycle/backups/`
| Entry | Owner | Rules |
|---|---|---|
| `pre-update-*.zip` | Update backup writer | Created on `--backup` or `[updates] pre_update_backup=true`. Excludes `backups/`, `checkpoints/`, and SQLite sidecars. Pruned to `backup_keep` (default 5). |

### `snapshots/`
| Entry | Owner | Rules |
|---|---|---|
| `<date>-pre-update/` | Release binary update | Pre-update binary snapshot (managed+published copies) for rollback. Written atomically before swap. |

### `system/`
| Entry | Owner | Rules |
|---|---|---|
| `state.json` | System events manager | Component presence/event registry. |

### `chrome-debug/`
| Entry | Owner | Rules |
|---|---|---|
| `*` | Browser debug/doctor | Chrome user data dir for remote debugging. |

## Directory patterns

### `profiles/*/gateway_state.json` + `gateway.pid`

Each profile that runs a separate gateway process writes its own runtime
status under `profiles/<id>/runtime/gateway_state.json` and `profiles/<id>/runtime/gateway.pid`.
The fleet supervisor reads from these paths when reporting status for all
enabled profiles. The `gormes update` restart policy enumerates every
enabled profile by reading `config.toml` `[profiles]`, resolves the profile
home root, and either delegates to a systemd per-profile unit or stops/waits
for the old process before starting a new one.

### `profiles/<id>/sessions/`

Named profiles maintain their own session persistence under the profile root.
The session index mirror writes to `profiles/<id>/sessions/index.yaml` for
operator audit. The default/main profile until materialised as a directory uses
the base home root directly.

## Lifecycle entries

### `backups/pre-update-*.zip`
| Owner | Writer trigger | Reader |
|---|---|---|
| `gormes update --backup` or `[updates] pre_update_backup=true` | Pre‑update lifecycle step | `gormes restore --path <zip> --yes` |

### `.restart_takeover.json`
| Owner | Writer trigger | Reader |
|---|---|---|
| `internal/gateway/restart.go` | Service‑managed gateway restart exit | New gateway process on startup; suppresses duplicate inbound platform updates. 5min TTL. |

### `.gateway-planned-stop.json`
| Owner | Writer trigger | Reader |
|---|---|---|
| `internal/gateway/status.go`/`PlannedStopStore` | `gormes gateway stop` writes marker before signal | Gateway SIGTERM handler reads and consumes marker; distinguishes planned stops from crashes. 1min TTL. |

## Workspace (default context workspace — `workspace/`)

| Entry | Owner | Rules |
|---|---|---|
| `SOUL.md` | Agent template seeder, live prompt loader | Persona; gateway/agent reset seed from default template. Loaded into live turn prompt with threat scan + truncation. |
| `AGENTS.md` | Agent template seeder | Workspace contract; first-match-wins project context precedence: `.hermes.md` > `AGENTS.md` > `CLAUDE.md` > `.cursorrules`. |
| `IDENTITY.md` | Agent template seeder | Stable identity/workspace facts; loaded as operational context. |
| `TOOLS.md` | Agent template seeder | Tool choices; loaded as operational context. |
| `memory/USER.md` | Agent template seeder | Durable user profile facts. |
| `memory/MEMORY.md` | Agent template seeder | Durable agent notes. |

**Rules**: Templates are seeded by gateway/agent reset. Live prompt routes
(`BuildContextFilesPrompt`): project context first (CWD walk for Hermes/AGENTS
files), then operational context (`IDENTITY.md`, `TOOLS.md`), then `SOUL.md`.
All context files scanned for 13 threat patterns + invisible Unicode. File
content truncated to `maxChars` (default 20000) using head+tail split.

## Uninstall groups

`gormes uninstall --dry-run` (default) enumerates these groups. `--yes` removes.
`--keep-config` skips `config` group; `--keep-credentials` skips `credentials`.
Default move to `gio trash` (recoverable) unless `GORMES_UNINSTALL_FORCE_PURGE=1`.

| Group | Paths |
|---|---|
| `config` | `config.toml`, `config.yaml`, `.env` |
| `credentials` | `auth.json` |
| `sessions` | `sessions.db`, `sessions/index.yaml` |
| `gateway-state` | `runtime/gateway_state.json`, `runtime/gateway-locks/`, `runtime/gateway.pid`, `runtime/gateway.log`, `channel_directory_sources.json` |
| `memory` | `memory.db`, `memory/` |
| `logs` | `gormes.log` |
| `cron` | `CRON.md` |
| `mcp-oauth` | `mcp_oauth.json` |
| `legacy-xdg` | legacy `$XDG_DATA_HOME/gormes/` |
| `published-binary` | published symlinks pointing into Gormes home (real binaries untouched) |
| `gormes-home` | entire `$GORMES_HOME` (catch-all) |
