---
title: "Providers Module Roadmap"
---

# Providers Module Roadmap

Generated from the single logical backlog. This page is a scoped review view; `progress.json` remains canonical.

**Module:** `providers`
**Rows:** 115
**Status counts:** `complete`: 115 · `in_progress`: 0 · `planned`: 0
**Priority counts:** `P0`: 9 · `P1`: 46 · `P2`: 23 · `P3`: 2 · `unset`: 35

## Phase 3 — The Black Box (Memory)

### 3.D — Semantic Fusion + Local Embeddings

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `providers` | Ollama embeddings |

## Phase 4 — The Brain Transplant

### 4.A — Provider Adapters

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `providers` | Provider interface + stream fixture harness |
| `complete` | `P1` | `providers` | Hermes provider registry and alias manifest |
| `complete` | `P1` | `providers` | OpenRouter Pareto router request plugin |
| `complete` | `unset` | `providers` | Tool-call normalization + continuation contract |
| `complete` | `unset` | `providers` | DeepSeek/Kimi reasoning_content echo for tool-call replay |
| `complete` | `unset` | `providers` | DeepSeek/Kimi cross-provider reasoning isolation |
| `complete` | `P1` | `providers` | DeepSeek/Kimi all-assistant reasoning_content replay |
| `complete` | `P1` | `providers` | Moonshot/Kimi tool-schema sanitizer |
| `complete` | `unset` | `providers` | Anthropic |
| `complete` | `P2` | `providers` | Azure OpenAI query/default_query transport contract |
| `complete` | `P2` | `providers` | Azure Anthropic Messages endpoint contract |
| `complete` | `P2` | `providers` | Azure Foundry transport probe read model |
| `complete` | `P2` | `providers` | Azure Foundry probe — path sniffing |
| `complete` | `P2` | `providers` | Azure Foundry probe — /models classification + Anthropic fallback |
| `complete` | `P2` | `providers` | Azure Foundry runtime env/config read model |
| `complete` | `P3` | `providers` | Azure Foundry CLI setup/status manual fallback |
| `complete` | `P2` | `providers` | Azure Foundry Responses-only model-family API mode |
| `complete` | `unset` | `providers` | Bedrock provider runtime binding |
| `complete` | `unset` | `providers` | Bedrock Converse payload mapping (no AWS SDK) |
| `complete` | `unset` | `providers` | Bedrock SigV4 + credential seam |
| `complete` | `unset` | `providers` | Bedrock stale-client eviction + retry classification |
| `complete` | `unset` | `providers` | Gemini Cloud Code request/stream mapper |
| `complete` | `unset` | `providers` | OpenRouter compatible-provider routing |
| `complete` | `P1` | `providers` | OpenRouter Grok prompt-cache affinity header |
| `complete` | `unset` | `providers` | Google Code Assist project/quota resolver |
| `complete` | `unset` | `providers` | Codex |
| `complete` | `unset` | `providers` | Codex Responses pure conversion harness |
| `complete` | `P1` | `providers` | Codex Responses assistant content role types |
| `complete` | `P1` | `providers` | Codex Responses HTTP client binding |
| `complete` | `unset` | `providers` | Codex OAuth state + stale-token relogin |
| `complete` | `unset` | `providers` | Codex stream repair + tool-call leak sanitizer |
| `complete` | `P1` | `providers` | Cross-provider reasoning-tag sanitization |
| `complete` | `unset` | `providers` | Tool-call argument repair + schema sanitizer |
| `complete` | `P1` | `providers` | OpenAI-compatible developer-role API-boundary swap |
| `complete` | `P1` | `providers` | xAI Grok provider adapter |
| `complete` | `P1` | `providers` | LM Studio provider adapter |
| `complete` | `P1` | `providers` | Vision-unsupported provider retry (strip-images-and-resend) |

### 4.B — Context Engine + Compression

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `providers` | Aux compression provider-aware context cap |

### 4.D — Smart Model Routing

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `providers` | Model metadata registry + context limits |
| `complete` | `unset` | `providers` | Provider-enforced context-length resolver |
| `complete` | `unset` | `providers` | Model pricing/capability registry fixtures |
| `complete` | `P1` | `providers` | Ollama Cloud models.dev suffix normalization |
| `complete` | `P1` | `providers` | Model catalog cache + preferred-provider live merge |
| `complete` | `unset` | `providers` | Routing policy and fallback selector |
| `complete` | `P1` | `providers` | Per-turn model selection |
| `complete` | `P2` | `providers` | Per-turn reasoning effort propagation |
| `complete` | `P1` | `providers` | Provider-default model resolution at config load |
| `complete` | `P1` | `providers` | OpenAI Codex Spark catalog and context parity |
| `complete` | `P1` | `providers` | Image input mode resolver + vision_analyze text fallback |

### 4.E — Trajectory + Insights

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `providers` | Trajectory writer + redaction gates |
| `complete` | `P0` | `providers` | Self-monitoring telemetry |

### 4.G — Credentials + OAuth

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P0` | `providers` | Token vault |
| `complete` | `P0` | `providers` | Anthropic OAuth/keychain credential discovery |
| `complete` | `P0` | `providers` | Multi-account auth |
| `complete` | `P0` | `providers` | Credential non-ASCII sanitizer + one-shot warning |
| `complete` | `unset` | `providers` | Google OAuth flow + refresh seam |
| `complete` | `P1` | `providers` | MiniMax OAuth provider registry and default auth routing |
| `complete` | `P0` | `providers` | GitHub Copilot token exchange + Responses mode selector |

### 4.H — Rate / Retry / Caching

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `providers` | Provider-side resilience |
| `complete` | `unset` | `providers` | Classified provider-error taxonomy |
| `complete` | `P1` | `providers` | Generic provider timeout message classifier |
| `complete` | `P2` | `providers` | Provider image-too-large error classification |
| `complete` | `P1` | `providers` | Unsupported temperature retry + Codex no-temperature guard |
| `complete` | `P2` | `providers` | Codex Responses temperature guard after flush removal |
| `complete` | `P1` | `providers` | Generic unsupported-parameter retry + max_tokens guard |
| `complete` | `unset` | `providers` | Jittered reconnect backoff schedule |
| `complete` | `P1` | `providers` | Retry-After header parsing + HTTPError hint |
| `complete` | `P1` | `providers` | Kernel retry honors Retry-After hint |
| `complete` | `P1` | `providers` | Streaming interrupt retry suppression |
| `complete` | `P1` | `providers` | Provider stream-drop retry diagnostics |
| `complete` | `P1` | `providers` | Provider stream-drop timing and upstream diagnostics |
| `complete` | `P3` | `providers` | Provider timeout config fail-closed helper |
| `complete` | `unset` | `providers` | Prompt-cache capability guard |
| `complete` | `P0` | `providers` | Provider account usage read model + renderer |
| `complete` | `unset` | `providers` | Provider rate guard + budget telemetry |
| `complete` | `P2` | `providers` | Provider rate guard — x-ratelimit header classification |
| `complete` | `P2` | `providers` | Provider rate guard — degraded-state + last-known-good evidence |
| `complete` | `P1` | `providers` | Hermes fast-mode request override serializer |

### 4.I — Native Agent Turn Closure

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `providers` | Provider-tool-memory golden transcript suite |
| `complete` | `P0` | `providers` | Gormes setup/channel/provider docs webpage parity gate |

### 4.K — Provider Fallback Chain

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `providers` | Resilient provider chain dispatch |
| `complete` | `P1` | `providers` | Hermes fallback activation + classifier carve-outs |
| `complete` | `P1` | `providers` | Fallback entry api_key_env credential alias |

### 4.M — Advanced Provider Routing

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `providers` | Circuit breaker per provider and API key |
| `complete` | `P2` | `providers` | P95 latency-aware failover |
| `complete` | `P2` | `providers` | Capability-based model tier routing |

## Phase 5 — The Final Purge

### 5.D — Vision + Image Generation

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P2` | `providers` | Image generation provider registry + plugin dispatch |

### 5.F — Skills System (Remaining)

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `providers` | Skills hub search result types + in-memory registry provider |
| `complete` | `unset` | `providers` | Skills hub search read-model function over registry providers |

### 5.G — MCP Integration

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `providers` | MCP OAuth state store + noninteractive auth errors |
| `complete` | `unset` | `providers` | MCP OAuth refresh + 401 session-expired recovery |

### 5.M — Mixture of Agents

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `providers` | Multi-model coordination |

### 5.N — Misc Operator Tools

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `providers` | Cron partial legacy job read-model normalization |

### 5.O — Hermes CLI Parity

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `providers` | Hermes auth credential-pool command surface |
| `complete` | `P1` | `providers` | Hermes auth OAuth provider adapters |
| `complete` | `P2` | `providers` | Hermes auth Spotify service-provider subcommand |
| `complete` | `P2` | `providers` | Gormes auth bare interactive credential-pool readout |
| `complete` | `P1` | `providers` | Gormes auth status per-provider aggregator |
| `complete` | `P1` | `providers` | Gormes auth add openai-codex strict isolation contract |
| `complete` | `P2` | `providers` | Gormes top-level logout provider shortcut |
| `complete` | `P1` | `providers` | Top-level logout configured-provider fallback |
| `complete` | `P2` | `providers` | Gormes model interactive provider/model picker |
| `complete` | `P1` | `providers` | Gormes setup model step uses the dynamic provider-tracked model picker |
| `complete` | `P1` | `providers` | Hermes fallback provider chain CLI commands |
| `complete` | `unset` | `providers` | Provider endpoint/API-key root flags + runtime resolution |
| `complete` | `P2` | `providers` | Scripted chat query model/provider resolver |
| `complete` | `P2` | `providers` | Custom provider model-switch credential preservation |
| `complete` | `P2` | `providers` | Custom provider model-switch key_env write guard |
| `complete` | `unset` | `providers` | Hermes config.yaml model/provider runtime bridge |
| `complete` | `P2` | `providers` | Nous OAuth device code + refresh token + agent key provisioning |

## Phase 6 — The Learning Loop (Soul)

### 6.E — Feedback Loop

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P0` | `providers` | Hermes curator auxiliary model routing slot |

## Phase 9 — Design & Security Hardening

### 9.B — Sandbox Provider Abstraction + Virtual Path System

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `providers` | Sandbox provider interface and virtual path security |

### 9.G — External Issue Radar Regression Guards

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `providers` | PicoClaw-derived session ledger read-model regression matrix |
| `complete` | `P1` | `providers` | PicoClaw-derived provider stream and auth regression matrix |
