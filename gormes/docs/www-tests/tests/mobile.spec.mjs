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
  test(`docs home (${vp.label} ${vp.width}×${vp.height}) has no horizontal overflow`, async ({ page }) => {
    await page.setViewportSize({ width: vp.width, height: vp.height });
    await page.goto('/');

    const overflow = await page.evaluate(() =>
      document.documentElement.scrollWidth > window.innerWidth,
    );
    expect(overflow, `page overflows at ${vp.width}px`).toBeFalsy();
  });

  test(`docs article page (${vp.label}) — Phase 6 — has no overflow and renders TOC correctly`, async ({ page }) => {
    await page.setViewportSize({ width: vp.width, height: vp.height });
    await page.goto('/building-gormes/architecture_plan/phase-6-learning-loop/');

    const overflow = await page.evaluate(() =>
      document.documentElement.scrollWidth > window.innerWidth,
    );
    expect(overflow, `article overflows at ${vp.width}px`).toBeFalsy();

    // Every code block has a tappable copy button
    const copyBoxes = await page.locator('button.copy-btn').evaluateAll(btns =>
      btns.map(b => b.getBoundingClientRect()).map(r => ({ h: r.height, w: r.width }))
    );
    for (const box of copyBoxes) {
      expect(box.h).toBeGreaterThanOrEqual(28);
      expect(box.w).toBeGreaterThanOrEqual(28);
    }

    // TOC is visible either as right-side panel (≥1024) or collapsed details (<1024)
    if (vp.width >= 1024) {
      await expect(page.locator('aside.docs-toc')).toBeVisible();
    } else {
      await expect(page.locator('.docs-toc-details')).toBeVisible();
    }
  });
}
