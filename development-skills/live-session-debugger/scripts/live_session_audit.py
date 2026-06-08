#!/usr/bin/env python3
"""Read-only Gormes live-session audit helper.

The script intentionally summarizes local state and avoids printing secrets or
full transcript bodies. It treats sessions.db as bbolt/opaque and memory.db as
SQLite.
"""

from __future__ import annotations

import json
import os
import pathlib
import sqlite3
import stat
import subprocess
import sys
from collections import Counter
from datetime import datetime, timezone
from typing import Any

SECRET_NAMES = {"auth.json", ".env", "config.toml"}


def iso(ts: float) -> str:
    return datetime.fromtimestamp(ts, tz=timezone.utc).isoformat().replace("+00:00", "Z")


def file_info(path: pathlib.Path) -> dict[str, Any]:
    try:
        st = path.stat()
    except FileNotFoundError:
        return {"path": str(path), "state": "missing"}
    return {
        "path": str(path),
        "state": "present",
        "kind": "dir" if path.is_dir() else "file",
        "mode": stat.filemode(st.st_mode),
        "size_bytes": st.st_size,
        "modified_at": iso(st.st_mtime),
        "world_readable": bool(st.st_mode & stat.S_IROTH),
        "group_writable": bool(st.st_mode & stat.S_IWGRP),
    }


def run(cmd: list[str], timeout: int = 5) -> dict[str, Any]:
    try:
        p = subprocess.run(cmd, text=True, capture_output=True, timeout=timeout)
    except FileNotFoundError:
        return {"cmd": cmd, "error": "not_found"}
    except subprocess.TimeoutExpired:
        return {"cmd": cmd, "error": "timeout"}
    return {"cmd": cmd, "returncode": p.returncode, "stdout": p.stdout[-4000:], "stderr": p.stderr[-1000:]}


def sqlite_scalar(cur: sqlite3.Cursor, sql: str) -> Any:
    cur.execute(sql)
    row = cur.fetchone()
    return row[0] if row else None


def audit_memory_db(path: pathlib.Path) -> dict[str, Any]:
    out: dict[str, Any] = {"path": str(path)}
    if not path.exists():
        out["state"] = "missing"
        return out
    try:
        con = sqlite3.connect(f"file:{path}?mode=ro", uri=True)
    except sqlite3.Error as e:
        out.update({"state": "error", "error": str(e)})
        return out
    try:
        cur = con.cursor()
        out["quick_check"] = sqlite_scalar(cur, "PRAGMA quick_check")
        out["journal_mode"] = sqlite_scalar(cur, "PRAGMA journal_mode")
        tables = {r[0] for r in cur.execute("SELECT name FROM sqlite_master WHERE type='table'")}
        out["tables_present"] = sorted(t for t in tables if t in {"turns", "goncho_session_summaries"})
        if "turns" in tables:
            cur.execute(
                """
                SELECT count(*), count(DISTINCT session_id),
                       min(ts_unix), max(ts_unix)
                FROM turns
                """
            )
            turns, sessions, first_ts, last_ts = cur.fetchone()
            out["turns"] = {
                "rows": turns,
                "sessions": sessions,
                "first_at": iso(first_ts) if first_ts else None,
                "last_at": iso(last_ts) if last_ts else None,
            }
            cur.execute("SELECT role, count(*) FROM turns GROUP BY role ORDER BY role")
            out["roles"] = dict(cur.fetchall())
            cur.execute("SELECT memory_sync_status, count(*) FROM turns GROUP BY memory_sync_status ORDER BY memory_sync_status")
            out["memory_sync_status"] = dict(cur.fetchall())
            cur.execute(
                """
                WITH groups AS (
                  SELECT session_id, role, ts_unix, content, count(*) cnt
                  FROM turns
                  GROUP BY session_id, role, ts_unix, content
                )
                SELECT count(*), coalesce(sum(cnt),0), coalesce(sum(cnt-1),0)
                FROM groups WHERE cnt > 1
                """
            )
            groups, grouped_rows, removable = cur.fetchone()
            out["duplicates"] = {
                "groups": groups,
                "rows_in_duplicate_groups": grouped_rows,
                "extra_rows_if_deduped": removable,
            }
            cur.execute(
                """
                SELECT session_id, chat_id, count(*), min(ts_unix), max(ts_unix)
                FROM turns
                GROUP BY session_id, chat_id
                ORDER BY max(ts_unix) DESC
                LIMIT 20
                """
            )
            out["session_chat_counts"] = [
                {
                    "session_id": sid,
                    "chat_id": chat,
                    "rows": rows,
                    "first_at": iso(first_ts) if first_ts else None,
                    "last_at": iso(last_ts) if last_ts else None,
                }
                for sid, chat, rows, first_ts, last_ts in cur.fetchall()
            ]
        if "goncho_session_summaries" in tables:
            out["goncho_session_summaries"] = sqlite_scalar(cur, "SELECT count(*) FROM goncho_session_summaries")
    except sqlite3.Error as e:
        out["error"] = str(e)
    finally:
        con.close()
    return out


def audit_jsonl(path: pathlib.Path) -> dict[str, Any]:
    out = file_info(path)
    if out.get("state") != "present":
        return out
    lines = bad = 0
    statuses: Counter[str] = Counter()
    keys: Counter[str] = Counter()
    try:
        with path.open(errors="replace") as f:
            for line in f:
                if not line.strip():
                    continue
                lines += 1
                try:
                    obj = json.loads(line)
                except json.JSONDecodeError:
                    bad += 1
                    continue
                keys.update(obj.keys())
                if "status" in obj:
                    statuses[str(obj["status"])] += 1
                if "event" in obj:
                    statuses[f"event:{obj['event']}"] += 1
    except OSError as e:
        out["error"] = str(e)
        return out
    out.update({"jsonl_lines": lines, "bad_json_lines": bad, "top_keys": keys.most_common(12), "statuses": statuses.most_common(12)})
    return out


def audit_profiles(home: pathlib.Path) -> dict[str, Any]:
    profiles_dir = home / "profiles"
    out: dict[str, Any] = {"path": str(profiles_dir), "state": "missing", "profiles": {}}
    if not profiles_dir.exists():
        return out
    out["state"] = "present"
    for profile_root in sorted(p for p in profiles_dir.iterdir() if p.is_dir()):
        name = profile_root.name
        out["profiles"][name] = {
            "root": str(profile_root),
            "memory_db": audit_memory_db(profile_root / "memory.db"),
            "sessions_db": file_info(profile_root / "sessions.db"),
            "session_index": file_info(profile_root / "sessions" / "index.yaml"),
        }
    return out


def scan_permissions(home: pathlib.Path) -> list[dict[str, Any]]:
    risky = []
    for path in [home, home / "sessions" / "index.yaml", home / "tools" / "audit.jsonl", home / "subagents" / "runs.jsonl", home / "memory" / "MEMORY.md", home / "memory" / "USER.md"]:
        info = file_info(path)
        if info.get("world_readable") or info.get("group_writable"):
            risky.append(info)
    audio = home / "cache" / "audio"
    if audio.exists():
        count = world = 0
        total = 0
        for p in audio.glob("*"):
            if not p.is_file():
                continue
            count += 1
            st = p.stat()
            total += st.st_size
            if st.st_mode & stat.S_IROTH:
                world += 1
        risky.append({"path": str(audio), "audio_files": count, "total_bytes": total, "world_readable_files": world})
    return risky


def main() -> int:
    home = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 and sys.argv[1] else os.environ.get("GORMES_HOME", "~/.gormes")).expanduser()
    report: dict[str, Any] = {
        "home": str(home),
        "generated_at": iso(datetime.now(tz=timezone.utc).timestamp()),
        "files": {name: file_info(home / name) for name in ["sessions.db", "memory.db", "gateway.pid", "gateway_state.json", "sessions/index.yaml"]},
        "runtime": {
            "gateway_pid": file_info(home / "runtime" / "gateway.pid"),
            "gateway_state": file_info(home / "runtime" / "gateway_state.json"),
            "runtime_log": file_info(home / "runtime" / "runtimegateway.log"),
        },
        "memory_db": audit_memory_db(home / "memory.db"),
        "profiles": audit_profiles(home),
        "jsonl": {
            "subagent_runs": audit_jsonl(home / "subagents" / "runs.jsonl"),
            "tool_audit": audit_jsonl(home / "tools" / "audit.jsonl"),
            "lifecycle_install": audit_jsonl(home / "lifecycle" / "install.log.jsonl"),
        },
        "permission_risks": scan_permissions(home),
        "process_locks": {},
    }
    sessions_db_candidates = [home / "sessions.db"] + sorted((home / "profiles").glob("*/sessions.db"))
    for sessions_db in sessions_db_candidates:
        if sessions_db.exists():
            key = str(sessions_db.relative_to(home))
            report["process_locks"][key] = {
                "lsof": run(["lsof", str(sessions_db)]),
                "fuser": run(["fuser", "-v", str(sessions_db)]),
            }
    print(json.dumps(report, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
