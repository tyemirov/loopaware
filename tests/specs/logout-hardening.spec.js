// @ts-check
import { test, expect } from '@playwright/test';
import { buildSessionToken } from '../helpers/auth.js';
import { resolveTestConfig } from '../helpers/config.js';
import { enableAutoGoogleCredentialOnClick } from '../helpers/externalAssets.js';
import {
  buildAdminUser,
  openAuthenticatedPage,
  openDashboardShell,
  openPublicPage as openSharedPublicPage,
  waitForDashboardAccountHydrated,
  waitForLogoutOverlayOrRedirect
} from '../helpers/fixtures.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);
const PUBLIC_PAGE_UNAUTH_CASES = Object.freeze([
  { name: 'login page', path: '/login' },
  { name: 'privacy page', path: '/privacy' },
  { name: 'confirmation page', path: '/subscriptions/confirm' },
  { name: 'unsubscribe page', path: '/subscriptions/unsubscribe' }
]);

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} path
 * @param {Record<string, string> | undefined} localStorageEntries
 * @returns {Promise<void>}
 */
async function openPublicPage(page, path, localStorageEntries) {
  await openSharedPublicPage(page, config, path, { localStorage: localStorageEntries });
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
async function openDashboardForSessionTimeoutRecovery(page) {
  await openAuthenticatedPage(page, config, adminUser, '/app', { waitForHeaderAuth: false });
  await page.waitForFunction(() => {
    const headerHost = document.querySelector('mpr-header');
    return !!(headerHost && headerHost.getAttribute('data-loopaware-auth-bound') === 'true');
  });
  await page.evaluate(() => {
    const headerHost = document.querySelector('mpr-header');
    if (!headerHost) {
      throw new Error('loopaware.header_missing');
    }
    headerHost.removeAttribute('data-loopaware-auth-redirect-on-logout');
    headerHost.setAttribute('data-mpr-auth-status', 'authenticated');
    headerHost.dispatchEvent(new CustomEvent('mpr-ui:auth:status-change', {
      bubbles: true,
      detail: { status: 'authenticated' }
    }));
  });
  await waitForDashboardAccountHydrated(page);
  await expect(page.locator('#user-name')).not.toHaveText('');
  await expect
    .poll(() =>
      page.evaluate(() => {
        const hooks = window.__loopawareDashboardIdleTestHooks;
        return !!(hooks && typeof hooks.started === 'function' && hooks.started());
      })
    )
    .toBe(true);
}

test('privacy page initializes theme and auth scripts instead of rendering raw JavaScript', async ({ page }) => {
  await openPublicPage(page, '/privacy', { loopaware_public_theme: 'light' });

  await expect(page.getByRole('heading', { level: 1, name: 'Privacy Policy — LoopAware' })).toBeVisible();
  await expect(page.locator('body')).toHaveAttribute('data-bs-theme', 'light');
  await expect(page.locator('html')).toHaveAttribute('data-loopaware-auth-script', 'true');
  await expect(page.locator('body')).not.toContainText('var publicThemeStorageKey');
  await expect(page.locator('body')).not.toContainText('function showLogoutOverlay');
});

test('confirm page shows friendly missing-token message', async ({ page }) => {
  await openPublicPage(page, '/subscriptions/confirm', undefined);

  await expect(page.locator('#subscription-link-heading')).toHaveText('Subscription confirmation');
  await expect(page.locator('#subscription-link-message')).toHaveText('Missing confirmation token.');
});

test('unsubscribe page shows friendly missing-token message', async ({ page }) => {
  await openPublicPage(page, '/subscriptions/unsubscribe', undefined);

  await expect(page.locator('#subscription-link-heading')).toHaveText('Unsubscribe');
  await expect(page.locator('#subscription-link-message')).toHaveText('Missing unsubscribe token.');
});

for (const publicPageCase of PUBLIC_PAGE_UNAUTH_CASES) {
  test(`${publicPageCase.name} keeps content visible on unauthenticated events`, async ({ page }) => {
    await openPublicPage(page, publicPageCase.path, undefined);

    await expect(page.locator('mpr-header')).toHaveAttribute('data-loopaware-auth-bound', 'true');
    await expect(page.locator('main')).toBeVisible();
    await expect(page.locator('#logout-overlay')).toBeHidden();

    await page.evaluate(() => {
      const headerHost = document.querySelector('mpr-header');
      const target = headerHost || document;
      target.dispatchEvent(new CustomEvent('mpr-ui:auth:unauthenticated'));
    });

    await expect(page.locator('main')).toBeVisible();
    await expect(page.locator('#logout-overlay')).toBeHidden();
    await expect(page.locator('body')).not.toHaveClass(/logging-out/);
  });
}

test('logout overlay appears and content hides on logout event', async ({ page }) => {
  await openDashboardShell(page, config, adminUser);

  await expect(page.locator('main')).toBeVisible();
  await expect(page.locator('#logout-overlay')).toBeHidden();

  await page.evaluate(() => {
    const headerHost = document.querySelector('mpr-header');
    if (!headerHost) {
      throw new Error('loopaware.header_missing');
    }
    document.dispatchEvent(new CustomEvent('mpr-user:logout'));
    headerHost.setAttribute('data-mpr-auth-status', 'authenticated');
    headerHost.dispatchEvent(new CustomEvent('mpr-ui:auth:status-change', {
      bubbles: true,
      detail: { status: 'authenticated' }
    }));
  });

  const redirectedToLogin = await waitForLogoutOverlayOrRedirect(page);
  if (redirectedToLogin) {
    return;
  }

  try {
    const isMainHidden = await page.evaluate(() => {
      const main = document.querySelector('main');
      if (!main) return true; // already gone
      return window.getComputedStyle(main).display === 'none';
    });
    expect(isMainHidden).toBe(true);
  } catch (e) {
    if (!e.message.includes('context was destroyed')) throw e;
  }
});

test('explicit logout event does not reopen the auth transition modal on login', async ({ page }) => {
  await openDashboardShell(page, config, adminUser);

  await page.evaluate(() => {
    document.dispatchEvent(new CustomEvent('mpr-user:logout'));
    window.location.href = '/login';
  });

  await expect(page).toHaveURL(/\/login\/?$/);
  await expect(page.locator('mpr-header [data-mpr-header="auth-transition"]')).toHaveAttribute('data-mpr-visible', 'false');
  await expect(page.locator('main')).toBeVisible();
});

test('public auth signs in through mpr-ui without app-owned auth globals', async ({ page }) => {
  await openSharedPublicPage(page, config, '/pricing', {
    tauth: { sessionCookieValue: buildLoginSessionCookieValue(adminUser) }
  });
  await enableAutoGoogleCredentialOnClick(page);

  const signInButton = page
    .locator('mpr-header button[data-test="google-signin"]:not([data-mpr-google-wrapper="true"])')
    .first();
  await expect(signInButton).toBeVisible();
  expect(
    await page.evaluate(() =>
      ['apiFetch', 'exchangeGoogleCredential', 'getAuthEndpoints', 'getCurrentUser', 'initAuthClient', 'logout', 'requestNonce', 'setAuthTenantId']
        .filter((key) => typeof window[key] === 'function')
    )
  ).toEqual([]);
  await signInButton.click();

  await page.waitForURL(/\/app\/?$/);
  expect(
    await page.evaluate(() =>
      ['apiFetch', 'exchangeGoogleCredential', 'getAuthEndpoints', 'getCurrentUser', 'initAuthClient', 'logout', 'requestNonce', 'setAuthTenantId']
        .filter((key) => typeof window[key] === 'function')
    )
  ).toEqual([]);
});

test('logout overlay appears and content hides on unauthenticated event', async ({ page }) => {
  await openDashboardShell(page, config, adminUser);

  await page.evaluate(() => {
    const headerHost = document.querySelector('mpr-header');
    const target = headerHost || document;
    target.dispatchEvent(new CustomEvent('mpr-ui:auth:unauthenticated'));
  });

  const redirectedToLogin = await waitForLogoutOverlayOrRedirect(page);
  if (redirectedToLogin) {
    return;
  }

  try {
    const isMainHidden = await page.evaluate(() => {
      const main = document.querySelector('main');
      if (!main) return true; // already gone
      return window.getComputedStyle(main).display === 'none';
    });
    expect(isMainHidden).toBe(true);
  } catch (e) {
    if (!e.message.includes('context was destroyed')) throw e;
  }
});

test('logout overlay appears and content hides on session timeout confirm', async ({ page }) => {
  await openDashboardShell(page, config, adminUser);

  await expect(async () => {
    const started = await page.evaluate(() => {
      const win = /** @type {any} */ (window);
      return !!(win.__loopawareDashboardIdleTestHooks && win.__loopawareDashboardIdleTestHooks.started());
    });
    expect(started).toBe(true);
  }).toPass();

  // Force the session timeout prompt
  await page.evaluate(() => {
    const win = /** @type {any} */ (window);
    if (win.__loopawareDashboardIdleTestHooks) {
      win.__loopawareDashboardIdleTestHooks.forcePrompt();
    }
  });

  await expect(page.locator('#session-timeout-notification')).toBeVisible();
  await page.locator('#session-timeout-confirm-button').click();

  const redirectedToLogin = await waitForLogoutOverlayOrRedirect(page);
  if (redirectedToLogin) {
    return;
  }

  try {
    const isMainHidden = await page.evaluate(() => {
      const main = document.querySelector('main');
      if (!main) return true; // already gone
      return window.getComputedStyle(main).display === 'none';
    });
    expect(isMainHidden).toBe(true);
  } catch (e) {
    // If navigation happened during evaluate, ignore it
    if (!e.message.includes('context was destroyed')) throw e;
  }
});

test('dashboard does not expose app-owned window.logout helper', async ({ page }) => {
  await openDashboardShell(page, config, adminUser);

  expect(await page.evaluate(() => typeof window['logout'])).toBe('undefined');
});

test('session timeout logout failure keeps the authenticated dashboard session', async ({ page }) => {
  await openDashboardForSessionTimeoutRecovery(page);

  let logoutRequests = 0;
  await page.route('**/auth/logout', async (route) => {
    logoutRequests += 1;
    await route.fulfill({
      status: 500,
      contentType: 'application/json; charset=utf-8',
      body: JSON.stringify({ error: 'logout failed' })
    });
  });

  await page.evaluate(() => {
    const win = /** @type {any} */ (window);
    if (win.__loopawareDashboardIdleTestHooks) {
      win.__loopawareDashboardIdleTestHooks.forcePrompt();
    }
  });
  await expect(page.locator('#session-timeout-notification')).toBeVisible();
  await page.locator('#session-timeout-confirm-button').click();

  await expect.poll(() => logoutRequests).toBe(1);
  await expect(page).toHaveURL(/\/app\/?$/);
  await expect(page.locator('main')).toBeVisible();
  await expect(page.locator('#logout-overlay')).toBeHidden();
  await expect(page.locator('#session-timeout-notification')).toHaveAttribute('aria-hidden', 'true');
  await expect(page.locator('body')).not.toHaveClass(/logging-out/);
  await expect
    .poll(() =>
      page.evaluate(() => {
        const hooks = window.__loopawareDashboardIdleTestHooks;
        return !!(hooks && typeof hooks.started === 'function' && hooks.started());
      })
    )
    .toBe(true);
  await expect(page.locator('#user-name')).not.toHaveText('');
  expect(await page.evaluate(() => typeof window['logout'])).toBe('undefined');
});
