import { test, expect } from '@playwright/test';

test('docs home renders through Starlight with operator-first content', async ({ page }) => {
  await page.goto('/');

  await expect(page).toHaveTitle(/Gormes Documentation \| Gormes Docs/);
  await expect(page.getByRole('heading', { name: 'Gormes Documentation' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Gormes', exact: true })).toBeVisible();
  await expect(page.getByText('Gormes runs AI agents as one Go-native agent runtime.')).toBeVisible();
  await expect(page.getByText('Start offline, prove the machine works')).toBeVisible();

  await expect(page.getByText('git clone https://github.com/TrebuchetDynamics/gormes-agent.git')).toBeVisible();
  await expect(page.getByText('gormes doctor --offline').first()).toBeVisible();
  await expect(page.getByText('gormes --offline').first()).toBeVisible();
  await expect(page.locator('main')).not.toContainText('curl -fsSL https://gormes.ai/install.sh | sh');
  await expect(page.locator('main')).not.toContainText('brew install trebuchet/gormes');

  const nav = page.getByRole('navigation', { name: 'Main' });
  for (const label of [
    'Getting Started',
    'Operate',
    'Using Gormes',
    'Reference',
    'Architecture',
    'Development',
    'Parity',
    'Building Gormes',
    'Upstream Hermes',
  ]) {
    await expect(nav.getByText(label, { exact: true }).first()).toBeVisible();
  }

  await expect(page.getByRole('link', { name: /GitHub/ })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Search' })).toBeVisible();
});
