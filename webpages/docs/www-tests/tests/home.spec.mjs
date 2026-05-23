import { test, expect } from '@playwright/test';

test('docs home renders through Starlight with operator-first content', async ({ page }) => {
  await page.goto('/');

  await expect(page).toHaveTitle(/Gormes Documentation \| Gormes Docs/);
  await expect(page.getByRole('heading', { name: 'Gormes Documentation' })).toBeVisible();
  await expect(page.getByText('Install, configure, and operate Gormes from one Go-native runtime.')).toBeVisible();

  await expect(page.getByText('curl -fsSL https://gormes.ai/install.sh | bash')).toBeVisible();
  await expect(page.getByText('gormes doctor --offline').first()).toBeVisible();
  await expect(page.getByText('gormes setup').first()).toBeVisible();
  await expect(page.getByText('gormes chat').first()).toBeVisible();
  await expect(page.getByRole('link', { name: 'Install on Windows' })).toHaveAttribute('href', /\/?install\/windows\//);
  await expect(page.getByRole('link', { name: 'Install on Termux' })).toHaveAttribute('href', /\/?install\/termux\//);
  await expect(page.locator('main')).not.toContainText('github.com/TrebuchetDynamics/gormes-agent/releases/latest/download/install.sh');
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

test('operator docs journey keeps install commands and runtime checks consistent', async ({ page }) => {
  await page.goto('/install/linux-macos/');

  await expect(page.getByRole('heading', { name: 'Linux and macOS' }).first()).toBeVisible();
  await expect(page.locator('main')).toContainText('https://gormes.ai/install.sh');
  await expect(page.locator('main')).not.toContainText('raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh');
  await expect(page.getByRole('link', { name: 'Providers', exact: true })).toHaveAttribute('href', /\/configure\/providers\//);

  await page.goto('/configure/providers/');
  await expect(page.getByRole('heading', { name: 'Providers' }).first()).toBeVisible();
  await expect(page.locator('main')).toContainText('gormes doctor --offline');
  await expect(page.locator('main')).toContainText('gormes chat -q "smoke test"');

  await page.goto('/operate/first-chat/');
  await expect(page.getByRole('heading', { name: 'Connect a provider and open chat' }).first()).toBeVisible();
  await expect(page.locator('main')).toContainText('gormes auth list');
  await expect(page.locator('main')).toContainText('gormes chat');
  await expect(page.locator('main')).not.toContainText('hermes chat');

  await page.goto('/troubleshooting/doctor/');
  await expect(page.getByRole('heading', { name: 'Doctor' }).first()).toBeVisible();
  await expect(page.locator('main')).toContainText('gormes doctor --offline');
  await expect(page.locator('main')).not.toContainText('python -m');
});

test('roadmap alias resolves to public status page', async ({ page }) => {
  await page.goto('/roadmap/');

  await expect(page).toHaveTitle(/Status & Roadmap \| Gormes Docs/);
  await expect(page.getByRole('heading', { name: 'Status & Roadmap' })).toBeVisible();
  await expect(page.getByText('This page is the public readiness view')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Implementation evidence' })).toBeVisible();
});
