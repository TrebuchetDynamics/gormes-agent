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
| 8 / 8.A | TD social presence connected to blog feed | Operator-selected social platform/account and non-repo credentials, Public social test-post URL evidence | TD blog (8.A row 1) is live and emitting a stable feed., Operator has chosen a social platform, created the account, and provided the minimum non-secret account URL/handle for docs., Automation can be developed against a local feed fixture or dry-run mode before any social credentials are configured. | - |
| 8 / 8.C | Engineering writeup #1: autonomous Hermes-porting loop | One week of loop cost telemetry data collected from the 8.F cost fields, Operator publication review/date/platform decision | TD blog (8.A row 1) is live., Loop $/iteration cost telemetry (8.F) has at least one week of data., Operator has decided the publication date and platform (HN/Lobsters/Reddit)., `Engineering writeup #1 local publication review packet` is complete and gives Juan a source-backed review checklist before public posting., `Engineering writeup #1 cost telemetry evidence packet` is complete and captures current unknown-cost evidence without fabricating numbers; final public cost claims still require one week of measured cost telemetry., `OpenCode part-cost telemetry adapter for builder loop` is complete and proves current local 30-day OpenCode spend extraction without satisfying the one-week 7-day window. | Engineering writeup #2: validation-gated agentic engineering, Engineering writeup #3: Gormes vs Hermes-Python benchmarks, HN launch post for Gormes 1.0 |
| 8 / 8.E | Agentic-porting-kit public repo scaffold | GitHub create-or-push access to TrebuchetDynamics/agentic-porting-kit, Operator confirmation of the public repo name before first push | Agentic-porting-kit local public layout assembly gate is complete and green, or the external repo builder intentionally replaces it with equivalent pre-push validation., Agentic-porting-kit local README and LICENSE fixtures are complete and green, or the external repo builder intentionally replaces them with validated equivalent files., Agentic-porting-kit local porting skill skeletons are complete and green, or the external repo builder intentionally replaces them with a validated equivalent., Agentic-porting-kit local standalone fixture is complete and green, or the external repo builder intentionally recreates an equivalent example in the public repo., Agentic-porting-kit extraction spec is complete., GitHub authentication can create or push to TrebuchetDynamics/agentic-porting-kit, or the operator has created the empty repo., The public repo name is confirmed as agentic-porting-kit or an equivalent name before the first push. | - |
<!-- PROGRESS:END -->
