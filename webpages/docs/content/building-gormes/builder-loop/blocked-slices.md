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
| 7 / 7.C | Matrix E2EE device-id crypto-store binding | Matrix real client/bootstrap layer | Matrix real client/bootstrap layer has auth, sync/invite handling, room-kind policy, and a fakeable E2EE bootstrap seam. | - |
| 7 / 7.E | QQ Bot transport/bootstrap layer | QQ Bot shared-chassis bot seam | QQ Bot shared-chassis bot seam is complete., Tests can inject fake dependency probes, token clients, websocket frames, REST send clients, fake clocks, and fake locks; no QQ credential, aiohttp/httpx equivalent, live websocket, or media upload is required., The row remains a bootstrap/runtime seam and does not expand the QQ tool catalog. | QQ Bot live transport smoke test, Voice attachment handling for Signal and QQ Bot, Paused adapter channel health/status readout |
| 8 / 8.A | TD social presence connected to blog feed | TD engineering blog scaffolded and live | TD blog (8.A row 1) is live and emitting a feed., Operator has chosen a social platform and created the account. | - |
| 8 / 8.B | README rewrite to methodology-first positioning | Sharp v1.0 differentiator decision | Sharp v1.0 differentiator is decided (8.D row 'Sharp v1.0 differentiator decision')., Operator has reviewed success-plan.md and confirmed the North Star wording. | Engineering writeup #1: autonomous Hermes-porting loop, Landing page (gormes.ai) positioning audit |
| 8 / 8.B | gormes.ai landing page positioning audit | README rewrite to methodology-first positioning, Sharp v1.0 differentiator decision | README rewrite (8.B row 1) has landed on `development`., Sharp v1.0 differentiator (8.D) is decided. | - |
| 8 / 8.C | Engineering writeup #1: autonomous Hermes-porting loop | TD engineering blog scaffolded and live, Loop $/iteration cost metric in status file | TD blog (8.A row 1) is live., Loop $/iteration cost telemetry (8.F) has at least one week of data., Operator has decided the publication date and platform (HN/Lobsters/Reddit). | Engineering writeup #2: validation-gated agentic engineering, Engineering writeup #3: Gormes vs Hermes-Python benchmarks, HN launch post for Gormes 1.0 |
| 8 / 8.D | Single-binary cross-platform release pipeline | Sharp v1.0 differentiator decision | Sharp v1.0 differentiator (8.D row 1) is decided., go build ./cmd/gormes succeeds on all seven target GOOS/GOARCH combinations from CI or a Linux runner where cross-compilation is supported. | - |
<!-- PROGRESS:END -->
