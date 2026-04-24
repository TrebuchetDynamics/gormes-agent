# docs.gormes.ai — UI Polish & Mobile Fix

**Status:** Draft 2026-04-20
**Owner:** xel
**Implementation skill:** writing-plans → subagent-driven-development
**Scope note:** This is a narrow UI polish on the existing Hugo shell. The broader IA redesign is already tracked in `2026-04-20-docs-site-redesign-design.md`; this spec is purely about the visual, hierarchy, and mobile-UX execution layer on top of whatever IA that work lands.

## Problem

`docs.gormes.ai` shares a palette with the landing (`gormes.ai`) but doesn't deliver on it. Three specific pain points:

1. **Visual quality** — home cards are three flat rectangles with 3 lines each; sidebar is a wall of 12 px mono text; breadcrumbs and kickers are shouty uppercase-tracked; the `☰` emoji hamburger looks cheap.
2. **Information hierarchy** — all three sidebar groups expand at once so Upstream Hermes (7 pages) dominates; no current-section affordance beyond link-level highlight; TOC at 200 px in mono competes with prose for attention.
3. **Mobile** — drawer has no close button inside, doesn't auto-close on link-tap (pure-CSS only), TOC is always open and bulky, `docs-title` clamp still lands at ~28 px on 360 px phones, copy-button overlap pushes code off-screen.

## Direction

Polish the current brand DNA — don't rebrand, don't reset to a generic dev-docs template. The landing's "operator's manual meets editorial quarterly" character is the right target; the docs just need to execute on it.

Ruled out during brainstorm: pushing harder into editorial (too risky for a docs site), resetting to Stripe/Linear minimalism (throws away brand cohesion), light mode, framework switch.

## Scope

### In

**Structural / behavior**

- Collapsible sidebar groups (current auto-open, others collapsed), state persisted per-group in `localStorage`.
- Mobile drawer auto-closes on link tap; drawer has a header row with "Navigation" label + `✕` close button.
- TOC scrollspy: active heading gets amber left-bar as user scrolls (IntersectionObserver).
- Prev/Next article links at the bottom of single pages, computed from `.Pages.ByWeight`.

**Visual**

- Redesigned home: hero title with amber accent, a "Quickstart in 60 seconds" strip with install command, three richer section cards showing an italic Fraunces ordinal + mini-TOC of top 3 pages + "Explore →" CTA.
- Refined type scale: body 15→16 px, lede 16→17 px, h1 clamp(30, 4.5vw, 44), h2 24→22 px, h3 18→17 px, sidebar 12→13 px, TOC switches from mono to sans 12 px.
- Kickers and breadcrumbs de-shouted (mixed case, tracking 0.16em).
- Hamburger: `☰` emoji → inline SVG inside a 30×30 bordered button.
- Code blocks gain a language badge top-left; copy button polished.
- Sidebar density pass: colored dot + label (not solid left-bar per group), page count per group, active item uses new `--bg-elev` = `#161c26`.
- Rhythm: h2 top margin 44→56 px, paragraph spacing 14→16 px, prose max-width 62 ch.

**Tooling**

- Stay on Hugo. Keep Pagefind. No new dependencies.
- One new CSS token (`--bg-elev`). Palette otherwise unchanged.
- One new JS file, ~120 lines, no deps (drawer + collapsibles + scrollspy).
- Playwright + Go build tests stay green.

### Out

- Light mode / theme toggle.
- "Edit this page on GitHub" link.
- Footer redesign beyond copy alignment.
- Any new icon library or CSS framework.
- Content / IA changes — that's the other spec.

## Approach: two phases

**Phase 1 — structure & behavior** (one PR)

All template + JS changes. No visual design beyond what the new DOM needs. Playwright suite updated and green. Reviewable in ~10 min.

**Phase 2 — visual polish** (separate PR, mostly CSS)

Type scale, home redesign, sidebar density, code block polish, kicker/breadcrumb de-noise, hamburger SVG styles. CSS grows from 496 → ~750 lines. No behavior changes.

Rationale: structural changes risk regressions (mobile drawer, test churn); polishing on top of stable behavior is low-risk iteration. Two small reviewable diffs beat one big one.

## Visual system (Phase 2 targets)

| Role | Now | New |
|---|---|---|
| Body | 15 px DM Sans | **16 px** |
| Lede | 16 px | **17 px** |
| h1 | `clamp(28, 4.5vw, 42)` | **`clamp(30, 4.5vw, 44)`** |
| h2 | 24 px Fraunces | **22 px**, border-bottom `--border`, top margin 56 px |
| h3 | 18 px Fraunces | **17 px**, tighter leading |
| Sidebar item | 12 px JetBrains Mono | **13 px** |
| TOC item | 11 px mono | **12 px DM Sans** |
| Kicker | 10–11 px mono, `letter-spacing: 0.22em` | **same size, 0.16em, mixed case** |
| Breadcrumb | uppercase mono tracked | **mono mixed case, 0.04em tracking** |
| Paragraph spacing | 14 px | **16 px** |
| Prose max-width | implicit | **62 ch** |

Palette: unchanged, add one token:

```css
--bg-elev: #161c26; /* used for hovered/active cards + sidebar current link */
```

## Home page

- Hero: `kicker "Gormes · Documentation"` + `h1 "The Go operator shell for <em>Hermes.</em>"` (amber `<em>`) + lede.
- Quickstart strip: amber left-border card with "New here? Start in 60 seconds" kicker, h2 "Install and run", short body, and a mono `brew install trebuchet/gormes` chip on the right.
- Three section cards:
  - Big italic Fraunces `01 / 02 / 03` watermark in the top-right at 10% opacity.
  - Colored `fc-label` (shipped/progress/next status dot colors retained).
  - h3 section title in Fraunces.
  - 2-line description.
  - Mini-TOC: top 3 pages by weight, each prefixed with `→ `, JetBrains Mono.
  - "Explore →" CTA label at card bottom.
- Hover: border → amber, `translateY(-2px)`.

Template: replace `docs/layouts/index.html` with new hero + quickstart + card markup; top-3-pages-per-section resolved via `site.GetPage "/using-gormes"` etc. and `.Pages.ByWeight | first 3`.

## Sidebar

DOM per group (collapsible via `<details>` + JS fallback):

```html
<details class="docs-nav-group" data-section="using-gormes" open>
  <summary class="docs-nav-group-header">
    <span class="nav-chev" aria-hidden="true">▶</span>
    <span class="nav-dot nav-dot--shipped"></span>
    <span class="nav-group-label nav-group-label--shipped">Using Gormes</span>
    <span class="nav-group-count">8</span>
  </summary>
  <ul class="docs-nav-list">…</ul>
</details>
```

Behavior:
- Current group (the one containing the current page) rendered with `open` and `data-current`; its summary gets the amber left-bar (`::before`).
- Other groups default closed; `localStorage` persists user toggles keyed by `data-section`. On load, JS replays the stored open/closed state but always forces the current group open.
- `nav-dot` = 6 px filled circle in the section status color (green/cyan/amber) — replaces the full 2 px left-border on the label, much quieter.
- Active link: `aria-current="page"` → amber text + `--bg-elev` background + amber 2 px left-border (keeps existing affordance).

## Article page

- Breadcrumbs: mixed-case, mono, tracking 0.04em. Drop the `| upper` template filter.
- h1 sized per new clamp; lede capped at 60 ch.
- Content column max-width 62 ch, h2 with softer border-bottom and 56 px top margin, h3 bold 17 px.
- Code blocks:
  - `.cmd-wrap[data-lang]` → CSS pseudo-element renders the language label top-left (`Shell`, `Go`, `TOML`…).
  - Copy button: smaller (28 px instead of 32 px), positioned with 8 px inset, retains the "Copied" green confirmation.
  - Mobile (<480 px): right-padding 80→70 px; copy button stays 28 px.
- Prev/Next: new partial `layouts/partials/prevnext.html` emits two `<a>` cards side-by-side (or stacked <480 px) with `← Previous / Next →` labels + page title.

## TOC (right rail)

- Switch from mono to DM Sans 12 px.
- Scrollspy: `IntersectionObserver` watching all `h2, h3` in `.docs-content`; the last intersecting heading toggles `.active` on its TOC `<a>`.
- Mobile: collapsed-by-default `<details>` with summary "On this page (N)" — not always-open.

## Mobile

- Topbar:
  - Hamburger: 30×30 `<button>` with inline SVG (3 horizontal strokes), border `--border`, radius 3 px. Replaces `<label>☰</label>`.
  - Drops the "GORMES.AI" top-nav link <767 px (already done). Keeps GitHub.
- Drawer (open state):
  - 280 px wide, left-aligned, backdrop = 55% black.
  - Header row inside drawer: "NAVIGATION" mono label + `✕` close button (both focusable).
  - Link tap → JS dispatches "close drawer" (unchecks `#drawer-toggle`). Backdrop tap and `✕` tap do the same. Esc key closes.
- Article on mobile:
  - `docs-title` clamp lands at ~26 px on 360 px screens (instead of 28).
  - TOC `<details>` closed by default.
  - `.cmd-wrap` gets horizontal-scroll overflow intact; copy button repositioned so code never hides under it.

## Files changed

### Phase 1 — structure & behavior

| File | Change |
|---|---|
| `docs/layouts/_default/baseof.html` | Replace `☰` label with SVG `<button>` (wired to drawer checkbox via JS). Include `site.js`. |
| `docs/layouts/partials/topbar.html` | Same hamburger swap. |
| `docs/layouts/partials/sidebar.html` | Wrap each group in `<details class="docs-nav-group" data-section>`. Add `nav-dot`, `nav-group-count`, `data-current` on containing-page's group. |
| `docs/layouts/partials/toc.html` | Emit plain `<nav class="docs-toc-body"><ul>…</ul></nav>` from `.TableOfContents` (re-render if needed so scrollspy can target `<a>` with clean hrefs). |
| `docs/layouts/partials/prevnext.html` | **New.** Use `.Parent.Pages.ByWeight` to find current page, output prev/next cards. |
| `docs/layouts/_default/single.html` | Add `{{ partial "prevnext.html" . }}` before TOC aside close. |
| `docs/layouts/_default/list.html` | Only render the auto "In this section" child list when `.Content` is empty; otherwise skip. |
| `docs/static/site.js` | **New.** ~120 lines vanilla JS: drawer open/close (button + esc + link-tap auto-close), collapsible sidebar groups with `localStorage` persistence, TOC scrollspy with IntersectionObserver. |
| `docs/www-tests/tests/drawer.spec.mjs` | Update: new button selector; add test for auto-close on link tap + Esc key. |
| `docs/www-tests/tests/mobile.spec.mjs` | Update: hamburger is a `<button>`; add collapsible-group toggle test. |
| `docs/www-tests/tests/home.spec.mjs` | If it references the topbar hamburger selector, update to the new `<button>`. Home template itself stays unchanged in Phase 1, so any assertions on the current three flat cards remain valid. New hero/quickstart/cards-with-mini-TOC assertions land in Phase 2. |
| `docs/docs_test.go`, `docs/build_test.go`, `docs/landing_page_docs_test.go` | Update any assertions tied to old DOM classes or structure. |

### Phase 2 — visual polish

| File | Change |
|---|---|
| `docs/static/site.css` | Apply type scale, new `--bg-elev` token, kicker/breadcrumb de-shout, h2/h3 rhythm, `.cmd-wrap[data-lang]` language badge, copy button refinement, hamburger button styles, `.docs-prevnext` styles, new home styles (`.docs-home-hero`, `.docs-home-quickstart`, `.docs-home-card` with ordinal + mini-list), sidebar dot/count/chevron styles. Expect growth 496 → ~750 lines. |
| `docs/layouts/index.html` | Replace with hero + quickstart + three enhanced cards. Pull top-3-pages-per-section via `site.GetPage` + `.Pages.ByWeight \| first 3`. |
| `docs/layouts/_default/_markup/render-codeblock.html` | Confirm it emits `data-lang="{{ .Type }}"` on the wrapper; add if missing. |
| `docs/layouts/partials/breadcrumbs.html` | Drop `\| upper` on titles; let CSS own the casing. |
| `docs/www-tests/tests/home.spec.mjs` | Add assertions for hero title, quickstart strip (amber left-border card + install command), and three cards each containing an ordinal + mini-TOC + "Explore →" CTA. |

## Risks & mitigations

1. **Hugo TOC markup.** Hugo's default `.TableOfContents` output may not have the clean `<a>` structure scrollspy needs. Mitigation: if the default breaks, re-emit TOC manually by walking `.Fragments.Headings` (Hugo 0.111+), which we're already on.
2. **Pagefind search after DOM changes.** Pagefind reads `data-pagefind-*` attributes from a crawl of the built site. Changing layouts shouldn't affect it as long as main article content stays inside the same tagged container. Mitigation: run `pagefind --site public` in CI step; if it misses, add `data-pagefind-body` on `.docs-content`.
3. **Collapsible state restoration on navigation.** Per-section, not per-page, keyed by `data-section`. Current group always overrides stored state and opens. localStorage key `docs-nav-groups` = `{sectionKey: "open"|"closed"}`.
4. **Playwright churn.** Selectors may break where tests targeted `.drawer-btn` as a `<label>`. Mitigation: add stable `data-testid` attributes on drawer button, drawer close, group summary, and active sidebar link in Phase 1, so future redesigns don't cascade into test churn.

## Testing expectations

- `make docs-build` passes (Go site-build tests for the Hugo output).
- `make docs-test` passes (Playwright: home + drawer + mobile specs).
- Manual smoke: open the three section landing pages and one leaf page at 1280 / 1024 / 768 / 360 px; verify sidebar state, drawer behavior, TOC scrollspy, prev/next, code copy, search.
- No regression in Pagefind search UI appearance or behavior.

## Non-goals (do not add even if tempting)

- Light mode toggle.
- "Edit on GitHub" links.
- Footer redesign (copy can move around; no new visual elements).
- Any new JS dependency or CSS framework.
- Icon library beyond the single inline SVG for the hamburger.
- Content / IA changes — covered by the separate `docs-site-redesign` spec.
