---
title: "Agent Memory Systems"
description: "Memory architectures for AI agents: episodic, semantic, hierarchical, and agentic memory management."
weight: 30
---

# Agent Memory Systems

Memory is the most critical and least solved problem in agentic OS design.
Without memory, agents are stateless function calls. With memory, they become persistent digital colleagues.
This section surveys the academic literature on memory system design, from foundational cognitive architectures to cutting-edge agentic memory.

## The Memory Problem

LLM-based agents face a fundamental tension:
- **Context windows are finite** — but agent sessions can span hours, days, or weeks
- **Relevance decays** — old memories must be summarized, compressed, or forgotten
- **Retrieval must be fast** — slower than 100ms breaks the interactive experience
- **Memory must be structured** — flat vector stores miss temporal, causal, and hierarchical relationships

## 1. Foundational: Generative Agents (Park et al.)

[See foundational-architectures for full details]

The memory architecture from Generative Agents (Park et al., UIST 2023) remains the canonical reference:

```
┌─────────────────────────────────────────────┐
│              Memory Architecture              │
│                                               │
│  Experiences ──→ Memory Stream (timeline)     │
│       │                                       │
│       ├──→ Retrieval (recency + importance    │
│       │    + relevance scoring)               │
│       │                                       │
│       ├──→ Reflection (periodic synthesis     │
│       │    into higher-level insights)        │
│       │                                       │
│       └──→ Planning (goal-driven behavior     │
│            from reflection output)            │
└─────────────────────────────────────────────┘
```

**Retrieval formula**: `score = α·recency + β·importance + γ·relevance`

**Why this matters**: Nearly every subsequent memory paper (AgeMem, Synapse, A-Mem, GSW, REMem) builds on or explicitly contrasts with this architecture.

---

## 2. Agentic Memory — The Agent Decides

### 2.1 AgeMem: Unified LTM + STM Management

| Field | Value |
|-------|-------|
| **Title** | Agentic Memory: Learning Unified Long-Term and Short-Term Memory Management |
| **ArXiv** | [2601.01885](https://arxiv.org/abs/2601.01885) |

**Core innovation**: The LLM autonomously manages its own memory through tool-based actions. No heuristic pipeline — the agent decides what, when, and how to store/retrieve/forget.

**Memory Operations as Tools**:
- `store(content, metadata)` — persist new memory
- `retrieve(query)` — semantic search of stored memories
- `update(id, content)` — modify existing memory
- `summarize(filter)` — compress related memories
- `discard(id)` — explicitly forget

**Training**: Three-stage progressive RL (GRPO) to handle the sparse, discontinuous reward problem of memory operations.

**Results**: Outperforms memory-augmented baselines on 5 long-horizon benchmarks.

**Why it matters**: This is the endgame for agent memory — not a retrieval pipeline, but an agentic skill the model learns to use. Memory management becomes a learned behavior, not an engineered heuristic.

---

### 2.2 A-Mem: Zettelkasten for Agents

| Field | Value |
|-------|-------|
| **Title** | A-Mem: Agentic Memory for LLM Agents |
| **ArXiv** | [2502.12110](https://arxiv.org/abs/2502.12110) |

Applies the Zettelkasten knowledge management method to agent memory:

- **Atomic notes**: Each memory is a self-contained unit with description, keywords, tags
- **Dynamic interconnection**: Notes form semantic similarity links as the knowledge base grows
- **Memory evolution**: When new memories integrate, they trigger updates to existing contextual representations

The memory network deepens and refines over time — new information doesn't just add nodes, it reshapes the graph.

---

### 2.3 Synapse: Brain-Inspired Graph Memory

| Field | Value |
|-------|-------|
| **Title** | Synapse: Empowering LLM Agents with Episodic-Semantic Memory via Spreading Activation |
| **ArXiv** | [2601.02744](https://arxiv.org/abs/2601.02744) |

**Core innovation**: Uses spreading activation (like biological neural networks) instead of vector similarity for retrieval.

**Architecture**:
- **Episodic nodes**: Individual events/experiences
- **Semantic nodes**: Abstract concepts
- **Three edge types**: Temporal, abstraction, association
- **Triple hybrid retrieval**: Embeddings + activation-based graph traversal + lateral inhibition

**Results**: Outperforms RAG baselines on LoCoMo benchmark.

**Why it matters**: Graph-based memory with dynamic activation captures relationships that flat vector stores miss. The spreading activation pattern means relevance "emerges" from network dynamics rather than being computed by a single similarity function.

---

## 3. Hierarchical Memory Architectures

### The Hierarchy Spectrum

| System | Levels | Mechanism | Scale |
|--------|--------|-----------|-------|
| **MemGPT/Letta** | 3-tier (core, recall, archival) | Agent self-edits core | GB-scale archival |
| **H-MEM** | 4-level (Domain→Category→Trace→Episode) | Pointer-based routing | Sublinear scaling |
| **xMemory** | 4-level + sparsity-semantics | Decoupling + aggregation | Top-down retrieval |
| **HiGMem** | Event-turn dual-layer | LLM-guided semantic anchors | Event indexing |
| **SEEM** | Graph + episodic dual-layer | Reverse provenance expansion | Structured events |
| **EHC** | Fast-Access + Deep-Retrieval pools | OS caching-inspired | Hot/cold migration |

### 3.1 H-MEM (2025)

| ArXiv | [2507.22925](https://arxiv.org/pdf/2507.22925) |

4-level hierarchy with positional index encoding:
```
Domain → Category → Trace → Episode
   │         │         │        │
   └─────────┴─────────┴────────┘
        Pointer-based routing
```

Sublinear scaling is the key claim — as memory grows, retrieval cost doesn't grow linearly because pointers route directly to relevant tiers.

### 3.2 xMemory (2025)

| ArXiv | [2602.02007](https://arxiv.org/pdf/2602.02007v2) |

Uses decoupling + aggregation with a sparsity-semantics objective. Top-down retrieval means the query first hits the highest-level summary, then drills down only into relevant branches.

---

## 4. Memory with Time

### 4.1 REMem: Reasoning with Episodic Memory

| ArXiv | [2602.13530](https://arxiv.org/abs/2602.13530v3) |

Two-phase architecture:
1. **Indexing**: Builds hybrid memory graph with time-aware gists + facts
2. **Agentic inference**: ReAct-style retriever with tools for iterative retrieval, graph exploration, temporal filtering

3.4% improvement over Mem0 and HippoRAG 2 on episodic benchmarks. The key contribution is treating temporal relationships as first-class query filters.

### 4.2 GSW: Generative Semantic Workspace

**Two components**:
- **Operator**: Maps observations to structured semantic representations
- **Reconciler**: Integrates into persistent workspace with temporal, spatial, and logical coherence

20% improvement over RAG, 51% token reduction. Structured episodic memory for agents that need space-time-anchored narratives for entity tracking.

---

## 5. Memory with Action

### 5.1 MemGen: Generative Latent Memory

| Year | 2025 (OpenReview) |

Memory as machine-native latent vectors, not human-readable text:
- **Memory trigger**: Monitors agent reasoning state, decides when to explicitly invoke memory
- **Memory weaver**: Uses current state as stimulus to build latent token sequences

Outperforms ExpeL and AWM by 38.22%, GRPO by 13.44%. Shows cross-domain generalization and emergent memory-like functions (planning memory, procedural memory, working memory).

### 5.2 RoboMemory: Embodied Multi-Memory

| ArXiv | [2508.01415](https://arxiv.org/abs/2508.01415) |

Unified spatial, temporal, episodic, and semantic memory in parallel architecture:
- Dynamic spatial knowledge graph for scalable, consistent updates
- Closed-loop planner with critic module for adaptive decision-making

26.5% improvement on EmbodiedBench using Qwen2.5-VL-72B. Shows multi-memory synergy for complex physical environments.

---

## 6. Survey: Memory for Autonomous LLM Agents (2026)

| ArXiv | [2603.07670](https://arxiv.org/abs/2603.07670v1) |

The most comprehensive survey to date (2022–early 2026). Key contributions:

**Formalizes memory as a write-manage-read loop**:
- **Write**: When to store, what to store, how to encode
- **Manage**: Update, summarize, compress, forget
- **Read**: Retrieve, rank, integrate into context

**Three-dimensional taxonomy**:
1. **Temporal scope** (short-term → working → long-term)
2. **Representational substrate** (text → embeddings → latent → hybrid)
3. **Control policy** (heuristic → learned → agentic)

**Five mechanism families**:
1. Context-resident compression (summarize to fit window)
2. RAG-based augmentation (external retrieval)
3. Reflective self-improvement (learn from past)
4. Hierarchical virtual context (MemGPT-style tiered storage)
5. Policy-learned management (AgeMem-style learned decisions)

---

## 7. How This Informs Memory System Design

### Design Principles

| Principle | Source | Implementation |
|-----------|--------|---------------|
| **Agent decides what to remember** | AgeMem | Memory operations as tools, not heuristics |
| **Hierarchy beats flat** | H-MEM, xMemory, MemGPT | Tier by recency, importance, abstraction |
| **Graph beats vectors** | Synapse, A-Mem | Edges capture temporal, causal, semantic relations |
| **Latent beats text** | MemGen | Machine-native representations for density |
| **Time is a first-class dimension** | REMem, GSW | Temporal filters on retrieval |
| **Memory self-evolves** | A-Mem, MemGen | New information reshapes existing structure |

### For Building Agentic OS

| OS Component | Memory Pattern to Use |
|---|---|
| **Session memory** | Context-resident compression + snapshot/restore (AIOS Context Manager) |
| **Cross-session memory** | Hierarchical with hot/cold migration (MemGPT/EHC) |
| **Skill library** | Zettelkasten interconnected notes (A-Mem) |
| **User preferences** | Agent-curated profile with self-update (Hermes USER.md pattern) |
| **Knowledge graph** | Spreading activation graph (Synapse) |
| **Long-term facts** | Policy-learned storage/retrieval/forget (AgeMem) |

### For Gormes/Goncho

Goncho (Gormes' in-binary SQLite memory) maps to:
- **Working memory**: Session-scoped context with FTS5 retrieval
- **Episodic memory**: Conversation history with timestamp indexing
- **Semantic memory**: User profiles, learned facts, skill metadata
- **Procedural memory**: Skill definitions, tool preferences, behavioral patterns
