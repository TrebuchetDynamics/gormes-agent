# Gormes Domain Language

This context names the domain concepts Gormes uses to coordinate progress work and operator-facing channel contracts without creating side backlogs or unsafe client assumptions.

## Language

### Progress Control Plane

**Progress Control Plane**:
The project-tracking context that coordinates Gormes planning, row selection, validation, and generated operator views through one authoritative backlog.
_Avoid_: TODO list, side backlog, issue queue

**Logical Backlog**:
The one authoritative set of progress rows, independent of how it is stored on disk or rendered for humans.
_Avoid_: monolith, split files, module queues

**Progress Row**:
One bounded Gormes work unit with enough contract evidence for a planner-to-builder handoff or enough shipped evidence to explain completed work.
_Avoid_: ticket, issue, TODO item

**Staged Deepening Program**:
An ordered sequence of architecture slices that deepens the Progress Control Plane while preserving the Logical Backlog.
_Avoid_: big-bang rewrite, parallel tracker

**Assignable Progress Row**:
A Progress Row that can be handed to a builder now because its contract evidence, test proof, write scope, and dependency state are sufficient.
_Avoid_: next TODO, selected issue

**Deferred Progress Row**:
A Progress Row that remains in the Logical Backlog but cannot be handed to a builder yet because it is blocked, too broad, missing proof, or intentionally parked.
_Avoid_: separate queue, hidden backlog

**Row Classification**:
The Progress Control Plane decision about a Progress Row's current handoff state, such as assignable, blocked, umbrella, missing-proof, complete, needs-human, or quarantined.
_Avoid_: status, label

**Blocking Dependency**:
A Progress Row or named condition that must be satisfied before another Progress Row can become an Assignable Progress Row.
_Avoid_: prerequisite ticket, external TODO

**Assignable Row Order**:
The deterministic order used by Progress Control Plane surfaces when presenting Assignable Progress Rows; `next-work` is the first row and generated next-work views are the first N rows.
_Avoid_: per-page priority, local sorting

**Progress Workspace**:
The Progress Control Plane module that hides where the Logical Backlog and generated progress artifacts live on disk, including monolith-vs-split layout details.
_Avoid_: path helper, filesystem wrapper

**Generated Artifact Plan**:
The ordered set of derived progress outputs that should be written from the Logical Backlog, including marker updates, module roadmap pages, and site mirrors.
_Avoid_: write script, docs side effects

**Progress Projection**:
A typed view over Progress Rows that exposes only the fields needed for one purpose, such as active handoff, shipped evidence, or row health, while preserving the Logical Backlog as the source.
_Avoid_: separate store, shadow backlog

### Gateway Runtime Contract

**Gateway CLI Orchestrator**:
The operator-facing gateway command surface that loads gateway configuration and secrets, validates startup safety, assembles enabled channel runtimes with session and memory services, and coordinates process lifecycle operations such as run, stop, restart, reload, and status. It is not a platform-specific message transport.
_Avoid_: Telegram adapter, channel implementation, generic gateway internals

**Channel Adapter**:
A platform-specific transport boundary that converts one messaging platform's inbound and outbound API into Gormes gateway events and replies. It owns platform quirks such as message formatting, attachment handling, batching, and API retries; it does not own gateway process lifecycle or cross-channel startup policy.
_Avoid_: gateway CLI, gateway process manager, runtime orchestrator

**Gateway Channel Registration**:
The Gateway CLI Orchestrator step that inspects configured channel credentials/accounts, invokes the relevant Channel Adapter factories, registers runnable channels with the gateway manager, and records degraded channel evidence for configured-but-unrunnable platforms. It is a startup assembly concern, not message handling or process lifecycle control.
_Avoid_: channel transport logic, gateway stop/restart logic, provider setup

### TUI Operator Experience

**Hermes TUI Contract**:
The user-visible terminal chat behavior Gormes treats as the compatibility baseline for interactive agent operation.
_Avoid_: Pi parity, arbitrary TUI redesign

**Pi-Inspired TUI Donor UX**:
Optional Gormes-owned terminal chat ergonomics adapted from Pi when they improve operator flow without weakening the Hermes TUI Contract.
_Avoid_: Pi parity, dual parity target, Pi compatibility contract

**TUI Input History**:
The composer recall surface for submitted operator inputs; it is distinct from transcript viewing and never means browsing assistant messages.
_Avoid_: transcript history, message history, shell history

**TUI Runtime Adapter Bundle**:
The local startup assembly for native TUI adapters that connect the chat TUI to runtime capabilities such as sessions, tools, skills, logs, model selection, account usage, voice, skin, and prompt templates. It preserves the TUI as a display/input module while concentrating runtime wiring outside the Bubble Tea model. It owns adapter wiring only; command startup, config loading, kernel construction, and Bubble Tea program creation remain outside the bundle. The first deepening slice should move one coherent adapter family rather than all TUI options at once; session adapters are the preferred first family, followed by model adapters and tool/skill adapters. Its first home is `internal/tuiadapter`, a command/runtime seam outside `internal/tui`, so the display module stays free of config, session database, and kernel construction knowledge. A bundle applies adapter families onto an existing `tui.Options` value so extraction can proceed incrementally. The session adapter family includes reset/resume functions provided by the already-built kernel because they are session-facing operator actions. Tests should first prove bundle application directly, then rely on existing startup characterization tests for end-to-end wiring.
_Avoid_: scattered TUI option wiring, config-aware TUI model, slash adapter sprawl, kernel-owning TUI bundle

### Navivox Channel Contract

**Navivox Capability Gate**:
The authoritative server advertisement that determines which Navivox client affordances may be enabled for an operator. Client probes may diagnose connectivity, but they must not enable profile management, attachment, voice, or stream features absent from the gate.
_Avoid_: endpoint probing, optimistic UI enablement, dashboard API discovery

**Closed Capability Mode**:
The safe Navivox state when a server is reachable but its Navivox Capability Gate is unavailable. Basic connection diagnostics may remain visible, but clients must not call feature endpoints or enable affordances that require advertised capabilities.
_Avoid_: fail-open mode, status-derived feature enablement, endpoint probing after gate failure

**Navivox Status Summary**:
A lightweight, non-authoritative server readout that may include rough capability names and current protocol names for humans or diagnostics. It is not a Navivox Capability Gate and must not enable client feature affordances by itself.
_Avoid_: status capability gate, feature source of truth, removed protocol advertisement

**Navivox Profile Management**:
The server-advertised set of profile-related actions a Navivox client may offer. It includes profile contact reads and constrained profile seed creation, while bulk import, rename, delete, and dashboard-profile operations remain unavailable unless separately gated.
_Avoid_: dashboard profile management, generic import/manage profiles

**Profile Seed Creation**:
A constrained Navivox Profile Management action where an operator-supplied profile description becomes a draft before any apply step. It is creation-by-confirmed-draft, not bulk import or unrestricted profile mutation.
_Avoid_: profile import, automatic profile creation

**Composed Profile Contract**:
The full Navivox profile shape assembled from profile contacts, profile routing, and voice-profile state. It is not a promise that every profile field appears in one JSON payload.
_Avoid_: single profile payload, contact-only contract

**Navivox Explicit Exclusion**:
A current Navivox contract statement that a behavior is unavailable without naming an endpoint that does not exist. Exclusions must not appear in endpoint lists and must not enable client affordances.
_Avoid_: unavailable feature as endpoint, hidden endpoint, optimistic API

**Navivox Unsupported Action**:
A structured, machine-readable action name that a Navivox client must not enable under the current capability document. Unsupported actions may explain disabled UI affordances, but they are not endpoint names or future promises.
_Avoid_: free-form note, fake endpoint, roadmap hint

**Navivox Setup Handoff**:
A pairing, connect, or status descriptor that helps an operator continue mobile setup in Navivox. It is not a feature capability and must not appear in capability lists.
_Avoid_: capability flag, setup feature endpoint

**Navivox Auth Mode**:
The exact active server-configured authentication requirement for a Navivox client, such as token-only, tailnet-identity-only, or layered token plus tailnet identity. It may name accepted credential headers or protocol slots with placeholders, but never token values, all possible server modes, secret references, environment variables, or private identity allowlists.
_Avoid_: broad auth bucket, token disclosure, secret source disclosure, possible-mode list

**Navivox Public Probe**:
A minimal unauthenticated liveness surface that proves the local bridge is reachable without revealing feature, profile, config, voice, or stream capabilities. Navivox capability and feature surfaces are authenticated.
_Avoid_: public capability discovery, unauthenticated status

**Navivox Capability**:
A feature affordance or API area that a Navivox client may use when advertised by the Navivox Capability Gate. It is separate from document identity, setup handoff, the stream message types used to render that feature, and live runtime counts.
_Avoid_: event kind, status summary item, runtime state, self-referential document flag

**Navivox Runtime State**:
Live, changing Navivox server information such as session counts, WebSocket connection counts, profile snapshots, and session records. Runtime state belongs in status or feature endpoints, not in the Navivox Capability Gate.
_Avoid_: capability state, contract metadata

**Navivox Event Kind**:
A concrete stream message type a Navivox client must parse and render. Event kinds describe transport messages, live under `streams.event_kinds`, and are not feature flags by themselves.
_Avoid_: capability, UI affordance, duplicate top-level event list

**Navivox Canonical Stream**:
The `/v1/navivox/stream` WebSocket stream used by Navivox clients for turns, contact updates, tool events, safety events, approvals, and completion. OpenAI-style run event streams are separate API-server surfaces unless the Navivox Capability Gate advertises a structured `openai_runs_bridge` capability.
_Avoid_: mixed run stream, OpenAI run events as Navivox transport, free-form bridge note

**Removed Navivox Surface**:
A previously supported Navivox protocol, endpoint, or contract shape that is no longer advertised or accepted after owner confirmation that no clients rely on it. Removal is explicit and scoped to Navivox surfaces, not a blanket cleanup of unrelated Gormes compatibility behavior.
_Avoid_: stale compatibility support, silent protocol removal, unrelated compatibility cleanup

**Navivox Capability Identity**:
The minimum fields that make a Navivox Capability Gate trustworthy: object identity, protocol version, auth contract, callable endpoints, and canonical stream. Missing identity fields make the document invalid; missing optional feature sections only disable that feature. `protocol_version` is the compatibility line until a breaking capability-document change requires a separate schema version.
_Avoid_: best-effort identity parsing, fail-open optional features, premature schema version

## Example dialogue

Developer: "Should we add a TODO file for the new work?"
Domain expert: "No. Add or refine a Progress Row in the Logical Backlog. Generated views may list it, but they are not separate queues."

Developer: "Can we improve all progress tooling at once?"
Domain expert: "Use a Staged Deepening Program. First make row selection consistent, then hide storage layout, then improve generated views, then revisit projections."

Developer: "Should the TUI match Pi now?"
Domain expert: "No. Preserve the Hermes TUI Contract, then adopt Pi-Inspired TUI Donor UX only where it improves Gormes without becoming a second parity target."

Developer: "Can Up arrow browse assistant replies?"
Domain expert: "No. Up and Down operate on TUI Input History; transcript viewing stays in the conversation viewport or slash surfaces."

### Home Layout & Profile Storage

**Main Profile**:
The built-in first Gormes profile. Its identifier is `main`, its runnable home is `$GORMES_BASE_HOME/profiles/main/`, and Gormes-owned profile commands should use `main` rather than treating `default` as a profile alias.
_Avoid_: default profile, root profile, base-home profile

**Profile Identity Default**:
The default profile identity used by Gormes-owned profile, setup, gateway, doctor, Navivox, chat, and profile-storage commands. It is always the Main Profile (`main`) and does not rename unrelated defaults such as skins, kanban boards, secret provider aliases, config fallback values, or upstream Hermes documentation examples.
_Avoid_: global default rename, default profile alias, unrelated default cleanup

**Reserved Profile Name**:
A string Gormes refuses as a user-created profile id because it would blur the Main Profile contract or a command namespace. `default` is reserved and must not be created as a normal profile.
_Avoid_: default alias, user profile named default, legacy profile recreation

**Profile‑Rooted Database Path**:
The convention for `memory.db` and `sessions.db` paths: profile runtime data lives under that profile's runnable home, including `profiles/main/memory.db` and `profiles/main/sessions.db` for the Main Profile.
_Avoid_: root DB paths, filesystem-probe-triggered profile DBs, automatic sharing through base home

**Profile Storage Contract**:
The typed path resolver (`ProfileStorageContract`) used by gateway channels and profile commands to compute profile-local paths for memory DBs, session DBs, workspace dirs, cache dirs, and runtime state under `$GORMES_BASE_HOME/profiles/<name>/`. Every runnable profile, including the Main Profile, uses this homogeneous layout. Callers remain responsible for directory existence.
_Avoid_: hardcoded path strings, per-command path resolution, root-home runtime state

**Profile Runtime Scope**:
The complete active-profile boundary for a Gormes operation: profile identity, runnable home, runtime environment, memory DB, sessions DB, workspace default, channel ownership, and setup persistence. A command should enter exactly one Profile Runtime Scope before reading or writing profile-owned state. Pure global metadata commands may stay outside a scope; commands that inspect multiple profiles resolve one explicit scope per inspected profile.
_Avoid_: scattered active-profile decisions, mixed profile roots, cross-profile runtime state

**Multi-Profile Gateway Process**:
One Gormes gateway OS process that can host multiple Profile Runtime Scopes at the same time. Profiles are isolation scopes inside the process, not one process per profile. The process keeps the base Gormes home as its global environment and resolves per-profile runtime scopes internally. A startup profile flag must not narrow gateway lifecycle commands; configured profile/channel bindings determine which profiles are hosted. Supplying a profile flag to a gateway lifecycle or gateway setup command is an operator error rather than a no-op; setup profile targeting belongs in setup/profile UI flows.
_Avoid_: per-profile gateway process, one PID per profile, cross-profile state sharing, profile-home environment for gateway process state, single-profile gateway flag

**Single-Profile Command Scope**:
A command execution mode such as chat or TUI that operates as one profile. It may apply one Profile Runtime Scope to process environment for the duration of that command. Profile/config metadata commands and gateway lifecycle commands stay outside this mode.
_Avoid_: multi-profile gateway mode, base-home-only runtime, unscoped profile state

**Profile Runtime Scope Resolver**:
The single Gormes service/API that derives a Profile Runtime Scope from operator input, sticky active profile state, and the Profile Storage Contract. It should be the only place that decides the runnable profile id and derived profile-owned paths for commands. It is pure path/identity resolution; command startup owns applying environment variables such as `GORMES_HOME`. It fails closed for invalid, reserved, or missing profile identities; setup and profile creation own materializing profile homes. A first-run startup path may explicitly ensure the Main Profile before resolution, but that creation is outside the resolver. Its first home is the existing CLI/profile boundary rather than a new package split. After the resolver-and-tests slice, the first callsite migration is process startup profile resolution because it mutates runtime environment for chat and gateway operations.
_Avoid_: command-local profile resolution, duplicated active-profile fallback, ad hoc path derivation, hidden environment mutation, implicit runtime directory creation

**Runtime Subdirectory**:
`$GORMES_HOME/runtime/` — the canonical subdirectory for gateway lifecycle state (`gateway_state.json`, `gateway.pid`, `gateway-locks/*`, `gateway.log`). Separates transient runtime state from durable config and data.
_Avoid_: root-level gateway files

**Lifecycle Subdirectory**:
`$GORMES_HOME/lifecycle/` — the canonical subdirectory for update/install lifecycle artifacts (`update.log`, `install.log.jsonl`, `backups/`). Excluded from pre-update backup to prevent geometric growth.
_Avoid_: root-level lifecycle logs

**Workspace Context Priority**:
The CWD workspace takes priority over `$GORMES_HOME` when resolving context files (`SOUL.md`, `AGENTS.md`, `IDENTITY.md`, `TOOLS.md`). The prompt loader checks workspace ancestors first, then falls back to the Gormes home directory. This ensures an explicitly configured workspace with seeded templates is the canonical context source, while default users (workspace = home) see no change.
_Avoid_: home-first priority, symlink-based context resolution, dual-source merge confusion

**Profile Workspace List**:
A profile may declare multiple workspaces: a profile-local default (`$GORMES_HOME/workspace/`) plus zero or more external repo or directory paths. The profile-local workspace is the implicit first entry and never needs explicit configuration. External workspaces are listed in the profile config and may reference repository checkouts or project dirs. The gateway seeds agent context templates into the profile-local workspace; external workspaces are file-access boundaries, not template targets.
_Avoid_: single-workspace assumption, workspace as config-only path, template seeding into external repos

**Profile Data Boundary**:
The set of filesystem paths that are scoped per-profile: `memory/`, `sessions/`, `workspace/`, `cache/`, `runtime/`. These directories live under the profile's runnable home (`$GORMES_BASE_HOME/profiles/<name>/`) and are never shared between profiles.
_Avoid_: root-level data dirs, cross-profile data sharing

**Credential Ownership Boundary**:
Provider API keys and generic secrets may be shared across profiles (owned by `"main"`), while channel bot tokens (Telegram token, Discord token, Slack token) are always per-profile (owned by the specific profile id). The runtime credential pool (`auth.json`) lives at `$GORMES_BASE_HOME` so all profiles read from the same store, and each stored entry carries an `owner_profile` field for filtering. The base `.env` holds all secrets with profile-scoped env var names (`GORMES_TULIN_TELEGRAM_BOT_TOKEN`). There is no per-profile `.env`.
_Avoid_: all-or-nothing credential scoping, per-profile auth.json, shared bot tokens, per-profile .env files
