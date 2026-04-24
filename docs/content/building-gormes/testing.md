---
title: "Testing"
weight: 50
---

# Testing

Three layers.

## Go tests

`go test ./...` from the repository root. Covers kernel, memory, tools, telegram adapter, session resume. Integration tests are tag-gated:

```bash
go test -tags=live ./...         # requires local Ollama
```

## Landing + docs smoke (Playwright)

`npm run test:e2e` from `www.gormes.ai/` and `docs/www-tests/`. Parametrized over mobile viewports (320 / 360 / 390 / 430 / 768 / 1024 px). Asserts:

- No horizontal overflow
- Copy buttons tappable (≥28×28 px bounding box)
- Section copy matches the locked strings in `content.go`
- Drawer opens/closes on mobile (docs only)

## Hugo build

`go test ./docs/... -run TestHugoBuild`. Shells out to `hugo --minify`, verifies every `_index.md` produces a rendered page, checks for broken internal links.

## Discipline

Every PR must keep all three layers green. The `deploy-gormes-landing.yml` and `deploy-gormes-docs.yml` workflows run the Go + Playwright subsets on every push to `main`.
