import { test, expect } from '@playwright/test';

test('homepage renders the operator-console story', async ({ page }) => {
  await page.goto('/');

  await expect(page).toHaveTitle('Gormes.ai | Run Hermes Through a Go Operator Console');
  await expect(page.getByRole('heading', { name: 'Run Hermes Through a Go Operator Console.' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Install Hermes fast. Then boot Gormes.' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Shipping State, Not Wishcasting' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Boot Gormes' })).toBeVisible();
  await expect(page.getByText('curl -fsSL https://raw.githubusercontent.com/NousResearch/hermes-agent/main/scripts/install.sh | bash')).toBeVisible();
  await expect(page.getByText('Works on Linux, macOS, WSL2, and Android via Termux.')).toBeVisible();
  await expect(page.getByText('Windows: Native Windows is not supported. Please install WSL2')).toBeVisible();
  await expect(page.getByText('source ~/.bashrc    # reload shell (or: source ~/.zshrc)')).toBeVisible();
  await expect(page.getByText('curl -fsSL https://gormes.ai/install.sh | sh')).toBeVisible();
  await expect(page.getByText('Today the Gormes installer shells out to Go itself. Keep Go 1.25+ on PATH until release artifacts exist.')).toBeVisible();
  await expect(page.getByText('If the installer prints an export PATH line, run it in this shell before launching gormes.')).toBeVisible();
  await expect(page.getByText('gormes doctor --offline')).toBeVisible();
  await expect(page.getByText('git clone https://github.com/TrebuchetDynamics/gormes-agent')).toBeVisible();
  await expect(page.getByText('7.9 MB Static Binary')).toHaveCount(0);
  await expect(page.getByText('8.2M shell')).toHaveCount(0);
  await expect(page.getByText('15M telegram edge')).toHaveCount(0);
  await expect(page.locator('link[href="/static/site.css"]')).toHaveCount(1);
  await expect(page.locator('script')).toHaveCount(0);
});

test('mobile keeps the run-now flow readable', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/');

  await expect(page.getByRole('link', { name: 'Boot Gormes' })).toBeVisible();
  await expect(page.getByText('curl -fsSL https://raw.githubusercontent.com/NousResearch/hermes-agent/main/scripts/install.sh | bash')).toBeVisible();
  await expect(page.getByText('curl -fsSL https://gormes.ai/install.sh | sh')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Install Hermes fast. Then boot Gormes.' })).toBeVisible();

  const hasOverflow = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth);
  expect(hasOverflow).toBeFalsy();
});
