---
title: "Getting Started"
description: "Install Gormes, verify the local runtime, and complete a first useful run."
weight: 10
---

# Getting Started

Start here when you want Gormes running, not explained from first principles.

1. [Install](installation/) from source or an inspectable installer.
2. [First Run](first-run/) with offline diagnostics and a first provider-backed turn.
3. [Configuration](configuration/) for `GORMES_HOME`, `config.toml`, `.env`, providers, gateway, and state paths.
4. [Provider Setup](../guides/provider-setup/) for credential and model selection.
5. [Gateway Operations](../guides/gateway-operations/) and [Telegram Bot](../guides/telegram-bot/) for messaging channels.
6. [CLI Commands](../reference/cli/), [Config](../reference/config/), [Environment](../reference/environment/), and [Providers](../reference/providers/) for exact operator surfaces.
7. [Troubleshoot](troubleshooting/) common setup and runtime failures.

The conservative path is to verify the local runtime before adding model or channel credentials:

```bash
git clone https://github.com/TrebuchetDynamics/gormes-agent.git
cd gormes-agent
make build
gormes doctor --offline
gormes --offline
```
