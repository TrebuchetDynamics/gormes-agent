# docs.gormes.ai UI Polish Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Polish the `docs.gormes.ai` Hugo site to land the landing-site brand DNA — fix visual quality, information hierarchy, and mobile UX — without changing the IA.

**Architecture:** Two-phase rollout against the existing Hugo shell. Phase 1 ships structure + behavior (SVG hamburger, drawer auto-close + close button, collapsible sidebar with localStorage, prev/next partial, scrollspy TOC) behind a new `site.js` (~120 lines vanilla, no deps). Phase 2 ships visual polish (type scale, new home with quickstart strip + ordinal cards, sidebar density, code-block language badge, breadcrumb/kicker de-shout). Each phase is a separate PR.

**Tech Stack:** Hugo 0.160.1, Pagefind, vanilla JS, CSS custom properties. No new deps. Playwright + Go build tests gate each phase.

**Reference spec:** `docs/superpowers/specs/2026-04-20-docs-ui-polish-design.md`

**Working directory for all commands:** `gormes/` unless otherwise noted. All paths are relative to `gormes/` unless absolute.

---

## File Structure (scope map)

### Phase 1 — structure & behavior

| File | Responsibility |
|---|---|
| `docs/layouts/_default/baseof.html` (modify) | Swap `<label>☰</label>` for a `<button>` with inline SVG wired to `#drawer-toggle`; include `site.js`. |
| `docs/layouts/partials/topbar.html` (modify) | Same hamburger swap; keep the `.drawer-btn` class for test selector stability. |
| `docs/layouts/partials/sidebar.html` (modify) | Re-emit each group as `<details class="docs-nav-group" data-section open?>` with a colored dot, a page count, and `data-current` on the containing-page's group. |
| `docs/layouts/partials/toc.html` (modify) | Wrap the TOC in a nav with a stable class so scrollspy can target `<a href="#id">` elements inside it. |
| `docs/layouts/partials/prevnext.html` (new) | Render prev/next cards from `.Parent.Pages.ByWeight`. |
| `docs/layouts/_default/single.html` (modify) | Call `prevnext.html` partial at end of article. |
| `docs/layouts/_default/list.html` (modify) | Suppress auto child-list when `.Content` already renders a list. |
| `docs/static/site.js` (new) | Vanilla JS: drawer open/close + link-tap + Esc + ✕; collapsible sidebar groups with `localStorage` persistence; IntersectionObserver-based TOC scrollspy. |
| `docs/www-tests/tests/drawer.spec.mjs` (modify) | New close-button + auto-close + Esc assertions. |
| `docs/www-tests/tests/mobile.spec.mjs` (modify) | Keep overflow/copy-button asserts; hamburger is now a `<button>`. |
| `docs/www-tests/tests/home.spec.mjs` (modify) | Bump script-count tolerance to ≤2 (pagefind + site.js). |

### Phase 2 — visual polish

| File | Responsibility |
|---|---|
| `docs/static/site.css` (modify) | Apply new type scale, `--bg-elev` token, kicker/breadcrumb de-shout, rhythm, sidebar dot/chevron/count styles, hamburger button + drawer header styles, `.cmd-wrap[data-lang]` language badge, `.docs-prevnext` block, new `.docs-home-*` styles for hero + quickstart + ordinal cards. |
| `docs/layouts/index.html` (rewrite) | Hero + quickstart strip + three enhanced cards with ordinal watermark + mini-TOC + "Explore →" CTA. |
| `docs/layouts/_default/_markup/render-codeblock.html` (modify) | Add `data-lang="{{ .Type }}"` to the wrapper so CSS can render the language badge. |
| `docs/layouts/partials/breadcrumbs.html` (modify) | Drop `| upper`; let CSS own casing. |
| `docs/www-tests/tests/home.spec.mjs` (modify) | Assert hero, quickstart strip, ordinal cards with mini-TOC + CTA. |

### Commands (referenced in tasks)

From `gormes/`:

```bash
# Go build + docs suite (includes Hugo build test)
go test ./docs

# Playwright (auto-starts Hugo server on :1313)
cd docs/www-tests && npm ci && npm run test:e2e
```

Playwright auto-starts `hugo server -D --bind 127.0.0.1 --port 1313` (configured in `docs/www-tests/playwright.config.mjs`).

---

## Phase 1 — Structure & Behavior

### Task 1: Baseline — capture current state, create placeholder site.js

**Files:**
- Create: `docs/static/site.js`
- Test: none (scaffolding)

- [ ] **Step 1: Confirm starting baseline passes**

Run from `gormes/`:

```bash
go test ./docs -run TestHugoBuild -count=1
cd docs/www-tests && npm ci && npx playwright install chromium && npm run test:e2e
```

Expected: all green. If anything fails before we touch code, stop and report — don't start editing until baseline is known-good.

- [ ] **Step 2: Create the placeholder site.js**

Create `docs/static/site.js` with this exact content:

```javascript
// docs.gormes.ai interactive behavior. Vanilla, no deps.
// Loaded deferred from baseof.html. Runs on DOMContentLoaded.
(function () {
  'use strict';

  function onReady(fn) {
    if (document.readyState !== 'loading') fn();
    else document.addEventListener('DOMContentLoaded', fn);
  }

  // Populated by later tasks: drawer, collapsibles, scrollspy.
  onReady(function () {
    // populated by later tasks: drawer (Task 2/3), collapsibles (Task 4), scrollspy (Task 7)
  });
})();
```

- [ ] **Step 3: Commit baseline**

```bash
cd gormes
git add docs/static/site.js
git commit -m "docs(ui): scaffold site.js for Phase 1 interactive behavior"
```

---

### Task 2: Swap hamburger emoji for an accessible SVG button

**Files:**
- Modify: `docs/layouts/partials/topbar.html`
- Modify: `docs/layouts/_default/baseof.html`
- Test: `docs/www-tests/tests/drawer.spec.mjs`

- [ ] **Step 1: Update the failing test first**

Edit `docs/www-tests/tests/drawer.spec.mjs`. Replace the entire file with:

```javascript
import { test, expect } from '@playwright/test';

test('mobile hamburger is an accessible button', async ({ page }) => {
  await page.setViewportSize({ width: 360, height: 760 });
  await page.goto('/using-gormes/quickstart/');

  const btn = page.locator('[data-testid="drawer-open"]');
  await expect(btn).toBeVisible();
  // It's a real button, not a label
  const tagName = await btn.evaluate(el => el.tagName);
  expect(tagName).toBe('BUTTON');
  // Accessible name is set
  await expect(btn).toHaveAttribute('aria-label', /nav/i);
});

test('mobile drawer opens via hamburger and closes via backdrop', async ({ page }) => {
  await page.setViewportSize({ width: 360, height: 760 });
  await page.goto('/using-gormes/quickstart/');

  const sidebar = page.locator('.docs-sidebar');
  let leftBefore = await sidebar.evaluate(el => el.getBoundingClientRect().left);
  expect(leftBefore).toBeLessThan(0);

  await page.locator('[data-testid="drawer-open"]').click();
  await page.waitForTimeout(250);
  const leftOpen = await sidebar.evaluate(el => el.getBoundingClientRect().left);
  expect(leftOpen).toBeGreaterThanOrEqual(0);

  await page.locator('.drawer-backdrop').click({ force: true });
  await page.waitForTimeout(250);
  const leftClosed = await sidebar.evaluate(el => el.getBoundingClientRect().left);
  expect(leftClosed).toBeLessThan(0);
});

test('desktop >=768px does not show the hamburger', async ({ page }) => {
  await page.setViewportSize({ width: 1024, height: 768 });
  await page.goto('/');
  const btn = page.locator('[data-testid="drawer-open"]');
  const display = await btn.evaluate(el => getComputedStyle(el).display);
  expect(display).toBe('none');
});
```

- [ ] **Step 2: Run the test and verify it fails**

From `docs/www-tests`:

```bash
npm run test:e2e -- drawer.spec.mjs
```

Expected: FAIL — `[data-testid="drawer-open"]` locator does not resolve. That's what we want; it proves the test is genuinely gating our change.

- [ ] **Step 3: Update topbar.html**

Replace `docs/layouts/partials/topbar.html` with:

```html
<header class="docs-topbar">
  <div class="docs-topbar-inner">
    <button type="button"
            class="drawer-btn"
            data-testid="drawer-open"
            aria-label="Open navigation"
            aria-controls="docs-sidebar"
            aria-expanded="false">
      <svg class="drawer-btn-icon" viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
        <path d="M3 6h18M3 12h18M3 18h18"/>
      </svg>
    </button>
    <a class="docs-brand" href="{{ "/" | relURL }}">Gormes</a>
    <div id="search" class="docs-search"></div>
    <nav class="docs-topnav">
      <a href="https://gormes.ai/">gormes.ai</a>
      <a href="https://github.com/TrebuchetDynamics/gormes-agent">GitHub</a>
    </nav>
  </div>
</header>
```

- [ ] **Step 4: Update baseof.html to remove the checkbox + wire site.js**

Replace `docs/layouts/_default/baseof.html` with:

```html
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{ if .IsHome }}{{ site.Title }}{{ else }}{{ .Title }} · {{ site.Title }}{{ end }}</title>
  <meta name="description" content="{{ if .Description }}{{ .Description }}{{ else }}{{ site.Params.description | default "Gormes documentation." }}{{ end }}">
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=DM+Sans:opsz,wght@9..40,400;9..40,500;9..40,700&family=Fraunces:ital,opsz,wght,SOFT@0,9..144,400;0,9..144,700;0,9..144,900;1,9..144,400;1,9..144,700&family=JetBrains+Mono:wght@400;500;700&display=swap">
  <link rel="stylesheet" href="{{ "site.css" | relURL }}">
  <link rel="stylesheet" href="/pagefind/pagefind-ui.css">
  <script src="/pagefind/pagefind-ui.js" defer></script>
  <script src="{{ "site.js" | relURL }}" defer></script>
</head>
<body>
  <div class="grain" aria-hidden="true"></div>
  {{ partial "topbar.html" . }}
  <div class="docs-shell">
    <aside class="docs-sidebar" id="docs-sidebar" data-state="closed">
      {{ partial "sidebar.html" . }}
    </aside>
    <div class="drawer-backdrop" data-testid="drawer-backdrop" aria-hidden="true"></div>
    <main class="docs-main">
      {{ block "main" . }}{{ end }}
    </main>
  </div>
  {{ partial "footer.html" . }}
  <script>
    function gormesCopy(b) {
      var code = b.parentElement.querySelector('code').innerText;
      navigator.clipboard.writeText(code).then(function () {
        var label = b.querySelector('.copy-label');
        var orig = label.textContent;
        label.textContent = 'Copied';
        b.classList.add('copied');
        setTimeout(function () {
          label.textContent = orig;
          b.classList.remove('copied');
        }, 1500);
      });
    }
    document.addEventListener('DOMContentLoaded', function() {
      if (window.PagefindUI) {
        new PagefindUI({ element: "#search", showSubResults: true });
      }
    });
  </script>
</body>
</html>
```

Changes vs current: removed `<input type="checkbox" id="drawer-toggle">`, removed `<label for="drawer-toggle" class="drawer-backdrop">` (backdrop is now a plain `<div>`), added `id="docs-sidebar"` + `data-state="closed"` on the sidebar, added `<script src="site.js" defer>`.

- [ ] **Step 5: Update CSS — decouple drawer from checkbox selector**

Edit `docs/static/site.css`. Inside the `@media (max-width: 767px)` block (around line 452), replace:

```css
  .drawer-toggle:checked ~ .docs-shell .docs-sidebar { transform: translateX(0); }
  .drawer-toggle:checked ~ .docs-shell .drawer-backdrop {
    display: block;
    position: fixed;
    inset: var(--topbar-h) 0 0 0;
    background: rgba(0, 0, 0, 0.55);
    z-index: 35;
    cursor: pointer;
  }
```

with:

```css
  .docs-sidebar[data-state="open"] { transform: translateX(0); }
  .docs-sidebar[data-state="open"] ~ .drawer-backdrop {
    display: block;
    position: fixed;
    inset: var(--topbar-h) 0 0 0;
    background: rgba(0, 0, 0, 0.55);
    z-index: 35;
    cursor: pointer;
  }
```

Also, near the top of the file (around line 65), remove the stale rule:

```css
.drawer-toggle { display: none; }
```

- [ ] **Step 6: Wire the drawer button in site.js (provisional)**

Edit `docs/static/site.js` and replace its content with:

```javascript
// docs.gormes.ai interactive behavior. Vanilla, no deps.
(function () {
  'use strict';

  function onReady(fn) {
    if (document.readyState !== 'loading') fn();
    else document.addEventListener('DOMContentLoaded', fn);
  }

  function setDrawer(state) {
    var sidebar = document.getElementById('docs-sidebar');
    var btn = document.querySelector('[data-testid="drawer-open"]');
    if (!sidebar) return;
    sidebar.setAttribute('data-state', state);
    if (btn) btn.setAttribute('aria-expanded', state === 'open' ? 'true' : 'false');
  }

  function initDrawer() {
    var btn = document.querySelector('[data-testid="drawer-open"]');
    var backdrop = document.querySelector('.drawer-backdrop');
    if (!btn) return;
    btn.addEventListener('click', function () {
      var sidebar = document.getElementById('docs-sidebar');
      var isOpen = sidebar && sidebar.getAttribute('data-state') === 'open';
      setDrawer(isOpen ? 'closed' : 'open');
    });
    if (backdrop) backdrop.addEventListener('click', function () { setDrawer('closed'); });
  }

  onReady(function () {
    initDrawer();
  });
})();
```

- [ ] **Step 7: Run tests and verify green**

From `gormes/`:

```bash
go test ./docs -run TestHugoBuild -count=1
cd docs/www-tests && npm run test:e2e -- drawer.spec.mjs
```

Expected: both pass. If `mobile drawer opens via hamburger and closes via backdrop` fails, the likely cause is the backdrop selector — verify the CSS rule for `.docs-sidebar[data-state="open"] ~ .drawer-backdrop` was placed correctly inside the `@media (max-width: 767px)` block.

- [ ] **Step 8: Commit**

```bash
cd gormes
git add docs/layouts/partials/topbar.html docs/layouts/_default/baseof.html docs/static/site.css docs/static/site.js docs/www-tests/tests/drawer.spec.mjs
git commit -m "docs(ui): svg hamburger button + js-driven drawer state"
```

---

### Task 3: Drawer header with close button + Esc key close + auto-close on link tap

**Files:**
- Modify: `docs/layouts/partials/sidebar.html`
- Modify: `docs/static/site.js`
- Modify: `docs/static/site.css`
- Test: `docs/www-tests/tests/drawer.spec.mjs`

- [ ] **Step 1: Extend the failing tests**

Append to `docs/www-tests/tests/drawer.spec.mjs` (at the end of the file):

```javascript
test('mobile drawer has a close button inside and Esc closes it', async ({ page }) => {
  await page.setViewportSize({ width: 360, height: 760 });
  await page.goto('/using-gormes/quickstart/');

  await page.locator('[data-testid="drawer-open"]').click();
  await page.waitForTimeout(250);

  const closeBtn = page.locator('[data-testid="drawer-close"]');
  await expect(closeBtn).toBeVisible();
  await closeBtn.click();
  await page.waitForTimeout(250);

  const sidebar = page.locator('.docs-sidebar');
  const leftAfterCloseClick = await sidebar.evaluate(el => el.getBoundingClientRect().left);
  expect(leftAfterCloseClick).toBeLessThan(0);

  // Re-open and close via Esc
  await page.locator('[data-testid="drawer-open"]').click();
  await page.waitForTimeout(250);
  await page.keyboard.press('Escape');
  await page.waitForTimeout(250);
  const leftAfterEsc = await sidebar.evaluate(el => el.getBoundingClientRect().left);
  expect(leftAfterEsc).toBeLessThan(0);
});

test('mobile drawer auto-closes on link tap', async ({ page }) => {
  await page.setViewportSize({ width: 360, height: 760 });
  await page.goto('/using-gormes/');

  await page.locator('[data-testid="drawer-open"]').click();
  await page.waitForTimeout(250);

  // Click any nav link inside the sidebar
  const link = page.locator('.docs-sidebar a[href]').first();
  await link.click();
  await page.waitForTimeout(400); // navigation + transition

  const sidebar = page.locator('.docs-sidebar');
  const left = await sidebar.evaluate(el => el.getBoundingClientRect().left);
  expect(left).toBeLessThan(0);
});
```

- [ ] **Step 2: Run and verify it fails**

From `docs/www-tests`:

```bash
npm run test:e2e -- drawer.spec.mjs
```

Expected: the two new tests fail — `[data-testid="drawer-close"]` not found, no Esc handler, no auto-close.

- [ ] **Step 3: Add the drawer header to sidebar.html**

Replace `docs/layouts/partials/sidebar.html` with:

```html
<div class="docs-drawer-header">
  <span class="docs-drawer-label">Navigation</span>
  <button type="button"
          class="docs-drawer-close"
          data-testid="drawer-close"
          aria-label="Close navigation">
    <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
      <path d="M6 6l12 12M18 6L6 18"/>
    </svg>
  </button>
</div>
<nav class="docs-nav" aria-label="Documentation navigation">
  {{ $sections := slice "using-gormes" "building-gormes" "upstream-hermes" }}
  {{ $toneMap := dict "using-gormes" "shipped" "building-gormes" "progress" "upstream-hermes" "next" }}
  {{ $labelMap := dict "using-gormes" "USING GORMES" "building-gormes" "BUILDING GORMES" "upstream-hermes" "UPSTREAM HERMES" }}
  {{ range $sections }}
    {{ $section := . }}
    {{ $tone := index $toneMap $section }}
    {{ $label := index $labelMap $section }}
    {{ with site.GetPage (printf "/%s" $section) }}
      <div class="docs-nav-group">
        <p class="docs-nav-group-label docs-nav-group-label-{{ $tone }}">{{ $label }}</p>
        <ul class="docs-nav-list">
          <li><a href="{{ .RelPermalink }}"{{ if eq $.RelPermalink .RelPermalink }} aria-current="page"{{ end }}>Overview</a></li>
          {{ range .Pages.ByWeight }}
          <li>
            <a href="{{ .RelPermalink }}"{{ if eq $.RelPermalink .RelPermalink }} aria-current="page"{{ end }}>{{ .Title }}</a>
            {{ if .Pages }}
            <ul class="docs-nav-sublist">
              {{ range .Pages.ByWeight }}
              <li><a href="{{ .RelPermalink }}"{{ if eq $.RelPermalink .RelPermalink }} aria-current="page"{{ end }}>{{ .Title }}</a></li>
              {{ end }}
            </ul>
            {{ end }}
          </li>
          {{ end }}
        </ul>
      </div>
    {{ end }}
  {{ end }}
</nav>
```

(Only addition vs prior version: the `<div class="docs-drawer-header">` at the top. Rest of the nav is unchanged — collapsible `<details>` wrapping comes in Task 4.)

- [ ] **Step 4: Add minimal CSS for the drawer header (structural only, polish in Phase 2)**

Append this block to `docs/static/site.css` (inside the `@media (max-width: 767px)` block, before the closing `}`):

```css
  .docs-drawer-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin: 0 0 16px;
    padding: 0 0 12px;
    border-bottom: 1px solid var(--border);
  }
  .docs-drawer-label {
    font-family: var(--font-mono);
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 0.18em;
    text-transform: uppercase;
    color: var(--label);
  }
  .docs-drawer-close {
    width: 28px;
    height: 28px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: 1px solid var(--border-strong);
    border-radius: 3px;
    color: var(--muted);
    cursor: pointer;
  }
  .docs-drawer-close:hover { color: var(--accent); border-color: var(--accent); }
```

And above 767px, hide the header (append inside the file, outside any `@media`):

```css
.docs-drawer-header { display: none; }
```

- [ ] **Step 5: Extend site.js for close button, Esc, and link auto-close**

Replace the `initDrawer` function inside `docs/static/site.js` with:

```javascript
  function initDrawer() {
    var openBtn = document.querySelector('[data-testid="drawer-open"]');
    var closeBtn = document.querySelector('[data-testid="drawer-close"]');
    var backdrop = document.querySelector('.drawer-backdrop');
    var sidebar = document.getElementById('docs-sidebar');
    if (!openBtn || !sidebar) return;

    openBtn.addEventListener('click', function () {
      var isOpen = sidebar.getAttribute('data-state') === 'open';
      setDrawer(isOpen ? 'closed' : 'open');
    });

    if (closeBtn) closeBtn.addEventListener('click', function () { setDrawer('closed'); });
    if (backdrop) backdrop.addEventListener('click', function () { setDrawer('closed'); });

    // Esc closes
    document.addEventListener('keydown', function (e) {
      if (e.key === 'Escape' && sidebar.getAttribute('data-state') === 'open') {
        setDrawer('closed');
      }
    });

    // Link tap inside drawer → close. Use event delegation so collapsible
    // groups added in Task 4 still get captured.
    sidebar.addEventListener('click', function (e) {
      var a = e.target.closest('a[href]');
      if (!a) return;
      if (sidebar.getAttribute('data-state') === 'open') {
        setDrawer('closed');
      }
    });
  }
```

- [ ] **Step 6: Run tests and verify green**

From `docs/www-tests`:

```bash
npm run test:e2e -- drawer.spec.mjs
```

Expected: all four drawer tests pass.

- [ ] **Step 7: Commit**

```bash
cd gormes
git add docs/layouts/partials/sidebar.html docs/static/site.js docs/static/site.css docs/www-tests/tests/drawer.spec.mjs
git commit -m "docs(ui): drawer close button, Esc handler, auto-close on link tap"
```

---

### Task 4: Collapsible sidebar groups with localStorage persistence

**Files:**
- Modify: `docs/layouts/partials/sidebar.html`
- Modify: `docs/static/site.js`
- Modify: `docs/static/site.css`
- Test: `docs/www-tests/tests/mobile.spec.mjs` (add collapsible group test)

- [ ] **Step 1: Add the failing test**

Append to `docs/www-tests/tests/mobile.spec.mjs` at the very end of the file (after the closing `}` of the `for (const vp...` loop):

```javascript
test('collapsible sidebar group: current auto-opens, others closed, click toggles', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await page.goto('/using-gormes/quickstart/');

  // Using Gormes contains Quickstart → that group is open.
  const usingGroup = page.locator('details.docs-nav-group[data-section="using-gormes"]');
  await expect(usingGroup).toHaveAttribute('open', '');

  // Building is closed by default (not current).
  const buildingGroup = page.locator('details.docs-nav-group[data-section="building-gormes"]');
  const openAttr = await buildingGroup.getAttribute('open');
  expect(openAttr).toBeNull();

  // Clicking the building summary expands it.
  await buildingGroup.locator('summary').click();
  await page.waitForTimeout(100);
  await expect(buildingGroup).toHaveAttribute('open', '');

  // Reload — persisted via localStorage.
  await page.reload();
  await expect(page.locator('details.docs-nav-group[data-section="building-gormes"]')).toHaveAttribute('open', '');
});
```

- [ ] **Step 2: Run and verify it fails**

From `docs/www-tests`:

```bash
npm run test:e2e -- mobile.spec.mjs -g "collapsible"
```

Expected: FAIL — `details.docs-nav-group` not found (they're `<div>` today).

- [ ] **Step 3: Replace `<div>` groups with `<details>` in sidebar.html**

Replace the `{{ range $sections }}` loop body inside `docs/layouts/partials/sidebar.html` (replace lines 10-28 of the current file — the `<div class="docs-nav-group">` block) with this — keeping the drawer header and outer `<nav>` untouched:

```html
    {{ with site.GetPage (printf "/%s" $section) }}
      {{ $isCurrentSection := or (eq $.RelPermalink .RelPermalink) (hasPrefix $.RelPermalink .RelPermalink) }}
      <details class="docs-nav-group"
               data-section="{{ $section }}"
               {{ if $isCurrentSection }}data-current open{{ end }}>
        <summary class="docs-nav-group-header">
          <span class="docs-nav-chev" aria-hidden="true">▶</span>
          <span class="docs-nav-dot docs-nav-dot--{{ $tone }}" aria-hidden="true"></span>
          <span class="docs-nav-group-label docs-nav-group-label-{{ $tone }}">{{ $label }}</span>
          <span class="docs-nav-group-count">{{ len .Pages }}</span>
        </summary>
        <ul class="docs-nav-list">
          <li><a href="{{ .RelPermalink }}"{{ if eq $.RelPermalink .RelPermalink }} aria-current="page"{{ end }}>Overview</a></li>
          {{ range .Pages.ByWeight }}
          <li>
            <a href="{{ .RelPermalink }}"{{ if eq $.RelPermalink .RelPermalink }} aria-current="page"{{ end }}>{{ .Title }}</a>
            {{ if .Pages }}
            <ul class="docs-nav-sublist">
              {{ range .Pages.ByWeight }}
              <li><a href="{{ .RelPermalink }}"{{ if eq $.RelPermalink .RelPermalink }} aria-current="page"{{ end }}>{{ .Title }}</a></li>
              {{ end }}
            </ul>
            {{ end }}
          </li>
          {{ end }}
        </ul>
      </details>
    {{ end }}
```

Note: `$isCurrentSection` uses a naive `hasPrefix` match on RelPermalink — this correctly flags both the section index page and any descendant. For a "current page" detection that is also robust at arbitrary nesting, use `.InSection` or `.IsDescendant` — but `hasPrefix` is fine for three top-level sections.

- [ ] **Step 4: Add minimal structural CSS for `<details>` summary**

Append to `docs/static/site.css` (at end of file, in the main scope — polish in Phase 2):

```css
/* Collapsible sidebar — structural only; polish in Phase 2 */
.docs-nav-group summary {
  list-style: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  margin: 0 0 4px;
  border-radius: 4px;
}
.docs-nav-group summary::-webkit-details-marker,
.docs-nav-group summary::marker { display: none; }
.docs-nav-group[open] > summary .docs-nav-chev { transform: rotate(90deg); }
.docs-nav-chev {
  font-size: 9px;
  color: var(--label);
  transition: transform 0.12s;
  width: 10px;
  display: inline-block;
}
.docs-nav-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}
.docs-nav-dot--shipped { background: var(--status-shipped-fg); }
.docs-nav-dot--progress { background: var(--status-progress-fg); }
.docs-nav-dot--next { background: var(--status-next-fg); }
.docs-nav-group-count {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--label);
  margin-left: auto;
}
```

Also, in the existing `.docs-nav-group-label` rule (around line 139), remove the `border-left: 2px solid currentColor` and `padding-left: 10px` (the colored dot replaces that visual). Change:

```css
.docs-nav-group-label {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.22em;
  text-transform: uppercase;
  margin: 0 0 10px;
  padding-left: 10px;
  border-left: 2px solid currentColor;
}
```

to:

```css
.docs-nav-group-label {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  margin: 0;
  flex: 1;
}
```

- [ ] **Step 5: Add collapsible-state persistence in site.js**

Append a new function and call inside the `onReady` at the bottom of `docs/static/site.js`:

```javascript
  var STORAGE_KEY = 'docs-nav-groups';

  function readGroupState() {
    try {
      var raw = localStorage.getItem(STORAGE_KEY);
      return raw ? JSON.parse(raw) : {};
    } catch (_) { return {}; }
  }

  function writeGroupState(state) {
    try { localStorage.setItem(STORAGE_KEY, JSON.stringify(state)); } catch (_) {}
  }

  function initCollapsibleGroups() {
    var groups = document.querySelectorAll('details.docs-nav-group');
    if (!groups.length) return;
    var state = readGroupState();

    groups.forEach(function (g) {
      var key = g.getAttribute('data-section');
      if (!key) return;
      // Current section always opens, regardless of stored preference.
      if (g.hasAttribute('data-current')) {
        g.setAttribute('open', '');
        return;
      }
      if (state[key] === 'open') g.setAttribute('open', '');
      else if (state[key] === 'closed') g.removeAttribute('open');
    });

    groups.forEach(function (g) {
      var key = g.getAttribute('data-section');
      if (!key) return;
      g.addEventListener('toggle', function () {
        var snapshot = readGroupState();
        snapshot[key] = g.hasAttribute('open') ? 'open' : 'closed';
        writeGroupState(snapshot);
      });
    });
  }
```

And update the bottom `onReady` callback:

```javascript
  onReady(function () {
    initDrawer();
    initCollapsibleGroups();
  });
```

- [ ] **Step 6: Run tests and verify green**

From `gormes/`:

```bash
go test ./docs -run TestHugoBuild -count=1
cd docs/www-tests && npm run test:e2e -- drawer.spec.mjs mobile.spec.mjs
```

Expected: all green. If the collapsible test fails because a reload clears localStorage, the Playwright driver preserves it by default — verify via `page.evaluate(() => localStorage.getItem('docs-nav-groups'))` in a debugger.

- [ ] **Step 7: Commit**

```bash
cd gormes
git add docs/layouts/partials/sidebar.html docs/static/site.js docs/static/site.css docs/www-tests/tests/mobile.spec.mjs
git commit -m "docs(ui): collapsible sidebar groups with localStorage persistence"
```

---

### Task 5: Prev/Next partial + single.html inclusion

**Files:**
- Create: `docs/layouts/partials/prevnext.html`
- Modify: `docs/layouts/_default/single.html`
- Test: new Playwright spec `docs/www-tests/tests/prevnext.spec.mjs`

- [ ] **Step 1: Create the failing test**

Create `docs/www-tests/tests/prevnext.spec.mjs` with:

```javascript
import { test, expect } from '@playwright/test';

test('single page has prev/next links at bottom', async ({ page }) => {
  await page.goto('/using-gormes/quickstart/');
  const nav = page.locator('nav.docs-prevnext');
  await expect(nav).toBeVisible();

  // At least one of prev or next must exist on a non-boundary page.
  const anchors = nav.locator('a');
  const count = await anchors.count();
  expect(count).toBeGreaterThanOrEqual(1);

  // Each anchor has a direction label + a page title.
  for (let i = 0; i < count; i++) {
    const a = anchors.nth(i);
    await expect(a.locator('.docs-prevnext-dir')).toBeVisible();
    await expect(a.locator('.docs-prevnext-title')).toBeVisible();
  }
});

test('section index page does not show prev/next', async ({ page }) => {
  await page.goto('/using-gormes/');
  const nav = page.locator('nav.docs-prevnext');
  await expect(nav).toHaveCount(0);
});
```

- [ ] **Step 2: Run and verify it fails**

From `docs/www-tests`:

```bash
npm run test:e2e -- prevnext.spec.mjs
```

Expected: FAIL — `nav.docs-prevnext` not found.

- [ ] **Step 3: Create the prevnext partial**

Create `docs/layouts/partials/prevnext.html` with:

```html
{{ with .Parent }}
  {{ $siblings := .Pages.ByWeight }}
  {{ $current := $ }}
  {{ $idx := -1 }}
  {{ range $i, $p := $siblings }}
    {{ if eq $p.RelPermalink $current.RelPermalink }}
      {{ $idx = $i }}
    {{ end }}
  {{ end }}
  {{ if ge $idx 0 }}
    {{ $prev := "" }}
    {{ $next := "" }}
    {{ if gt $idx 0 }}{{ $prev = index $siblings (sub $idx 1) }}{{ end }}
    {{ $last := sub (len $siblings) 1 }}
    {{ if lt $idx $last }}{{ $next = index $siblings (add $idx 1) }}{{ end }}
    {{ if or $prev $next }}
    <nav class="docs-prevnext" aria-label="Previous and next pages">
      {{ with $prev }}
      <a class="docs-prevnext-link docs-prevnext-link--prev" href="{{ .RelPermalink }}">
        <span class="docs-prevnext-dir">← Previous</span>
        <span class="docs-prevnext-title">{{ .Title }}</span>
      </a>
      {{ else }}<span></span>{{ end }}
      {{ with $next }}
      <a class="docs-prevnext-link docs-prevnext-link--next" href="{{ .RelPermalink }}">
        <span class="docs-prevnext-dir">Next →</span>
        <span class="docs-prevnext-title">{{ .Title }}</span>
      </a>
      {{ else }}<span></span>{{ end }}
    </nav>
    {{ end }}
  {{ end }}
{{ end }}
```

- [ ] **Step 4: Include the partial in single.html**

Replace `docs/layouts/_default/single.html` with:

```html
{{ define "main" }}
<article class="docs-article">
  {{ partial "breadcrumbs.html" . }}
  <header class="docs-article-header">
    <h1 class="docs-title">{{ .Title }}</h1>
    {{ if .Description }}<p class="docs-lede">{{ .Description }}</p>{{ end }}
  </header>
  <div class="docs-layout-with-toc">
    <div class="docs-content">
      {{ .Content }}
      {{ partial "prevnext.html" . }}
    </div>
    <aside class="docs-toc">
      {{ partial "toc.html" . }}
    </aside>
  </div>
</article>
{{ end }}
```

- [ ] **Step 5: Add minimal CSS for prev/next layout (polish in Phase 2)**

Append to `docs/static/site.css`:

```css
/* Prev/Next — structural only; polish in Phase 2 */
.docs-prevnext {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
  margin-top: 40px;
  padding-top: 22px;
  border-top: 1px solid var(--border);
}
.docs-prevnext-link {
  display: block;
  background: var(--bg-1);
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 14px 18px;
  color: var(--text);
  transition: border-color 0.15s;
}
.docs-prevnext-link:hover { border-color: var(--accent); color: var(--text); }
.docs-prevnext-link--next { text-align: right; }
.docs-prevnext-dir {
  display: block;
  font-family: var(--font-mono);
  font-size: 9px;
  letter-spacing: 0.16em;
  color: var(--label);
  text-transform: uppercase;
  margin: 0 0 4px;
}
.docs-prevnext-title {
  display: block;
  font-family: var(--font-display);
  font-size: 16px;
  font-weight: 700;
}
@media (max-width: 480px) {
  .docs-prevnext { grid-template-columns: 1fr; }
  .docs-prevnext-link--next { text-align: left; }
}
```

- [ ] **Step 6: Run tests and verify green**

From `gormes/`:

```bash
go test ./docs -run TestHugoBuild -count=1
cd docs/www-tests && npm run test:e2e -- prevnext.spec.mjs
```

Expected: both tests pass.

- [ ] **Step 7: Commit**

```bash
cd gormes
git add docs/layouts/partials/prevnext.html docs/layouts/_default/single.html docs/static/site.css docs/www-tests/tests/prevnext.spec.mjs
git commit -m "docs(ui): prev/next article nav from sibling pages"
```

---

### Task 6: De-duplicate "In this section" child list in list.html

**Files:**
- Modify: `docs/layouts/_default/list.html`

- [ ] **Step 1: Decide the rule**

Current `list.html` unconditionally appends an `<h2>In this section</h2>` with a child-list AFTER the `_index.md` body content renders. This is often redundant when `_index.md` already walks the reader through each child.

Rule: render the auto-list ONLY when `.Content` is effectively empty (length under 40 chars of stripped markdown).

- [ ] **Step 2: Update list.html**

Replace `docs/layouts/_default/list.html` with:

```html
{{ define "main" }}
<article class="docs-article">
  {{ partial "breadcrumbs.html" . }}
  <header class="docs-article-header">
    <h1 class="docs-title">{{ .Title }}</h1>
    {{ if .Description }}<p class="docs-lede">{{ .Description }}</p>{{ end }}
  </header>
  <div class="docs-content">
    {{ .Content }}
    {{ $stripped := replaceRE "(?s)<[^>]+>" "" .Content }}
    {{ $stripped = trim $stripped " \t\n\r" }}
    {{ if lt (len $stripped) 40 }}
    <h2>In this section</h2>
    <ul class="docs-child-list">
      {{ range (.Pages.ByWeight) }}
      <li>
        <a href="{{ .RelPermalink }}">{{ .Title }}</a>
        {{ if .Description }}<p>{{ .Description }}</p>{{ end }}
      </li>
      {{ end }}
    </ul>
    {{ end }}
  </div>
</article>
{{ end }}
```

- [ ] **Step 3: Verify Hugo build still emits all expected pages**

From `gormes/`:

```bash
go test ./docs -run TestHugoBuild -count=1
```

Expected: PASS (build_test.go only asserts pages exist, not that they contain the auto-list).

- [ ] **Step 4: Quick smoke — section indexes still render**

From `gormes/`:

```bash
hugo --minify -d /tmp/docs-smoke && grep -l "In this section" /tmp/docs-smoke/using-gormes/index.html /tmp/docs-smoke/building-gormes/index.html /tmp/docs-smoke/upstream-hermes/index.html 2>&1 | head
```

Expected: zero or one match depending on whether the three `_index.md` files have body content. Either way, the pages themselves must exist and contain the title + rendered body.

- [ ] **Step 5: Commit**

```bash
cd gormes
git add docs/layouts/_default/list.html
git commit -m "docs(ui): suppress auto child-list when section _index already has body"
```

---

### Task 7: TOC scrollspy (IntersectionObserver)

**Files:**
- Modify: `docs/layouts/partials/toc.html`
- Modify: `docs/static/site.js`
- Modify: `docs/static/site.css`
- Test: new Playwright spec `docs/www-tests/tests/scrollspy.spec.mjs`

- [ ] **Step 1: Create the failing test**

Create `docs/www-tests/tests/scrollspy.spec.mjs` with:

```javascript
import { test, expect } from '@playwright/test';

test('TOC scrollspy highlights the currently visible heading', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await page.goto('/building-gormes/architecture_plan/phase-6-learning-loop/');

  const toc = page.locator('.docs-toc-body');
  await expect(toc).toBeVisible();

  // Find all anchors in the TOC
  const links = toc.locator('a[href^="#"]');
  const count = await links.count();
  if (count < 2) test.skip(); // page doesn't have enough headings

  // Scroll to the second heading; the second TOC link should be .active
  const secondHref = await links.nth(1).getAttribute('href');
  expect(secondHref).toBeTruthy();
  const anchorId = secondHref.replace('#', '');
  await page.evaluate(id => {
    document.getElementById(id).scrollIntoView({ behavior: 'instant', block: 'start' });
  }, anchorId);
  await page.waitForTimeout(250);

  const activeCount = await toc.locator('a.active').count();
  expect(activeCount).toBeGreaterThanOrEqual(1);
  const firstActiveHref = await toc.locator('a.active').first().getAttribute('href');
  expect(firstActiveHref).toBe(secondHref);
});
```

- [ ] **Step 2: Run and verify it fails**

From `docs/www-tests`:

```bash
npm run test:e2e -- scrollspy.spec.mjs
```

Expected: FAIL — no `.active` class gets toggled on scroll today.

- [ ] **Step 3: Keep toc.html simple — the Hugo default already emits `<a href="#id">`**

Verify the current `docs/layouts/partials/toc.html` emits anchors with `href="#..."`. It currently uses `.TableOfContents`, which Hugo generates from heading IDs. Inspect the output:

```bash
hugo --minify -d /tmp/docs-toc && grep -A2 docs-toc-body /tmp/docs-toc/building-gormes/architecture_plan/phase-6-learning-loop/index.html | head -40
```

If the output shows `<a href="#some-id">…</a>` inside `.docs-toc-body`, proceed. If Hugo emits fragments without hrefs, re-emit manually (see Step 3b below).

Assuming the default output is usable, replace `docs/layouts/partials/toc.html` with:

```html
{{ if .TableOfContents }}
<details class="docs-toc-details" open>
  <summary class="docs-toc-summary">On this page</summary>
  <nav class="docs-toc-body" aria-label="Table of contents">
    {{ .TableOfContents }}
  </nav>
</details>
{{ end }}
```

- [ ] **Step 3b: Fallback — manual TOC re-emission (skip if Step 3 works)**

If `.TableOfContents` doesn't produce a targetable structure, replace with:

```html
{{ if .Fragments.Headings }}
<details class="docs-toc-details" open>
  <summary class="docs-toc-summary">On this page</summary>
  <nav class="docs-toc-body" aria-label="Table of contents">
    <ul>
      {{ range .Fragments.Headings }}
        {{ if and (ge .Level 2) (le .Level 3) }}
        <li>
          <a href="#{{ .ID }}">{{ .Title }}</a>
          {{ if .Headings }}
          <ul>
            {{ range .Headings }}
              {{ if le .Level 3 }}
              <li><a href="#{{ .ID }}">{{ .Title }}</a></li>
              {{ end }}
            {{ end }}
          </ul>
          {{ end }}
        </li>
        {{ end }}
      {{ end }}
    </ul>
  </nav>
</details>
{{ end }}
```

- [ ] **Step 4: Add the scrollspy to site.js**

Append to `docs/static/site.js`, inside the IIFE, above the final `onReady` call:

```javascript
  function initTocScrollspy() {
    var tocBody = document.querySelector('.docs-toc-body');
    if (!tocBody) return;
    var links = tocBody.querySelectorAll('a[href^="#"]');
    if (!links.length) return;
    var idToLink = {};
    links.forEach(function (a) {
      var id = decodeURIComponent(a.getAttribute('href').slice(1));
      if (id) idToLink[id] = a;
    });
    var headings = [];
    Object.keys(idToLink).forEach(function (id) {
      var el = document.getElementById(id);
      if (el) headings.push(el);
    });
    if (!headings.length) return;

    function setActive(id) {
      links.forEach(function (a) { a.classList.remove('active'); });
      if (idToLink[id]) idToLink[id].classList.add('active');
    }

    var observer = new IntersectionObserver(function (entries) {
      // Pick the entry closest to the top of the viewport that is intersecting.
      var best = null;
      entries.forEach(function (e) {
        if (!e.isIntersecting) return;
        if (!best || e.boundingClientRect.top < best.boundingClientRect.top) best = e;
      });
      if (best) setActive(best.target.id);
    }, { rootMargin: '-80px 0px -65% 0px', threshold: [0, 1.0] });

    headings.forEach(function (h) { observer.observe(h); });
  }
```

Update the `onReady` call:

```javascript
  onReady(function () {
    initDrawer();
    initCollapsibleGroups();
    initTocScrollspy();
  });
```

- [ ] **Step 5: Add minimal active-state CSS**

Append to `docs/static/site.css`:

```css
.docs-toc-body a.active {
  color: var(--accent);
  border-left-color: var(--accent);
}
```

- [ ] **Step 6: Run tests and verify green**

From `docs/www-tests`:

```bash
npm run test:e2e -- scrollspy.spec.mjs
```

Expected: PASS. If the test is `skipped` because the page doesn't have multiple headings, re-point the test at another page with headings (adjust the URL in Step 1) and rerun.

- [ ] **Step 7: Commit**

```bash
cd gormes
git add docs/layouts/partials/toc.html docs/static/site.js docs/static/site.css docs/www-tests/tests/scrollspy.spec.mjs
git commit -m "docs(ui): TOC scrollspy via IntersectionObserver"
```

---

### Task 8: Update home.spec.mjs for the new Phase 1 baseline

**Files:**
- Modify: `docs/www-tests/tests/home.spec.mjs`

- [ ] **Step 1: Update the script-count assertion**

Replace `docs/www-tests/tests/home.spec.mjs` with:

```javascript
import { test, expect } from '@playwright/test';

test('docs home renders the three-audience split', async ({ page }) => {
  await page.goto('/');

  await expect(page).toHaveTitle(/Gormes Docs/);
  await expect(page.getByRole('heading', { name: /Gormes Docs|Gormes/, level: 1 })).toBeVisible();

  // Three cards, one per audience
  await expect(page.getByRole('link', { name: /USING GORMES/i })).toBeVisible();
  await expect(page.getByRole('link', { name: /BUILDING GORMES/i })).toBeVisible();
  await expect(page.getByRole('link', { name: /UPSTREAM HERMES/i })).toBeVisible();

  // Sidebar has colored group labels
  await expect(page.locator('.docs-nav-group-label-shipped')).toBeVisible();
  await expect(page.locator('.docs-nav-group-label-progress')).toBeVisible();
  await expect(page.locator('.docs-nav-group-label-next')).toBeVisible();

  // Phase 1 adds site.js alongside pagefind-ui.js — allow up to 2 external scripts.
  const scripts = await page.locator('script[src]').count();
  expect(scripts).toBeLessThanOrEqual(2);
});
```

- [ ] **Step 2: Run tests and verify green**

```bash
cd docs/www-tests && npm run test:e2e -- home.spec.mjs
```

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
cd gormes
git add docs/www-tests/tests/home.spec.mjs
git commit -m "docs(ui): home test allows pagefind + site.js scripts"
```

---

### Task 9: Phase 1 integration check

**Files:**
- None (verification only)

- [ ] **Step 1: Full Go test pass**

From `gormes/`:

```bash
go test ./docs -count=1
```

Expected: PASS.

- [ ] **Step 2: Full Playwright pass**

From `docs/www-tests`:

```bash
npm run test:e2e
```

Expected: PASS across `drawer.spec.mjs`, `mobile.spec.mjs`, `home.spec.mjs`, `prevnext.spec.mjs`, `scrollspy.spec.mjs`.

- [ ] **Step 3: Manual smoke — serve and click through**

From `gormes/docs`:

```bash
hugo server -D --bind 127.0.0.1 --port 1313
```

Open `http://127.0.0.1:1313` and click through:

1. Home page loads, three cards visible, sidebar renders three `<details>` groups with counts.
2. Navigate to `/using-gormes/quickstart/` — "Using Gormes" group is open, others collapsed.
3. Click "Building Gormes" summary — expands; reload the page — stays expanded.
4. At 360px viewport (DevTools): hamburger is a button, click opens drawer, the `✕` close button appears, clicking a link closes drawer, Esc closes drawer.
5. Scroll a long article — the TOC highlights the current h2/h3.
6. Bottom of article has prev/next cards.

Stop the server (Ctrl-C).

- [ ] **Step 4: Tag Phase 1 complete**

```bash
cd gormes
git tag docs-ui-polish-phase-1
```

Phase 1 is now ready to merge. Before starting Phase 2, confirm Phase 1 is on `main` (or wherever the user merges it) so Phase 2 builds on stable DOM.

---

## Phase 2 — Visual Polish

### Task 10: Typography scale + `--bg-elev` token + rhythm

**Files:**
- Modify: `docs/static/site.css`

- [ ] **Step 1: Add the new CSS token**

Edit `docs/static/site.css` inside the `:root { }` block (around line 20). Add after the `--border-strong` line:

```css
  --bg-elev: #161c26;
```

- [ ] **Step 2: Bump body/lede size and line-height**

Change the `html, body` rule (around line 36):

```css
html, body {
  margin: 0;
  padding: 0;
  background: var(--bg-0);
  color: var(--text);
  font-family: var(--font-body);
  font-size: 16px;
  line-height: 1.6;
  -webkit-font-smoothing: antialiased;
  text-rendering: optimizeLegibility;
  font-feature-settings: 'kern', 'liga', 'calt';
}
```

Change the `.docs-lede` rule (around line 223):

```css
.docs-lede {
  font-size: 17px;
  color: var(--muted-strong);
  margin: 0 0 28px;
  line-height: 1.55;
  max-width: 60ch;
  overflow-wrap: break-word;
}
```

- [ ] **Step 3: Update h1/h2/h3 sizes + rhythm**

Change `.docs-title` (around line 213):

```css
.docs-title {
  font-family: var(--font-display);
  font-weight: 900;
  font-size: clamp(30px, 4.5vw, 44px);
  line-height: 1.05;
  letter-spacing: -0.02em;
  margin: 0 0 14px;
  font-variation-settings: "opsz" 144, "SOFT" 30;
  overflow-wrap: break-word;
}
```

Change `.docs-content h2` (around line 232):

```css
.docs-content h2 {
  font-family: var(--font-display);
  font-size: 22px;
  font-weight: 700;
  font-variation-settings: "opsz" 60, "SOFT" 20;
  letter-spacing: -0.01em;
  margin: 56px 0 14px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border);
  overflow-wrap: break-word;
}
```

Change `.docs-content h3` (around line 243):

```css
.docs-content h3 {
  font-family: var(--font-display);
  font-size: 17px;
  font-weight: 700;
  font-variation-settings: "opsz" 48, "SOFT" 15;
  margin: 28px 0 10px;
  overflow-wrap: break-word;
}
```

Change `.docs-content p` (around line 251):

```css
.docs-content p { margin: 0 0 16px; overflow-wrap: break-word; max-width: 62ch; }
```

- [ ] **Step 4: Mobile `<480px` h1 clamp tweak**

Inside the existing `@media (max-width: 480px)` block (around line 484), add:

```css
  .docs-title { font-size: clamp(26px, 7vw, 32px); }
```

- [ ] **Step 5: Verify Hugo build + mobile Playwright still green**

From `gormes/`:

```bash
go test ./docs -run TestHugoBuild -count=1
cd docs/www-tests && npm run test:e2e -- mobile.spec.mjs
```

Expected: PASS (mobile overflow assertion must still pass with the larger body font).

- [ ] **Step 6: Commit**

```bash
cd gormes
git add docs/static/site.css
git commit -m "docs(ui): tighter type scale, --bg-elev token, new rhythm"
```

---

### Task 11: Kicker + breadcrumb de-shout

**Files:**
- Modify: `docs/static/site.css`
- Modify: `docs/layouts/partials/breadcrumbs.html`

- [ ] **Step 1: Update `.kicker` and the breadcrumbs block**

In `docs/static/site.css`, change the `.kicker` rule (around line 376):

```css
.kicker {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--label);
  margin: 0 0 8px;
}
```

Change the breadcrumb block (around line 203):

```css
.docs-breadcrumbs {
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.04em;
  color: var(--label);
  margin-bottom: 14px;
  text-transform: none;
}
.docs-breadcrumbs a { color: var(--muted); }
.docs-breadcrumb-sep { margin: 0 6px; color: var(--label); }
```

- [ ] **Step 2: Drop `| upper` from breadcrumb titles**

Replace `docs/layouts/partials/breadcrumbs.html` with:

```html
<nav class="docs-breadcrumbs" aria-label="Breadcrumbs">
  {{ range .Ancestors.Reverse }}
    <a href="{{ .RelPermalink }}">{{ .Title }}</a>
    <span class="docs-breadcrumb-sep">/</span>
  {{ end }}
</nav>
```

- [ ] **Step 3: Verify build + Playwright**

```bash
cd gormes && go test ./docs -run TestHugoBuild -count=1
cd docs/www-tests && npm run test:e2e -- mobile.spec.mjs home.spec.mjs
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
cd gormes
git add docs/static/site.css docs/layouts/partials/breadcrumbs.html
git commit -m "docs(ui): quiet kickers + mixed-case breadcrumbs"
```

---

### Task 12: Sidebar density pass (13px items, hover treatment, active elev)

**Files:**
- Modify: `docs/static/site.css`

- [ ] **Step 1: Bump sidebar item sizes and active-state surface**

In `docs/static/site.css`, change `.docs-nav-list a, .docs-nav-sublist a` (around line 154):

```css
.docs-nav-list a, .docs-nav-sublist a {
  display: block;
  padding: 5px 10px;
  font-family: var(--font-mono);
  font-size: 13px;
  color: var(--muted-strong);
  border-left: 2px solid transparent;
  border-radius: 0;
  text-decoration: none;
  line-height: 1.5;
}
.docs-nav-list a:hover, .docs-nav-sublist a:hover {
  color: var(--accent);
  border-left-color: var(--border-strong);
  background: rgba(232, 197, 71, 0.04);
}
.docs-nav-list a[aria-current="page"],
.docs-nav-sublist a[aria-current="page"] {
  color: var(--accent);
  border-left-color: var(--accent);
  background: var(--bg-elev);
}
.docs-nav-sublist a { font-size: 12px; color: var(--muted); }
```

- [ ] **Step 2: Current-group marker — amber left bar on summary**

Append to `docs/static/site.css`:

```css
.docs-nav-group[data-current] > summary {
  background: rgba(232, 197, 71, 0.06);
  position: relative;
}
.docs-nav-group[data-current] > summary::before {
  content: '';
  position: absolute;
  left: -14px;
  top: 8px;
  bottom: 8px;
  width: 2px;
  background: var(--accent);
  border-radius: 2px;
}
.docs-nav-group > summary:hover { background: var(--bg-1); }
```

- [ ] **Step 3: Verify build + visual smoke**

```bash
cd gormes && go test ./docs -run TestHugoBuild -count=1
cd docs/www-tests && npm run test:e2e -- mobile.spec.mjs
```

Expected: PASS. Visual check via `hugo server`:
1. Sidebar items read as 13px mono.
2. Current group has amber left-bar on its summary.
3. Active link (`aria-current="page"`) uses the elevated bg (`#161c26`).

- [ ] **Step 4: Commit**

```bash
cd gormes
git add docs/static/site.css
git commit -m "docs(ui): sidebar density pass — 13px items, current-group marker, elevated active state"
```

---

### Task 13: TOC — switch to sans 12px, polish active state

**Files:**
- Modify: `docs/static/site.css`

- [ ] **Step 1: Update `.docs-toc-body a`**

In `docs/static/site.css`, change the rule (around line 354):

```css
.docs-toc-body a {
  display: block;
  padding: 3px 0 3px 10px;
  font-family: var(--font-body);
  font-size: 12px;
  color: var(--muted);
  border-left: 2px solid transparent;
  line-height: 1.45;
  text-decoration: none;
  transition: color 0.12s, border-left-color 0.12s;
}
.docs-toc-body a:hover { color: var(--accent); }
.docs-toc-body a.active {
  color: var(--accent);
  border-left-color: var(--accent);
  font-weight: 600;
}
.docs-toc-body ul ul { padding-left: 10px; }
```

- [ ] **Step 2: TOC summary stays quiet**

The existing `.docs-toc-details summary` is already good (mono uppercase label at 10px). Leave alone.

- [ ] **Step 3: Verify + commit**

```bash
cd gormes && go test ./docs -run TestHugoBuild -count=1
cd docs/www-tests && npm run test:e2e -- scrollspy.spec.mjs
```

```bash
cd gormes
git add docs/static/site.css
git commit -m "docs(ui): TOC — sans 12px with polished active state"
```

---

### Task 14: Code block language badge + copy button refinement

**Files:**
- Modify: `docs/layouts/_default/_markup/render-codeblock.html`
- Modify: `docs/static/site.css`

- [ ] **Step 1: Emit `data-lang` on the wrapper**

Replace `docs/layouts/_default/_markup/render-codeblock.html` with:

```html
<div class="cmd-wrap"{{ with .Type }} data-lang="{{ . }}"{{ end }}>
  <pre class="cmd"><code{{ with .Type }} class="language-{{ . }}"{{ end }}>{{ .Inner | safeHTML }}</code></pre>
  <button type="button" class="copy-btn" aria-label="Copy code" onclick="gormesCopy(this)">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
    <span class="copy-label">Copy</span>
  </button>
</div>
```

- [ ] **Step 2: Add language-badge + polished copy-button styles**

In `docs/static/site.css`, change the `.cmd` block (around line 302) and add the badge rules:

```css
.cmd-wrap {
  position: relative;
  min-width: 0;
  margin: 18px 0;
}
.cmd-wrap[data-lang]::before {
  content: attr(data-lang);
  position: absolute;
  top: 8px;
  left: 14px;
  font-family: var(--font-mono);
  font-size: 9px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--label);
  pointer-events: none;
}
.cmd {
  background: var(--bg-1);
  border: 1px solid var(--border);
  padding: 28px 84px 16px 16px;
  border-radius: 4px;
  font-family: var(--font-mono);
  font-size: 12.5px;
  margin: 0;
  min-width: 0;
  max-width: 100%;
  overflow-x: auto;
  line-height: 1.55;
}
.cmd-wrap:not([data-lang]) .cmd {
  padding-top: 16px;
}
.copy-btn {
  position: absolute;
  top: 8px;
  right: 8px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: transparent;
  border: 1px solid var(--border-strong);
  color: var(--muted);
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  padding: 5px 9px;
  border-radius: 2px;
  cursor: pointer;
  min-height: 28px;
  min-width: 28px;
}
.copy-btn:hover { color: var(--accent); border-color: var(--accent); }
.copy-btn.copied { background: rgba(14, 59, 33, 1); color: var(--status-shipped-fg); border-color: rgba(14, 59, 33, 1); }
```

In the `@media (max-width: 480px)` block, update:

```css
  .cmd { padding-right: 74px; }
```

- [ ] **Step 3: Verify build + Playwright**

```bash
cd gormes && go test ./docs -run TestHugoBuild -count=1
cd docs/www-tests && npm run test:e2e -- mobile.spec.mjs
```

The mobile test asserts copy buttons are ≥28×28; new `min-height`/`min-width` ensures that.

- [ ] **Step 4: Commit**

```bash
cd gormes
git add docs/layouts/_default/_markup/render-codeblock.html docs/static/site.css
git commit -m "docs(ui): code block language badge + polished copy button"
```

---

### Task 15: Hamburger button + drawer close styling

**Files:**
- Modify: `docs/static/site.css`

- [ ] **Step 1: Replace `.drawer-btn` styles**

In `docs/static/site.css`, replace the `.drawer-btn` rule (around line 66) with:

```css
.drawer-btn {
  display: none;
  width: 30px;
  height: 30px;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: 1px solid var(--border-strong);
  border-radius: 3px;
  color: var(--text);
  cursor: pointer;
  padding: 0;
}
.drawer-btn:hover { color: var(--accent); border-color: var(--accent); }
.drawer-btn-icon { display: block; }
```

In the `@media (max-width: 767px)` block, change:

```css
  .drawer-btn { display: inline-flex; }
```

- [ ] **Step 2: Verify**

```bash
cd gormes && go test ./docs -run TestHugoBuild -count=1
cd docs/www-tests && npm run test:e2e -- drawer.spec.mjs
```

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
cd gormes
git add docs/static/site.css
git commit -m "docs(ui): polished hamburger button (30x30, bordered, amber hover)"
```

---

### Task 16: New home page — hero + quickstart strip + ordinal cards with mini-TOC

**Files:**
- Modify: `docs/layouts/index.html`
- Modify: `docs/static/site.css`
- Modify: `docs/www-tests/tests/home.spec.mjs`

- [ ] **Step 1: Update the home test with new assertions**

Replace `docs/www-tests/tests/home.spec.mjs` with:

```javascript
import { test, expect } from '@playwright/test';

test('docs home hero, quickstart, and three enhanced cards render', async ({ page }) => {
  await page.goto('/');

  await expect(page).toHaveTitle(/Gormes Docs/);
  // Hero
  await expect(page.locator('.docs-home-hero h1')).toBeVisible();
  await expect(page.locator('.docs-home-hero .kicker')).toBeVisible();

  // Quickstart strip
  const qs = page.locator('.docs-home-quickstart');
  await expect(qs).toBeVisible();
  await expect(qs.locator('code')).toContainText(/brew|curl|go install|go run/i);

  // Three enhanced cards with ordinals and mini-TOCs
  const cards = page.locator('.docs-home-card');
  await expect(cards).toHaveCount(3);
  for (let i = 0; i < 3; i++) {
    const c = cards.nth(i);
    await expect(c.locator('.docs-home-card-ordinal')).toBeVisible();
    await expect(c.locator('.docs-home-card-mini-toc li')).toHaveCount(3);
    await expect(c.locator('.docs-home-card-cta')).toContainText(/Explore/i);
  }

  // Kickers map to the existing colored labels
  await expect(page.getByRole('link', { name: /USING GORMES/i })).toBeVisible();
  await expect(page.getByRole('link', { name: /BUILDING GORMES/i })).toBeVisible();
  await expect(page.getByRole('link', { name: /UPSTREAM HERMES/i })).toBeVisible();

  // Sidebar unchanged
  await expect(page.locator('.docs-nav-group-label-shipped')).toBeVisible();
  await expect(page.locator('.docs-nav-group-label-progress')).toBeVisible();
  await expect(page.locator('.docs-nav-group-label-next')).toBeVisible();

  // Still no runaway external scripts
  const scripts = await page.locator('script[src]').count();
  expect(scripts).toBeLessThanOrEqual(2);
});
```

- [ ] **Step 2: Run and verify it fails**

```bash
cd docs/www-tests && npm run test:e2e -- home.spec.mjs
```

Expected: FAIL — `.docs-home-hero`, `.docs-home-quickstart`, `.docs-home-card-ordinal`, etc. don't exist.

- [ ] **Step 3: Rewrite `docs/layouts/index.html`**

Replace with:

```html
{{ define "main" }}
{{ $using := site.GetPage "/using-gormes" }}
{{ $building := site.GetPage "/building-gormes" }}
{{ $upstream := site.GetPage "/upstream-hermes" }}
{{ $sections := slice
  (dict "page" $using "tone" "shipped" "label" "USING GORMES" "heading" "For operators" "blurb" "Install, run, configure. Wire up Telegram. Troubleshoot with the doctor." "ord" "01")
  (dict "page" $building "tone" "progress" "label" "BUILDING GORMES" "heading" "For contributors" "blurb" "Architecture, roadmap, how to port a subsystem from upstream." "ord" "02")
  (dict "page" $upstream "tone" "next" "label" "UPSTREAM HERMES" "heading" "For reference" "blurb" "Inherited Hermes guides. What the Python upstream does that Gormes is porting." "ord" "03")
}}

<article class="docs-home">
  <header class="docs-home-hero">
    <p class="kicker">Gormes · Documentation</p>
    <h1 class="docs-home-title">The Go operator shell for <em>Hermes.</em></h1>
    <p class="docs-lede">Install it, run it, understand how it's built. A single binary, no runtime — designed to replace the Python Hermes stack piece by piece.</p>
  </header>

  <section class="docs-home-quickstart">
    <div class="docs-home-quickstart-text">
      <p class="kicker docs-home-quickstart-kicker">New here? Start in 60 seconds</p>
      <h2 class="docs-home-quickstart-title">Install and run</h2>
      <p>A single binary on macOS or Linux. Telegram adapter optional.</p>
    </div>
    <code class="docs-home-quickstart-cmd">brew install trebuchet/gormes</code>
  </section>

  <section class="docs-home-cards">
    {{ range $sections }}
      {{ with .page }}
      <a class="docs-home-card" href="{{ .RelPermalink }}">
        <span class="docs-home-card-ordinal" aria-hidden="true">{{ $.ord }}</span>
        <p class="kicker docs-home-card-label docs-home-card-label--{{ $.tone }}">{{ $.label }}</p>
        <h2 class="docs-home-card-heading">{{ $.heading }}</h2>
        <p class="docs-home-card-blurb">{{ $.blurb }}</p>
        <ul class="docs-home-card-mini-toc">
          {{ range first 3 .Pages.ByWeight }}
          <li>{{ .Title }}</li>
          {{ end }}
        </ul>
        <span class="docs-home-card-cta">Explore →</span>
      </a>
      {{ end }}
    {{ end }}
  </section>
</article>
{{ end }}
```

- [ ] **Step 4: Replace old home styles and add new ones**

In `docs/static/site.css`, replace everything from `/* ── Home page ─ */` (around line 365) down to the existing `.docs-child-list li p` rule (around line 415) with:

```css
/* ── Home page ─────────────────────────────────────── */
.docs-home { padding-top: 32px; }
.docs-home-hero { margin: 0 0 32px; }
.docs-home-title {
  font-family: var(--font-display);
  font-weight: 900;
  font-size: clamp(36px, 6vw, 58px);
  line-height: 1;
  letter-spacing: -0.02em;
  margin: 14px 0 16px;
  font-variation-settings: "opsz" 144, "SOFT" 30;
}
.docs-home-title em { color: var(--accent); font-style: normal; }

.docs-home-quickstart {
  display: flex;
  align-items: center;
  gap: 24px;
  background: linear-gradient(135deg, var(--bg-2) 0%, var(--bg-1) 100%);
  border: 1px solid var(--border-strong);
  border-left: 3px solid var(--accent);
  border-radius: 6px;
  padding: 20px 24px;
  margin: 0 0 40px;
}
.docs-home-quickstart-text { flex: 1; min-width: 0; }
.docs-home-quickstart-kicker { color: var(--accent); margin: 0 0 4px; }
.docs-home-quickstart-title {
  font-family: var(--font-display);
  font-size: 20px;
  font-weight: 700;
  margin: 0 0 4px;
}
.docs-home-quickstart p { margin: 0; font-size: 14px; color: var(--muted-strong); }
.docs-home-quickstart-cmd {
  font-family: var(--font-mono);
  font-size: 12.5px;
  background: var(--bg-0);
  border: 1px solid var(--border);
  padding: 10px 14px;
  border-radius: 4px;
  color: var(--status-shipped-fg);
  white-space: nowrap;
}

.docs-home-cards {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 18px;
  margin-top: 0;
}
.docs-home-card {
  background: var(--bg-1);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 24px;
  color: var(--text);
  position: relative;
  display: flex;
  flex-direction: column;
  transition: border-color 0.2s, transform 0.2s;
}
.docs-home-card:hover { border-color: var(--accent); transform: translateY(-2px); color: var(--text); }
.docs-home-card-ordinal {
  font-family: var(--font-display);
  font-size: 48px;
  font-weight: 900;
  font-style: italic;
  color: rgba(235, 233, 226, 0.10);
  position: absolute;
  top: 12px;
  right: 18px;
  line-height: 1;
  pointer-events: none;
}
.docs-home-card-label { margin: 0 0 10px; }
.docs-home-card-label--shipped { color: var(--status-shipped-fg); }
.docs-home-card-label--progress { color: var(--status-progress-fg); }
.docs-home-card-label--next { color: var(--status-next-fg); }
.docs-home-card-heading {
  font-family: var(--font-display);
  font-size: 22px;
  font-weight: 700;
  margin: 0 0 8px;
  color: var(--text);
}
.docs-home-card-blurb {
  font-size: 13.5px;
  color: var(--muted-strong);
  margin: 0 0 18px;
  line-height: 1.5;
  flex: 1;
}
.docs-home-card-mini-toc {
  list-style: none;
  margin: 0 0 16px;
  padding: 12px 0 0;
  border-top: 1px solid var(--border);
}
.docs-home-card-mini-toc li {
  font-family: var(--font-mono);
  font-size: 12px;
  padding: 4px 0;
  color: var(--muted-strong);
}
.docs-home-card-mini-toc li::before { content: '→ '; color: var(--label); }
.docs-home-card-cta {
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--accent);
  margin-top: auto;
}

/* Legacy list — still used by list.html fallback */
.docs-child-list { list-style: none; padding: 0; display: grid; gap: 10px; }
.docs-child-list li {
  background: var(--bg-1);
  border: 1px solid var(--border);
  border-radius: 3px;
  padding: 12px 14px;
}
.docs-child-list li a { font-family: var(--font-mono); font-size: 13px; }
.docs-child-list li p { margin: 4px 0 0; font-size: 12px; color: var(--muted-strong); }
```

Inside the `@media (max-width: 1023px)` block (around line 443), replace:

```css
  .docs-home-cards { grid-template-columns: 1fr; }
```

with:

```css
  .docs-home-cards { grid-template-columns: 1fr; }
  .docs-home-quickstart { flex-direction: column; align-items: flex-start; }
  .docs-home-quickstart-cmd { align-self: stretch; text-align: center; }
```

- [ ] **Step 5: Run tests and verify green**

```bash
cd gormes && go test ./docs -run TestHugoBuild -count=1
cd docs/www-tests && npm run test:e2e -- home.spec.mjs mobile.spec.mjs
```

Expected: PASS. If mobile overflow fails, the quickstart `<code>` may be too wide at 320 px — verify the `flex-direction: column` rule activates at ≤1023 px, and consider adding `overflow-x: auto` on `.docs-home-quickstart-cmd`.

- [ ] **Step 6: Commit**

```bash
cd gormes
git add docs/layouts/index.html docs/static/site.css docs/www-tests/tests/home.spec.mjs
git commit -m "docs(ui): new home — hero, quickstart strip, ordinal cards with mini-toc"
```

---

### Task 17: Phase 2 integration check

**Files:**
- None (verification only)

- [ ] **Step 1: Full Go test pass**

From `gormes/`:

```bash
go test ./docs -count=1
```

Expected: PASS.

- [ ] **Step 2: Full Playwright pass**

From `docs/www-tests`:

```bash
npm run test:e2e
```

Expected: PASS across all specs: drawer, mobile (including all six viewports), home, prevnext, scrollspy.

- [ ] **Step 3: Manual visual smoke at multiple viewports**

From `gormes/docs`:

```bash
hugo server -D --bind 127.0.0.1 --port 1313
```

Open each viewport in DevTools and spot-check:

- **1280 px (desktop)**: Home hero has amber "Hermes." accent; quickstart strip has amber left-border; three ordinal cards; sidebar shows colored dots + counts; current-group has amber left-bar; prev/next at bottom of articles; TOC on the right highlights current heading in amber.
- **1024 px**: same, slightly tighter gutters.
- **768 px**: TOC collapses to a `<details>` with summary "On this page"; home cards become single column.
- **430 px / 390 px / 360 px / 320 px (mobile)**: hamburger button visible; drawer opens; close `✕` works; drawer auto-closes on link tap; Esc closes; code blocks show a language badge; no horizontal overflow on any page.

Stop the server (Ctrl-C).

- [ ] **Step 4: Tag Phase 2 complete**

```bash
cd gormes
git tag docs-ui-polish-phase-2
```

- [ ] **Step 5: Line-count sanity check**

```bash
wc -l docs/static/site.css docs/static/site.js
```

Expected: `site.css` in the ~700–800 range (was 496); `site.js` in the ~120–150 range (was 0). If either is substantially larger, look for opportunities to tighten.

---

## Post-implementation verification

After both phases merge, from `gormes/`:

```bash
go test ./docs -count=1 -v
cd docs/www-tests && npm run test:e2e
```

Both must pass. Manually verify Pagefind search still works (type into the top-bar search, results should appear in an overlay). If Pagefind breaks, confirm that `.docs-content` is still intact in article templates and rebuild the pagefind index per your existing deploy workflow.
