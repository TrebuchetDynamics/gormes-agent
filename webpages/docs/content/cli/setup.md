---
title: "gormes setup"
description: "Guided interactive setup — provider, model, and more"
---

# gormes setup

Guided interactive setup — provider, model, and more.

`setup` is section-driven, not a subcommand tree. Pass a section as a
positional argument, or run `gormes setup` to walk the full wizard.

## Sections

| Section | Purpose |
|---|---|
| `provider` | Configure provider credentials and routing |
| `model` | Select the active provider/model |
| `agent` | Show multi-agent setup guidance |
| `workspace` | Show default workspace guidance |
| `profiles` | Manage profiles and persist per-profile workspace/channel lists |
| `bindings` | Show channel-to-agent binding guidance |
| `tts` | Configure text-to-speech |
| `terminal` | Configure the terminal backend |
| `gateway` | Configure messaging gateways |
| `tools` | Configure tool groups |

`gormes setup profiles` writes `agents.defaults.workspaces` into the selected
profile's own `config.toml`. Today that list is persisted profile metadata; the
runtime workspace allow-list enforcement is row-backed. When that policy ships,
an empty list means the operator home is the default project workspace, while a
non-empty list restricts project read/write work to those roots. Runtime
internals still use the active profile root for state, but model-facing profile
edits are limited to explicit profile-owned content such as `SOUL.md`,
`IDENTITY.md`, and `skills/`; secrets and runtime databases are not project
workspaces.

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
