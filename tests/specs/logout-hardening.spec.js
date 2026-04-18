// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import {
  buildAdminUser,
  openDashboardShell,
  openPublicPage as openSharedPublicPage,
  waitForLogoutOverlayOrRedirect
} from '../helpers/fixtures.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);
const STALE_AUTH_AFTER_LOGOUT_FLAG = '__loopawareTestStaleAuthAfterLogout';
const AUTH_TRANSITION_SEEN_FLAG = '__loopawareTestAuthTransitionSeen';
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
    document.dispatchEvent(new CustomEvent('mpr-user:logout'));
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

test('explicit logout does not reopen the auth transition modal on login', async ({ page }) => {
  await page.addInitScript((resolvedUser) => {
    try {
      localStorage.removeItem('__loopawareTestStaleAuthAfterLogout');
      localStorage.removeItem('__loopawareTestAuthTransitionSeen');
    } catch (error) {}

    /** @type {typeof Element.prototype.setAttribute & { __loopawareLogoutTransitionRecorder?: boolean }} */
    var originalSetAttribute = Element.prototype.setAttribute;
    if (originalSetAttribute && originalSetAttribute.__loopawareLogoutTransitionRecorder !== true) {
      /** @type {typeof Element.prototype.setAttribute & { __loopawareLogoutTransitionRecorder?: boolean }} */
      var wrappedSetAttribute = function(name, value) {
        var result = originalSetAttribute.apply(this, arguments);
        try {
          if (
            name === 'data-mpr-visible' &&
            value === 'true' &&
            this &&
            typeof this.getAttribute === 'function' &&
            this.getAttribute('data-mpr-header') === 'auth-transition'
          ) {
            localStorage.setItem('__loopawareTestAuthTransitionSeen', 'true');
          }
        } catch (error) {}
        return result;
      };
      wrappedSetAttribute.__loopawareLogoutTransitionRecorder = true;
      Element.prototype.setAttribute = wrappedSetAttribute;
    }

    Object.defineProperty(window, 'getCurrentUser', {
      configurable: true,
      enumerable: true,
      get: function() {
        return undefined;
      },
      set: function(value) {
        if (typeof value !== 'function') {
          Object.defineProperty(window, 'getCurrentUser', {
            configurable: true,
            enumerable: true,
            writable: true,
            value: value
          });
          return;
        }
        var wrappedGetCurrentUser = function() {
          var normalizedPath = window.location && typeof window.location.pathname === 'string'
            ? window.location.pathname.replace(/\/$/, '')
            : '';
          if (normalizedPath === '/login' && localStorage.getItem('__loopawareTestStaleAuthAfterLogout') === 'true') {
            return Promise.resolve({
              user_id: String(resolvedUser.userId || ''),
              user_email: String(resolvedUser.email || ''),
              email: String(resolvedUser.email || ''),
              display: String(resolvedUser.displayName || resolvedUser.email || ''),
              avatar_url: String(resolvedUser.avatarUrl || ''),
              roles: []
            });
          }
          return value.apply(this, arguments);
        };
        Object.defineProperty(window, 'getCurrentUser', {
          configurable: true,
          enumerable: true,
          writable: true,
          value: wrappedGetCurrentUser
        });
      }
    });
  }, adminUser);

  await openDashboardShell(page, config, adminUser);

  await page.evaluate(() => {
    localStorage.setItem('__loopawareTestStaleAuthAfterLogout', 'true');
    localStorage.removeItem('__loopawareTestAuthTransitionSeen');
    document.dispatchEvent(new CustomEvent('mpr-user:logout'));
    Promise.resolve(typeof window.logout === 'function' ? window.logout() : undefined)
      .catch(() => undefined)
      .finally(() => {
        window.location.href = '/login';
      });
  });

  await page.waitForTimeout(1200);
  await expect(page).toHaveURL(/\/login\/?$/);
  await expect(page.locator('mpr-header [data-mpr-header="auth-transition"]')).toHaveAttribute('data-mpr-visible', 'false');
  await expect
    .poll(() =>
      page.evaluate(() => localStorage.getItem('__loopawareTestAuthTransitionSeen'))
    )
    .not.toBe('true');
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

test('manual window.logout() call triggers overlay', async ({ page }) => {
  await openDashboardShell(page, config, adminUser);

  await page.route('**/auth/logout', async route => {
    await new Promise(resolve => setTimeout(resolve, 5000));
  });

  await page.evaluate(() => {
    const win = /** @type {any} */ (window);
    if (typeof win.logout === 'function') {
      win.logout();
    }
  });

  await expect(page.locator('#logout-overlay')).toBeVisible();
  await page.waitForTimeout(500);
  await expect(page.locator('#logout-overlay')).toBeVisible();
  
  const isMainHidden = await page.evaluate(() => {
    const main = document.querySelector('main');
    return main && window.getComputedStyle(main).display === 'none';
  });
  expect(isMainHidden).toBe(true);
});

test('logout failure clears overlay and restores dashboard content', async ({ page }) => {
  await openDashboardShell(page, config, adminUser);

  await page.route('**/auth/logout', async (route) => {
    await route.fulfill({
      status: 500,
      contentType: 'application/json; charset=utf-8',
      body: JSON.stringify({ error: 'logout failed' })
    });
  });

  await page.evaluate(async () => {
    const win = /** @type {any} */ (window);
    if (typeof win.logout === 'function') {
      await Promise.resolve(win.logout()).catch(() => null);
    }
  });

  await expect(page.locator('#logout-overlay')).toBeHidden();
  await expect(page.locator('body')).not.toHaveClass(/logging-out/);
  await expect(page.locator('main')).toBeVisible();
});
