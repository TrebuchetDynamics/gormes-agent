import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';

const landingBenchmarks = JSON.parse(readFileSync(new URL('../src/data/benchmarks.json', import.meta.url), 'utf8'));
const rootBenchmarks = JSON.parse(readFileSync(new URL('../../../benchmarks.json', import.meta.url), 'utf8'));
const logoSvg = readFileSync(new URL('../public/static/gormes-agent-logo-blue.svg', import.meta.url), 'utf8');

test('homepage sells the short buyer-focused landing', async ({ page }) => {
  expect(landingBenchmarks.binary.size_mb).toBe(rootBenchmarks.binary.size_mb);
  expect(landingBenchmarks.binary.last_measured).toBe(rootBenchmarks.binary.last_measured);
  expect(logoSvg).toContain('fill="#73cedd"');
  expect(logoSvg).toContain('shape-rendering="crispEdges"');
  expect(logoSvg).toContain('Straight block-grid GORMES-AGENT logo');
  expect(logoSvg).not.toContain('<text');
  expect(logoSvg).not.toContain('<tspan');
  expect(logoSvg).not.toContain('font-family');

  await page.goto('/');

  await expect(page).toHaveTitle('Gormes — Go-Native AI Agent Runtime Without Python or Docker');
  await expect(page.locator('html[data-site-runtime="astro-tailwind"]')).toHaveCount(1);
  await expect(page.locator('main > section')).toHaveCount(7);

  await expect(page.locator('meta[name="description"]')).toHaveAttribute(
    'content',
    'Gormes runs AI agents from one static Go binary with offline diagnostics, SQLite memory, provider chat, skills, dashboard, and Telegram/Discord/Slack gateways.',
  );
  await expect(page.locator('link[rel="canonical"]')).toHaveAttribute('href', 'https://gormes.ai/');
  await expect(page.locator('link[rel="sitemap"]')).toHaveAttribute('href', '/sitemap.xml');
  await expect(page.locator('meta[name="keywords"]')).toHaveAttribute('content', /Go AI agent runtime/);
  await expect(page.locator('meta[name="keywords"]')).toHaveAttribute('content', /Hermes agent alternative/);
  await expect(page.locator('meta[name="keywords"]')).toHaveAttribute('content', /Termux AI agent/);

  const schema = await page.locator('script[type="application/ld+json"]').textContent();
  expect(schema).toContain('"@type":"SoftwareApplication"');
  expect(schema).toContain('"name":"Gormes"');
  expect(schema).toContain('"@type":"Organization"');
  expect(schema).toContain('"name":"TrebuchetDynamics"');

  await expect(page.getByRole('img', { name: 'Gormes', exact: true })).toHaveAttribute('src', '/static/gormes-agent-logo-blue.svg');
  const primaryNav = page.locator('nav[aria-label="Primary"]');
  await expect(primaryNav.getByRole('link', { name: 'Install' })).toHaveAttribute('href', '/install');
  await expect(primaryNav.getByRole('link', { name: 'Docs' })).toHaveAttribute('href', '/docs');
  await expect(primaryNav.getByRole('link', { name: 'Roadmap' })).toHaveAttribute('href', '/roadmap');
  await expect(page.getByRole('img', { name: 'Gormes Go-native AI agent runtime mascot' })).toHaveAttribute('src', '/static/go-gopher-bear-lowpoly.png');

  await expect(page.getByRole('heading', { name: 'Go-native AI agent runtime without Python or Docker.' })).toBeVisible();
  await expect(page.getByText('Run local and server-side AI agents from one static binary')).toBeVisible();
  await expect(page.getByText('with Hermes-style skills, offline diagnostics, SQLite memory')).toBeVisible();
  await expect(page.locator('.hero-ctas .btn-primary')).toHaveText('Install Gormes');
  await expect(page.locator('.hero-ctas .btn-secondary')).toHaveText('View GitHub');
  await expect(page.locator('.proof-item-pop').getByText('Static Go binary', { exact: true })).toBeVisible();
  await expect(page.locator('.proof-item-pop').getByText('No venv drift', { exact: true })).toBeVisible();
  await expect(page.locator('.proof-item').getByText('Offline doctor', { exact: true })).toBeVisible();
  await expect(page.locator('.proof-item').getByText('Termux-ready', { exact: true })).toBeVisible();

  await expect(page.getByRole('heading', { name: 'Python agents break for boring reasons.' })).toBeVisible();
  await expect(page.getByText('Venvs drift, installs fail, streams drop, tools miswire, and servers rot.')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'No venv drift' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Offline doctor first' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Local SQLite memory' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'One gateway process' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Static Go binary' })).toBeVisible();

  await expect(page.getByRole('heading', { name: 'What works today' })).toBeVisible();
  await expect(page.locator('.works-card')).toHaveCount(5);
  await expect(page.getByText('CLI and offline TUI', { exact: true })).toBeVisible();
  await expect(page.getByText('Provider-backed chat', { exact: true })).toBeVisible();
  await expect(page.getByText('SQLite memory and sessions', { exact: true })).toBeVisible();
  await expect(page.getByText('Local dashboard', { exact: true })).toBeVisible();
  await expect(page.getByText('Telegram, Discord, and Slack gateways', { exact: true })).toBeVisible();

  await expect(page.getByRole('heading', { name: 'Install Gormes' })).toBeVisible();
  const installCommand = page.locator('#install pre code');
  await expect(installCommand).toContainText('curl -fsSL https://github.com/TrebuchetDynamics/gormes-agent/releases/latest/download/install.sh | sh');
  await expect(installCommand).toContainText('gormes doctor --offline');
  await expect(installCommand).toContainText('gormes setup');
  await expect(installCommand).toContainText('gormes chat');
  await expect(installCommand).not.toContainText('raw.githubusercontent.com');
  await expect(page.locator('button.copy-btn')).toHaveCount(1);

  await expect(page.getByRole('heading', { name: 'Evidence, not a sidecar stack' })).toBeVisible();
  await expect(page.locator('.proof-card')).toHaveCount(4);
  await expect(page.getByText(`${landingBenchmarks.code.test_count.toLocaleString()}`, { exact: true })).toBeVisible();
  await expect(page.getByText(`~${landingBenchmarks.binary.size_mb} MB`, { exact: true })).toBeVisible();
  await expect(page.getByText('offline', { exact: true })).toBeVisible();
  await expect(page.getByText('SHA-256 + SBOM', { exact: true })).toBeVisible();

  await expect(page.getByRole('heading', { name: 'Available now. Expanding next.' })).toBeVisible();
  await expect(page.getByText('More Hermes compatibility', { exact: true })).toBeVisible();
  await expect(page.getByText('Voice/TTS', { exact: true })).toBeVisible();
  await expect(page.getByText('MCP/plugin support', { exact: true })).toBeVisible();
  await expect(page.getByText('Package-manager installs', { exact: true })).toBeVisible();
  await expect(page.getByText('More gateways', { exact: true })).toBeVisible();
  await expect(page.getByText('Full Hermes parity', { exact: true })).toHaveCount(0);

  await expect(page.getByRole('heading', { name: 'Run the offline doctor before you spend a token.' })).toBeVisible();
  await expect(page.getByText('Install Gormes, prove the runtime locally, then configure a provider and gateway when the machine is ready.')).toBeVisible();
  await expect(page.locator('.final-cta-actions .btn-primary')).toHaveText('Run offline doctor');

  await expect(page.locator('#trust')).toHaveCount(0);
  await expect(page.locator('#methodology')).toHaveCount(0);
  await expect(page.locator('#explore')).toHaveCount(0);
  await expect(page.getByText('770+')).toHaveCount(0);
  await expect(page.getByText('Validated rows shipped')).toHaveCount(0);
  await expect(page.getByText('Runtime RSS')).toHaveCount(0);
  await expect(page.getByText('Scout release.')).toHaveCount(0);
  await expect(page.getByText('Teams that require signed enterprise releases')).toHaveCount(0);
  await expect(page.getByText(/https:\/\/gormes[.]ai\/install[.]sh/)).toHaveCount(0);
});

test('static SEO helper files are shipped', async ({ page }) => {
  const sitemap = await page.request.get('/sitemap.xml');
  expect(sitemap.status()).toBe(200);
  expect(await sitemap.text()).toContain('https://gormes.ai/');

  const llms = await page.request.get('/llms.txt');
  expect(llms.status()).toBe(200);
  const llmsText = await llms.text();
  expect(llmsText).toContain('Go-native AI agent runtime');
  expect(llmsText).toContain('Hermes agent alternative');
  expect(llmsText).toContain('Termux AI agent runtime');

  const redirects = readFileSync(new URL('../public/_redirects', import.meta.url), 'utf8');
  expect(redirects).toContain('/install https://docs.gormes.ai/install/ 302');
  expect(redirects).toContain('/docs https://docs.gormes.ai/ 302');
  expect(redirects).toContain('/roadmap https://docs.gormes.ai/reference/status-readiness/ 302');
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
  test(`mobile (${vp.label} ${vp.width}x${vp.height}) has no horizontal overflow`, async ({ page }) => {
    await page.setViewportSize({ width: vp.width, height: vp.height });
    await page.goto('/');

    await expect(page.getByRole('heading', { name: 'Go-native AI agent runtime without Python or Docker.' })).toBeVisible();
    await expect(page.getByText('curl -fsSL https://github.com/TrebuchetDynamics/gormes-agent/releases/latest/download/install.sh')).toBeVisible();

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
    await expect(copyButtons).toHaveCount(1);
    const box = await copyButtons.first().boundingBox();
    expect(box, 'copy button has no bounding box').not.toBeNull();
    expect(box.height, `copy button too short at ${vp.width}px`).toBeGreaterThanOrEqual(28);
    expect(box.width, `copy button too narrow at ${vp.width}px`).toBeGreaterThanOrEqual(28);

    const overflowingNodes = await page.evaluate(() => {
      const nodes = Array.from(
        document.querySelectorAll('.roadmap-summary-block, .roadmap-summary-heading, .roadmap-summary-list'),
      );
      return nodes
        .filter((n) => n.scrollWidth > n.clientWidth + 1)
        .map((n) => `${n.className}: ${n.textContent.trim().slice(0, 60)}`);
    });
    expect(overflowingNodes, 'roadmap nodes overflow their container').toHaveLength(0);

    await expect(page.locator('.roadmap-summary-block')).toHaveCount(2);
    await expect(page.locator('.roadmap-details')).toHaveCount(0);
    await expect(page.locator('.roadmap-phase')).toHaveCount(0);
  });
}
