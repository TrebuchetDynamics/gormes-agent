---
title: "Providers"
description: "The inference providers Gormes ships with, and how --provider and --model resolve."
weight: 33
aliases:
  - /reference/providers/
  - /guides/provider-setup/
  - /reference/web-backends/
---

# Providers

Gormes ships a curated provider registry. Each entry has a fallback default
model and may support OAuth, API-key, or both. The list below tracks the
binary's `providerDefaultModelFloor` map plus the OAuth-defaulting set; that
is what `gormes auth add <provider>` and `gormes config set hermes.provider
<name>` accept today.

| Provider | Default auth | Curated fallback model |
|---|---|---|
| `anthropic` | OAuth | `claude-opus-4-7` |
| `openai` | API-key | `gpt-5.4` |
| `openai-codex` | OAuth | resolves from `~/.codex/config.toml`, then the Codex cache, else `gpt-5.5` |
| `openrouter` | API-key | `moonshotai/kimi-k2.6` |
| `ai-gateway` | API-key | `moonshotai/kimi-k2.6` |
| `alibaba` / `alibaba-coding-plan` | API-key | `qwen3.6-plus` |
| `bedrock` | API-key | `us.anthropic.claude-sonnet-4-6` |
| `copilot` | API-key | `gpt-5.4` |
| `copilot-acp` | API-key | `copilot-acp` |
| `deepseek` | API-key | `deepseek-v4-pro` |
| `gemini` | API-key | `gemini-3.1-pro-preview` |
| `google-gemini-cli` | OAuth | `gemini-3.1-pro-preview` |
| `gmi` | API-key | `zai-org/GLM-5.1-FP8` |
| `huggingface` | API-key | `moonshotai/Kimi-K2.5` |
| `kilocode` | API-key | `anthropic/claude-opus-4.6` |
| `minimax-cn` | API-key | `MiniMax-M2.7` |
| `minimax-oauth` | OAuth | (resolved by provider) |
| `nous` | OAuth | `moonshotai/kimi-k2.6` |
| `nvidia` | API-key | `nvidia/nemotron-3-super-120b-a12b` |
| `opencode-go` | API-key | `kimi-k2.6` |
| `opencode-zen` | API-key | `kimi-k2.5` |
| `qwen-oauth` | OAuth | (resolved by provider) |
| `stepfun` | API-key | `step-3.5-flash` |
| `tencent-tokenhub` | API-key | `hy3-preview` |
| `xai` | API-key | `grok-4.20-0309-reasoning` |
| `xiaomi` | API-key | `mimo-v2.5-pro` |
| `zai` | API-key | `glm-5.1` |

Any provider name outside this set may be supplied to `gormes auth add` and
`hermes.provider`, but Gormes will not resolve a default model or base URL
for it — supply both explicitly.

## Configure a provider

The supported paths, in order of recommendation:

```bash
gormes setup provider                       # interactive provider wizard
gormes setup model                          # interactive model wizard
gormes model                                # standalone interactive model picker
gormes auth add openai --api-key sk-...     # store an API key in .env + auth.json
gormes auth add anthropic --type oauth      # OAuth flow with browser
gormes auth add openrouter --type api-key --api-key sk-or-...
gormes auth add openai-codex                # OAuth defaults for the Codex provider
```

Direct file edits work but skip the auth pool:

```bash
gormes config set hermes.provider openai
gormes config set hermes.endpoint https://api.openai.com/v1
gormes config set hermes.model gpt-4o
gormes config set hermes.api_key sk-...     # routed to .env automatically
```

Verify before relying on the configuration:

```bash
gormes config show
gormes doctor --offline
gormes chat -q "smoke test"
```

## `--provider` and `--model` coupling

The TUI and `gormes chat -q` resolve provider and model independently. The
binary enforces this rule: if you override the provider via flag or env, you
must also override the model. The reverse is allowed (overriding the model
alone keeps the configured provider and may trigger provider auto-detection).

Setting `GORMES_INFERENCE_PROVIDER` without `GORMES_INFERENCE_MODEL` returns:

```
gormes chat -q: GORMES_INFERENCE_PROVIDER requires --model or GORMES_INFERENCE_MODEL.
Set both inference env vars, pass both flags, or neither to use your configured defaults
```

Passing `--provider` without `--model` returns:

```
gormes chat -q: --provider requires --model (or GORMES_INFERENCE_MODEL).
Pass both explicitly, or neither to use your configured defaults.
```

Resolution precedence per axis:

1. `--provider` / `--model` CLI flag
2. `GORMES_INFERENCE_PROVIDER` / `GORMES_INFERENCE_MODEL`
3. `[hermes].provider` / `[hermes].model` in `config.toml`

When the model is left empty or set to the sentinel `"hermes-agent"`, Gormes
resolves the provider's curated default model from the registry above.

## Fallback chain

The fallback chain is configured per-agent and traversed by the runtime when
the primary provider fails. See the
[Fallback provider chain](../../recipes/fallback/) recipe for the
operator-facing workflow.

## Web backends

`web_search` and related web tools route through a separate backend selector:

| Knob | Default | Purpose |
|---|---|---|
| `[web].backend` / `GORMES_WEB_BACKEND` | `""` | Web search backend name (e.g. `tavily`, `serper`). Lowercased on read. |
| `[web].use_gateway` / `GORMES_WEB_USE_GATEWAY` | `false` | Route web requests through the Gormes gateway. |
| `[browser].cdp_url` / `GORMES_BROWSER_CDP_URL` | `""` | CDP endpoint for browser-backed fallback. Aliases: `BROWSER_CDP_URL`, `CHROME_REMOTE_DEBUGGING_URL`. |

Web behavior is capability-based: search returns candidates, extract reads
URLs, crawl is provider-specific, and browser tools handle dynamic pages.
Free-fallback search remains a search-only capability — it is not equivalent
to a full extract or crawl backend.
