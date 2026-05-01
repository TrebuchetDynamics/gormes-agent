---
title: "Next Slices"
weight: 30
aliases:
  - /building-gormes/next-slices/
---

# Next Slices

This page is generated from the canonical progress file and lists the highest
leverage contract-bearing roadmap rows to execute next.

The ordering is:

1. unblocked `P0` handoffs;
2. active `in_progress` rows;
3. `fixture_ready` rows;
4. unblocked rows that unblock other slices;
5. remaining `draft` contract rows.

Use this page when choosing implementation work. If a row is too broad, split
the row in `progress.json` before assigning it.

If no slices are listed, the next correct action is planner work: choose one
planned row from `progress.json` or a phase page and add enough contract detail
for it to appear here. Do not infer that an empty generated list means the
roadmap is complete.

<!-- PROGRESS:START kind=next-slices -->
| Phase | Slice | Contract | Trust class | Fixture | Why now |
|---|---|---|---|---|---|
| 5 / 5.A | Tool descriptor layer (OperationSpec) | Every tool in the registry carries a declarative descriptor (OperationSpec) that generates model schemas, CLI commands, gateway slash commands, doctor checks, and audit taxonomy from one source | operator, gateway, child-agent, system | `internal/tools/operation_spec_test.go` | P0 handoff; needs contract proof before closeout. |
| 5 / 5.J | Shell blocklist (36+ dangerous patterns) | The shared tool executor blocks 36+ dangerous shell patterns before execution, with category-specific evidence and override policies | operator, gateway, child-agent, system | `internal/tools/shell_blocklist_test.go` | P0 handoff; needs contract proof before closeout. |
| 5 / 5.J | Filesystem scoping (folder-level read/write restrictions) | File tools enforce folder-level read/write scope restrictions so agents cannot access paths outside configured boundaries | operator, gateway, child-agent | `internal/tools/filesystem_scope_test.go` | P0 handoff; needs contract proof before closeout. |
| 5 / 5.J | Permission approval UX (inline y/n/always) | Dangerous actions trigger an inline approval prompt (y/n/always) with clear command preview, risk category, and persistent preference storage | operator, gateway | `internal/tools/approval_ux_test.go` | P0 handoff; needs contract proof before closeout. |
| 5 / 5.J | Trust-class enforcement in shared tool executor | The shared tool executor rejects tool calls from disallowed trust classes before a handler runs, preventing gateway/child-agent callers from exercising operator-local tools | operator, gateway, child-agent, system | `internal/tools/trust_class_test.go` | P0 handoff; needs contract proof before closeout. |
| 6 / 6.G | 6 typed memory categories with confidence scoring | Identity, preference, goal, habit, episode, reflection with confidence/durability scoring and conflict resolution | operator, system | `internal/goncho/typed_memory_test.go` | Unblocks Memory auto-extraction, Memory consolidation. |
| 6 / 6.I | Regex-based auto-link extraction + brain-first lookup | Markdown links, wikilinks, qualified wikilinks auto-extracted; typed inference; brain-first 5-step lookup | operator, system | `internal/goncho/auto_link_test.go` | Unblocks Compiled truth pattern, Tiered enrichment. |
| 4 / 4.A | xAI Grok provider adapter | Gormes can route tool-call turns through xAI Grok API with native request/response mapping, streaming, and error classification | system | `internal/hermes/grok_adapter_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 4 / 4.A | LM Studio provider adapter | Gormes can route turns through LM Studio local inference server with OpenAI-compatible request/response mapping | system | `internal/hermes/lmstudio_adapter_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 4 / 4.K | Resilient provider chain dispatch | DeepSeek → OpenAI → Anthropic → Grok → Ollama resilient routing with chain failure detection | operator, system | `internal/hermes/fallback_chain_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
