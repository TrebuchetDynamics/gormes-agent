---
title: "Agentic Operating System Design"
description: "Designing OS-level agent architectures: AIOS, ACOS, OpenCode, Hermes Agent, and the LLM-as-kernel paradigm."
weight: 20
---

# Agentic Operating System Design

Building a better agentic OS requires understanding both the academic architecture proposals and the production systems that have shipped. This section covers the OS-level design patterns, the kernel abstractions, and detailed analyses of OpenCode and Hermes Agent as reference implementations.

## 1. The LLM-as-Kernel Paradigm

### 1.1 AIOS: LLM Agent Operating System

| Field | Value |
|-------|-------|
| **Title** | AIOS: LLM Agent Operating System |
| **Authors** | Kai Mei, Zelong Li, Shuyuan Xu, Ruosong Ye, Yingqiang Ge, Yongfeng Zhang |
| **Institution** | Nanjing University / UCLA |
| **Venue** | NeurIPS 2024 |
| **ArXiv** | [2403.16971](https://arxiv.org/abs/2403.16971) |

AIOS first systematically proposed embedding LLMs into the OS kernel, treating the LLM as the central cognitive engine. The kernel contains six modules:

| Module | Function | Design Insight |
|--------|----------|---------------|
| **Agent Scheduler** | FIFO/RR algorithms for LLM resource utilization | Agents are processes, need scheduling |
| **Context Manager** | Snapshot + interrupt/resume | Context is state, needs persistence |
| **Memory Manager** | Short-term memory with TTL | Working memory for active tasks |
| **Storage Manager** | Long-term persistent memory | Episodic knowledge base |
| **Tool Manager** | External API orchestration | Tools are OS services |
| **Access Manager** | Permission-group access control | Security boundaries between agents |

**Results**: 2.1× throughput improvement in multi-agent concurrency; 60-70% context switching latency reduction; 3× efficiency at 2,000 concurrent agents.

**Architecture diagram**:
```
┌─────────────────────────────────────────────┐
│                  AIOS Kernel                  │
│  ┌─────────┐ ┌──────────┐ ┌──────────────┐  │
│  │Scheduler│ │ Context  │ │   Memory     │  │
│  │         │ │ Manager  │ │   Manager    │  │
│  └─────────┘ └──────────┘ └──────────────┘  │
│  ┌─────────┐ ┌──────────┐ ┌──────────────┐  │
│  │  Tool   │ │ Storage  │ │   Access     │  │
│  │ Manager │ │ Manager  │ │   Manager    │  │
│  └─────────┘ └──────────┘ └──────────────┘  │
│                   LLM Core                    │
└─────────────────────────────────────────────┘
```

---

### 1.2 ACOS: Agent-Centric Operating System

| Field | Value |
|-------|-------|
| **Title** | Agent Centric Operating System – a Comprehensive Review and Outlook |
| **Authors** | Shian Jia, Xinbo Wang, Mingli Song, Gang Chen |
| **Institution** | Zhejiang University |
| **ArXiv** | [2411.17710](https://arxiv.org/abs/2411.17710) |

ACOS proposes that every OS component should be abstracted as an agent, creating a modular, adaptable, cross-platform architecture. Key differentiators from traditional OS:

| Dimension | Traditional OS | Agent OS |
|-----------|---------------|----------|
| Resource granularity | Process-level | Agent-level |
| Task scheduling | Deterministic | Probabilistic-deterministic hybrid |
| IPC mechanism | Signals, pipes, sockets | Agent-to-agent protocols |
| Resource management | CPU, memory, I/O | LLM context, tool access, memory |

**Key contribution**: First academic paper to establish a complete "technology framework" for Agent OS, defining the "LLM → Agent → Resource Management" collaboration logic.

---

### 1.3 Architecting AgentOS (2026)

| Field | Value |
|-------|-------|
| **Title** | Architecting AgentOS: From Token-Level Context to Emergent System-Level Intelligence |
| **Authors** | ChengYou Li, XiaoDong Liu, XiangBao Meng, XinYu Zhao |
| **ArXiv** | [2602.20934](https://arxiv.org/abs/2602.20934) |

Maps classical OS abstractions (memory paging, interrupt handling, process scheduling) onto LLM-native constructs:

- **Deep Context Management**: Context window redefined as an "addressable semantic space" rather than a passive buffer
- **Semantic Slicing**: Time-aligned context partitioning to mitigate cognitive drift in multi-agent collaboration
- **Interrupt Model**: OS-style interrupts for agent preemption and resumption

**Why it matters**: This is the frontier — bridging classical OS theory with LLM-native system design.

---

## 2. OpenCode — Event-Driven Terminal Agent

**Repository**: [github.com/anomalyco/opencode](https://github.com/anomalyco/opencode)
**Language**: TypeScript (Bun) | **License**: MIT | **Stars**: 133k+

### Architecture Analysis

OpenCode is an event-driven, multi-provider terminal coding agent. Its architecture reveals key design decisions for agentic OS builders:

**Core Design Principles**:
1. **75+ provider support, zero lock-in** — abstract LLM as a swappable backend
2. **Global pub/sub event bus** — decouples logic execution from terminal rendering
3. **ReAct cycle** — standardized Thought→Action→Observation processing
4. **18 built-in tools + TS custom tools + plugin tools** — extensible tool ecosystem
5. **SQLite-backed layered storage** — session persistence, shareable transcripts

**Architecture**:
```
┌──────────────────────────────────────────────────┐
│                   OpenCode CLI                     │
│  ┌─────────┐  ┌──────────┐  ┌─────────────────┐  │
│  │ Prompt  │→│  Agent    │→│  Event Bus       │  │
│  │ Parser  │  │  Loop     │  │  (pub/sub)      │  │
│  └─────────┘  └──────────┘  └─────────────────┘  │
│       ↑                           ↓               │
│  ┌─────────┐  ┌──────────┐  ┌─────────────────┐  │
│  │ Providers│  │  Tools   │  │  Renderer       │  │
│  │ (75+)   │  │  (18+)   │  │  (Terminal/Web) │  │
│  └─────────┘  └──────────┘  └─────────────────┘  │
│                    SQLite Store                    │
└──────────────────────────────────────────────────┘
```

**Key Design Decisions**:
- Three internal agents (build, plan, general) with shared event bus — configuration templates, not true multi-agent
- Event bus decouples I/O from processing — the terminal is just one subscriber
- Tool execution through MCP server with permission gates

**What to learn**:
- Event-driven architecture for agent systems
- Provider abstraction layer design
- Multi-client rendering (terminal + web) from shared core

---

## 3. Hermes Agent — Self-Evolving Python Agent

**Repository**: [github.com/NousResearch/hermes-agent](https://github.com/NousResearch/hermes-agent)
**Language**: Python (~369k lines) | **License**: MIT | **Stars**: 70k+

### Architecture Analysis

Hermes is a "persistently online digital employee" — not a Q&A window but a colleague with memory, self-evolving skills, and multi-platform presence.

**Core Systems**:

### 3.1 GEPA Self-Evolution Engine

The most architecturally distinctive feature. GEPA (no public paper, but documented in code) optimizes agent prompts through a backpropagation-like mechanism:

- 100–500 evaluations per iteration (vs. traditional RL requiring 10,000+)
- Policy iteration without gradient descent
- Prompt optimization emerges from iterative evaluation + mutation

**Architectural significance**: This treats the agent's personality/behavior as an optimizable artifact, not a static prompt. The engine can adjust tool selection heuristics, response style, and task decomposition strategies.

### 3.2 Persistent Memory Architecture

```
┌─────────────────────────────────────────┐
│            Memory System                 │
│  ┌───────────┐  ┌────────────────────┐  │
│  │ MEMORY.md │  │ USER.md            │  │
│  │ (Facts,   │  │ (Preferences,      │  │
│  │  Lessons) │  │  Patterns)         │  │
│  └───────────┘  └────────────────────┘  │
│  ┌───────────────────────────────────┐  │
│  │ SQLite FTS5 + LLM Summaries       │  │
│  └───────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

- **MEMORY.md**: Environmental facts, lessons learned, task outcomes — grows with usage
- **USER.md**: User preferences, communication style, constraints
- **SQLite FTS5**: Full-text search with LLM-generated summaries for retrieval

### 3.3 Multi-Platform Gateway

15+ messaging platforms from a single gateway process:
- Western: Telegram, Discord, Slack, WhatsApp, Signal, Matrix
- Chinese: 飞书 (Feishu), 钉钉 (DingTalk), 企业微信 (WeCom), 微信 (WeChat)
- Other: Email, SMS, Mattermost, Webhooks

Each adapter translates channel-native events into the shared agent kernel.

### 3.4 Security Architecture

- Instruction-level approval for dangerous operations
- Dangerous mode blocking with configurable thresholds
- Docker sandbox isolation for code execution
- Path traversal protection
- SSRF mitigation
- Zero CVE record (as of documentation)

**Key insight**: Security is layered — prompt-level filtering, runtime approval gates, and OS-level sandboxing work together.

---

## 4. OS Decomposition for Agent Systems

Based on analysis of AIOS, ACOS, OpenCode, Hermes, and the broader literature, an agentic OS should decompose into these layers:

```
┌──────────────────────────────────────────────┐
│              User Interface Layer              │
│  ┌─────────┐ ┌──────────┐ ┌───────────────┐  │
│  │   TUI   │ │   Web    │ │   Chat Apps    │  │
│  └─────────┘ └──────────┘ └───────────────┘  │
├──────────────────────────────────────────────┤
│              Orchestration Layer              │
│  ┌─────────┐ ┌──────────┐ ┌───────────────┐  │
│  │ReAct    │ │  Subagent│ │   Task        │  │
│  │Loop     │ │  Manager │ │   Decomposer  │  │
│  └─────────┘ └──────────┘ └───────────────┘  │
├──────────────────────────────────────────────┤
│               Kernel Layer                    │
│  ┌─────────┐ ┌──────────┐ ┌───────────────┐  │
│  │Provider │ │ Context  │ │   Memory      │  │
│  │Router   │ │ Manager  │ │   Manager     │  │
│  └─────────┘ └──────────┘ └───────────────┘  │
│  ┌─────────┐ ┌──────────┐ ┌───────────────┐  │
│  │  Tool   │ │ Session  │ │   Safety      │  │
│  │ Registry│ │ Manager  │ │   Guard       │  │
│  └─────────┘ └──────────┘ └───────────────┘  │
├──────────────────────────────────────────────┤
│              Infrastructure Layer             │
│  ┌───────────────┐  ┌─────────────────────┐  │
│  │  SQLite/DB    │  │  Sandbox Runtime    │  │
│  └───────────────┘  └─────────────────────┘  │
└──────────────────────────────────────────────┘
```

| Layer | Gormes Component | Hermes Equivalent | OpenCode Equivalent |
|-------|-----------------|-------------------|---------------------|
| UI | TUI + Web Dashboard + Gateway | TUI + Web + Gateway | Terminal CLI |
| Orchestration | ReAct loop + subagent spawn | GEPA engine | Agent loop + 3 agents |
| Kernel | Provider router, Goncho memory | Provider routing, MEMORY.md | Provider + SQLite |
| Infrastructure | SQLite, Go sandbox | Docker, SQLite | SQLite, Bun runtime |

## 5. Design Principles for Agentic OS

From analysis of these systems, key principles emerge:

### 5.1 The OS is the Agent

Don't bolt agents onto an existing OS. Design the OS around agents as first-class citizens.
- AIOS: LLM is the kernel
- ACOS: Everything is an agent

### 5.2 Event-Driven Architecture

Agents are reactive. Polling loops are anti-patterns.
- OpenCode: Global pub/sub event bus
- Hermes: Channel adapters publish to shared kernel

### 5.3 Memory is a First-Class Kernel Service

Not an add-on vector DB. The kernel must manage memory lifecycle.
- Hermes: MEMORY.md + USER.md as kernel-maintained files
- Gormes: Goncho as in-binary SQLite memory

### 5.4 Self-Evolution is the Target State

Static prompts are brittle. The OS should improve with use.
- Hermes GEPA: Prompt optimization through iterative evaluation
- Voyager: Skill library grows with experience

### 5.5 Security is Layered, Not Bolted On

Prompt-level filters → Runtime approval gates → OS-level isolation.
- Hermes: Three-layer security model
- Agent-C: Formal temporal guarantees
