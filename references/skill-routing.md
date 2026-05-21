# Gormes Skill Routing Reference

Use this table with `docs/development-skills/gormes-skill-manager/SKILL.md` when selecting the smallest repo-local skill chain.

| Intent | Primary skill | Typical follow-up |
|---|---|---|
| Unsure which Gormes workflow applies | `gormes-skill-manager` | One or two targeted skills below |
| Persistent long-running Gormes objective or `/goal` command | `gormes-goal` | Skill selected from active goal |
| Recurring Hermes/Gormes parity sweep or taxonomy refresh | `gormes-hermes-parity` | `gormes-parity-auditor`, `gormes-planner`, `gormes-tdd-slice` |
| Map missing Hermes/Honcho behavior before implementation | `gormes-parity-auditor` | `gormes-planner` |
| Discover OpenClaw-only behavior worth adopting | `gormes-openclaw-parity` | `gormes-planner` |
| Discover reusable Pi harness techniques without treating Pi as a parity contract | `gormes-pi-parity` | `gormes-planner`, `gormes-interface-designer`, or `gormes-tdd-slice` |
| Refine phases, rows, dependencies, or progress docs | `gormes-planner` | none unless implementation starts |
| Slice a broad plan, PRD, parity gap, or review finding into progress rows | `gormes-progress-slicer` | `gormes-planner` to write/update rows |
| Implement one builder-ready row | `gormes-builder` | `gormes-tdd-slice` |
| Red-green-refactor one behavior or failing row | `gormes-tdd-slice` | `gormes-builder` if row metadata changes |
| Provider/auth/model/streaming/rate-limit behavior | `gormes-provider-parity` | `gormes-tdd-slice` |
| Browser Use, CDP, Browserbase, Firecrawl, or `/browser connect` | `gormes-browser-harness` | `gormes-planner` or `gormes-tdd-slice` |
| Local binary, install, gateway process, PATH, or sessions.db locks | `gormes-dev-runtime` | `gormes-install` for installer-specific validation |
| End-to-end installer or setup validation | `gormes-install` | `gormes-dev-runtime`, `gormes-planner` |
| Architecture zoom-out before unfamiliar/cross-package work or architecture improvement | `gormes-architecture-zoomout` | `gormes-interface-designer`, `gormes-service-layer-refactor`, or `gormes-tdd-slice` |
| API/package boundary design | `gormes-interface-designer` | `gormes-builder` |
| Find Go implementation donor shapes | `gormes-references` | `gormes-tdd-slice` |
| Throwaway design/state/UI experiment before production work | `gormes-prototype-spike` | `gormes-tdd-slice` after decision |
| Source-backed external library/framework/upstream context | `gormes-context-sourcing` | parity/planner/builder skill selected from evidence |
| Use a tagged external Go module release in Gormes | `gormes-context-sourcing` | `gormes-tdd-slice` for E2E import/behavior proof |
| Repeated runtime mechanics or service-layer cleanup | `gormes-service-layer-refactor` | `gormes-interface-designer` when boundary is unclear |
| PR readiness audit before review/merge | `gormes-pr-check` | `gormes-review-loop` or `gormes-greptile-loop` |
| Greptile sub-5/5 review loop | `gormes-greptile-loop` | `gormes-tdd-slice`, then `gormes-git` when committing |
| External review finding that must become canonical backlog work | `gormes-review-loop` | `gormes-planner` to refine/add one progress row |
| Local production-readiness score when Greptile is unavailable | `gormes-review-scorecard` | `gormes-tdd-slice` or `gormes-review-loop` |
| PR feedback, CI failures, or bounded review-to-green loops | `gormes-review-loop` | `gormes-tdd-slice`, then `gormes-git` when committing |
| README/public repository messaging | `gormes-readme` | `gormes-git` when committing |
| Landing page content or UI | `gormes-landing-web` | `gormes-git` when committing |
| Dashboard screenshots, hero images, social cards, or image-based dashboard assets | `dashboard-image-design` | `gormes-landing-web` only when the asset ships on www.gormes.ai |
| Commit all dirty development work and push | `gormes-git` | none |
| Release development to main and tag | `gormes-release` | `gormes-git` as subroutine |
| Stress-test a plan with the user | global `grill-me` | skill selected from outcome |
| Flutter Navivox Telegram-like chat/contact UI | `navivox-telegram-ui` | `gormes-tdd-slice` for widget-test implementation |

Rules:

- Keep chains to at most three skills.
- Edit canonical skills under `docs/development-skills/<name>/SKILL.md` only; `.agents/skills`, `.claude/skills`, and `.codex/skills` are symlink loader views.
- Do not create side backlogs; implementation intent belongs in `progress.json` through `internal/progress` or `cmd/progress`.
- Stay on the existing `development` branch for Gormes work.
