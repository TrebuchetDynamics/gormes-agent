import { test, expect } from '@playwright/test';

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
    await page.setViewportSize({ width: vp.width, height: vp.height });
    await page.goto('/');

    const overflow = await page.evaluate(() =>
      document.documentElement.scrollWidth > window.innerWidth,
    );
    expect(overflow, `page overflows at ${vp.width}px`).toBeFalsy();
  });

  test(`docs article page (${vp.label}) has no overflow and renders Starlight TOC`, async ({ page }) => {
    await page.setViewportSize({ width: vp.width, height: vp.height });
    await page.goto('/building-gormes/architecture_plan/phase-6-learning-loop/');

    const overflow = await page.evaluate(() =>
      document.documentElement.scrollWidth > window.innerWidth,
    );
    expect(overflow, `article overflows at ${vp.width}px`).toBeFalsy();

    await expect(page.getByRole('heading', { name: 'Phase 6 — The Learning Loop' }).first()).toBeVisible();
    await expect(page.locator('mobile-starlight-toc')).toHaveCount(1);

    if (vp.width >= 1280) {
      await expect(page.locator('.right-sidebar-panel')).toBeVisible();
      await expect(page.locator('.right-sidebar-panel a[href^="#"]').first()).toBeVisible();
    }
  });
}

test('mobile menu button is accessible and desktop hides it', async ({ page }) => {
  await page.setViewportSize({ width: 360, height: 760 });
  await page.goto('/getting-started/first-run/');

  await expect(page.getByRole('button', { name: 'Menu' })).toBeVisible();
  await expect(page.getByRole('navigation', { name: 'Main' })).toHaveCount(1);

  await page.setViewportSize({ width: 1024, height: 768 });
  await expect(page.getByRole('button', { name: 'Menu' })).toBeHidden();
});
