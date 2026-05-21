---
title: "Configure"
description: "Guided Gormes setup for providers, models, profiles, channels, secrets, and local state."
weight: 30
aliases:
  - /getting-started/configuration/
  - /using-gormes/configuration/
---

# Configure

Gormes resolves settings in a fixed precedence chain: CLI flags win, then
environment variables, then values in `config.toml`, then built-in defaults.
Secrets are routed to a separate dotenv file so `config.toml` stays safe to
share. Use this section for guided setup. Exact lookup pages for `config.toml`,
environment variables, and paths live under [Reference](../reference/).

Configuration is canonical at `$GORMES_HOME/config.toml` (default
`~/.gormes/config.toml`). Secrets live in `$GORMES_HOME/.env`. There is no
project-local override and no XDG path.

- [Setup wizard](./setup-wizard/) — `gormes setup`, focused setup sections,
  verification, and command reference links.
- [Providers](./providers/) — the inference providers Gormes ships
  with, the env vars they need, and how the `--provider`/`--model` flags
  resolve.
- [Models and routing](./models-routing/) — active provider/model selection,
  invocation overrides, and fallback pointers.
- [Profiles and workspaces](./profiles-workspaces/) — profile state boundaries
  and workspace allow-list semantics.
- [Channel credentials](./telegram/) — Telegram token setup plus gateway
  credential pointers.
- [Secrets and local state](./secrets-local-state/) — `config.toml` vs `.env`,
  `$GORMES_HOME`, and what not to share.

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

For exact lookup, use:

- [Config file](./config-file/)
- [Environment variables](./environment/)
- [Paths and logs](./paths/)
