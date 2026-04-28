# Gormes Planner Parity Map

Use this checklist to decide whether Gormes is still missing Hermes or Honcho behavior. The canonical full map is `docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md`; this file is only a quick trigger list. Do not paste this whole list into progress rows; use it to drive focused passes.

## Hermes-In-Go Surface

Map every Hermes feature group to Go code, docs, fixtures, and progress rows:

- CLI lifecycle: install, configure, auth/status, run, chat, resume, inspect, doctor.
- Session model: session IDs, transcript persistence, replay/resume, title generation, metadata, search.
- Provider routing: model selection, streaming, structured outputs, retries, degraded modes, provider-specific errors, token accounting.
- Tool execution: typed tool registry, schema exposure, trust classes, approvals, shell execution, file edits, search, web/doc helpers, MCP-like extension points where applicable.
- Prompt/context assembly: system/developer/user layering, skills, project instructions, memory/context injection, compression, truncation, citations/evidence.
- Subagents: child task spawning, work isolation, result integration, failure classification, concurrency limits.
- Gateway/API: local server, request/response contracts, streaming endpoints, auth, logs, health.
- TUI/UI: interactive chat, tool visibility, approvals, progress, history, errors.
- Channels: Telegram, Discord, Slack, and long-tail adapters only when the core runtime contracts are present.
- Cron/scheduling: durable jobs, triggers, retries, idempotence, status.
- Plugins/skills: discovery, parsing, installation, invocation, lockfiles, provenance.
- Config/secrets: config files, env overrides, credential redaction, doctor checks.
- Observability: structured logs, audit ledger, telemetry, metrics, debug bundles.
- Packaging/release: binaries, service units, install docs, version surfaces.

## Goncho As Honcho-In-Go

Goncho lives inside Gormes but must preserve Honcho-compatible public behavior where users or tools depend on it:

- External compatibility names may remain `honcho_*`; internal package/code names should stay `goncho`.
- Preserve Honcho concepts: sessions, messages, users/workspaces where applicable, memories/facts, search, provenance, timestamps, and deletion/update semantics.
- Prefer SQLite/FTS/graph storage already present in Gormes unless a compatibility row proves another shape is needed.
- Add compatibility fixtures that compare expected Honcho request/response behavior without requiring a live Honcho service.
- Plan migration/export/import explicitly if existing Honcho data compatibility matters.

## Decision Rules

- If upstream behavior exists and Gormes lacks it, add/refine a row.
- If Gormes has a better Go-native equivalent, mark the subphase `owned` and explain the origin decision.
- If a feature needs live credentials or network access, plan a hermetic fixture first.
- If the row cannot be implemented by one builder in one bounded pass, split it.
