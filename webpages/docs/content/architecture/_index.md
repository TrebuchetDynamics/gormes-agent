---
title: "Architecture"
description: "How the Go runtime connects CLI, TUI, gateway, providers, tools, sessions, and memory."
weight: 40
---

# Architecture

Gormes is a Go-native runtime that keeps Hermes-compatible behavior where Hermes is the active contract.

```text
CLI / TUI / Gateway
  -> config and auth
  -> provider client
  -> agent loop
  -> tool registry
  -> sessions and memory
  -> terminal or channel renderer
```

Start with:

- [Runtime model](runtime-model/)
- [Gateway pipeline](gateway-pipeline/)
- [Tool execution](tool-execution/)
- [Memory and sessions](memory-and-sessions/)
- [Hermes parity](hermes-parity/)
