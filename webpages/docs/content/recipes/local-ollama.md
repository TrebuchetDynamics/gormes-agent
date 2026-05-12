---
title: "Run a local model with Ollama"
description: "Send a one-shot turn through a locally hosted Ollama server."
difficulty: "S"
---

# Run a local model with Ollama

> **Outcome:** A one-shot turn completes against a model running on local Ollama — no cloud provider required.
>
> **Prerequisites:** `gormes` installed. An [Ollama](https://ollama.com) server reachable at `http://localhost:11434/v1`. At least one model pulled (e.g. `ollama pull llama3.1`).

## Steps

1. **Confirm Ollama is up**
   ```bash
   curl -s http://localhost:11434/v1/models | head
   ```
   You should see a JSON document listing local models.

2. **Run a one-shot through Ollama**
   ```bash
   gormes --oneshot "test local model" \
     --provider ollama \
     --endpoint http://localhost:11434/v1 \
     --model llama3.1
   ```
   Replace `llama3.1` with the exact tag returned by step 1.

## Verify

```bash
gormes --oneshot "say hi" --provider ollama --endpoint http://localhost:11434/v1 --model llama3.1
```

Expected: a model-generated reply on stdout. The process exits with status 0.

## Troubleshooting

- **`Not Found: model 'xxx' not found`** → The model tag does not exist locally. Pull it: `ollama pull <tag>`, then pass the exact tag to `--model`.
- **Connection refused / timeout** → Ollama is not running on `localhost:11434`. Start it (`ollama serve`) or correct the `--endpoint` URL.
- **Want it as the default?** → Persist with `gormes setup provider` and pick `custom`/`ollama`, or set `inference.provider`, `inference.endpoint`, and `inference.model` via `gormes config set`.

## See also

- [Connect a provider and run a one-shot](../first-turn/)
- [Add a fallback provider chain](../fallback/) — keep Ollama as a free fallback.
