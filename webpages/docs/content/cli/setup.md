---
title: "gormes setup"
description: "Guided interactive setup — provider, model, and more"
---

# gormes setup

Guided interactive setup — provider, model, and more.

`setup` is section-driven, not a subcommand tree. Pass a section as a
positional argument, or run `gormes setup` to walk the full wizard.

## Synopsis

```
gormes setup [section] [flags]
```

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `-h`, `--help` | | help for setup |
| `--json` | `false` | emit machine-readable JSON for `--reset`: `{build, action: 'reset', config_path, breadcrumb_path}` |
| `--non-interactive` | `false` | use defaults/env and never prompt |
| `--quick` | `false` | configure missing setup items only |
| `--reconfigure` | `false` | re-run the setup wizard against the current config (non-destructive; existing values are kept where the operator skips a step) |
| `--reset` | `false` | DESTRUCTIVE: overwrite `config.toml` back to defaults, then re-run the setup wizard |

## See also

- [CLI reference](../)
- [`gormes model`](../model/)
- [Provider setup](../../configure/providers/)
