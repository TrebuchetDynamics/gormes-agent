---
title: "Agent Evaluation Benchmarks"
description: "Complete landscape of AI agent evaluation benchmarks: coding, web, general, multi-agent, safety, tool-use, long-horizon, memory."
weight: 55
---

# Agent Evaluation Benchmarks — Complete Landscape

## 1. Coding Benchmarks

### SWE-bench Family

| Variant | Tasks | Top Score | Status |
|---------|-------|-----------|--------|
| **Verified** | 500 | 80.9% (Claude Opus 4.5) | ⚠️ Contaminated — OpenAI stopped reporting |
| **Pro** | 1,865 | 59% (agent systems) | ✅ Contamination-resistant |
| **Multilingual** | 300 | ~45% | 9 languages |
| **Live** | 1,565+ | ~40% | Monthly rolling, contamination-free |

**Critical insight**: The 35-point gap between Verified (80.9%) and Pro (45.9%) for the same model reveals Verified is heavily contaminated.

**SWE-bench Pro leaderboard** (standardized scaffolding):
- Claude Opus 4.5: 45.9%, Claude Sonnet 4.5: 43.6%, Gemini 3 Pro: 43.3%, GPT-5 High: 41.8%

**Agent systems (custom scaffolding)** outperform raw models:
- GPT-5.3-Codex CLI: 57.0%, Claude Code + Opus 4.5: 55.4%

### Other Coding Benchmarks

| Benchmark | Tasks | Top Score | Saturation |
|-----------|-------|-----------|------------|
| **HumanEval+** | 164 | ~99% | Saturated |
| **MBPP+** | 974 | ~90% | Near-saturated |
| **BigCodeBench** | 1,140 | ~35% | Active |
| **LiveCodeBench** | 1,000+ | 91.7% | Rolling, clean |
| **SWE-rebench** | Various | ~40% | Realistic multi-file |
| **Commit0** | New | Emerging | Code from intent |

---

## 2. Web/GUI Benchmarks

### WebArena Family

| Benchmark | Tasks | Best Score | Focus |
|-----------|-------|-----------|-------|
| **WebArena** | 812 | 71.6% (OpAgent) | Multi-step web tasks |
| **VisualWebArena** | 910 | ~19.8% | Visual understanding |
| **WebArena-Lite** | 241 | ~60% | Lighter eval |

**WebArena Leaderboard (April 2026)**:
- Claude Mythos Preview: 68.7%, GPT-5.4 Pro: 65.8%, Claude Opus 4.6: 64.5%
- OpAgent (CodeFuse AI) leads at 71.6% with Planner-Grounder-Reflector-Summarizer pipeline

### OSWorld (Desktop)

- 69+ real OS tasks on actual VMs
- Best: 76.26% (OSAgent), Human baseline: 72.4%
- Hardest task type: LibreOffice Calc (~8%)

### Other Web/GUI

| Benchmark | Tasks | Notes |
|-----------|-------|-------|
| **Mind2Web** | 2,350 | 137 real websites |
| **Mind2Web 2** | 130 live | NeurIPS 2025, long-horizon agentic search |
| **Online-Mind2Web** | 300 live | Real-time browsing + cross-site synthesis |
| **WebLINX** | 2,337 | Multimodal web agent |
| **OmniACT** | 9,802 | Large-scale UI automation |
| **WindowsAgentArena** | Windows desktop | ~25% best |
| **Android-in-the-Wild** | Real devices | ~90%+ (saturated) |
| **Computer Agent Arena** | Human-centric | 2,201 votes head-to-head |

---

## 3. General Agent Benchmarks

### GAIA (General AI Assistants)

The benchmark for real-world assistant capability beyond coding.

| Level | Steps | Human | Best Agent |
|-------|-------|-------|-----------|
| Level 1 | ~10 | 92% overall | 82.1% (Claude Sonnet 4.5 + HAL) |
| Level 2 | ~20 | — | 74.4% |
| Level 3 | 30+ | — | 65.4% (HAL), 61% (Writer's Action Agent) |

**Key insight**: HAL scaffolding framework adds ~30 points on GAIA. Tool orchestration matters more than raw model capability.

### τ-bench (Tool-Agent-User)

| Domain | Best Score | Model |
|--------|-----------|-------|
| Retail | 89.2% | Claude Mythos |
| Airline | 87.5% | Claude Sonnet 4.6 |
| Telecom | ~80% | Various |

**τ³-bench adds**: Voice full-duplex, knowledge retrieval (698 banking docs), 75+ task fixes.

### AgentBench

8 environments (OS, databases, knowledge graphs, web navigation). 29+ LLMs benchmarked. GPT-4 leads 6/8 datasets.

### Other General Benchmarks

| Benchmark | Focus | Status |
|-----------|-------|--------|
| **AgentGym** | Unified agent environment | Active |
| **ALFWorld** | Text adventure | Mature |
| **ScienceWorld** | Scientific experimentation | Active |
| **WebShop** | E-commerce | Mature |
| **WorkArena** | Enterprise workflows | 23k+ tasks |

---

## 4. Multi-Agent Benchmarks

| Benchmark | What It Tests | Key Finding |
|-----------|---------------|-------------|
| **ChatDev** | Software company simulation | ~88% executability (self-reported) |
| **MetaGPT** | SOP-based multi-agent | 3.75/5 executability vs ChatDev's 2.25 |
| **Silo-Bench** | Distributed coordination | Agents fail at cross-agent synthesis |
| **AgentsNet** | 100-agent networks | Self-organization, communication |

---

## 5. Safety Benchmarks

### AgentHarm

110 malicious agent tasks, 11 harm categories.

| Model | Harm Score (no jailbreak) | Harm Score (jailbroken) |
|-------|--------------------------|------------------------|
| Claude 3.5 Sonnet | 13.5% | 68.7% |
| GPT-4o | 48.4% | 72.7% |
| Mistral Large 2 | 82.2% | — |

**Key finding**: Standard refusal training doesn't transfer to agentic settings. Claude's ~3% conversational HarmBench ASR becomes ~15% in agentic AgentHarm.

### Other Safety Benchmarks

| Benchmark | Focus | Best Result |
|-----------|-------|------------|
| **HarmBench** | 400+ harmful behaviors | Automated red teaming |
| **S-Eval** | Safety-specific evaluation | Adversarial pressure |
| **AgentDojo** | Prompt injection multi-environment | Diverse attack vectors |
| **ASB** (Anthropic) | 4 attack categories | Baseline established |

---

## 6. Tool-Use / Function Calling

### BFCL (Berkeley Function Calling Leaderboard)

| Version | Focus | Best Score |
|---------|-------|-----------|
| V1 | Simple, parallel, multiple calls | Varies |
| V2 Live | Real-world APIs | 2,251+ pairs |
| V3 | Multi-turn, stateful | 76.7% (GLM 4.5) |
| V4 | Web search, memory, formats | Agentic capabilities |

**Notable**: Claude Opus 4 scores 25.3% (bottom) on BFCL v3 despite dominating other benchmarks — its conversational wrapping trips AST parsing even when tool selection is correct.

### Other Tool Benchmarks

| Benchmark | Scale | Feature |
|-----------|-------|---------|
| **ToolBench** | 16,000+ APIs | Generalization to unseen tools |
| **API-Bank** | 264 tasks | 3-level: call/search/plan |
| **ToolQA** | Various | QA over structured tools |
| **StableToolBench** | Varies | Reproducible eval |
| **Nexus** | 1,500 | Paired with NexusRaven model |

---

## 7. Long-Horizon Planning

### TravelPlanner

1,225 tasks, 13 coupled constraints (8 commonsense, 5 hard).

| Model | Score | Year |
|-------|-------|------|
| GPT-4 (launch) | 0.6% | 2024 |
| Planner-R1 32B | **56.9%** | Dec 2025 |

A single budget overage invalidates an otherwise perfect plan.

### DeepPlanning (2026)

| Domain | Best Accuracy |
|--------|--------------|
| Travel | 35.0% (GPT-5.2) |
| Shopping | 60.0% (Gemini 3 Flash) |

Models with explicit reasoning significantly outperform non-reasoning (85.8 vs 54.3 composite).

---

## 8. Memory Benchmarks

| Benchmark | Focus | Status |
|-----------|-------|--------|
| **LoCoMo** | Long-term context memory | Emerging |
| **MemBench** | Extended conversations | Developing |
| **LongMemEval** | Extended context | Active |

*Memory benchmarks are less mature. BFCL v4 introduced memory-related tasks as part of agentic evaluation.*

---

## Benchmark Saturation Map

| Saturated (>90%) | Near (70-90%) | Active (<70%) |
|---|---|---|
| HumanEval, MBPP | SWE-bench Verified ⚠️ | **SWE-bench Pro (59%)** |
| Android-in-the-Wild | τ-bench Retail (89%) | **GAIA Level 3 (65%)** |
| | WebArena (71%) | **TravelPlanner (57%)** |
| | | **DeepPlanning Travel (35%)** |

## Critical Nuances

1. **Verified ≠ Real**: 35-point gap between Verified and Pro for same model
2. **Scaffolding matters**: HAL adds ~30 points on GAIA
3. **Agentic safety ≠ chatbot safety**: 3% → 15% jump on agentic HarmBench
4. **Function calling ≠ general capability**: Claude top on τ-bench, bottom on BFCL
