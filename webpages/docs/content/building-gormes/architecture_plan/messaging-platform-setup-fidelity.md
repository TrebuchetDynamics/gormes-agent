---
title: "Messaging Platform Setup Fidelity"
description: "Hermes-fidelity contract for Gormes messaging-platform setup, channel config, and migration behavior."
date: 2026-05-14
draft: false
---

# Messaging Platform Setup Fidelity

This page records the operator-approved contract for the next `gormes setup
gateway` slice. It is not a parallel backlog; the executable backlog item lives
in `progress.json` as `Bubble Tea Messaging Platforms setup: Telegram-first
Hermes fidelity`.

## Source Baseline

Gormes should mirror Hermes user-visible behavior as closely as possible while
remaining a Go-native runtime with Gormes-owned storage paths.

Source refs for the first implementation slice:

- `hermes-agent@9ed751b96:hermes_cli/setup.py`
- `hermes-agent@9ed751b96:hermes_cli/gateway.py`
- `hermes-agent@9ed751b96:gateway/config.py`
- `hermes-agent@9ed751b96:cli-config.yaml.example`
- `hermes-agent@9ed751b96:website/docs/user-guide/messaging/telegram.md`
- `hermes-agent@9ed751b96:website/docs/user-guide/messaging/whatsapp.md`

Hermes calls the setup area "Messaging Platforms" and exposes it through
`setup gateway`. Gormes keeps `gormes setup gateway` as the compatibility
command and may label the top-level setup entry "Configure channels" when that
reads better in the Go TUI.

## Runtime Boundaries

- TTY setup interaction uses Bubble Tea only. Raw key-reading and ad hoc
  line-menu navigation are out of scope for new setup flows.
- Non-TTY setup uses flags, env, config, piped input, or `--plan`; otherwise it
  returns typed guidance without hanging.
- Runtime paths are Gormes-owned. `GORMES_HOME` may select the Gormes runtime
  root. `HERMES_HOME` is migration input only and must not change normal
  runtime paths.
- Normal runtime must not read `~/.hermes/config.yaml` or `~/.hermes/.env`.
  Explicit `gormes migrate hermes` may read Hermes fixtures and must redact its
  report.
- Gormes runtime config stays TOML. Hermes YAML is an import source, not a live
  runtime format.

## Precedence

The channel setup and runtime loaders use this precedence:

1. CLI flags.
2. `GORMES_*` env.
3. Gormes config and secret refs.
4. Hermes-compatible env aliases, including unprefixed Hermes dotenv names and
   any supported `HERMES_*` compatibility names.
5. Explicit migration fixtures.
6. Defaults.

Setup writes new values with Gormes-owned names. For Telegram, the preferred
secret name is `GORMES_TELEGRAM_BOT_TOKEN`; existing Gormes aliases such as
`GORMES_TELEGRAM_TOKEN` and Hermes names such as `TELEGRAM_BOT_TOKEN` remain
read-compatible. Secrets go to `$GORMES_HOME/.env` or a `SecretRef`; non-secret
policy goes to `config.toml`.

## Setup Registry

Channel setup needs a registry separate from the runtime adapter registry. Each
setup entry should declare:

- platform id and label;
- status: `unconfigured`, `partial`, `configured`, `paired`, `running`, or
  `failed`;
- required secrets;
- access-policy fields;
- home-channel fields;
- typed Bubble Tea flow;
- `--plan` output;
- gateway lifecycle impact.

The checklist should show registered channels and status. Reconfigure mode may
preselect configured channels. Repair mode should default to the next missing
step for a partial channel.

## Telegram First Slice

The Telegram setup flow mirrors Hermes' order:

1. Bot token.
2. Access policy.
3. Home channel.

The flow includes compact BotFather guidance, token format validation, and a
security warning. Group setup guidance includes BotFather privacy mode, admin
status, and remove/re-add instructions. Defaults are conservative:
`require_mention=true`, `guest_mode=false`, and allowlist-first access.

Access policy is central. A token without a policy is `partial`, not
`configured`. Operators choose allowlist, pairing approvals, or open access;
open access is an explicit risky choice.

Home channel maps to structured config:

```toml
[telegram.home_channel]
chat_id = "123456789"
name = "Home"
thread_id = ""
```

The old `allowed_chat_id` field remains read-compatible but should be
soft-deprecated. `TELEGRAM_HOME_CHANNEL` imports into
`telegram.home_channel.chat_id`. `TELEGRAM_ALLOWED_USERS` imports into
`telegram.allowed_user_ids`; runtime also reads the env fallback. User IDs stay
numeric where the existing schema requires `[]int64`. Chat and home-channel IDs
stay strings.

`/set-home` behavior is preserved conceptually. A missing home channel is a
warning, not an unconfigured state.

## Review, Plan, And Lifecycle

Setup stages changes, shows a redacted diff summary, and applies only after
explicit confirmation. Canceling before apply writes nothing. Applying partial
configuration is allowed only as an explicit operator choice and leaves the
channel marked `partial`.

`--plan` prints required fields, redacted current status, planned writes, and
gateway action. It does not write files, contact live platform APIs, or show a
live QR code.

Gateway lifecycle handling is consolidated after all selected channel flows.
Safe config changes should reload once when possible. Restart-required changes
should produce restart guidance or run the existing lifecycle command only when
the operator explicitly asks.

## WhatsApp Boundary

The first phase may list WhatsApp but should hand off pairing to `gormes
whatsapp`; setup must not embed the QR flow. WhatsApp status should distinguish
`enabled` from `paired`.

## Verification Contract

The implementation must include redaction tests proving synthetic tokens do not
appear in setup stdout/stderr, review summaries, doctor output, capabilities
JSON, or migration reports.

Channel-scoped doctor checks run offline by default. A live "Test connection"
action may exist, but it must be explicit and skipped in unit tests.
