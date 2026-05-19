---
title: "Setup wizard"
description: "Use gormes setup to configure provider, model, channel, tool, and terminal settings."
aliases:
  - /setup/
  - /configure/setup/
---

Use the setup wizard after `gormes doctor --offline` passes.

```bash
gormes setup
```

The wizard guides the operator through provider, model, channel, tool, and
terminal settings without requiring manual edits to `config.toml`.

## Focused setup

Run a specific setup section when you already know what needs to change:

```bash
gormes setup provider
gormes setup model
gormes setup gateway
gormes setup tools
gormes setup terminal
```

Use the exact command reference for flags such as `--quick`,
`--non-interactive`, `--reconfigure`, and `--reset`:

- [CLI reference: setup](../../cli/setup/)

## Verify

```bash
gormes config show
gormes config check
gormes doctor --offline
```
