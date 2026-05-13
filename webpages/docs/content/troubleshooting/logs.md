---
title: "Logs"
description: "Where the Gormes runtime log lives, how to rotate it, and how to grep for tool errors."
---

# Logs

Gormes writes a single runtime log under the Gormes home directory.

## Locations

| File | Contents |
|---|---|
| `~/.gormes/gormes.log` | Current runtime log: provider turns, tool calls, gateway events, doctor warnings. |
| `~/.gormes/crash-<unix>.log` | Timestamped TUI panic dump. The stderr message after a crash names the file. |

`GORMES_HOME` overrides the default `~/.gormes` location. There is no separate `gateway.log` — gateway events are written into `gormes.log`.

## Tailing live

```bash
tail -f ~/.gormes/gormes.log
```

Or, if you only want the tail through the gateway HTTP endpoint when the gateway is running:

```bash
gormes logs            # human-readable tail
gormes logs --json     # machine-readable entries
```

## Grepping for tool errors

```bash
grep -i "tool=" ~/.gormes/gormes.log | tail -50
grep -i error ~/.gormes/gormes.log | tail -50
grep -i panic ~/.gormes/gormes.log | tail -50
```

When you suspect a single tool is failing, narrow by tool name:

```bash
grep -i "tool=browser_navigate" ~/.gormes/gormes.log | tail -50
grep -i "tool=web_search" ~/.gormes/gormes.log | tail -50
```

## Rotating

Gormes does not rotate `gormes.log` itself. To keep the file from growing without bound, archive and truncate:

```bash
mv ~/.gormes/gormes.log ~/.gormes/gormes.log.$(date +%Y%m%d)
: > ~/.gormes/gormes.log
```

Or set up host-level rotation (`logrotate(8)` on Linux, `newsyslog(8)` on macOS) pointing at `~/.gormes/gormes.log`.

## Crash logs

After a TUI panic, look for the most recent `crash-*.log` under `~/.gormes/`:

```bash
ls -lt ~/.gormes/crash-*.log | head -5
```

Each crash dump contains the panic stack trace and the goroutine state at the moment of failure.
