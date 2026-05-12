---
title: "gormes gateway"
description: "Run Gormes as a multi-channel messaging gateway"
---

# gormes gateway

Runs every configured channel through one `gateway.Manager` that drives the
same kernel + tool loop as the TUI.

`gormes gateway status` and `gormes whatsapp` are runtime-ready and
row-backed: they accept invocation even when the underlying live service is
not running, and respond with deterministic structured output instead of an
unknown-command failure.

## Synopsis

```
gormes gateway [flags]
gormes gateway [command]
```

## Subcommands

| Command | Purpose |
|---|---|
| `gormes gateway discover` | Discover local Gormes gateways via Bonjour/mDNS |
| `gormes gateway install` | Manage gateway install through the platform service helper |
| `gormes gateway probe` | Probe gateway reachability, discovery, health, and runtime status |
| `gormes gateway reload` | Reload live Gormes gateway config without restarting |
| `gormes gateway restart` | Manage gateway restart through the platform service helper |
| `gormes gateway start` | Manage gateway start through the platform service helper |
| `gormes gateway status` | Inspect configured gateway channels and persisted runtime state |
| `gormes gateway stop` | Stop the live Gormes gateway recorded in runtime status |
| `gormes gateway uninstall` | Manage gateway uninstall through the platform service helper |
| `gormes gateway usage-cost` | Summarize token usage costs from gateway session metadata |

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-h`, `--help` | | help for gateway |

## See also

- [CLI reference](../)
- [`gormes channels`](../channels/)
- [`gormes logs`](../logs/)
- [`gormes telegram`](../telegram/)
- [`gormes whatsapp`](../whatsapp/)
- [Gateway operations](../../recipes/multi-channel/)
