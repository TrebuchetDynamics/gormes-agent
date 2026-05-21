---
title: "Run a local model with Ollama"
description: "Send a single chat query through a locally hosted Ollama server."
difficulty: "S"
aliases:
  - /recipes/local-ollama/
---

> **Outcome:** A single chat query completes against a model running on local Ollama — no cloud provider required.
>
> **Prerequisites:** `gormes` installed. An [Ollama](https://ollama.com) server reachable at `http://localhost:11434/v1`. At least one model pulled (e.g. `ollama pull llama3.1`).

## Steps

1. **Confirm Ollama is up**
   ```bash
   curl -s http://localhost:11434/v1/models | head
   ```
   You should see a JSON document listing local models.

2. **Run a scripted chat query through Ollama**

   The provider, endpoint, and model are invocation overrides set through
   environment variables (`gormes chat` itself only takes `-q/--query`):
   ```bash
   GORMES_INFERENCE_PROVIDER=ollama \
   GORMES_ENDPOINT=http://localhost:11434/v1 \
   GORMES_INFERENCE_MODEL=llama3.1 \
     gormes chat -q "test local model"
   ```
   Replace `llama3.1` with the exact tag returned by step 1.

## Verify

```bash
GORMES_INFERENCE_PROVIDER=ollama GORMES_ENDPOINT=http://localhost:11434/v1 GORMES_INFERENCE_MODEL=llama3.1 gormes chat -q "say hi"
```

Expected: a model-generated reply on stdout. The process exits with status 0.

## Troubleshooting

- **`Not Found: model 'xxx' not found`** → The model tag does not exist locally. Pull it: `ollama pull <tag>`, then set `GORMES_INFERENCE_MODEL` to the exact tag.
- **Connection refused / timeout** → Ollama is not running on `localhost:11434`. Start it (`ollama serve`) or correct the `GORMES_ENDPOINT` URL.
- **Want it as the default?** → Persist with `gormes setup provider`, or set `hermes.provider`, `hermes.endpoint`, and `hermes.model` via `gormes config set`.

## See also

- [Connect a provider and open chat](../first-chat/)
- [Add a fallback provider chain](../fallback-providers/) — keep Ollama as a free fallback.
