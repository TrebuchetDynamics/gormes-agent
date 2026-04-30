---
title: "Upstream Lessons"
weight: 25
---

# Upstream Lessons

Gormes does not copy Hermes or GBrain. It absorbs their durable contracts.
The [Completion Plan](../architecture_plan/completion-plan/) is the delivery
contract for turning those lessons into Hermes-in-Go parity.

Hermes is the capability ledger for the agent runtime: provider routing,
prompt assembly, tool continuations, gateway sessions, cron, memory providers,
skills, plugins, and operator commands. GBrain is the architecture donor for
contract-first operations, durable jobs, knowledge graph provenance, retrieval
evaluation, and skills as auditable runtime knowledge.

The combined lesson is simple:

```text
port contracts
reject monoliths
preserve Go ownership boundaries
prove behavior with fixtures
show degraded mode to operators
```

## Live Dogfood Lessons

Recent Gormes-vs-Hermes dogfood exposed a recurring pattern: the final answer
can be correct while the operator contract is still wrong. Treat these as
source-backed parity lessons for future rows and tests:

| Artifact | Lesson for Gormes |
|---|---|
| Installed `gormes` asking the operator to start `hermes gateway start` | Gormes startup, installers, and shell-wide binaries must never depend on Hermes services. Runtime validation must prove the exact binary path, source checkout, and `GORMES_HOME` before debugging higher layers. |
| Switching between `go run ./cmd/gormes`, `./bin/gormes`, and installed `gormes` caused `sessions.db` locks | Development source, local binary, installed binary, runtime home, and gateway owner are separate surfaces. Use `gormes gateway status/stop` or isolated homes; never delete session state to clear a lock. |
| Telegram showed hourglass/status bubbles, duplicate final replies, or raw `tool iteration limit exceeded` text | Gateway UX is a first-class contract. Fixtures must assert message count, send/edit/delete order, tool-progress visibility, and final text together. |
| Pasted tool progress like `📚 skill_view: "plan"` and `🐍 execute_code: "..."` | This is Hermes gateway/channel progress, not the current Ink TUI shelf. Gormes must preserve emoji, snake_case tool names, preview truncation, `new/all/verbose` modes, and `(×N)` collapse where channels expose that form. |
| Gormes only showed `🔧 tool done: execute_code` while Hermes exposed `read_file`, `search_files`, `terminal`, `write_file`, and `patch` as distinct actions | Tool progress rendering is not tool inventory. Gormes needs both the shared renderer that formats started-tool events and native descriptors/handlers registered in the default runtime registry so the model can choose the same task tools Hermes exposes. |
| Gormes bot answered as Hermes or lacked the expected default persona/reset behavior | Persona, SOUL.md, USER.md, MEMORY.md, skill templates, and agent-template reset behavior are part of the live-turn prompt contract, not copy-only docs. |
| Confusion about whether `install.sh` needs pushed `main` | Final-user install validates a cloned branch. Dirty development work is validated with `go run ./cmd/gormes` or a rebuilt `./bin/gormes`; pushing is not required for local proof. |

## Durable Contracts To Absorb

| Contract | Donor | Gormes target |
|---|---|---|
| Contract-first operations | GBrain | Operation/tool descriptors drive schemas, commands, doctor, audit, and fixtures. |
| Trust-class enforcement | GBrain + Hermes gateway | `operator`, `gateway`, `child-agent`, and `system` are enforced before handlers run. |
| Stable prompt assembly | Hermes | Stable system layers; ephemeral recall injected into the current user turn. |
| Provider-neutral events | Hermes | Adapters own provider quirks; `internal/hermes` emits one stream/tool-call contract. |
| Durable jobs | GBrain | Cron, long work, and subagents get restartable ledgers, bounded queues, worker-health evidence, and operator inspection. |
| Provenance-rich memory | GBrain | Entities and relationships carry source turn, extractor, confidence, freshness, and review state. |
| Explainable code-context retrieval | GBrain | Parent-scope symbols and call edges become optional evidence for skill/retrieval explanations, not a required TypeScript indexer in the runtime. |
| Skills as reviewed code | GBrain + Hermes | Skills have metadata, resolver tests, inactive drafts, review, feedback, and version history. |
| Visible degraded mode | GBrain + Hermes | Missing embeddings, provider limits, stale extraction, plugin/tool gaps, and dead letters surface in status/doctor/audit. |

## Latest Sync Notes

The 2026-04-30 Gormes dogfood/docs pass locks two more operator lessons into
the roadmap:

- Gateway/TUI/Slack/Discord progress now share one Hermes-style tool-trace
  renderer, including emoji labels, quoted previews, `(×N)` duplicate
  collapse, and suppression of legacy `tool done:` noise.
- The default registry now exposes first-pass native local task tools:
  `read_file`, `search_files`, `write_file`, `patch`, and foreground
  `terminal`. This closes the immediate `execute_code` collapse, while
  background process registry, PTY, fuzzy patch/lint/checkpoint restore,
  session-bound `todo`, and session-bound `session_search` remain explicit
  follow-up parity work.

The 2026-04-27 upstream Hermes sync adds a few contract deltas that are now
split in `progress.json`:

- Yuanbao is a new gateway family; Gormes tracks protocol/markdown, media
  normalization, then disabled-by-default runtime/toolset registration as
  separate Phase 7.E rows.
- Airtable is a bundled productivity skill; Gormes treats it as reviewed
  SKILL.md content plus optional credential evidence, not as a live integration
  at startup.
- Checkpoint shadow-repo cleanup and file-read dedup guards become the first
  Phase 5.L slices before write-capable file tools.
- Compression now needs an explicit ContextEngine boundary notification after
  successful compression so cache state cannot span a lineage change silently.
- Session search recent mode must exclude the current lineage root
  deterministically while preserving GONCHO/Honcho-compatible scope rules.

## The Four Questions

Every planned subsystem should answer these before implementation:

1. **What contract are we porting?** Name the source files and external
   behavior. Do not use "port file X" as the requirement.
2. **What trust class can call it?** Operator-local, gateway-user,
   child-agent, and system/cron paths do not share the same permissions.
3. **How is degraded mode reported?** Partial capability is acceptable only
   when operators can see it in status, doctor, audit, logs, or docs.
4. **What fixture proves compatibility?** Prefer replayable local fixtures over
   live credentials, live platforms, or a real provider.

## Phase Mapping

- [Phase 2 Gateway](../architecture_plan/phase-2-gateway/) owns command policy,
  active-turn behavior, adapter contracts, cron, and subagent runtime.
- [Phase 3 Memory](../architecture_plan/phase-3-memory/) owns provenance,
  scoped recall, retrieval evaluation, and degraded memory health.
- [Phase 4 Brain Transplant](../architecture_plan/phase-4-brain-transplant/)
  owns stable prompt assembly, context compression, provider adapters, and
  transcript fixtures.
- [Phase 5 Final Purge](../architecture_plan/phase-5-final-purge/) owns
  operation/tool descriptor parity before handler ports.
- [Phase 6 Learning Loop](../architecture_plan/phase-6-learning-loop/) owns
  skills as reviewed code, resolver evals, feedback records, and safe
  self-improvement.

## Upstream Study References

- [Upstream GBrain Study](../../upstream-gbrain/)
- [GBrain Architecture](../../upstream-gbrain/architecture/)
- [GBrain Good And Bad](../../upstream-gbrain/good-and-bad/)
- [GBrain Gormes Takeaways](../../upstream-gbrain/gormes-takeaways/)
- [Upstream Hermes Reference](../../upstream-hermes/)
- [Hermes Source Study](../../upstream-hermes/source-study/)
- [Hermes Good And Bad](../../upstream-hermes/good-and-bad/)
- [Hermes Gormes Takeaways](../../upstream-hermes/gormes-takeaways/)

## Go Donor Anchors

Hermes Python defines what to port; Go donors under `references/go-agent-os/`
define how to shape it idiomatically in Go. Each durable contract has at least
one donor file that already implements its shape:

| Contract | Go donor anchor |
|---|---|
| Contract-first operations | `nanobot/pkg/tools/service.go`, `nanobot/pkg/tools/flows.go` |
| Trust-class enforcement | `nanobot/pkg/tools/flows.go`, `axe/internal/tool/` |
| Stable prompt assembly | `nanobot/pkg/runtime/runtime.go` |
| Provider-neutral events | `references/go-agent-os/GORMES-PROVIDER-PATTERN-REFERENCES.md` |
| Durable jobs | `engram/internal/mcp/write_queue.go`, `engram/internal/mcp/activity.go` |
| Provenance-rich memory | `engram/internal/store/store.go`, `engram/internal/store/relations.go` |
| Skills as reviewed code | `engram/internal/store/store.go` for storage; extractor itself is `provenance.origin_type: gormes` |
| Visible degraded mode | `goclaw/internal/oauth/openai_quota_transport.go` (degraded-state classification) |
| Cancellable session-scoped workers | `trpc-agent-go/agent/await_user_reply.go` |
| Before/after callback pipeline | `trpc-agent-go/model/callbacks.go`, `trpc-agent-go/agent/callbacks.go` |

Use the `gormes-references` skill
(`docs/development-skills/gormes-references/SKILL.md`) for
runtime/tools/memory/utility lookups and the `gormes-provider-parity` skill
(`docs/development-skills/gormes-provider-parity/SKILL.md`) for
provider/auth/streaming/quota/retry. Always add a `// Adapted from
<donor>/...::Symbol` comment when porting code.

## Decision

The better Gormes architecture is:

```text
Hermes-class capability
+ GBrain-style operation contracts
+ Go single-owner kernel
+ provider-neutral stream fixtures
+ registry-owned gateway policy
+ descriptor-owned tool safety
+ GONCHO-scoped memory provenance
+ reviewed skill lifecycle
+ visible degraded-mode checks
```

That keeps the upstream lessons while preserving the product promise: one
small Go runtime, explicit boundaries, local-first state, and no Python runtime
dependency in the final path.
