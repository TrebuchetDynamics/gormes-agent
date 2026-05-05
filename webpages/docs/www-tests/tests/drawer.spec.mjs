import { test, expect } from '@playwright/test';

test('Starlight mobile navigation exposes core docs links', async ({ page }) => {
  await page.setViewportSize({ width: 360, height: 760 });
  await page.goto('/getting-started/');

  await expect(page.getByRole('button', { name: 'Menu' })).toBeVisible();
  const nav = page.getByRole('navigation', { name: 'Main' });
  await expect(nav).toHaveCount(1);
  await expect(nav.locator('a[href="/getting-started/first-run/"]')).toHaveCount(1);
  await expect(nav.locator('a[href="/getting-started/configuration/"]')).toHaveCount(1);
  await expect(nav.locator('a[href="/getting-started/troubleshooting/"]')).toHaveCount(1);
});

test('desktop navigation keeps the active page visible', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await page.goto('/getting-started/first-run/');

  const nav = page.getByRole('navigation', { name: 'Main' });
  await expect(page.getByRole('button', { name: 'Menu' })).toBeHidden();
  await expect(nav.getByRole('link', { name: 'First Run' })).toHaveAttribute('aria-current', 'page');
});
