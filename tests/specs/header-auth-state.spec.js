// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { installAssetInspectionStubs } from '../helpers/externalAssets.js';
import { buildAdminUser, openAuthenticatedPage, openPublicPage } from '../helpers/fixtures.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);
const MPR_UI_STYLE_URL = 'https://cdn.jsdelivr.net/gh/MarcoPoloResearchLab/mpr-ui@latest/mpr-ui.css';
const MPR_UI_SCRIPT_URL = 'https://cdn.jsdelivr.net/gh/MarcoPoloResearchLab/mpr-ui@latest/mpr-ui.js';
const TAUTH_SCRIPT_URL = 'https://cdn.jsdelivr.net/gh/tyemirov/TAuth@v1.0.1/web/tauth.js';
const SITE_WIDGET_SITE_ID = 'a7ea8b8a-ff37-4a99-81fa-09a5952f83a9';
const PUBLIC_LOGIN_ENTRY_CASES = Object.freeze([
  Object.freeze({ label: 'pricing page', path: '/pricing' }),
  Object.freeze({ label: 'privacy page', path: '/privacy' }),
  Object.freeze({ label: 'terms page', path: '/terms' }),
  Object.freeze({ label: 'subscription confirmation page', path: '/subscriptions/confirm' }),
  Object.freeze({ label: 'subscription unsubscribe page', path: '/subscriptions/unsubscribe' })
]);
const DASHBOARD_PREVIEW_CASES = Object.freeze([
  Object.freeze({ label: 'widget test page', path: '/app/widget-test' }),
  Object.freeze({ label: 'subscribe test page', path: '/app/subscribe-test' }),
  Object.freeze({ label: 'traffic test page', path: '/app/traffic-test' })
]);

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} path
 * @param {{ silentBootstrap?: boolean, delayMs?: number, bootstrapDelayMs?: number, currentUserDelayMs?: number }} [tauthOptions]
 * @param {{ waitForHeaderAuth?: boolean, waitUntil?: 'commit' | 'domcontentloaded' | 'load' | 'networkidle' }} [options]
 * @returns {Promise<void>}
 */
async function openPageWithoutSession(page, path, tauthOptions, options) {
  const resolvedOptions = options || {};
  await installSiteWidgetConfigStub(page);
  await openPublicPage(page, config, path, {
    tauth: tauthOptions,
    waitForHeaderAuth: resolvedOptions.waitForHeaderAuth,
    waitUntil: resolvedOptions.waitUntil
  });
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} path
 * @param {{ silentBootstrap?: boolean, delayMs?: number, bootstrapDelayMs?: number, currentUserDelayMs?: number }} [tauthOptions]
 * @param {{ waitForHeaderAuth?: boolean, waitUntil?: 'commit' | 'domcontentloaded' | 'load' | 'networkidle' }} [options]
 * @returns {Promise<void>}
 */
async function openPageWithSession(page, path, tauthOptions, options) {
  const resolvedOptions = options || {};
  await installSiteWidgetConfigStub(page);
  await openAuthenticatedPage(page, config, adminUser, path, {
    tauth: tauthOptions,
    waitForHeaderAuth: resolvedOptions.waitForHeaderAuth,
    waitUntil: resolvedOptions.waitUntil
  });
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} path
 * @returns {Promise<void>}
 */
async function openPublicPageForAssetInspection(page, path) {
  await installAssetInspectionStubs(page);
  await openPageWithoutSession(page, path, undefined, { waitForHeaderAuth: false });
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} path
 * @returns {Promise<void>}
 */
async function openAuthenticatedPageForAssetInspection(page, path) {
  await installAssetInspectionStubs(page);
  await openPageWithSession(page, path, undefined, { waitForHeaderAuth: false });
}

/**
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<void>}
 */
async function installSiteWidgetConfigStub(page) {
  const escapedSiteId = SITE_WIDGET_SITE_ID.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  await page.route(new RegExp(`/public/widget-config\\?site_id=${escapedSiteId}$`), async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json; charset=utf-8',
      body: JSON.stringify({
        site_id: SITE_WIDGET_SITE_ID,
        widget_bubble_side: 'right',
        widget_bubble_bottom_offset: 16
      })
    });
  });
}

/**
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<void>}
 */
async function expectLatestCdnAssets(page) {
  await expect
    .poll(() =>
      page.evaluate(() => {
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
      })
    )
    .toEqual({
      styleHref: MPR_UI_STYLE_URL,
      tauthSrc: TAUTH_SCRIPT_URL,
      mprUiSrc: MPR_UI_SCRIPT_URL,
      vendorUrls: []
    });
}

/**
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<void>}
 */
async function expectFooterUtilityLinks(page) {
  const footerLayout = page.locator('mpr-footer [data-mpr-footer="layout"]');
  await expect(footerLayout.locator('[data-mpr-footer="horizontal-links"] a')).toHaveCount(2);
  await expect(footerLayout.locator('[data-mpr-footer="menu"] a')).toHaveCount(10);

  const horizontalLinkLabels = await footerLayout
    .locator('[data-mpr-footer="horizontal-links"] a')
    .allTextContents();
  const menuLinkLabels = await footerLayout
    .locator('[data-mpr-footer="menu"] a')
    .allTextContents();

  await expect(footerLayout.locator('[data-mpr-footer="privacy-link"]')).toHaveText('Privacy');
  await expect(footerLayout.locator('[data-mpr-footer="horizontal-links"] a')).toHaveText([
    'Terms of Service',
    'Pricing'
  ]);
  expect(horizontalLinkLabels).toEqual(['Terms of Service', 'Pricing']);
  expect(menuLinkLabels).not.toContain('Terms of Service');
  expect(menuLinkLabels).not.toContain('Pricing');

  const ordering = await footerLayout.evaluate((layoutElement) => {
    const privacyLink = layoutElement.querySelector('[data-mpr-footer="privacy-link"]');
    const horizontalLinks = layoutElement.querySelector('[data-mpr-footer="horizontal-links"]');
    const toggleButton = layoutElement.querySelector('[data-mpr-footer="toggle-button"]');
    return {
      privacyBeforeHorizontal: Boolean(
        privacyLink &&
          horizontalLinks &&
          (privacyLink.compareDocumentPosition(horizontalLinks) & Node.DOCUMENT_POSITION_FOLLOWING)
      ),
      horizontalBeforeToggle: Boolean(
        horizontalLinks &&
          toggleButton &&
          (horizontalLinks.compareDocumentPosition(toggleButton) & Node.DOCUMENT_POSITION_FOLLOWING)
      )
    };
  });

  expect(ordering.privacyBeforeHorizontal).toBe(true);
  expect(ordering.horizontalBeforeToggle).toBe(true);
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {{ email: string, displayName: string, avatarUrl: string, userId: string }} user
 * @returns {Promise<void>}
 */
async function seedRuntimeAuthenticatedUser(page, user) {
  await page.evaluate((resolvedUser) => {
    const runtime = window['__loopawareTestTauthRuntime'];
    if (!runtime || typeof runtime !== 'object') {
      throw new Error('tauth runtime not found');
    }
    runtime.profile = {
      user_id: String(resolvedUser.userId || ''),
      user_email: String(resolvedUser.email || ''),
      email: String(resolvedUser.email || ''),
      display: String(resolvedUser.displayName || resolvedUser.email || ''),
      avatar_url: String(resolvedUser.avatarUrl || ''),
      roles: []
    };
  }, user);
}

/**
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<void>}
 */
async function beginHeaderLoginFlow(page) {
  await expect(page.locator('mpr-header [data-mpr-header="google-signin"]')).toHaveCount(1);
  await page.evaluate(() => {
    const signinTarget = document.querySelector('mpr-header [data-mpr-header="google-signin"]');
    if (!signinTarget) {
      throw new Error('google sign-in target not found');
    }
    signinTarget.dispatchEvent(new MouseEvent('click', {
      bubbles: true,
      cancelable: true,
      composed: true
    }));
  });
  await expect
    .poll(() =>
      page.evaluate(() => {
        const authStore = window['__loopawareHeaderAuthStore'];
        return !!(authStore && authStore.loginRedirectPending === true);
      })
    )
    .toBe(true);
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

test('login page keeps public content visible while sign-in is still pending', async ({ page }) => {
  await openPageWithoutSession(page, '/login', { currentUserDelayMs: 1000 });
  await beginHeaderLoginFlow(page);

  const waitingScreen = page.locator('#loopaware-public-auth-screen');
  await expect(waitingScreen).toBeHidden();
  await expect(page.getByRole('heading', { level: 1, name: /Privacy-first feedback widget and traffic analytics for developers/i })).toBeVisible();

  await expect
    .poll(() =>
      page.evaluate(() => {
        const transition = document.querySelector('mpr-header [data-mpr-header="auth-transition"]');
        if (!transition) {
          return false;
        }
        return window.getComputedStyle(transition).display !== 'none';
      })
    )
    .toBe(false);
});

test('login page keeps public content visible after a canceled sign-in click', async ({ page }) => {
  await openPageWithoutSession(page, '/login');
  await beginHeaderLoginFlow(page);

  const waitingScreen = page.locator('#loopaware-public-auth-screen');
  await expect(waitingScreen).toBeHidden();
  await expect(page.getByRole('heading', { level: 1, name: /Privacy-first feedback widget and traffic analytics for developers/i })).toBeVisible();
});

test('dashboard keeps the auth transition visible until the authenticated UI finishes loading', async ({ page }) => {
  /** @type {(value?: unknown) => void} */
  let releaseSitesResponse = () => {};
  const sitesResponseGate = new Promise((resolve) => {
    releaseSitesResponse = resolve;
  });

  await page.route(/\/api\/sites(?:\?.*)?$/, async (route) => {
    if (route.request().method() !== 'GET') {
      await route.fallback();
      return;
    }
    await sitesResponseGate;
    await route.fallback();
  });

  await openPageWithSession(
    page,
    '/app',
    { currentUserDelayMs: 150 },
    { waitForHeaderAuth: false, waitUntil: 'domcontentloaded' }
  );

  const transition = page.locator('mpr-header [data-mpr-header="auth-transition"]');
  const transitionTitle = page.locator('mpr-header [data-mpr-header="auth-transition-title"]');
  const transitionMessage = page.locator('mpr-header [data-mpr-header="auth-transition-message"]');

  await expect(transition).toHaveAttribute('data-mpr-visible', 'true');
  await expect(transitionTitle).toHaveText('Opening LoopAware');
  await expect(transitionMessage).toHaveText('Loading your authenticated workspace.');

  await page.waitForTimeout(250);
  await expect(transition).toHaveAttribute('data-mpr-visible', 'true');

  releaseSitesResponse();

  await expect(transition).toHaveAttribute('data-mpr-visible', 'false');
  await expect(page.locator('#user-name')).not.toHaveText('');
});

test('dashboard hides the auth transition after a normal authenticated boot', async ({ page }) => {
  await openPageWithSession(
    page,
    '/app',
    undefined,
    { waitForHeaderAuth: false, waitUntil: 'domcontentloaded' }
  );

  const transition = page.locator('mpr-header [data-mpr-header="auth-transition"]');

  await expect(transition).toHaveAttribute('data-mpr-visible', 'false');
  await expect(page.locator('#user-name')).not.toHaveText('');
});

for (const { label, path } of PUBLIC_LOGIN_ENTRY_CASES) {
  test(`${label} redirects to the dashboard after login flow auth completion`, async ({ page }) => {
    await openPageWithoutSession(page, path);
    await expect(page.locator('mpr-header')).toHaveAttribute('data-loopaware-auth-bound', 'true');
    await beginHeaderLoginFlow(page);

    await Promise.all([
      page.waitForURL(/\/app\/?$/),
      page.evaluate(() => {
        const headerHost = document.querySelector('mpr-header');
        if (!headerHost) {
          throw new Error('mpr-header not found');
        }
        headerHost.dispatchEvent(new CustomEvent('mpr-ui:auth:authenticated'));
      })
    ]);
  });
}

for (const { label, path } of PUBLIC_LOGIN_ENTRY_CASES) {
  test(`${label} redirects to the dashboard after login flow runtime recovery`, async ({ page }) => {
    await openPageWithoutSession(page, path);
    await expect(page.locator('mpr-header')).toHaveAttribute('data-loopaware-auth-bound', 'true');
    await beginHeaderLoginFlow(page);
    await seedRuntimeAuthenticatedUser(page, adminUser);

    await Promise.all([
      page.waitForURL(/\/app\/?$/),
      page.evaluate(() => {
        window.dispatchEvent(new Event('focus'));
      })
    ]);
  });
}

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

test('login page loads latest CDN assets for auth UI', async ({ page }) => {
  await openPublicPageForAssetInspection(page, '/login');
  await expectLatestCdnAssets(page);
});

for (const { label, path } of PUBLIC_LOGIN_ENTRY_CASES) {
  test(`${label} loads latest CDN assets for auth UI`, async ({ page }) => {
    await openPublicPageForAssetInspection(page, path);
    await expectLatestCdnAssets(page);
  });
}

test('public pages render privacy separately and inline utility links before the toggle', async ({ page }) => {
  await openPageWithoutSession(page, '/login');
  await expectFooterUtilityLinks(page);

  for (const { path } of PUBLIC_LOGIN_ENTRY_CASES) {
    await page.goto(path, { waitUntil: 'domcontentloaded' });
    await expectFooterUtilityLinks(page);
  }
});

test('dashboard footer renders privacy separately and inline utility links before the toggle', async ({ page }) => {
  await openPageWithSession(page, '/app');
  await expectFooterUtilityLinks(page);
});

test('login page does not bootstrap the landing widget when runtime widget site is unset', async ({ page }) => {
  const widgetRequests = [];
  page.on('request', (request) => {
    const url = request.url();
    if (url.includes('/widget.js') || url.includes('/public/widget-config')) {
      widgetRequests.push(url);
    }
  });

  await openPageWithoutSession(page, '/login');
  await expect(page.locator('mpr-header > header.mpr-header')).toBeVisible();
  await expect(page.locator('mpr-footer footer.mpr-footer')).toBeVisible();
  await expect(page.locator('#mp-feedback-bubble')).toHaveCount(0);
  expect(widgetRequests).toEqual([]);
  expect(await page.evaluate(() => String(window['__LOOPAWARE_SITE_WIDGET_SITE_ID__'] || ''))).toBe('');
});

test('login page bootstraps the landing widget when runtime widget site is configured', async ({ page }) => {
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

  await openPageWithoutSession(page, `/login?site_widget_site_id=${encodeURIComponent(SITE_WIDGET_SITE_ID)}`);
  await expect(page.locator('mpr-header > header.mpr-header')).toBeVisible();
  await expect(page.locator('mpr-footer footer.mpr-footer')).toBeVisible();
  await expect(page.locator('#mp-feedback-bubble')).toBeVisible();

  expect(await page.evaluate(() => String(window['__LOOPAWARE_SITE_WIDGET_SITE_ID__'] || ''))).toBe(SITE_WIDGET_SITE_ID);
  expect(widgetRequests.some((url) => url.includes(`/widget.js?site_id=${SITE_WIDGET_SITE_ID}`))).toBe(true);
  expect(widgetRequests.some((url) => url.includes('api_origin='))).toBe(false);
  expect(widgetRequests.some((url) => url.includes(`/public/widget-config?site_id=${SITE_WIDGET_SITE_ID}`))).toBe(true);
  expect(
    consoleErrors.filter((message) => message.includes('widget.js: initialize_failed') || message.includes('widget_config_forbidden'))
  ).toHaveLength(0);
});

test('dashboard bootstraps the site widget when runtime widget site is configured', async ({ page }) => {
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

  await openPageWithSession(page, `/app?site_widget_site_id=${encodeURIComponent(SITE_WIDGET_SITE_ID)}`);
  await expect(page.locator('#mp-feedback-bubble')).toBeVisible();

  expect(await page.evaluate(() => String(window['__LOOPAWARE_SITE_WIDGET_SITE_ID__'] || ''))).toBe(SITE_WIDGET_SITE_ID);
  expect(widgetRequests.some((url) => url.includes(`/widget.js?site_id=${SITE_WIDGET_SITE_ID}`))).toBe(true);
  expect(widgetRequests.some((url) => url.includes('api_origin='))).toBe(false);
  expect(widgetRequests.some((url) => url.includes(`/public/widget-config?site_id=${SITE_WIDGET_SITE_ID}`))).toBe(true);
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

for (const { label, path } of PUBLIC_LOGIN_ENTRY_CASES) {
  test(`${label} keeps authenticated users on the current page until login flow starts`, async ({ page }) => {
    const consoleErrors = [];
    page.on('console', (message) => {
      if (message.type() === 'error') {
        consoleErrors.push(message.text());
      }
    });

    await openPageWithSession(page, path);
    await expect(page.locator('mpr-header')).toHaveAttribute('data-loopaware-auth-state', 'authenticated');
    await expect(page.locator('mpr-header > header.mpr-header')).toHaveClass(/mpr-header--authenticated/);
    await expect(page.locator('mpr-user[data-loopaware-user-menu="true"]')).toHaveAttribute('data-mpr-user-status', 'authenticated');
    await expect(page.locator('mpr-header [data-mpr-header="google-signin"]')).toBeHidden();
    await expect(page.locator('mpr-user[data-loopaware-user-menu="true"]')).toBeVisible();
    await expect
      .poll(() =>
        page.evaluate(() => {
          const currentPath = window.location.pathname.replace(/\/$/, '');
          return currentPath || '/';
        })
      )
      .toBe(path);
    expect(consoleErrors.filter((message) => message.includes('mpr-ui.tenant_id_required'))).toHaveLength(0);
  });
}

test('privacy page shows logout overlay for static-page sign-out', async ({ page }) => {
  await openPageWithSession(page, '/privacy');
  await page.evaluate(() => {
    document.dispatchEvent(new CustomEvent('mpr-user:logout'));
  });
  await expect(page.locator('#logout-overlay')).toBeVisible();
  await expect(page.locator('body')).toHaveClass(/logging-out/);
});

test('dashboard loads latest CDN assets for auth UI', async ({ page }) => {
  await openAuthenticatedPageForAssetInspection(page, '/app');
  await expectLatestCdnAssets(page);
});

for (const { label, path } of DASHBOARD_PREVIEW_CASES) {
  test(`${label} loads latest CDN assets for auth UI`, async ({ page }) => {
    await openAuthenticatedPageForAssetInspection(page, path);
    await expectLatestCdnAssets(page);
  });
}

test('vendored mpr-ui asset URLs are not served', async ({ page }) => {
  const scriptResponse = await page.request.get('/vendor/mpr-ui/mpr-ui.js');
  const styleResponse = await page.request.get('/vendor/mpr-ui/mpr-ui.css');
  expect(scriptResponse.status()).toBe(404);
  expect(styleResponse.status()).toBe(404);
});
