---
title: "Blocked Slices"
weight: 40
aliases:
  - /building-gormes/blocked-slices/
---

# Blocked Slices

This page is generated from canonical `progress.json` rows that declare
`blocked_by`.

Use it to avoid assigning work before the dependency chain is ready.

<!-- PROGRESS:START kind=blocked-slices -->
| Phase | Slice | Blocked by | Ready when | Unblocks |
|---|---|---|---|---|
| 2 / 2.H | gormes agent spawn/list/inspect/bind/unbind CLI | Goncho-backed dynamic agent registry | 2.H.1 ships a usable DynamicAgentRegistry interface and migration., The CLI can be tested entirely through executeOneshotFlagCommand with a tempdir GORMES_HOME; no live channel adapter is needed., Existing cmd/gormes/agent.go subcommand registration pattern admits new sibling commands without restructuring. | Telegram /spawn opens forum topic bound to spawned agent, Discord /spawn opens thread bound to spawned agent |
| 2 / 2.H | Telegram /spawn opens forum topic bound to spawned agent | Goncho-backed dynamic agent registry, gormes agent spawn/list/inspect/bind/unbind CLI | 2.H.1 and 2.H.2 are validated; the registry and CLI behavior are stable., Telegram createForumTopic and editForumTopic helpers exist in internal/channels/telegram or can be added behind the existing raw-call seam without touching unrelated send paths., Slash-command dispatch from 2.F.1 admits a new '/spawn' handler without restructuring the dispatcher. | Discord /spawn opens thread bound to spawned agent |
| 2 / 2.H | Discord /spawn opens thread bound to spawned agent | Telegram /spawn opens forum topic bound to spawned agent | 2.H.3 is validated and the slash-dispatch + registry-bind pattern is stable., internal/channels/discord/bot.go exposes a thread-create call (or one can be added behind the existing raw-discordgo seam) without changing unrelated participation policy. | - |
| 8 / 8.A | TD social presence connected to blog feed | TD engineering blog scaffolded and live | TD blog (8.A row 1) is live and emitting a feed., Operator has chosen a social platform and created the account. | - |
| 8 / 8.C | Engineering writeup #1: autonomous Hermes-porting loop | TD engineering blog scaffolded and live, Loop $/iteration cost metric in status file | TD blog (8.A row 1) is live., Loop $/iteration cost telemetry (8.F) has at least one week of data., Operator has decided the publication date and platform (HN/Lobsters/Reddit). | Engineering writeup #2: validation-gated agentic engineering, Engineering writeup #3: Gormes vs Hermes-Python benchmarks, HN launch post for Gormes 1.0 |
| 8 / 8.D | Single-binary cross-platform release pipeline | Sharp v1.0 differentiator decision | Sharp v1.0 differentiator (8.D row 1) is decided., go build ./cmd/gormes succeeds on all seven target GOOS/GOARCH combinations from CI or a Linux runner where cross-compilation is supported. | - |
<!-- PROGRESS:END -->
