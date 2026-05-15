---
title: "Provider Setup"
description: "Configure model/provider credentials without leaking secrets into docs or config examples."
weight: 30
---

# Provider Setup

Start with the interactive commands:

```bash
gormes setup provider
gormes setup model
gormes model
```

The config commands are useful when you need an inspectable or scripted path:

```bash
gormes config show
gormes config set hermes.provider openai-codex
gormes config set hermes.model gpt-5.5
```

Secret-like values should live in the dotenv path reported by:

```bash
gormes config env-path
```

That path is usually `~/.gormes/.env`.

Then test one provider-backed call:

```bash
gormes chat -q "hello from Gormes"
gormes doctor
```

Provider credentials can also go through the credential pool:

```bash
gormes auth add openai --api-key "$OPENAI_API_KEY"
gormes auth add openai-codex --type oauth
```

The provider manifest contains multiple implementation statuses. Public docs should distinguish runtime-implemented providers from row-backed parity entries so users do not mistake audit coverage for live support.

Use the upstream Hermes [provider docs](../../upstream-hermes/integrations/providers/) and [model catalog](../../upstream-hermes/reference/model-catalog/) as parity context. Gormes-native setup still follows the commands above.
