---
title: "Profile config v2"
description: "Target single-root config.toml schema for Gormes profile services."
---

# Profile config v2

This is the target architecture for the next profile-control-center slices. It
is a planning contract, not a claim that every runtime path already writes this
shape.

## Invariants

- `$GORMES_HOME/config.toml` is the only canonical config file.
- New setup writes `config_version = 2`.
- A fresh install seeds `profiles.main` with `enabled = true` and `name = ""`.
- `profiles.<id>` is the stable profile id. `name` is display text and can
  change without changing storage keys, sessions, memory, or credentials.
- All `enabled = true` profiles are active services. There is no
  `active_profile` or `default_profile` field in config v2.
- Profile directories may hold runtime data such as sessions, memory, logs, or
  subprocess HOME, but not canonical per-profile config files.
- Credentials live in one registry and are referenced by id. Sharing a
  credential across profiles is explicit and visible; silent global fallback is
  not v2 setup.
- Raw API keys, bot tokens, OAuth refresh tokens, and channel secrets are never
  stored inline in `config.toml`.

## Root shape

```toml
config_version = 2

[profiles.main]
enabled = true
name = ""
description = ""
workspaces = []
tags = []

[profiles.main.runtime]
max_turns = 60
session_reset_policy = "inactivity"
goncho_workspace = "main"

[profiles.main.providers.openrouter]
enabled = true
credential = "main-openrouter"
default_model = "anthropic/claude-sonnet-4.5"
allowed_models = [
  "anthropic/claude-sonnet-4.5",
  "openai/gpt-5.1",
  "google/gemini-2.5-pro"
]

[profiles.main.channels.telegram]
enabled = true
credential = "main-telegram"
allowed_chats = ["123456789"]
allowed_users = ["123456789"]
require_mention = false
tool_progress = "compact"

[profiles.main.channels.navivox]
enabled = true
servers = ["local", "tailnet"]
voice_profile = "main-spanish"

[profiles.tulin]
enabled = true
name = "Tulin"
description = "Voice-first research assistant"
workspaces = ["/home/xel/git/sages-openclaw/workspace-tulin"]
tags = ["voice", "research"]

[profiles.tulin.providers.openrouter]
enabled = true
credential = "tulin-openrouter"
default_model = "openai/gpt-5.1"

[profiles.tulin.channels.telegram]
enabled = true
credential = "tulin-telegram"
allowed_chats = ["987654321"]

[credentials.main-openrouter]
kind = "provider"
provider = "openrouter"
owner_profile = "main"
secret_ref = { env = "GORMES_MAIN_OPENROUTER_API_KEY" }

[credentials.tulin-openrouter]
kind = "provider"
provider = "openrouter"
owner_profile = "tulin"
secret_ref = { env = "GORMES_TULIN_OPENROUTER_API_KEY" }

[credentials.main-telegram]
kind = "channel"
channel = "telegram"
owner_profile = "main"
secret_ref = { env = "GORMES_MAIN_TELEGRAM_BOT_TOKEN" }

[credentials.tulin-telegram]
kind = "channel"
channel = "telegram"
owner_profile = "tulin"
secret_ref = { env = "GORMES_TULIN_TELEGRAM_BOT_TOKEN" }

[navivox]
enabled = true

[navivox.servers.local]
enabled = true
bind = "127.0.0.1:8787"
profiles = ["main", "tulin"]
transports = ["http", "ws"]

[navivox.servers.tailnet]
enabled = true
bind = "100.64.0.5:8787"
profiles = ["main"]
transports = ["http", "ws"]

[[gateway.routing.telegram.rules]]
profile = "main"
credential = "main-telegram"
chat_id = "123456789"

[[gateway.routing.telegram.rules]]
profile = "tulin"
credential = "tulin-telegram"
chat_id = "987654321"
```

## Profiles

The table key is identity. `profiles.main` stays `main` even after the user
names it "Yunobo". Runtime state, credential ownership, gateway routing, and
Navivox routes refer to the stable id.

`name = ""` is valid only as an incomplete setup state. `gormes setup profiles`
and first runtime touch should ask the operator for the display name, stage the
edit, and write it only on Apply.

## Credentials

Credentials are global records with owner metadata because secrets often need a
central redaction, migration, and rotation path. Profile blocks reference them
by id:

```toml
[profiles.main.providers.openrouter]
credential = "main-openrouter"

[credentials.main-openrouter]
kind = "provider"
provider = "openrouter"
owner_profile = "main"
secret_ref = { env = "GORMES_MAIN_OPENROUTER_API_KEY" }
```

Credential sharing is legal only when another profile references the same id
and the UI shows that it is shared. Setup must not infer sharing because two
providers happen to read the same legacy env var.

## Providers

Provider setup is per profile. A profile can use OpenRouter, OpenAI, Anthropic,
Codex OAuth, Ollama, or any future provider without changing another profile.
Provider readiness must report missing credentials, OAuth-vs-API-key mismatch,
endpoint problems, and model-catalog failures without leaking secret refs.

OpenRouter and other aggregators need a catalog-backed picker, with manual
entry as fallback. A static short model list is not enough for v2 setup.

## Channels

Channel setup is per profile. A Telegram, Slack, Discord, WhatsApp, or Navivox
channel block declares profile-specific enablement, credential id, and routing
or allow-list policy.

Legacy top-level `[telegram]`, `[discord]`, and `[slack]` sections remain
migration inputs and read fallbacks. New setup writes only
`[profiles.<id>.channels.<channel>]` plus credential records.

## Navivox

Navivox manages a profile fleet. It does not select one startup/default profile.
Each server declares which profile ids it can expose, and each profile declares
which Navivox servers it uses.

This supports one local server for every profile, separate tailnet/server
groups, and future remote/mobile control surfaces without changing the profile
schema.

## Migration rules

`internal/config` owns migration planning and apply. `gormes setup profiles`
can present the migration, but it should not hand-write TOML itself.

Migration inputs:

- legacy root `config.toml` fields such as `[hermes]`, `[telegram]`, `[slack]`,
  `[discord]`, `[agents.defaults]`, and `[[bindings]]`
- old named profile directories under `$GORMES_HOME/profiles/<id>/`
- old per-profile `config.toml` files as import sources
- `$GORMES_HOME/active_profile` as compatibility state only
- `$GORMES_HOME/.env` and external auth stores as secret sources

Migration output:

- one root `config.toml`
- one or more `[profiles.<id>]` records
- one or more `[credentials.<id>]` records with secret refs
- profile provider/channel/workspace settings under the profile record
- a redacted preview diff before Apply
- a backup before mutation

Migration must not delete old profile runtime directories, write raw secrets
into TOML, or promote the legacy active profile into a v2 default.

## Profile Control Center

`gormes setup profiles` should open a Profile Control Center over this schema:

- first screen: every profile service, with enabled/disabled grouping
- lanes: Runtime, Readiness, Activity
- details: workspaces, providers, channels, credential ownership, Navivox
  servers, migration state, and typed available actions
- edits: draft first, explicit Apply, redacted diff summary
- creation: stage stable id and display name before write
- accessibility: stable screen-reader text is the contract; visual layout is
  an enhancement
- excluded from slice 1: live start/stop/reset/restart and arbitrary command
  launchers

The control center is allowed to manage profile configuration deeply. Live
runtime supervision belongs to later fleet rows.
