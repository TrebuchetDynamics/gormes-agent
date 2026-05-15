---
title: "Next Slices"
weight: 30
aliases:
  - /building-gormes/next-slices/
---

# Next Slices

This page is generated from the canonical progress file and lists the highest
leverage contract-bearing roadmap rows to execute next.

The ordering is:

1. unblocked `P0` handoffs;
2. active `in_progress` rows;
3. `fixture_ready` rows;
4. unblocked rows that unblock other slices;
5. remaining `draft` contract rows.

Use this page when choosing implementation work. If a row is too broad, split
the row in `progress.json` before assigning it.

If no slices are listed, the next correct action is planner work: choose one
planned row from `progress.json` or a phase page and add enough contract detail
for it to appear here. Do not infer that an empty generated list means the
roadmap is complete.

<!-- PROGRESS:START kind=next-slices -->
| Phase | Slice | Contract | Trust class | Fixture | Why now |
|---|---|---|---|---|---|
| 9 / 9.F | PicoClaw-derived tool path safety regression pack | Add a workspace path-safety regression pack that proves Gormes command/file tools keep relative paths relative to the active workspace, reject root enumeration such as `find /`, preserve virtual-path masking, and apply command guards to the parsed command body before execution. | operator, system | `internal/tools/picoclaw_path_safety_regression_test.go` | P0 handoff; needs contract proof before closeout. |
| 2 / 2.A | Coding-agent delegation: Phase 1 scaffold (internal/codingagents) | Shared internal/codingagents package providing the CodingAgent interface, CodingAgentRequest/Result, mode constants, binary availability detection, workspace guard with default deny list, git snapshot/diff helper, and prompt wrapper. No tools are registered in this slice; adapters and registry exposure land in later phases. | operator, system | `internal/codingagents` | Already active; contract metadata keeps execution bounded. |
| 9 / 9.F | PicoClaw-derived channel media and identity regression matrix | Add a fake-adapter regression matrix, sourced from current PicoClaw channel reports, that proves Gormes preserves sender identity, allowlist decisions, durable rich-media envelopes, voice transcript text, PDF/document attachment metadata, Feishu-style tool-progress notifications, and same-agent final delivery semantics through the channel-neutral gateway path. | operator, gateway, system | `internal/gateway/picoclaw_channel_regression_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 9 / 9.F | PicoClaw-derived session ledger read-model regression matrix | Add a session-ledger read-model regression matrix that proves Gormes stores and renders multiple user messages in a turn, per-message timestamps, sender attribution, durable attachment references, and non-destructive reset metadata without collapsing history to session.updated or deleting older channel history. | operator, gateway, system | `internal/session/picoclaw_ledger_regression_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 9 / 9.F | PicoClaw-derived provider stream and auth regression matrix | Add a provider regression matrix from PicoClaw reports that replays fake OpenRouter reasoning-model chunks, Codex Responses output_item.done events, 401/auth failures, local LM Studio/OpenAI-compatible model routing, and retryable LLM-call failures through Gormes provider seams with no live credentials. | operator, system | `internal/hermes/picoclaw_provider_regression_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 9 / 9.F | MCP Streamable HTTP session lifecycle compatibility | Extend the MCP HTTP client compatibility fixture to the current Streamable HTTP contract: initialize captures `Mcp-Session-Id`, all subsequent POST/GET/DELETE requests replay that header, SSE responses are accepted from the single MCP endpoint, 404 with a session header triggers a new initialization path, and legacy HTTP+SSE `/sse` endpoint events with `sessionId` are classified as backwards-compatibility input rather than silently dropping the session. | operator, system | `internal/tools/mcp_streamable_http_session_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 9 / 9.F | Dynamic agent identity inheritance regression matrix | Add a dynamic-agent identity regression matrix that proves spawned or delegated agents keep explicit parent linkage while receiving their own SOUL/persona, tool policy, AGENTS.md scope, and memory/search scope, so child agents do not silently inherit the root agent role as their own identity. | operator, child-agent, system | `internal/subagent/picoclaw_identity_regression_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.Q | Native TUI /model slash command binding over the existing model picker | The native Bubble Tea TUI treats `/model` (and the `/m` prefix) as a local operator command, not prompt text: dispatching it opens the already-implemented ModelPicker overlay (internal/tui/model_picker.go RenderModelPicker/UpdateModelPicker — a TUI-LOCAL overlay, unlike the kernel-driven Approval/Clarify/Secret panels, so it needs its own Model overlay state + update.go key routing + view.go render slot), clears the editor, never calls Submitter; confirming applies an IN-SESSION model switch; cancel returns unchanged. BLOCKED: builder-pass 2026-05-15 established there is NO in-session model-switch seam in the local kernel path — PlatformEventKind is {Submit,Cancel,Quit,ResetSession,Steer} with no model override; kernel.go SetModel is construction-only; the completed 5.O picker is config-TOML-persist only; SessionModelOverride is gateway-server-only and not wired to the local Bubble Tea kernel. This row therefore depends on the new 'Kernel in-session model-switch seam for the native TUI' prerequisite. The picker render/key engine already exists and MUST be reused, not reimplemented; the missing piece is the apply seam plus a model-catalog -> internal/tui data seam. | - | `internal/tui/slash_model_test.go; cmd/gormes/tui_model_slash_test.go` | Unblocks Native TUI slash handler-port coverage. |
| 1 / 5.X | Termux storage and path safety audit | Audit and test Gormes path selection under synthetic Termux env so config, dotenv secrets, sessions, gateway state, SQLite/Goncho, browser temp dirs, and generated files land only under configured GORMES_HOME/XDG/HOME locations while install publication remains $PREFIX/bin/gormes. No runtime code may hardcode desktop workspace paths such as /home/xel or workspace-mineru. | operator, system | `internal/config Termux path fixtures plus cmd/gormes doctor/config/goncho smoke fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 1 / 5.X | Termux gateway foreground tmux lifecycle | Gateway lifecycle commands and docs present a Termux-specific foreground/tmux model: Telegram/Discord/Slack gateways are supported from a foreground shell or tmux session, systemd/Windows service assumptions are not advertised, and doctor/status guidance names termux-wake-lock plus Android battery settings as best-effort survival aids. The implementation must preserve the same gateway command names and JSON contracts as desktop Linux. | operator, gateway, system | `cmd/gormes gateway/doctor fixtures under synthetic Termux env` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
