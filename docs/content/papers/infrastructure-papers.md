---
title: "Agent Infrastructure"
description: "Infrastructure layer: context management, model serving, state, observability, security, scaling."
weight: 52
---

# Agent Infrastructure — The Boring But Critical Layer

## 1. Context Window Management

### Context Compression

| Resource | Approach | Key Result |
|----------|----------|-----------|
| **LLMLingua** (Microsoft) | Budget controller + iterative token compression | 2-4x compression, minimal degradation |
| **Selective Context** | Perplexity-based low-value token removal | 40% memory reduction, 2x content capacity |
| **ICAE** | LoRA-adapted encoder + frozen decoder | 4x compression into memory slots, ~1% extra params |
| **SCA** | Greedy KV cache compression | Retains distinctive vectors over similar ones |

### Context Caching

| Provider | Mechanism | Savings |
|----------|----------|---------|
| **Anthropic** | `cache_control` marker, 5-min TTL (extendable 1hr) | 90% cost reduction |
| **Google Gemini** | Implicit (auto) + Explicit (manual) caching | 90% discount |
| **Vertex AI** | Regional cache, TTL expiration, CMEK encryption | TTL-based |
| **LiteLLM** | Unified API across providers | Auto-translation of cache markers |

### Infinite Context via Retrieval

| Resource | Approach | Scale |
|----------|----------|-------|
| **Unlimiformer** (NeurIPS 2023) | kNN index over hidden states; sublinear query | Unlimited input |
| **MemWalker** | Tree-based navigation of long contexts | Query-driven |
| **RAPTOR** | Hierarchical clustering + recursive summarization | Flattened tree retrieval |
| **Infini-attention** | Compressive memory + local attention | 1M token passkey on 1B/8B models |
| **EM-LLM** | Bayesian surprise + graph boundary detection | 10M token retrieval |

---

## 2. Model Serving for Agents

### Inference Optimization

| Resource | Approach | Result |
|----------|----------|--------|
| **NVIDIA Dynamo** | `agent_hints` for latency sensitivity, priority KV eviction, speculative prefill | 3x TTFT on turn 2+ |
| **StreamServe** | Disaggregated prefill-decode + metric routing + adaptive speculation | 6.8x throughput |
| **EAGLE-3** | SuffixDecoding, KV reuse, speculative tool execution | Agent loop acceleration |

### Batching & Scheduling

| Resource | Approach | Result |
|----------|----------|--------|
| **Concur** | AIMD-based cache-aware admission at agent granularity | Prevents middle-phase thrashing |
| **Halo** | DAG-based query plan optimization, adaptive batching, KV-cache sharing | 18.6x speedup |
| **AdaServe** | SLO-customized speculation depth, latency targets | 4.3x SLO violation reduction |

---

## 3. State Management

### Durable Execution Patterns

| Pattern | Mechanism | Provider |
|---------|----------|----------|
| **Checkpoint/Replay** | Atomic state snapshots, replay from checkpoint | AWS Lambda, Temporal |
| **Sagas** | Compensation-based rollback for multi-step workflows | AWS Step Functions |
| **Event Sourcing** | Append-only event log, rebuild state from events | Custom implementations |

### Checkpointing Systems

| System | Key Feature |
|--------|------------|
| **LangGraph** | PostgreSQL/SQLite checkpointing with time-travel debugging |
| **Temporal** | Durable execution SDK: steps, waits, callbacks, 1-year max |
| **CRIU** | OS-level process checkpointing |
| **Dapr** | KEDA queue-depth autoscaling, actors for stateful agents |

---

## 4. Observability

### Tracing Standards

| Standard | Scope | Status |
|----------|-------|--------|
| **OpenTelemetry GenAI** | `invoke_agent`, `execute_tool` span types | Stabilizing |
| **Agent Spec** | Open standard for agent traces, framework-agnostic | Emerging |

### Tool Comparison

| Tool | Strengths | Best For |
|------|-----------|----------|
| **LangSmith** | Deep LangChain integration, prompt playground | LangGraph-heavy teams |
| **Langfuse** | Self-hosted, MIT license, framework-agnostic | General-purpose, self-hosting |
| **Arize Phoenix** | RAG evals, UMAP drift detection, local-first | RAG-first teams, ML monitoring |
| **Helicone** | Simple integration, cost tracking | Lightweight observability |

---

## 5. Security Infrastructure

### Prompt Injection Defense

| Resource | Approach | Result |
|----------|----------|--------|
| **LlamaFirewall** | BERT classifier + CoT auditing + static analysis | Multi-layer defense |
| **AgentSentinel** | Real-time system-level tracing, event dependency analysis | CUA defense |
| **Multi-Agent Defense Pipeline** | Chain-of-agents + coordinator | 100% ASR mitigation (0% from 30%) |
| **Prompt Fencing** | Cryptographically signed prompt segments | Verifiable trust boundaries |
| **OpenClaw Isolation** | Two-agent privilege separation + JSON formatting | 323x improvement over baseline |
| **MeshAI Detection** | Proxy-layer detection, 15+ patterns | <2ms overhead |

### Sandboxing

| Resource | Approach |
|----------|----------|
| **ceLLMate** | Browser-level HTTP interception; agent-agnostic Chrome extension |
| **AgentBound** | Android-style permission model for MCP servers; 80.9% auto-policy accuracy |
| **Progent** | Programmable privilege control DSL; 0% attack success |
| **ClawLess** | BPF syscall interception + formal security model |

### Multi-Tenant Security

| Pattern | Mechanism |
|---------|----------|
| **Row-Level Security** | Tenant ID on every row, query filters |
| **Virtual Keys** | Per-tenant API keys, AES-256-GCM encryption, hash-based lookup |
| **Hierarchical Budgets** | org → team → user → key budget chains |
| **MicroVM Isolation** | Per-tenant Firecracker VMs, default-deny egress |

---

## 6. Scaling

### Architecture Patterns

| Pattern | When to Use | Example |
|---------|------------|---------|
| **Work queues** | Async task processing, priority ordering | Redis/Kafka-backed queues |
| **Agent pools** | Warm workers, scale-to-N | Docker Compose, K8s deployments |
| **Slot-based** | Fixed capacity, backpressure | PostgreSQL queue, Consul discovery |
| **Actor model** | Stateful agents, message-passing | Dapr actors for stateful agents |

### Cold Start Mitigation

| Technique | Result |
|-----------|--------|
| **Warm containers** (`warm_containers=N`, `min_containers=N`) | Prevents scale-to-zero latency |
| **Speculative prefill** (NVIDIA Dynamo) | Pre-warm KV cache for predicted next-turn prefix; 3x TTFT |
| **Shared KV cache** (Halo) | Cache sharing across operators in same workflow |

---

## 7. Replay & Debugging

| Tool | Feature |
|------|---------|
| **AgentOptics** | Fork at failure, replay with fix, AI diagnosis |
| **Cogitator** | Checkpoint replay, deterministic vs live, trace comparison |
| **agent-replay** | SQLite-powered CLI, hallucination detection, AI root cause |
| **AgentReplay** | Local-only, time-scrub slider, DAG-based trace storage |

---

## Quick Reference: Key Trade-offs

| Problem | Solution | Trade-off |
|---------|----------|-----------|
| Context window limits | Retrieval (Unlimiformer, RAPTOR) | Latency vs compression quality |
| Prefix redundancy | Prompt caching | Cache invalidation complexity |
| Decode latency | Speculative decoding | Extra compute on rejection |
| Agent isolation | Privilege separation | Cross-agent communication overhead |
| Multi-tenant cost | Virtual keys + budgets | Cache namespace collision |
| KV cache eviction | Priority-based + agent hints | Tuning complexity |
| Cold starts | Warm containers / speculative prefill | Idle resource cost |
