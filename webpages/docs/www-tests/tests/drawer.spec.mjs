import { test, expect } from '@playwright/test';

test('Starlight mobile navigation exposes core docs links', async ({ page }) => {
  await page.setViewportSize({ width: 360, height: 760 });
  await page.goto('/install/');

  await expect(page.getByRole('button', { name: 'Menu' })).toBeVisible();
  const nav = page.getByRole('navigation', { name: 'Main' });
  await expect(nav).toHaveCount(1);
  await expect(nav.locator('a[href="/install/linux-macos/"]')).toHaveCount(1);
  await expect(nav.locator('a[href="/install/termux/"]')).toHaveCount(1);
  await expect(nav.locator('a[href="/configure/providers/"]')).toHaveCount(1);
  await expect(nav.locator('a[href="/operate/"]')).toHaveCount(1);
  await expect(nav.locator('a[href="/operate/first-chat/"]')).toHaveCount(1);
  await expect(nav.locator('a[href="/troubleshooting/doctor/"]')).toHaveCount(1);
  await expect(nav.locator('a[href="/reference/status-readiness/"]')).toHaveCount(1);
});

test('desktop navigation keeps the active page visible', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await page.goto('/install/linux-macos/');

  const nav = page.getByRole('navigation', { name: 'Main' });
  await expect(page.getByRole('button', { name: 'Menu' })).toBeHidden();
  await expect(nav.getByRole('link', { name: 'Linux & macOS' })).toHaveAttribute('aria-current', 'page');
});
