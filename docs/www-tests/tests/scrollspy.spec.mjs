import { test, expect } from '@playwright/test';

test('Starlight TOC scrollspy highlights the currently visible heading', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await page.goto('/building-gormes/architecture_plan/phase-6-learning-loop/');

  const toc = page.locator('.right-sidebar-panel');
  await expect(toc).toBeVisible();

  const links = toc.locator('a[href^="#"]');
  const count = await links.count();
  if (count < 2) test.skip();

  const secondHref = await links.nth(1).getAttribute('href');
  expect(secondHref).toBeTruthy();
  await page.evaluate((id) => {
    document.getElementById(id).scrollIntoView({ behavior: 'instant', block: 'start' });
  }, secondHref.replace('#', ''));
  await page.waitForTimeout(250);

  await expect(links.nth(1)).toHaveAttribute('aria-current', 'true');
});
