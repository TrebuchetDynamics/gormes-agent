---
title: "Testing"
description: "Focused checks for docs and runtime-adjacent changes."
weight: 20
---

# Testing

Docs-focused checks:

```bash
go test ./webpages/docs -count=1
go run ./cmd/progress validate
git diff --check
```

When changing rendered docs behavior, also run the Playwright docs tests:

```bash
cd webpages/docs/www-tests
npm run test:e2e
```

Runtime features need package-specific Go tests and, when relevant, a progress row with source-backed evidence.
