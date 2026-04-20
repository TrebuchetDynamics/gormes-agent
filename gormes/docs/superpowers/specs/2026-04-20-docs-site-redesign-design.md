# Docs Site (docs.gormes.ai) Redesign

**Status:** Approved 2026-04-20
**Owner:** xel
**Implementation skill:** subagent-driven-development

## Goal

Rebuild `docs.gormes.ai` from the current minimal Hugo shell into a coherent two-audience docs site: **regular users** (install, run, configure Gormes) and **collaborators** (understand the architecture, port a subsystem, contribute). Match the `gormes.ai` landing aesthetic so the two sites read as one product. Bulletproof mobile across iPhone SE → iPhone Plus. Promote the Learning Loop (skill extraction) from a Phase 5 sub-phase to a dedicated Phase 6 throughout the site and ARCH_PLAN.

## Approach

Keep the Hugo static-site generator, keep the existing deploy workflow (`.github/workflows/deploy-gormes-docs.yml`). Replace the empty custom layouts with a hand-built, landing-aligned theme. Split the 68 KB `gormes/docs/ARCH_PLAN.md` into digestible per-phase pages under `content/building-gormes/architecture_plan/`. Keep the 20 inherited upstream-Hermes guides in a clearly labelled "Upstream Hermes · Reference" section with a disclaimer banner — don't delete valuable context, but don't misrepresent what ships in Gormes.

## Information Architecture

Single unified left sidebar with three colored section headers (no audience-split top nav, no lane-forcing):

### USING GORMES (user-facing, green header)

Content lives under `content/using-gormes/`:

- `_index.md` — section landing, 1-paragraph overview
- `quickstart.md` — 60-second path: `curl | sh` + `gormes doctor --offline` + `gormes`
- `install.md` — full install matrix (Linux, macOS, WSL2, Termux), `go install` path, systemd service hints
- `tui-mode.md` — how the TUI works, keybindings, render mailbox behaviour
- `telegram-adapter.md` — `gormes telegram` subcommand, token setup, edit coalescing
- `configuration.md` — config file format, env vars, where state lives on disk
- `wire-doctor.md` — what `gormes doctor` validates, how to read its output
- `faq.md` — common operator questions (offline mode, memory location, log files)

### BUILDING GORMES (contributor-facing, cyan header)

Content lives under `content/building-gormes/`:

- `_index.md` — "Gormes in one sentence: a production runtime for self-improving agents. Four core systems: Learning Loop + Memory + Tool Execution + Gateway." Opens with the user's strategic framing verbatim.
- `core-systems/_index.md` — overview of the 4 systems
- `core-systems/learning-loop.md` — Phase 6 system; complexity detection → skill extraction → storage → improvement; points at `architecture_plan/phase-6-learning-loop.md`
- `core-systems/memory.md` — SQLite + FTS5 + ontological graph + USER.md mirror; points at `architecture_plan/phase-3-memory.md`
- `core-systems/tool-execution.md` — typed Go interfaces, in-process registry, streamed tool_calls; points at `architecture_plan/phase-2-gateway.md`
- `core-systems/gateway.md` — multi-interface (TUI, Telegram, future Discord/Slack); points at `architecture_plan/phase-2-gateway.md`
- `what-hermes-gets-wrong.md` — Python stack, execution chaos, conceptual subagents, startup cost. Framing from user's 2026-04-20 strategic note.
- `architecture_plan/` — see §"ARCH_PLAN Migration" below
- `porting-a-subsystem.md` — contributor path: pick from §7 Upstream Subsystem Inventory, write spec + plan, open PR
- `testing.md` — Go test suite, Playwright smoke suite, Hugo build test rig

### UPSTREAM HERMES · REFERENCE (amber header)

Content lives under `content/upstream-hermes/`:

- `_index.md` — disclaimer banner: "These guides document the Python upstream `hermes-agent`. Gormes is porting them gradually — see §5 Final Purge in the roadmap. What you read here may or may not be implemented in Gormes today."
- **Migration rule**: every file currently under `content/guides/`, `content/developer-guide/`, `content/integrations/`, `content/reference/`, `content/user-guide/`, and `content/getting-started/` re-homes verbatim under `content/upstream-hermes/` preserving its directory structure. No per-file triage in this spec — it's mechanical. Any file that maps 1:1 to Gormes today can graduate into `using-gormes/` in a later, smaller spec after this one lands.

### Top nav

Thin and unified across every page:

- `Gormes` brand mark (links to gormes.ai landing)
- Mini search input (Pagefind-backed)
- `GitHub` link (external, opens new tab)
- `Back to Site →` link to gormes.ai

## Visual System

Matches `gormes.ai` landing:

- **Fonts**: Fraunces (display), JetBrains Mono (technical), DM Sans (body) — same Google Fonts import as the landing
- **Accent**: `#e8c547` amber; same `--accent-ink` (`#1a1300`) on filled surfaces
- **Bed**: `#0a0d11`; surface `#121720`; border `#1e232e`; text `#ebe9e2`
- **Status tones**: same four-tone system from the landing roadmap (shipped/progress/planned/later) for phase pages + roadmap index
- **Section-header colors**: green (`--status-shipped-fg`) for USING, cyan (`--status-progress-fg`) for BUILDING, amber (`--status-next-fg`) for UPSTREAM
- **Grain overlay**: same inline-SVG noise at `opacity: 0.05, mix-blend-mode: overlay`
- **Section kicker style**: `§ USING · TUI mode` mono uppercase letter-spaced, matching the landing's `§ 01 · …` treatment

### Docs ergonomics beyond landing

- **Right-side TOC** on article pages: auto-generated from `<h2>` / `<h3>` anchors; sticky scroll; highlight active section via IntersectionObserver (inline JS, ~30 lines, scoped)
- **Breadcrumbs** above the page title: `USING GORMES / Install` format, mono, muted
- **Inline copy buttons** on every code block: reuse the landing's `gormesCopy(btn)` helper verbatim; Hugo render hook adds `<button class="copy-btn">` into every `<pre>`
- **Search**: Pagefind (static, zero-backend). Build step runs `pagefind` after `hugo --minify` to index `public/`; search input uses Pagefind's ~15 KB JS UI
- **Active nav highlighting**: current page's sidebar entry carries `aria-current="page"` and a left amber bar

## Mobile Behavior (bulletproof)

Three breakpoints:

- **≥1024px** — 3-column: sidebar (240px) · content · TOC (180px)
- **640–1023px** — 2-column: sidebar (240px) · content · TOC collapses into `<details>` above the page title
- **<640px** — single-column: top-only nav with hamburger, sidebar becomes a slide-in drawer (CSS `:has()` + checkbox trick, no framework), TOC becomes `<details>` at the top of each page

Invariants enforced by Playwright:

- No horizontal scroll at any of 320/360/390/430/768/1024 px viewports
- Sidebar drawer opens/closes via hamburger button; overlay dismisses on click-outside
- All code blocks fit: `overflow-x: auto` inside a `min-width: 0` parent
- Long ARCH_PLAN phase-page bodies wrap cleanly: `overflow-wrap: anywhere` on text containers
- Every clickable element has a tappable ≥28×28 px bounding box (matches the landing's enforced floor)
- `prefers-reduced-motion: reduce` disables all transitions

## ARCH_PLAN Migration

Current: `gormes/docs/ARCH_PLAN.md` (68 KB, at docs root, unrendered by Hugo).

Target: split across `content/building-gormes/architecture_plan/`:

```
architecture_plan/
  _index.md                       # Roadmap overview (§0 Thesis + §4 Milestone Status table)
  phase-1-dashboard.md            # §4 Phase 1 detail + user's "complete but evolving" framing
  phase-2-gateway.md              # §4 Phase 2 + Phase 2 Ledger (2.A/B.1/C shipped; 2.B.2+/D/E/F planned)
  phase-3-memory.md               # §4 Phase 3 + sub-status (3.A/B/C shipped, 3.D planned, 3.D.5 shipped)
  phase-4-brain-transplant.md     # §4 Phase 4 + Phase 4 Sub-phase Outline (4.A–4.H); 4.A–4.D ship hermes-off
  phase-5-final-purge.md          # §4 Phase 5 + Phase 5 Sub-phase Outline (5.A–5.P)
  phase-6-learning-loop.md        # NEW — promoted from 5.F; user's "SOUL" framing; skill extraction algorithm placeholder
  subsystem-inventory.md          # §7 Upstream Subsystem Inventory (all tables: platforms, operational, memory, brain, tools, CLI, out-of-scope, cadence)
  mirror-strategy.md              # §8 Mirror Strategy (auditability roadmap; what Hermes actually has vs Gormes; remaining mirror candidates; principles)
  technology-radar.md             # §9 Technology Radar (vector libs, SQLite drivers, upstream tracking)
  boundaries.md                   # §5 Project Boundaries (no Python edits, bridge not destination)
  why-go.md                       # §2 Why Go + §3 Hybrid Manifesto
```

`gormes/docs/ARCH_PLAN.md` at the docs root is replaced with a **one-line stub** pointing at the canonical split:

```markdown
# ARCH_PLAN moved

Canonical source: `content/building-gormes/architecture_plan/`.
Published at: https://docs.gormes.ai/building-gormes/architecture_plan/
```

This keeps existing git searches and external links working. The `docs_test.go` and `landing_page_docs_test.go` tests are updated to read the split files.

## Phase 6 — Learning Loop (Promotion)

**Why**: User's strategic framing elevates skill extraction from plumbing (5.F) to "THE SOUL" — the compounding-intelligence differentiator Gormes ships that Hermes doesn't articulate cleanly. It deserves equal ledger weight with Phase 3 Memory.

**ARCH_PLAN changes**:

- §4 Milestone Status table grows to 6 phases: add `Phase 6 — Learning Loop (The Soul) · ⏳ planned · Native Go skill extraction: complexity detection, pattern distillation, storage, feedback-driven improvement`
- New `### Phase 6 Sub-phase Outline` section (like Phase 4/5):
  - 6.A — Complexity detector (when is a task "worth" learning from?)
  - 6.B — Skill extractor (LLM-assisted pattern distillation from conversation + actions)
  - 6.C — Skill storage format (Markdown + metadata, portable, human-editable)
  - 6.D — Skill retrieval / matching (hybrid lexical + semantic)
  - 6.E — Feedback loop (did the skill help? adjust weight)
  - 6.F — Skill surface in TUI / Telegram (skill list, manual edit, disable)
- §7 Subsystem Inventory adds a "Learning Loop (Phase 6 — new)" row group listing the components above
- §8 Mirror Strategy adds: "Skills mirror to `~/.hermes/memory/SKILLS.md` alongside USER.md"

**Landing page changes**:

- `RoadmapPhases` grows from 5 to 6 entries
- Phase 6 card: status `PLANNED · 0/6`, title `Phase 6 — Learning Loop`, subtitle `Compounding intelligence. The feature Hermes doesn't have.`, six pending rows

## Testing

### Go tests (new)

- `gormes/docs/docs_test.go` — already exists; extend to:
  - Assert every split ARCH_PLAN page exists at expected path
  - Assert the front-matter on each (title, weight)
  - Assert no page references the old `ARCH_PLAN.md` filename (except the stub)

- `gormes/docs/build_test.go` (new) — shells out to `hugo --minify` into a temp dir, asserts:
  - Every `_index.md` in content/ produces a `public/*/index.html`
  - No broken internal links (use `hugo check links` or a small Go walker)
  - Build completes in under 5 seconds (guard against theme bloat)

### Playwright (new)

- `gormes/docs/www-tests/` (new directory mirroring `www.gormes.ai/tests/`)
- `home.spec.mjs` — docs landing renders, sidebar has 3 sections, search input present, theme matches (`--bg-0` present, Fraunces loaded)
- `mobile.spec.mjs` — parametrized over 320/360/390/430/768/1024 px:
  - No horizontal overflow
  - Hamburger opens drawer at <640px; drawer dismisses on backdrop click
  - TOC is a `<details>` at <1024px, right-side panel at ≥1024px
  - Every code block has a tappable copy button ≥28×28 px
- `toc.spec.mjs` — IntersectionObserver highlights the correct TOC entry on scroll
- `search.spec.mjs` — typing in the search input shows Pagefind results (integration test with prebuilt index)

### Hugo tests

- Existing `landing_page_docs_test.go` — keep, update references if paths changed
- New `docs_test.go` assertions for the split plan pages

## Deployment

No workflow changes. Existing `.github/workflows/deploy-gormes-docs.yml`:

- Triggers on push to main touching `gormes/docs/**`
- Installs Hugo 0.140.0 extended (version pin unchanged)
- Builds: `hugo --minify` from `gormes/docs/`
- New step (add to existing workflow): `npx pagefind --source public` to generate search index into `public/pagefind/`
- Deploys `gormes/docs/public/` to Cloudflare Pages `gormes-docs` project
- Attaches `docs.gormes.ai` custom domain (idempotent)

## Out of Scope

- Localization — English only. Hugo's i18n system stays dormant.
- Versioned docs — no `/v0.1/` URL prefix. Docs are always-current for trunk.
- Dark/light toggle — dark only, matches landing.
- Comments, feedback widgets, analytics pixels — the docs ship no client-side tracking.
- Editing the inherited `content/guides/*` text — files move but copy stays verbatim; editing them would be a separate spec.
- Phase 6 implementation itself — this spec only promotes Phase 6 in documentation and ledger. Actual skill-extraction code is a separate spec (future work).
- Redesigning the landing page — out of scope; only cosmetic additions to the roadmap section to add Phase 6 as a 6th card.

## Acceptance Criteria

On the live `docs.gormes.ai` after merge:

1. `https://docs.gormes.ai/` loads, matches `gormes.ai` aesthetically (Fraunces headline, amber accent, dark bed, grain overlay).
2. Sidebar shows three sections with colored headers: `USING GORMES` (green), `BUILDING GORMES` (cyan), `UPSTREAM HERMES · REFERENCE` (amber).
3. `https://docs.gormes.ai/using-gormes/quickstart/` renders with breadcrumbs, right-side TOC, copyable code blocks.
4. `https://docs.gormes.ai/building-gormes/architecture_plan/` renders the roadmap overview with all 6 phases listed.
5. `https://docs.gormes.ai/building-gormes/architecture_plan/phase-6-learning-loop/` exists and renders the new Phase 6 content.
6. `https://docs.gormes.ai/upstream-hermes/` renders with the disclaimer banner at the top.
7. Search input at top returns Pagefind results when typing (e.g. "Phase 3" returns the memory phase page).
8. On a 320×568 viewport: no horizontal overflow, hamburger opens a drawer sidebar, drawer closes on backdrop click, every page is readable with wrapping text.
9. On a 1024×768 viewport: 3-column layout renders (sidebar + content + TOC).
10. Go tests pass: `hugo --minify` builds without broken links, every split ARCH_PLAN page present.
11. Playwright tests pass across 6 mobile viewports.
12. Landing page (`gormes.ai`) shows 6 phase cards including the new Phase 6 — Learning Loop.
13. `gormes/docs/ARCH_PLAN.md` is reduced to the one-line stub.
14. ARCH_PLAN content is preserved and split across `architecture_plan/*.md` — `git log -S` can find any original phrase.

## Implementation Note

This spec is sized for subagent-driven execution. Each major bullet (content migration, layout templates, mobile CSS, ARCH_PLAN split, Phase 6 promotion, tests) decomposes into ~5 isolated subtasks an implementer subagent can handle end-to-end with spec review + code review between each. Expected task count in the writing-plans output: 8–10 tasks.
