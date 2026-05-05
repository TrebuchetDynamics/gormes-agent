---
title: "Multi-Agent Systems & Tool Ecosystems"
description: "Multi-agent orchestration, collaboration scaling laws, MCP tool protocol, and tool learning."
weight: 40
---

# Multi-Agent Systems & Tool Ecosystems

Two forces shape the scalability of agentic OS: multi-agent orchestration (how agents collaborate) and tool ecosystems (how agents act on the world).

## PART 1: Multi-Agent Orchestration

### 1.1 MetaGPT — SOP-Based Collaboration

| Field | Value |
|-------|-------|
| **Title** | MetaGPT: Meta Programming for a Multi-Agent Collaborative Framework |
| **Authors** | Sirui Hong, Mingchen Zhuge, Jonathan Chen, Jürgen Schmidhuber |
| **Venue** | ICLR 2024 (Oral) |
| **ArXiv** | [2308.00352](https://arxiv.org/abs/2308.00352) |

**Core Pattern**: Encode Standardized Operating Procedures (SOPs) into prompts, modeling agents as a simulated software company.

**Architecture**:
```
┌──────────────────────────────────────────────┐
│            MetaGPT Software Company            │
│                                                │
│  Product Manager ──→ Architect ──→ Engineer   │
│        │                │              │       │
│        └────────────────┴──────────────┘       │
│                   │                            │
│         Shared Message Pool (pub/sub)          │
│                   │                            │
│           Executable Feedback Loop             │
└──────────────────────────────────────────────┘
```

**Key insights**:
- Role specialization with SOP-constrained communication
- Structured messages via shared pool + publish-subscribe
- Executable feedback for self-correction (run the code, not just review it)

**Results**: Outperforms ChatDev on HumanEval and MBPP.

**Why it matters**: Proves that organizational structure matters for agent teams. Well-defined roles + SOP-constrained communication outperform free-form collaboration. The assembly-line paradigm is surprisingly effective.

---

### 1.2 MACNet — Collaboration Scaling Laws

| Field | Value |
|-------|-------|
| **Title** | 基于大型语言模型的多智能体协作扩展研究 |
| **Authors** | Chen Qian, Zihao Xie, Maosong Sun |
| **Institution** | Tsinghua University, Peng Cheng Laboratory |
| **Venue** | ICLR 2025 |

**Core question**: Does adding more agents improve performance, like neural scaling laws?

**Key findings**:
1. **Logistic growth**: Performance improves with agent count, saturating around ~100 agents
2. **Irregular topology wins**: Random graph topologies outperform regular ones due to small-world properties (shorter interaction paths)
3. **Collaborative emergence is cheap**: Neural scaling needs billions of parameters; collaborative emergence appears at hundreds of agents

**Architecture**: DAG-organized agents — nodes are Actors, edges are Critics.

**Why it matters**: Provides the first empirical answer to "how many agents should we spawn?" The answer: more helps, but with diminishing returns. And topology matters more than you'd think.

---

### 1.3 CAMEL — Role-Playing Framework

| Field | Value |
|-------|-------|
| **Title** | CAMEL: Communicative Agents for "Mind" Exploration |
| **ArXiv** | [2303.17760](https://arxiv.org/abs/2303.17760) |

**Core Pattern**: AI user + AI assistant role-play through multi-turn conversation with "inception prompting" to guide autonomous cooperation.

**Challenges identified**: Conversation deviation, role flipping, termination conditions — issues that remain relevant for any autonomous multi-agent system.

---

### 1.4 MegaAgent — No Predefined SOPs

| Field | Value |
|-------|-------|
| **Title** | MegaAgent: A Large-Scale Autonomous LLM-based Multi-Agent System Without Predefined SOPs |
| **Authors** | Qian Wang, Tianyu Wang, et al. (Huazhong University of Science & Technology) |
| **Venue** | ACL 2025 Findings |

**Core pattern**: Dynamic agent generation by task complexity — no hand-crafted SOPs. Automatic task decomposition, parallel execution, efficient communication, system-level monitoring.

**Results**: Scaled to 590 agents in national policy simulation; completed Gomoku game development in 800 seconds. Significantly outperforms MetaGPT on task completion efficiency and scalability.

---

### 1.5 Puppeteer — Dynamic Orchestration

| Authors | Yufan Dang, Chen Qian, et al. (Tsinghua) |
| ArXiv | [2505.19591](https://arxiv.org/abs/2505.19591) |

Central "puppeteer" dynamically selects and sequences agent activation:
- Dynamic routing based on current context
- Adaptive evolution through REINFORCE learning
- Converges toward tighter, more cyclic reasoning structures

**Why it matters**: Proves that a central orchestrator (vs. fully decentralized) is efficient for structured tasks.

---

### 1.6 Multi-Agent Architecture Spectrum

| Architecture | Control | Example | Best For |
|---|---|---|---|
| **Centralized** | One supervisor routes tasks | Puppeteer, MAC | Structured, decomposable tasks |
| **Decentralized** | Peer-to-peer messaging | CAMEL | Creative, open-ended tasks |
| **Hybrid** | Hierarchy with local autonomy | MetaGPT, MACNet | Complex organizational tasks |
| **Dynamic** | Topology evolves with task | MegaAgent | Unknown-scope problems |

---

## PART 2: Tool Ecosystems

### 2.1 Model Context Protocol (MCP)

**Specification**: [modelcontextprotocol.io](https://modelcontextprotocol.io)
**GitHub**: [modelcontextprotocol](https://github.com/modelcontextprotocol) (42 repos, 79k+ stars)

MCP standardizes how agents discover, invoke, and reason about tools:

```
┌──────────┐     ┌──────────────┐     ┌──────────┐
│  Agent   │────→│  MCP Client  │────→│  Tool    │
│  (LLM)   │←────│  (Protocol)  │←────│  Server  │
└──────────┘     └──────────────┘     └──────────┘
```

**Core primitives**:
- **Tools**: Functions the agent can call (with schema-based input/output)
- **Resources**: Data the agent can read (via URI)
- **Prompts**: Reusable interaction templates
- **Roots**: Named filesystem locations for scoped access
- **Sampling**: Interactive human-in-the-loop prompts

**Transport**: stdio (local), SSE, HTTP streaming

**Key design insight (Cloudflare)**: "Code Mode" — expose 2 tools (`search()` + `execute()`) rather than 2,500 API endpoints as 2,500 tools. ~1,000 tokens fixed cost regardless of API size.

---

### 2.2 Tool Learning — How Agents Get Better at Tools

#### ToolACE (ICLR 2025)

Automated function-calling data generation:
1. **Tool Self-Evolution Synthesis (TSS)**: LLM-as-evaluator generates 26,507 diverse APIs
2. **Self-guided complexifying**: Multi-agent interaction generates four call types (single, parallel, dependent, non-tool)
3. **Dual-layer verification**: Rule-based + model-based validation

8B model outperforms GPT-4 on function calling benchmarks.

#### ToolCoder (2025)

| ArXiv | [2502.11404](https://arxiv.org/abs/2502.11404) |

Reframes tool learning as code generation. Converts natural language queries into structured Python function scaffolds. Successfully executed code is stored in a function repository for reuse. Error backtracking for systematic debugging.

#### InfTool (2025)

| ArXiv | [2512.23611](https://arxiv.org/abs/2512.23611) |

Fully autonomous self-evolving framework — three collaborative agents (user simulator, tool assistant, MCP server) generate diverse, verified trajectories from raw API specs. Closed-loop: synthesized data trains models, improved models generate better data.

InfTool-7B (61.7) surpasses GPT-5.2 (60.4) on BFCL benchmark.

#### Tool-R1 (2025)

| ArXiv | [2509.12867](https://arxiv.org/abs/2509.12867) |

RL-based tool use training. Generates executable Python code for flexible tool calling; reward function combines LLM judgment + execution success rate. Dynamic sample queue for training efficiency. ~10% accuracy improvement on GAIA.

---

### 2.3 Agent-Computer Interfaces (ACI)

| Field | Value |
|-------|-------|
| **Title** | SWE-Agent: Agent-Computer Interfaces Enable Automated Software Engineering |
| **ArXiv** | [2405.15793](https://arxiv.org/abs/2405.15793) |

**Key insight**: Interface design for AI agents matters as much as model capability.

SWE-Agent provides a custom ACI with a small set of simple, high-level actions (view, search, edit files) instead of a granular Linux shell. Results: 12.5% SWE-bench resolution (vs. 3.8% prior SOTA). ACI reduces errors from 35% to 9%.

**Why it matters**: Don't give the agent raw bash. Design an interface that matches how the model thinks.

---

### 2.4 Tool System Design Principles

| Principle | Source | Implementation |
|-----------|--------|---------------|
| **Few high-level tools beat many low-level ones** | SWE-Agent | `< 10` curated actions, not raw shell |
| **Code mode for complex APIs** | Cloudflare MCP | `search()` + `execute()` for any API |
| **Tools should self-evolve** | ToolACE, InfTool | Auto-generate tool data from specs |
| **Schema-driven safety** | MCP Guardrails | Input/output types + pre/post conditions |
| **Composition over enumeration** | Voyager | Reusable skills compose into new capabilities |

---

## 3. Multi-Agent + Tool Integration

### The Agent-Tool Interface

```
┌──────────────────────────────────────────────┐
│              Orchestration Layer              │
│  ┌──────────┐  ┌──────────┐  ┌───────────┐  │
│  │Supervisor│  │  Router  │  │  Delegate │  │
│  └──────────┘  └──────────┘  └───────────┘  │
│         │            │             │         │
│         ▼            ▼             ▼         │
│  ┌──────────────────────────────────────┐   │
│  │           MCP Client Layer            │   │
│  │  ┌──────┐ ┌──────┐ ┌──────┐         │   │
│  │  │ FS   │ │Browser│ │ DB  │  ...     │   │
│  │  │Server│ │Server │ │Server│         │   │
│  │  └──────┘ └──────┘ └──────┘         │   │
│  └──────────────────────────────────────┘   │
└──────────────────────────────────────────────┘
```

**Key pattern**: Every agent (orchestrator, worker, specialist) talks to tools through the same MCP interface. This means:
- Tools can be shared across agents without duplication
- Tool permissions can be managed centrally
- Adding a new tool benefits all agents immediately

### For Agentic OS Design

| OS Layer | Multi-Agent Pattern | Tool Pattern |
|----------|-------------------|--------------|
| **Orchestration** | Puppeteer-style central routing | Task-based tool selection |
| **Kernel** | Dynamic agent spawn/destroy | Tool registry with permission model |
| **Safety** | Agent capability scoping | Tool-level access control |
| **Memory** | Shared memory pool with provenance | Tool results as memory entries |
| **Evolution** | Agent specialization over time | Self-evolving tool catalogs |
