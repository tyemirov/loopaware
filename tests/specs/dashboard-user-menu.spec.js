// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildAdminUser, openDashboard } from '../helpers/fixtures.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);

test('single loopaware user menu rendered', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await expect(page.locator('mpr-user[data-loopaware-user-menu="true"]')).toHaveCount(1);
});

test('user menu includes account settings action', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  const menuItems = await page.locator('mpr-user[data-loopaware-user-menu="true"]').getAttribute('menu-items');
  const parsed = menuItems ? JSON.parse(menuItems) : [];
  const actions = Array.isArray(parsed) ? parsed.map((item) => item.action) : [];
  expect(actions).toContain('account-settings');
});

test('account settings menu event opens modal', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await page.evaluate(() => {
    document.dispatchEvent(new CustomEvent('mpr-user:menu-item', { detail: { action: 'account-settings' } }));
  });
  await expect(page.locator('#settings-modal')).toBeVisible();
});

test('header settings click opens modal', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await page.evaluate(() => {
    document.dispatchEvent(new CustomEvent('mpr-ui:header:settings-click'));
  });
  await expect(page.locator('#settings-modal')).toBeVisible();
});

test('unknown menu action leaves modal hidden', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await expect(page.locator('#settings-modal')).toBeHidden();
  await page.evaluate(() => {
    document.dispatchEvent(new CustomEvent('mpr-user:menu-item', { detail: { action: 'unknown' } }));
  });
  await expect(page.locator('#settings-modal')).toBeHidden();
});

test('user menu renders in avatar mode', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await expect(page.locator('mpr-user[data-loopaware-user-menu="true"]')).toHaveAttribute('display-mode', 'avatar');
});
