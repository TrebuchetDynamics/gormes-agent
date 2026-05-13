---
title: "Common errors"
description: "Symptom, likely cause, and the smallest useful next step."
aliases:
  - /using-gormes/faq/
---

# Common errors

Start every investigation with the running binary and the offline doctor:

```bash
which -a gormes
gormes version
gormes doctor --offline
gormes config check
gormes gateway status
```

| Symptom | Likely cause | Fix |
|---|---|---|
| `gormes` not found after `install.sh` | Install put the binary in `$HOME/go/bin` but it is not on `PATH`. | Add `export PATH="$HOME/go/bin:$PATH"` to your shell rc. |
| Command behavior looks stale or matches an older release | Multiple `gormes` binaries on `PATH`. | `which -a gormes` and run the intended path directly, or remove the older copy. |
| `gormes --oneshot "..."` fails with "provider auth missing" | No API key for the configured provider. | `gormes auth add <provider> --api-key ...` or run `gormes setup provider`. |
| `gormes doctor` reports provider not reachable | Endpoint, network, or credential mismatch. | `gormes config show` and verify `[hermes].endpoint`, then re-run `gormes doctor`. |
| Config edits do not take effect | Config file is not where you think it is. | `gormes config path` and edit the file it reports; `GORMES_HOME` overrides the default `~/.gormes`. |
| Secrets do not load | `.env` is missing or in the wrong place. | `gormes config env-path` and put provider tokens at the reported path. |
| Telegram bot stays silent | Gateway not running, or token/chat-id misconfigured. | `gormes gateway status`; verify `GORMES_TELEGRAM_TOKEN` and the allowed chat ids in config. |
| Browser tools fail | Chrome/Chromium not started with remote debugging. | Launch Chrome with `--remote-debugging-port=9222 --user-data-dir=$HOME/.gormes/chrome-debug`, then `export BROWSER_CDP_URL=http://127.0.0.1:9222`. |
| `gormes` crashes on launch | Panic before TUI start. | Look in `~/.gormes/` for the most recent `crash-<unix>.log`; the stderr message also names the crash file. |
| TUI session resumes with stale state | bbolt session map at `~/.gormes/sessions.db`. | `gormes --resume new` starts a fresh default session. |
| `gormes doctor` warns about GitHub auth | `gh` not authenticated and no token in env. | `gh auth login`, or `export GITHUB_TOKEN=...`. |

## Other useful answers

**Do I need Hermes running?** No. `gormes --offline` boots the native TUI and keeps typed messages local. No Hermes process, Python, Node, or Docker required for the offline path.

**Where does memory live?** Goncho-backed local memory under `~/.gormes/` (see `gormes goncho doctor --json` for the active configuration).

**How do I reset a session?** `gormes --resume new`.
