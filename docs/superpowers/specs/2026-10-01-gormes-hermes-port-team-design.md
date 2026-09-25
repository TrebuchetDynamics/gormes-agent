# Gormes Hermes-Port Team Design

- **Status:** Draft for owner review
- **Date:** 2026-10-01
- **Repository:** `/home/xel/git/gormes/gormes-agent`
- **Mode:** Create a new Hermes native-profile team
- **Scope:** Team design and safe provisioning contract only; this document does not create profiles, configure credentials or gateways, launch agents, or authorize inference/release.

## 1. Goal and success criteria

Provide a human-facing Hermes coordinator and a broad, responsibility-based set of specialists to support the Gormes goal of porting upstream Hermes Agent to Go 1:1. Use the existing Gormes parity evidence, subsystem inventory, and canonical progress backlog to derive scoped work; do not create a parallel backlog or infer that a profile's existence means its remit is ready to execute.

Success means:

1. A flat sibling profile set, with a clear remit for each profile and no collision or overwrite.
2. The explicitly selected `openai-codex` provider and `gpt-6-luna` model configured for each profile, without silent substitution or overrides.
3. A root-only human interaction surface, explicitly selected Gormes Kanban board, and verified dispatcher ownership before routing work.
4. A durable, nonsecret team briefing, clear task acceptance and review ownership, and readiness evidence that distinguishes configuration, authentication, inference, and release.
5. No worker task released until the repository workspace policy, independent validation, recovery bounds, and authorization gates are feasible and verified.

## 2. Repository evidence and constraints

The target is `gormes-agent`, not the parent directory or adjacent `goncho`/`hermes-wing` projects. At design time its checkout was at `main`, clean, at `b0054a97e`. Its local `AGENTS.md` describes Gormes as Hermes-in-Go with Goncho as the Honcho-compatible Go port; routes Hermes parity work through the repo-local skills and canonical roadmap/progress representation. The repository has distinct implementation surfaces including kernel/runtime, provider/auth/model behavior, memory/Goncho, tools/security, skills/plugins/MCP/ACP, gateways/channels, CLI/TUI/API, and installation/packaging. Its parity and subsystem docs classify work across six lanes. This supports separate specialist remits rather than one undifferentiated generalist pool.

The user has now explicitly authorized development on `main` for this work, overriding the local branch restriction for the design step. No code changes, commits, branch operations, or release actions are included in this design.

A repository-specific workspace conflict remains: existing repo instructions forbid worktrees and short-lived branches, while safe parallel Kanban coding normally needs isolated worktrees and one writer per workspace. This team must not dispatch code-writing work until an owner-approved workspace approach is verified. Sequential ownership in a shared checkout is a possible alternative only if native task execution can safely enforce one writer and quiescence; this has not been established.

## 3. Profile topology and responsibilities

Create profiles as direct siblings under the actual native Hermes profiles root. Resolve that root from the installed Hermes implementation at provisioning time; do not hard-code an assumed home. The proposed IDs are below, subject to installed naming-limit and collision checks. No profile is to be cloned from an existing profile.

| Profile ID | Responsibility | Boundaries / evidence |
|---|---|---|
| `gormes-agent-team` | Human-facing coordinator; clarify requirements, decompose roadmap-backed work, assign owners, inspect evidence, surface blockers, and synthesize acceptance. | Not a worker on its own assigned task, not a reviewer of its own candidate, and not permission to alter the shared dispatcher or gateway. |
| `gormes-agent-team-kernel` | Agent runtime, loop, context assembly/compression, prompts, session behavior, and runtime-level parity. | Work only on explicitly scoped backlog rows and source-backed Hermes behavior. |
| `gormes-agent-team-providers` | Provider clients, authentication behavior, model selection/routing, streaming, usage, rate limits, and failover. | Credentials remain in supported secure stores; no tokens in cards, logs, or team records. |
| `gormes-agent-team-memory` | Memory, session indexing/search, and Goncho/Honcho compatibility. | Keep internal Gormes/Goncho implementation distinct from external Honcho-compatible contracts; use the canonical progress model. |
| `gormes-agent-team-tools` | Tool registry/execution, sandbox boundaries, permissions, security checks, and tool parity. | Sensitive or external effects need their own authorization; a task assignment does not grant them. |
| `gormes-agent-team-extensions` | Skills, plugins, MCP, ACP, and related extensibility surfaces. | Reuse repo-local skill-routing conventions; do not invent an alternate backlog or install dependencies without approval. |
| `gormes-agent-team-gateway` | Gateway lifecycle, delivery, messaging adapters, and channel parity. | No gateway start/restart, channel credential mutation, or outbound test is implied by profile creation. |
| `gormes-agent-team-cli` | CLI, TUI, API surfaces, installation, packaging, and service/runtime integration. | Treat Gormes-local validation and release instructions as authoritative for the affected surface. |
| `gormes-agent-team-parity` | Upstream research and source-backed parity evidence, plus relevant documentation work. | Findings cite exact upstream and Gormes paths/tests and feed the canonical backlog; research is not implementation approval. |

Use the same selected provider/model for all rows: `openai-codex / gpt-6-luna`; no per-profile overrides. This is the user's accepted selection, not yet provider-catalog, entitlement, authentication, or inference evidence.

The roster is broad to reflect separate code ownership. It is not a requirement to spawn every specialist for every task. Integration ownership is assigned per task from a suitable specialist; there is no permanent integrator profile. Independent validation is assigned to a profile that did not author the candidate. `parity` can supply evidence but cannot validate implementation it authored.

## 4. Root, board, workspaces, and human interface

- The coordinator is the sole human-facing root. Humans work sequentially through the root, not by concurrently conversing with specialists.
- CLI/TUI is the initial root interface proposal. Exact launch instructions depend on provisioning and capability verification.
- Telegram is desired only for the root. The previously supplied bot token is exposed and must be revoked. Any replacement must be entered only through a supported secure mechanism; it must not appear in chat, configuration committed to the repo, team briefing, Kanban metadata, or logs. Do not configure, start/rescan a gateway, send messages, or test handoff under this design.
- Resolve the actual Gormes Kanban board and database root explicitly; profile naming does not bind a board. Record the real dispatcher owner and configuration scope before changing any dispatcher setting. Do not modify shared auto-decomposition, concurrency, gateway, or notification settings as part of team creation without specific approval.
- Set and verify an absolute repository anchor and a supported task workspace kind. Do not assume board defaults create worktrees. Due to the repo's no-worktree rule, code-writing tasks remain gated until an owner-approved, technically verified safe workspace policy is available. Scratch workspaces are not a substitute for a candidate that must be integrated into Gormes.

## 5. Briefing, task lifecycle, acceptance, and recovery

After profile homes and team ownership are verified, prepare one nonsecret `TEAM.md` under the coordinator home, unless an owner-approved equivalent already exists. The coordinator's SOUL should point to that briefing and instruct root to read/reload it. The briefing is plain Markdown, not a Hermes registry or an automatic prompt feature; verify actual loading before using it for routing.

Every implementation task must identify: source-backed scope and acceptance criteria; one author; candidate path and base; required focused tests; an independent reviewer; an integration owner when combination is needed; and separate release authorization. Coding tasks must name exact worktree/candidate evidence if the repository workspace restriction is resolved. Validate and review the exact combined candidate, not just individual worker branches. Any candidate change after validation invalidates affected evidence.

Before any unattended dispatch, establish a finite, approved policy for total attempts, review/rework cycles, deadline, applicable inference/spend ceiling, escalation owner, stop action, and how each limit is enforced. Hermes retry counters alone do not prove a total attempt or cost bound. If the required enforcement or repository-safe workspace is unavailable, keep code work undispatched; do not add a custom scheduler/daemon or silently adjust shared settings.

## 6. Provisioning and readiness gates

Profile provisioning, if separately approved after this design review, should make only the scoped profile/config/instruction changes. Do not create cards or start specialists as probes.

For every profile, report separately:

- configured provider/model and effective overrides;
- authentication status in the actual profile launch context, redacted;
- inference status (not run unless specifically authorized); and
- release status (not applicable to the root; explicitly approved per worker task).

Before treating the root as ready, verify the actual installed schemas and effective capabilities for CLI/TUI and, only after secure credential setup and separate approval, Telegram. Check top-level tool availability gates, board selectors, loaded context/skills, task/review controls, notification delivery versus agent wake, and whether the interactive root has any contradictory worker-only prompt. An enabled option or SOUL instruction is not runtime evidence.

Verify provider-qualified catalog evidence for `openai-codex / gpt-6-luna` and profile-name/home resolution before configuration. Do not silently substitute another model/provider. Authentication and paid/live inference are not authorized by this design; live inference requires separate approval.

## 7. Explicit exclusions and unresolved decisions

- No profile, board, card, workspace, credential, allowlist, subscription, or gateway changes in this design stage.
- No inference, outbound Telegram message, canary, worker dispatch, commit, push, PR, or release.
- No copying of credentials, cloning profile state, or reading private memory/session/auth files.
- Board identity/database root, dispatcher owner, resolved profiles root, provider catalog evidence, auth readiness, root tool schemas, secure Telegram credential entry, and workspace-policy resolution remain unverified.
- Recovery limits must be agreed and enforceable before unattended task release; do not invent spending limits or assume prose alone enforces them.

## 8. Review and next stage

Owner review should confirm the scope, eight specialist remits, single coordinator, provider/model, CLI/TUI-first sequencing, Telegram security gate, and no-dispatch-until-workspace/recovery-readiness conditions. Requested changes should be incorporated into this spec and reviewed again. Only after explicit approval of the written spec should a separate implementation plan be prepared. Spec approval is not provisioning, credential, gateway, inference, dispatch, or release authorization.
