---
title: "Run a multi-channel gateway"
description: "Run every configured channel through one Gormes gateway process."
difficulty: "L"
---

# Run a multi-channel gateway

> **Outcome:** One `gormes gateway` process drives Telegram, Discord, Slack, and any other configured channel through the same kernel + tool loop, with persistent runtime state visible via `gateway status`.
>
> **Prerequisites:** Working `gormes --oneshot` (see [first turn](../first-turn/)). At least one channel adapter configured with secrets and an allowlist. Runtime-ready channels today are Telegram, Discord, and Slack; WhatsApp/Teams/Yuanbao are row-backed or fixture-backed until promoted.

## Steps

1. **Configure each channel locally**
   - Telegram: see [Run a Telegram bot](../telegram-bot/).
   - Discord: set the bot token (`GORMES_DISCORD_TOKEN` in `.env`) and allowlisted guild/channel ids in `config.toml`.
   - Slack: set the app/bot tokens in `.env` and the configured channels in `config.toml`.

2. **Verify the config compiles**
   ```bash
   gormes config check
   gormes config show
   ```
   `config check` reports `_config_version`, missing/empty fields, and dotenv presence without writing.

3. **Start the gateway**
   ```bash
   gormes gateway
   ```
   The Gateway Manager binds every configured channel adapter; per-channel lifecycle errors are reported on startup.

4. **Inspect runtime state**
   ```bash
   gormes gateway status
   gormes gateway status --json
   ```
   For each channel, the status block lists `lifecycle`, `pairing`, and `target` allowlists. Runtime-ready channel readiness comes from this output — not from roadmap rows.

5. **Reload config without restarting**
   ```bash
   gormes gateway reload
   ```
   Reload covers swappable runtime settings: allowlists, first-run discovery flags, display/tool-progress modes, provider/model routing, skills root, and agent bindings. Use a full restart only for binary updates, database path changes, or channel transport changes that require reconnecting clients.

6. **Stop the gateway**
   ```bash
   gormes gateway stop
   ```

## Verify

```bash
gormes gateway status
```

Expected: each configured channel reports `lifecycle=running` and any allowlist or pairing state. Send a message through one channel; `gormes session list` shows a new session row.

## Troubleshooting

- **`runtime: stopped`** → No gateway is live. Start it with `gormes gateway`.
- **`stale_pid live=false`** → A previous gateway crashed; the persisted pid is dead. Run `gormes gateway stop` to clear runtime state.
- **`lifecycle=failed`** → The channel adapter failed to start. The `error` field names the cause (token conflict, missing allowlist, etc.).
- **Probing reachability from another host** → `gormes gateway probe --url host:port --json`.

## See also

- [Route channels to different agents](../bindings/)
- [Run a Telegram bot](../telegram-bot/)
- [Diagnose a broken install](../diagnose/)
