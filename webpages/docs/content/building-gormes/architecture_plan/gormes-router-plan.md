# Gormes Router Plan

## Goal

Add a Go-native **Gormes Router**: one local OpenAI-compatible endpoint for the LLM providers the operator has already configured in Gormes.

Framing: **Gormes Router = one local endpoint for your configured LLM providers.** It is not a free-model bypass, not account automation, and not an OAuth harvester.

## Source context

External reference inspected: `router-for-me/CLIProxyAPI` at commit `50d19e2`.

Useful CLIProxyAPI evidence:

- `README.md` describes a Go proxy exposing OpenAI/Gemini/Claude/Codex/Grok-compatible APIs, streaming/non-streaming/WebSocket responses, function/tool calling, multimodal input, multi-account routing, OpenAI-compatible upstream providers, and provider-specific route aliases such as `/api/provider/{provider}/v1/chat/completions`.
- `config.example.yaml` shows top-level `api-keys`, `request-retry`, cooldown controls, session affinity, OpenAI-compatible providers with `base-url`, per-provider headers, API-key entries, model aliases, and fallback-style alias pools.
- `docs/sdk-usage.md` exposes an embeddable Go SDK (`cliproxy.NewBuilder`) with config/auth watching, route hooks, request logging, and core auth manager execution.
- `docs/sdk-access.md` documents inbound access providers: `Authorization: Bearer`, `X-Goog-Api-Key`, `X-Api-Key`, `?key=`, and `?auth_token=`; it also defines no-credentials/invalid/internal auth semantics.
- `examples/custom-provider/main.go` demonstrates executor registration, request translation, bearer injection, streaming chunks, and model registration.
- `LICENSE` is MIT, but v1 Gormes Router should not copy code. Use architecture lessons only unless a later implementation pass adds attribution review.

Gormes source context:

- `internal/hermes/http_client.go` already has OpenAI-compatible and Codex Responses client behavior.
- `internal/hermes/provider_transport.go` separates request-shape transports (`chat_completions`, `anthropic_messages`, `bedrock_converse`, `codex_responses`) from HTTP client construction.
- `internal/hermes/provider_registry_manifest.go` lists provider IDs, aliases, auth types, base URLs, and implementation status.
- `internal/hermes/fallback_chain.go` and `internal/hermes/model_routing.go` already provide fallback/routing primitives.
- `cmd/gormes/provider_client.go` resolves configured provider endpoints/credentials and redacts setup errors.

## Non-goals and safety boundary

MVP explicitly excludes:

- OAuth automation, browser token scraping, or account acquisition.
- Multi-account abuse patterns, subscription sharing, or “free unlimited LLM” claims.
- Requiring Ollama, LM Studio, or any local model runtime to be installed.
- Importing or embedding CLIProxyAPI wholesale.
- CLIProxyAPI management API compatibility.
- WebSocket `/v1/responses/ws` support.
- Claude/Gemini/Codex OAuth proxying beyond credentials already configured by Gormes provider/auth flows.
- External network tests or live provider credentials as acceptance proof.

MVP only uses user-owned credentials that already exist in Gormes config, env, or credential pools. If no usable fallback route is configured, the router should say so clearly instead of pretending a free/local model is available.

## Recommended architecture

### Package boundary

Create `internal/provider/router` as a deep service package. It should hide provider selection, request shaping, fallback, counters, and HTTP rendering behind a small server-facing API.

Proposed public-ish internal interfaces:

```go
type Config struct {
    Enabled bool
    Listen string
    APIKeys []string
    DefaultRoute Route
    Routes []Route
    Fallback []Route
    RedactLogs bool
}

type Route struct {
    Name string
    Provider string
    Model string
    BaseURL string
    APIKeyRef string
    APIKeyEnv string
    Transport string // chat_completions first; future: responses, anthropic_messages
    Weight int
}

type Provider interface {
    Models(ctx context.Context) ([]Model, error)
    Chat(ctx context.Context, req ChatRequest) (ChatStream, error)
    Health(ctx context.Context) Health
}

type Registry interface {
    Resolve(model string) ([]Route, error)
    Models(ctx context.Context) ([]Model, error)
}

type Counters interface {
    RecordAttempt(Attempt)
    Snapshot() Status
}
```

`cmd/gormes router` should only wire config, instantiate the router service, and run HTTP. It should not own routing logic.

### Setup UX and provider-picker boundary

`gormes setup` should manage Router as a **local serving feature**, not as a normal upstream provider.

Recommended command split:

- `gormes setup provider` — configure upstream providers and user-owned credentials as it does today.
- `gormes setup fallback` — choose fallback order from configured/healthy upstream routes.
- `gormes setup router` — enable the local OpenAI-compatible endpoint, choose listen address, create/select an inbound local API key, pick route aliases, and optionally attach fallback routes.
- `gormes router status` — show redacted health and copy-paste client config for external tools.

Do **not** put `gormes-router` in the normal provider picker by default. That picker chooses the upstream model Gormes should call. If Router appeared there, users could accidentally configure Gormes to call itself.

Allowed special handling:

- Expose Router as a synthetic **client endpoint** in setup output: “Point OpenAI-compatible tools at `http://127.0.0.1:8787/v1`.”
- Permit an advanced explicit provider entry such as `custom` + `http://127.0.0.1:8787/v1` only with recursion detection: if Gormes Router receives a request whose selected upstream is itself, fail fast with `router_recursion_detected`.
- In `gormes setup router`, suggest fallback candidates from configured providers, not from hard-coded defaults. Free-tier candidates must be labelled “requires your provider account/API key; quota controlled by provider.” Local endpoint candidates must be labelled “optional; only enabled if already installed and healthy.”

First-run flow should not force Router. A good path is:

1. `gormes setup provider` configures at least one usable upstream or records that no provider is configured yet.
2. `gormes setup fallback` is optional and can suggest free-tier/provider-controlled or local routes only when evidence exists.
3. `gormes setup router` is offered when the user wants OpenAI-compatible tools to use Gormes as a local gateway.

### HTTP surface

MVP endpoints:

- `GET /healthz` — local process health, no provider secrets.
- `GET /v1/models` — OpenAI-compatible model list derived from configured routes and model aliases.
- `POST /v1/chat/completions` — OpenAI-compatible chat-completions, including `stream: true` SSE.
- `GET /v1/status` (or `/router/status`) — Gormes-owned redacted status with route health, attempt counts, fallback counts, token/usage counters when available, and last error class.
- `GET /router/client-config` — optional Gormes-owned helper returning redacted copy-paste config for OpenAI-compatible clients, e.g. base URL and header name, never the secret value unless the operator explicitly asks to print/regenerate a local key.

Future optional endpoint:

- `POST /api/provider/{provider}/v1/chat/completions` — only if strict backend pinning is needed after MVP.

### Config schema

Add an opt-in `[router]` section. Keep secrets as env refs or existing credentials.

```toml
[router]
enabled = true
listen = "127.0.0.1:8787"
api_keys = ["dev-local-key"] # optional; dev-only inline, prefer api_key_env/secret refs later
api_key_env = "GORMES_ROUTER_API_KEY"
redact_logs = true
setup_mode = "local_gateway" # not an upstream provider

[[router.routes]]
name = "primary-openai-compatible"
provider = "custom"
model = "gpt-4.1-mini"
alias = "primary-chat"
base_url = "https://api.openai.com/v1"
api_key_env = "OPENAI_API_KEY"
transport = "chat_completions"

[[router.routes]]
name = "openrouter-free-tier-candidate"
provider = "openrouter"
model = "moonshotai/kimi-k2:free"
alias = "budget-chat"
base_url = "https://openrouter.ai/api/v1"
api_key_env = "OPENROUTER_API_KEY"
transport = "chat_completions"
optional = true # requires the user's OpenRouter key; availability/quotas are provider-controlled

[[router.fallback]]
from = "primary-chat"
to = "budget-chat"
on = ["rate_limit", "server_error", "timeout"]
```

Optional local routes use the same schema but are never assumed:

```toml
[[router.routes]]
name = "local-openai-compatible"
provider = "custom"
model = "llama3.2"
alias = "local-chat"
base_url = "http://127.0.0.1:11434/v1"
transport = "chat_completions"
optional = true # enabled only when the endpoint responds during health checks
```

MVP route candidates are configured and health-gated, not hard-coded:

1. The user's primary configured provider/model.
2. User-configured OpenAI-compatible providers (`custom`, including CLIProxyAPI later as a normal upstream).
3. User-configured provider free-tier candidates such as OpenRouter, Groq, or Gemini when the user supplies credentials and accepts provider-controlled quotas.
4. Optional local OpenAI-compatible endpoints such as Ollama or LM Studio only when the endpoint is configured and healthy.

CLIProxyAPI itself should be a later optional upstream route, configured like any other OpenAI-compatible base URL. Do not special-case it until Gormes has the local router MVP.

### Fallback and health

- Retry/fallback only on classified 429, 408, 500, 502, 503, 504, timeout, or connection failures before output starts.
- Do not fallback on auth failures, policy blocks, malformed requests, or after partial streamed output has been emitted.
- Do not install, start, or assume Ollama/LM Studio. Local routes are just OpenAI-compatible URLs that become `available` only after a bounded health probe succeeds.
- Health checks are shallow: configured, credential-present, optional `/v1/models` probe through fakeable client.
- Free-tier routes are status-labelled as `provider_controlled_quota`; they are usable only with user-owned credentials and should degrade to `quota_exhausted`, `missing_credential`, or `unavailable` instead of promising free capacity.
- Counters are in-memory for MVP: attempts, successes, failures by kind, fallback count, last error class, last success timestamp. Persist later only if needed.

### Logging/redaction

- Never log raw API keys, Authorization headers, query credentials, or full upstream error bodies.
- Log route name, provider, model alias, status class, latency, and counter deltas.
- Status responses should show `configured`, `missing_credential`, `unhealthy`, or `available` evidence, not secrets.

## Interface designs considered

### Option A — thin reverse proxy

Forward OpenAI-compatible requests to a selected upstream URL with minimal transformation.

Pros: fastest. Cons: shallow, hard to test fallback before output, cannot reuse Gormes provider transports and error classifiers cleanly.

### Option B — deep Gormes-native router (recommended)

Parse OpenAI-compatible inbound requests into Gormes/hermes request types, route through fakeable provider clients/transports, and render OpenAI-compatible responses/SSE.

Pros: reuses existing provider registry, transport, classifier, fallback, redaction, fake-provider tests. Keeps CLIProxyAPI as context, not dependency. Cons: more upfront mapping work.

### Option C — embed CLIProxyAPI SDK

Use `cliproxy.NewBuilder` directly.

Pros: many features immediately. Cons: imports a second router/runtime, config model, auth watcher, management API, OAuth/multi-account behavior, and larger dependency surface; conflicts with the Gormes single-binary/provider architecture. Not MVP.

Recommendation: **Option B**.

## Test plan

Use fake providers and `httptest`; no live credentials.

Required fixtures:

- `TestRouterModelsListsConfiguredAliases` — `/v1/models` returns configured aliases and redacted provider metadata.
- `TestRouterChatCompletionsNonStreaming` — fake OpenAI-compatible provider response maps to OpenAI chat-completion JSON.
- `TestRouterChatCompletionsStreamingSSE` — `stream: true` emits valid `data:` chunks and `[DONE]`.
- `TestRouterFallbackOnRateLimitBeforeOutput` — primary 429 falls through to fallback route and records evidence.
- `TestRouterDoesNotFallbackAfterStreamStarted` — partial output followed by failure closes with safe evidence; no second provider starts.
- `TestRouterRejectsMissingOrBadAPIKey` — local inbound auth accepts configured bearer/key sources and rejects bad keys without leaking expected keys.
- `TestRouterStatusRedactsSecrets` — `/v1/status` includes route/counter/health data and excludes API keys, Authorization headers, and raw upstream bodies.
- `TestRouterHealthUsesFakeProbe` — provider health checks are fakeable and bounded.
- `TestRouterDoesNotAssumeLocalRuntime` — an absent Ollama/LM Studio endpoint is reported as optional/unavailable and does not block cloud or custom routes.
- `TestRouterFreeTierCandidateRequiresCredential` — free-tier-labelled routes require user-owned credentials and show provider-controlled quota status, not free/unlimited claims.
- `TestSetupRouterDoesNotPolluteProviderPicker` — `gormes setup router` writes router config while `gormes setup provider` still lists upstream providers, not `gormes-router` as a default provider choice.
- `TestRouterRecursionDetected` — an explicit custom route pointing back at the same router endpoint fails fast with redacted `router_recursion_detected` evidence.

Suggested validation for implementation rows:

```sh
go test ./internal/provider/router -count=1
go test ./cmd/gormes -run 'TestRouter' -count=1
go run ./cmd/progress validate
git diff --check
```

## MVP task rows

Planned in `progress.json` as separate provider-module slices:

1. `Gormes Router config and route registry read model`
2. `Gormes Router setup wizard and provider-picker boundary`
3. `Gormes Router OpenAI-compatible models/chat endpoint`
4. `Gormes Router streaming SSE and fallback safety`
5. `Gormes Router health/status counters and redacted logs`
6. Later, non-MVP: `CLIProxyAPI-compatible upstream route adapter`

## Open questions for implementation

- Endpoint naming: use `/v1/status` or `/router/status` for Gormes-owned status. `/router/status` avoids pretending it is OpenAI-standard.
- Inbound auth shape: start with `Authorization: Bearer` and `X-Api-Key`; add CLIProxyAPI-compatible `X-Goog-Api-Key` and query keys only if needed.
- Local key printing: decide whether `gormes setup router` prints a generated local API key once or stores only an env/secret reference and tells the user where to retrieve/regenerate it.
- Persistence: keep counters in-memory for MVP; persist quota history only after a real operator need appears.
