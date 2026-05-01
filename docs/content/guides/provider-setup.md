---
title: "Provider Setup"
description: "Configure model/provider credentials without leaking secrets into docs or config examples."
weight: 30
---

# Provider Setup

Start with the config commands:

```bash
gormes config show
gormes config set hermes.provider openai-codex
gormes config set hermes.model gpt-5.5
```

Secret-like values should live in the dotenv path reported by:

```bash
gormes config env-path
```

Then test one provider-backed call:

```bash
gormes --oneshot "hello from Gormes"
```

The provider manifest contains multiple implementation statuses. Public docs should distinguish runtime-implemented providers from row-backed parity entries so users do not mistake audit coverage for live support.
