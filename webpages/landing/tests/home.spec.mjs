import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';

const release = JSON.parse(readFileSync(new URL('../src/data/release.json', import.meta.url), 'utf8'));
const landingBenchmarks = JSON.parse(readFileSync(new URL('../src/data/benchmarks.json', import.meta.url), 'utf8'));
const rootBenchmarks = JSON.parse(readFileSync(new URL('../../../benchmarks.json', import.meta.url), 'utf8'));
const logoSvg = readFileSync(new URL('../public/static/gormes-agent-logo-blue.svg', import.meta.url), 'utf8');
const releaseTag = release.tag || `v${release.version}`;
const releaseLabel = `Current scout release: ${releaseTag}`;
const releaseLabelPattern = new RegExp(escapeRegExp(releaseLabel));

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

test('homepage renders the redesigned landing', async ({ page }) => {
  expect(landingBenchmarks.binary.size_mb).toBe(rootBenchmarks.binary.size_mb);
  expect(landingBenchmarks.binary.last_measured).toBe(rootBenchmarks.binary.last_measured);
  expect(logoSvg).toContain('fill="#73cedd"');
  expect(logoSvg).toContain('shape-rendering="crispEdges"');
  expect(logoSvg).toContain('Straight block-grid GORMES-AGENT logo');
  expect(logoSvg).not.toContain('<text');
  expect(logoSvg).not.toContain('<tspan');
  expect(logoSvg).not.toContain('font-family');

  await page.goto('/');

  await expect(page).toHaveTitle('Gormes — Autonomous Porting Methodology in One Go Binary');
  await expect(page.locator('meta[name="description"]')).toHaveAttribute(
    'content',
    "Gormes is TrebuchetDynamics' autonomous-porting receipt: the 30 most-used Hermes skills running unchanged in a single 30 MB Go binary for Termux, Windows without Python, and locked-down Linux.",
  );
  await expect(page.locator('html[data-site-runtime="astro-tailwind"]')).toHaveCount(1);
  await expect(page.getByRole('heading', { name: 'Autonomous porting you can run.' })).toBeVisible();
  await expect(page.getByText("Gormes is TrebuchetDynamics' receipt for a validation-gated loop that ports Python agent systems to Go.")).toBeVisible();
  await expect(page.getByText('The v1.0 cut runs the 30 most-used Hermes skills unchanged in one 30 MB Go binary on Termux, Windows without Python, and locked-down Linux.')).toBeVisible();
  await expect(page.getByText('Hermes compatibility is the proof, not the pitch: build or install, prove the machine offline, then add provider and gateway credentials.')).toBeVisible();
  await expect(page.getByText('Scout release, honest limits.')).toBeVisible();
  await expect(page.getByText('Offline TUI, onboarding, provider turns, local SQLite memory, dashboard, and Telegram/Discord/Slack gateway paths are available now.')).toBeVisible();
  await expect(page.getByText('Release signing, voice/TTS, and full Hermes parity are still hardening.')).toBeVisible();
  await expect(page.locator('.topnav a')).toHaveText(['How it is built', 'Install', 'Trust', 'Roadmap', 'GitHub']);
  await expect(page.getByRole('img', { name: 'GORMES-AGENT' })).toHaveAttribute('src', '/static/gormes-agent-logo-blue.svg');
  await expect(page.locator('img[src="/static/go-gopher-bear-lowpoly.png"]')).toHaveCount(1);
  await expect(page.locator('.hero-ctas .btn-primary')).toHaveText('Choose an install path');
  await expect(page.locator('.hero-ctas .btn-secondary')).toHaveText('See how it is built');

  // Methodology section — the reputation-building lede.
  await expect(page.getByRole('heading', { name: 'An autonomous engineering loop ports Hermes to Go, every day.' })).toBeVisible();
  await expect(page.getByText("Gormes is the artifact TrebuchetDynamics' agentic engineering system produces.")).toBeVisible();
  await expect(page.getByText('Loop output, measured today')).toBeVisible();
  await expect(page.locator('.methodology-metric').getByText('Validated rows shipped', { exact: true })).toBeVisible();
  await expect(page.locator('.methodology-metric').getByText('770+', { exact: true })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Validation-gated commits' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'progress.json as system of record' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Hermes is the parity oracle' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Reusable porting toolkit' })).toBeVisible();
  await expect(page.getByText('Read how the loop works ->')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Who this is for' })).toBeVisible();
  await expect(page.getByText('Developers and operators who want local, inspectable agent infrastructure')).toBeVisible();
  await expect(page.getByText('Teams that require signed enterprise releases, full Hermes parity, voice/TTS, or broad channel parity today.')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Trust posture' })).toBeVisible();
  await expect(page.getByText('Source build, inspectable install.sh (Linux/macOS/WSL2), and install.ps1 (native Windows) are the three promoted scout-release paths.')).toBeVisible();
  await expect(page.getByText('install.sh and install.ps1 clone or update a managed source checkout, build gormes, and verify the command. install.sh can hand off to setup; install.ps1 leaves setup as the next explicit command.')).toBeVisible();
  await expect(page.getByText(`Current measured Linux build: ~${landingBenchmarks.binary.size_mb} MB (${landingBenchmarks.binary.last_measured})`)).toBeVisible();
  await expect(page.getByText('Progress and benchmark data sync from repo sources during every landing build.')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'What works today' })).toBeVisible();
  await expect(page.getByText('Run a local agent UI with zero runtime dependencies on the offline path')).toBeVisible();
  await expect(page.getByText('Send one-shot prompts to a provider-compatible endpoint')).toBeVisible();
  await expect(page.getByText('Validate your environment before spending tokens')).toBeVisible();
  await expect(page.getByText('Run onboard/setup flows that surface config, providers, skills, agents, and channel bindings')).toBeVisible();
  await expect(page.getByText('Operate Telegram, Discord, and Slack paths from one binary when configured')).toBeVisible();
  await expect(page.getByText('Inspect and debug local SQLite memory ("Goncho")')).toBeVisible();
  await expect(page.getByText('Browse sessions, config, skills, logs, and audits from local operator surfaces')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Gateway support status' })).toBeVisible();
  await expect(page.locator('.support-card').getByText('Runtime-ready', { exact: true })).toBeVisible();
  await expect(page.locator('.support-card').getByText('In roadmap validation', { exact: true })).toBeVisible();
  await expect(page.getByText('Telegram, Discord, and Slack.', { exact: true })).toBeVisible();
  await expect(page.getByText('WhatsApp, WeChat, Signal, Matrix, and Mattermost.', { exact: true })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Python-stack agents fail for boring reasons.' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Gormes cuts out that failure class' })).toBeVisible();
  await expect(page.getByText('The model is not usually the fragile part. Operations are:')).toBeVisible();
  await expect(page.getByText('dev, staging, and prod stop matching')).toBeVisible();
  await expect(page.getByText('virtualenvs and package wheels drift across hosts')).toBeVisible();
  await expect(page.getByText('long turns die on dropped streams')).toBeVisible();
  await expect(page.getByText('tool wiring fails after tokens are already burning')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Core runtime shipped. Production hardening and broader parity are in progress.' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Shipped in scout' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Hardening now' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Later', exact: true })).toBeVisible();
  await expect(page.getByText('Source-backed install.sh and setup handoff')).toBeVisible();
  await expect(page.getByText('Onboard/setup flows', { exact: true })).toBeVisible();
  await expect(page.getByText('Local SQLite memory and sessions')).toBeVisible();
  await expect(page.getByText('Logs, security audit, and secrets audit')).toBeVisible();
  await expect(page.getByText('Learning loop and operator feedback paths')).toBeVisible();
  await expect(page.getByText('Release checksums, signing, and package-manager lanes')).toBeVisible();
  await expect(page.getByText('Next milestone')).toBeVisible();
  await expect(page.getByText('Production-stable Go-native runtime with signed releases and broader Hermes parity')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Three install paths. One gormes command.' })).toBeVisible();
  await expect(page.getByText('Build from source when you want maximum inspection.')).toBeVisible();
  await expect(page.getByText('Use install.sh on Linux/macOS/WSL2 or install.ps1 on native Windows when you want a source-backed managed install that publishes the stable gormes command.')).toBeVisible();
  const installCommands = page.locator('#install pre code');
  const sourceBuildCommand = installCommands.nth(0);
  const installScriptCommand = installCommands.nth(1);
  const installPS1Command = installCommands.nth(2);
  await expect(sourceBuildCommand).toContainText('git clone https://github.com/TrebuchetDynamics/gormes-agent.git');
  await expect(sourceBuildCommand).toContainText('cd gormes-agent');
  await expect(sourceBuildCommand).toContainText('make build');
  await expect(sourceBuildCommand).toContainText('export PATH="$PWD/bin:$PATH"');
  await expect(sourceBuildCommand).toContainText('gormes doctor --offline');
  await expect(sourceBuildCommand).toContainText('gormes --offline');
  await expect(installScriptCommand).toContainText('curl -fsSLO https://gormes.ai/install.sh');
  await expect(installScriptCommand).toContainText('less install.sh');
  await expect(installScriptCommand).toContainText('sh install.sh');
  await expect(installScriptCommand).toContainText('gormes doctor --offline');
  await expect(installPS1Command).toContainText('https://gormes.ai/install.ps1');
  await expect(installPS1Command).toContainText('powershell -ExecutionPolicy Bypass -File');
  await expect(installPS1Command).toContainText('gormes doctor --offline');
  await expect(page.getByRole('heading', { name: 'After offline proof' })).toBeVisible();
  await expect(page.locator('#install').getByText('gormes setup provider', { exact: true })).toBeVisible();
  await expect(page.locator('#install').getByText('gormes --oneshot "hello"', { exact: true })).toBeVisible();
  await expect(page.locator('#install').getByText('gormes gateway status', { exact: true })).toBeVisible();
  await expect(page.locator('#install').getByText('./bin/gormes goncho doctor --json', { exact: true })).toHaveCount(0);
  await expect(page.locator('#install').getByText('./bin/gormes')).toHaveCount(0);
  await expect(page.locator('#install').getByText('GORMES_ENDPOINT=')).toHaveCount(0);
  await expect(page.getByText('All paths end at the same gormes command. install.sh can run gormes setup when a terminal is available; install.ps1 verifies offline doctor and then you run gormes setup explicitly.')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Single Binary Runtime' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Offline Proof', exact: true })).toBeVisible();
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
  await expect(page.getByRole('heading', { name: 'Prove the runtime locally before you ever spend a token.' })).toBeVisible();
  await expect(page.getByText('Build from source or inspect install.sh, run the offline doctor, then add credentials only after the machine has proven itself.')).toBeVisible();
  await expect(page.getByText('Requires Hermes backend at localhost:8642.')).toHaveCount(0);
  await expect(page.getByText('Run Hermes Through a Go Operator Console.')).toHaveCount(0);
  await expect(page.getByText('Why Hermes breaks in production')).toHaveCount(0);
  await expect(page.getByText('irm https://gormes.ai/install.ps1 | iex')).toHaveCount(0);
  await expect(page.getByText('curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh | sh')).toHaveCount(0);
  await expect(page.getByText('Deeper reference material lives at')).toHaveCount(0);
  await expect(page.locator('link[href="/static/site.css"]')).toHaveCount(0);
  // Copy buttons require a tiny inline clipboard script — bounded to the three install methods.
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

    await expect(page.getByRole('heading', { name: 'Autonomous porting you can run.' })).toBeVisible();
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
    // Three install methods: source build, install.sh, and install.ps1.
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
