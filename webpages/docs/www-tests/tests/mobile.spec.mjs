import { test, expect } from '@playwright/test';

import { expectMainHeading, expectNoHorizontalOverflow, visitDocsPage } from './helpers.mjs';

const VIEWPORTS = [
  { label: 'iPhone SE', width: 320, height: 568 },
  { label: 'small Android', width: 360, height: 760 },
  { label: 'iPhone 15', width: 390, height: 844 },
  { label: 'iPhone Plus', width: 430, height: 932 },
  { label: 'iPad portrait', width: 768, height: 1024 },
  { label: 'Laptop', width: 1024, height: 768 },
];

for (const vp of VIEWPORTS) {
  test(`docs home (${vp.label} ${vp.width}x${vp.height}) has no horizontal overflow`, async ({ page }) => {
    await visitDocsPage(page, '/', vp);

    await expectNoHorizontalOverflow(page, `page overflows at ${vp.width}px`);
  });

  test(`docs article page (${vp.label}) has no overflow and renders Starlight TOC`, async ({ page }) => {
    await visitDocsPage(page, '/building-gormes/architecture_plan/phase-6-learning-loop/', vp);

    await expectNoHorizontalOverflow(page, `article overflows at ${vp.width}px`);

    await expectMainHeading(page, 'Phase 6 — The Learning Loop');
    await expect(page.locator('mobile-starlight-toc')).toHaveCount(1);

    if (vp.width >= 1280) {
      await expect(page.locator('.right-sidebar-panel')).toBeVisible();
      await expect(page.locator('.right-sidebar-panel a[href^="#"]').first()).toBeVisible();
    }
  });
}

test('mobile operator install journey keeps code blocks and next links usable', async ({ page }) => {
  await visitDocsPage(page, '/install/linux-macos/', { width: 320, height: 568 });

  await expectMainHeading(page, 'Linux and macOS');
  await expect(page.getByText('curl -fsSL https://gormes.ai/install.sh | bash').first()).toBeVisible();
  await expect(page.locator('main')).not.toContainText('raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh');

  const codeBlocks = page.locator('main pre, main .expressive-code');
  const count = await codeBlocks.count();
  expect(count, 'expected install article code blocks').toBeGreaterThan(1);
  for (let i = 0; i < count; i++) {
    const box = await codeBlocks.nth(i).boundingBox();
    expect(box, `code block ${i} has no layout box`).not.toBeNull();
    expect(box.width, `code block ${i} overflows mobile viewport`).toBeLessThanOrEqual(320);
  }

  const next = page.getByRole('link', { name: /Next Windows/ });
  await expect(next).toBeVisible();
  await next.click();
  await expect(page).toHaveURL(/\/install\/windows\//);
  await expectMainHeading(page, 'Windows');
});

test('mobile menu button is accessible and desktop hides it', async ({ page }) => {
  await visitDocsPage(page, '/getting-started/first-run/', { width: 360, height: 760 });

  await expect(page.getByRole('button', { name: 'Menu' })).toBeVisible();
  await expect(page.getByRole('navigation', { name: 'Main' })).toHaveCount(1);

  await page.setViewportSize({ width: 1024, height: 768 });
  await expect(page.getByRole('button', { name: 'Menu' })).toBeHidden();
});
