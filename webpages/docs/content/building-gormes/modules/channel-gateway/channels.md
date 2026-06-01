---
title: "Channels Module Roadmap"
aliases:
  - /building-gormes/modules/channels/
---

# Channels Module Roadmap

Generated from the single logical backlog. This page is a scoped review view; `progress.json` remains canonical.

**Module group:** Channel Gateway
**Module:** `channels`
**Rows:** 134
**Status counts:** `complete`: 134 · `in_progress`: 0 · `planned`: 0
**Priority counts:** `P0`: 7 · `P1`: 47 · `P2`: 27 · `P3`: 4 · `P4`: 2 · `unset`: 47

## Phase 2 — The Gateway

### 2.B.1 — Telegram Scout

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `channels` | Telegram adapter |
| `complete` | `unset` | `channels` | Long-poll ingress |
| `complete` | `unset` | `channels` | Edit coalescing |
| `complete` | `P1` | `channels` | Telegram important notification default |

### 2.B.2 — Gateway Chassis + Discord

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `channels` | Reusable gateway chassis |
| `complete` | `unset` | `channels` | Telegram on shared chassis |
| `complete` | `unset` | `channels` | gormes gateway multi-channel entrypoint |
| `complete` | `unset` | `channels` | Discord |

### 2.B.3 — Slack on Shared Chassis

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `channels` | Slack Socket Mode adapter |
| `complete` | `unset` | `channels` | Thread routing + coalesced reply flow |
| `complete` | `unset` | `channels` | Slack CommandRegistry parser wiring |
| `complete` | `unset` | `channels` | Slack gateway.Channel adapter shim |
| `complete` | `unset` | `channels` | Slack config + cmd/gormes gateway registration |
| `complete` | `P2` | `channels` | Slack env-token enabled-state preservation |
| `complete` | `P1` | `channels` | Slack app manifest App Home and private-channel scopes |

### 2.B.4 — WhatsApp Adapter

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `channels` | Bridge-vs-native runtime decision |
| `complete` | `unset` | `channels` | WhatsApp identity resolution + self-chat guard |
| `complete` | `unset` | `channels` | Inbound normalization + command passthrough |
| `complete` | `unset` | `channels` | Pairing, reconnect, and send contract |
| `complete` | `unset` | `channels` | WhatsApp outbound pairing gate + raw peer mapping |
| `complete` | `unset` | `channels` | WhatsApp reconnect backoff + send retry policy |

### 2.B.5 — Session Context + Delivery Routing

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `channels` | BlueBubbles iMessage session-context prompt guidance |
| `complete` | `P0` | `channels` | Telegram production live-turn provider payload golden |
| `complete` | `P0` | `channels` | Telegram /status Hermes-format closeout |
| `complete` | `P1` | `channels` | Telegram reply_to_mode and reply-context parity |
| `complete` | `P1` | `channels` | Telegram sendChatAction typing API |
| `complete` | `P2` | `channels` | Telegram dynamic BotCommand menu wiring |
| `complete` | `P2` | `channels` | WhatsApp identifier safety predicate |
| `complete` | `P2` | `channels` | WhatsApp unsafe sender/chat inbound evidence |
| `complete` | `P2` | `channels` | WhatsApp unsafe alias endpoint inbound evidence |
| `complete` | `P1` | `channels` | Telegram fresh-final delete and config exposure |
| `complete` | `P2` | `channels` | Telegram group bot-command mention gate helper |
| `complete` | `P3` | `channels` | Telegram require-mention config fields |
| `complete` | `P3` | `channels` | Telegram group require-mention bot binding |
| `complete` | `P1` | `channels` | Slack rich-text quotes/lists + link-unfurl ingress |
| `complete` | `P2` | `channels` | Slack thread-parent context + team-scoped cache key |
| `complete` | `P3` | `channels` | Email outbound Date header contract |
| `complete` | `P0` | `channels` | Telegram MarkdownV2 parse-mode rendering closeout |
| `complete` | `P1` | `channels` | Telegram topic mode off/help/auth/debounce closeout |
| `complete` | `P1` | `channels` | Telegram document/photo cache + batch attachment parity |
| `complete` | `P1` | `channels` | Discord authenticated attachment download safety |
| `complete` | `P1` | `channels` | Slack Block Kit approval buttons + action callback |
| `complete` | `P1` | `channels` | Discord thread participation persistence |
| `complete` | `P1` | `channels` | Telegram inline approval buttons + callback auth |
| `complete` | `P1` | `channels` | Telegram polling conflict + webhook secret startup guard |
| `complete` | `P0` | `channels` | Slack mention/free-response gating + strict thread-memory guard |
| `complete` | `P1` | `channels` | Discord interaction authorization + mention safety guards |
| `complete` | `P1` | `channels` | Gateway processing lifecycle reactions for Telegram and Discord |
| `complete` | `P1` | `channels` | Telegram text batching + caption merge parity |
| `complete` | `P0` | `channels` | Discord message admission + reply-mode policy |
| `complete` | `P1` | `channels` | Webhook dynamic route reload + signed rate-limit order |
| `complete` | `P1` | `channels` | Slack/Discord channel-scoped skills, prompts, and reload resync |
| `complete` | `P1` | `channels` | Telegram fallback transport + polling reconnect recovery |
| `complete` | `P1` | `channels` | Telegram sticker vision adapter binding |
| `complete` | `P2` | `channels` | Discord native slash/thread command registration parity |
| `complete` | `P1` | `channels` | Telegram entity-only mention boundary closeout |
| `complete` | `P2` | `channels` | Telegram thread-aware outbound text + typing seam |
| `complete` | `P2` | `channels` | Telegram forum thread fallback + send retry safety |
| `complete` | `P0` | `channels` | Telegram DM topic reply-fallback routing |
| `complete` | `P1` | `channels` | Telegram semantic MarkdownV2 formatter + table rewrite |
| `complete` | `P1` | `channels` | Telegram Markdown table row-label bullet suppression |
| `complete` | `P0` | `channels` | Telegram streaming edit Markdown safety |
| `complete` | `P1` | `channels` | Telegram guest mention allowlist bypass |

### 2.B.10 — WeChat Adapter

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `channels` | WeCom + WeiXin shared-chassis bot seam |
| `complete` | `unset` | `channels` | WeCom + WeiXin transport/bootstrap layer |

### 2.B.11 — Discord Forum Channels

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `channels` | Discord forum channel ingress + thread lifecycle |
| `complete` | `unset` | `channels` | Discord SessionSource guild/parent/message evidence |
| `complete` | `unset` | `channels` | Discord forum media + polish parity |

### 2.B.12 — Channel-Neutral Native Runtime Adapter

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `channels` | MSGraph webhook platform manifest drift closeout |
| `complete` | `P1` | `channels` | Hermes gateway platform strict-fidelity source-pair expansion |

### 2.H — Gormes-owned: Dynamic agents and per-thread spawn UX

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P2` | `channels` | Telegram /spawn opens forum topic bound to spawned agent |
| `complete` | `P2` | `channels` | Discord /spawn opens thread bound to spawned agent |

## Phase 5 — The Final Purge

### 5.A — Tool Surface Port

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `channels` | Discord tool split + platform-scoped toolsets |
| `complete` | `P2` | `channels` | Discord tool limit coercion helper |

### 5.G — MCP Integration

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `channels` | MCP channels_list tool |

### 5.J — Approval / Security Guards

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `channels` | Email allowlist pre-dispatch loop guard |

### 5.N — Misc Operator Tools

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `channels` | Teams configured-state in channel capabilities |
| `complete` | `P1` | `channels` | Per-agent channel bot tokens (Telegram/Discord/Slack) |

### 5.O — Hermes CLI Parity

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `channels` | Deterministic helper-file ports (banner/output/tips/webhook/dump) |
| `complete` | `unset` | `channels` | CLI webhook URL normalizer |
| `complete` | `unset` | `channels` | Hermes config.yaml Telegram compatibility bridge |
| `complete` | `unset` | `channels` | Gateway, platform, webhook, and cron management CLI |
| `complete` | `unset` | `channels` | WhatsApp top-level pairing wizard shell |
| `complete` | `P1` | `channels` | WhatsApp live Baileys QR pairing wizard |
| `complete` | `P1` | `channels` | cmd/gormes channels capabilities command package extraction |
| `complete` | `P1` | `channels` | cmd/gormes channel service command package extraction |
| `complete` | `P1` | `channels` | Per-profile channel credential readiness and allow-lists |

### 5.V — Unified Event Bus

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `channels` | Gateway channel adapters publish to event bus |
| `complete` | `P1` | `channels` | Weixin gateway event-bus adapter |
| `complete` | `P1` | `channels` | WeCom gateway event-bus adapter |
| `complete` | `P1` | `channels` | Telegram gateway event-bus adapter |
| `complete` | `P1` | `channels` | Discord gateway event-bus adapter |
| `complete` | `P1` | `channels` | Slack gateway event-bus adapter |
| `complete` | `P1` | `channels` | WhatsApp gateway event-bus adapter |

## Phase 6 — The Learning Loop (Soul)

### 6.F — Skill Surface

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P2` | `channels` | TUI + Telegram browsing |

## Phase 7 — Paused Channel Backlog

### 7.A — Signal Adapter

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `channels` | Inbound event normalization + session identity |
| `complete` | `unset` | `channels` | Reply/send contract on shared chassis |
| `complete` | `P1` | `channels` | Signal transport/bootstrap layer |
| `complete` | `P2` | `channels` | Signal markdown bodyRanges + attachment rate scheduler |

### 7.B — Email + SMS Adapters

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `channels` | Email ingress + outbound delivery contract |
| `complete` | `unset` | `channels` | SMS ingress + outbound delivery contract |

### 7.C — Matrix + Mattermost Adapters

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `channels` | Threaded text adapter contract suite |
| `complete` | `P2` | `channels` | Matrix shared-chassis bot seam |
| `complete` | `P2` | `channels` | Matrix self/bridge sender drop helper |
| `complete` | `P2` | `channels` | Mattermost shared-chassis bot seam |
| `complete` | `P1` | `channels` | Matrix real client/bootstrap layer |
| `complete` | `P2` | `channels` | Matrix E2EE device-id crypto-store binding |
| `complete` | `P1` | `channels` | Mattermost REST/WS bootstrap layer |

### 7.D — Webhook + Trigger Ingress

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `channels` | Signed event parsing + auth gates |
| `complete` | `unset` | `channels` | Prompt-to-delivery routing bridge |

### 7.E — Regional + Device Adapter Backlog

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `channels` | BlueBubbles + HomeAssistant adapters |
| `complete` | `P3` | `channels` | BlueBubbles iMessage bubble formatting parity |
| `complete` | `unset` | `channels` | Feishu shared-chassis bot seam |
| `complete` | `unset` | `channels` | DingTalk shared-chassis bot seam |
| `complete` | `unset` | `channels` | QQ Bot shared-chassis bot seam |
| `complete` | `unset` | `channels` | Feishu transport/bootstrap layer |
| `complete` | `P2` | `channels` | Feishu native update prompt cards |
| `complete` | `unset` | `channels` | Feishu drive-comment rule + pairing seam |
| `complete` | `unset` | `channels` | Feishu drive-comment reply workflow |
| `complete` | `unset` | `channels` | DingTalk transport/bootstrap layer |
| `complete` | `P2` | `channels` | DingTalk real SDK binding |
| `complete` | `unset` | `channels` | DingTalk AI Cards streaming-update contract |
| `complete` | `unset` | `channels` | DingTalk emoji reaction send/receive parity |
| `complete` | `unset` | `channels` | DingTalk media (image/file) attachment routing |
| `complete` | `P4` | `channels` | Yuanbao protocol envelope + markdown fixtures |
| `complete` | `P4` | `channels` | Yuanbao media/sticker attachment normalization |
| `complete` | `P2` | `channels` | Microsoft Teams adapter plugin seam |
| `complete` | `P2` | `channels` | QQ Bot transport/bootstrap layer |
| `complete` | `P2` | `channels` | Google Chat shared-chassis platform adapter seam |
| `complete` | `P1` | `channels` | Google Chat relay sender-type self-filter |
| `complete` | `P2` | `channels` | Google Chat standalone cron sender |
| `complete` | `P2` | `channels` | Google Chat install dependency hint refresh |
| `complete` | `P2` | `channels` | SimpleX Chat platform plugin parity |

## Phase 8 — Reputation & Publication

### 8.B — Repository Messaging

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `channels` | Channel capability matrix with stable/fixture/planned labels |
