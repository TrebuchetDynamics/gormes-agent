import { test, expect } from '@playwright/test';

test('docs home renders through Starlight with operator-first content', async ({ page }) => {
  await page.goto('/');

  await expect(page).toHaveTitle(/Gormes Documentation \| Gormes Docs/);
  await expect(page.getByRole('heading', { name: 'Gormes Documentation' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Gormes', exact: true })).toBeVisible();
  await expect(page.getByText('Gormes runs AI agents from one Go-native runtime.')).toBeVisible();
  await expect(page.locator('main').getByText(/Choose source build,.*install\.sh.*install\.ps1/)).toBeVisible();

  await expect(page.getByText('git clone https://github.com/TrebuchetDynamics/gormes-agent.git')).toBeVisible();
  await expect(page.getByText('gormes doctor --offline').first()).toBeVisible();
  await expect(page.getByText('gormes --offline').first()).toBeVisible();
  await expect(page.locator('main')).not.toContainText('curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh | sh');
  await expect(page.locator('main')).not.toContainText('brew install trebuchet/gormes');

  const nav = page.getByRole('navigation', { name: 'Main' });
  for (const label of [
    'Start here',
    'Install',
    'Configure',
    'CLI reference',
    'Recipes',
    'Troubleshooting',
    'Why Gormes',
  ]) {
    await expect(nav.getByText(label, { exact: true }).first()).toBeVisible();
  }

  await expect(page.getByRole('link', { name: /GitHub/ })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Search' })).toBeVisible();
});
