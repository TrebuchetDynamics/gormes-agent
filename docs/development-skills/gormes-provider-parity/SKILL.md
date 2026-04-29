---
name: gormes-provider-parity
description: Use when fixing or implementing Gormes provider, auth, credential, endpoint, model-routing, streaming, account-usage, rate-limit, retry, or Telegram-visible provider error behavior.
---

# Gormes Provider Parity

Use this skill whenever Gormes has provider, auth, credential, endpoint, model-routing, streaming, account-usage, rate-limit, retry, or Telegram-visible provider error problems.

## Mission

Make Gormes provider behavior feel like Hermes to existing Hermes users while implementing the runtime natively in Go.

Hermes parity is P0:

- same user-facing config semantics and `config.yaml` placement;
- same command/operator expectations;
- same provider selection/fallback behavior unless an explicit progress row justifies a divergence;
- same gateway/Telegram response style and safe error behavior;
- no dependency on Python `hermes-agent` runtime services.

External Go projects are implementation references, not the product contract.

## Mandatory Source Order

1. Read the active Gormes row in `docs/content/building-gormes/architecture_plan/progress.json` when one exists.
2. Read Hermes Python for the expected user/operator behavior. Useful anchors:
   - `/home/xel/git/sages-openclaw/workspace-mineru/hermes-agent/AGENTS.md`
   - provider/credential/runtime code under `/home/xel/git/sages-openclaw/workspace-mineru/hermes-agent/agent/`
   - gateway/platform behavior under `/home/xel/git/sages-openclaw/workspace-mineru/hermes-agent/gateway/`
3. Read the provider reference note:
   - `/home/xel/git/sages-openclaw/workspace-mineru/references/go-agent-os/GORMES-PROVIDER-PATTERN-REFERENCES.md`
4. Inspect local Go donor implementations for the specific problem:
   - GoClaw OAuth/provider/error classification:
     - `/home/xel/git/sages-openclaw/workspace-mineru/references/go-agent-os/goclaw/internal/oauth/openai.go`
     - `/home/xel/git/sages-openclaw/workspace-mineru/references/go-agent-os/goclaw/internal/oauth/openai_quota_transport.go`
     - `/home/xel/git/sages-openclaw/workspace-mineru/references/go-agent-os/goclaw/internal/providers/`
   - Plandex provider retry/drift/rate-limit patterns:
     - `/home/xel/git/sages-openclaw/workspace-mineru/references/go-agent-os/plandex`
   - Nanobot/trpc-agent-go/ADK-Go runtime boundaries:
     - `/home/xel/git/sages-openclaw/workspace-mineru/references/go-agent-os/nanobot`
     - `/home/xel/git/sages-openclaw/workspace-mineru/references/go-agent-os/trpc-agent-go`
     - `/home/xel/git/sages-openclaw/workspace-mineru/references/go-agent-os/adk-go`
5. Only then write or update Gormes code/tests.

Juan has stated that GoClaw is permitted as a reference source. Still keep provenance clear: use it to learn and adapt patterns; do not blindly copy UX/config choices that would break Hermes parity.

## Workflow

### 1. Diagnose The Provider Failure

Gather evidence before editing:

```sh
git status --short --branch
go run ./cmd/gormes gateway status || true
go run ./cmd/gormes -z 'health probe' --model <model> --provider <provider> || true
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
- no implicit localhost or relative `/v1/responses` URL;
- credential-pool OAuth entries can supply base URL and bearer token when Hermes parity expects it;
- missing/stale/forbidden/rate-limited provider failures surface structured safe evidence, not raw HTML or secret-bearing text;
- gateway active turn is cleared after provider setup/stream failure so Telegram does not wedge on `admission: still processing previous turn`.

### 5. Validation

Minimum validation for provider/auth/runtime changes:

```sh
go test ./cmd/gormes ./internal/config ./internal/hermes ./internal/runtime ./internal/gateway -count=1
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
