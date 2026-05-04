---
title: "Tool Execution"
description: "How tools are registered, routed, and rendered across local and channel surfaces."
weight: 30
---

# Tool Execution

Tools are registered in the Go runtime and presented to providers through descriptors. Results must remain useful in both terminal and channel contexts.

Important documentation boundaries:

- Tool availability is not the same as backend availability.
- A registered tool can still return a typed unavailable result when credentials or local dependencies are missing.
- Web and browser tools need capability-specific docs because search, extraction, crawl, and browser interaction have different costs and failure modes.
