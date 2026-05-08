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
| 5 / 5.M | Hermes Kanban specify triage parity | Hermes Kanban slash/gateway/dashboard surfaces | Hermes Kanban durable board core is complete., Hermes Kanban slash/gateway/dashboard surfaces row has landed the base /kanban command handler., The auxiliary LLM call can reuse the existing internal/provider router and fakeable provider seams. | - |
| 7 / 7.C | Matrix self/bridge sender drop helper | Matrix shared-chassis bot seam | Matrix shared-chassis bot seam exists so the helper can live under internal/channels/matrix without inventing a parallel gateway adapter., The slice is pure string classification plus table tests; no live Matrix sync, login, pairing store, E2EE store, room join, or gateway manager behavior is required., Tests should cover sender localpart parsing with Matrix IDs only; no mautrix/nio SDK dependency is needed. | Matrix real client/bootstrap layer |
| 7 / 7.C | Matrix real client/bootstrap layer | Matrix shared-chassis bot seam, Matrix self/bridge sender drop helper | Matrix shared-chassis bot seam is complete and exposes fakeable inbound/outbound hook points., Matrix self/bridge sender drop helper is complete or the bootstrap row wires only a placeholder sender-filter interface., Tests can drive a fake Matrix client with whoami/login/sync/send/upload methods; no Matrix SDK, homeserver, E2EE key store, or network is required. | Matrix E2EE device-id crypto-store binding, Matrix media upload/download contract, Matrix live transport binding |
| 7 / 7.C | Matrix E2EE device-id crypto-store binding | Matrix real client/bootstrap layer | Matrix real client/bootstrap layer has auth, sync/invite handling, room-kind policy, and a fakeable E2EE bootstrap seam. | - |
| 7 / 7.C | Mattermost REST/WS bootstrap layer | Mattermost shared-chassis bot seam | Mattermost shared-chassis bot seam is complete and exposes pure parser/reply-target behavior., Tests can drive fake REST and websocket clients with scripted responses; no Mattermost server, aiohttp equivalent, or network is required., Attachment evidence can reuse bounded artifact/path patterns instead of persisting arbitrary remote file names. | Mattermost live transport binding, Mattermost media upload contract |
| 7 / 7.E | QQ Bot transport/bootstrap layer | QQ Bot shared-chassis bot seam | QQ Bot shared-chassis bot seam is complete., Tests can inject fake dependency probes, token clients, websocket frames, REST send clients, fake clocks, and fake locks; no QQ credential, aiohttp/httpx equivalent, live websocket, or media upload is required., The row remains a bootstrap/runtime seam and does not expand the QQ tool catalog. | QQ Bot live transport smoke test, Voice attachment handling for Signal and QQ Bot, Paused adapter channel health/status readout |
| 7 / 7.E | Google Chat shared-chassis platform adapter seam | Microsoft Teams adapter plugin seam | 7.E `Microsoft Teams adapter plugin seam` is complete and provides a reference shape for shared-chassis adapter rows., Shared bot chassis exposes adapter-registration and event-normalizer seams reachable from tests., A fake Google Chat event payload fixture exists derived from upstream test fixtures. | Google Chat Workspace transport binding, Google Chat approval keyboard parity |
| 8 / 8.A | TD social presence connected to blog feed | TD engineering blog scaffolded and live | TD blog (8.A row 1) is live and emitting a feed., Operator has chosen a social platform and created the account. | - |
| 8 / 8.B | README rewrite to methodology-first positioning | Sharp v1.0 differentiator decision | Sharp v1.0 differentiator is decided (8.D row 'Sharp v1.0 differentiator decision')., Operator has reviewed success-plan.md and confirmed the North Star wording. | Engineering writeup #1: autonomous Hermes-porting loop, Landing page (gormes.ai) positioning audit |
| 8 / 8.B | gormes.ai landing page positioning audit | README rewrite to methodology-first positioning, Sharp v1.0 differentiator decision | README rewrite (8.B row 1) has landed on `development`., Sharp v1.0 differentiator (8.D) is decided. | - |
| 8 / 8.C | Engineering writeup #1: autonomous Hermes-porting loop | TD engineering blog scaffolded and live, Loop $/iteration cost metric in status file | TD blog (8.A row 1) is live., Loop $/iteration cost telemetry (8.F) has at least one week of data., Operator has decided the publication date and platform (HN/Lobsters/Reddit). | Engineering writeup #2: validation-gated agentic engineering, Engineering writeup #3: Gormes vs Hermes-Python benchmarks, HN launch post for Gormes 1.0 |
| 8 / 8.D | Single-binary cross-platform release pipeline | Sharp v1.0 differentiator decision | Sharp v1.0 differentiator (8.D row 1) is decided., go build ./cmd/gormes succeeds on all six target GOOS/GOARCH combinations from a Linux runner. | - |
<!-- PROGRESS:END -->
