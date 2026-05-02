---
title: "Academic Research Survey"
description: "Survey of academic papers and technical research on building better agentic operative systems."
weight: 0
slug: "/papers/"
---

# Agentic Operating Systems — Academic Research Survey

This directory contains a comprehensive survey of academic papers, technical research, and open-source implementations relevant to building better agentic operative systems like OpenCode, Hermes Agent, and Claude Code.

Research covers English and Chinese (中文) academic sources across 2022–2026, with findings organized by domain.

## Contents

| File | Topic | Coverage |
|------|-------|----------|
| [Foundational Architectures](foundational-architectures/) | ReAct, Reflexion, ToT, Voyager, Generative Agents | Core agent loop patterns |
| [Agentic OS Design](agentic-os-design/) | AIOS, ACOS, OpenCode, Hermes Agent | OS-level agent architecture |
| [Agent Memory Systems](agent-memory-systems/) | AgeMem, Synapse, A-Mem, H-MEM, MemGen | Episodic, semantic, hierarchical memory |
| [Multi-Agent & Tools](multi-agent-and-tools/) | MetaGPT, CAMEL, MCP, ToolACE | Orchestration & tool ecosystems |
| [Safety & Deployment](safety-and-deployment/) | MOSAIC, Agent-C, OpenSandbox, production patterns | Guardrails & production reliability |
| [Benchmarks Landscape](benchmarks-landscape/) | SWE-bench, GAIA, τ-bench, BFCL, TravelPlanner, AgentHarm | Complete evaluation landscape |
| [Agent Infrastructure](infrastructure-papers/) | Context mgmt, model serving, state, observability, scaling | Production infrastructure |
| [European Research](european-research/) | DFKI, Fraunhofer, INRIA, ETH Zurich, Oxford, EU projects | German, French, Swiss, UK, EU |
| [Asian Research (日韓)](asian-research/) | Sakana AI, KAIST, Naver, Kakao, Preferred Networks | Japanese & Korean research |
| [Chinese Research (中文综述)](chinese-research/) | 智能体操作系统、多智能体协作、工具学习 | Complete Chinese-language survey |
| [Reading List](reading-list/) | Curated priority reading order | Where to start |

## Key Themes

| Theme | Core Insight |
|-------|-------------|
| **Agent Loop** | Thought→Action→Observation cycle (ReAct, 2022) remains foundational; variants add reflection, tree search, and hierarchical decomposition |
| **OS Design** | AIOS (2024) establishes "LLM as OS kernel" paradigm with scheduler, context manager, tool manager, and access control |
| **Memory** | Shift from passive vector stores to agentic memory with autonomous store/retrieve/forget decisions (AgeMem, 2026) |
| **Multi-Agent** | MetaGPT (ICLR 2024) shows SOP-based collaboration; MACNet (ICLR 2025) verifies collaboration scaling laws |
| **Tool Systems** | MCP standardizes agent-tool interface; SWE-Agent's ACI concept shows interface design matters as much as model quality |
| **Safety** | MOSAIC (plan→check→act loop), Agent-C (formal temporal guarantees), IntentGuard (training-free runtime gates) |
| **Deployment** | Event-driven pub/sub beats polling; checkpointing enables pause/resume; multi-channel gateways with shared kernel |

## Research Methodology

Sources surveyed:
- **English**: ArXiv, Google Scholar, NeurIPS, ICLR, ACL, EMNLP, ICML proceedings (2022–2026)
- **Chinese**: 知网 (CNKI), 万方, ArXiv Chinese teams, CCF-recommended venues
- **Open-source**: GitHub repositories, documentation, architecture RFCs

## Contributing

Gormes is a Go-native Hermes-compatible agent runtime. This research informs its architecture decisions.
If you find papers or systems not yet covered, open an issue or PR.
