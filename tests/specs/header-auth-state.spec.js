// @ts-check
import { test, expect } from '@playwright/test';
import { applySessionCookie } from '../helpers/browser.js';
import { resolveTestConfig } from '../helpers/config.js';
import { buildAdminUser } from '../helpers/fixtures.js';
import { installTauthStub } from '../helpers/tauthStub.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);

/**
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<void>}
 */
async function installGoogleIdentityStub(page) {
  await page.route('https://accounts.google.com/gsi/client', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/javascript; charset=utf-8',
      body: `window.google = window.google || {};
window.google.accounts = window.google.accounts || {};
window.google.accounts.id = {
  initialize: function() {},
  renderButton: function(target) {
    if (target && typeof target.setAttribute === 'function') {
      target.setAttribute('data-google-stubbed', 'true');
    }
  },
  prompt: function() {},
  cancel: function() {},
  disableAutoSelect: function() {},
  revoke: function() {}
};`
    });
  });
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} path
 * @param {{ silentBootstrap?: boolean, delayMs?: number }} [tauthOptions]
 * @returns {Promise<void>}
 */
async function openPageWithoutSession(page, path, tauthOptions) {
  await installGoogleIdentityStub(page);
  await installTauthStub(page, config, tauthOptions);
  await page.goto(path, { waitUntil: 'domcontentloaded' });
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} path
 * @param {{ silentBootstrap?: boolean, delayMs?: number }} [tauthOptions]
 * @returns {Promise<void>}
 */
async function openPageWithSession(page, path, tauthOptions) {
  await installGoogleIdentityStub(page);
  await installTauthStub(page, config, tauthOptions);
  await applySessionCookie(page.context(), config, adminUser);
  await page.goto(path, { waitUntil: 'domcontentloaded' });
}

test('dashboard requires authentication and redirects unauthenticated users to login', async ({ page }) => {
  await openPageWithoutSession(page, '/app');
  await expect(page).toHaveURL(/\/login\/?$/);
});

test('login page redirects authenticated users to the dashboard', async ({ page }) => {
  await openPageWithSession(page, '/login');
  await expect(page).toHaveURL(/\/app\/?$/);
});

test('login page redirects authenticated users after silent session recovery', async ({ page }) => {
  await openPageWithSession(page, '/login', { silentBootstrap: true });
  await expect(page).toHaveURL(/\/app\/?$/);
});

test('login page renders header while tauth bootstrap is delayed', async ({ page }) => {
  await openPageWithoutSession(page, '/login', { delayMs: 2500 });
  await expect(page.locator('mpr-header > header.mpr-header')).toBeVisible({ timeout: 1000 });
  await expect(page.locator('mpr-footer footer.mpr-footer')).toBeVisible({ timeout: 1000 });
  await expect(page).toHaveURL(/\/login\/?$/);
});

test('privacy page keeps header auth state synchronized for authenticated sessions', async ({ page }) => {
  await openPageWithSession(page, '/privacy');
  await expect(page.locator('mpr-header')).toHaveAttribute('data-loopaware-auth-state', 'authenticated');
  await expect(page.locator('mpr-header > header.mpr-header')).toHaveClass(/mpr-header--authenticated/);
  await expect(page.locator('mpr-user[data-loopaware-user-menu="true"]')).toHaveAttribute('data-mpr-user-status', 'authenticated');
  await expect(page.locator('mpr-header [data-mpr-header="google-signin"]')).toBeHidden();
  await expect(page.locator('mpr-user[data-loopaware-user-menu="true"]')).toBeVisible();
});

test('privacy page shows logout overlay for static-page sign-out', async ({ page }) => {
  await openPageWithSession(page, '/privacy');
  await page.evaluate(() => {
    document.dispatchEvent(new CustomEvent('mpr-user:logout'));
  });
  await expect(page.locator('#logout-overlay')).toBeVisible();
  await expect(page.locator('body')).toHaveClass(/logging-out/);
});
