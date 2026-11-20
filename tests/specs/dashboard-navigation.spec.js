// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildSessionCookie } from '../helpers/auth.js';
import { buildAdminUser, buildUniqueEmail, buildUniqueOrigin, createTestSite, openDashboard, selectSite } from '../helpers/fixtures.js';
import { capturePageScreenshot, ensureScreenshotDirectory } from '../helpers/screenshot.js';

const config = resolveTestConfig();

async function createNavigationSites() {
  const owner = buildAdminUser(config, { email: buildUniqueEmail('navigation-owner') });
  const cookie = buildSessionCookie(config, owner);
  const first = await createTestSite(config, cookie, { name: 'First navigation site', allowedOrigin: buildUniqueOrigin('navigation-first'), ownerEmail: owner.email });
  const second = await createTestSite(config, cookie, { name: 'Second navigation site', allowedOrigin: buildUniqueOrigin('navigation-second'), ownerEmail: owner.email });
  return { owner, first, second };
}

for (const width of [1280, 390]) {
  test(`Traffic is first and selected when a site opens at ${width}px`, async ({ page }) => {
    const { owner } = await createNavigationSites();
    await page.setViewportSize({ width, height: 900 });
    await openDashboard(page, config, owner);
    const tabs = page.locator('#dashboard-section-tabs [role="tab"]');
    await expect(tabs).toHaveText(['Traffic', 'Feedback', 'Subscriptions', 'Health', 'LA Sentry', 'Admin']);
    await expect(page.getByRole('tab', { name: 'Traffic', exact: true })).toHaveAttribute('aria-selected', 'true');
    await expect(page.locator('[data-dashboard-card="traffic"]')).toBeVisible();
    await expect(page.locator('[data-widget-card="feedback"]')).toBeHidden();
    await capturePageScreenshot(page, ensureScreenshotDirectory('dashboard-I066'), `traffic-default-${width}`);
  });
}

test('opening another site resets Traffic while autosave and report return preserve an explicit tab', async ({ page }) => {
  const { owner, first, second } = await createNavigationSites();
  await openDashboard(page, config, owner);
  await selectSite(page, first.id);
  const feedback = page.getByRole('tab', { name: 'Feedback', exact: true });
  const traffic = page.getByRole('tab', { name: 'Traffic', exact: true });
  await feedback.click();
  await page.locator('#edit-site-name').fill('Renamed navigation site');
  await expect(page.locator(`#sites-list [data-site-id="${first.id}"]`)).toContainText('Renamed navigation site');
  await expect(feedback).toHaveAttribute('aria-selected', 'true');
  await page.getByRole('button', { name: 'Statistics', exact: true }).click();
  await expect(page.locator('#all-sites-traffic-view')).toBeVisible();
  await page.getByRole('button', { name: 'Dashboard', exact: true }).click();
  await expect(feedback).toHaveAttribute('aria-selected', 'true');
  await selectSite(page, second.id);
  await expect(traffic).toHaveAttribute('aria-selected', 'true');
  await expect(page.locator('[data-dashboard-card="traffic"]')).toBeVisible();
  await feedback.click();
  await selectSite(page, first.id);
  await expect(traffic).toHaveAttribute('aria-selected', 'true');
});
