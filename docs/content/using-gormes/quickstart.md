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
```

Builds `./bin/gormes` from the source tree you inspected. Requires Go 1.25+. For convenience installer paths see [Install](../install/).

## 2. Verify the local stack

Offline diagnostics do not contact a model provider:

```bash
./bin/gormes doctor --offline
./bin/gormes goncho doctor --json
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
./bin/gormes --oneshot "hello from Gormes"
```

## 4. Run

```bash
./bin/gormes --offline
```

You're in the local TUI. Offline mode keeps typed messages local and does not call a provider. Press `Ctrl+C` to exit.

## Next

- [TUI mode](../tui-mode/) — keybindings, layout
- [Telegram adapter](../telegram-adapter/) — use the same brain from Telegram
- [Configuration](../configuration/) — persistent settings
