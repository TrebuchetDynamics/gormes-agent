# Live Session Audit Checklist

Use this detail page when the first-pass audit finds anomalies or when the user asks for "all aspects".

## Read-only commands

```bash
export GORMES_HOME=${GORMES_HOME:-$HOME/.gormes}

# Repo and binary surface
pwd
git rev-parse --show-toplevel
git rev-parse --abbrev-ref HEAD
git status --short
which -a gormes || true
readlink -f "$(command -v gormes)" 2>/dev/null || true

# Runtime ownership
GORMES_HOME="$GORMES_HOME" go run ./cmd/gormes gateway status --json 2>/dev/null || true
lsof "$GORMES_HOME/sessions.db" 2>/dev/null || true
fuser -v "$GORMES_HOME/sessions.db" 2>/dev/null || true
ps -eo pid,ppid,stat,lstart,cmd | grep -E 'gormes|cmd/gormes' | grep -v grep || true

# Session and memory surfaces
GORMES_HOME="$GORMES_HOME" go run ./cmd/gormes session list --json 2>/dev/null || true
GORMES_HOME="$GORMES_HOME" go run ./cmd/gormes memory status --json 2>/dev/null || true
sqlite3 "$GORMES_HOME/memory.db" 'PRAGMA quick_check; PRAGMA journal_mode;' 2>/dev/null || true
```

## SQLite transcript probes

Run only against `memory.db`, never against `sessions.db`:

```sql
SELECT count(*) AS turns,
       count(DISTINCT session_id) AS sessions,
       datetime(min(ts_unix),'unixepoch') AS first_utc,
       datetime(max(ts_unix),'unixepoch') AS last_utc
FROM turns;

SELECT session_id, chat_id, count(*) AS turns,
       datetime(min(ts_unix),'unixepoch') AS first_utc,
       datetime(max(ts_unix),'unixepoch') AS last_utc,
       sum(CASE WHEN memory_sync_status!='synced' THEN 1 ELSE 0 END) AS not_synced
FROM turns
GROUP BY session_id, chat_id
ORDER BY max(ts_unix) DESC;

WITH groups AS (
  SELECT session_id, role, ts_unix, content, count(*) cnt
  FROM turns
  GROUP BY session_id, role, ts_unix, content
)
SELECT count(*) AS duplicate_groups,
       sum(cnt) AS rows_in_duplicate_groups,
       sum(cnt-1) AS removable_if_deduped
FROM groups
WHERE cnt > 1;
```

## Findings to look for

- `sessions.db` lock owner exists but `gateway status` says missing/dead.
- Session mirror source differs from `session list` or export platform.
- Transcript rows are duplicated by mirrored `chat_id` rows.
- `turn_key` is missing on one of two otherwise-identical rows.
- `memory status` reports extractor backlog or dead letters.
- `session recap` or read-only commands fail only because the gateway owns bbolt.
- `session stats` is still row-backed or classified but unimplemented.
- Profile `gateway.pid` files are stale and point at dead processes.
- Files containing transcript/audit/audio data are world-readable or group-writable.

## Repair planning pattern

1. **Back up first**: copy DBs, YAML mirrors, gateway pid/state, and logs to a timestamped directory.
2. **Fix product behavior before live data**: add tests for duplicate export/list, source metadata, lock-safe read-only commands, and status validation.
3. **Run focused tests**: prefer `go test ./cmd/gormes ./internal/persistence/session ./internal/memory -run '<specific>' -count=1` before full gates.
4. **Repair data only after approval**: stop gateway, verify backup, run targeted migration/cleanup, restart/reload, and compare counts.
5. **Harden permissions carefully**: chmod specific sensitive files/dirs, not broad recursive trees unless the operator approves.

## Sanitization

Never paste raw `auth.json`, `.env`, provider headers, Telegram tokens, full transcript bodies, or private audio file contents. Prefer counts, hashes, short redacted previews, and command receipts.
