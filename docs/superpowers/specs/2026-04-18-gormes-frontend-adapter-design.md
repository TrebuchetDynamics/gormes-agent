# Gormes — Phase 1 Frontend Adapter Design Spec

**Date:** 2026-04-18
**Author:** Xel (via Codex brainstorm)
**Status:** Approved design direction; ready for planning
**Scope:** Phase 1 of the 5-phase Ship-of-Theseus migration: the Go dashboard, drop-in CLI facade, and Hugo docs mirror.
**Supersedes:** `2026-04-18-gormes-ignition-design.md`
**Supersedes:** `2026-04-18-gormes-ignition-deterministic-kernel-design.md` for Phase 1 transport and ownership boundaries

---

## 1. Purpose

Phase 1 is the first supplier swap in the move from Python Hermes to absolute Go purity.

The objective is not to rewrite the agent core yet. The objective is to ship a **drop-in Go frontend** that current Hermes users can adopt immediately:

- same command names;
- same flags;
- same documentation shape;
- new Go dashboard experience;
- Python still owns the brain underneath.

In truck terms, this is the GoCo dashboard plugged into the PythonCo chassis. The driver feels the upgrade immediately; the powertrain swap comes later.

---

## 2. Locked Decisions

These decisions are final for M0/M1.

### 2.1 Canonical Phase 1 seam: `tui_gateway` JSON-RPC over stdio

The Go dashboard will **not** use `mcp_serve.py` as its chat transport.

Recon showed:

- `mcp_serve.py` is a conversation and messaging bridge backed by polling and MCP tools;
- it is useful for external integrations;
- it is **not** the correct low-latency dashboard seam.

The correct Phase 1 seam is the existing Python TUI backend:

- backend process: `python -m tui_gateway.entry`
- transport: newline-delimited JSON-RPC over stdio
- existing events: `gateway.ready`, `message.start`, `message.delta`, `message.complete`, `thinking.delta`, `reasoning.delta`, `status.update`, `tool.start`, `tool.complete`, `approval.request`

### 2.2 Hugo is locked for docs

`gormes/docs` will become a Hugo site and will be deployed at `docs.gormes.io`.

### 2.3 Markdown must be Hugo Goldmark-safe

Phase 1 docs must validate against Hugo's Goldmark/CommonMark-compatible parser rules. MDX, JSX, and Docusaurus-only constructs are not allowed in Gormes docs.

### 2.4 Documentation IA must mirror Hermes

The information architecture of the Gormes docs site must mirror the Python Hermes docs structure so current Hermes users can find content exactly where they expect it.

### 2.5 CLI parity is a product requirement

The `gormes` binary must expose the same command tree and flag surface as the Python `hermes` CLI.

That does **not** mean every command is reimplemented in Go in Phase 1.

It **does** mean:

- a Hermes user can swap `hermes` for `gormes` without relearning command names;
- any command not yet native in Go must be proxied cleanly to Python Hermes;
- exit codes, stdio behavior, and argument parsing must remain predictable.

---

## 3. Product Boundary

### 3.1 What Go owns in Phase 1

Go owns:

- the `gormes` CLI entrypoint;
- command parsing and routing;
- the Bubble Tea dashboard;
- the JSON-RPC client for `tui_gateway`;
- local render state and UI buffering;
- Go-native docs scaffolding in Hugo;
- parity manifests and tests.

### 3.2 What Python owns in Phase 1

Python continues to own:

- `AIAgent` orchestration;
- prompt building;
- tool execution;
- context compression;
- reasoning behavior;
- session persistence via `SessionDB`;
- model/provider routing;
- approvals and existing gateway-side logic.

### 3.3 What is allowed to change in Python

The previous greenfield rule of "no Python files modified" is superseded for this Phase 1 strangler design.

Allowed Python changes are narrow and seam-oriented:

- `tui_gateway` protocol additions needed by the Go frontend;
- small compatibility hooks required for telemetry and session metadata;
- no broad rewrite of `run_agent.py`, memory systems, or provider logic.

The Python brain remains the supplier. We are only tightening the dashboard harness.

---

## 4. Phase 1 Runtime Architecture

### 4.1 Process model

Phase 1 runs as two local processes:

```text
+-------------------------------+        stdio JSON-RPC        +------------------------------+
| gormes (Go)                   | <-------------------------> | python -m tui_gateway.entry  |
|                               |                             |                              |
| - cobra CLI facade            |                             | - session management         |
| - Bubble Tea dashboard        |                             | - AIAgent                    |
| - event/render state          |                             | - SessionDB                  |
| - docs/CLI parity checks      |                             | - tool orchestration         |
+-------------------------------+                             +------------------------------+
```

The Go process launches the Python backend directly, using the same backend contract the existing Node/Ink TUI uses today.

### 4.2 Why stdio JSON-RPC wins

This seam is preferred because it already supports:

- live token streaming;
- per-turn lifecycle events;
- session list and resume;
- slash and command dispatch;
- approval prompts;
- reasoning and tool-progress events.

It is local, low-latency, and already production-shaped inside the repo.

### 4.3 MCP remains secondary

`mcp_serve.py` remains useful for:

- tool-style external clients;
- session and message inspection;
- conversation polling;
- approval workflows from third-party MCP consumers.

But it is explicitly **not** the Phase 1 dashboard transport.

---

## 5. JSON-RPC Contract for Gormes

### 5.1 Requests Gormes must support

Phase 1 Go client must support at least these existing methods:

- `session.list`
- `session.resume`
- `prompt.submit`
- `command.dispatch`
- `complete.slash`
- `approval.respond`

### 5.2 Events Gormes must consume

Phase 1 Go client must consume at least these existing events:

- `gateway.ready`
- `session.info`
- `message.start`
- `message.delta`
- `message.complete`
- `thinking.delta`
- `reasoning.delta`
- `reasoning.available`
- `status.update`
- `tool.start`
- `tool.progress`
- `tool.complete`
- `approval.request`
- `gateway.stderr`
- `gateway.protocol_error`
- `gateway.start_timeout`

### 5.3 Required protocol addition: quantitative telemetry

Recon found a real gap:

- streamed text exists today;
- qualitative phase/status events exist today;
- quantitative usage/telemetry is mostly surfaced only at turn completion.

Phase 1 requires a new backend event:

```json
{
  "type": "telemetry.update",
  "payload": {
    "model": "string",
    "input": 123,
    "output": 45,
    "total": 168,
    "calls": 1,
    "context_used": 2048,
    "context_max": 8192,
    "context_percent": 25
  }
}
```

Emission policy:

- once shortly after `message.start`;
- periodically during a streaming turn when counters change materially;
- once on `message.complete`.

This event should be sourced from existing `_get_usage(agent)`-style counters inside `tui_gateway`, not from a new agent-side telemetry subsystem.

This is the minimum seam extension required for a real Soul Monitor.

---

## 6. CLI Drop-In Contract

### 6.1 Source of truth

Phase 1 CLI parity is defined against two sources:

- `hermes_cli/main.py`
- `website/docs/reference/cli-commands.md`

If they disagree, `hermes_cli/main.py` wins.

### 6.2 Global options to preserve

The Go CLI must preserve these global flags from Hermes:

- `--version`, `-V`
- `--profile`, `-p`
- `--resume`, `-r`
- `--continue`, `-c`
- `--worktree`, `-w`
- `--yolo`
- `--pass-session-id`
- `--tui`
- `--dev`

### 6.3 Top-level command tree to preserve

The Go CLI must recognize the same top-level commands as current Hermes, including:

- `chat`
- `model`
- `gateway`
- `setup`
- `whatsapp`
- `auth`
- `login`
- `logout`
- `status`
- `cron`
- `webhook`
- `doctor`
- `dump`
- `debug`
- `backup`
- `import`
- `config`
- `pairing`
- `skills`
- `plugins`
- `memory`
- `tools`
- `mcp`
- `sessions`
- `insights`
- `claw`
- `version`
- `update`
- `uninstall`
- `acp`
- `profile`
- `completion`
- `dashboard`
- `logs`

Subcommands and aliases must also be mirrored.

### 6.4 Command routing model

Phase 1 uses a split command router:

1. **Go-native interactive dashboard path**
   - bare `gormes`
   - `gormes chat` in interactive TTY mode
   - `gormes --tui`
2. **Python pass-through path**
   - all non-native top-level commands
   - all unimplemented subcommands
   - non-interactive `chat` use cases that already work correctly in Python, such as one-shot query flows

Pass-through behavior:

- invoke Python Hermes with the original argv tail;
- preserve stdin/stdout/stderr passthrough;
- preserve exit codes;
- avoid reinterpreting Python-owned flags in Go after route selection.

### 6.5 CLI framework choice

Inference: `spf13/cobra` is the correct Go-side command framework for Phase 1 because it provides:

- a stable command tree;
- subcommand decomposition;
- flag inheritance;
- shell completion support;
- testable command registration.

The plan will assume Cobra unless a repo-local constraint blocks it.

---

## 7. Documentation Contract

### 7.1 Site root

`gormes/docs` is the Hugo site root.

Phase 1 Hugo scaffolding must create a standard Hugo layout there, including:

- `hugo.toml`
- `content/`
- `layouts/`
- `static/`
- `themes/` or Hugo module configuration

### 7.2 Deployment target

The site is intended to deploy as `docs.gormes.io`.

Phase 1 only needs:

- correct Hugo site structure;
- correct content tree;
- correct base URL and site configuration for docs hosting;
- local build success.

Actual deployment automation can remain M1.5 work if needed.

### 7.3 Theme direction

The theme must be minimalist and documentation-first.

Default Phase 1 choice: **Hugo Book**.

Reason:

- lower entropy for a mirrored docs tree;
- simpler mental model than a marketing-heavy theme;
- better fit for “current Hermes users need to find things fast.”

If implementation reveals a hard blocker, Doks is the fallback, but the plan should optimize for Hugo Book first.

### 7.4 Information architecture mirror

The Hugo content tree must mirror the current Hermes docs structure rooted at `website/docs`, including these primary sections:

- `getting-started`
- `user-guide`
- `user-guide/features`
- `user-guide/messaging`
- `user-guide/skills`
- `guides`
- `integrations`
- `reference`
- `developer-guide`

Phase 1 requirement is **path and navigation parity**, not complete prose parity for every page.

That means:

- users can find every expected section/path;
- missing Go-native content may use clearly labeled parity stubs;
- key pages for Phase 1 must be fully authored:
  - landing page
  - installation/getting-started
  - CLI commands reference
  - architecture overview
  - Go migration status / parity note

### 7.5 Goldmark validation

`docs/docs_test.go` must validate Hugo-compatible markdown.

The validation rule set must reject or flag:

- MDX/JSX fragments;
- Docusaurus admonition syntax;
- raw React-style components;
- unsupported front matter assumptions;
- root-relative link patterns that break on `docs.gormes.io`.

Because Context7 Hugo queries timed out during research, this Goldmark requirement is an inference from Hugo’s documented renderer architecture plus the repo’s current Docusaurus-style docs usage. The plan must keep the linter implementation small and explicit.

---

## 8. Session and Persistence Ownership

Phase 1 session persistence remains entirely Python-owned.

Verified behaviors already present:

- `tui_gateway` constructs `AIAgent` with `session_db=_get_db()`;
- `run_agent.py` persists messages into `SessionDB`;
- session resume loads conversation from Python storage.

Therefore:

- Go must not write to SQLite in Phase 1;
- Go may cache view state in memory only;
- session continuity flows through the backend session ID and JSON-RPC session methods.

This is non-negotiable until Phase 3.

---

## 9. Phase 1 UX Contract

### 9.1 User-visible behavior

A Hermes user launching `gormes` should experience:

- a faster, cinematic Bubble Tea dashboard;
- the same CLI muscle memory;
- the same session continuity;
- the same Python brain behavior underneath.

### 9.2 Soul Monitor contract

The Go dashboard sidebar may display:

- thinking/reasoning activity from streamed backend events;
- tool start/complete activity;
- quantitative telemetry from `telemetry.update`.

It must not invent fake backend state. If Python does not emit it, Go should display an explicit “backend does not provide this yet” fallback rather than hallucinating precision.

### 9.3 Failure mode contract

If the Python backend is unavailable:

- the Go CLI must fail clearly;
- error text must say how to start the backend;
- pass-through commands should still work if they do not require the dashboard backend process.

If the dashboard backend crashes:

- Go should surface the stderr/protocol error;
- the TUI must exit cleanly or offer restart behavior;
- no local user data is lost because Go owns no persistence in Phase 1.

---

## 10. Success Criteria

Phase 1 is complete only when all of the following hold:

1. `gormes` starts a Go-native dashboard that speaks to `python -m tui_gateway.entry` over stdio JSON-RPC.
2. `gormes chat` in interactive mode uses the same dashboard path.
3. Non-native CLI commands are accepted with the same command names and flags as Hermes and are proxied to Python Hermes with preserved exit codes.
4. A parity test verifies the Go command tree against the Python Hermes command tree source of truth.
5. `gormes/docs` builds as a Hugo site locally.
6. The Hugo content/navigation tree mirrors the primary section layout of `website/docs`.
7. `docs/docs_test.go` rejects markdown constructs incompatible with Hugo Goldmark.
8. The backend emits `telemetry.update` and the Go dashboard renders it.
9. Session history remains Python-owned end to end; Go writes no SQLite state.
10. A current Hermes user can discover Go docs at `docs.gormes.io` and find CLI/docs topics where they expect them.

---

## 11. Out of Scope

Still out of scope for Phase 1:

- rewriting the agent core in Go;
- moving SessionDB/SQLite into Go;
- replacing Python tools or skills;
- using MCP as the live dashboard transport;
- parity for internal slash-command implementation details beyond what the frontend needs;
- complete prose migration of every Hermes docs page.

---

## 12. Next Step

The next artifact is the implementation plan for this frontend-adapter phase.

That plan must cover:

- Cobra CLI facade and pass-through routing;
- Python backend launcher and JSON-RPC client;
- Bubble Tea dashboard;
- telemetry protocol extension in `tui_gateway`;
- Hugo site scaffolding;
- docs IA mirror and Goldmark lint;
- CLI parity tests.

This document is now the source of truth for Phase 1.
