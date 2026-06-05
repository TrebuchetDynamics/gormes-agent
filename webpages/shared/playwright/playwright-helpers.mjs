import assert from 'node:assert/strict';

export async function visitPage(page, path, viewport) {
  if (viewport) {
    await page.setViewportSize({
      width: viewport.width,
      height: viewport.height,
    });
  }
  await page.goto(path, { waitUntil: 'domcontentloaded' });
  await page.waitForLoadState('networkidle');
}

export async function expectMainHeading(page, name, options = {}) {
  await page.getByRole('heading', { name, ...options }).first().waitFor({
    state: 'visible',
  });
}

export async function expectNoHorizontalOverflow(page, message = 'page has no horizontal overflow') {
  const overflow = await page.evaluate(() =>
    document.documentElement.scrollWidth > window.innerWidth,
  );
  assert.equal(overflow, false, message);
}
