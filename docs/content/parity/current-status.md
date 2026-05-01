---
title: "Current Status"
description: "How to read status claims in Gormes docs."
weight: 10
---

# Current Status

Use status labels precisely:

| Label | Meaning |
|---|---|
| `runtime-ready` | Verified in the current Gormes runtime. |
| `fixture-backed` | Tested with local fixtures or fake clients. |
| `row-backed` | Tracked in progress/planning, but not promised as live behavior. |
| `planned` | Accepted direction without a shipped implementation. |
| `unverified` | Needs source or runtime evidence. |

This distinction matters because a progress row can be complete while the user-visible feature still needs wiring, live validation, or UX parity work.
