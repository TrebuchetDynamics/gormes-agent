---
title: "Route channels to different agents"
description: "Bind one gateway's channels to different agent personas with different workspaces or models."
difficulty: "M"
---

# Route channels to different agents

> **Outcome:** Two named agents (for example `support` and `coder`) live inside one Gormes gateway; each Telegram or Slack channel routes to the right agent.
>
> **Prerequisites:** A working multi-channel gateway (see [run a multi-channel gateway](../multi-channel/)).

## Steps

1. **List configured agents and bindings**
   ```bash
   gormes config show
   ```
   The `agents` and `bindings` sections show what is configured now. Unbound channels route to the default agent.

2. **Edit `config.toml`**
   ```bash
   gormes config edit
   ```
   Add at least one agent definition and one binding. Example:

   ```toml
   [[agents.list]]
   id = "support"
   name = "Support"
   workspace = "/home/alice/support"
   model = "gpt-4o"

   [[agents.list]]
   id = "coder"
   name = "Coder"
   workspace = "/home/alice/projects"
   model = "claude-sonnet-4-20250514"

   [[bindings]]
   agent_id = "coder"
   [bindings.match]
   channel = "telegram"
   account_id = "coding-bot"
   ```

   `account_id` is the channel-side identifier (Telegram bot username, Slack app id, etc.). Unmatched messages fall through to the default agent.

3. **Validate the config**
   ```bash
   gormes config check
   ```
   Empty bindings or missing agent ids are reported here, before runtime.

4. **Apply without restart**
   ```bash
   gormes gateway reload
   ```
   Agent definitions and bindings are reload-safe.

## Verify

```bash
gormes gateway status
gormes config show
```

Expected: `config show` shows the new agent ids; `gateway status` shows each channel mapped to the bound agent. Send a message to each channel — the reply should reflect the bound agent's workspace and model.

## Troubleshooting

- **All messages still hit the default agent** → The binding `match` block did not match. Confirm `channel` and `account_id` against the values shown in `gormes gateway status`.
- **`agent_id` rejected at config check** → The id in `bindings` must match an `agents.list` entry's `id`.
- **`gormes setup bindings` errors with `setup_requires_tty`** → The guided wizard needs an interactive TTY. Hand-edit `config.toml` instead (step 2 above).

## See also

- [Run a multi-channel gateway](../multi-channel/)
- [Switch profiles for client work](../profiles/)
- [Config file](../../configure/config-file/)
