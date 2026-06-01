# Gormes Full Test Suite

Gormes uses a test pyramid. E2E tests prove the most important user journeys,
but most confidence should come from faster unit, contract, and integration
checks that run with fake providers and fake gateways.

## Layers

| Layer | Purpose | Examples | CI default |
|---|---|---|---|
| Unit | Pure logic and small command behavior. | config parsing, provider routing decisions, memory ranking, skill registry, flags, token redaction. | Yes, via `make test`. |
| Contract | Stable subsystem boundaries. | provider interface, tool schemas, gateway message shape, Navivox descriptor, installer output contract. | Yes, via `make test` and `make test-integration`. |
| Integration | Real components together with fake edges. | SQLite memory/session restore, fake OpenAI server + chat/router, fake webhook + gateway path, dashboard health, migration dry-run. | Yes, bounded by `make test-integration`. |
| E2E | User-visible journeys. | install/fresh-home smoke, doctor, setup, fake-provider chat, offline TUI, gateway status, Navivox pair, Termux install smoke. | Yes for offline/fake flows via `make test-e2e`; live/provider tests are manual. |
| Release | Publishing invariants. | all-target build, version/tag consistency, install asset naming, checksums/SBOM, `go install` viability, docs/install command sync. | Yes before release via `make test-release`. |

## Commands

```sh
make test
make test-integration
make test-e2e
make test-release
```

`make test` is the broad Go suite and must not require real provider tokens.
`make test-integration` exercises real subsystem seams with fake external
services. `make test-e2e` is limited to high-value offline journeys so it stays
fast enough for CI. `make test-release` is the pre-publish gate and may build
binaries, inspect installer contracts, and verify release-facing metadata.

## Fake CI dependencies

CI must use fake external edges:

- Fake OpenAI-compatible provider: `internal/support/testutil/fakeopenai`.
  It implements `/v1/models` and `/v1/chat/completions` for normal and streamed
  responses.
- Fake gateway webhook server: `internal/support/testutil/fakegateway`.
  It records webhook events and exposes `/health`, `/reload`, and `/logs`.
- Fixture Gormes home: `internal/support/testutil/gormeshome`.
  It isolates `GORMES_HOME`, `HERMES_HOME`, `CODEX_HOME`, and XDG paths.

Real OpenAI/Anthropic/Nous/OpenRouter/Ollama credentials are manual-only. A test
that needs one must be behind an explicit live/manual gate and skipped by
default.

## Coverage topics

Every Hermes feature family should have at least one fast contract/integration
check before it gets a broad E2E journey:

| Topic | Fast layer | Wide/user layer |
|---|---|---|
| Providers/chat | Fake OpenAI-compatible provider contract and router tests. | Setup fake key/model, then fake-provider chat or in-process LLM E2E. |
| Gateway | Command registry, fake webhook, status/reload/logs contracts. | Gateway status/probe/logs journey with no live platform tokens. |
| Channels | Per-adapter formatting/capability tests with fake transports. | Channel capability matrix and selected fake webhook delivery. |
| Tools | Tool descriptor/schema validation and execution fixtures. | Chat/tool-loop E2E with mock tool calls and clean final output. |
| Skills | Registry/sync tests and fixture skill roots. | Bootstrap journey verifies skill inventory and forced-skill chat behavior. |
| Memory/sessions | SQLite persistence and recall ranking tests. | Restart-style E2E proves sessions/memory survive reopening. |
| Learning loop/curator | Learning signal and curator state tests. | Curator status/dry-run in the fresh-user bootstrap journey. |
| Navivox | Descriptor/token contract tests. | Pairing command emits QR/token/URL-safe connection details. |
| Dashboard/API | HTTP handler auth/health contracts. | Dashboard health/status smoke with ephemeral token. |
| Installer/release | Installer plan/output and asset/checksum contracts. | Sandbox install/uninstall dry-run only; no production state writes. |

## Required E2E scenarios

| ID | Scenario | CI shape |
|---|---|---|
| E2E-001 | Offline install/fresh-home smoke. | Fresh `GORMES_HOME`, command tree and read-only inventory probes. |
| E2E-002 | Doctor without credentials. | `gormes doctor --offline --target terminal --json`. |
| E2E-003 | Setup provider with fake key. | `config set hermes.model`, `config set hermes.api_key`; assert redaction. |
| E2E-004 | Fake OpenAI chat turn. | Fake OpenAI-compatible server or in-process LLM mock; no real tokens. |
| E2E-005 | SQLite memory persists after restart. | Temp DB, write turn, reopen, recall/session list. |
| E2E-006 | Gateway status/reload/logs. | Fake gateway/webhook/status endpoints; no live Telegram/Slack. |
| E2E-007 | Navivox pairing descriptor. | Deterministic descriptor asserts URL/token/QR-safe fields. |
| E2E-008 | Migration dry-run redacts secrets. | Fixture Hermes/OpenClaw homes with synthetic secrets. |
| E2E-009 | Dashboard health endpoints. | Local HTTP handler/server with auth boundary assertions. |
| E2E-010 | Installer uninstall dry-run. | Sandbox install home/bin dirs; no production state writes. |

## Golden outputs

Golden files live under `testdata/golden/` for stable release-facing surfaces:

- `version_json.golden` — required JSON keys and shape for `gormes version --json`.
- `navivox_pair_descriptor.golden` — required Navivox descriptor fields.
- `installer_plan.golden` — installer dry-run output contract fragments.

Prefer fragment/shape goldens over exact timestamps, ports, temp paths, or build
hashes. Exact goldens are acceptable only when the output is deterministic.

## Manual-only tests

Manual tests may use real providers, real messaging platforms, or real device
pairing. They must be opt-in and documented with required environment variables.
Never let CI discover credentials implicitly from a developer machine.

## Rule of thumb

If a behavior can be proven with a unit, contract, or integration test, do that
first. Add E2E only for the user journeys where the wiring itself is the product.
