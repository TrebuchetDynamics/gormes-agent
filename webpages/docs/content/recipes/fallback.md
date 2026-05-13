---
title: "Add a fallback provider chain"
description: "Append backup providers so a primary outage doesn't kill turns."
difficulty: "S"
---

# Add a fallback provider chain

> **Outcome:** When the primary provider fails (rate limit, outage, expired key), Gormes automatically tries the next provider/model in the configured chain.
>
> **Prerequisites:** At least two configured providers (see [first turn](../first-turn/)).

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
gormes --oneshot "test fallback"
```

The turn should still return a reply — served by the next entry in the chain.

## Troubleshooting

- **Fallback never activates** → The primary call must fail, not partially succeed. Streaming or partial errors may not promote to the next entry; check `gormes logs`.
- **`fallback add` opens an empty picker** → Add provider credentials first (`gormes auth add <provider>`).

## See also

- [Connect a provider and run a one-shot](../first-turn/)
- [Provider setup](../../guides/provider-setup/)
