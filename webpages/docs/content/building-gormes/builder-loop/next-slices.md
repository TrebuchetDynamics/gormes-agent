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
| 9 / 9.C | Transcribe audio tool registration + local whisper provider | Gormes registers the existing transcribe_audio tool in the default tool registry so STT works by default with no API key. A LocalSTTProvider wraps the WASI whisper runtime (internal/wasi/whisper/) into the TranscriptionProvider interface with auto-downloading tiny.en model (~77MB from HuggingFace). Cloud STT providers (OpenAI, Groq, Mistral, XAI) are registered alongside and activate when their API keys are present. | operator, system | `internal/tools/transcription_providers_local.go` | Already active; contract metadata keeps execution bounded. |
| 8 / 8.A | TD engineering blog scaffolded and live | TrebuchetDynamics has a publicly reachable engineering blog with a working Atom/RSS feed, an /about page that names the org and the methodology, and a deploy pipeline so a markdown commit becomes a published post without manual intervention. Hosting choice is owner's call (Astro/Hugo/Eleventy + Cloudflare/Vercel/GitHub Pages); the row is done when a stranger can subscribe to a feed and read one published post. | operator | `webpages/blog/ (or chosen blog repo path)` | Unblocks Engineering writeup #1: autonomous Hermes-porting loop, Monthly digest pipeline. |
| 5 / 5.J | Tirith external security finding ingestion | Port the Hermes Tirith external security finding ingestion path: load findings from a JSON file or env-sourced source, classify by severity, and expose a security guard decision that gateway/cron/CLI callers can query before executing dangerous commands. | operator, system | `internal/security/tirith_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.J | Unified security guard decision composer | Compose Tirith findings, path-based allowlists, URL safety rules, and website policies into one security guard decision that gateway/cron/CLI can call before executing any tool. The composer resolves conflicts deterministically (deny wins over allow, policy overrides Tirith) and always returns typed evidence explaining the decision. | operator, system | `internal/security/guard_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.M | Kanban multi-board isolation | Kanban dispatcher enforces board-scoped isolation: workers spawned for board A cannot see or mutate board B's tasks. The SQLite store uses per-board database files or namespaced tables, and the dispatcher validates the board name before spawning. | operator, system | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.M | Kanban workspace context injection | Kanban worker spawning injects the board's workspace directory as the worker's working directory and loads the workspace's AGENTS.md/CLAUDE.md context, mirroring Hermes workspace-path isolation. | operator, system | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.M | Kanban run history persistence | Kanban run history records spawn attempts, successes, failures, and completion evidence per task so operators and the dispatcher can inspect past runs and detect spin-loop failures. | operator, system | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.M | Kanban notification delivery parity | Kanban worker completion triggers notification delivery to the board owner's configured channel (Telegram/Discord/Slack) with task summary and run evidence. | operator, system | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.P | Installer script serving and MIME validation | Wire install.sh, install.ps1, and install.cmd into the www.gormes.ai Go server with correct Content-Type headers (text/x-shellscript, text/plain, text/plain), cache-control, and static export. Tests verifies each script is embedded, served with the correct MIME, and static-exported. | operator, system | `www.gormes.ai/internal/site/assets_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 7 / 7.E | DingTalk real SDK binding | Bind the Gormes DingTalk channel to a real DingTalk Stream Mode SDK (replacing the current stub/fake). Implement credential loading (AppKey/AppSecret from config.toml), receive loop via the SDK's callback, send lifecycle, and reconnection with the existing retry seam. | operator, system | `internal/channels/dingtalk/client_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
