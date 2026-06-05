---
title: "gormes whatsapp"
description: "Set up WhatsApp pairing through the Hermes-compatible Baileys bridge"
---

# gormes whatsapp

Sets up WhatsApp mode, allowlist state, bridge dependencies, and QR pairing
through the Hermes-compatible Baileys bridge.

## Synopsis

```
gormes whatsapp [flags]
```

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `--allow-all-users` | `false` | render allow-all sender configuration |
| `--allowed-users` | (none) | comma-separated allowed phone numbers with country code and no punctuation |
| `--bridge-script` | (default) | override the WhatsApp `bridge.js` path |
| `--debug` | `false` | render `WHATSAPP_DEBUG=true` in the dotenv plan |
| `-h`, `--help` | | help for whatsapp |
| `--json` | `false` | with `--plan`, emit the plan as machine-readable JSON |
| `--mode` | `bot` | WhatsApp mode: `bot` or `self-chat` |
| `--plan` | `false` | render the WhatsApp bridge plan without starting QR pairing |

## See also

- [CLI reference](../../)
- [`gormes gateway`](../../runtime/gateway/)
