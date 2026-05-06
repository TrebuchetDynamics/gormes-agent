---
title: "First Run"
description: "Run local diagnostics, configure your provider, and complete your first turn."
weight: 20
---

# First Run

## 1. Local Diagnostics

```bash
gormes doctor --offline
```

Verifies TUI, tools, gateway, Goncho memory, and web backend — no credentials needed.

## 2. Onboarding

```bash
gormes onboard
```

Shows what's configured and what's missing:
- Provider status (configured vs. needs setup)
- Agent list with default marker (★) and workspace paths
- Channel→agent bindings
- Skills count (local and bundled)
- Next steps tailored to your state

## 3. Provider Setup

The easiest path — interactive wizard:

```bash
gormes setup provider
gormes setup model
gormes model
```

Select a provider: OpenAI, Anthropic, DeepSeek, Codex, OpenCode, Groq, Ollama, or custom. Enter your API key. Done.

**API keys go to `~/.gormes/.env` (never in config.toml).**

For advanced users, direct config editing works too:

```bash
gormes config set hermes.endpoint https://api.openai.com/v1
gormes config set hermes.api_key sk-...        # → .env
gormes config set hermes.model gpt-4o
```

Or with OAuth (Codex, Anthropic):

```bash
gormes auth add openai-codex --type oauth
gormes auth add anthropic --type oauth
```

## 4. Test It

```bash
gormes --oneshot "hello from Gormes"
```

If successful, you'll see the model's response. If not:

```bash
gormes config show      # see current config (secrets redacted)
gormes config check     # validate schema
gormes doctor           # full diagnostics
```

## 5. Offline TUI

```bash
gormes --offline
```

Offline mode is a smoke test. Typed messages stay local — no network calls.

## 6. Gateway (Optional)

Once provider is configured, run a multi-channel agent:

```bash
gormes gateway
```

Or specific platforms:

```bash
gormes telegram
```

Keep channel secrets in `.env` or set them through the config helpers. Keep non-secret allowlist and routing fields in `config.toml`:

```dotenv
GORMES_TELEGRAM_TOKEN=123:abc
GORMES_TELEGRAM_CHAT_ID=42
```

Use `gormes gateway status` to confirm what is connected before treating a channel as runtime-ready.

## What's Next

```bash
gormes setup model     # pick your default model
gormes setup agent     # create additional agents
gormes setup bindings  # route channels to specific agents
gormes dashboard       # web UI at http://127.0.0.1:43827
```
