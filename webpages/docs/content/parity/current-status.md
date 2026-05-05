---
title: "Current Status"
description: "How to read status claims in Gormes docs."
weight: 10
---

# Current Status

Use status labels precisely:

| Label | Meaning |
|---|---|
| `runtime-ready` | Verified in the current Gormes runtime. |
| `fixture-backed` | Tested with local fixtures or fake clients. |
| `row-backed` | Tracked in progress/planning, but not promised as live behavior. |
| `planned` | Accepted direction without a shipped implementation. |
| `unverified` | Needs source or runtime evidence. |

This distinction matters because a progress row can be complete while the user-visible feature still needs wiring, live validation, or UX parity work.

## Gateway Channels

| Channel | Public status | Evidence |
|---|---|---|
| Telegram | `runtime-ready` | Registered by `gormes gateway` when configured; also available through `gormes telegram`. |
| Discord | `runtime-ready` | Registered by `gormes gateway` when token and discovery/allowlist config are present. |
| Slack | `runtime-ready` | Registered by `gormes gateway` when enabled with bot/app tokens; status reports missing-token degradation. |
| WhatsApp | `fixture-backed` / `row-backed` | Pairing and bridge/runtime contracts are tested, but live Baileys bundling and gateway registration are still tracked work. |
| WeChat / Weixin and long-tail adapters | `row-backed` / `planned` | Adapter contracts live in the roadmap until runtime registration and live validation are complete. |
