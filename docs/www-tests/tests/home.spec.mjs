import { test, expect } from '@playwright/test';

test('docs home hero, quickstart, and user-first cards render', async ({ page }) => {
  await page.goto('/');

  await expect(page).toHaveTitle(/Gormes Docs/);
  // Hero
  await expect(page.locator('.docs-home-hero h1')).toBeVisible();
  await expect(page.locator('.docs-home-hero .kicker')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Run agents from one Go-native runtime.' })).toBeVisible();
  await expect(page.getByText('native terminal UI, offline diagnostics, provider-backed turns')).toBeVisible();

  // OpenClaw-inspired first choices: get started, run diagnostics, operate gateways.
  const actions = page.locator('.docs-home-action');
  await expect(actions).toHaveCount(3);
  await expect(actions.nth(0)).toContainText('Get Started');
  await expect(actions.nth(1)).toContainText('Run Doctor');
  await expect(actions.nth(2)).toContainText('Operate Gateways');

  // What/how section
  await expect(page.getByRole('heading', { name: 'A self-hosted agent runtime for machines that need to keep working.' })).toBeVisible();
  await expect(page.locator('.docs-home-flow-node--core')).toHaveText('Gormes Runtime');

  // Quickstart strip
  const qs = page.locator('.docs-home-quickstart');
  await expect(qs).toBeVisible();
  await expect(qs.locator('.docs-home-steps li')).toHaveCount(3);
  await expect(qs.locator('code').first()).toContainText('git clone https://github.com/TrebuchetDynamics/gormes-agent.git');
  await expect(qs.getByText('./bin/gormes doctor --offline')).toBeVisible();
  await expect(qs.getByText('./bin/gormes --offline')).toBeVisible();
  await expect(qs).not.toContainText('curl -fsSL https://gormes.ai/install.sh | sh');
  await expect(qs).not.toContainText('brew install trebuchet/gormes');

  // Capabilities and start-here hubs
  await expect(page.getByRole('heading', { name: 'Current operator surface' })).toBeVisible();
  await expect(page.locator('.docs-home-capability')).toHaveCount(6);
  await expect(page.getByRole('heading', { name: 'Docs by job' })).toBeVisible();
  await expect(page.locator('.docs-home-start-grid a')).toHaveCount(6);

  // User-first cards with ordinals and mini-TOCs
  const cards = page.locator('.docs-home-card');
  await expect(cards).toHaveCount(4);
  for (let i = 0; i < 4; i++) {
    const c = cards.nth(i);
    await expect(c.locator('.docs-home-card-ordinal')).toBeVisible();
    await expect(c.locator('.docs-home-card-mini-toc li')).toHaveCount(3);
    await expect(c.locator('.docs-home-card-cta')).toContainText(/Explore/i);
  }

  // Kickers map to the existing colored labels
  await expect(cards.nth(0)).toContainText('START');
  await expect(cards.nth(1)).toContainText('OPERATE');
  await expect(cards.nth(2)).toContainText('REFERENCE');
  await expect(cards.nth(3)).toContainText('ARCHITECTURE');

  // Sidebar prioritizes user/operator docs and keeps roadmap/parity last.
  await expect(page.locator('.docs-nav-group-label-shipped').first()).toBeVisible();
  await expect(page.locator('.docs-nav-group-label-progress').first()).toBeVisible();
  await expect(page.locator('.docs-nav-group-label-next').first()).toBeVisible();
  await expect(page.getByText('ROADMAP & PARITY')).toBeVisible();

  // External script budget: pagefind-ui.js + site.js (always) + livereload.js
  // (Hugo dev server only). Filter livereload so the assertion holds in both
  // dev and prod modes.
  const scripts = await page
    .locator('script[src]')
    .evaluateAll(els => els.filter(el => !el.src.includes('livereload')).length);
  expect(scripts).toBeLessThanOrEqual(2);
});
