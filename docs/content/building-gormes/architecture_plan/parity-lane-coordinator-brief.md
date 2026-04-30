---
title: "Parity lane coordinator brief"
description: "Dispatch order and per-lane backlog summary derived from the Hermes <-> Gormes parity matrix. Builders consult this to pick the next slice without waiting for an operator screenshot."
date: 2026-04-29
draft: false
---

# Parity lane coordinator brief

Companion to
[hermes-gormes-parity-matrix](../hermes-gormes-parity-matrix/).

This brief is the dispatch surface for the parity operating system.
Each lane (B-H) corresponds to one builder workflow that closes a
specific class of parity gap.

## Top 3 P0 slices recommended for immediate dispatch

These are the highest-leverage slices for immediate planner/builder
dispatch. `Telegram MarkdownV2 parse-mode rendering closeout` used to
be in this trio but is now complete/validated in `progress.json`, with
wire-level ParseMode and fallback fixtures in
`internal/channels/telegram/bot_parse_mode_test.go`.

1. **`Telegram production live-turn provider payload golden`** (Lane D,
   priority P0).
   - Why: closes the dogfood identity regression at the actual
     production binary. The gateway-level integration test
     (`internal/gateway/live_turn_prompt_test.go::TestLiveTurn_TelegramFinalProviderRequestIncludesOperatorContext`)
     already passes; the production-level golden at
     `cmd/gormes/telegram.go::telegramManagerConfig` is what proves
     real `gormes telegram` users get Gormes identity in the system
     prompt.
   - Fixture: `cmd/gormes/telegram_test.go` +
     `internal/gateway/live_turn_prompt_test.go`.
   - Test command:
     `GOCACHE=/tmp/gormes-go-cache go test ./cmd/gormes ./internal/gateway -run 'Telegram.*ProviderPayload|LiveTurn' -count=1`.

2. **`Telegram /status Hermes-format closeout`** (Lane B+C, priority
   P0).
   - Why: visible-everywhere chat command. Today's output already
     reply-quotes the triggering message and bypasses the model, and
     the completed MarkdownV2 parse-mode row now lets bold labels
     render. The remaining closeout is status content: a real
     LLM-generated Title and session metadata instead of synthetic
     title fallback.
   - Fixture: `internal/gateway/status_command_test.go` +
     `internal/channels/telegram/bot_test.go`.
   - Test command:
     `GOCACHE=/tmp/gormes-go-cache go test ./internal/gateway ./internal/channels/telegram -run 'Status|Title|Telegram|Reply' -count=1`.

3. **`Gateway session token accounting parity`** (Lane C, priority P0).
   - Why: `/status` should report durable session token totals even
     after the live render frame is stale; the planned row persists
     provider usage into session metadata instead of relying only on
     the manager's last in-memory frame.
   - Fixture: `internal/gateway/status_command_test.go` + session
     metadata fixtures.
   - Test command:
     `GOCACHE=/tmp/gormes-go-cache go test ./internal/gateway -run 'Status|Token|Usage' -count=1`.

The fourth complementary slice (not in the P0 trio because it's an
inventory, not a behavior change) is the parity matrix itself, which
this coordinator brief and `hermes-gormes-parity-matrix.md` deliver.

## Per-lane backlog summary

Counts derived from the matrix doc's classifications. Italicized
entries are existing planned rows; bold entries need planner refinement
if they are not already represented in `progress.json`.

### Lane B — Telegram UX and command parity

**Missing/regressed/partial: 8 areas.**

P0:
- _Telegram /status Hermes-format closeout_ (planned; unblocked by the
  completed MarkdownV2 parse-mode row).
- _Telegram production live-turn provider payload golden_ (planned).

Completed P0:
- _Telegram MarkdownV2 parse-mode rendering closeout_ (complete/validated;
  ParseMode and parse-error fallback are covered by
  `internal/channels/telegram/bot_parse_mode_test.go`).

P1:
- _Telegram reply_to_mode and reply-context parity_ (planned). Covers
  reply-mode toggle, deleted-target fallback, and message_thread_id
  for forum topics.
- _Telegram dynamic BotCommand menu wiring_ (planned).
- _Gateway stream/tool trace formatting fixture matrix_ (planned).
- _Telegram typing action + placeholder lifecycle parity_ (planned).
- _Gateway active-turn policy manifest closeout_ (planned).

P2:
- **`Gateway slash registry parity sweep (recognized-name expansion)`**
  (planner refinement). Mostly mechanical — surface every Hermes-registered command
  in `CommandRegistry` so unknown-command responses get quieter.

### Lane C — Session/status/title parity

**Missing/regressed/partial: 7 areas.**

P0:
- _Gateway conversational session metadata refresh_ (planned). Single
  parent slice for: created/last-activity timestamps, title
  persistence, fallback session ID format. Title auto-generation
  wiring (`hermes.GenerateTitle` is unwired today) should be a child
  slice here.
- _Gateway session token accounting parity_ (planned).

P1:
- _Hermes memory tool over Goncho/local durable store_ (planned, also
  Lane E).

### Lane D — Prompt/provider identity parity

**Mostly parity already (gateway level). 2 partial areas.**

P0:
- _Telegram production live-turn provider payload golden_ (planned).
- _Live-turn model/tool guidance wiring_ (planned). Wires the existing
  `internal/hermes/prompt_guidance.go` constants into the live-turn
  assembly site.

P1:
- _Toolset-aware skills prompt snapshot_ (planned, blocking the
  umbrella `Hermes live-turn prompt assembly parity`).

### Lane E — Memory/Goncho parity

**Missing/partial: 5 areas.**

P0:
- _Hermes memory tool over Goncho/local durable store_ (planned).

P1:
- _Goncho memory provider lifecycle adapter_ (planned).
- _Long session management_, _Context compression_,
  _Manual compression feedback + context references_,
  _Kernel compression-boundary callback binding_ (all planned).

### Lane F — Provider/auth/runtime parity

**Mostly already complete or row-backed. 6 partial areas.**

P0:
- _Codex_ (planned provider runtime row).
- _Hermes auth OAuth provider adapters_ (planned).

P1:
- _Bedrock provider runtime binding_ (planned).
- _Gemini Cloud Code request/stream mapper_ (planned).
- _OpenRouter compatible-provider routing_ (planned).
- _Google Code Assist project/quota resolver_ (planned).
- _Gormes auth bare interactive credential-pool readout_ (planned).

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

P0:
- _Hermes auth OAuth provider adapters_ (planned).

P1:
- _49-file CLI tree port_ (planned umbrella).
- _Config, profile, auth, and setup command surfaces_ (planned).
- _Diagnostics, backup, logs, and status CLI_ (planned).
- _Gormes config edit/check/native schema-migrate closeout_ (planned).
- _Hermes fallback provider chain CLI commands_ (planned).

## Cross-lane dependencies

- **Lane B #2 (`Telegram MarkdownV2 parse-mode rendering closeout`)
  is complete and unblocks Lane B #3 (`Telegram /status Hermes-format
  closeout`):** the /status Hermes target uses `**Field:**` bold labels;
  `internal/channels/telegram/bot_parse_mode_test.go` now proves
  MarkdownV2 ParseMode and parse-error fallback on send/reply/edit paths.
- **Lane D #1 (`Telegram production live-turn provider payload golden`)
  unblocks no other slice but proves the full chain end-to-end.** The
  gateway-level test passes; the production binary test does not yet
  exist.
- **Lane C `Gateway conversational session metadata refresh` blocks
  the Title field in `/status`**: until session metadata is refreshed
  every turn and title generation is wired, /status will still render
  a synthetic title. The /status closeout should depend on this row.
- **Lane G `go-browser-harness Chromedp action backend` blocks all
  browser_* runtime behavior**: the contract is shipped but no
  navigation actually fires until that backend lands.
- **Lane G new umbrella (`Go browser harness binary repo`) is a
  no-runtime row** — it documents the integration plan and the JSON
  contract, and unblocks the actual Chromedp backend slice when the
  sibling repo materializes.
- **Lane E `Hermes memory tool over Goncho/local durable store`
  blocks the durable-memory write side of the live turn**: today the
  live-turn prompt reads MEMORY.md but writes go nowhere.

## Risks

1. **Tool name parity gap**: `internal/gateway/render.go::toolEmoji`
   hardcodes `🧠` for memory but Hermes' memory tool name is
   `memory_flush` / `memory` depending on the variant. If the
   provider returns a tool name Gormes doesn't recognize (e.g.
   `memory_save`), the trace falls back to `🔧`. Low-impact, but
   worth a fixture matrix as the planned row promises.
2. **Session-ID format change implications**: changing the fallback
   format from `YYYYMMDD_HHMMSS_<fnv>` to `YYYYMMDD_HHMMSS_<random>`
   will invalidate any persisted-but-not-yet-resolved chat -> session
   mapping. The session catalog and mirror outputs must be tested
   for the transition.
3. **Identity injection paths that don't reach the live provider**:
   the gateway-level integration test passes, but the actual
   production `gormes telegram` entrypoint at
   `cmd/gormes/telegram.go::telegramManagerConfig` may construct the
   manager with different seam defaults than the test. The Lane D #1
   slice exists specifically to close this risk.
4. **MarkdownV2 fallback strategy**: shipped. When Telegram rejects
   MarkdownV2 parsing, `sendWithParseFallback` retries the byte-identical
   body once with `ParseMode` unset; keep the fallback covered by
   `TestBot_Send_FallsBackToPlainOnMarkdownV2ParseError` as later
   renderer rows change output shape.
5. **Auto-title side effects**: wiring `hermes.GenerateTitle` into the
   gateway live turn means an extra provider call per session, which
   will increase token spend and may produce a delayed title-update
   message that Juan can't reply-thread to. The slice must define
   when (after first complete turn?) and how (background goroutine
   with admission gate?) the title is generated.
6. **Browser harness binary not present**: until the sibling
   `go-browser-harness` repo exists, every browser_* tool will
   degrade to the `ErrBrowserHarnessUnavailable` evidence path. This
   is correct behavior but means Telegram users cannot use any
   browser_* tool until Lane G ships.
7. **Slash registry expansion noise**: surfacing all ~30 missing
   Hermes commands as recognized-but-unavailable will increase the
   "/<cmd> is recognized but unavailable" reply rate. Acceptable
   trade-off — friendlier than "unknown command" — but worth noting.

## Pre-Phase-2 live gateway smoke recommendation

After lanes B/C/D ship the P0 trio plus
`Gateway conversational session metadata refresh`, Juan should run
these four Telegram interactions to confirm dogfood parity:

1. **`Whats ur name?`** — confirms identity reaches the provider. The
   bot must reply identifying as Gormes (per SOUL.md), not as
   ChatGPT. This validates Lane D #1.
2. **`/status`** — confirms the field block renders with bold labels
   (validates Lane B #2 MarkdownV2), shows a real LLM-generated Title
   instead of "Telegram conversation with <user_id>" (validates Lane
   C #1), shows accurate Created and Last Activity timestamps in
   Juan's local time (validates Lane C #1), and the bot reply is
   threaded to the `/status` request (already parity but confirm the
   regression hasn't crept back).
3. **`*Send a long answer*` followed by `*reply*` — confirms reply
   quoting works on the long-final case and that bold/italic/code
   actually render on mobile (validates Lane B #2). Also confirms
   the coalescer's fresh-final-then-delete behavior doesn't break
   reply threading.
4. **A code-block-heavy answer** — e.g. ask for `/help` or any answer
   where the model is likely to emit triple-backticked code. Confirms
   the parse-mode wiring renders code fences correctly and falls back
   gracefully if MarkdownV2 parsing fails.

If any of these fail, dispatch the relevant lane row before declaring
Phase 2 closed. Each smoke step maps directly to one of the P0 slices
above.
