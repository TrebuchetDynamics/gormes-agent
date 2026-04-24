import { test, expect } from '@playwright/test';

test('mobile hamburger is an accessible button', async ({ page }) => {
  await page.setViewportSize({ width: 360, height: 760 });
  await page.goto('/using-gormes/quickstart/');

  const btn = page.locator('[data-testid="drawer-open"]');
  await expect(btn).toBeVisible();
  // It's a real button, not a label
  const tagName = await btn.evaluate(el => el.tagName);
  expect(tagName).toBe('BUTTON');
  // Accessible name is set
  await expect(btn).toHaveAttribute('aria-label', /nav/i);
});

test('mobile drawer opens via hamburger and closes via backdrop', async ({ page }) => {
  await page.setViewportSize({ width: 360, height: 760 });
  await page.goto('/using-gormes/quickstart/');

  const sidebar = page.locator('.docs-sidebar');
  let leftBefore = await sidebar.evaluate(el => el.getBoundingClientRect().left);
  expect(leftBefore).toBeLessThan(0);

  await page.locator('[data-testid="drawer-open"]').click();
  await page.waitForTimeout(250);
  const leftOpen = await sidebar.evaluate(el => el.getBoundingClientRect().left);
  expect(leftOpen).toBeGreaterThanOrEqual(0);

  await page.locator('.drawer-backdrop').click({ force: true });
  await page.waitForTimeout(250);
  const leftClosed = await sidebar.evaluate(el => el.getBoundingClientRect().left);
  expect(leftClosed).toBeLessThan(0);
});

test('desktop >=768px does not show the hamburger', async ({ page }) => {
  await page.setViewportSize({ width: 1024, height: 768 });
  await page.goto('/');
  const btn = page.locator('[data-testid="drawer-open"]');
  const display = await btn.evaluate(el => getComputedStyle(el).display);
  expect(display).toBe('none');
});

test('mobile drawer has a close button inside and Esc closes it', async ({ page }) => {
  await page.setViewportSize({ width: 360, height: 760 });
  await page.goto('/using-gormes/quickstart/');

  await page.locator('[data-testid="drawer-open"]').click();
  await page.waitForTimeout(250);

  const closeBtn = page.locator('[data-testid="drawer-close"]');
  await expect(closeBtn).toBeVisible();
  await closeBtn.click();
  await page.waitForTimeout(250);

  const sidebar = page.locator('.docs-sidebar');
  const leftAfterCloseClick = await sidebar.evaluate(el => el.getBoundingClientRect().left);
  expect(leftAfterCloseClick).toBeLessThan(0);

  // Re-open and close via Esc
  await page.locator('[data-testid="drawer-open"]').click();
  await page.waitForTimeout(250);
  await page.keyboard.press('Escape');
  await page.waitForTimeout(250);
  const leftAfterEsc = await sidebar.evaluate(el => el.getBoundingClientRect().left);
  expect(leftAfterEsc).toBeLessThan(0);
});

test('mobile drawer auto-closes on link tap', async ({ page }) => {
  await page.setViewportSize({ width: 360, height: 760 });
  await page.goto('/using-gormes/');

  await page.locator('[data-testid="drawer-open"]').click();
  await page.waitForTimeout(250);

  // Click any nav link inside the sidebar
  const link = page.locator('.docs-sidebar a[href]').first();
  await link.click();
  await page.waitForTimeout(400); // navigation + transition

  const sidebar = page.locator('.docs-sidebar');
  const left = await sidebar.evaluate(el => el.getBoundingClientRect().left);
  expect(left).toBeLessThan(0);
});
