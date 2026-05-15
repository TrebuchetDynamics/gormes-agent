---
title: "Blocked Slices"
weight: 40
aliases:
  - /building-gormes/blocked-slices/
---

# Blocked Slices

This page is generated from canonical `progress.json` rows that declare
`blocked_by`.

Use it to avoid assigning work before the dependency chain is ready.

<!-- PROGRESS:START kind=blocked-slices -->
| Phase | Slice | Blocked by | Ready when | Unblocks |
|---|---|---|---|---|
| 5 / 5.Q | Native TUI /model slash command binding over the existing model picker | Kernel in-session model-switch seam for the native TUI | The 'Kernel in-session model-switch seam for the native TUI' prerequisite row is complete (a real apply seam exists)., A model-catalog -> internal/tui data seam exists so the picker can be populated without internal/tui importing the cmd/gormes catalog., The native TUI slash registry consumes recognized commands before kernel submit (5.Q 'Native TUI slash-command dispatch table' complete, satisfied). | Native TUI slash handler-port coverage |
| 8 / 8.A | TD social presence connected to blog feed | TD engineering blog scaffolded and live | TD blog (8.A row 1) is live and emitting a feed., Operator has chosen a social platform and created the account. | - |
| 8 / 8.C | Engineering writeup #1: autonomous Hermes-porting loop | TD engineering blog scaffolded and live, Loop $/iteration cost metric in status file | TD blog (8.A row 1) is live., Loop $/iteration cost telemetry (8.F) has at least one week of data., Operator has decided the publication date and platform (HN/Lobsters/Reddit). | Engineering writeup #2: validation-gated agentic engineering, Engineering writeup #3: Gormes vs Hermes-Python benchmarks, HN launch post for Gormes 1.0 |
| 8 / 8.D | Single-binary cross-platform release pipeline | Sharp v1.0 differentiator decision | Sharp v1.0 differentiator (8.D row 1) is decided., go build ./cmd/gormes succeeds on all seven target GOOS/GOARCH combinations from CI or a Linux runner where cross-compilation is supported. | - |
| 8 / 8.D | Gormes streaming tool-trail status + spinner cadence wiring | Gormes-owned streaming feedback uplift | Gormes-owned streaming feedback uplift (R3) is complete (thinking indicator + collapsible output shipped). | - |
<!-- PROGRESS:END -->
