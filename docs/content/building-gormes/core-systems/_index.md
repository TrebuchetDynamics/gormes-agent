---
title: "Core Systems"
weight: 10
---

# Core Systems

Four subsystems that make Gormes a runtime instead of a chatbot wrapper.

Use these pages for the stable runtime model. Use the phase pages when you need
the current delivery ledger, status, or execution queue.

| System | Owns | Page | Roadmap |
|---|---|---|---|
| Learning Loop | Skill detection, distillation, review, retrieval, and feedback | [learning-loop](./learning-loop/) | [Phase 6](../architecture_plan/phase-6-learning-loop/) |
| Memory | Local recall, graph provenance, scoped search, mirrors, and Goncho substrate | [memory](./memory/) | [Phase 3](../architecture_plan/phase-3-memory/) |
| Tool Execution | Typed operation registry, schema parity, trust classes, and tool loops | [tool-execution](./tool-execution/) | [Phase 2.A](../architecture_plan/phase-2-gateway/) + [Phase 5](../architecture_plan/phase-5-final-purge/) |
| Gateway | Platform adapters, command policy, session routing, delivery, hooks, and active-turn behavior | [gateway](./gateway/) | [Phase 2.B-2.G](../architecture_plan/phase-2-gateway/) |

Miss any one of these and you don't have "Hermes in Go" — you have a chatbot with tools.

## Before Editing

Every core-system change should name the upstream or Gormes-native contract, the
caller trust class, the degraded-mode surface, and the fixture that proves it.
Those fields belong in [Progress Schema](../autoloop/progress-schema/) before a row is
assigned through [Agent Queue](../autoloop/agent-queue/).
