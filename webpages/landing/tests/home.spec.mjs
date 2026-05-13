import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';

const release = JSON.parse(readFileSync(new URL('../src/data/release.json', import.meta.url), 'utf8'));
const landingBenchmarks = JSON.parse(readFileSync(new URL('../src/data/benchmarks.json', import.meta.url), 'utf8'));
const rootBenchmarks = JSON.parse(readFileSync(new URL('../../../benchmarks.json', import.meta.url), 'utf8'));
const logoSvg = readFileSync(new URL('../public/static/gormes-agent-logo-blue.svg', import.meta.url), 'utf8');
const releaseTag = release.tag || `v${release.version}`;
const releaseDateAlias = release.date_alias || '';
const releaseLabel = releaseDateAlias
  ? `Current scout release: ${releaseTag} (${releaseDateAlias})`
  : `Current scout release: ${releaseTag}`;
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

  await expect(page).toHaveTitle('Gormes — Run AI agents from a single binary');
  await expect(page.locator('html[data-site-runtime="astro-tailwind"]')).toHaveCount(1);

  // Hero
  await expect(page.getByRole('heading', { name: 'Run AI agents from a single binary.' })).toBeVisible();
  await expect(page.getByText('One static binary. No Python runtime. No Docker daemon. No dependency drift.')).toBeVisible();
  await expect(page.getByText('Scout release.')).toBeVisible();
  await expect(page.getByText('Offline TUI, onboarding, provider turns, local SQLite memory, dashboard, and Telegram/Discord/Slack gateway paths are available now.')).toBeVisible();
  await expect(page.getByText('Release signing, voice/TTS, and full Hermes parity are still hardening.')).toBeVisible();

  // Nav — 3 items, clean
  await expect(page.locator('.topnav a')).toHaveText(['Docs', 'Install', 'GitHub']);

  await expect(page.getByRole('img', { name: 'GORMES-AGENT' })).toHaveAttribute('src', '/static/gormes-agent-logo-blue.svg');
  await expect(page.locator('img[src="/static/go-gopher-bear-lowpoly.png"]')).toHaveCount(1);
  await expect(page.locator('.hero-ctas .btn-primary')).toHaveText('Install');
  await expect(page.locator('.hero-ctas .btn-secondary')).toHaveText('See features');

  // Proof strip — 4 items, no jargon
  await expect(page.locator('.proof-item-pop').getByText('30 Hermes skills', { exact: true })).toBeVisible();
  await expect(page.locator('.proof-item-pop').getByText('1 Go binary', { exact: true })).toBeVisible();
  await expect(page.locator('.proof-item-pop').getByText(`${landingBenchmarks.code.test_count.toLocaleString()} tests`, { exact: true })).toBeVisible();
  await expect(page.locator('.proof-item').getByText('MIT License', { exact: true })).toBeVisible();

  // No stats bar
  await expect(page.locator('.stats-grid')).toHaveCount(0);

  // Why section — pain points
  await expect(page.getByRole('heading', { name: 'Python-stack agents fail for boring reasons.' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Gormes cuts out that failure class' })).toBeVisible();
  await expect(page.getByText('The model is not usually the fragile part. Operations are:')).toBeVisible();
  await expect(page.getByText('dev, staging, and prod stop matching')).toBeVisible();
  await expect(page.getByText('virtualenvs and package wheels drift across hosts')).toBeVisible();
  await expect(page.getByText('long turns die on dropped streams')).toBeVisible();
  await expect(page.getByText('tool wiring fails after tokens are already burning')).toBeVisible();

  // Feature cards
  await expect(page.getByRole('heading', { name: 'Single Binary Runtime' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Offline Proof', exact: true })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Built-In Doctor' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Provider Turns' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Local SQLite Memory' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Visible Limits' })).toBeVisible();
  await expect(page.getByText('Full Hermes parity, broad channel parity, voice/TTS, MCP/plugin parity')).toBeVisible();

  // Built-for — grouped into 4 categories
  await expect(page.getByRole('heading', { name: 'What works today' })).toBeVisible();
  await expect(page.locator('.built-for-group')).toHaveCount(4);
  await expect(page.getByText('Runtime', { exact: true })).toBeVisible();
  await expect(page.getByText('Memory & State', { exact: true })).toBeVisible();
  await expect(page.getByText('Gateways', { exact: true })).toBeVisible();
  await expect(page.getByText('Operations', { exact: true })).toBeVisible();
  await expect(page.getByText('Offline TUI with zero dependencies')).toBeVisible();
  await expect(page.getByText('Local SQLite sessions ("Goncho")')).toBeVisible();
  await expect(page.getByText('Telegram bot integration')).toBeVisible();
  await expect(page.getByText('Local dashboard at 127.0.0.1:43827')).toBeVisible();

  // Gateway support
  await expect(page.getByRole('heading', { name: 'Gateway support status' })).toBeVisible();
  await expect(page.locator('.support-card').getByText('Runtime-ready', { exact: true })).toBeVisible();
  await expect(page.locator('.support-card').getByText('In roadmap validation', { exact: true })).toBeVisible();
  await expect(page.getByText('Telegram, Discord, and Slack.', { exact: true })).toBeVisible();
  await expect(page.getByText('WhatsApp, WeChat, Signal, Matrix, and Mattermost.', { exact: true })).toBeVisible();

  // Install section
  await expect(page.getByRole('heading', { name: 'Two install paths. One gormes command.' })).toBeVisible();
  await expect(page.getByText('Build from source when you want maximum inspection.')).toBeVisible();
  const installCommands = page.locator('#install pre code');
  const sourceBuildCommand = installCommands.nth(0);
  const installScriptCommand = installCommands.nth(1);
  await expect(sourceBuildCommand).toContainText('git clone https://github.com/TrebuchetDynamics/gormes-agent.git');
  await expect(sourceBuildCommand).toContainText('cd gormes-agent');
  await expect(sourceBuildCommand).toContainText('make build');
  await expect(sourceBuildCommand).toContainText('export PATH="$PWD/bin:$PATH"');
  await expect(sourceBuildCommand).toContainText('gormes doctor --offline');
  await expect(sourceBuildCommand).toContainText('gormes --offline');
  await expect(installScriptCommand).toContainText('curl -fsSLO https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh');
  await expect(installScriptCommand).toContainText('less install.sh');
  await expect(installScriptCommand).toContainText('sh install.sh');
  await expect(installScriptCommand).toContainText('gormes doctor --offline');
  await expect(page.getByRole('heading', { name: 'After offline proof' })).toBeVisible();
  await expect(page.locator('#install').getByText('gormes setup provider', { exact: true })).toBeVisible();
  await expect(page.locator('#install').getByText('gormes --oneshot "hello"', { exact: true })).toBeVisible();
  await expect(page.locator('#install').getByText('gormes gateway status', { exact: true })).toBeVisible();
  await expect(page.locator('#install').getByText('./bin/gormes goncho doctor --json', { exact: true })).toHaveCount(0);
  await expect(page.locator('#install').getByText('./bin/gormes')).toHaveCount(0);
  await expect(page.locator('#install').getByText('GORMES_ENDPOINT=')).toHaveCount(0);
  await expect(page.getByText('Both paths end at the same gormes command. install.sh also runs gormes setup when a terminal is available.')).toBeVisible();

  // Trust section
  await expect(page.getByRole('heading', { name: 'Who this is for' })).toBeVisible();
  await expect(page.getByText('Developers and operators who need reliable, local agent infrastructure that survives restarts, bad networks, and dependency drift.')).toBeVisible();
  await expect(page.getByText('Teams that require signed enterprise releases, full Hermes parity, voice/TTS, or broad channel parity today.')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Trust posture' })).toBeVisible();
  await expect(page.getByText('Offline doctor runs before any token spend.')).toBeVisible();
  await expect(page.getByText('Secrets stay local under ~/.gormes.')).toBeVisible();
  await expect(page.getByText('Source-backed install.sh you can inspect before running.')).toBeVisible();
  await expect(page.getByText('Every commit passes go test, progress validate, and git diff --check.')).toBeVisible();
  await expect(page.getByText('Tagged releases with SHA-256 checksums.')).toBeVisible();

  // Methodology — demoted, shorter
  await expect(page.getByRole('heading', { name: "Systematic porting with full test coverage." })).toBeVisible();
  await expect(page.getByText("Every generated change passes tests, parity checks, and repo validation before landing. Hermes is the parity oracle; engineering rigor is the differentiator.")).toBeVisible();
  await expect(page.getByText('Loop output, measured today')).toBeVisible();
  await expect(page.locator('.methodology-metric').getByText('Validated rows shipped', { exact: true })).toBeVisible();
  await expect(page.locator('.methodology-metric').getByText('770+', { exact: true })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Validation-gated commits' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Hermes is the parity oracle' })).toBeVisible();
  // Only 2 pillars now
  await expect(page.locator('.methodology-pillar')).toHaveCount(2);
  await expect(page.getByText('Read how the loop works ->')).toBeVisible();

  // Roadmap
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

  // Explore + Final CTA
  await expect(page.getByRole('heading', { name: 'Explore' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Prove the runtime locally before you ever spend a token.' })).toBeVisible();
  await expect(page.getByText('Build from source or inspect install.sh, run the offline doctor, then add credentials only after the machine has proven itself.')).toBeVisible();

  // Footer release label
  await expect(page.locator('.footer-left').getByText(releaseLabelPattern)).toBeVisible();

  // Footer nav — 3 items
  await expect(page.locator('.footer-nav a')).toHaveCount(3);

  // Stale content regression guards
  await expect(page.getByText('Requires Hermes backend at localhost:8642.')).toHaveCount(0);
  await expect(page.getByText('Run Hermes Through a Go Operator Console.')).toHaveCount(0);
  await expect(page.getByText('Why Hermes breaks in production')).toHaveCount(0);
  await expect(page.getByText('irm https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.ps1 | iex')).toHaveCount(0);
  await expect(page.getByText('curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh | sh')).toHaveCount(0);
  await expect(page.getByText('Deeper reference material lives at')).toHaveCount(0);
  await expect(page.locator('link[href="/static/site.css"]')).toHaveCount(0);

  // Copy buttons: exactly 2 install methods
  await expect(page.locator('button.copy-btn')).toHaveCount(2);
});

test('built-with page lists truthful deployments and submission template', async ({ page }) => {
  await page.goto('/built-with');

  await expect(page).toHaveTitle('Built with Gormes — Real Deployments and Self-Hosted Uses');
  await expect(page.getByRole('heading', { name: 'Built with Gormes' })).toBeVisible();
  await expect(page.getByText('Real deployments only. No fabricated customer logos, no placeholder companies.')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'TrebuchetDynamics operator loop' })).toBeVisible();
  await expect(page.getByText('TrebuchetDynamics', { exact: true })).toBeVisible();
  await expect(page.getByText('Self-hosted operator deployment')).toBeVisible();
  await expect(page.getByText('Runs the autonomous Hermes-to-Go porting loop against the public Gormes repository.')).toBeVisible();
  await expect(page.getByText('development branch progress.json, Go test gates, GitHub Actions release workflow')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Submit a deployment' })).toBeVisible();
  await expect(page.getByText('Open a pull request that adds one entry to webpages/landing/src/data/builtWith.js.')).toBeVisible();
  await expect(page.getByText('Required fields: name, href, operator, status, summary, proof, stack, submissionContact.')).toBeVisible();
  await expect(page.locator('main')).not.toContainText('Acme');
  await expect(page.locator('main')).not.toContainText('Example customer');
});

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

    await expect(page.getByRole('heading', { name: 'Run AI agents from a single binary.' })).toBeVisible();
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

    const pageOverflow = await page.evaluate(() =>
      document.documentElement.scrollWidth > window.innerWidth,
    );
    expect(pageOverflow, `page body overflows at ${vp.width}px`).toBeFalsy();

    const copyButtons = page.locator('button.copy-btn');
    await expect(copyButtons).toHaveCount(2);
    for (let i = 0; i < 2; i++) {
      const btn = copyButtons.nth(i);
      await expect(btn).toBeVisible();
      const box = await btn.boundingBox();
      expect(box, `copy button ${i} has no bounding box`).not.toBeNull();
      expect(box.height, `copy button ${i} too short at ${vp.width}px`).toBeGreaterThanOrEqual(28);
      expect(box.width, `copy button ${i} too narrow at ${vp.width}px`).toBeGreaterThanOrEqual(28);
    }

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
