---
name: gormes-landing-web
description: Improve the www.gormes.ai landing page content, UI, visual hierarchy, install flow, and public trust signals from current Gormes evidence. Use when the user asks to audit, redesign, polish, rewrite, or periodically improve the Gormes landing page or homepage UI.
---

# Gormes Landing Web

## Repository Branch Rule

For Gormes work, stay on the existing `development` branch. Do not create or
use feature branches, short-lived branches, or git worktrees. If the checkout
is not on `development`, stop before editing and switch safely or report the
blocker.

## Mission

Keep `www.gormes.ai` clear, accurate, and usable for strangers. This skill is
for landing-page content and UI passes, not runtime feature implementation or
roadmap invention.

## Source Files
- `www.gormes.ai/internal/site/content.go` - landing copy, CTAs, links, proof.
- `www.gormes.ai/internal/site/templates/` - HTML structure.
- `www.gormes.ai/internal/site/static/` - CSS and static assets.
- `www.gormes.ai/tests/home.spec.mjs` - Playwright homepage expectations.
- `www.gormes.ai/internal/site/data/progress.json` - embedded roadmap mirror.
- `benchmarks.json` and `docs/content/building-gormes/architecture_plan/progress.json` - canonical proof inputs.

## Workflow

### 1. Bound The Pass
State whether the pass is about:
- hero and first viewport;
- install and model-backed quick start;
- "what works today" and current limits;
- proof/trust signals;
- visual hierarchy, spacing, responsive layout, or accessibility;
- stale Hermes/Goncho/Honcho wording;
- progress, benchmark, or release status.

If the change would require runtime behavior that is not shipped, route through
`gormes-skill-manager` instead of promising it on the landing page.

### 2. Gather Evidence
Start with:
```sh
git status --short
sed -n '1,220p' www.gormes.ai/README.md
sed -n '1,260p' www.gormes.ai/internal/site/content.go
go run ./cmd/progress validate
rg -n "Hermes|backend|install|doctor|offline|oneshot|Goncho|No Python|progress|benchmark|production" README.md docs www.gormes.ai
```
Use current code, tests, generated progress, and benchmarks as the source of
truth. Do not invent performance, release, or production-readiness claims.

### 3. Apply The Landing Lens
For broad audits, rewrites, or UI polish, read
`references/landing-quality-rubric.md`. Prioritize a first-time visitor who
needs to understand what Gormes is, why it matters, what works today, how to try
it, and why the claims are trustworthy.

### 4. Edit In Scope
Prefer direct edits to the existing Go-rendered site:
- copy and link data in `content.go`;
- page structure in `templates/`;
- presentation in `static/site.css`;
- Playwright assertions in `tests/home.spec.mjs` when user-facing behavior changes.

Keep the page fast, static-exportable, and readable without client-side
JavaScript except the existing bounded install-copy behavior.

### 5. Validate
Run focused checks:
```sh
(cd www.gormes.ai && go test ./... -count=1)
(cd www.gormes.ai && npm run test:e2e)
go run ./cmd/progress validate
```
If progress or benchmark data changed, also run the repo tool that refreshes the
embedded data or rebuilds the site. For UI layout work, inspect desktop and
mobile Playwright output or screenshots before reporting done.

## Final Report
Report:
1. Landing surfaces checked
2. Evidence used
3. Content/UI changes made
4. Claims kept conservative
5. Validation run
6. Remaining landing-page gaps
