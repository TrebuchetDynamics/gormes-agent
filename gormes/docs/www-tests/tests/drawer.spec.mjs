import { test, expect } from '@playwright/test';

test('mobile drawer opens and closes via hamburger', async ({ page }) => {
  await page.setViewportSize({ width: 360, height: 760 });
  await page.goto('/using-gormes/quickstart/');

  // Sidebar starts off-screen (transform: translateX(-100%))
  const sidebar = page.locator('.docs-sidebar');
  let transform = await sidebar.evaluate(el => getComputedStyle(el).transform);
  expect(transform).toContain('matrix'); // some transform applied

  // Click the hamburger
  await page.locator('.drawer-btn').click();
  await page.waitForTimeout(250); // transition

  // Sidebar now visible (transform ~= translateX(0))
  const isVisibleByCoord = await sidebar.evaluate(el => el.getBoundingClientRect().left >= 0);
  expect(isVisibleByCoord).toBeTruthy();

  // Click the backdrop — drawer closes
  await page.locator('.drawer-backdrop').click({ force: true });
  await page.waitForTimeout(250);
  const isHiddenAgain = await sidebar.evaluate(el => el.getBoundingClientRect().left < 0);
  expect(isHiddenAgain).toBeTruthy();
});

test('desktop >=768px has no hamburger', async ({ page }) => {
  await page.setViewportSize({ width: 1024, height: 768 });
  await page.goto('/');
  const btn = page.locator('.drawer-btn');
  // drawer-btn exists in DOM but hidden via CSS at this viewport
  const display = await btn.evaluate(el => getComputedStyle(el).display);
  expect(display).toBe('none');
});
