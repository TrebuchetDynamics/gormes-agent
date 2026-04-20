---
title: "Why Go + Hybrid Manifesto"
weight: 120
---

## 2. Why Go — for a Python developer

Five concrete bullets, no hype:

1. **Binary portability.** One ~12 MB static binary (CGO-free). No `uv`, `pip`, venv, or system Python on the target host. `scp`-and-run on a $5 VPS or Termux.
2. **Static types and compile-time contracts.** Tool schemas, Provider envelopes, and MCP payloads become typed structs. Schema drift is a compile error, not a silent agent-loop failure.
3. **True concurrency.** Goroutines over channels replace `asyncio`. The gateway scales to 10+ platform connections without event-loop starvation.
4. **Lower idle footprint.** Target ≈ 10 MB RSS at idle vs. ≈ 80+ MB for Python Hermes. Meaningful on always-on or low-spec hosts.
5. **Explicit trade-off.** The Python AI-library moat (`litellm`, `instructor`, heavyweight ML, research skills) stays in Python until Phase 4–5.

---

## 3. Hybrid Manifesto — the Motherboard Strategy

The hybrid is **temporary**. The long-term state is 100% Go.

During Phases 1–4, Go is the chassis (orchestrator, state, persistence, platform I/O, agent cognition) and Python is the peripheral library (research tools, legacy skills, ML heavy lifting). Each phase shrinks Python's footprint. Phase 5 deletes the last Python dependency.

Phase 3 (The Black Box) is substantially delivered as of 2026-04-20: the SQLite + FTS5 lattice (3.A), ontological graph with async LLM extraction (3.B), lexical/FTS5 recall with `<memory-context>` fence injection (3.C), semantic fusion via Ollama embeddings with cosine similarity recall (3.D), and the operator-facing memory mirror (3.D.5) are all implemented. Remaining Phase 3 work is 3.E — decay, cross-chat synthesis, and the operational-visibility mirrors (session index, insights audit, tool audit, transcript export).

Phase 1 should be read correctly: it is a tactical Strangler Fig bridge, not a philosophical compromise. It exists to deliver immediate value to existing Hermes users while preserving a clean migration path toward a pure Go runtime that owns the entire lifecycle end to end.

---

## 3.5 Build Priority Framework — The Four Systems That Matter

Based on analysis of Hermes architecture and Gormes current state, here is the build priority order. **Skip even one of these and you don't have "Hermes in Go"—you have a chatbot with tools.**

### P0: Skills System — The Learning Loop (THE SOUL)

**Why first:** This is the only truly unique thing in Hermes. Without it, Gormes is undifferentiated from any other agent framework.

**What it does:**
- Detects "this task was complex" (heuristic or LLM-based)
- Extracts a reusable pattern from conversations and actions
- Saves it as a skill (structured, versioned, improvable)
- Improves that skill over time through feedback

**Minimum viable implementation:**

```go
type SkillExtractor interface {
    IsComplex(task Task) bool                    // Detect complex work
    ExtractPattern(conv Conversation) Skill      // LLM extraction
    Save(skill Skill) error                      // SQLite storage
    Improve(skillID string, feedback Feedback)   // Iterative refinement
}
```

**Without this:** You lose compounding intelligence, differentiation, and long-term value. Gormes becomes a stateless chat interface.

**Status:** ⏳ **Not implemented.** Currently marked as Phase 5.F (Skills System port). **Elevated to P0 priority.**

---

### P1: Subagent System — Execution Isolation Model

**Why second:** Enables parallel workstreams with real isolation—a Gormes **advantage over Hermes**, which has loosely-defined subagent lifecycles.

**What it does:**
- Spawns isolated subagents for parallel tasks
- Provides resource boundaries (memory limits, timeouts)
- Maintains context isolation (no cross-contamination)
- Implements scoped cancellation (parent cancels children, but not vice versa)
- Contains failures (subagent crashes don't cascade)

**Minimum viable implementation:**

```go
type Subagent struct {
    ID       string
    Context  context.Context    // Isolated conversation context
    Cancel   context.CancelFunc // Scoped cancellation
    MemoryMB int                // Soft memory limit
    Tools    []Tool             // Restricted tool subset
    ParentID string               // For lineage tracking
}
```

**Why this beats Hermes:** Python's "isolated subagents" are loosely-defined processes. Gormes can provide **process-adjacent isolation within a single binary**—strict logical boundaries with resource accounting and deterministic cleanup.

**Status:** ⏳ Mentioned in README ("isolated subagents") but **no actual implementation**. Listed as Phase 2.E (Subagent Delegation). **Elevated to P1 priority.**

---

### P2: Multi-Platform Gateway

**Why third:** Telegram proves the pattern. Scale to Discord/Slack/WhatsApp/Signal/Email. Platform breadth matters for adoption, but it doesn't differentiate architecturally.

**Status:** 🔨 Telegram complete (2.B.1). 23 platforms pending (2.B.2–2.B.16).

---

### P3: Native Agent Loop (Phase 4)

**Why last:** The Python bridge works. Replace it only after Skills and Subagents prove the architecture is correct. Phase 4 is **optimization**, not **differentiation**.

**Status:** ⏳ Phase 4.A–4.H (Provider adapters, context engine, prompt builder, smart routing, insights, etc.)

---

### Summary: What to Build and When

| Priority | System | Differentiation | Risk if Skipped |
|----------|--------|----------------|-----------------|
| **P0** | Skills System | Compounding intelligence | Undifferentiated chatbot |
| **P1** | Subagent System | Execution isolation | Unreliable parallel work |
| **P2** | Multi-Platform Gateway | Reach | Limited user access |
| **P3** | Native Agent Loop | Performance optimization | Bridge dependency continues |

**The rule:** Build P0 before P1, P1 before P2, P2 before P3. Each layer validates the architecture for the next.
