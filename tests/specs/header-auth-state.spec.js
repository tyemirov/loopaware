// @ts-check
import { test, expect } from '@playwright/test';
import { buildSessionToken } from '../helpers/auth.js';
import { resolveTestConfig } from '../helpers/config.js';
import {
  enableAutoGoogleCredentialOnClick,
  getGoogleIdentityInitializedNonce,
  getGoogleIdentityInitializeCallCount,
  installAssetInspectionStubs,
  waitForExternalAssetStubsToSettle,
  waitForGoogleIdentityStubInitialized
} from '../helpers/externalAssets.js';
import { buildAdminUser, openAuthenticatedPage, openPublicPage, waitForDashboardReady } from '../helpers/fixtures.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);
const MPR_UI_VERSION = 'latest';
const MPR_UI_STYLE_URL = `https://cdn.jsdelivr.net/gh/MarcoPoloResearchLab/mpr-ui@${MPR_UI_VERSION}/mpr-ui.css`;
const MPR_UI_CONFIG_URL = `https://cdn.jsdelivr.net/gh/MarcoPoloResearchLab/mpr-ui@${MPR_UI_VERSION}/mpr-ui-config.js`;
const MPR_UI_SCRIPT_URL = `https://cdn.jsdelivr.net/gh/MarcoPoloResearchLab/mpr-ui@${MPR_UI_VERSION}/mpr-ui.js`;
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
const SHARED_AUTH_HTML_CASES = Object.freeze([
  Object.freeze({ label: 'login page', path: '/login' }),
  Object.freeze({ label: 'dashboard page', path: '/app' }),
  ...PUBLIC_LOGIN_ENTRY_CASES,
  ...DASHBOARD_PREVIEW_CASES
]);
const GOOGLE_IDENTITY_SCRIPT_URL = 'https://accounts.google.com/gsi/client';
const FOUR_HOURS_MS = 4 * 60 * 60 * 1000;

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} path
 * @param {{ silentBootstrap?: boolean, delayMs?: number, bootstrapDelayMs?: number, currentUserDelayMs?: number, exchangeDelayMs?: number, sessionCookieValue?: string }} [tauthOptions]
 * @param {{ authenticateMprUiSession?: boolean, waitForHeaderAuth?: boolean, waitUntil?: 'commit' | 'domcontentloaded' | 'load' | 'networkidle' }} [options]
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
 * @param {{ silentBootstrap?: boolean, delayMs?: number, bootstrapDelayMs?: number, currentUserDelayMs?: number, exchangeDelayMs?: number, sessionCookieValue?: string }} [tauthOptions]
 * @param {{ authenticateMprUiSession?: boolean, waitForHeaderAuth?: boolean, waitUntil?: 'commit' | 'domcontentloaded' | 'load' | 'networkidle' }} [options]
 * @returns {Promise<void>}
 */
async function openPageWithSession(page, path, tauthOptions, options) {
  const resolvedOptions = options || {};
  await installSiteWidgetConfigStub(page);
  await openAuthenticatedPage(page, config, adminUser, path, {
    tauth: tauthOptions,
    authenticateMprUiSession: resolvedOptions.authenticateMprUiSession,
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
  await openPageWithSession(page, path, undefined, {
    authenticateMprUiSession: false,
    waitForHeaderAuth: false
  });
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
      page.evaluate((expectedConfigUrl) => {
        const mprUiStyle = document.getElementById('mpr-ui-style');
        const mprUiBundle = document.getElementById('mpr-ui-bundle');
        const browserAssetUrls = Array.from(document.querySelectorAll('script[src], link[rel="stylesheet"][href]'))
          .map((element) => element.getAttribute('src') || element.getAttribute('href') || '');
        return {
          styleHref: mprUiStyle ? mprUiStyle.getAttribute('href') || '' : '',
          configScriptPresent: browserAssetUrls.includes(expectedConfigUrl),
          bundleSrc: mprUiBundle ? mprUiBundle.getAttribute('data-mpr-ui-bundle-src') || '' : '',
          tauthScriptCount: document.querySelectorAll('script[src*="tauth.js"], #tauth-script').length,
          vendorUrls: browserAssetUrls.filter((url) => url.includes('/vendor/'))
        };
      }, MPR_UI_CONFIG_URL)
    )
    .toEqual({
      styleHref: MPR_UI_STYLE_URL,
      configScriptPresent: true,
      bundleSrc: MPR_UI_SCRIPT_URL,
      tauthScriptCount: 0,
      vendorUrls: []
    });
}

/**
 * @param {import('@playwright/test').APIRequestContext} request
 * @param {string} path
 * @returns {Promise<void>}
 */
async function expectServedHTMLDoesNotLoadGoogleIdentity(request, path) {
  const response = await request.get(new URL(path, config.baseURL).toString());
  expect(response.ok()).toBe(true);
  const html = await response.text();
  expect(html).not.toContain(GOOGLE_IDENTITY_SCRIPT_URL);
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
 * @param {{ email: string, displayName: string, avatarUrl: string, userId: string, issuer?: string }} user
 * @returns {string}
 */
function buildLoginSessionCookieValue(user) {
  return buildSessionToken({
    signingKey: config.signingKey,
    tenantId: config.tenantId,
    email: user.email,
    displayName: user.displayName,
    avatarUrl: user.avatarUrl,
    userId: user.userId,
    issuer: user.issuer
  });
}

/**
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<void>}
 */
async function beginHeaderLoginFlow(page) {
  await expect(page.locator('mpr-header [data-mpr-header="google-signin"]')).toHaveCount(1);
  const signInButton = page
    .locator('mpr-header button[data-test="google-signin"]:not([data-mpr-google-wrapper="true"])')
    .first();
  await expect(signInButton).toBeVisible();
  await signInButton.click();
}

/**
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<void>}
 */
async function expectLandingLoginControls(page) {
  await expect(page.locator('[data-loopaware-dashboard-login="true"]')).toHaveCount(0);
  await expect(page.locator('mpr-login-button')).toHaveCount(0);
  await expect(page.locator('[data-loopaware-landing-login]')).toHaveCount(0);
  await expect(page.locator('mpr-header .mpr-header__actions [data-mpr-header="google-signin"]')).toHaveCount(1);
  await expect(page.locator('mpr-header button[data-test="google-signin"]:not([data-mpr-google-wrapper="true"])')).toHaveCount(1);

  const placement = await page.locator('mpr-header > header.mpr-header').evaluate((headerElement) => {
    const brandElement = headerElement.querySelector('.mpr-header__brand');
    const actionsElement = headerElement.querySelector('.mpr-header__actions');
    const googleElement = headerElement.querySelector('[data-mpr-header="google-signin"]');
    if (!brandElement || !actionsElement || !googleElement) {
      return { googleInActions: false, googleRightOfBrand: false };
    }
    const brandRect = brandElement.getBoundingClientRect();
    const actionsRect = actionsElement.getBoundingClientRect();
    const googleRect = googleElement.getBoundingClientRect();
    return {
      googleInActions: actionsRect.left <= googleRect.left && googleRect.right <= actionsRect.right,
      googleRightOfBrand: googleRect.left > brandRect.right
    };
  });
  expect(placement).toEqual({ googleInActions: true, googleRightOfBrand: true });
}

/**
 * @param {import('@playwright/test').Page} page
 * @returns {import('@playwright/test').Locator}
 */
function headerGoogleSigninButton(page) {
  return page
    .locator('mpr-header button[data-test="google-signin"]:not([data-mpr-google-wrapper="true"])')
    .first();
}

/**
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<void>}
 */
async function beginLoginPageHeaderLoginFlow(page) {
  await expectLandingLoginControls(page);
  const signInButton = headerGoogleSigninButton(page);
  await expect(signInButton).toBeVisible();
  await signInButton.click();
}

/**
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<void>}
 */
async function installPageClockControl(page) {
  await page.addInitScript(() => {
    const win = /** @type {any} */ (window);
    let currentTimeMilliseconds = 1_000_000;
    Date.now = function now() {
      return currentTimeMilliseconds;
    };
    win.__loopawareAdvanceClock = function advanceClock(milliseconds) {
      currentTimeMilliseconds += Number(milliseconds) || 0;
      return currentTimeMilliseconds;
    };
  });
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

test('login page header sign-in reports authenticating while sign-in is still pending', async ({ page }) => {
  await openPageWithoutSession(page, '/login', {
    exchangeDelayMs: 1000,
    sessionCookieValue: buildLoginSessionCookieValue(adminUser)
  });
  await enableAutoGoogleCredentialOnClick(page);
  await beginLoginPageHeaderLoginFlow(page);

  await expect(page.locator('mpr-header')).toHaveAttribute('data-mpr-auth-status', 'authenticating');
});

test('login page keeps public content visible after a canceled sign-in click', async ({ page }) => {
  await openPageWithoutSession(page, '/login');
  await beginLoginPageHeaderLoginFlow(page);

  await expect(
    page.getByRole('heading', {
      level: 1,
      name: /Collect feedback, capture subscribers, catch errors, and understand what visitors are doing from one dashboard/i
    })
  ).toBeVisible();
});

test('login page header sign-in uses one mpr-ui Google control before redirecting to the dashboard', async ({ page }) => {
  await openPageWithoutSession(page, '/login', {
    exchangeDelayMs: 1000,
    sessionCookieValue: buildLoginSessionCookieValue(adminUser)
  });
  await enableAutoGoogleCredentialOnClick(page);
  await expect(page.locator('mpr-header')).toHaveAttribute('data-loopaware-auth-bound', 'true');
  await beginLoginPageHeaderLoginFlow(page);

  await expect(page.locator('mpr-header')).toHaveAttribute('data-mpr-auth-status', 'authenticating');
  await page.waitForURL(/\/app\/?$/);
});

test('login page completed sign-in loads the authenticated dashboard', async ({ page }) => {
  await openPageWithoutSession(page, '/login', {
    sessionCookieValue: buildLoginSessionCookieValue(adminUser)
  });
  await enableAutoGoogleCredentialOnClick(page);

  await beginLoginPageHeaderLoginFlow(page);

  await page.waitForURL(/\/app\/?$/);
  await waitForDashboardReady(page, { allowEmptySites: true });
  await expect(page.locator('mpr-header')).toHaveAttribute('data-loopaware-auth-state', 'authenticated');
  await expect(page.locator('#user-email')).toHaveText(adminUser.email);
});

test('login page signs in after staying loaded for four hours', async ({ page }) => {
  await page.clock.install({ time: new Date('2026-06-05T12:00:00.000Z') });
  await openPageWithoutSession(page, '/login', {
    sessionCookieValue: buildLoginSessionCookieValue(adminUser)
  });
  await enableAutoGoogleCredentialOnClick(page);
  await expect(page.locator('mpr-header')).toHaveAttribute('data-loopaware-auth-bound', 'true');
  await waitForGoogleIdentityStubInitialized(page);
  const loadedPageNonce = await getGoogleIdentityInitializedNonce(page);
  expect(loadedPageNonce).not.toBe('');

  await page.route(/\/auth\/google(?:\?.*)?$/, async (route) => {
    const payload = JSON.parse(route.request().postData() || '{}');
    if (payload && payload.nonce_token === loadedPageNonce) {
      await route.fulfill({
        status: 400,
        contentType: 'application/json; charset=utf-8',
        body: JSON.stringify({ error: 'stale_nonce' })
      });
      return;
    }
    await route.fallback();
  });

  await page.clock.runFor(FOUR_HOURS_MS);
  await expect
    .poll(() =>
      getGoogleIdentityInitializedNonce(page)
        .then((currentNonce) => currentNonce !== '' && currentNonce !== loadedPageNonce)
    )
    .toBe(true);

  await beginLoginPageHeaderLoginFlow(page);

  await page.waitForURL(/\/app\/?$/);
  await waitForDashboardReady(page, { allowEmptySites: true });
  await expect(page.locator('#user-email')).toHaveText(adminUser.email);
});

test('login page recovers after a long-idle Google nonce expires', async ({ page }) => {
  const authGooglePayloads = [];
  await installPageClockControl(page);
  page.on('request', (request) => {
    const url = new URL(request.url());
    if (url.pathname !== '/auth/google') {
      return;
    }
    try {
      authGooglePayloads.push(JSON.parse(request.postData() || '{}'));
    } catch (error) {
      authGooglePayloads.push({});
    }
  });

  await openPageWithoutSession(page, '/login', {
    sessionCookieValue: buildLoginSessionCookieValue(adminUser)
  });
  await enableAutoGoogleCredentialOnClick(page);
  await expect(page.locator('mpr-header')).toHaveAttribute('data-loopaware-auth-bound', 'true');
  await waitForGoogleIdentityStubInitialized(page);
  const staleNonce = await getGoogleIdentityInitializedNonce(page);
  expect(staleNonce).not.toBe('');
  await page.evaluate(() => {
    const win = /** @type {any} */ (window);
    if (typeof win.__loopawareAdvanceClock !== 'function') {
      throw new Error('loopaware.clock_control_missing');
    }
    win.__loopawareAdvanceClock(6 * 60 * 1000);
  });

  await beginLoginPageHeaderLoginFlow(page);
  await expect
    .poll(() =>
      getGoogleIdentityInitializedNonce(page)
        .then((currentNonce) => currentNonce !== staleNonce)
    )
    .toBe(true);
  expect(authGooglePayloads).toEqual([]);
  const freshNonce = await getGoogleIdentityInitializedNonce(page);
  expect(freshNonce).not.toBe('');
  expect(freshNonce).not.toBe(staleNonce);

  await beginLoginPageHeaderLoginFlow(page);

  await page.waitForURL(/\/app\/?$/);
  await waitForDashboardReady(page, { allowEmptySites: true });
  expect(authGooglePayloads).toEqual([
    {
      google_id_token: `stub-google-credential::${freshNonce}`,
      nonce_token: freshNonce
    }
  ]);
  await expect(page.locator('#user-email')).toHaveText(adminUser.email);
});

test('login page completed sign-in retries a transient dashboard API unauthorized response', async ({ page }) => {
  let apiMeRequests = 0;
  let authRefreshRequests = 0;

  page.on('request', (request) => {
    const url = new URL(request.url());
    if (url.pathname === '/auth/refresh') {
      authRefreshRequests += 1;
    }
  });

  await page.route(/\/api\/me(?:\?.*)?$/, async (route) => {
    if (route.request().method() !== 'GET') {
      await route.fallback();
      return;
    }
    apiMeRequests += 1;
    if (apiMeRequests === 1) {
      await route.fulfill({
        status: 401,
        contentType: 'application/json; charset=utf-8',
        body: JSON.stringify({ error: 'unauthorized' })
      });
      return;
    }
    await route.fallback();
  });

  await openPageWithoutSession(page, '/login', {
    sessionCookieValue: buildLoginSessionCookieValue(adminUser)
  });
  await enableAutoGoogleCredentialOnClick(page);

  await beginLoginPageHeaderLoginFlow(page);

  await page.waitForURL(/\/app\/?$/);
  await waitForDashboardReady(page, { allowEmptySites: true });
  await expect(page.locator('#user-email')).toHaveText(adminUser.email);
  expect(apiMeRequests).toBeGreaterThanOrEqual(2);
  expect(authRefreshRequests).toBeGreaterThanOrEqual(1);
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
        headerHost.dispatchEvent(new CustomEvent('mpr-ui:auth:status-change', {
          detail: {
            status: 'authenticating',
            previousStatus: 'unauthenticated'
          },
          bubbles: true
        }));
        headerHost.dispatchEvent(new CustomEvent('mpr-ui:auth:authenticated', { bubbles: true }));
      })
    ]);
  });
}

for (const { label, path } of PUBLIC_LOGIN_ENTRY_CASES) {
  test(`${label} redirects to the dashboard after TAuth credential exchange`, async ({ page }) => {
    await openPageWithoutSession(page, path, {
      sessionCookieValue: buildLoginSessionCookieValue(adminUser)
    });
    await enableAutoGoogleCredentialOnClick(page);
    await expect(page.locator('mpr-header')).toHaveAttribute('data-loopaware-auth-bound', 'true');
    await beginHeaderLoginFlow(page);

    await page.waitForURL(/\/app\/?$/);
  });
}

test('login page renders header while tauth session bootstrap is delayed', async ({ page }) => {
  await openPageWithoutSession(page, '/login', { bootstrapDelayMs: 2500 });
  await expect(page.locator('mpr-header > header.mpr-header')).toBeVisible({ timeout: 2000 });
  await expect(page.locator('mpr-footer footer.mpr-footer')).toBeVisible({ timeout: 2000 });
  await expect(page).toHaveURL(/\/login\/?$/);
});

test('dashboard preserves authenticated session state while TAuth endpoint responses are delayed', async ({ page }) => {
  await openPageWithSession(page, '/app', { delayMs: 2500 });
  await expect(page).toHaveURL(/\/app\/?$/);
  await expect(page.locator('mpr-header')).toHaveAttribute('data-loopaware-auth-state', 'authenticated');
});

test('dashboard does not bounce to login while authenticated session recovery is still settling', async ({ page }) => {
  await openPageWithSession(page, '/app', { currentUserDelayMs: 2500 });
  await expect(page).toHaveURL(/\/app\/?$/);
  await expect(page.locator('mpr-header')).toHaveAttribute('data-loopaware-auth-state', 'authenticated', { timeout: 5000 });
});

test('login page loads latest CDN assets for auth UI', async ({ page }) => {
  await openPublicPageForAssetInspection(page, '/login');
  await expectLatestCdnAssets(page);
});

for (const { label, path } of SHARED_AUTH_HTML_CASES) {
  test(`${label} delegates Google Identity script loading to mpr-ui`, async ({ request }) => {
    await expectServedHTMLDoesNotLoadGoogleIdentity(request, path);
  });
}

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

test('login page keeps TAuth origin query state out of mpr-ui auth controls', async ({ page }) => {
  const tauthOrigin = 'https://tauth.example.test';
  await openPageWithoutSession(page, `/login?tauth_origin=${encodeURIComponent(tauthOrigin)}`);
  await expect(page.locator('mpr-header')).not.toHaveAttribute('tauth-url', tauthOrigin);
  await expect(page.locator('mpr-header')).toHaveAttribute('tauth-tenant-id', 'loopaware');
  await expect(page.locator('mpr-header')).toHaveAttribute('tauth-login-path', '/auth/google');
  await expect(page.locator('mpr-login-button')).toHaveCount(0);
  expect(await page.evaluate(() => String(window['__LOOPAWARE_TAUTH_ORIGIN__'] || ''))).toBe(tauthOrigin);
});

test('login page boots one mpr-ui auth controller without anonymous session probes', async ({ page }) => {
  /** @type {string[]} */
  const authRequests = [];
  page.on('request', (request) => {
    const url = new URL(request.url());
    if (url.pathname === '/me' || url.pathname === '/auth/refresh') {
      authRequests.push(url.pathname);
    }
  });

  await openPageWithoutSession(page, '/login');
  await expectLandingLoginControls(page);
  await waitForExternalAssetStubsToSettle(page);

  await expect
    .poll(() =>
      getGoogleIdentityInitializeCallCount(page)
    )
    .toBe(1);

  expect(authRequests.filter((path) => path === '/me')).toHaveLength(0);
  expect(authRequests.filter((path) => path === '/auth/refresh')).toHaveLength(0);
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
  test(`${label} brand link sends unauthenticated users to the landing page`, async ({ page }) => {
    await openPageWithoutSession(page, path);
    const brandLink = page.locator('mpr-header a[slot="brand"]');
    await expect(brandLink).toHaveAttribute('href', '/login');
    await Promise.all([
      page.waitForURL(/\/login\/?$/, { waitUntil: 'domcontentloaded' }),
      brandLink.click({ noWaitAfter: true })
    ]);
    await waitForExternalAssetStubsToSettle(page);
  });

  test(`${label} brand link sends authenticated users to the dashboard`, async ({ page }) => {
    await openPageWithSession(page, path);
    const brandLink = page.locator('mpr-header a[slot="brand"]');
    await expect(brandLink).toHaveAttribute('href', '/app');
    await Promise.all([
      page.waitForURL(/\/app\/?$/, { waitUntil: 'domcontentloaded' }),
      brandLink.click({ noWaitAfter: true })
    ]);
    await waitForExternalAssetStubsToSettle(page);
  });

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
