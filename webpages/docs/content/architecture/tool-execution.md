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

## Goscrapling Extraction Engine

The in-repo `./goscrapling` module is the Go-native web extraction engine used
inside Gormes. It is wired today: `web_extract` fetches public URLs locally
through goscrapling (with an Instant Answer fallback). It is an adapter-backed
tool dependency, not a replacement for the whole tool registry.

Gormes owns:

- tool descriptors and routing;
- approval and network policy;
- result truncation and channel rendering;
- typed unavailable results when browser, network, or credentials are missing.

goscrapling owns:

- HTML parsing and selector behavior;
- static fetcher and response construction;
- browser fetcher contracts after the browser seam is stable;
- future spider, cache, robots, and checkpoint primitives.

The shipped integration is static and hermetic: a Gormes `web_extract` tool
call fetches a URL locally, optionally applies a CSS selector, and returns
structured evidence such as URL, status, selected text, and response metadata.
Browser-backed extraction, crawling, proxy rotation, and stealth behavior
remain behind later documented capability gates.

See `goscrapling/docs/content/building-goscrapling/strategy/portfolio-and-gormes-fit.md`
in this repository for the goscrapling-side boundary.
