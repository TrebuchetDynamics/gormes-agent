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

test('mobile operator install journey keeps code blocks and next links usable', async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 568 });
  await page.goto('/install/linux-macos/');

  await expect(page.getByRole('heading', { name: 'Linux and macOS' }).first()).toBeVisible();
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
  await expect(page.getByRole('heading', { name: 'Windows' }).first()).toBeVisible();
});

test('mobile menu button is accessible and desktop hides it', async ({ page }) => {
  await page.setViewportSize({ width: 360, height: 760 });
  await page.goto('/getting-started/first-run/');

  await expect(page.getByRole('button', { name: 'Menu' })).toBeVisible();
  await expect(page.getByRole('navigation', { name: 'Main' })).toHaveCount(1);

  await page.setViewportSize({ width: 1024, height: 768 });
  await expect(page.getByRole('button', { name: 'Menu' })).toBeHidden();
});
