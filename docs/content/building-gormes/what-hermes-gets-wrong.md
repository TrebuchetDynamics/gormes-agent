---
title: "What Hermes Gets Wrong"
weight: 30
---

# What Hermes Gets Wrong

The gaps that justify Gormes's existence. This is not a competitive teardown — upstream Hermes is excellent research. These are the operational-runtime problems Gormes is positioned to solve.

## 1. Python dependency stack

Hermes requires `uv`, a venv, Python 3.11, and platform-specific extras (`.[all]`, `.[termux]`). Every host install is a moving target.

**Gormes's answer:** one binary. No runtime. `scp` and run.

## 2. Execution chaos

Hermes tool execution is dynamic Python — flexible but hard to reason about. Schemas drift. Subprocess boundaries blur.

**Gormes's answer:** typed Go interfaces, controlled execution, bounded side effects.

## 3. Subagents are conceptual

Upstream documents subagent delegation but lacks a robust lifecycle — processes spin up, state leaks, cancellation is best-effort.

**Gormes's answer (planned, Phase 2.E):** real subagents with explicit `context.Context`, cancel funcs, memory scoping, resource limits:

```go
type Agent struct {
    ID      string
    Context context.Context
    Cancel  context.CancelFunc
}
```

## 4. Startup + recovery cost

Python startup is measured in seconds on every cold boot. Recovery after a crash requires re-scanning venv + re-importing half the world.

**Gormes's answer:** instant start. Crash → restart → continue, measured in milliseconds.

## The positioning

> *Hermes-class agents, without Python.*
>
> The production runtime for self-improving agents — not a research artifact. An **industrial-grade agent runtime** that runs 24/7 without babysitting.
