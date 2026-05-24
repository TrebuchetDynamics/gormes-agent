import { test, expect } from '@playwright/test';

import { expectMainHeading, visitDocsPage } from './helpers.mjs';

test('single page has Starlight prev/next links at bottom', async ({ page }) => {
  await visitDocsPage(page, '/using-gormes/quickstart/');

  const links = page.locator('a[rel="prev"], a[rel="next"]');
  await expect(links.first()).toBeVisible();
  await expect(links.first()).toContainText(/Previous|Next/);
});

test('redirect aliases still resolve old builder-loop URLs', async ({ page }) => {
  await visitDocsPage(page, '/building-gormes/agent-queue/');
  await expect(page).toHaveTitle(/Agent Queue \| Gormes Docs/);
  await expectMainHeading(page, 'Agent Queue', { exact: true });
});
