---
title: "Add a fallback provider chain"
description: "Append backup providers so a primary outage doesn't kill turns."
difficulty: "S"
aliases:
  - /recipes/fallback/
---

> **Outcome:** When the primary provider fails (rate limit, outage, expired key), Gormes automatically tries the next provider/model in the configured chain.
>
> **Prerequisites:** At least two configured providers (see [first chat](../first-chat/)).

## Steps

1. **List the current chain**
   ```bash
   gormes fallback list
   ```
   On a fresh install:
   ```
   No fallback providers configured.

   Add one with:  gormes fallback add
   ```

2. **Append a fallback**
   ```bash
   gormes fallback add
   ```
   The interactive picker asks for the provider and model to append. Repeat the command to chain more entries; order is the order you add them.

   For a low-cost personal-agent chain, add API-key providers such as Google AI Studio, OpenRouter, and Groq first:
   ```bash
   gormes auth add google-ai-studio --type api-key --api-key AIza...
   gormes auth add openrouter-free --type api-key --api-key sk-or-...
   gormes auth add groq --type api-key --api-key gsk-...
   gormes fallback add
   ```
   Pick `gemini` with `gemini-2.5-flash` for generous free summarization/briefing fallback, `openrouter` with `deepseek/deepseek-chat-v3-0324:free` or another `:free` model for research/conversation variety, then `groq` with `llama-3.3-70b-versatile` for fast heartbeat or background-task fallback.

3. **Inspect the chain**
   ```bash
   gormes fallback list
   ```

4. **Remove one entry, or clear all**
   ```bash
   gormes fallback remove
   gormes fallback clear
   ```

## Verify

```bash
gormes fallback list
```

Expected: each fallback entry appears with provider and model. Then trigger a primary failure (revoke the primary key, point its endpoint at an unreachable host, or use an unavailable model id) and run:

```bash
gormes chat -q "test fallback"
```

The turn should still return a reply — served by the next entry in the chain.

## Troubleshooting

- **Fallback never activates** → The primary call must fail, not partially succeed. Streaming or partial errors may not promote to the next entry; check `gormes logs`.
- **`fallback add` opens an empty picker** → Add provider credentials first (`gormes auth add <provider>`).

## See also

- [Connect a provider and open chat](../first-chat/)
- [Provider setup](../../configure/providers/)
