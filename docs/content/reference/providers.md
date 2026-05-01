---
title: "Providers"
description: "Provider status taxonomy and where provider support is defined."
weight: 40
---

# Providers

Provider support is defined in the Go provider registry and related runtime code. Do not flatten all provider rows into "supported".

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
