import { test, expect } from '@playwright/test';

test('TOC scrollspy highlights the currently visible heading', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await page.goto('/building-gormes/architecture_plan/phase-6-learning-loop/');

  const toc = page.locator('.docs-toc-body');
  await expect(toc).toBeVisible();

  // Find all anchors in the TOC
  const links = toc.locator('a[href^="#"]');
  const count = await links.count();
  if (count < 2) test.skip(); // page doesn't have enough headings

  // Scroll to the second heading; the second TOC link should be .active
  const secondHref = await links.nth(1).getAttribute('href');
  expect(secondHref).toBeTruthy();
  const anchorId = secondHref.replace('#', '');
  await page.evaluate(id => {
    document.getElementById(id).scrollIntoView({ behavior: 'instant', block: 'start' });
  }, anchorId);
  await page.waitForTimeout(250);

  const activeCount = await toc.locator('a.active').count();
  expect(activeCount).toBeGreaterThanOrEqual(1);
  const firstActiveHref = await toc.locator('a.active').first().getAttribute('href');
  expect(firstActiveHref).toBe(secondHref);
});
