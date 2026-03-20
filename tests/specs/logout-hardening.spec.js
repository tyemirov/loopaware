// @ts-check
import { test, expect } from '@playwright/test';
import { setLocalStorage } from '../helpers/browser.js';
import { resolveTestConfig } from '../helpers/config.js';
import { buildAdminUser, openDashboard } from '../helpers/fixtures.js';
import { installTauthStub } from '../helpers/tauthStub.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);
const EXTERNAL_SCRIPT_URLS = Object.freeze([
  'https://cdn.jsdelivr.net/npm/js-yaml@4.1.0/dist/js-yaml.min.js',
  'https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/js/bootstrap.bundle.min.js',
  'https://accounts.google.com/gsi/client'
]);
const LOGIN_URL_PATTERN = /\/login(?:\/)?(?:[?#].*)?$/;
const PUBLIC_PAGE_UNAUTH_CASES = Object.freeze([
  { name: 'login page', path: '/login' },
  { name: 'privacy page', path: '/privacy' },
  { name: 'confirmation page', path: '/subscriptions/confirm' },
  { name: 'unsubscribe page', path: '/subscriptions/unsubscribe' }
]);

/**
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<void>}
 */
async function installExternalScriptStubs(page) {
  for (const scriptUrl of EXTERNAL_SCRIPT_URLS) {
    await page.route(scriptUrl, async (route) => {
      let body = '';
      if (scriptUrl.includes('js-yaml')) {
        body = `window.jsyaml = {
  load: function() {
    return {
      environments: [
        {
          name: 'production',
          hostnames: ['loopaware.mprlab.com', 'tyemirov.github.io'],
          services: {
            apiOrigin: 'https://loopaware-api.mprlab.com',
            tauthOrigin: 'https://tauth-api.mprlab.com'
          }
        },
        {
          name: 'development',
          hostnames: ['localhost', '127.0.0.1'],
          services: {
            apiOrigin: '',
            tauthOrigin: ''
          }
        }
      ]
    };
  }
};`;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/javascript; charset=utf-8',
        body
      });
    });
  }
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} path
 * @param {Record<string, string> | undefined} localStorageEntries
 * @returns {Promise<void>}
 */
async function openPublicPage(page, path, localStorageEntries) {
  await installExternalScriptStubs(page);
  await installTauthStub(page, config);
  if (localStorageEntries) {
    await setLocalStorage(page, localStorageEntries);
  }
  await page.goto(path, { waitUntil: 'domcontentloaded' });
}

/**
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<boolean>}
 */
async function expectLogoutOverlayOrRedirect(page) {
  const overlay = page.locator('#logout-overlay');
  let redirectedToLogin = false;
  await Promise.any([
    page.waitForURL(LOGIN_URL_PATTERN, { timeout: 15_000 }).then(() => {
      redirectedToLogin = true;
    }),
    expect(overlay).toBeVisible({ timeout: 15_000 })
  ]);
  return redirectedToLogin || LOGIN_URL_PATTERN.test(page.url());
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
  await openDashboard(page, config, adminUser);
  
  // Wait for auth listeners to be attached
  await expect(page.locator('mpr-header')).toHaveAttribute('data-loopaware-auth-bound', 'true');

  // Verify content is visible initially
  await expect(page.locator('main')).toBeVisible();
  await expect(page.locator('#logout-overlay')).toBeHidden();

  // Trigger logout event
  await page.evaluate(() => {
    document.dispatchEvent(new CustomEvent('mpr-user:logout'));
  });

  const redirectedToLogin = await expectLogoutOverlayOrRedirect(page);
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

test('logout overlay appears and content hides on unauthenticated event', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  
  // Wait for auth listeners to be attached
  await expect(page.locator('mpr-header')).toHaveAttribute('data-loopaware-auth-bound', 'true');

  // Trigger unauthenticated event from the bound header host
  await page.evaluate(() => {
    const headerHost = document.querySelector('mpr-header');
    const target = headerHost || document;
    target.dispatchEvent(new CustomEvent('mpr-ui:auth:unauthenticated'));
  });

  const redirectedToLogin = await expectLogoutOverlayOrRedirect(page);
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
  await openDashboard(page, config, adminUser);
  
  // Wait for session timeout manager to be started
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
  
  // Click confirm (Yes)
  await page.locator('#session-timeout-confirm-button').click();

  const redirectedToLogin = await expectLogoutOverlayOrRedirect(page);
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
  await openDashboard(page, config, adminUser);
  
  // Wait for auth listeners to be attached
  await expect(page.locator('mpr-header')).toHaveAttribute('data-loopaware-auth-bound', 'true');

  // Intercept the fetch call to /auth/logout so it doesn't actually redirect yet
  await page.route('**/auth/logout', async route => {
    // Just hang or delay
    await new Promise(resolve => setTimeout(resolve, 5000));
  });

  // Call window.logout()
  await page.evaluate(() => {
    const win = /** @type {any} */ (window);
    if (typeof win.logout === 'function') {
      win.logout();
    }
  });

  // Verify overlay is visible immediately even while fetch is "pending"
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
  await openDashboard(page, config, adminUser);

  await expect(page.locator('mpr-header')).toHaveAttribute('data-loopaware-auth-bound', 'true');

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
