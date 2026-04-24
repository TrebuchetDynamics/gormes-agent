---
title: "Blocked Slices"
weight: 36
---

# Blocked Slices

This page is generated from canonical `progress.json` rows that declare
`blocked_by`.

Use it to avoid assigning work before the dependency chain is ready.

<!-- PROGRESS:START kind=blocked-slices -->
| Phase | Slice | Blocked by | Ready when | Unblocks |
|---|---|---|---|---|
| 2 / 2.F.5 | Steer slash command registry + queue fallback | 2.E.2 | 2.E.2 is complete and the shared CommandDef registry is stable for gateway commands. | Mid-run steer injection between tool calls, Gateway-handled slash commands bypass active-session guard |
| 3 / 3.E.7 | Cross-chat deny-path fixtures | Honcho-compatible scope/source tool schema | Honcho-compatible scope/source tool schema is complete and exposes source allowlist semantics. | Cross-chat operator evidence, parent_session_id lineage for compression splits |
| 4 / 4.B | ContextEngine interface + status tool contract | Provider interface + stream fixture harness | Provider interface + stream fixture harness can replay context status without live provider calls. | Compression token-budget trigger + summary sizing, Tool-result pruning + protected head/tail summary |
| 4 / 4.H | Provider-side resilience | Provider interface + stream fixture harness | Provider interface + stream fixture harness is available for resilience fixture coverage. | Retry-After header parsing + HTTPError hint, Kernel retry honors Retry-After hint, Provider rate guard + budget telemetry |
| 6 / 6.C | Portable SKILL.md format | Phase 2.G skills runtime | Phase 2.G skills runtime is complete and the parser/store seam is stable enough for versioned metadata. | LLM-assisted pattern distillation, Hybrid lexical + semantic lookup, Skill effectiveness scoring |
<!-- PROGRESS:END -->
