---
title: "FAQ"
weight: 70
---

# FAQ

### Do I need Hermes running?

No running Hermes process is required for the local smoke path. `gormes --offline` boots the native TUI and keeps typed messages local so you can verify the runtime without credentials, provider calls, Python, Node, or Docker.

### Can I use it without Python?

Yes for the native TUI, doctor diagnostics, Goncho diagnostics, and provider-compatible one-shot/TUI startup paths. Some deeper Hermes parity surfaces are still active roadmap work; see the [Roadmap](../../building-gormes/architecture_plan/).

### Where does memory live?

`~/.hermes/memory/memory.db` (SQLite) with a human-readable mirror at `~/.hermes/memory/USER.md`. The mirror refreshes every 30 seconds.

### How do I back up memory?

Copy `~/.hermes/memory/memory.db` — it's a single SQLite file. USER.md regenerates from it.

### The install script installed Gormes to `$HOME/go/bin` but it's not on my PATH.

Add it: `export PATH="$HOME/go/bin:$PATH"` in your shell rc.

### How do I reset a session?

```bash
gormes --resume new
```

### Logs?

`~/.hermes/gormes.log` (current run) and `~/.hermes/crash-*.log` (panics). Crash logs are timestamped.
