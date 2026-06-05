---
name: gormes-provider-parity
description: Use when fixing or implementing Gormes provider, auth, credential, endpoint, model-routing, streaming, account-usage, rate-limit, retry, or Telegram-visible provider error behavior.
---

# Gormes Provider Parity

## Repository Branch Rule

For Gormes work, stay on the existing `development` branch. Do not create or
use feature branches, short-lived branches, or git worktrees. If the checkout
is not on `development`, stop before editing and switch safely or report the
blocker.

Use this skill whenever Gormes has provider, auth, credential, endpoint, model-routing, streaming, account-usage, rate-limit, retry, or Telegram-visible provider error problems.

Do not use this skill as a catch-all for every "tool calling is bad" report.
Raw iteration-budget errors, duplicate final messages, hourglass/status leaks,
and TUI/tool-progress exposure start in `gormes-hermes-parity` and usually land
in `gormes-tdd-slice` against kernel or channel fixtures. Use this skill when
the evidence points to provider request/response shaping, stream repair,
malformed tool-call payloads, retry/rate-limit handling, or safe provider error
rendering.

## Mission

Make Gormes provider behavior feel like Hermes to existing Hermes users while implementing the runtime natively in Go.

Hermes Agent is the Python upstream/father implementation for Gormes. Prefer
the in-repo checkout at `./hermes-agent`; fall back to `../hermes-agent` only
when absent. Resolve it as `$HERMES_SRC` before reading provider behavior.

Hermes parity is P0:

- same user-facing config semantics and `config.yaml` placement;
- same command/operator expectations;
- same provider selection/fallback behavior unless an explicit progress row justifies a divergence;
- same gateway/Telegram response style and safe error behavior;
- no dependency on Python `hermes-agent` runtime services.

External Go projects are implementation references, not the product contract.

## Mandatory Source Order

1. Read the active logical progress row when one exists. Use `cmd/progress` / `internal/planning/progress`; do not hand-parse split progress layouts.
2. Read Hermes Python for the expected user/operator behavior. Useful anchors:
   - `$HERMES_SRC/AGENTS.md`
   - provider/credential/runtime code under `$HERMES_SRC/agent/`
   - gateway/platform behavior under `$HERMES_SRC/gateway/`
3. Resolve an optional Go donor root before reading donor files:
   ```sh
   DONOR_ROOT="$(for p in ./references/go-agent-os ../references/go-agent-os "$GORMES_GO_AGENT_OS_REFS"; do [ -n "$p" ] && [ -d "$p" ] && { printf '%s\n' "$p"; break; }; done)"
   ```
   If no donor root exists, continue from Hermes + local Gormes code or route to `gormes-references`/`gormes-context-sourcing`; do not assume a stale absolute path.
4. When `DONOR_ROOT` exists, inspect the smallest relevant donor files: GoClaw OAuth/provider/error classification, Plandex retry/rate-limit patterns, or Nanobot/trpc-agent-go/ADK-Go runtime boundaries.
5. Only then write or update Gormes code/tests.

### GoClaw porting recipe (2026-04-29 permission update)

Juan granted Gormes explicit permission to use GoClaw code, not just patterns. The earlier CC BY-NC 4.0 caution is superseded for Gormes' use. When porting GoClaw code into Gormes:

1. Add a provenance comment on the receiving Gormes file naming the source file and function:

   ```go
   // Adapted from goclaw/internal/oauth/openai.go::ExchangeCode
   // Reason: Gormes needed PKCE + pasted-redirect support to match Hermes Codex login UX.
   ```

2. Convert types and imports to Gormes-native names; never let `goclaw_*` symbols leak into Gormes' public surfaces. The donor's package layout is informational only.
3. Apply the Hermes parity filter: GoClaw UX/config choices that diverge from Hermes get adapted, not adopted wholesale.
4. Add Gormes tests covering the ported behavior (do not rely on the donor's tests as proof).
5. Verify `go doc ./<package> | grep -iE "(goclaw|nextlevelbuilder)"` returns nothing before merging.

Other reference repos (`nanobot`, `plandex`, `engram`, `trpc-agent-go`, `adk-go`, `axe`, `agentcontrolplane`, `uzi`) stay patterns-only unless individually authorized — see the donor root README when present for the per-donor permission map. When in doubt about whether a donor file is a pattern source or a code source, consult the `gormes-references` skill before porting.

## Workflow

### 1. Diagnose The Provider Failure

Gather evidence before editing:

```sh
git status --short --branch
go run ./cmd/gormes gateway status || true
go run ./cmd/gormes --model <model> --provider <provider> chat -q 'health probe' || true
```

Classify the failure:

- configuration/endpoint missing;
- credential missing;
- token expired / relogin required;
- forbidden / entitlement / workspace mismatch;
- rate-limited / Retry-After;
- provider unavailable / network timeout;
- model/provider drift;
- unsafe raw provider error leaking to Telegram;
- runtime bug such as constructing a relative `/v1/responses` URL.
- provider stream/tool-call repair failure, as distinct from a kernel
  iteration-budget or channel duplicate-send bug.

### 2. Preserve Hermes UX Before Choosing Implementation

State the Hermes-compatible behavior first:

```text
Hermes parity target:
Go donor pattern:
Gormes-native seam:
Telegram/operator response shape:
```

If GoClaw or another donor does something differently from Hermes UX, adapt the implementation idea but keep Hermes behavior.

### 3. Use Donor Patterns Deliberately

High-value GoClaw patterns for provider work:

- OAuth PKCE with verifier/state and pasted redirect URL support.
- Token exchange/refresh with bounded HTTP timeout.
- Provider/credential resolver interfaces for hermetic tests.
- User-actionable error classes: `reauth_required`, `payment_required`, `quota_api_forbidden`, `quota_endpoint_not_found`, `rate_limited`, `provider_unavailable`, `network_timeout`, `network_error`.
- Sanitize/truncate HTML or raw upstream bodies before Telegram/operator display.

High-value Plandex patterns:

- Provider/model drift detection.
- Retry/backoff and `Retry-After` handling.
- Preflight provider settings before starting a turn.

High-value runtime-boundary patterns from Nanobot/trpc-agent-go/ADK-Go:

- Narrow provider interfaces with fake clients in tests.
- Cancellable/session-scoped workers.
- Tool/MCP runtime separated from provider transport.

### 4. TDD Requirements

Use `gormes-tdd-slice` for implementation.

Write RED tests with:

- temp `HERMES_HOME` / auth store;
- neutral placeholder tokens;
- `httptest` provider endpoint;
- no live credentials;
- Telegram-safe error assertions when the failure is channel-visible.

Required test classes for provider fixes:

- explicit endpoint/proxy remains supported;
- Codex auth import uses synthetic `CODEX_HOME/auth.json` fixtures: fresh
  Codex CLI tokens import into Gormes' own pool, expired tokens fall back to
  device-code, and logs/status output redacts tokens and absolute auth paths;
- no implicit localhost or relative `/v1/responses` URL;
- credential-pool OAuth entries can supply base URL and bearer token when Hermes parity expects it;
- missing/stale/forbidden/rate-limited provider failures surface structured safe evidence, not raw HTML or secret-bearing text;
- gateway active turn is cleared after provider setup/stream failure so Telegram does not wedge on `admission: still processing previous turn`.

### 5. Validation

Minimum validation for provider/auth/runtime changes:

```sh
go test ./cmd/gormes ./internal/config ./internal/llm ./internal/runtime ./internal/gateway -count=1
go run ./cmd/progress validate
git diff --check
```

If the change affects Telegram-visible behavior, also source-run the gateway and verify:

```sh
go run ./cmd/gormes gateway status
```

Use live provider calls only as final smoke tests, never as the only proof.

## Non-Negotiables

- Hermes parity is P0.
- GoClaw and other references are useful, but Gormes must not become GoClaw-shaped if that breaks Hermes operator experience.
- Do not print or commit provider tokens.
- Do not hide provider/auth blockers behind generic `unknown` errors.
- Do not let Telegram show raw HTML or secret-bearing upstream errors.
- Do not leave failed provider turns wedged behind `admission: still processing previous turn`.

## Final Report Shape

```text
Provider issue:
Hermes parity target:
References inspected:
Implementation/tests:
Validation:
Telegram/operator impact:
Remaining provider blockers:
```
