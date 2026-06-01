---
title: "Parity lane coordinator brief"
description: "Dispatch order and per-lane backlog summary derived from the Hermes <-> Gormes parity matrix. Builders consult this to pick the next slice without waiting for an operator screenshot."
date: 2026-04-29
draft: false
aliases:
  - /building-gormes/architecture_plan/parity-lane-coordinator-brief/
---

# Parity lane coordinator brief

Companion to
[hermes-gormes-parity-matrix](../hermes-gormes-parity-matrix/).

This brief is the dispatch surface for the parity operating system.
Each lane (B-H) corresponds to one builder workflow that closes a
specific class of parity gap.

## Current P0 Dispatch State

2026-04-30 refresh: the earlier dogfood P0 trio is no longer pending.
`Telegram production live-turn provider payload golden`,
`Telegram /status Hermes-format closeout`, and
`Gateway session token accounting parity` are all `complete/validated` in
`progress.json`.

Do not dispatch those rows as planned work. For the live queue, pick the next
`planned` row directly from `progress.json` and verify it with
`go run ./cmd/progress validate` before handing it to a builder.

The tool/UX rows that closed the visible "Gormes only shows execute_code"
class of bugs are also complete/validated:

- `Gateway stream/tool trace formatting fixture matrix`
- `Telegram typing action + placeholder lifecycle parity`
- `Gateway active-turn policy manifest closeout`
- `Native TUI Hermes tool progress + modal panel renderers`

## Per-lane backlog summary

Counts in this section are the original 2026-04-29 lane counts. The row state
annotations below have been refreshed from `progress.json` where later builder
passes have already landed; use `progress.json` as the current dispatch queue.

### Lane B — Telegram UX and command parity

Completed:
- _Telegram /status Hermes-format closeout_ (complete/validated).
- _Telegram production live-turn provider payload golden_ (complete/validated).
- _Telegram MarkdownV2 parse-mode rendering closeout_ (complete/validated;
  ParseMode and parse-error fallback are covered by
  `internal/channels/telegram/bot_parse_mode_test.go`).
- _Telegram reply_to_mode and reply-context parity_ (complete/validated).
- _Telegram dynamic BotCommand menu wiring_ (complete/validated).
- _Gateway stream/tool trace formatting fixture matrix_ (complete/validated).
- _Telegram typing action + placeholder lifecycle parity_ (complete/validated).
- _Gateway active-turn policy manifest closeout_ (complete/validated).
- _Gateway slash registry parity sweep (recognized-name expansion)_ is covered
  by the validated active-turn/registry rows; use `progress.json` for any
  remaining command-specific handler ports.

### Lane C — Session/status/title parity

Completed:
- _Gateway conversational session metadata refresh_ (complete/validated).
- _Gateway session token accounting parity_ (complete/validated).
- _Gateway /title manual session title command_ (complete/validated).
- _Hermes memory tool over Goncho/local durable store_ (complete/validated,
  also Lane E).

### Lane D — Prompt/provider identity parity

**Mostly parity already (gateway level). 2 partial areas.**

Completed:
- _Telegram production live-turn provider payload golden_ (complete/validated).
- _Live-turn model/tool guidance wiring_ (complete/validated).

P1:
- _Toolset-aware skills prompt snapshot_ (planned, blocking the
  umbrella `Hermes live-turn prompt assembly parity`).

### Lane E — Memory/Goncho parity

**Missing/partial: 5 areas.**

Completed:
- _Hermes memory tool over Goncho/local durable store_ (complete/validated).
- _Goncho memory provider lifecycle adapter_ (complete/validated).

P1:
- _Long session management_, _Context compression_,
  _Manual compression feedback + context references_,
  _Kernel compression-boundary callback binding_ (all planned).

### Lane F — Provider/auth/runtime parity

**Mostly already complete or row-backed. 6 partial areas.**

P0:
- _Codex_ (planned provider runtime row).

P1:
- _Bedrock provider runtime binding_ (planned).
- _Gemini Cloud Code request/stream mapper_ (planned).
- _OpenRouter compatible-provider routing_ (planned).
- _Google Code Assist project/quota resolver_ (planned).

Completed:
- _Hermes auth OAuth provider adapters_ (complete/validated).
- _Gormes auth bare interactive credential-pool readout_ (complete/validated).

### Lane G — Browser harness/tool parity

**Missing/partial: 12 areas (most depend on the same harness binary).**

P0:
- _go-browser-harness Chromedp action backend_ (planned). Single
  parent for navigation/click/type/scroll/snapshot/screenshot/console.
- **`Go browser harness binary repo + integration lane (placeholder)`**
  (new umbrella). Reserves the integration slot for the future
  sibling Go repo that hosts the `go-browser-harness` binary.

P1:
- _Browser artifact and console render contract_ (planned).
- _Telegram browser artifact rendering_ (planned).

### Lane H — CLI/config parity

**Mostly row-backed. 6 partial areas.**

P1:
- _49-file CLI tree port_ (planned umbrella).
- _Config, profile, auth, and setup command surfaces_ (planned).
- _Diagnostics, backup, logs, and status CLI_ (planned).
- _Gormes config edit/check/native schema-migrate closeout_ (planned).
- _Hermes fallback provider chain CLI commands_ (planned).

## Cross-lane dependencies

- **Lane B MarkdownV2, Lane C session metadata, and Lane D production
  provider-payload identity are complete/validated.** Keep their fixtures as
  regression gates rather than dispatching those rows again.
- **Lane C auto-title wiring and `/title` are complete/validated.** Future
  title work should be treated as UX polish or provider-cost policy, not as a
  blocker for `/status` parity.
- **Lane G `go-browser-harness Chromedp action backend` blocks all
  browser_* runtime behavior**: the contract is shipped but no
  navigation actually fires until that backend lands.
- **Lane G new umbrella (`Go browser harness binary repo`) is a
  no-runtime row** — it documents the integration plan and the JSON
  contract, and unblocks the actual Chromedp backend slice when the
  sibling repo materializes.
- **Lane E `Hermes memory tool over Goncho/local durable store` is
  complete/validated.** Remaining memory work is long-session compression,
  frozen snapshot semantics, and provider lifecycle depth rather than the
  basic model-visible `memory` tool.

## Risks

1. **Tool name parity gap**: `internal/tooltrace` now centralizes the
   Hermes-style tool progress renderer across gateway, TUI, Slack, and
   Discord, and the fixture matrix is complete/validated. The remaining
   risk is configurable skin/tool emoji override parity: Hermes'
   `skin_engine.py` can load `tool_emojis`, while Gormes still uses a fixed
   Go mapping.
2. **Session-ID format change implications**: changing the fallback
   format from `YYYYMMDD_HHMMSS_<fnv>` to `YYYYMMDD_HHMMSS_<random>`
   will invalidate any persisted-but-not-yet-resolved chat -> session
   mapping. The session catalog and mirror outputs must be tested
   for the transition.
3. **Identity injection path drift**: the production provider-payload golden
   is complete/validated. Keep it in the regression set any time
   `cmd/gormes` manager construction, live-turn prompt assembly, or provider
   role mapping changes.
4. **MarkdownV2 fallback strategy**: shipped. When Telegram rejects
   MarkdownV2 parsing, `sendWithParseFallback` retries the byte-identical
   body once with `ParseMode` unset; keep the fallback covered by
   `TestBot_Send_FallsBackToPlainOnMarkdownV2ParseError` as later
   renderer rows change output shape.
5. **Auto-title cost and notification policy**: auto-title wiring is
   complete/validated. Future changes should preserve the auxiliary-failure
   evidence path and avoid adding unbounded background title calls.
6. **Browser harness binary not present**: until the sibling
   `go-browser-harness` repo exists, every browser_* tool will
   degrade to the `ErrBrowserHarnessUnavailable` evidence path. This
   is correct behavior but means Telegram users cannot use any
   browser_* tool until Lane G ships.
7. **Slash registry expansion noise**: the recognized-name expansion is
   complete/validated. Future command ports should keep unavailable commands
   visible but should avoid routing slash text into the model.

## Live Gateway Smoke Recommendation

After touching gateway/TUI/channel UX, Juan should still run these four
Telegram interactions as a regression smoke, even though the original P0 rows
are now complete/validated:

1. **`Whats ur name?`** — confirms identity reaches the provider. The
   bot must reply identifying as Gormes (per SOUL.md), not as
   ChatGPT. This guards the production provider-payload golden.
2. **`/status`** — confirms the field block renders with bold labels
   (validates Lane B #2 MarkdownV2), shows a real LLM-generated Title
   instead of "Telegram conversation with <user_id>" (validates Lane
   C metadata rows), shows accurate Created and Last Activity timestamps in
   Juan's local time, and the bot reply is
   threaded to the `/status` request (already parity but confirm the
   regression hasn't crept back).
3. **`*Send a long answer*` followed by `*reply*`** — confirms reply
   quoting works on the long-final case and that bold/italic/code
   actually render on mobile. Also confirms
   the coalescer's fresh-final-then-delete behavior doesn't break
   reply threading.
4. **A code-block-heavy answer** — e.g. ask for `/help` or any answer
   where the model is likely to emit triple-backticked code. Confirms
   the parse-mode wiring renders code fences correctly and falls back
   gracefully if MarkdownV2 parsing fails.

If any of these fail, classify it as a regression against the completed row
first; create or refine a new row only when the current fixture does not cover
the failing behavior.
