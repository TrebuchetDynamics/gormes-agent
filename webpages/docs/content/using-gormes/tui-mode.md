---
title: "TUI Mode"
weight: 30
---

# TUI Mode

The default interface. A Bubble Tea terminal shell for local smoke checks and configured provider-backed turns.

## Launch

```bash
gormes --offline
```

`--offline` keeps typed messages local. Run `gormes` without `--offline` after configuring a provider-compatible endpoint.

## Keybindings

| Key | Action |
|---|---|
| `Ctrl+C` | Quit when idle or failed; cancel during an active turn |
| `Ctrl+L` | Clear output |
| `↑` / `↓` | Cycle through history |
| `Enter` | Send current text |

## Layout

The TUI coalesces streamed tokens at 16 ms (the render mailbox), so scrolling under load stays responsive. Route-B reconnect recovers dropped SSE streams without resetting the turn.

## Session resume

Each invocation reattaches to the last session via a bbolt map at `~/.gormes/sessions.db`. To start fresh: `gormes --resume new`.
