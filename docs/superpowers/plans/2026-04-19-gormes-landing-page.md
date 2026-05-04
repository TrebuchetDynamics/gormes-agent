# Gormes.ai Landing Page Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: use
> `gormes-landing-web` for landing-page work in this repository.

**Goal:** Keep the public `www.gormes.ai` landing page truthful, static, and
easy to verify while serving the current Gormes install and progress story.

**Current implementation:** The active site is now an Astro + Tailwind static
site. The former Go-rendered implementation is deprecated and retained under
`www.gormes.ai/legacy/go-renderer/` for reference and rollback only.

## Architecture

- `www.gormes.ai/src/pages/index.astro` owns the route, page structure, head
  metadata, and bounded copy-button script.
- `www.gormes.ai/src/data/landing.js` owns landing copy, CTA/link data,
  benchmark-derived binary size, and progress-derived roadmap projection.
- `www.gormes.ai/src/styles/global.css` imports Tailwind and defines shared
  theme/base states.
- `www.gormes.ai/scripts/sync-assets.mjs` mirrors canonical installers,
  benchmark data, progress data, and static image assets before dev/build.
- `www.gormes.ai/public/static/*` serves favicons, social card, and visual
  assets.
- `www.gormes.ai/tests/home.spec.mjs` provides browser smoke and mobile
  overflow coverage.
- `www.gormes.ai/legacy/go-renderer/` preserves the old Go renderer; do not add
  new landing-page work there unless the old renderer is intentionally
  restored.

## Work Items

- [ ] Keep the landing-page design and implementation docs in `gormes/docs` so
  Goldmark validation covers both.
- [ ] Document the active Astro + Tailwind layout in `www.gormes.ai/README.md`.
- [ ] Keep the installer aliases sourced from canonical root scripts through
  `scripts/sync-assets.mjs`.
- [ ] Keep progress and benchmark mirrors generated from the repo source of
  truth.
- [ ] Keep public copy conservative: no production-readiness, release, or parity
  claims without matching repository evidence.

## Verification

- `go test ./docs`
- `cd www.gormes.ai && npm run build`
- `cd www.gormes.ai && npm run test:e2e`
- `go run ./cmd/progress validate`

## Notes

- This plan supersedes the original Phase 1.5 Go-template implementation plan.
- The old Go paths were moved under `legacy/go-renderer/` during the Astro
  migration; they are not the active landing-page editing surface.
- The page remains static-exportable and readable without JavaScript, except for
  the optional install command copy buttons.
