import { test, expect } from '@playwright/test';

test('docs home renders the three-audience split', async ({ page }) => {
  await page.goto('/');

  await expect(page).toHaveTitle(/Gormes Docs/);
  await expect(page.getByRole('heading', { name: 'Gormes Docs', level: 1 })).toBeVisible();

  // Three cards, one per audience
  await expect(page.getByRole('link', { name: /USING GORMES/i })).toBeVisible();
  await expect(page.getByRole('link', { name: /BUILDING GORMES/i })).toBeVisible();
  await expect(page.getByRole('link', { name: /UPSTREAM HERMES/i })).toBeVisible();

  // Sidebar has colored group labels
  await expect(page.locator('.docs-nav-group-label-shipped')).toBeVisible();
  await expect(page.locator('.docs-nav-group-label-progress')).toBeVisible();
  await expect(page.locator('.docs-nav-group-label-next')).toBeVisible();

  // External script budget: pagefind-ui.js (always) + livereload.js (Hugo dev
  // server injects this, prod build doesn't). Phase 1 of the UI polish will
  // add site.js (Tasks 2-7), bumping the prod budget to 2.
  const scripts = await page
    .locator('script[src]')
    .evaluateAll(els => els.filter(el => !el.src.includes('livereload')).length);
  expect(scripts).toBeLessThanOrEqual(1); // pagefind-ui.js only (Phase 1 raises this to 2)
});
