---
title: "Repo Layout"
description: "Where core Gormes subsystems live."
weight: 10
---

# Repo Layout

High-level map:

| Path | Purpose |
|---|---|
| `cmd/gormes` | CLI, TUI startup, dashboard, gateway commands, setup/config. |
| `internal/config` | Native config, `GORMES_HOME`, dotenv, schema validation. |
| `internal/hermes` | Provider and Hermes-compatible runtime surfaces. |
| `internal/gateway` | Shared command, event, session, and rendering logic for channels. |
| `internal/tools` | Tool registry and tool implementations. |
| `internal/goncho` | Honcho-compatible memory facade. |
| `docs/content` | Hugo documentation content. |
| `docs/layouts` | Custom Hugo templates. |
