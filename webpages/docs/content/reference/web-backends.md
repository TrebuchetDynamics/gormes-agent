---
title: "Web Backends"
description: "Search, extract, crawl, browser, and fallback status."
weight: 50
---

# Web Backends

Web behavior is capability-based:

| Capability | Examples | Notes |
|---|---|---|
| Search | API providers, free fallback search | Search finds candidate URLs or facts. |
| Extract | API extractors, browser/CDP fallback | Extraction reads a URL into content. |
| Crawl | Provider-specific crawl support | Crawl may not exist for every backend. |
| Browser | Local CDP or harness-backed browser tools | Browser tools are slower but handle dynamic pages better. |

Docs should show cost and fallback behavior clearly. Free fallback search is useful, but it is not the same as a full extraction or crawl backend.
