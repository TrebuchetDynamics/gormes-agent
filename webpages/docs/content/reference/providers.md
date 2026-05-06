---
title: "Providers"
description: "Provider status taxonomy and where provider support is defined."
weight: 40
---

# Providers

Provider support is defined in the Go provider registry and related runtime code. Do not flatten all provider rows into "supported".

Use upstream Hermes [provider docs](../../upstream-hermes/integrations/providers/) and the [model catalog](../../upstream-hermes/reference/model-catalog/) for parity context. Gormes marks runtime-implemented providers separately from row-backed parity entries.

Status labels:

| Label | Meaning |
|---|---|
| `implemented` | Runtime code exists for the provider path. |
| `owned` | Gormes owns a native integration path that differs from a plain API-key provider. |
| `row-backed` | The provider is represented in parity planning or registry evidence, but should not be promised as live without runtime verification. |
| `excluded` | Deliberately not supported. |

Before documenting a provider as user-ready, verify:

```bash
gormes config set hermes.provider <provider>
gormes config set hermes.model <model>
gormes --oneshot "provider smoke test"
```
