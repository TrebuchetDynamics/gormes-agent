---
title: "CLI Commands"
description: "Current top-level Gormes commands and important operator subcommands."
weight: 10
---

# CLI Commands

Snapshot from `gormes --help` in this workspace:

| Command | Purpose |
|---|---|
| `gormes` | Start the TUI or run with flags such as `--offline` and `--oneshot`. |
| `gormes agent` | Manage agent context templates. |
| `gormes auth` | Manage provider credentials. |
| `gormes config` | Inspect or update native config files. |
| `gormes dashboard` | Start the local web dashboard. |
| `gormes doctor` | Verify runtime readiness. |
| `gormes gateway` | Run or inspect the multi-channel messaging gateway. |
| `gormes goncho` | Inspect local Goncho memory diagnostics. |
| `gormes logout` | Clear stored provider auth. |
| `gormes mcp` | Manage Hermes-compatible MCP servers. |
| `gormes memory` | Inspect memory and extractor state. |
| `gormes migrate` | Migrate state from upstream agents. |
| `gormes model` | Select model/provider. |
| `gormes onboard` | Show first-run setup status, runtime skills root, and next steps. |
| `gormes profile` | Inspect and switch profiles. |
| `gormes session` | Inspect and export sessions. |
| `gormes setup` | Configure runtime sections. Full wizard behavior is not complete in this slice. |
| `gormes skills` | List runtime and bundled skills, or install a direct `SKILL.md` URL. |
| `gormes telegram` | Run the direct Telegram adapter. |
| `gormes usage` | Show runtime/provider account usage. |
| `gormes version` | Print version. |

Regenerate this page from Cobra output before release.
