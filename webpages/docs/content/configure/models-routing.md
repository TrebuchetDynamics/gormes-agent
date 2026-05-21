---
title: "Models and routing"
description: "Choose the active provider/model and understand invocation overrides and fallback routing."
aliases:
  - /configure/models/
  - /configure/routing/
---

Providers answer where Gormes sends model requests. Models and routing answer
which model is selected for a run and how overrides behave.

## Choose interactively

```bash
gormes setup model
gormes model
```

## Override for one run

```bash
gormes chat --provider openai --model gpt-5.5
gormes chat -q "hello" --provider openai --model gpt-5.5
```

Invocation flags override environment variables, which override `config.toml`,
which override built-in defaults.

## Fallback chains

Use fallback when you want resilience across providers:

```bash
gormes fallback --help
```

The operator workflow lives in [Fallback provider chain](../../operate/fallback-providers/).
Exact provider credential setup lives in [Providers](../providers/).
