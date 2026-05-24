import { expect } from '@playwright/test';

export async function visitDocsPage(page, path, viewport) {
  if (viewport) {
    await page.setViewportSize({
      width: viewport.width,
      height: viewport.height,
    });
  }
  await page.goto(path);
}

export async function expectMainHeading(page, name, options = {}) {
  await expect(
    page.getByRole('heading', { name, ...options }).first(),
  ).toBeVisible();
}

export async function expectNoHorizontalOverflow(page, message = 'page has no horizontal overflow') {
  const overflow = await page.evaluate(() =>
    document.documentElement.scrollWidth > window.innerWidth,
  );
  expect(overflow, message).toBeFalsy();
}
