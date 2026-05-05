import { test, expect } from '@playwright/test';

test('single page has Starlight prev/next links at bottom', async ({ page }) => {
  await page.goto('/using-gormes/quickstart/');

  const links = page.locator('a[rel="prev"], a[rel="next"]');
  await expect(links.first()).toBeVisible();
  await expect(links.first()).toContainText(/Previous|Next/);
});

test('redirect aliases still resolve old builder-loop URLs', async ({ page }) => {
  await page.goto('/building-gormes/agent-queue/');
  await expect(page).toHaveTitle(/Agent Queue \| Gormes Docs/);
  await expect(page.getByRole('heading', { name: 'Agent Queue', exact: true }).first()).toBeVisible();
});
