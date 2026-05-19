---
title: "CLI reference"
description: "One page per top-level Gormes command, audited against the binary."
weight: 30
aliases:
  - /reference/cli/
  - /cli-reference/
---

# CLI reference

These pages are generated from `gormes --help`. If a flag is documented here,
the running binary supports it. Pages are audited against the v0.2.11 binary.

Runtime-ready commands are promoted for configured local use in the current
Gormes binary. Row-backed commands are part of the Hermes parity inventory or
a planned runtime lane, where the deterministic error path is more useful
than an unknown-command failure. Examples: `gormes gateway status`,
`gormes whatsapp`.

To re-check at any time:

```bash
gormes --help
gormes <command> --help
```

## Top-level commands

| Command | Purpose |
|---|---|
| [`gormes acp`](acp/) | Run ACP bridge tools |
| [`gormes agent`](agent/) | Manage Gormes agent context templates |
| [`gormes auth`](auth/) | Manage Hermes-compatible provider credentials |
| [`gormes channels`](channels/) | Inspect channel capability metadata |
| [`gormes chat`](chat/) | Open chat or send a single query |
| [`gormes checkpoints`](checkpoints/) | Inspect and manage Gormes file-operation rollback state |
| [`gormes claw`](claw/) | Hermes-compatible OpenClaw migration tools |
| [`gormes completion`](completion/) | Generate shell completion script |
| [`gormes config`](config/) | Inspect or update the Gormes configuration files |
| [`gormes cron`](cron/) | Manage scheduled cron jobs |
| [`gormes curator`](curator/) | Manage Hermes-compatible background skill curation |
| [`gormes dashboard`](dashboard/) | Start the Gormes web dashboard |
| [`gormes doctor`](doctor/) | Verify Gormes runtime: provider readiness + built-in tools |
| [`gormes fallback`](fallback/) | Manage fallback providers |
| [`gormes gateway`](gateway/) | Run Gormes as a multi-channel messaging gateway |
| [`gormes goncho`](goncho/) | Inspect local Goncho memory diagnostics |
| [`gormes kanban`](kanban/) | Manage the durable local multi-agent Kanban board |
| [`gormes logout`](logout/) | Clear stored authentication for a Hermes-compatible provider |
| [`gormes logs`](logs/) | Show recent Gormes gateway logs |
| [`gormes mcp`](mcp/) | Manage Hermes-compatible MCP servers |
| [`gormes memory`](memory/) | Inspect persisted memory and extractor state |
| [`gormes migrate`](migrate/) | Migrate state from upstream agents into Gormes (dry-run only in this slice) |
| [`gormes model`](model/) | Interactively select the active model/provider |
| [`gormes navivox`](navivox/) | Navivox HTTP channel utilities |
| [`gormes plugins`](plugins/) | Manage Hermes-compatible plugins |
| [`gormes profile`](profile/) | Inspect and switch the active Gormes profile |
| [`gormes restore`](restore/) | Discover and restore from a pre-update backup zip |
| [`gormes secrets`](secrets/) | Apply, audit, configure, and reload SecretRef-backed runtime secrets |
| [`gormes security`](security/) | Audit gateway, channel, tool, filesystem, and credential security |
| [`gormes session`](session/) | Inspect and export persisted sessions |
| [`gormes setup`](setup/) | Guided interactive setup — provider, model, and more |
| [`gormes skills`](skills/) | Manage skills |
| [`gormes slack`](slack/) | Slack integration helpers |
| [`gormes status`](status/) | Show Gormes runtime and progress blockers |
| [`gormes system`](system/) | Enqueue system events and inspect heartbeat or presence state |
| [`gormes telegram`](telegram/) | Run Gormes as a Telegram bot adapter |
| [`gormes uninstall`](uninstall/) | Remove Gormes artifacts from this system |
| [`gormes update`](update/) | Update a managed Gormes source checkout |
| [`gormes usage`](usage/) | Show runtime/provider account usage |
| [`gormes version`](version/) | Print gormes version |
| [`gormes whatsapp`](whatsapp/) | Set up WhatsApp pairing through the Hermes-compatible Baileys bridge |

## Global flags

These flags work on every command:

| Flag | Default | Purpose |
|---|---|---|
| `--offline` | `false` | run the TUI as a local smoke test without provider health checks or network submits |
| `-p`, `--profile` | (active) | profile name for this invocation |
| `--skills` | (none) | runtime skill allowlist for this invocation; repeat or comma-separate |

## Top-level invocation flags

These flags work on the bare `gormes` invocation:

| Flag | Default | Purpose |
|---|---|---|
| `--api-key` | (none) | provider API key override for chat or TUI startup; invocation-only and never persisted |
| `-c`, `--continue` | `last` | resolve a session id or unique prefix and resume it |
| `--endpoint` | (none) | provider endpoint override for chat or TUI startup; invocation-only and also settable via `GORMES_ENDPOINT` |
| `-m`, `--model` | (none) | model override for chat or TUI startup; also settable via `GORMES_INFERENCE_MODEL` |
| `--provider` | (none) | provider override for chat or TUI startup; also settable via `GORMES_INFERENCE_PROVIDER` |
| `--remote` | (none) | connect the TUI to a remote Gormes gateway over SSE (consumes `/events`; bypasses local kernel and provider setup) |
| `--resume` | (none) | override persisted `session_id` for the TUI's default key |
| `-V`, `--version` | | version for gormes |
