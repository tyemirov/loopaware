// @ts-check
import { test, expect } from '@playwright/test';
import { applySessionCookie } from '../helpers/browser.js';
import { resolveTestConfig } from '../helpers/config.js';
import { buildAdminUser } from '../helpers/fixtures.js';
import { installTauthStub } from '../helpers/tauthStub.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);
const MPR_UI_STYLE_URL = 'https://cdn.jsdelivr.net/gh/MarcoPoloResearchLab/mpr-ui@v3.8.2/mpr-ui.css';
const MPR_UI_SCRIPT_URL = 'https://cdn.jsdelivr.net/gh/MarcoPoloResearchLab/mpr-ui@v3.8.2/mpr-ui.js';
const TAUTH_SCRIPT_URL = 'https://cdn.jsdelivr.net/gh/tyemirov/TAuth@v1.0.1/web/tauth.js';

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
 * @param {{ silentBootstrap?: boolean, delayMs?: number, bootstrapDelayMs?: number, currentUserDelayMs?: number }} [tauthOptions]
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
 * @param {{ silentBootstrap?: boolean, delayMs?: number, bootstrapDelayMs?: number, currentUserDelayMs?: number }} [tauthOptions]
 * @returns {Promise<void>}
 */
async function openPageWithSession(page, path, tauthOptions) {
  await installGoogleIdentityStub(page);
  await installTauthStub(page, config, tauthOptions);
  await applySessionCookie(page.context(), config, adminUser);
  await page.goto(path, { waitUntil: 'domcontentloaded' });
}

/**
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<void>}
 */
async function expectPinnedCdnAssets(page) {
  const assetUrls = await page.evaluate(() => {
    const mprUiStyle = document.getElementById('mpr-ui-style');
    const tauthScript = document.getElementById('tauth-script');
    const mprUiScript = document.getElementById('mpr-ui-script');
    const browserAssetUrls = Array.from(document.querySelectorAll('script[src], link[rel="stylesheet"][href]'))
      .map((element) => element.getAttribute('src') || element.getAttribute('href') || '');
    return {
      styleHref: mprUiStyle ? mprUiStyle.getAttribute('href') || '' : '',
      tauthSrc: tauthScript ? tauthScript.getAttribute('src') || '' : '',
      mprUiSrc: mprUiScript ? mprUiScript.getAttribute('src') || '' : '',
      vendorUrls: browserAssetUrls.filter((url) => url.includes('/vendor/'))
    };
  });
  expect(assetUrls.styleHref).toBe(MPR_UI_STYLE_URL);
  expect(assetUrls.tauthSrc).toBe(TAUTH_SCRIPT_URL);
  expect(assetUrls.mprUiSrc).toBe(MPR_UI_SCRIPT_URL);
  expect(assetUrls.vendorUrls).toEqual([]);
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

test('login page renders header while tauth session bootstrap is delayed', async ({ page }) => {
  await openPageWithoutSession(page, '/login', { bootstrapDelayMs: 2500 });
  await expect(page.locator('mpr-header > header.mpr-header')).toBeVisible({ timeout: 2000 });
  await expect(page.locator('mpr-footer footer.mpr-footer')).toBeVisible({ timeout: 2000 });
  await expect(page).toHaveURL(/\/login\/?$/);
});

test('dashboard preserves authenticated session state while tauth script load is delayed', async ({ page }) => {
  await openPageWithSession(page, '/app', { delayMs: 2500 });
  await expect(page).toHaveURL(/\/app\/?$/);
  await expect(page.locator('mpr-header')).toHaveAttribute('data-loopaware-auth-state', 'authenticated');
});

test('dashboard does not bounce to login while authenticated session recovery is still settling', async ({ page }) => {
  await openPageWithSession(page, '/app', { silentBootstrap: true, currentUserDelayMs: 2500 });
  await expect(page).toHaveURL(/\/app\/?$/);
  await expect(page.locator('mpr-header')).toHaveAttribute('data-loopaware-auth-state', 'authenticated', { timeout: 5000 });
});

test('login page loads pinned CDN assets for auth UI', async ({ page }) => {
  await openPageWithoutSession(page, '/login');
  await expectPinnedCdnAssets(page);
});

test('login page does not bootstrap the landing widget', async ({ page }) => {
  const widgetRequests = [];
  const consoleErrors = [];
  page.on('request', (request) => {
    const url = request.url();
    if (url.includes('/widget.js') || url.includes('/public/widget-config')) {
      widgetRequests.push(url);
    }
  });
  page.on('console', (message) => {
    if (message.type() === 'error') {
      consoleErrors.push(message.text());
    }
  });

  await openPageWithoutSession(page, '/login');
  await expect(page.locator('mpr-header > header.mpr-header')).toBeVisible();
  await expect(page.locator('mpr-footer footer.mpr-footer')).toBeVisible();

  expect(widgetRequests).toEqual([]);
  expect(
    consoleErrors.filter((message) => message.includes('widget.js: initialize_failed') || message.includes('widget_config_forbidden'))
  ).toHaveLength(0);
});

test('login page applies configured tauth origin to the header', async ({ page }) => {
  const tauthOrigin = 'https://tauth.example.test';
  await openPageWithoutSession(page, `/login?tauth_origin=${encodeURIComponent(tauthOrigin)}`);
  await expect(page.locator('mpr-header')).toHaveAttribute('tauth-url', tauthOrigin);
  expect(await page.evaluate(() => String(window['__LOOPAWARE_TAUTH_ORIGIN__'] || ''))).toBe(tauthOrigin);
});

test('login page user menu does not emit tenant bootstrap errors', async ({ page }) => {
  const consoleErrors = [];
  page.on('console', (message) => {
    if (message.type() === 'error') {
      consoleErrors.push(message.text());
    }
  });
  await openPageWithoutSession(page, '/login');
  await expect(page.locator('mpr-header > header.mpr-header')).toBeVisible();
  expect(consoleErrors.filter((message) => message.includes('mpr-ui.tenant_id_required'))).toHaveLength(0);
});

test('privacy page keeps header auth state synchronized for authenticated sessions', async ({ page }) => {
  await openPageWithSession(page, '/privacy');
  await expect(page.locator('mpr-header')).toHaveAttribute('data-loopaware-auth-state', 'authenticated');
  await expect(page.locator('mpr-header > header.mpr-header')).toHaveClass(/mpr-header--authenticated/);
  await expect(page.locator('mpr-user[data-loopaware-user-menu="true"]')).toHaveAttribute('data-mpr-user-status', 'authenticated');
  await expect(page.locator('mpr-header [data-mpr-header="google-signin"]')).toBeHidden();
  await expect(page.locator('mpr-user[data-loopaware-user-menu="true"]')).toBeVisible();
});

test('privacy page user menu does not emit tenant bootstrap errors', async ({ page }) => {
  const consoleErrors = [];
  page.on('console', (message) => {
    if (message.type() === 'error') {
      consoleErrors.push(message.text());
    }
  });
  await openPageWithSession(page, '/privacy');
  await expect(page.locator('mpr-user[data-loopaware-user-menu="true"]')).toHaveAttribute('data-mpr-user-status', 'authenticated');
  expect(consoleErrors.filter((message) => message.includes('mpr-ui.tenant_id_required'))).toHaveLength(0);
});

test('privacy page shows logout overlay for static-page sign-out', async ({ page }) => {
  await openPageWithSession(page, '/privacy');
  await page.evaluate(() => {
    document.dispatchEvent(new CustomEvent('mpr-user:logout'));
  });
  await expect(page.locator('#logout-overlay')).toBeVisible();
  await expect(page.locator('body')).toHaveClass(/logging-out/);
});

test('privacy page loads pinned CDN assets for auth UI', async ({ page }) => {
  await openPageWithoutSession(page, '/privacy');
  await expectPinnedCdnAssets(page);
});

test('dashboard loads pinned CDN assets for auth UI', async ({ page }) => {
  await openPageWithSession(page, '/app');
  await expectPinnedCdnAssets(page);
});

test('vendored mpr-ui asset URLs are not served', async ({ page }) => {
  const scriptResponse = await page.request.get('/vendor/mpr-ui/mpr-ui.js');
  const styleResponse = await page.request.get('/vendor/mpr-ui/mpr-ui.css');
  expect(scriptResponse.status()).toBe(404);
  expect(styleResponse.status()).toBe(404);
});
