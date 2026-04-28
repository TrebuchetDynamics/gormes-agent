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

```bash
export GORMES_ENDPOINT="https://your-provider.example/v1"
export GORMES_API_KEY="..."
export GORMES_MODEL="your-model"
./bin/gormes --oneshot "hello from Gormes"
```

## 4. Run

```bash
./bin/gormes --offline
```

You're in the local TUI. Press `Ctrl+C` to exit.

## Next

- [TUI mode](../tui-mode/) — keybindings, layout
- [Telegram adapter](../telegram-adapter/) — use the same brain from Telegram
- [Configuration](../configuration/) — persistent settings
