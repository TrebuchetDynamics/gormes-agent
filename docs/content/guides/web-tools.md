---
title: "Web Tools"
description: "Understand web search, extraction, backend routing, and cost boundaries."
weight: 40
---

# Web Tools

Gormes exposes web tools through the tool registry, including search and extraction surfaces. Backend availability depends on configured credentials and local fallbacks.

Operator rules:

- Use free fallback search only when no API-backed search provider is configured.
- Use API-backed search/extract providers when credentials exist and cost is acceptable.
- Use browser/CDP paths when a dynamic site or blocked scraper needs an actual browser.
- Treat "opened page" without extracted useful content as a weak result that needs better routing.

Docs should show the active backend, what each backend can do, whether it costs money, and what fallback happens when extraction fails.
