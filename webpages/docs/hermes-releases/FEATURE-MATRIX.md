# Hermes release feature matrix for Gormes planning

Source set: upstream Hermes release notes mirrored in this directory, `RELEASE_v0.2.0.md` through `RELEASE_v0.15.1.md`.

Use this matrix as a study aid, not as final parity proof. Release notes identify user-visible feature families and maturity signals; a Gormes parity row still needs exact upstream source refs from `./hermes-agent`, an existing atom in `docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md`, and a focused Gormes test target.

## How to use this matrix

1. Pick one improvement lane from the priority list below.
2. Search the release notes for the feature family and capture the release context.
3. Search `docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md` for matching atoms.
4. If `hermes-knowledge-graph.json` exists, use its layers/tour as a topology index to pick likely upstream files, then read those files directly.
5. Inspect the current upstream source under `./hermes-agent` before creating or refining any builder row.
6. If Gormes already has the behavior, reconcile the atom to `covered` or `partial` instead of duplicating work.

Useful commands:

```sh
rg -n "kanban|goal|handoff|session_search|promptware|Bitwarden|Responses" docs/hermes-releases docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md
rg -n "gateway|Telegram|Discord|Slack|ntfy|LINE|SimpleX|Teams|QQBot" docs/hermes-releases docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md
rg -n "MCP|ACP|browser|computer use|tool gateway|image_gen|web search" docs/hermes-releases docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md
node -e 'const g=require("./hermes-knowledge-graph.json"); for (const l of g.layers||[]) console.log(`${l.id}\t${l.name}\t${(l.nodeIds||[]).length}`)' 2>/dev/null || true
```

## Recommended improvement priorities

| Priority | Improvement lane | Why this lane matters | Release signal | Gormes planning action |
|---|---|---|---|---|
| P0 | Provider routing, credentials, and secrets | Hermes repeatedly adds provider paths, OAuth flows, credential pools, live model discovery, fallback chains, Codex/Responses maturity, and Bitwarden-backed secrets. This is the setup path operators notice first. | v0.2 provider router; v0.3 Anthropic/OAuth; v0.6 fallback chain; v0.7 credential pools; v0.8 live `/model`; v0.11 transport layer; v0.14 proxy/prompt cache; v0.15 Codex/Responses + Bitwarden. | Reconcile provider atoms, then build narrow tests for status/auth/model routing/fallback/secrets rather than broad provider umbrellas. |
| P0 | Gateway core reliability and channel-neutral rendering | Hermes treats messaging as a flagship surface, expanding from 7+ platforms to 23 while repeatedly hardening streaming, approvals, dedupe, restart/resume, policy gates, and plugin adapters. | v0.2 gateway foundation; v0.4 adapter wave; v0.6 Feishu/WeCom/Slack OAuth; v0.9 dashboard + WeChat/iMessage; v0.11 QQBot; v0.12 pluggable platforms; v0.13 restart session durability; v0.14 LINE/SimpleX/Microsoft Graph; v0.15 ntfy + gateway core. | Prioritize channel-neutral gateway primitives before long-tail adapter breadth. Use adapter work only when it proves the shared registry/rendering/session contract. |
| P0 | Security, promptware defense, file safety, and credential redaction | Later Hermes releases ship security waves as headline work, not polish: P0 closures, redaction defaults, path/TOCTOU fixes, promptware chokepoints, file write safety, and credential handling. | v0.3 PII redaction; v0.5 supply-chain hardening; v0.7 secret exfiltration blocking; v0.13 8 P0 closures; v0.14 advisory checker; v0.15 promptware/file/credential hardening; v0.15.1 redaction hotfix. | Keep security rows small and fixture-backed: path traversal, prompt injection, redaction boundaries, auth file permissions, unsafe write prevention. |
| P1 | Durable goals, checkpoints, session search, handoff, and memory/Goncho | Hermes' durable state story grows from sessions/memory into goals, checkpoints v2, live handoff, restart resume, and no-LLM session search. This is central to "Hermes in Go" and Goncho compatibility. | v0.2 checkpoints; v0.3 Honcho memory; v0.4 context refs/compression; v0.7 pluggable memory; v0.13 `/goal`, checkpoints v2, restart resume; v0.14 `/handoff`; v0.15 `session_search` rebuild. | Audit Goncho/session atoms, then split builder rows by one operator-visible command or tool: `/goal`, `/handoff`, checkpoint prune/rollback, session search. |
| P1 | Skills, bundles, curator, and self-improvement | Hermes release notes repeatedly frame skills as an ecosystem, culminating in background curator, self-improvement loops, bundles, and a huge skills catalog. | v0.2 skills hub; v0.5 lifecycle hooks; v0.8 plugin/session hooks; v0.11 skill system expansion; v0.12 curator/self-improvement; v0.14 Skills Hub/curator; v0.15 bundles/catalog; v0.15.1 catalog fix. | Separate deterministic skill loading/sync/bundles from autonomous curator behavior. Build the loader/catalog contract before curator loops. |
| P1 | Tool system, MCP/ACP, browser, and OpenAI-compatible API surface | Hermes makes tools a cross-product integration layer: MCP clients/server management, ACP IDE integration, Browser/CDP/Camofox/Browser Use, API server, approvals, and plugin-dispatched tools. | v0.2 MCP/browser/API foundations; v0.3 `/browser connect` + ACP; v0.4 API server/MCP CLI; v0.7 API continuity + Camofox; v0.8 MCP OAuth; v0.10 Tool Gateway; v0.11 plugin/tool expansion; v0.14 computer use/LSP; v0.15 MCP/browser/tool surface. | Favor protocol fixtures and descriptor parity over one-off tool implementations. Browser/CDP and MCP OAuth are high-leverage slices. |
| P1 | TUI, CLI, dashboard, and multi-session operator UX | Hermes moves from interactive CLI to Ink TUI and a local dashboard, then adds multi-session orchestration and dashboard auth fixes. This is the operator control plane. | v0.2 interactive CLI/skin/setup; v0.8 logs/config validation; v0.9 dashboard; v0.11 Ink TUI; v0.12 `-z`/update/onboarding; v0.14 PyPI/proxy/dashboard; v0.15 TUI session orchestrator; v0.15.1 dashboard hotfix. | Use current `ui-tui` upstream for TUI truth. Plan by operator tasks: switch sessions, inspect config/status, approve tools, launch gateway, view skills. |
| P2 | Kanban and multi-agent orchestration | Hermes' Kanban becomes a durable multi-agent platform across three releases. It is high-impact, but needs runtime/session/tool foundations first. | v0.13 durable multi-profile board; v0.14 Kanban maturation; v0.15 Swarm v1 graph, worker scheduling, dashboard, reliability; v0.15.1 worker kill fix. | Treat as a first-class subsystem only after session, checkpoint, provider, and tool execution contracts are stable. |
| P2 | Install, packaging, Windows, Docker, Nix, and cross-platform support | Hermes broadens distribution and platform support. Gormes has a different Go-native install surface, so use this as behavior inspiration rather than exact Python packaging parity. | v0.5 Nix; v0.6 Docker; v0.9 Termux; v0.14 native Windows beta + PyPI + supply-chain + Nix/Docker; v0.15 distribution/install; v0.15.1 Docker fixes. | Map to Gormes-owned install/update/doctor/onboard contracts. Do not copy Python packaging details unless they define user-visible behavior. |
| P2 | Voice, image, video, media, and external service skills | Useful breadth and demo value, but many features depend on provider/tool foundations. | v0.3 voice mode; v0.10 Tool Gateway TTS/image/search/browser; v0.12 Spotify/Google Meet; v0.13 video analyze + voice clone; v0.14 vision/video/computer use; v0.15 image providers. | Defer media breadth until provider secrets, tool descriptors, and gateway media rendering are solid. |

## Release-by-release study matrix

| Release | Product theme | Feature-bearing highlights | Gormes study signal |
|---|---|---|---|
| [v0.2.0](RELEASE_v0.2.0.md) | Foundational Hermes platform | Multi-platform gateway, MCP client, skills ecosystem, centralized provider router, ACP server, CLI skin engine, worktree isolation, filesystem checkpoints, broad tests. | Baseline contract: provider router, gateway, skills, tools, sessions/checkpoints, ACP. |
| [v0.3.0](RELEASE_v0.3.0.md) | Streaming, plugins, memory, approvals | Unified streaming, plugin architecture, native Anthropic/OAuth, smart approvals, Honcho memory, voice mode, concurrent tools, PII redaction, `/browser connect`, Vercel AI Gateway. | Prioritize streaming/tool execution/memory/plugin interfaces before long-tail features. |
| [v0.4.0](RELEASE_v0.4.0.md) | API server, adapters, context, compression | OpenAI-compatible API server, six new adapters, `@file`/`@url` context refs, new providers, MCP server management, gateway prompt caching, context compression, streaming default. | API server and context/compression are core parity surfaces, not optional docs features. |
| [v0.5.0](RELEASE_v0.5.0.md) | Provider breadth and supply-chain hardening | 400+ Nous models, Hugging Face provider, Telegram topics, Modal backend, plugin lifecycle hooks, GPT tool guidance, Nix flake, supply-chain hardening, Anthropic output limits. | Provider behavior needs model catalogs, setup, limits, and safety, not just request dispatch. |
| [v0.6.0](RELEASE_v0.6.0.md) | Profiles and fallback | Multi-instance profiles, MCP server mode, Docker, ordered fallback provider chain, Feishu/Lark, WeCom, Slack multi-workspace OAuth, Telegram webhook mode, Exa search, remote backend credentials. | Gormes profiles/provider fallback/gateway modes should be tested as user-visible setup flows. |
| [v0.7.0](RELEASE_v0.7.0.md) | Pluggable memory and gateway hardening | Memory provider interface, credential pools, Camofox backend, inline diffs, API session continuity/tool streaming, ACP client MCP servers, gateway hardening, secret exfiltration blocking. | High-leverage atoms: credential pools, memory provider, browser/CDP/Camofox, file diff display, API continuity. |
| [v0.8.0](RELEASE_v0.8.0.md) | Live operations and observability | Background process notifications, free-tier model gating, live `/model`, tool-use guidance benchmarks, Google AI Studio, activity-based timeouts, approval buttons, MCP OAuth/OSV scanning, logs/config validation, plugin expansion. | Operator controls and logs should become fixture-backed CLI/gateway tests. |
| [v0.9.0](RELEASE_v0.9.0.md) | Dashboard and platform support | Local dashboard, `/fast`, BlueBubbles iMessage, WeChat/WeCom callback, Termux/Android, watch patterns, xAI/Xiaomi providers, pluggable context engine, proxy support, security hardening. | Dashboard/context/proxy/security are durable platform work; adapters are breadth after shared primitives. |
| [v0.10.0](RELEASE_v0.10.0.md) | Nous Tool Gateway | Subscription-backed web search, image generation, TTS, and Browser Use through Nous Portal; per-tool opt-in, `hermes tools`, `hermes status`, gateway preference over direct keys. | Tool-provider routing and status visibility are one coherent Gormes provider/tool slice. |
| [v0.11.0](RELEASE_v0.11.0.md) | TUI rewrite and plugin/tool expansion | Ink TUI, transport layer, Bedrock and new inference paths, GPT-5.5 via Codex OAuth, QQBot, expanded plugins, `/steer`, shell hooks, webhook direct-delivery, smarter delegation. | Current TUI truth starts here; transport/plugins/delegation need source-backed atom reconciliation. |
| [v0.12.0](RELEASE_v0.12.0.md) | Curator, self-improvement, pluggable gateways | Autonomous curator, upgraded self-improvement fork, skill integrations, LM Studio and other providers, gateway plugin host, Teams/Yuanbao, Spotify, Google Meet, `hermes -z`, update preflight. | Split skills into deterministic sync/catalog versus autonomous curator; classify `-z` as Hermes-specific CLI behavior. |
| [v0.13.0](RELEASE_v0.13.0.md) | Durable multi-agent and goals | Kanban, `/goal`, video analyze, xAI voice clone, i18n, Google Chat, session restart resume, P0 security closures, checkpoints v2, write-time lint. | Goals/checkpoints/session durability/security are core. Kanban is major but should follow foundations. |
| [v0.14.0](RELEASE_v0.14.0.md) | Native Windows, performance, proxy, handoff | Windows beta, PyPI install, cold-start wins, faster browser console, advisory checker, OpenAI-compatible local proxy, cross-session Claude prompt cache, LINE/SimpleX, Microsoft Graph, `/handoff`. | Gormes should translate distribution work into Go install/doctor flows; `/handoff` and proxy are high-value parity candidates. |
| [v0.15.0](RELEASE_v0.15.0.md) | Runtime refactor, Kanban platform, security wave | `run_agent.py` refactor, Kanban Swarm, performance, no-LLM `session_search`, promptware defense, Bitwarden secrets, ntfy, skill bundles, TUI session orchestrator, image providers. | Current top-of-tree maturity signal: session search, promptware, secrets, TUI sessions, Kanban platform. |
| [v0.15.1](RELEASE_v0.15.1.md) | Hotfix and stabilization | Dashboard 401 loop, Docker `--insecure`, MCP command resolution in Docker, skills page/sidebar, Kanban worker kill, full skills catalog. | Study as regression-prevention evidence: auth loops, Docker safety, MCP resolution, skill catalog completeness, worker termination. |

## Subsystem evolution matrix

| Subsystem | First release signal | Maturity arc across releases | Improvement decision for Gormes |
|---|---|---|---|
| Agent runtime / kernel | v0.2 centralized provider router, worktrees, checkpoints | Streaming, concurrent tools, context refs, compression, goal persistence, handoff, session search, major runtime refactor. | Invest in Go kernel boundaries and durable session state before broad surface breadth. |
| Providers / models / credentials | v0.2 router | OAuth/native providers, fallback chain, credential pools, live model switching, transport ABC, proxy, Codex/Responses, Bitwarden. | Keep provider rows vertical: setup/status/auth/model-discovery/request/fallback/secrets. |
| Gateway / channels | v0.2 multi-platform gateway | Adapter waves, streaming and approval UX, plugin host, restart resume, Microsoft Graph, direct webhooks, ntfy. | Build channel-neutral contracts; use adapters to prove the contract rather than accumulating bespoke code. |
| CLI / TUI / dashboard | v0.2 interactive CLI | Logs/config validation, dashboard, Ink TUI, setup/update flows, dashboard auth, TUI session orchestrator. | Treat operator UX as parity-critical. Current TUI refs come from `ui-tui`, not legacy CLI only. |
| Tools / MCP / ACP / API | v0.2 MCP, browser, terminal, ACP | Browser CDP/Camofox/Browser Use, API server, MCP OAuth/server management, ACP, approvals, plugin-dispatched tools, computer use. | Protocol fixtures and descriptor parity should drive implementation order. |
| Skills / plugins / curator | v0.2 skills hub | Plugin hooks, install prompts, skill expansion, curator, self-improvement, skills hub, bundles/catalog. | Loader/catalog/bundle behavior first; autonomous curator after deterministic skill lifecycle is stable. |
| Sessions / memory / Goncho | v0.2 sessions/checkpoints | Honcho memory, pluggable memory providers, prompt cache, restart resume, handoff, no-LLM search. | Align Goncho with Honcho-compatible contracts and expose one tested operator command/tool at a time. |
| Kanban / multi-agent | v0.13 | Durable board, worker lifecycle, dashboard, Swarm graph, scheduling, reliability fixes. | High-value but foundation-dependent; plan after session/tool/provider primitives are reliable. |
| Security / reliability | v0.2 onward | PII redaction, supply-chain, secret exfil blocking, P0 closures, advisory checks, promptware defense, file/credential safety. | Maintain a standing security lane with fixtures; never defer security to broad refactor rows. |
| Install / distribution / platforms | v0.5 Nix, v0.6 Docker | Termux, Windows beta, PyPI, Docker, Nix, ACP/Zed packaging, update/install safety. | Translate to Go-native Gormes install/doctor/onboard/runtime checks; do not copy Python-specific packaging mechanics. |
| Media / voice / image / video | v0.3 voice | TTS/image/search gateway, Spotify/Meet, video analysis, voice cloning, vision/video, image providers. | Breadth lane after provider secrets, tool schemas, and gateway media rendering are mature. |
| Cron / background operations | v0.3 cron | API jobs, background notifications/watch patterns, curator ticker, scheduling, Kanban workers. | Tie cron work to observable status/logs and graceful cancellation before autonomous loops. |

## Next planner passes suggested by this matrix

1. **Provider/secrets pass:** reconcile atoms for Codex/Responses, fallback chains, credential pools, live model discovery, and Bitwarden-style secret sourcing.
2. **Gateway core pass:** map shared rendering/session/restart/direct-delivery behavior before adding any new adapter-specific rows.
3. **Security pass:** extract promptware, file safety, redaction, and credential-boundary atoms into fixture-backed builder rows.
4. **Skills pass:** separate skill loader/catalog/bundles from curator/self-improvement loops.
5. **Durability pass:** align `/goal`, `/handoff`, checkpoint v2, restart resume, and `session_search` with Goncho/session memory contracts.

## Guardrail

Do not mark any Gormes behavior `covered` from this matrix alone. This document names study targets. Parity status lives in `docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md`, backed by upstream source paths and Gormes tests.
