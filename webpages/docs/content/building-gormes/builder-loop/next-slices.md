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
| 8 / 8.D | Sharp v1.0 differentiator decision | A short decision document records the sharp v1.0 differentiator: a single paragraph stating what Gormes 1.0 will be (recommended: "runs the 30 most-used Hermes skills unchanged, in a single 30 MB Go binary, on Termux + Windows-without-Python + locked-down corp Linux"), the curated 30-skill list, the explicit exclusion list of what 1.0 will NOT do, and the date the decision was ratified. The decision unblocks every downstream messaging row. | operator | `docs/content/building-gormes/strategy/v1-differentiator.md` | P0 handoff; needs contract proof before closeout. |
| 5 / 5.J | Gateway allowed_chats/channels/rooms whitelist parity | Gormes ports Hermes' allowed_chats/allowed_channels/allowed_rooms whitelist config for gateway platforms: Telegram, Slack, Discord, Mattermost, Matrix, and DingTalk. Config is loaded from config.toml per-platform (e.g., telegram.allowed_chats, slack.allowed_channels) with env-var fallback as escape hatch. Inbound messages from non-whitelisted chat/channel/room IDs are silently dropped before reaching the kernel, with status/doctor reporting whitelist active count and skipped-message evidence. Mirrors Hermes 69d025e4a (Telegram/Mattermost/Matrix/DingTalk) and cd3ef685c (Slack), extending to Discord which already has the feature in both repos. | gateway, operator, system | `internal/gateway/whitelist_test.go` | Unblocks Telegram allowed_chats whitelist adapter, Slack allowed_channels whitelist adapter, Discord allowed_channels whitelist adapter, Mattermost allowed_channels whitelist adapter, Matrix allowed_rooms whitelist adapter, DingTalk allowed_chats whitelist adapter. |
| 8 / 8.A | TD engineering blog scaffolded and live | TrebuchetDynamics has a publicly reachable engineering blog with a working Atom/RSS feed, an /about page that names the org and the methodology, and a deploy pipeline so a markdown commit becomes a published post without manual intervention. Hosting choice is owner's call (Astro/Hugo/Eleventy + Cloudflare/Vercel/GitHub Pages); the row is done when a stranger can subscribe to a feed and read one published post. | operator | `webpages/blog/ (or chosen blog repo path)` | Unblocks Engineering writeup #1: autonomous Hermes-porting loop, Monthly digest pipeline. |
| 6 / 6.K | Behavioral pattern extraction from session logs | Mine session logs and tool execution audits for behavioral patterns: which tool sequences succeed vs fail, which reasoning patterns precede good outcomes, which response styles correlate with user satisfaction. Patterns feed into the self-evolution loop as candidate mutations. | operator | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 8 / 8.E | Agentic-porting-kit repo scaffold | The gormes-* skill set (gormes-planner, gormes-builder, gormes-tdd-slice, gormes-parity-auditor, gormes-references, gormes-skill-manager) is extracted into a separate public TrebuchetDynamics repo (`agentic-porting-kit` or equivalent), with a README that frames the kit as a generic Python→Go porting toolkit, a worked example using a small non-Hermes target, and a clear license. The kit must work standalone — its rows must be loadable by Codex or Claude Code in any repo, not just Gormes. | operator | `(separate repo: TrebuchetDynamics/agentic-porting-kit)` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 8 / 8.G | Built-with-Gormes page scaffold | A page at gormes.ai/built-with (or equivalent path on the docs site) lists real production deployments of Gormes, even if there is initially only one entry (the operator's own). The page has a documented submission process (PR-based) and a template entry shape. The point is to make the slot exist so it can be filled, not to fake usage. | operator | `webpages/landing/src/pages/built-with.astro (or equivalent)` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
