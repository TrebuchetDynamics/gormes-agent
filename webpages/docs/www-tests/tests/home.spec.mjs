import { test, expect } from '@playwright/test';

test('docs home renders through Starlight with operator-first content', async ({ page }) => {
  await page.goto('/');

  await expect(page).toHaveTitle(/Gormes Documentation \| Gormes Docs/);
  await expect(page.getByRole('heading', { name: 'Gormes Documentation' })).toBeVisible();
  await expect(page.getByText('Install, configure, and operate Gormes from one Go-native runtime.')).toBeVisible();

  await expect(page.getByText('curl -fsSL https://github.com/TrebuchetDynamics/gormes-agent/releases/latest/download/install.sh | sh')).toBeVisible();
  await expect(page.getByText('gormes doctor --offline').first()).toBeVisible();
  await expect(page.getByText('gormes setup').first()).toBeVisible();
  await expect(page.getByText('gormes chat').first()).toBeVisible();
  await expect(page.getByRole('link', { name: 'Install on Windows' })).toHaveAttribute('href', /\/?install\/windows\//);
  await expect(page.getByRole('link', { name: 'Install on Termux' })).toHaveAttribute('href', /\/?install\/termux\//);
  await expect(page.locator('main')).not.toContainText(/https:\/\/gormes[.]ai\/install[.]sh/);
  await expect(page.locator('main')).not.toContainText('curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh | sh');
  await expect(page.locator('main')).not.toContainText('brew install trebuchet/gormes');

  const nav = page.getByRole('navigation', { name: 'Main' });
  for (const label of [
    'Quickstart',
    'Install',
    'Configure',
    'Operate',
    'Troubleshoot',
    'Reference',
    'Concepts',
    'Build Gormes',
    'Archive & Research',
  ]) {
    await expect(nav.getByText(label, { exact: true }).first()).toBeVisible();
  }

  await expect(page.getByRole('link', { name: /GitHub/ })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Search' })).toBeVisible();
});

test('roadmap alias resolves to public status page', async ({ page }) => {
  await page.goto('/roadmap/');

  await expect(page).toHaveTitle(/Status & Roadmap \| Gormes Docs/);
  await expect(page.getByRole('heading', { name: 'Status & Roadmap' })).toBeVisible();
  await expect(page.getByText('This page is the public readiness view')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Implementation evidence' })).toBeVisible();
});
