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
A lightweight, non-authoritative server readout that may include rough capability names for humans or legacy clients. It is not a Navivox Capability Gate and must not enable client feature affordances by itself.
_Avoid_: status capability gate, feature source of truth

**Navivox Profile Management**:
The server-advertised set of profile-related actions a Navivox client may offer. It includes profile contact reads and constrained profile seed creation, while bulk import, rename, delete, and dashboard-profile operations remain unavailable unless separately gated.
_Avoid_: dashboard profile management, generic import/manage profiles

**Profile Seed Creation**:
A constrained Navivox Profile Management action where an operator-supplied profile description becomes a draft before any apply step. It is creation-by-confirmed-draft, not bulk import or unrestricted profile mutation.
_Avoid_: profile import, automatic profile creation

## Example dialogue

Developer: "Should we add a TODO file for the new work?"
Domain expert: "No. Add or refine a Progress Row in the Logical Backlog. Generated views may list it, but they are not separate queues."

Developer: "Can we improve all progress tooling at once?"
Domain expert: "Use a Staged Deepening Program. First make row selection consistent, then hide storage layout, then improve generated views, then revisit projections."
