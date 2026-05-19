---
title: "Connect a provider and open chat"
description: "Add provider credentials and start provider-backed chat from the shell."
difficulty: "S"
aliases:
  - /recipes/first-turn/
---

> **Outcome:** Provider-backed chat opens from the shell, proving credentials and routing work.
>
> **Prerequisites:** `gormes` installed; an API key for one of: OpenAI, Anthropic, DeepSeek, Groq, OpenRouter, OpenAI Codex.

## Steps

1. **Add a provider credential**
   ```bash
   gormes auth add openai --api-key sk-...
   ```
   Replace `openai` with your provider id (`anthropic`, `deepseek`, `groq`, `openrouter`, `codex`, ...). The key is written to `~/.gormes/.env`, never echoed back.

2. **Confirm auth status**
   ```bash
   gormes auth list
   ```
   You should see a row with `auth_type=api_key` and `status=ok`.

3. **Start chat**
   ```bash
   gormes chat
   ```
   The default provider/model resolves from your config.

## Verify

```bash
gormes chat -q "say hi in three words"
```

Expected output: a short model reply on stdout, then the query exits with status 0.

## Troubleshooting

- **`Not Found: model 'xxx' not found`** → The configured model is not available on this provider. Run `gormes setup model`, or set the model with `gormes config set hermes.model <id>`.
- **`accepts 1 arg(s), received 0`** → `gormes auth add` needs the provider name. Re-run `gormes auth add <provider> --api-key ...`.
- **Auth `status=invalid` or `status=missing`** → See [provider setup](../../configure/providers/).

## See also

- [Run a local model with Ollama](../local-ollama/)
- [Add a fallback provider chain](../fallback-providers/)
- [Provider setup](../../configure/providers/)
