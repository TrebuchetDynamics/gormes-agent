---
title: "First Run"
description: "Run local diagnostics, offline TUI mode, and a provider-backed one-shot."
weight: 20
---

# First Run

## Local Diagnostics

```bash
./bin/gormes doctor --offline
```

Offline doctor checks local runtime readiness without contacting a model provider.

## Offline TUI

```bash
./bin/gormes --offline
```

Offline mode is a smoke test. Typed messages stay local.

## Provider-Backed One-Shot

After configuring provider credentials:

```bash
./bin/gormes --oneshot "hello from Gormes"
```

If the one-shot fails, run:

```bash
./bin/gormes config show
./bin/gormes config check
./bin/gormes doctor
```
