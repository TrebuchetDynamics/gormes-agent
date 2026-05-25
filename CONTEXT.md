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

### Navivox Channel Contract

**Navivox Capability Gate**:
The authoritative server advertisement that determines which Navivox client affordances may be enabled for an operator. Client probes may diagnose connectivity, but they must not enable profile management, attachment, voice, or stream features absent from the gate.
_Avoid_: endpoint probing, optimistic UI enablement, dashboard API discovery

**Closed Capability Mode**:
The safe Navivox state when a server is reachable but its Navivox Capability Gate is unavailable. Basic connection diagnostics may remain visible, but feature affordances that require advertised capabilities remain disabled.
_Avoid_: fail-open mode, status-derived feature enablement

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

**Navivox Reserved Surface**:
A future Navivox API shape named only as unavailable product direction. Reserved surfaces must not appear in callable endpoint lists and must not enable client affordances.
_Avoid_: planned endpoint, hidden endpoint, optimistic API

**Navivox Auth Mode**:
The exact server-configured authentication requirement for a Navivox client, such as token-only, tailnet-identity-only, or layered token plus tailnet identity. It may name accepted credential headers or protocol slots with placeholders, but never token values, secret references, environment variables, or private identity allowlists.
_Avoid_: broad auth bucket, token disclosure, secret source disclosure

**Navivox Public Probe**:
A minimal unauthenticated liveness surface that proves the local bridge is reachable without revealing feature, profile, config, voice, or stream capabilities. Navivox capability and feature surfaces are authenticated.
_Avoid_: public capability discovery, unauthenticated status

**Navivox Capability**:
A feature affordance or API area that a Navivox client may use when advertised by the Navivox Capability Gate. It is separate from the stream message types used to render that feature and excludes live runtime counts.
_Avoid_: event kind, status summary item, runtime state

**Navivox Runtime State**:
Live, changing Navivox server information such as session counts, WebSocket connection counts, profile snapshots, and session records. Runtime state belongs in status or feature endpoints, not in the Navivox Capability Gate.
_Avoid_: capability state, contract metadata

**Navivox Event Kind**:
A concrete stream message type a Navivox client must parse and render. Event kinds describe transport messages; they are not feature flags by themselves.
_Avoid_: capability, UI affordance

**Navivox Canonical Stream**:
The `/v1/navivox/stream` WebSocket stream used by Navivox clients for turns, contact updates, tool events, safety events, approvals, and completion. OpenAI-style run event streams are separate API-server surfaces unless the Navivox Capability Gate advertises an explicit bridge.
_Avoid_: mixed run stream, OpenAI run events as Navivox transport

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
