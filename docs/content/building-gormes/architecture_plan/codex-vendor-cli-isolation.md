---
title: "Codex Vendor-CLI Isolation"
weight: 60
aliases:
  - /building-gormes/codex-vendor-cli-isolation/
---

# Codex Vendor-CLI Isolation

The race. Codex CLI and the VS Code Codex extension both read and write
`~/.codex/auth.json`. Both refresh access tokens independently using the same
refresh token. When refresh tokens rotate, the loser of the race holds an
invalidated token. Any third client (Hermes, Gormes) sharing the file inherits
the same race envelope: a successful refresh by Codex CLI or the extension can
silently invalidate the third client's last-known refresh token, and the third
client only discovers it on the next inference call.

## How Hermes handles the race

Hermes' `auth_commands.py:auth_add_command:openai-codex` branch is strict by
default. It always runs a fresh device-code flow against the Hermes-owned
`auth.json` under the credential pool and never imports tokens from
`~/.codex/auth.json`. The screen-filling warning text in
`auth.py:_codex_device_code_login` and `auth.py:_login_openai_codex` describes
the race envelope explicitly and recommends a separate device-code login for
safety even when an operator is tempted to share state with Codex CLI.

The legacy `_login_openai_codex` import path remains reachable from
`hermes model` for back-compat, but Hermes' own warning recommends a separate
device-code login rather than the import for any new setup.

## What Gormes does

- `gormes auth add openai-codex` is strict-by-default: device-code only, no
  `~/.codex/auth.json` import. See progress row
  `Gormes auth add openai-codex strict isolation contract`.
- `gormes model` is selection-only. Codex CLI import is NOT offered (Q6
  decision 2026-04-29). See progress row
  `Gormes model interactive provider/model picker`.
- Operators with a genuine offline / corporate need invoke
  `gormes auth add openai-codex --emergency-import-from-codex-cli <path>`.
  This path emits a screen-filling race-envelope warning and refuses an
  imported file whose JWT access token is expired before writing anything to
  the credential pool. The flag is opt-in only and is the only labeled
  emergency bridge from `~/.codex/auth.json` into Gormes-owned credentials.

## When this could be relaxed

Two conditions would change the calculus:

1. Codex CLI's auth model changes such that refresh tokens are no longer
   rotated on use, removing the race.
2. Gormes acquires the `auth.json` mutex (via cooperative file lock with
   Codex CLI / VS Code), serializing refresh attempts.

Neither is currently in scope. Until then, the strict isolation contract
holds.

## Citations

- `hermes-agent/hermes_cli/auth_commands.py:auth_add_command` (line 161,
  `openai-codex` branch) — strict path
- `hermes-agent/hermes_cli/auth.py:_codex_device_code_login` (line 3974) —
  race warning text
- `hermes-agent/hermes_cli/auth.py:_login_openai_codex` (line 3900) — legacy
  permissive path
- Progress rows: `Gormes auth add openai-codex strict isolation contract`,
  `Gormes model interactive provider/model picker`,
  `Codex OAuth state + stale-token relogin` (8908)
