---
title: "Configure"
description: "Where Gormes reads its configuration and how to change it safely."
weight: 30
aliases:
  - /getting-started/configuration/
  - /using-gormes/configuration/
---

# Configure

Gormes resolves settings in a fixed precedence chain: CLI flags win, then
environment variables, then values in `config.toml`, then built-in defaults.
Secrets are routed to a separate dotenv file so `config.toml` stays safe to
share. Use the pages below to find the exact knob you need.

Configuration is canonical at `$GORMES_HOME/config.toml` (default
`~/.gormes/config.toml`). Secrets live in `$GORMES_HOME/.env`. There is no
project-local override and no XDG path.

- [Config file](./config-file/) — every `[section]` and field that
  `config.toml` accepts, with verified defaults from the binary.
- [Environment](./environment/) — every `GORMES_*` and legacy
  `HERMES_*`/`TELEGRAM_*` variable Gormes reads, grouped by family.
- [Providers](./providers/) — the inference providers Gormes ships
  with, the env vars they need, and how the `--provider`/`--model` flags
  resolve.
- [Telegram](./telegram/) — the minimal bot-token + allow-list path
  and the advanced channel/guest-mode/notification options.
- [Paths & logs](./paths/) — the exact files Gormes reads and writes
  under `$GORMES_HOME`, and how each is resolved.

## Quick orientation

Use the binary as the source of truth for paths and current values:

```bash
gormes config path          # prints $GORMES_HOME/config.toml
gormes config env-path      # prints $GORMES_HOME/.env
gormes config show          # resolved config with secrets redacted
gormes config check         # schema and dotenv presence
```

To change a value, prefer `gormes config set`. It writes secrets
(`*_API_KEY`, `*_TOKEN`, `api_key`) to `.env` and everything else to
`config.toml`:

```bash
gormes config set hermes.provider openai
gormes config set hermes.api_key sk-...   # routed to .env automatically
```

For provider credentials, use the credential pool so OAuth refresh works:

```bash
gormes auth add anthropic --type oauth
gormes auth add openai --api-key sk-...
```
