---
title: "Foundational Agent Architectures"
description: "Core agent loop patterns: ReAct, Reflexion, Tree of Thoughts, Voyager, and Generative Agents."
weight: 10
---

# Foundational Agent Architectures

The modern agentic OS inherits a set of foundational architectures developed between 2022–2024.
These define the core loop patterns, reasoning strategies, and memory primitives used by virtually all agent systems today.

## 1. ReAct — The Foundational Loop

| Field | Value |
|-------|-------|
| **Title** | ReAct: Synergizing Reasoning and Acting in Language Models |
| **Authors** | Shunyu Yao, Jeffrey Zhao, et al. (Princeton, Google Research) |
| **Venue** | ICLR 2023 |
| **ArXiv** | [2210.03629](https://arxiv.org/abs/2210.03629) |

**Core Pattern**: `Thought → Action → Observation → Thought → ...`

ReAct interleaves explicit reasoning traces with tool-calling actions. Reasoning helps induce, track, and update action plans while actions gather external information that grounds reasoning. This synergy proved transformative:

- 34% improvement on ALFWorld interactive decision-making
- 10% improvement on WebShop vs. imitation/RL baselines
- Ablation: removing reasoning or action both degrade performance

**Why it matters**: ReAct is the default control loop in 7 of 13 open-source coding agents studied as of 2025. LangGraph, CrewAI, AutoGen, and OpenCode all default to ReAct or ReAct-derived loops.

**Variants**:
- **ReflAct** (EMNLP 2025): Adds goal-state reflection to reduce hallucination; 93.3% success in ALFWorld (+27.7%)
- **Autono**: ReAct + timely abandonment strategy + probabilistic penalty mechanism for robustness

---

## 2. Reflexion — Verbal Reinforcement Learning

| Field | Value |
|-------|-------|
| **Title** | Reflexion: Language Agents with Verbal Reinforcement Learning |
| **Authors** | Noah Shinn, et al. |
| **Venue** | NeurIPS 2023 |
| **ArXiv** | [2303.11366](https://arxiv.org/abs/2303.11366) |

**Core Pattern**: `Act → Critique → Revise → Act`

Instead of gradient-based RL, Reflexion uses linguistic feedback stored in episodic memory. When an action fails, the agent generates a verbal reflection about what went wrong and stores it. Future attempts retrieve relevant reflections as context.

- 91% Pass@1 on HumanEval (vs. GPT-4's 80%)
- Ablation shows reflection + memory jointly critical — removing either drops performance

**Why it matters**: Self-critique with memory is the core pattern used by Replit, Devin, Cursor's agent mode, and modern coding scaffolds. Agents learn from failure without weight updates.

**Architecture**: Three models work together:
1. **Actor**: Generates text and actions (LLM)
2. **Evaluator**: Produces scalar reward from environment
3. **Self-Reflection**: Converts trajectory + reward into verbal summary stored in episodic memory

---

## 3. Tree of Thoughts (ToT) — Deliberate Search

| Field | Value |
|-------|-------|
| **Title** | Tree of Thoughts: Deliberate Problem Solving with Large Language Models |
| **Authors** | Yao et al. (2023) |
| **ArXiv** | [2305.10601](https://arxiv.org/abs/2305.10601) |

**Core Pattern**: Frame reasoning as BFS/DFS search over a tree of thought nodes.

Each node is a reasoning path scored by the LLM itself. When one path fails, the agent backtracks to explore alternatives. This breaks the linear constraint of ReAct — enabling exploration of multiple solution branches.

**Key Operations**:
- **Generate**: Produce k candidate "next thoughts" from current state
- **Evaluate**: Score each candidate (value of continuing from this state)
- **Select/BFS/DFS**: Choose how to traverse the tree

**Why it matters**: Enables structured exploration when multiple valid approaches exist. Complements ReAct for tasks with branching solution spaces (math proofs, creative writing, game strategies).

**Derivative**: **LATS** (Language Agent Tree Search) — Combines ReAct + ToT with Monte Carlo Tree Search (MCTS). External tools generate states; self-evaluation guides search.

---

## 4. Voyager — Skill Libraries

| Field | Value |
|-------|-------|
| **Title** | Voyager: An Open-Ended Embodied Agent with Large Language Models |
| **Authors** | Guanzhi Wang, Yuqi Xie, et al. |
| **Venue** | NeurIPS 2023 Workshops |
| **ArXiv** | [2305.16291](https://arxiv.org/abs/2305.16291) |

**Core Pattern**: Store reusable, executable code as skills; compose them; learn from execution feedback.

Three components:
1. **Automatic curriculum**: Progressive difficulty exploration
2. **Skill library**: Ever-growing repository of executable code programs
3. **Iterative prompting**: Environment feedback + execution errors + self-verification loop

Results in Minecraft: 3.3× more unique items discovered, 15.3× faster tech tree unlocking vs. prior SOTA.

**Why it matters**: Voyager pioneered the skill library pattern — storing interpretable, reusable, compositional action programs. This directly maps to OpenCode's skill system, Hermes' GEPA self-evolution engine, and IDE agent skill files.

**Key insight**: Using code as the action space (not low-level motor commands) enables composition, reuse, and interpretability.

---

## 5. Generative Agents — Memory Architecture

| Field | Value |
|-------|-------|
| **Title** | Generative Agents: Interactive Simulacra of Human Behavior |
| **Authors** | Joon Sung Park, Joseph O'Brien, Carrie Cai, Percy Liang, Michael Bernstein (Stanford/Google) |
| **Venue** | UIST 2023 |
| **ArXiv** | [2304.03442](https://arxiv.org/abs/2304.03442) |
| **Citations** | 1,300+ |

**Core Architecture**:
1. **Memory stream**: Natural language record of all experiences, timestamped
2. **Reflection**: Periodic synthesis of memories into higher-level abstractions
3. **Planning**: Goal-driven behavior generation from reflection output

**Retrieval mechanism**: Memories are retrieved by combining:
- **Recency**: Recent events weighted higher
- **Importance**: LLM judges significance at storage time
- **Relevance**: Embedding similarity to current context

**Demonstration**: 25 agents autonomously coordinated a Valentine's Day party — planning, invitations, decor, timing — with zero human scripting.

**Why it matters**: This is the canonical reference for agent memory system design. Nearly every subsequent memory paper (AgeMem, Synapse, A-Mem, GSW) builds on or contrasts with this architecture. The memory stream → retrieval → reflection pipeline maps directly to Goncho's design in Gormes.

---

## 6. Plan-and-Execute

**Core Pattern**: Split planning from execution. Planner produces full plan upfront; executor walks step-by-step; re-plan only on failure.

**Implementations**: BabyAGI (2023), LangChain Plan-and-Execute, LlamaIndex planners, Devin (Cognition AI).

**Trade-offs vs. ReAct**:
- **Advantage**: Lower token cost (doesn't re-deliberate at every step), better for long-horizon tasks
- **Disadvantage**: Less adaptive to surprises, initial plan errors propagate

**Why it matters**: Choice between ReAct and Plan-and-Execute is the primary architectural decision when designing an agent loop. Gormes uses ReAct for interactive TUI turns and Plan-and-Execute for background/multi-step tasks.

---

## Architecture Comparison

| Architecture | Control Flow | Best For | Limitations |
|---|---|---|---|
| **ReAct** | Sequential Think→Act→Observe | Interactive tasks, tool use | Linear, no backtracking |
| **Reflexion** | Act→Critique→Revise | Self-improvement, debugging | Requires evaluator signal |
| **Tree of Thoughts** | BFS/DFS over thought tree | Multi-solution problems | High token cost |
| **LATS** | MCTS + ReAct + ToT | Complex exploration | Highest token cost |
| **Plan-and-Execute** | Plan→Step→Re-plan | Long-horizon tasks | Inflexible to surprises |
| **Voyager** | Curriculum-driven skill building | Lifelong learning | Requires execution environment |

## How These Inform Agentic OS Design

| OS Component | Architectural Inspiration |
|---|---|
| **Turn loop** | ReAct cycle with Reflexion self-critique |
| **Skill system** | Voyager's skill library + code-as-action |
| **Memory** | Generative Agents' memory stream + retrieval |
| **Long tasks** | Plan-and-Execute with dynamic re-planning |
| **Complex decisions** | ToT/LATS for branching exploration |
