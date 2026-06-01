---
name: live-session-debugger
description: Audit live Gormes agent sessions under ~/.gormes, including sessions.db locks, memory.db transcripts, gateway state, profile mirrors, permissions, logs, and repair planning. Use when the user asks to audit or debug local live sessions, agent sessions, ~/.gormes state, session duplication, stale gateways, or session repair.
---

# Live Session Debugger

## Quick start

Start read-only. Confirm the runtime home and ownership before inspecting state:

```bash
pwd
git rev-parse --show-toplevel
git rev-parse --abbrev-ref HEAD
git status --short
GORMES_HOME=${GORMES_HOME:-$HOME/.gormes} python3 development-skills/live-session-debugger/scripts/live_session_audit.py "$GORMES_HOME"
```

Then run focused Gormes commands through the named home, never by assuming the
installed binary is current:

```bash
which -a gormes || true
readlink -f "$(command -v gormes)" 2>/dev/null || true
GORMES_HOME="$GORMES_HOME" go run ./cmd/gormes gateway status --json 2>/dev/null || true
GORMES_HOME="$GORMES_HOME" go run ./cmd/gormes session list --json 2>/dev/null || true
GORMES_HOME="$GORMES_HOME" go run ./cmd/gormes memory status --json 2>/dev/null || true
```

## Workflow

1. **Bound the pass**
   - Identify `GORMES_HOME`, binary surface (`go run`, `./bin/gormes`, or installed), current branch, and dirty files.
   - Treat existing dirty source edits as user/agent work. Do not discard them.
   - If the home is real operator state (`~/.gormes`), default to audit-only.

2. **Map live owners**
   - Inspect `gateway.pid`, `gateway_state.json`, `gateway.log`, `sessions.db` locks, and active `gormes` processes.
   - If status says dead while a process owns `sessions.db`, record the mismatch; do not kill it without explicit approval.

3. **Audit persistence surfaces**
   - `sessions.db`: bbolt map and metadata mirror; avoid SQLite commands against it.
   - `sessions/index.yaml` and `profiles/*/sessions/index.yaml`: operator mirror freshness, lineage status, stale profile state.
   - `memory.db`: SQLite transcript integrity, turn counts, duplicate turn groups, extractor backlog, summaries, WAL state.
   - `tools/audit.jsonl`, `subagents/runs.jsonl`, lifecycle logs: JSON validity, failed statuses, high-level counts only.

4. **Classify issues**
   - Data safety: corrupt DBs, locks, stale owners, missing backups.
   - Transcript quality: duplicate rows, source/platform mismatch, missing titles, export/recap failures.
   - Runtime health: stale gateway status, live process mismatch, failed reload/restart, extractor backlog.
   - Privacy: broad permissions on transcripts, audit logs, audio cache, memory/context files.
   - Product gaps: read-only commands blocked by live gateway, unimplemented parity rows, misleading errors.

5. **Plan repairs before changing state**
   - Propose code fixes and data repair separately.
   - For data repair, require backup and explicit operator approval before stopping gateway, deleting rows, chmodding broad trees, or rotating logs.
   - Prefer adding regression tests before migration or cleanup code.

## Skill contract

### Entry protocol
- Trivial status question: run read-only audit commands and summarize.
- Medium ambiguity: assume `$HOME/.gormes`, state that assumption, and keep the pass read-only.
- High risk: stop and ask before stopping gateway, modifying DBs, deleting files, chmodding recursively, or exposing transcript content.

### Topology check
- Is the home real operator state or an isolated temp home?
- Which process owns `sessions.db`?
- Which binary surface is being audited?
- Are `sessions.db` (bbolt) and `memory.db` (SQLite) being treated correctly?
- Are logs/transcripts being summarized without leaking secrets or private message bodies?

### Verification gate
A useful audit report includes:
- branch/status and `GORMES_HOME` evidence;
- live owner/process/lock evidence;
- session count, transcript count, duplicate count, and source metadata findings;
- memory extractor state;
- permission risks;
- exact read-only commands run;
- a repair plan split into code fixes, data fixes, and operator-approval actions.

### Red lines
- Do not delete, rewrite, or vacuum live databases without backup and explicit approval.
- Do not kill or restart a live gateway unless the operator approves interruption.
- Do not print secrets from `auth.json`, `.env`, config tokens, raw Authorization headers, or full private transcripts.
- Do not treat `sessions.db` as SQLite; it is bbolt.
- Do not fix lock problems by deleting lock files or databases.

### Output contract
Report compactly:

```text
Live session audit:
- home: <path>
- binary surface: <go run/bin/installed evidence>
- live owner: <pid/none/mismatch>
- sessions: <counts and notable IDs>
- memory: <integrity/backlog/duplicates>
- privacy: <permission risks>
- detected issues: <ranked list>
- repair plan: <safe ordered plan with approval gates>
- commands run: <read-only evidence>
```

## References

- [Audit checklist](references/audit-checklist.md)
