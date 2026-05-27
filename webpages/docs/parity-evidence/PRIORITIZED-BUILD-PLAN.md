# Parity Atom Implementation Plan

Prioritized build order for all 164 remaining missing atoms across 45 sections.
Scored by: **impact** (user-visible, unblocks others), **buildability** (slice size, existing infra),
**dependency** (blocks other atoms or rows).

## Priority Tiers

### P0 — High impact, small surface, immediate user-visible value

| # | Section | Missing | Target | Est. slices | Why |
|---|---|---|---|---|---|
| 1 | 41. Model Switch | 5 | `internal/hermes/`, `cmd/gormes/model.go` | 2 | Direct aliases, ModelIdentity parsing, --global flag. Existing picker, small gaps. |
| 2 | 46. Personality | 5 | `internal/hermes/`, `cmd/gormes/` | 2 | List/switch/none clear. `/personality` is advertised unavailable — user-facing. |
| 3 | 19. Config System | 2 | `internal/config/` | 1 | Schema validation and `cli-config.yaml.example` mirror. |
| 4 | 28. Goal/Subgoal | 2 | `internal/tools/`, `internal/tui/` | 1 | Subgoal add/remove and budget enforcement. Already have goal infrastructure. |

### P1 — High volume, medium complexity, unblocks delivery surface

| # | Section | Missing | Target | Est. slices | Why |
|---|---|---|---|---|---|
| 5 | 41. Cron delivery | 3 | `internal/cron/`, `internal/gateway/` | 2 | Grace seconds, delivery resolution, multi-target delivery. Cron infra exists. |
| 6 | 42. Send Command | 6 | `cmd/gormes/send.go` | 2 | Target resolution, body reading, result formatting. Oneshot path exists. |
| 7 | 25. Voice/PTY | 7 | `internal/tools/`, `internal/hermes/` | 3 | TTS result envelope, voice state machine, TTS provider abstraction. Whisper STT exists. |
| 8 | 7. Cron (from 13) | 3 | `internal/cron/` | 2 | Context_from chaining, resource release, job lock files, prompt guard, recovery. |
| 9 | 12. Release | 5 | `Makefile`, `www.gormes.ai`, `install.sh` | 3 | OCI image, Homebrew, Nix build, release script, docker entrypoint. |
| 10 | 16. Gateway Deep | 3 | `internal/gateway/` | 2 | Boot hooks, webhook CLI, backup CLI. Existing gateway infra. |

### P2 — Medium volume, some depth, good TDD targets

| # | Section | Missing | Target | Est. slices | Why |
|---|---|---|---|---|---|
| 11 | 15. Provider Deep | 13 | `internal/hermes/` | 6 | Bedrock SigV4, Gemini adapter, Google Code Assist. Largest block — needs splitting. |
| 12 | 13. Goncho Memory | 10 | `internal/goncho/`, `internal/memory/` | 5 | File-backed messages, conclusions, representations, context retrieval, dialectic, dreaming. Honcho SDK fixture. |
| 13 | 44. PTY | 11 | `internal/tools/`, `internal/cmdrunner/` | 4 | PTY bridge (spawn/read/write/resize/close), process registry. No existing infra. |
| 14 | 17. Tools Deep | 18 | `internal/tools/`, `internal/cmdrunner/` | 6 | URL safety, website policy, OSV check, send-message tool, debug helpers, sandbox environments. |
| 15 | 30. Cron/Learning | 8 | `internal/cron/`, `internal/skills/` | 4 | Curator state machine, entity discovery, candidate extraction, review. |

### P3 — Low urgency, large scope, deferred

| # | Section | Missing | Target | Est. slices | Why |
|---|---|---|---|---|---|
| 16 | 45. Env Passthrough | 8 | `internal/tools/` | 3 | Credential file mounts, skills mount, env passthrough registration. Sandbox-only. |
| 17 | 24. CLI Utilities | 9 | `cmd/gormes/` | 4 | Tips, inventory, dump, timeouts, callbacks, relaunch. Low user impact. |
| 18 | 27. Kanban Deep | 6 | `internal/tools/kanban/` | 3 | Link/unlink, comment, heartbeat/reclaim/zombie, archiving. Existing kanban; gaps are edge features. |
| 19 | 37. Security Advisories | 7 | `internal/doctor/`, `cmd/gormes/` | 3 | Advisory class, detection, ack, banner. Important but no immediate breakage. |
| 20 | 40. Backup/Restore | 7 | `cmd/gormes/backup.go` | 3 | Quick snapshot, list, prune, import. Existing checkpoint/rollback in TUI. |

### P4 — Research/future, needs scoping before building

| # | Section | Missing | Target | Est. slices | Why |
|---|---|---|---|---|---|
| 21 | 10. APIServer | 20 | `internal/apiserver/` | 6 | 20 API endpoints (models, responses, runs, jobs). Large surface; deprioritize until API server is the focus. |
| 22 | 33. Codex Runtime | 6 | `internal/hermes/`, `cmd/gormes/` | 3 | Codex runtime switch, migration, provider, copilot/vercel/dingtalk auth. Depends on upstream Codex OAuth status. |
| 23 | 31. ACP Adapter | 6 | `internal/plugins/`, `internal/apiserver/` | 3 | ACP auth, server, events, permissions, session, tools. No consumer yet. |
| 24 | 34. Root Environments | 5 | `internal/hermes/` | 3 | Agentic OPD, web research, raw tool-call parsers. Research-only until sandbox rows land. |
| 25 | 23. Batch/SWE/RL | 5 | deferred | 4 | Batch runner, mini-SWE, RL, datagen. Research modes; defer until normal turn is stable. |
| 26 | 50. STDIO | 1 | `cmd/gormes/` | 1 | Pipe mode. Low usage in Gormes context. |

## Execution Strategy

| Phase | Sections | Slices | Est. sessions |
|---|---|---|---|
| 1. Quick wins (P0) | 41, 46, 19, 28 | 6 | 2-3 |
| 2. Delivery surface (P1) | 7, 42, 25, 41, 12, 16 | 14 | 5-7 |
| 3. Core parity (P2) | 15, 13, 44, 17, 30 | 24 | 8-12 |
| 4. Polish (P3) | 45, 24, 27, 37, 40 | 16 | 6-8 |
| 5. Future (P4) | 10, 33, 31, 34, 23, 50 | 20 | 8-12 |

**Total:** ~80 slices across 5 phases. Estimated 30-40 sessions.

## Next Slice (Phase 1, #1)

**Section 41 (Model Switch) — 5 missing atoms:**
1. Direct alias resolution (`_ensure_direct_aliases`)
2. ModelIdentity parsing (`provider/model` → struct)
3. ModelSwitchResult (structured result)
4. `--global` flag parity
5. Model sort key (`_model_sort_key`)

Target: `internal/hermes/model_switch.go` + `cmd/gormes/model.go`