# Gormes Planner Parity Map

Use this checklist to decide whether Gormes is still missing Hermes or Honcho behavior. The canonical feature-family map is `docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md`; the canonical reconciled implementation map is `docs/content/building-gormes/architecture_plan/hermes-honcho-go-runtime-plan.md`. This file is only a quick trigger list. Do not paste this whole list into progress rows; use it to drive focused passes.

Broad globs such as `agent/**`, `tools/**`, `src/**`, `sdks/**`, or `mcp/**`
do not prove feature coverage. A full-map pass must classify every subsystem as
one of: `owned`, `mapped-by-symbol`, `mapped-by-contract`, `excluded`,
`still row-backed`, or `unknown/gap`. Do not leave `unknown/gap` in final
planner output; convert it to an owned/excluded decision or a progress-backed
row with exact nested refs.

## Hermes-In-Go Surface

Map every Hermes feature group to Go code, docs, fixtures, and progress rows:

- CLI lifecycle: install, configure, auth/status, run, chat, resume, inspect, doctor, and a source-backed manifest for every top-level command, nested subcommand, root/global flag, slash command, dynamic plugin command, alias, and gateway handler before claiming command parity. Hermes-owned `-z/--oneshot` is removed-command guidance in Gormes; `gormes chat -q` is canonical.
- Session model: session IDs, transcript persistence, replay/resume, title generation, metadata, search.
- Provider routing: model selection, streaming, structured outputs, retries, degraded modes, provider-specific errors, token accounting.
- Provider transports: shared `ProviderTransport` interface and normalized response/event/tool-call/error contracts across Anthropic, Bedrock, Codex Responses, Chat Completions, Gemini native and Cloud Code, OpenRouter, Moonshot/Kimi sanitizer, and auxiliary clients.
- Provider account usage: Codex, Anthropic, and OpenRouter account-usage fetchers, quota-window normalization, shared renderer, separate CLI/status and gateway `/usage` command binding, running-agent/cached-agent/history fallback, and redacted degraded evidence.
- Credentials and OAuth: credential pool/sources, multi-account vault, Google/Codex/Copilot OAuth refresh, redaction at error and audit boundaries, token-vault interface separate from provider clients.
- Tool execution: typed tool registry, schema exposure, trust classes, approvals, shell execution, file edits, search, web/doc helpers, MCP-like extension points where applicable.
- Sandboxes and environments: local, Docker, Modal/managed Modal, Daytona, SSH, Singularity, file_sync, env scrubbing, credential mounts, env passthrough.
- Browser providers and web research: chromedp/rod core plus Browserbase, browser_use, and Firecrawl provider bridges with snapshot/dialog/CDP routing.
- Voice and media surfaces: TTS, voice mode, transcription, NeuTTS synth, image input/output routing, image generation registry.
- Security and policy surfaces: approval modes, path/URL/website policy, Tirith security, OSV checks, prompt-injection scan, prompt-visible redaction.
- Prompt/context assembly: system/developer/user layering, skills, project instructions, memory/context injection, compression, truncation, citations/evidence, context references, manual compression feedback, subdirectory hints.
- Subagents: child task spawning, work isolation, result integration, failure classification, concurrency limits, durable resilience.
- Gateway/API: local server, request/response contracts, streaming endpoints, auth, logs, health, OpenAI-compatible Responses/Runs, dashboard management routes.
- TUI/UI: interactive chat, tool visibility, approvals, progress, history, errors, skin engine, busy-input policy.
- Channels: Telegram, Discord forum, Slack, WhatsApp, WeChat (wecom + weixin), Signal, Matrix, Mattermost, Email, SMS, Feishu (+ feishu_comment), DingTalk, BlueBubbles, HomeAssistant, QQ, Yuanbao, and webhook-only adapters; long-tail adapters only when the core runtime contracts are present.
- Channel bootstrap registry: per-platform manifest, transport, identity/self-filter, inbound normalization, outbound delivery, sticker/mirror/directory cache.
- Cron/scheduling: durable jobs, triggers, retries, idempotence, status, context_from chaining.
- ACP and MCP catalogs: ACP auth/session/events/tools/permissions registry; MCP server/runtime, sampling, conversation bridge, prompt/resource wrappers, OAuth state machines, managed gateway clients.
- Webhooks: endpoint CRUD/test/signature, outbound webhook delivery, HMAC, retries/backoff/failure state, disabled endpoints, and queue-empty events.
- Dreaming and learning loop: scheduled dream execution, observations, peer-card side effects, skill candidate extraction, review/promotion.
- Plugins/skills: discovery, parsing, installation, invocation, lockfiles, provenance.
- Config/secrets: config files (incl. `cli-config.yaml.example` schema parity), env overrides, credential redaction, doctor checks, native `config edit/check/migrate`, explicit `gormes migrate hermes` / `gormes migrate openclaw` import commands, and typo-suggestion behavior for requests like `gormes migrate ooenclaw`.
- Observability: structured logs, audit ledger, telemetry, metrics, debug bundles, redaction-before-emit.
- Packaging/release: binaries, service units, install docs, version surfaces, OCI/Docker, Homebrew, Nix/flake/NixOS, public installer/site.

## Goncho As Honcho-In-Go

Goncho lives inside Gormes but must preserve Honcho-compatible public behavior where users or tools depend on it:

- External compatibility names may remain `honcho_*`; internal package/code names should stay `goncho`.
- Preserve Honcho concepts: workspaces, peers, peer cards, sessions, session peers (with `observe_me`/`observe_others`), messages, conclusions, observations (explicit/deductive/inductive/contradiction), representations, summaries, queue items, file uploads, keys, and webhooks.
- Honcho HTTP surface: every route in `docs/v3/openapi.json` and `src/routers/**` must be mapped to a Goncho `internal/goncho/http` (or `internal/apiserver`) handler or explicitly excluded with replacement evidence; preserve pagination cursors, status codes, and error envelopes.
- Honcho MCP catalog: every tool name under `mcp/src/tools/**` (`conclusions.ts`, `peers.ts`, `sessions.ts`, `system.ts`, `workspace.ts`) must be mapped to a Goncho descriptor or marked unsupported with a fixture.
- Honcho CLI compatibility: every command group under `honcho-cli/src/honcho_cli/commands/**` must be mapped to a `gormes goncho` subcommand or marked unsupported with a doctor diagnostic.
- Dreaming, deriver queue, reconciler, and dialectic chat are first-class subsystems, not "future work"; bind LLM-backed execution behind fake-provider fixtures.
- Preserve search, filter grammar, summaries, provenance, timestamps, and deletion/update semantics; reject unsupported filters visibly.
- Prefer SQLite/FTS/graph storage already present in Gormes unless a compatibility row proves another shape is needed; vector store is `owned` divergence with a testable contract, not a TODO.
- Add compatibility fixtures that compare expected Honcho request/response behavior without requiring a live Honcho service.
- Plan migration/export/import explicitly if existing Honcho data compatibility matters.
- Keep self-hosting docs separate from Goncho local: hosted FastAPI, Postgres, Redis, Fly, Alembic, and `database/init.sql` map to owned/excluded divergence rows, not implementation rows.

## Decision Rules

- If upstream behavior exists and Gormes lacks it, add/refine a row.
- If Gormes has a better Go-native equivalent, mark the subphase `owned` and explain the origin decision.
- If a feature needs live credentials or network access, plan a hermetic fixture first.
- If the row cannot be implemented by one builder in one bounded pass, split it.
