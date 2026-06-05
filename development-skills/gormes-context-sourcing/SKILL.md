---
name: gormes-context-sourcing
description: Use when Gormes work needs external library, framework, upstream repo, provider API, or reference-project context before planning or implementation.
---

# Gormes Context Sourcing

## Overview

Best context is the smallest verified source slice. For Gormes, prefer source-backed evidence over memory, summaries, or broad documentation dumps.

## When to Use

Use for Gormes tasks involving:
- unfamiliar Go packages, CLIs, SDKs, providers, browser automation, or installer behavior;
- Hermes, Honcho, OpenClaw, GoClaw, or `references/go-agent-os` comparison;
- prompts asking to “reference the codebase”, “check upstream”, “use docs”, or “avoid guessing”.

Do not use this to bypass `gormes-hermes-parity`; Hermes behavior remains the product contract.

## Workflow

1. Define the exact question the reference must answer.
2. Locate the nearest source of truth:
   - in-repo Gormes code first;
   - `./hermes-agent` for Hermes behavior;
   - `references/go-agent-os/` for Go donor patterns;
   - external docs only when source is unavailable or API semantics are ambiguous.
3. Pull only narrow evidence: file path, function/type name, command output, or minimal snippet.
4. State how the evidence affects the plan.
5. Proceed with the smallest relevant Gormes skill chain.

## Quick Reference

| Need | Action |
|---|---|
| Find behavior contract | Inspect Hermes source or fixture |
| Find Go implementation shape | Inspect local Go reference donor |
| Avoid context bloat | Quote path + symbol, not whole repo |
| Resolve stale memory | Prefer filesystem and command output |
| External package risk | Check package age, repo activity, and lockfile impact |

## Common Mistakes

- Dumping entire docs into context instead of searching for one symbol.
- Treating OpenClaw or donor code as the contract when Hermes parity decides behavior.
- Installing new packages before checking whether existing dependencies or standard library suffice.
- Letting the agent infer APIs from memory when source is locally available.
