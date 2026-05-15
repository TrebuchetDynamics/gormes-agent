---
title: "Paths & logs"
description: "Where Gormes reads and writes on disk, with every path verified against the binary."
weight: 35
aliases:
  - /reference/paths-and-logs/
---

# Paths & logs

Ask the binary for the active paths whenever possible:

```bash
gormes config path          # absolute path to config.toml
gormes config env-path      # absolute path to .env
```

Every default below resolves under `$GORMES_HOME`. The default
`$GORMES_HOME` is `~/.gormes`. Overriding `$GORMES_HOME` moves all of these
paths in lockstep.

## Canonical layout

| Path | Purpose | How it resolves |
|---|---|---|
| `$GORMES_HOME/config.toml` | Native TOML config. | `config.ConfigPath()`. The YAML fallback at `$GORMES_HOME/config.yaml` is only read when `config.toml` is missing. |
| `$GORMES_HOME/.env` | Dotenv secrets. Loaded before `config.toml`; shell env wins on conflicts. | `config.EnvPath()`. Only this single dotenv file is loaded automatically. |
| `$GORMES_HOME/auth.json` | Hermes-compatible credential pool managed by `gormes auth`. | Lives directly under `$GORMES_HOME`. |
| `$GORMES_HOME/active_profile` | Sticky active profile marker from `gormes profile use`. | Lives directly under `$GORMES_HOME`. |
| `$GORMES_HOME/sessions.db` | bbolt session map for resume. | `config.SessionDBPath()`. |
| `$GORMES_HOME/sessions/index.yaml` | Read-only YAML mirror of the session map. | `config.SessionIndexMirrorPath()`. |
| `$GORMES_HOME/memory.db` | SQLite memory database. | `config.MemoryDBPath()`. |
| `$GORMES_HOME/memory/USER.md` | Markdown mirror of extracted memory entities and edges. | `[telegram].mirror_path` default. |
| `$GORMES_HOME/gormes.log` | Runtime log. | `config.LogPath()`. There is no separate `gateway.log`. |
| `$GORMES_HOME` | TUI panic crash dumps. | `config.CrashLogDir()`. |
| `$GORMES_HOME/kanban.db` | Default Kanban SQLite database. | `config.KanbanDBPath()`. Override with `GORMES_KANBAN_DB`, or relocate the parent with `GORMES_KANBAN_HOME`. |
| `$GORMES_HOME/kanban/boards/<slug>/kanban.db` | Named Kanban boards. | Lives under `KanbanHome()`. |
| `$GORMES_HOME/skills` | Skills root. | `Config.SkillsRoot()`. Override via `[skills].root` or `GORMES_SKILLS_ROOT`. |
| `<skills root>/usage.jsonl` | Append-only skill usage log. | `Config.SkillsUsageLogPath()`. |
| `$GORMES_HOME/hooks` | Gateway `HOOK.yaml` directory. | `config.HooksRoot()`. |
| `$GORMES_HOME/gateway_state.json` | Gateway lifecycle read-model. | `config.GatewayRuntimeStatusPath()`. |
| `$GORMES_HOME/gateway-locks` | Token-scoped gateway credential locks. | `config.GatewayLockDir()`. |
| `$GORMES_HOME/BOOT.md` | Boot hook prompt rendered by the gateway. | `config.BootPath()`. |
| `$GORMES_HOME/cron/CRON.md` | Cron mirror destination. | `Config.CronMirrorPath()`. Override via `[cron].mirror_path`. |
| `$GORMES_HOME/subagents/runs.jsonl` | Subagent run log. | `DelegationCfg.ResolvedRunLogPath()`. Override via `[delegation].run_log_path`. |
| `$GORMES_HOME/tools/audit.jsonl` | Tool execution audit log. | `config.ToolAuditLogPath()`. |

## Resolution rules

- `$GORMES_HOME` is read once per process. The value comes from the
  environment; when unset, Gormes uses `~/.gormes`. There is no fallback to
  `$XDG_CONFIG_HOME/gormes` or `$XDG_STATE_HOME/gormes`.
- `config.toml` lives only at `$GORMES_HOME/config.toml`. There is no
  project-local override.
- `.env` is loaded from `$GORMES_HOME/.env` only. The shell environment
  always wins over a value read from `.env`.
- Source-managed clones owned by `install.sh` live outside `$GORMES_HOME` —
  they are tracked separately and not implied by these paths. Override the
  clone location with `GORMES_SOURCE_ROOT` or `GORMES_SOURCE_DIR`.

## Runtime log

The single runtime log lives at `$GORMES_HOME/gormes.log`. It is the source
for `gormes logs` and the dashboard log feed. Crash dumps from TUI panics are
written next to it under `$GORMES_HOME`.

## Verifying paths

```bash
gormes config path          # prints config.toml location
gormes config env-path      # prints .env location
gormes config show          # resolved config, secrets redacted
gormes doctor --offline     # validates readiness without provider calls
gormes setup --quick        # fills missing setup items only
```
