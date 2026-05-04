---
title: "Reading List"
description: "Curated priority reading order for understanding and building agentic operating systems."
weight: 70
---

# Curated Reading List

A priority-ordered reading list for building better agentic operative systems.
Each tier builds on the previous; start at Tier 1 and work down.

---

## Tier 1 — Core Foundations (Start Here)

These papers define the fundamental concepts every subsequent paper references.

### 1.1 Agent Survey: Wang et al. 2023

| ArXiv | [2308.11432](https://arxiv.org/abs/2308.11432) |
|-------|------|

**A Survey on Large Language Model Based Autonomous Agents**

The canonical taxonomy: Profiling → Memory → Planning → Action. Every agent system paper references this framework. Read this first to understand the vocabulary.

**Key takeaway**: Agents decompose into four modules. Design each with clear boundaries.

---

### 1.2 ReAct: Yao et al. 2022

| ArXiv | [2210.03629](https://arxiv.org/abs/2210.03629) |
|-------|------|

**ReAct: Synergizing Reasoning and Acting in Language Models**

The Thought → Action → Observation loop that powers virtually every agent system today. Understanding ReAct means understanding the default control flow of modern agents.

**Key takeaway**: Interleaving reasoning and action improves both. Don't separate "thinking" from "doing."

---

### 1.3 Generative Agents: Park et al. 2023

| ArXiv | [2304.03442](https://arxiv.org/abs/2304.03442) |
|-------|------|

**Generative Agents: Interactive Simulacra of Human Behavior**

The canonical memory architecture: Memory Stream → Retrieval → Reflection → Planning. The foundation for every agent memory paper since.

**Key takeaway**: Memory is a pipeline — store, retrieve, reflect, plan. Each step matters.

---

## Tier 2 — How Agents Get Better

Once you understand the basic loop, learn how agents improve themselves.

### 2.1 Reflexion: Shinn et al. 2023

| ArXiv | [2303.11366](https://arxiv.org/abs/2303.11366) |
|-------|------|

**Reflexion: Language Agents with Verbal Reinforcement Learning**

Self-critique as verbal RL. Agents store reflections in memory, learn from past failures without gradient descent.

**Key takeaway**: The Act → Critique → Revise → Act loop adds learning to the basic ReAct cycle.

---

### 2.2 Voyager: Wang et al. 2023

| ArXiv | [2305.16291](https://arxiv.org/abs/2305.16291) |
|-------|------|

**Voyager: An Open-Ended Embodied Agent with Large Language Models**

Skill libraries that grow with experience. The code-as-action pattern that enables composition and reuse.

**Key takeaway**: Store reusable code, not static instructions. Skills should be executable and composable.

---

### 2.3 SWE-Agent: Yang et al. 2024

| ArXiv | [2405.15793](https://arxiv.org/abs/2405.15793) |
|-------|------|

**SWE-Agent: Agent-Computer Interfaces Enable Automated Software Engineering**

The Agent-Computer Interface (ACI) concept: interface design matters as much as model capability. Few high-level actions beat raw shell access.

**Key takeaway**: Design the tool interface for how the model thinks, not how humans use computers.

---

## Tier 3 — Multi-Agent & OS Design

Now think at system scale — multiple agents, OS-level architecture.

### 3.1 MetaGPT: Hong et al. 2024

| ArXiv | [2308.00352](https://arxiv.org/abs/2308.00352) |
|-------|------|

**MetaGPT: Meta Programming for a Multi-Agent Collaborative Framework**

SOP-based multi-agent collaboration. Organizational structure improves agent team performance.

**Key takeaway**: Define roles and communication protocols. Free-form collaboration is chaotic.

---

### 3.2 AIOS: Mei et al. 2024

| ArXiv | [2403.16971](https://arxiv.org/abs/2403.16971) |
|-------|------|

**AIOS: LLM Agent Operating System**

The LLM-as-kernel paradigm. Six kernel modules for agent scheduling, context, memory, tools, and access control.

**Key takeaway**: Treat agents as OS processes. The kernel manages resources shared across agents.

---

### 3.3 MACNet: Qian et al. 2025

| ICLR 2025 | 基于大型语言模型的多智能体协作扩展研究 |

Collaboration scaling laws: more agents help (up to ~100). Topology matters more than count.

**Key takeaway**: Don't assume "more agents = better." There's a sweet spot, and how they're connected matters.

---

### 3.4 Architecting AgentOS: Li et al. 2026

| ArXiv | [2602.20934](https://arxiv.org/abs/2602.20934) |
|-------|------|

**Architecting AgentOS: From Token-Level Context to Emergent System-Level Intelligence**

OS abstractions mapped to LLM constructs — paging, interrupts, scheduling for context.

**Key takeaway**: The frontier. Classical OS theory applied to LLM-native systems.

---

## Tier 4 — Memory & Safety (Deep Dives)

Now specialize in the hardest problems.

### 4.1 AgeMem (2026)

| ArXiv | [2601.01885](https://arxiv.org/abs/2601.01885) |
|-------|------|

**Agentic Memory: Learning Unified Long-Term and Short-Term Memory Management**

The agent decides what to remember, not a heuristic pipeline. Memory operations as tools.

**Key takeaway**: Memory management should be learned behavior, not engineered rules.

---

### 4.2 Memory Survey (2026)

| ArXiv | [2603.07670](https://arxiv.org/abs/2603.07670v1) |
|-------|------|

**Memory for Autonomous LLM Agents: Mechanisms, Evaluation, and Emerging Frontiers**

The definitive survey. Three-dimensional taxonomy, five mechanism families. Use this as your memory design reference.

**Key takeaway**: Formal framework for memory: write-manage-read loop with temporal, representational, and policy dimensions.

---

### 4.3 MOSAIC (2025)

**MOSAIC: Aligning Multi-Step Tool Use Safely**

Plan → Check → Act/Refuse as default safety loop. Explicit safety reasoning, refusal as first-class action.

**Key takeaway**: Safety isn't bolted on — it's embedded in the control flow.

---

### 4.4 Agent-C (2025)

| ArXiv | [2512.23738](https://arxiv.org/abs/2512.23738) |
|-------|------|

**Agent-C: Formal Temporal Safety Guarantees for LLM Agents**

Formal methods for agent safety. DSL for temporal constraints ("authenticate BEFORE query"). 100% safety conformance.

**Key takeaway**: Safety can be proven, not just hoped for.

---

## Tier 5 — Production & Deployment

Where research meets reality.

### 5.1 OpenCode Architecture

**Repository**: [github.com/anomalyco/opencode](https://github.com/anomalyco/opencode)

Study the event-driven architecture, 75+ provider abstraction, and SQLite session persistence. A production reference implementation for many of the patterns described in the academic papers.

**Key takeaway**: Event bus decouples I/O from processing. Provider abstraction enables zero-lock-in.

---

### 5.2 Hermes Agent Security Model

**Repository**: [github.com/NousResearch/hermes-agent](https://github.com/NousResearch/hermes-agent)

Study the layered security architecture (prompt → runtime gates → OS isolation), the GEPA self-evolution engine, and the 15+ channel gateway.

**Key takeaway**: Multi-layer defense. Self-evolution as target state.

---

### 5.3 MCP Specification

**Spec**: [modelcontextprotocol.io](https://modelcontextprotocol.io)

The emerging standard for agent-tool interaction. Understand the tool/resource/prompt/roots primitives.

**Key takeaway**: Standardize tool interfaces. Every agent benefits from every tool.

---

## Quick Reference: By Topic

### Agent Loops
1. ReAct (Yao 2022) — start here
2. Reflexion (Shinn 2023) — adds self-critique
3. Plan-and-Execute — separates planning from execution

### Memory
1. Generative Agents (Park 2023) — foundational
2. Memory Survey (2026) — comprehensive reference
3. AgeMem (2026) — agentic, learned memory

### Multi-Agent
1. MetaGPT (ICLR 2024) — SOP-based
2. MACNet (ICLR 2025) — scaling laws
3. MegaAgent (ACL 2025) — dynamic generation

### OS Design
1. AIOS (NeurIPS 2024) — LLM kernel
2. ACOS (2024) — agent-centric
3. Architecting AgentOS (2026) — frontier

### Safety
1. MOSAIC (2025) — plan-check-act
2. Agent-C (2025) — formal guarantees
3. IntentGuard — runtime gates

### Tools
1. SWE-Agent (NeurIPS 2024) — ACI concept
2. ToolACE (ICLR 2025) — data generation
3. MCP Spec — standard protocol

---

## Prioritized Timeline

| Week 1 | Tier 1: Agent Survey + ReAct + Generative Agents |
| Week 2 | Tier 2: Reflexion + Voyager + SWE-Agent |
| Week 3 | Tier 3: MetaGPT + AIOS + MACNet |
| Week 4 | Tier 4: AgeMem + Memory Survey + MOSAIC |
| Week 5 | Tier 5: OpenCode architecture + Hermes security model + MCP spec |

**Total**: ~5 weeks for comprehensive understanding at 2-3 papers per week.
