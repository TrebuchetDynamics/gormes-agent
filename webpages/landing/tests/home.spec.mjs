import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';

import { expectMainHeading, expectNoHorizontalOverflow, visitPage } from '../../shared/playwright/playwright-helpers.mjs';

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

  await visitPage(page, '/');

  await expect(page).toHaveTitle('Gormes — Run AI Agents Anywhere from One Go Binary');
  await expect(page.locator('html[data-site-runtime="astro-tailwind"]')).toHaveCount(1);
  await expect(page.locator('main#content[tabindex="-1"] > section')).toHaveCount(3);
  await expect(page.getByRole('link', { name: 'Skip to content' })).toHaveAttribute('href', '#content');

  await expect(page.locator('meta[name="description"]')).toHaveAttribute(
    'content',
    'Gormes is a Go-native AI agent runtime for local or server-side agents with offline diagnostics, SQLite memory, provider chat, dashboards, trusted gateways, and experimental Navivox phone pairing.',
  );
  await expect(page.locator('link[rel="canonical"]')).toHaveAttribute('href', 'https://gormes.ai/');
  await expect(page.locator('link[rel="sitemap"]')).toHaveAttribute('href', '/sitemap.xml');
  await expect(page.locator('meta[name="keywords"]')).toHaveAttribute('content', /Go AI agent runtime/);
  await expect(page.locator('meta[name="keywords"]')).toHaveAttribute('content', /local AI agent runtime/);
  await expect(page.locator('meta[name="keywords"]')).toHaveAttribute('content', /single binary AI agent/);
  await expect(page.locator('meta[name="keywords"]')).toHaveAttribute('content', /AI gateway runtime/);

  const schema = await page.locator('script[type="application/ld+json"]').textContent();
  expect(schema).toContain('"@type":"SoftwareApplication"');
  expect(schema).toContain('"name":"Gormes"');
  expect(schema).toContain('"@type":"Organization"');
  expect(schema).toContain('"name":"TrebuchetDynamics"');

  await expect(page.getByRole('img', { name: 'Gormes', exact: true })).toHaveAttribute('src', '/static/gormes-agent-logo-blue.svg');
  await expect(page.getByRole('img', { name: 'Gormes', exact: true })).toHaveAttribute('width', '1200');
  await expect(page.getByRole('img', { name: 'Gormes', exact: true })).toHaveAttribute('height', '150');
  const primaryNav = page.locator('nav[aria-label="Primary"]');
  await expect(primaryNav.locator('ul.topnav')).toHaveAttribute('role', 'list');
  await expect(primaryNav.getByRole('link', { name: 'Install' })).toHaveAttribute('href', '#install');
  await expect(primaryNav.getByRole('link', { name: 'Docs' })).toHaveAttribute('href', '/docs');
  await expect(page.getByRole('img', { name: 'Gormes Go-native AI agent runtime mascot' })).toHaveAttribute('src', '/static/go-gopher-bear-lowpoly.png');
  await expect(page.getByRole('img', { name: 'Gormes Go-native AI agent runtime mascot' })).toHaveAttribute('width', '800');
  await expect(page.getByRole('img', { name: 'Gormes Go-native AI agent runtime mascot' })).toHaveAttribute('height', '800');
  await expect(page.getByRole('img', { name: 'Gormes Go-native AI agent runtime mascot' })).toHaveAttribute('fetchpriority', 'low');

  await expectMainHeading(page, 'Run AI agents anywhere.');
  await expect(page.getByText('Gormes runs local or server-side agents with chat')).toBeVisible();
  await expect(page.getByText('no Python, Docker, or venv drift')).toBeVisible();
  await expect(page.getByLabel('Gormes operator quick path')).toHaveCount(0);
  await expect(page.locator('.hero-ctas .btn-primary')).toHaveText('Install Gormes');
  await expect(page.locator('.hero-ctas .btn-secondary')).toHaveText('Read docs');
  await expect(page.getByRole('link', { name: 'View GitHub' })).toHaveAttribute('href', /github\.com\/TrebuchetDynamics\/gormes-agent/);
  await expect(page.locator('.proof-item-pop').getByText(`~${landingBenchmarks.binary.size_mb} MB static binary`, { exact: true })).toBeVisible();
  await expect(page.locator('.proof-item').getByText('Linux · macOS · Windows · Android', { exact: true })).toBeVisible();
  await expect(page.locator('.proof-item').getByText('MIT', { exact: true })).toBeVisible();

  await expect(page.getByRole('heading', { name: 'Less setup. More runtime.' })).toBeVisible();
  await expect(page.getByText('A small runtime you can inspect, move, and run without a language toolchain.')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'No Python drift' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Offline doctor' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'SQLite memory' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Trusted gateways' })).toHaveCount(0);
  await expect(page.locator('.why-card')).toHaveCount(3);
  await expect(page.getByRole('heading', { name: 'Who is it for?' })).toHaveCount(0);
  await expect(page.getByRole('heading', { name: 'Where can Gormes run?' })).toHaveCount(0);

  await expect(page.getByRole('heading', { name: 'Run it' })).toBeVisible();
  const installCommand = page.locator('#install pre code');
  await expect(installCommand).toContainText('curl -fsSL https://gormes.ai/install.sh | bash');
  await expect(installCommand).toContainText('gormes version');
  await expect(installCommand).toContainText('gormes doctor --offline');
  await expect(installCommand).toContainText('gormes setup');
  await expect(installCommand).toContainText('gormes chat');
  await expect(installCommand).not.toContainText('raw.githubusercontent.com');
  await expect(page.locator('button.copy-btn')).toHaveCount(1);
  await expect(page.getByText('Windows, source builds, Termux details, and advanced installer flags')).toBeVisible();
  await expect(page.locator('main')).not.toContainText('v0.2.20 executable-argument bug');

  await expect(page.getByRole('heading', { name: 'Real runtime, not a mockup' })).toHaveCount(0);
  await expect(page.getByRole('heading', { name: 'Use cases' })).toHaveCount(0);
  await expect(page.getByRole('heading', { name: 'Proof you can verify' })).toHaveCount(0);
  await expect(page.locator('.use-case-card')).toHaveCount(0);
  await expect(page.locator('.proof-card')).toHaveCount(0);

  await expect(page.locator('#final-cta')).toHaveCount(0);

  await expect(page.locator('#trust')).toHaveCount(0);
  await expect(page.locator('#methodology')).toHaveCount(0);
  await expect(page.locator('#explore')).toHaveCount(0);
  await expect(page.getByText('770+')).toHaveCount(0);
  await expect(page.getByText('Validated rows shipped')).toHaveCount(0);
  await expect(page.getByText('Runtime RSS')).toHaveCount(0);
  await expect(page.getByText('Scout release.')).toHaveCount(0);
  await expect(page.getByText('Teams that require signed enterprise releases')).toHaveCount(0);
});

test('install copy interaction copies the exact release-first command', async ({ page }) => {
  const copied = [];
  await page.addInitScript(() => {
    window.__copiedText = [];
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: {
        writeText: async (text) => {
          window.__copiedText.push(text);
        },
      },
    });
  });

  await visitPage(page, '/');
  const install = page.locator('#install');
  const command = await install.locator('pre code').innerText();
  expect(command).toContain('https://gormes.ai/install.sh');
  expect(command).toContain('gormes version');
  expect(command).toContain('gormes doctor --offline');
  expect(command).toContain('gormes setup');
  expect(command).toContain('gormes chat');
  expect(command).not.toContain('raw.githubusercontent.com');
  expect(command).not.toContain('github.com/TrebuchetDynamics/gormes-agent/releases/latest/download/install.sh');

  await install.getByRole('button', { name: 'Copy install command' }).click();
  await expect(install.locator('.copy-label')).toHaveText('Copied');
  copied.push(...await page.evaluate(() => window.__copiedText));
  expect(copied).toEqual([command]);

  await expect(page.locator('.hero-ctas .btn-primary')).toHaveAttribute('href', '#install');
  await expect(page.locator('.final-cta-actions')).toHaveCount(0);
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
  expect(llmsText).toContain('Local AI agent runtime');
  expect(llmsText).toContain('AI gateway runtime');
  expect(llmsText).toContain('Experimental Navivox mobile control app');

  const redirects = readFileSync(new URL('../public/_redirects', import.meta.url), 'utf8');
  expect(redirects).toContain('/install https://docs.gormes.ai/install/ 302');
  expect(redirects).toContain('/docs https://docs.gormes.ai/ 302');
  expect(redirects).toContain('/roadmap https://docs.gormes.ai/reference/status-readiness/ 302');
});

test('primary landing CTAs navigate without leaving stale sections visible', async ({ page }) => {
  await visitPage(page, '/');

  await page.locator('.hero-ctas .btn-primary').click();
  await expect(page).toHaveURL(/#install$/);
  await expect(page.getByRole('heading', { name: 'Run it' })).toBeVisible();

  await expect(page.locator('#install pre code')).toContainText('gormes doctor --offline');

  await expect(page.locator('.hero-ctas .btn-secondary')).toHaveAttribute('href', '/docs');
  await expect(page.locator('#trust')).toHaveCount(0);
  await expect(page.locator('#methodology')).toHaveCount(0);
  await expect(page.locator('#explore')).toHaveCount(0);
});

test('built-with page lists truthful deployments and submission template', async ({ page }) => {
  await visitPage(page, '/built-with');

  await expect(page).toHaveTitle('Built with Gormes — Real Deployments and Self-Hosted Uses');
  await expect(page.locator('link[rel="canonical"]')).toHaveAttribute('href', 'https://gormes.ai/built-with');
  await expect(page.locator('meta[property="og:url"]')).toHaveAttribute('content', 'https://gormes.ai/built-with');
  await expect(page.locator('meta[property="og:image:width"]')).toHaveAttribute('content', '1200');
  await expectMainHeading(page, 'Built with Gormes');
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
    await visitPage(page, '/', vp);

    await expectMainHeading(page, 'Run AI agents anywhere.');
    await expect(page.locator('#install pre code')).toContainText('curl -fsSL https://gormes.ai/install.sh');

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

    await expectNoHorizontalOverflow(page, `page body overflows at ${vp.width}px`);

    const copyButtons = page.locator('button.copy-btn');
    await expect(copyButtons).toHaveCount(1);
    const box = await copyButtons.first().boundingBox();
    expect(box, 'copy button has no bounding box').not.toBeNull();
    expect(box.height, `copy button too short at ${vp.width}px`).toBeGreaterThanOrEqual(28);
    expect(box.width, `copy button too narrow at ${vp.width}px`).toBeGreaterThanOrEqual(28);

    await expect(page.locator('#roadmap')).toHaveCount(0);
    await expect(page.locator('.roadmap-details')).toHaveCount(0);
    await expect(page.locator('.roadmap-phase')).toHaveCount(0);
  });
}
