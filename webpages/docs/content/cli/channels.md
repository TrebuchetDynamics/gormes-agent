---
title: "gormes channels"
description: "Inspect channel capability metadata"
---

# gormes channels

Inspect channel capability metadata and setup entry points. Use this command as the runtime-facing channel proof before treating a gateway adapter as ready.

## Capability Matrix

Status labels are evidence labels, not roadmap counts:

| Channel | Status | Setup/status path | Proof surface | Notes |
|---|---|---|---|---|
| Telegram | Runtime-ready | [Telegram](../../operate/telegram-bot/) | `gormes channels telegram`; `gormes channels setup telegram`; `gormes gateway status --json` | Shared gateway path with native commands, media, allowlists, and status output. |
| Discord | Runtime-ready | [Discord](../gateway/) | `gormes channels capabilities --channel discord`; `gormes gateway status --json` | Shared gateway path with configured bot token and channel/guild allowlists. |
| Slack | Runtime-ready | [Slack](../slack/) | `gormes channels capabilities --channel slack`; `gormes gateway status --json` | Socket Mode adapter with configured app/bot tokens and allowlisted channels. |
| WhatsApp | Fixture-backed | [WhatsApp](../whatsapp/) | `gormes channels capabilities --channel whatsapp` | Pairing and bridge/runtime rows exist; do not count as stable runtime-ready until promoted. |
| Teams / Yuanbao / long-tail adapters | Planned | `gormes channels capabilities` | capability report plus module roadmap | Planned or row-backed adapters stay separate from stable gateway proof. |

`gormes channels <channel>` and `gormes channels capabilities --channel <channel>` report implementation and support surfaces from the checked-in platform manifest. `gormes channels setup <channel>` prints the setup entry point for every manifest channel: runtime-ready channels get focused `gormes setup <channel>` / quick-target commands, while row-backed or planned channels get `gormes setup gateway --plan` and config-edit guidance. Channel credentials and allowlists are profile-scoped: setup records `profiles.<id>.channels.<channel>` plus a channel credential `SecretRef`, then writes runtime compatibility refs for the selected profile. `gormes gateway status --json` proves configured runtime state for the live gateway. Use both before claiming an adapter is ready.

## Synopsis

```
gormes channels [channel] [flags]
gormes channels [command]
```

## Subcommands

| Command | Purpose |
|---|---|
| `gormes channels capabilities` | Show channel capabilities |
| `gormes channels setup [channel]` | Show focused setup commands for one channel, or list setup entry points |

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-h`, `--help` | | help for channels |

## See also

- [CLI reference](../)
- [`gormes gateway`](../gateway/)
