import { test, expect } from '@playwright/test';

test('homepage renders the redesigned landing', async ({ page }) => {
  await page.goto('/');

  await expect(page).toHaveTitle('Gormes — Run AI Agents as One Go Binary');
  await expect(page.getByRole('heading', { name: 'Run AI Agents as One Go Binary.' })).toBeVisible();
  await expect(page.getByText('Gormes is a Go-native runtime for long-running AI agents.')).toBeVisible();
  await expect(page.getByText('One static binary. No Python runtime. No Hermes process.')).toBeVisible();
  await expect(page.getByText('Ship the same binary you test. Run it anywhere.')).toBeVisible();
  await expect(page.getByText('Early-stage and shipping.')).toBeVisible();
  await expect(page.getByText('Offline TUI, local doctor, provider-backed one-shots, and Goncho memory are ready today.')).toBeVisible();
  await expect(page.locator('.topnav a')).toHaveText(['Docs', 'Roadmap', 'GitHub']);
  await expect(page.locator('.hero-image')).toHaveCount(0);
  await expect(page.locator('img[src="/static/go-gopher-bear-lowpoly.png"]')).toHaveCount(0);
  await expect(page.locator('.hero-ctas .btn-primary')).toHaveText('Install');
  await expect(page.locator('.hero-ctas .btn-secondary')).toHaveText('View on GitHub');
  await expect(page.getByRole('heading', { name: 'Why agent runtimes fail in production' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'How Gormes fixes the runtime surface' })).toBeVisible();
  await expect(page.getByText('Python-stack agents fail operationally when:')).toBeVisible();
  await expect(page.getByText('dev, staging, and prod stop matching')).toBeVisible();
  await expect(page.getByText('install scripts depend on host package luck')).toBeVisible();
  await expect(page.getByText('long turns die on dropped streams')).toBeVisible();
  await expect(page.getByText('tool wiring fails after tokens are already burning')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Useful today, still early.' })).toBeVisible();
  await expect(page.getByText('Current focus')).toBeVisible();
  await expect(page.getByText('Offline TUI, local doctor, and provider-backed one-shots')).toBeVisible();
  await expect(page.getByText('Gateway stability and shared channel contracts')).toBeVisible();
  await expect(page.getByText('Next milestone')).toBeVisible();
  await expect(page.getByText('Production-stable Go-native runtime, no Hermes process')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Install in one command. Verify everything works.' })).toBeVisible();
  await expect(page.getByText('One command to install')).toBeVisible();
  await expect(page.locator('#install').getByText('curl -fsSL https://gormes.ai/install.sh | sh', { exact: true })).toBeVisible();
  await expect(page.locator('#install').getByText('gormes --offline', { exact: true })).toBeVisible();
  await expect(page.locator('#install').getByText('gormes doctor --offline', { exact: true })).toBeVisible();
  await expect(page.locator('#install').getByText('gormes goncho doctor --json', { exact: true })).toBeVisible();
  await expect(page.locator('#install').getByText('GORMES_ENDPOINT=')).toBeVisible();
  await expect(page.locator('#install').getByText('git clone https://github.com/TrebuchetDynamics/gormes-agent.git')).toBeVisible();
  await expect(page.locator('#install').getByText('cd gormes-agent')).toBeVisible();
  await expect(page.locator('#install').getByText('make build')).toBeVisible();
  await expect(page.getByText('The installer clones, builds, and links gormes into your PATH')).toBeVisible();
  await expect(page.getByText('Transparent Local Memory')).toBeVisible();
  await expect(page.getByText('Release Trust Roadmap')).toBeVisible();
  await expect(page.getByText('AV false-positive submission')).toBeVisible();
  await expect(page.getByText('Read the installer source →')).toBeVisible();
  // New conversion sections
  await expect(page.getByText('~16.2 MB static binary')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'See it work in 30 seconds' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Built for operators who ship' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Explore' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Ready to try Gormes?' })).toBeVisible();
  await expect(page.getByText('Requires Hermes backend at localhost:8642.')).toHaveCount(0);
  await expect(page.getByText('Run Hermes Through a Go Operator Console.')).toHaveCount(0);
  await expect(page.getByText('Why Hermes breaks in production')).toHaveCount(0);
  await expect(page.getByText('irm https://gormes.ai/install.ps1 | iex')).toHaveCount(0);
  await expect(page.getByText('Deeper reference material lives at')).toHaveCount(0);
  await expect(page.locator('link[href="/static/site.css"]')).toHaveCount(1);
  // Copy buttons require a tiny inline clipboard script — bounded to install steps.
  // Six steps: install, offline TUI, doctor, memory audit, model-backed turn, build from source.
  await expect(page.locator('button.copy-btn')).toHaveCount(6);
});

// Long-term bulletproof: the page must stay readable as content
// grows (longer phase names, more ledger rows, more feature cards).
// Parametrize over multiple mobile widths so narrow viewports catch
// regressions from future copy/inventory expansion.
const MOBILE_VIEWPORTS = [
  { label: 'iPhone SE', width: 320, height: 568 },
  { label: 'small Android', width: 360, height: 760 },
  { label: 'iPhone 15', width: 390, height: 844 },
  { label: 'iPhone Plus', width: 430, height: 932 },
];

for (const vp of MOBILE_VIEWPORTS) {
  test(`mobile (${vp.label} ${vp.width}×${vp.height}) has no horizontal overflow`, async ({ page }) => {
    await page.setViewportSize({ width: vp.width, height: vp.height });
    await page.goto('/');

    await expect(page.getByRole('heading', { name: 'Run AI Agents as One Go Binary.' })).toBeVisible();
  await expect(page.getByText('curl -fsSL https://gormes.ai/install.sh | sh', { exact: true })).toBeVisible();

    const heroLayout = await page.evaluate(() => {
      const content = document.querySelector('.hero-content')?.getBoundingClientRect();
      const title = document.querySelector('.hero-title')?.getBoundingClientRect();
      return {
        contentWidth: content?.width ?? 0,
        titleWidth: title?.width ?? 0,
      };
    });
    expect(heroLayout.contentWidth, `hero content collapsed at ${vp.width}px`).toBeGreaterThan(vp.width * 0.6);
    expect(heroLayout.titleWidth, `hero title too wide at ${vp.width}px`).toBeLessThanOrEqual(vp.width);

    // The page itself must never generate a horizontal scrollbar. Long code
    // blocks get their own scroll inside .cmd via overflow-x: auto.
    const pageOverflow = await page.evaluate(() =>
      document.documentElement.scrollWidth > window.innerWidth,
    );
    expect(pageOverflow, `page body overflows at ${vp.width}px`).toBeFalsy();

    // Copy buttons stay visible + tappable on every supported viewport.
    // Six install steps: install, offline TUI, doctor, memory audit, model-backed turn, build from source.
    const copyButtons = page.locator('button.copy-btn');
    await expect(copyButtons).toHaveCount(6);
    for (let i = 0; i < 6; i++) {
      const btn = copyButtons.nth(i);
      await expect(btn).toBeVisible();
      const box = await btn.boundingBox();
      expect(box, `copy button ${i} has no bounding box`).not.toBeNull();
      // iOS HIG minimum touch target is 44×44; we pass with 32 min-height +
      // padding, but enforce at least 28×28 so future CSS tweaks can't
      // silently shrink the button below usability.
      expect(box.height, `copy button ${i} too short at ${vp.width}px`).toBeGreaterThanOrEqual(28);
      expect(box.width, `copy button ${i} too narrow at ${vp.width}px`).toBeGreaterThanOrEqual(28);
    }

    // The roadmap has 7 phase groups under a single disclosure. No phase
    // card or roadmap item should overflow its container on any mobile
    // viewport — long sub-item labels (4.A Provider adapters has ~100
    // chars, Phase 5 collapsed row has ~200 chars) must wrap cleanly.
    const overflowingNodes = await page.evaluate(() => {
      const nodes = Array.from(
        document.querySelectorAll('.roadmap-phase, .roadmap-item, .roadmap-label'),
      );
      return nodes
        .filter((n) => n.scrollWidth > n.clientWidth + 1)
        .map((n) => `${n.className}: ${n.textContent.trim().slice(0, 60)}`);
    });
    expect(overflowingNodes, 'roadmap nodes overflow their container').toHaveLength(0);

    // All seven phase groups are present in the generated roadmap, but the
    // full checklist starts collapsed so mobile users get a clear entry point.
    await expect(page.locator('.roadmap-phase')).toHaveCount(7);
    await expect(page.locator('.roadmap-details')).not.toHaveAttribute('open', '');
  });
}
