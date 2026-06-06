---
title: "Upstream Lessons"
weight: 25
---

# Upstream Lessons

Gormes does not copy upstream projects line by line. It absorbs their durable contracts.
The [Completion Plan](../architecture_plan/completion-plan/) is the delivery
contract for turning those lessons into Hermes-in-Go parity.

Hermes is the capability ledger for the agent runtime: provider routing,
prompt assembly, tool continuations, gateway sessions, cron, memory providers,
skills, plugins, and operator commands. Honcho and the Go donor references shape
memory, durable work, retrieval evidence, and auditable runtime knowledge.

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
| Pasted tool progress like `📚 skill_view: "plan"` and `💻 execute_code: "..."` | This is gateway/channel progress, not the current Ink TUI shelf. Gormes must preserve snake_case tool names, preview truncation, `new/all/verbose` modes, and `(×N)` collapse where channels expose that form; shell-only `execute_code` uses the generic exec emoji instead of a Python-specific one. |
| Gormes only showed `🔧 tool done: execute_code` while Hermes exposed `read_file`, `search_files`, `terminal`, `write_file`, and `patch` as distinct actions | Tool progress rendering is not tool inventory. Gormes needs both the shared renderer that formats started-tool events and native descriptors/handlers registered in the default runtime registry so the model can choose the same task tools Hermes exposes. |
| Gormes bot answered as Hermes or lacked the expected default persona/reset behavior | Persona, SOUL.md, USER.md, MEMORY.md, skill templates, and agent-template reset behavior are part of the live-turn prompt contract, not copy-only docs. |
| Confusion about whether `install.sh` needs pushed `main` | Final-user install validates a cloned branch. Dirty development work is validated with `go run ./cmd/gormes` or a rebuilt `./bin/gormes`; pushing is not required for local proof. |

## Durable Contracts To Absorb

| Contract | Donor | Gormes target |
|---|---|---|
| Contract-first operations | Go donors | Operation/tool descriptors drive schemas, commands, doctor, audit, and fixtures. |
| Trust-class enforcement | Hermes gateway | `operator`, `gateway`, `child-agent`, and `system` are enforced before handlers run. |
| Stable prompt assembly | Hermes | Stable system layers; ephemeral recall injected into the current user turn. |
| Provider-neutral events | Hermes | Adapters own provider quirks; `internal/llm` emits one stream/tool-call contract. |
| Durable jobs | Go donors | Cron, long work, and subagents get restartable ledgers, bounded queues, worker-health evidence, and operator inspection. |
| Provenance-rich memory | Honcho + Go donors | Entities and relationships carry source turn, extractor, confidence, freshness, and review state. |
| Explainable code-context retrieval | Gormes-owned | Parent-scope symbols and call edges become optional evidence for skill/retrieval explanations, not a required TypeScript indexer in the runtime. |
| Skills as reviewed code | Hermes | Skills have metadata, resolver tests, inactive drafts, review, feedback, and version history. |
| Visible degraded mode | Hermes | Missing embeddings, provider limits, stale extraction, plugin/tool gaps, and dead letters surface in status/doctor/audit. |

## Space Agent Reference Lessons

Space Agent was cloned locally at `../space-agent` from
`https://github.com/agent0ai/space-agent` and inspected at commit `9c26f9f`.
It is not a Hermes parity source, but it is a useful design donor for
agent-maintained runtime surfaces, browser automation UX, and durable
operator contracts.

| Pattern | Space Agent source | Lesson for Gormes |
|---|---|---|
| Hierarchical agent docs as runtime contracts | `../space-agent/AGENTS.md`; `../space-agent/app/AGENTS.md`; module-local `AGENTS.md` files | Keep Gormes' repo-local skills, AGENTS.md, and building docs as binding contracts near the owning code. When a stable seam changes, update the closest contract doc in the same pass instead of relying on README prose. |
| Layered customware model | `../space-agent/app/AGENTS.md`; `../space-agent/server/lib/customware/AGENTS.md` | Space Agent's `L0` firmware, `L1` group, and `L2` user layering is a useful mental model for future Gormes skill/plugin/user-state layering. Gormes should still preserve Hermes/Goncho homes and Go package ownership rather than importing the browser-first filesystem shape directly. |
| Metadata-driven skill visibility and placement | `../space-agent/app/L0/_all/mod/_core/skillset/AGENTS.md`; `../space-agent/app/L0/_all/mod/_core/skillset/skills.js` | Skill eligibility, auto-load, and system/transient/history placement are better expressed in skill metadata plus context tags than hardcoded prompt-builder branches. This reinforces Gormes skill parity work around platform/context filters, load order, and prompt placement. |
| Prompt contributors as keyed, budgeted items | `../space-agent/app/L0/_all/mod/_core/agent_prompt/prompt-items.js`; `../space-agent/app/L0/_all/mod/_core/agent_prompt/prompt-runtime.js` | Gormes prompt assembly and compression should model system/history/transient contributors as keyed objects with cached token counts and trim policy. Long-context trimming should expose a runtime read-back seam instead of silently dropping text. |
| Plain markdown memory includes | `../space-agent/app/L0/_all/mod/_core/memory/AGENTS.md` | Space Agent keeps durable behavior and rolling memories as prompt-include markdown files. Gormes already has USER.md/MEMORY.md/SOUL.md pressure; future memory UX should keep plain-text operator-editable files as first-class, even when Goncho adds structured provenance underneath. |
| Browser automation typed refs and action effects | `../space-agent/app/L0/_all/mod/_core/web_browsing/AGENTS.md`; `../space-agent/tests/browser_content_format_test.mjs`; `../space-agent/app/L0/_all/mod/_core/skillset/ext/skills/browser-control/SKILL.md` | Browser captures should prefer compact typed refs such as `[button 6] Confirm`, scoped reads, detail-on-demand, and action-effect evidence (`reacted`, `noObservedEffect`, validation text) over raw DOM dumps. This is directly relevant to Gormes browser-harness parity. |
| Stream-aware supervisor and replaceable children | `../space-agent/commands/lib/supervisor/AGENTS.md`; `../space-agent/commands/supervise.js`; `../space-agent/tests/supervise_command_test.mjs` | A production supervisor can keep children opaque, bind public traffic itself, use private child ports, stage updates outside the live checkout, drain long streams, and expose short process titles. Gormes gateway/service management should borrow the operator clarity without moving core runtime logic into the supervisor. |
| Centralized app-file permissions and mutation validation | `../space-agent/server/api/AGENTS.md`; `../space-agent/server/lib/customware/AGENTS.md`; `../space-agent/tests/file_write_operations_test.mjs` | File tools should centralize path normalization, permission checks, batch preflight, and mutation semantics before writes. Gormes file-tool parity remains Hermes-defined, but Space Agent's append/prepend/insert tests are useful fixture shapes for safe local file operations. |
| Git-backed user/group time travel | `../space-agent/server/lib/git/AGENTS.md`; `../space-agent/server/lib/customware/AGENTS.md` | Per-owner local Git history with serialized operations, rollback/revert previews, and protected auth metadata is a strong donor concept for Gormes `/snapshot`, `/branch`, and recovery UX. The Go implementation should stay local-first and explicit about which roots are protected. |

Do not copy Space Agent's browser-first architecture wholesale. Gormes remains
a Go Hermes-compatible runtime with gateway, CLI, provider, tool, memory, and
Goncho boundaries. The reusable lessons are mostly contract shape,
metadata-driven discovery, prompt budgeting, browser ref UX, and operator
supervision patterns.

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

- [Upstream Hermes Reference](../../upstream-hermes/)
- [Hermes Source Study](../../upstream-hermes/source-study/)
- [Hermes Good And Bad](../../upstream-hermes/good-and-bad/)

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
| Provenance-rich memory | `engram/internal/persistence/store/store.go`, `engram/internal/persistence/store/relations.go` |
| Skills as reviewed code | `engram/internal/persistence/store/store.go` for storage; extractor itself is `provenance.origin_type: gormes` |
| Visible degraded mode | `goclaw/internal/oauth/openai_quota_transport.go` (degraded-state classification) |
| Cancellable session-scoped workers | `trpc-agent-go/agent/await_user_reply.go` |
| Before/after callback pipeline | `trpc-agent-go/model/callbacks.go`, `trpc-agent-go/agent/callbacks.go` |

Use the `gormes-references` skill
(`development-skills/gormes-references/SKILL.md`) for
runtime/tools/memory/utility lookups and the `gormes-provider-parity` skill
(`development-skills/gormes-provider-parity/SKILL.md`) for
provider/auth/streaming/quota/retry. Always add a `// Adapted from
<donor>/...::Symbol` comment when porting code.

## Decision

The better Gormes architecture is:

```text
Hermes-class capability
+ Gormes-owned operation contracts
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
