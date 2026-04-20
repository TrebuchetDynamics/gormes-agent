import { test, expect } from '@playwright/test';

test('homepage renders the redesigned landing', async ({ page }) => {
  await page.goto('/');

  await expect(page).toHaveTitle('Gormes — One Go Binary. Same Hermes Brain.');
  await expect(page.getByRole('heading', { name: 'One Go Binary. Same Hermes Brain.' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Why a Go layer matters.' })).toBeVisible();
  await expect(page.getByRole('heading', { name: "What ships now, what doesn't." })).toBeVisible();
  await expect(page.getByText('curl -fsSL https://gormes.ai/install.sh | sh')).toBeVisible();
  await expect(page.getByText('Requires Hermes backend at localhost:8642.')).toBeVisible();
  await expect(page.getByText('Run Hermes Through a Go Operator Console.')).toHaveCount(0);
  await expect(page.locator('link[href="/static/site.css"]')).toHaveCount(1);
  // Copy buttons require a tiny inline clipboard script — bounded to install steps.
  await expect(page.locator('button.copy-btn')).toHaveCount(2);
});

test('mobile keeps the install command readable', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/');

  await expect(page.getByRole('heading', { name: 'One Go Binary. Same Hermes Brain.' })).toBeVisible();
  await expect(page.getByText('curl -fsSL https://gormes.ai/install.sh | sh')).toBeVisible();

  const hasOverflow = await page.evaluate(() =>
    document.documentElement.scrollWidth > window.innerWidth
  );
  expect(hasOverflow).toBeFalsy();
});
