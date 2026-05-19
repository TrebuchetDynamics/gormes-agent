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
| 5 / 5.A | Hermes tool tail strict-fidelity source-pair expansion | Classify remaining unmapped Hermes tools into covered Gormes tool rows, focused builder rows, or explicit exclusions. The pass must cover web/search providers, voice/TTS/STT tools, video/image tools, environment backends, tool result storage, process/zombie guards, URL and website policy helpers, and x_search auth behavior without hiding them behind the existing broad 61-tool row. | operator, system | `internal/tools strict-fidelity tail mapping fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 8 / 8.C | Strict-fidelity upstream test-suite classifier | Classify the strict-fidelity test blockers by upstream suite, exact Hermes source under test, current Gormes progress row or explicit exclusion. The row must turn the 1,206 unmapped upstream test files from one giant blocker count into deterministic report groups that builders can chase without creating a side backlog. | operator, system | `internal/repoctl/hermes_contract_inventory_test.go:TestWriteHermesContractInventoryWritesJSONAndMarkdown; webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.md` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 2 / 2.D | Scheduled briefing job emits operator run report | Wire scheduled briefing cron/fleet jobs so every unattended run writes an OperatorRunReport after completion. The slice should cover local delivery, suppressed/no-agent/script-only jobs, provider-backed runs, timeout/error paths, and repeat/terminal completion evidence while preserving existing cron_runs and CRON.md mirror behavior. | operator, system | `internal/cron/operator_briefing_report_test.go::TestScheduledBriefingWritesOperatorRunReport` | Unblocks Morning degraded-status summary over latest run report. |
| 2 / 2.B.12 | Hermes gateway platform strict-fidelity source-pair expansion | Expand strict-fidelity mappings for Hermes gateway platform implementations and TUI gateway bridge files. The pass must classify platform adapters, platform helper modules, API-server platform surface, TUI gateway websocket/render/protocol bridge, and platform docs into completed channel rows, planned adapter rows, or explicit exclusions. | operator, system | `internal/channels strict-fidelity gateway platform mapping fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 2 / 2.F.4 | Gateway delivery evidence in operator run report | Join gateway/cron delivery results into OperatorRunReport artifacts so unattended jobs record target platform, chat/thread identity, delivery path, delivered boolean, fallback path, and stable failure evidence for live adapter, standalone sender, and fallback sink paths. The slice must reuse existing cron delivery planning/outcome evidence and never require a live Telegram/Discord/Slack bot in tests. | operator, system | `internal/cron/operator_delivery_report_test.go::TestOperatorRunReportIncludesDeliveryEvidence` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 4 / 4.I | Hermes agent runtime strict-fidelity source-pair expansion | Expand source-pair and progress mappings for unmapped Hermes `agent` runtime files before treating them as runtime implementation gaps. The pass must classify transports, LSP helpers, context/compression helpers, prompt caching, retry/rate diagnostics, conversation loop helpers, tool dispatch helpers, and safety/redaction helpers into existing Gormes provider/runtime/tool rows or new builder rows. | operator, system | `internal/fidelity agent runtime strict-fidelity mapping fixture` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.I | Hermes plugin catalog strict-fidelity classifier | Classify Hermes first-party plugin families into Gormes plugin/provider/channel/memory rows or explicit exclusions. The classifier must cover model-provider plugin manifests, memory plugins, web/browser/image/video plugins, platform plugins, Google Meet, Teams pipeline, Spotify, and plugin docs so strict fidelity can distinguish runtime gaps from catalog-only compatibility evidence. | operator, system | `internal/plugins hermes plugin catalog strict-fidelity fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.L | Hermes LSP write-time semantic diagnostics | After `write_file` or `patch`, Gormes runs a language-server diagnostic pass equivalent to Hermes' write-time LSP surface, shifts baseline ranges through edits, and returns new semantic errors to the agent without blocking unsupported languages. | - | `internal/tools LSP diagnostic fake-server fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.N | Hermes x_search tool and auth surface | Expose Hermes' first-class `x_search` tool in Gormes with a descriptor, OAuth/API-key auth status, query/result envelope, rate-limit/degraded errors, and registry/toolset visibility without requiring live X credentials in tests. | - | `internal/tools x_search fake transport fixtures` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.N | Morning degraded-status summary over latest run report | Add a read-only operator summary surface that renders the latest OperatorRunReport into text and JSON for morning review. The summary must show whether the unattended run succeeded, degraded, or failed; include job/run identity, delivery status, provider/auth readiness, redacted error details, and a recommended next command; and integrate with existing gormes status-style output without mutating cron or gateway state. | operator, system | `cmd/gormes/status_operator_report_test.go::TestStatusRendersLatestOperatorRunReport` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
