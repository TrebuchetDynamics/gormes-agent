---
title: "Connect a provider and run a one-shot"
description: "Add provider credentials and run one provider-backed turn from the shell."
difficulty: "S"
---

# Connect a provider and run a one-shot

> **Outcome:** One provider-backed model turn completes from the shell, proving credentials and routing work.
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

3. **Run a one-shot turn**
   ```bash
   gormes --oneshot "hello"
   ```
   The default provider/model resolves from your config. Override per-invocation with `--provider`, `--model`, and `--endpoint`.

## Verify

```bash
gormes --oneshot "say hi in three words"
```

Expected output: a short model reply on stdout, then the process exits with status 0.

## Troubleshooting

- **`Not Found: model 'xxx' not found`** → The configured model is not available on this provider. Run `gormes setup model` or pass `--model <id>`.
- **`missing required <provider> argument`** → Re-run `gormes auth add <provider>` with the provider name.
- **Auth `status=invalid` or `status=missing`** → See [provider setup](../../guides/provider-setup/).

## See also

- [Run a local model with Ollama](../local-ollama/)
- [Add a fallback provider chain](../fallback/)
- [Provider setup](../../guides/provider-setup/)
