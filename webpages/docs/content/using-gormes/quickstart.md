---
title: "Quickstart"
weight: 10
---

# Quickstart

Get Gormes running locally from source.

## 1. Build

```bash
git clone https://github.com/TrebuchetDynamics/gormes-agent.git
cd gormes-agent
make build
export PATH="$PWD/bin:$PATH"
```

Builds `gormes` from the source tree you inspected and puts that fresh command
first on `PATH`. Requires Go 1.25+. For install.sh see [Install](../install/).

## 2. Verify the local stack

Offline diagnostics do not contact a model provider:

```bash
gormes doctor --offline
gormes goncho doctor --json
```

See [Wire Doctor](../wire-doctor/) for what this checks.

## 3. Optional model-backed turn

```toml
# ~/.config/gormes/config.toml
[hermes]
endpoint = "https://your-provider.example/v1"
api_key = "..."
model = "your-model"
```

```bash
gormes --oneshot "hello from Gormes"
```

## 4. Run

```bash
gormes --offline
```

You're in the local TUI. Offline mode keeps typed messages local and does not call a provider. Press `Ctrl+C` to exit.

## Next

- [TUI mode](../tui-mode/) — keybindings, layout
- [Telegram adapter](../telegram-adapter/) — use the same brain from Telegram
- [Configuration](../configuration/) — persistent settings
