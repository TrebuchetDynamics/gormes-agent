import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';

const release = JSON.parse(readFileSync(new URL('../src/data/release.json', import.meta.url), 'utf8'));
const releaseTag = release.tag || `v${release.version}`;
const releaseLabel = `Current scout release: ${releaseTag}`;
const releaseLabelPattern = new RegExp(escapeRegExp(releaseLabel));

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

test('homepage renders the redesigned landing', async ({ page }) => {
  await page.goto('/');

  await expect(page).toHaveTitle('Gormes — Run AI Agents From One Go Binary');
  await expect(page.locator('html[data-site-runtime="astro-tailwind"]')).toHaveCount(1);
  await expect(page.getByRole('heading', { name: 'Run AI agents from one Go binary.' })).toBeVisible();
  await expect(page.getByText('Gormes runs local agent sessions, provider turns, memory, dashboards, and chat gateways from one Go binary.')).toBeVisible();
  await expect(page.getByText('No Python runtime. No virtualenv repair. No backend service just to open the UI.')).toBeVisible();
  await expect(page.getByText('Start offline, prove the machine works, then add provider and gateway credentials.')).toBeVisible();
  await expect(page.getByText('Scout release: useful today, still early.')).toBeVisible();
  await expect(page.getByText('Offline TUI, doctor diagnostics, provider one-shots, local SQLite memory, dashboard, and runtime-ready Telegram/Discord/Slack paths are available.')).toBeVisible();
  await expect(page.getByText('Hermes parity, broad channel parity, voice/TTS, plugin/MCP support, and release signing are still hardening.')).toBeVisible();
  await expect(page.locator('.topnav a')).toHaveText(['Docs', 'Trust', 'Roadmap', 'GitHub']);
  await expect(page.getByRole('img', { name: 'GORMES-AGENT' })).toHaveAttribute('src', '/static/gormes-agent-logo-blue.svg');
  await expect(page.locator('img[src="/static/go-gopher-bear-lowpoly.png"]')).toHaveCount(1);
  await expect(page.locator('.hero-ctas .btn-primary')).toHaveText('Run offline in 3 commands');
  await expect(page.locator('.hero-ctas .btn-secondary')).toHaveText('View on GitHub');
  await expect(page.getByRole('heading', { name: 'Who this is for' })).toBeVisible();
  await expect(page.getByText('Developers and operators who want local, inspectable agent infrastructure')).toBeVisible();
  await expect(page.getByText('Teams that require signed enterprise releases, full Hermes parity, voice/TTS, or broad channel parity today.')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Trust posture' })).toBeVisible();
  await expect(page.getByText('Source build is the recommended scout-release path.')).toBeVisible();
  await expect(page.getByText('Current measured Linux build:')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'What works today' })).toBeVisible();
  await expect(page.getByText('Run a local agent UI with zero runtime dependencies on the offline path')).toBeVisible();
  await expect(page.getByText('Send one-shot prompts to a provider-compatible endpoint')).toBeVisible();
  await expect(page.getByText('Validate your environment before spending tokens')).toBeVisible();
  await expect(page.getByText('Operate Telegram, Discord, and Slack paths from one binary when configured')).toBeVisible();
  await expect(page.getByText('Inspect and debug local SQLite memory ("Goncho")')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Gateway support status' })).toBeVisible();
  await expect(page.locator('.support-card').getByText('Runtime-ready', { exact: true })).toBeVisible();
  await expect(page.locator('.support-card').getByText('Tracked, not promoted here', { exact: true })).toBeVisible();
  await expect(page.getByText('WhatsApp, WeChat, Signal, Matrix, Mattermost, and regional channels')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Python-stack agents fail for boring reasons.' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Gormes cuts out that failure class' })).toBeVisible();
  await expect(page.getByText('The model is not usually the fragile part. Operations are:')).toBeVisible();
  await expect(page.getByText('dev, staging, and prod stop matching')).toBeVisible();
  await expect(page.getByText('virtualenvs and package wheels drift across hosts')).toBeVisible();
  await expect(page.getByText('long turns die on dropped streams')).toBeVisible();
  await expect(page.getByText('tool wiring fails after tokens are already burning')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Useful today, still early.' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Shipped in scout' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Hardening now' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Later', exact: true })).toBeVisible();
  await expect(page.getByText('Local SQLite memory and sessions')).toBeVisible();
  await expect(page.getByText('Release checksums, signing, and package-manager lanes')).toBeVisible();
  await expect(page.getByText('Next milestone')).toBeVisible();
  await expect(page.getByText('Production-stable Go-native runtime with signed releases and broader Hermes parity')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Build it. Prove it offline.' })).toBeVisible();
  await expect(page.getByText('Start with the inspectable source path.')).toBeVisible();
  await expect(page.locator('#install').getByText('git clone https://github.com/TrebuchetDynamics/gormes-agent.git')).toBeVisible();
  await expect(page.locator('#install').getByText('cd gormes-agent')).toBeVisible();
  await expect(page.locator('#install').getByText('make build')).toBeVisible();
  await expect(page.locator('#install').getByText('./bin/gormes doctor --offline', { exact: true })).toBeVisible();
  await expect(page.locator('#install').getByText('./bin/gormes --offline', { exact: true })).toBeVisible();
  await expect(page.locator('#install').getByText('./bin/gormes goncho doctor --json', { exact: true })).toHaveCount(0);
  await expect(page.locator('#install').getByText('GORMES_ENDPOINT=')).toHaveCount(0);
  await expect(page.locator('#install').getByText('curl -fsSLO https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh')).toHaveCount(0);
  await expect(page.getByText('Provider setup, gateway setup, and convenience installers come after the offline proof.')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Single Static Binary' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Offline Proof' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Built-In Doctor' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Provider Turns' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Local SQLite Memory' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Visible Limits' })).toBeVisible();
  await expect(page.getByText('Full Hermes parity, broad channel parity, voice/TTS, MCP/plugin parity')).toBeVisible();
  await expect(page.getByText('Read the install docs ->')).toBeVisible();
  // New conversion sections
  await expect(page.locator('.proof-item').getByText('Static Go binary', { exact: true })).toBeVisible();
  await expect(page.locator('.proof-item').getByText(releaseLabel, { exact: true })).toBeVisible();
  await expect(page.locator('.footer-left').getByText(releaseLabelPattern)).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Explore' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Start offline. Add credentials later.' })).toBeVisible();
  await expect(page.getByText('Requires Hermes backend at localhost:8642.')).toHaveCount(0);
  await expect(page.getByText('Run Hermes Through a Go Operator Console.')).toHaveCount(0);
  await expect(page.getByText('Why Hermes breaks in production')).toHaveCount(0);
  await expect(page.getByText('irm https://gormes.ai/install.ps1 | iex')).toHaveCount(0);
  await expect(page.getByText('curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh | sh')).toHaveCount(0);
  await expect(page.getByText('Deeper reference material lives at')).toHaveCount(0);
  await expect(page.locator('link[href="/static/site.css"]')).toHaveCount(0);
  // Copy buttons require a tiny inline clipboard script — bounded to the three-step offline proof.
  await expect(page.locator('button.copy-btn')).toHaveCount(3);
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

    await expect(page.getByRole('heading', { name: 'Run AI agents from one Go binary.' })).toBeVisible();
    await expect(page.getByText('git clone https://github.com/TrebuchetDynamics/gormes-agent.git')).toBeVisible();

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
    // Three install steps: source build, offline TUI, doctor.
    const copyButtons = page.locator('button.copy-btn');
    await expect(copyButtons).toHaveCount(3);
    for (let i = 0; i < 3; i++) {
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

    // The roadmap is deliberately compact on the landing page. Detailed
    // phase inventory lives in docs, so these summary cards must wrap cleanly.
    const overflowingNodes = await page.evaluate(() => {
      const nodes = Array.from(
        document.querySelectorAll('.roadmap-summary-block, .roadmap-summary-heading, .roadmap-summary-list'),
      );
      return nodes
        .filter((n) => n.scrollWidth > n.clientWidth + 1)
        .map((n) => `${n.className}: ${n.textContent.trim().slice(0, 60)}`);
    });
    expect(overflowingNodes, 'roadmap nodes overflow their container').toHaveLength(0);

    await expect(page.locator('.roadmap-summary-block')).toHaveCount(4);
    await expect(page.locator('.roadmap-details')).toHaveCount(0);
    await expect(page.locator('.roadmap-phase')).toHaveCount(0);
  });
}
