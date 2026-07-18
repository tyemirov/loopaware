// @ts-check
import { createSite, listSites } from './api.js';
import { applySessionCookie, setLocalStorage } from './browser.js';
import { installExternalAssetStubs, waitForExternalAssetStubsToSettle } from './externalAssets.js';
import { installTauthStub } from './tauthStub.js';

const DEFAULT_AVATAR_DATA_URL = 'data:image/gif;base64,R0lGODlhAQABAIABAP///wAAACwAAAAAAQABAAACAkQBADs=';
const DEFAULT_LOGOUT_REDIRECT_PATTERN = /\/login(?:\/)?(?:[?#].*)?$/;
const APP_PATHNAME = '/app';
const AUTH_RESTORE_HINT_PREFIX = 'tauth.restore.v1:';
const AUTH_RESTORE_HINT_VALUE = '1';
const LOGIN_PATHNAME = '/login';

function randomSuffix() {
  return `${Date.now().toString(36)}${Math.random().toString(16).slice(2, 8)}`;
}

export function buildUniqueName(prefix) {
  const resolvedPrefix = prefix || 'Test Site';
  return `${resolvedPrefix} ${randomSuffix()}`;
}

export function buildUniqueOrigin(prefix) {
  const normalizedPrefix = prefix
    ? String(prefix).trim().toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-+|-+$)/g, '')
    : '';
  const resolvedPrefix = normalizedPrefix ? `${normalizedPrefix}-` : '';
  return `http://${resolvedPrefix}${randomSuffix()}.example.com`;
}

export function buildUniqueEmail(prefix) {
  const resolvedPrefix = prefix || 'user';
  return `${resolvedPrefix}-${randomSuffix()}@example.com`;
}

export function buildAdminUser(config, overrides) {
  const resolvedOverrides = overrides || {};
  return {
    email: resolvedOverrides.email || config.adminEmail,
    displayName: resolvedOverrides.displayName || config.adminDisplayName,
    avatarUrl: resolvedOverrides.avatarUrl || DEFAULT_AVATAR_DATA_URL,
    issuer: resolvedOverrides.issuer,
    userId: resolvedOverrides.userId || `user-${randomSuffix()}`
  };
}

export function buildBaseOrigin(config) {
  if (config.baseOrigin) {
    return config.baseOrigin;
  }
  return new URL(config.baseURL).origin;
}

export async function createTestSite(config, cookie, overrides) {
  const resolvedOverrides = overrides || {};
  const origin = resolvedOverrides.allowedOrigin || buildBaseOrigin(config);
  const name = resolvedOverrides.name || buildUniqueName('Test Site');
  const ownerEmail = resolvedOverrides.ownerEmail || config.adminEmail;
  return createSite(config, cookie, {
    name,
    allowedOrigin: origin,
    ownerEmail
  });
}

export async function ensureSiteForOrigin(config, cookie, overrides) {
  const resolvedOverrides = overrides || {};
  const origin = resolvedOverrides.allowedOrigin || buildBaseOrigin(config);
  const payload = await listSites(config, cookie);
  const sites = Array.isArray(payload?.sites) ? payload.sites : [];
  const existing = sites.find((entry) => entry.allowed_origin === origin);
  if (existing) {
    return existing;
  }
  return createTestSite(config, cookie, { ...resolvedOverrides, allowedOrigin: origin });
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {{ baseOrigin?: string, baseURL?: string, sessionCookieName?: string }} config
 * @param {{
 *   clipboard?: boolean,
 *   localStorage?: Record<string, string | number | boolean | null | undefined>,
 *   tauth?: {
 *     silentBootstrap?: boolean,
 *     delayMs?: number,
 *     bootstrapDelayMs?: number,
 *     currentUserDelayMs?: number,
 *     exchangeDelayMs?: number,
 *     sessionCookieValue?: string
 *   }
 * }} [options]
 * @returns {Promise<void>}
 */
async function prepareLoopAwarePage(page, config, options) {
  const resolvedOptions = options || {};
  await installExternalAssetStubs(page, config);
  await installTauthStub(page, config, resolvedOptions.tauth);
  if (resolvedOptions.clipboard === true) {
    await installClipboardStub(page);
  }
  if (resolvedOptions.localStorage && typeof resolvedOptions.localStorage === 'object') {
    await setLocalStorage(page, resolvedOptions.localStorage);
  }
}

/**
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<void>}
 */
export async function waitForHeaderAuthBound(page) {
  await page.waitForFunction(() => {
    const header = document.querySelector('mpr-header');
    return !!(header && header.getAttribute('data-loopaware-auth-bound') === 'true');
  });
}

/**
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<void>}
 */
export async function waitForHeaderAuthReady(page) {
  await waitForHeaderAuthBound(page);
  await page.waitForFunction(() => {
    const header = document.querySelector('mpr-header');
    if (!header) {
      return false;
    }
    const authState = header.getAttribute('data-loopaware-auth-state') || '';
    return authState !== '' && authState !== 'syncing';
  });
  await waitForExternalAssetStubsToSettle(page);
}

/**
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<void>}
 */
export async function waitForDashboardAccountHydrated(page) {
  await page.waitForFunction(() => {
    const nameElement = document.getElementById('user-name');
    const emailElement = document.getElementById('user-email');
    const nameText = nameElement && nameElement.textContent ? nameElement.textContent.trim() : '';
    const emailText = emailElement && emailElement.textContent ? emailElement.textContent.trim() : '';
    return nameText.length > 0 && emailText.length > 0;
  });
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {{ baseOrigin?: string, baseURL: string, tenantId: string }} config
 * @returns {Promise<void>}
 */
async function seedMprUiAuthRestoreHint(page, config) {
  const authOrigin = config.baseOrigin || new URL(config.baseURL).origin;
  const storageKey = `${AUTH_RESTORE_HINT_PREFIX}${encodeURIComponent(authOrigin)}:${encodeURIComponent(config.tenantId)}`;
  await setLocalStorage(page, { [storageKey]: AUTH_RESTORE_HINT_VALUE });
  if (new URL(page.url()).origin === authOrigin) {
    await page.evaluate(({ key, value }) => {
      window.localStorage.setItem(key, value);
    }, { key: storageKey, value: AUTH_RESTORE_HINT_VALUE });
  }
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} path
 * @returns {Promise<void>}
 */
async function waitForRestoredMprUiSession(page, path) {
  const requestedPathname = new URL(path, 'http://loopaware.test').pathname.replace(/\/+$/g, '') || '/';
  const expectedPath = requestedPathname === LOGIN_PATHNAME ? APP_PATHNAME : path;
  await page.waitForFunction((targetPath) => {
    const normalizePathname = (value) => {
      const pathname = new URL(value, window.location.origin).pathname.replace(/\/+$/g, '');
      return pathname || '/';
    };
    if (normalizePathname(window.location.pathname) !== normalizePathname(targetPath)) {
      return false;
    }
    const headerHost = document.querySelector('mpr-header');
    if (!headerHost || headerHost.getAttribute('data-loopaware-auth-bound') !== 'true') {
      return false;
    }
    return (
      normalizePathname(window.location.pathname) === normalizePathname(targetPath) &&
      headerHost.getAttribute('data-loopaware-auth-state') === 'authenticated' &&
      headerHost.getAttribute('data-mpr-auth-status') === 'authenticated'
    );
  }, expectedPath);
}

/**
 * Wait until the dashboard auth-transition overlay stops intercepting input.
 *
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<void>}
 */
async function waitForDashboardAuthTransitionToHide(page) {
  await page.waitForFunction(() => {
    const header = document.querySelector('mpr-header');
    if (!header) {
      return false;
    }
    if (typeof header.getAttribute === 'function' && !header.getAttribute('auth-transition')) {
      return true;
    }
    const transition = header.querySelector('[data-mpr-header="auth-transition"]');
    if (!transition) {
      return false;
    }
    return transition.getAttribute('data-mpr-visible') !== 'true';
  });
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {{ allowEmptySites?: boolean }} [options]
 * @returns {Promise<void>}
 */
export async function waitForDashboardReady(page, options) {
  const resolvedOptions = options || {};
  await waitForHeaderAuthReady(page);
  await waitForDashboardAccountHydrated(page);
  const allowEmpty = resolvedOptions.allowEmptySites === true;
  await page.waitForFunction((expectEmpty) => {
    const list = document.getElementById('sites-list');
    if (!list) {
      return false;
    }
    const items = list.querySelectorAll('[data-site-id]');
    if (items.length === 0) {
      return expectEmpty;
    }
    const selected = list.querySelectorAll('[data-site-id].active');
    return selected.length > 0;
  }, allowEmpty);
  await waitForDashboardAuthTransitionToHide(page);
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {{ sessionCookieName?: string }} config
 * @param {string} path
 * @param {{
 *   clipboard?: boolean,
 *   localStorage?: Record<string, string | number | boolean | null | undefined>,
 *   tauth?: {
 *     silentBootstrap?: boolean,
 *     delayMs?: number,
 *     bootstrapDelayMs?: number,
 *     currentUserDelayMs?: number,
 *     exchangeDelayMs?: number,
 *     sessionCookieValue?: string
 *   },
 *   waitUntil?: 'commit' | 'domcontentloaded' | 'load' | 'networkidle',
 *   waitForHeaderAuth?: boolean
 * }} [options]
 * @returns {Promise<void>}
 */
export async function openPublicPage(page, config, path, options) {
  const resolvedOptions = options || {};
  const waitUntil = resolvedOptions.waitUntil || 'commit';
  await prepareLoopAwarePage(page, config, resolvedOptions);
  await page.goto(path, { waitUntil });
  if (resolvedOptions.waitForHeaderAuth !== false) {
    await waitForHeaderAuthReady(page);
  }
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {{ baseOrigin?: string, baseURL: string, sessionCookieName?: string, tenantId: string }} config
 * @param {{ email: string, displayName: string, avatarUrl: string, userId: string, issuer?: string }} user
 * @param {string} path
 * @param {{
 *   clipboard?: boolean,
 *   localStorage?: Record<string, string | number | boolean | null | undefined>,
 *   tauth?: {
 *     silentBootstrap?: boolean,
 *     delayMs?: number,
 *     bootstrapDelayMs?: number,
 *     currentUserDelayMs?: number,
 *     exchangeDelayMs?: number,
 *     sessionCookieValue?: string
 *   },
 *   waitUntil?: 'commit' | 'domcontentloaded' | 'load' | 'networkidle',
 *   restoreMprUiSession?: boolean,
 *   waitForHeaderAuth?: boolean
 * }} [options]
 * @returns {Promise<void>}
 */
export async function openAuthenticatedPage(page, config, user, path, options) {
  const resolvedOptions = options || {};
  const waitUntil = resolvedOptions.waitUntil || 'commit';
  await applySessionCookie(page.context(), config, user);
  await prepareLoopAwarePage(page, config, resolvedOptions);
  if (resolvedOptions.restoreMprUiSession !== false) {
    await seedMprUiAuthRestoreHint(page, config);
  }
  await page.goto(path, { waitUntil });
  if (resolvedOptions.restoreMprUiSession !== false) {
    await waitForRestoredMprUiSession(page, path);
  }
  if (resolvedOptions.waitForHeaderAuth !== false) {
    await waitForHeaderAuthReady(page);
  }
}

export async function openDashboard(page, config, user, options) {
  const resolvedOptions = options || {};
  await openAuthenticatedPage(page, config, user, '/app', {
    clipboard: resolvedOptions.clipboard,
    localStorage: resolvedOptions.localStorage,
    tauth: resolvedOptions.tauth
  });
  if (resolvedOptions.waitForSites !== false) {
    await waitForDashboardReady(page, { allowEmptySites: resolvedOptions.allowEmptySites });
    return;
  }
  await waitForHeaderAuthReady(page);
  await waitForDashboardAccountHydrated(page);
  await waitForDashboardAuthTransitionToHide(page);
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {{ sessionCookieName?: string }} config
 * @param {{ email: string, displayName: string, avatarUrl: string, userId: string, issuer?: string }} user
 * @param {{
 *   clipboard?: boolean,
 *   localStorage?: Record<string, string | number | boolean | null | undefined>,
 *   tauth?: {
 *     silentBootstrap?: boolean,
 *     delayMs?: number,
 *     bootstrapDelayMs?: number,
 *     currentUserDelayMs?: number,
 *     exchangeDelayMs?: number,
 *     sessionCookieValue?: string
 *   }
 * }} [options]
 * @returns {Promise<void>}
 */
export async function openDashboardShell(page, config, user, options) {
  const resolvedOptions = options || {};
  await openDashboard(page, config, user, {
    clipboard: resolvedOptions.clipboard,
    localStorage: resolvedOptions.localStorage,
    tauth: resolvedOptions.tauth,
    waitForSites: false
  });
}

export async function selectSite(page, siteId) {
  const siteItem = page.locator(`#sites-list [data-site-id="${siteId}"]`).first();
  await siteItem.waitFor();
  await siteItem.click();
  return siteItem;
}

export async function installClipboardStub(page) {
  await page.addInitScript(() => {
    if (typeof navigator === 'undefined') {
      return;
    }
    if (!navigator.clipboard) {
      Object.defineProperty(navigator, 'clipboard', {
        value: {
          writeText: async () => {}
        },
        configurable: true
      });
    }
  });
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {{
 *   overlaySelector?: string,
 *   redirectPattern?: RegExp,
 *   timeoutMs?: number,
 *   redirectGraceMs?: number
 * }} [options]
 * @returns {Promise<boolean>}
 */
export async function waitForLogoutOverlayOrRedirect(page, options) {
  const resolvedOptions = options || {};
  const overlaySelector = resolvedOptions.overlaySelector || '#logout-overlay';
  const redirectPattern = resolvedOptions.redirectPattern || DEFAULT_LOGOUT_REDIRECT_PATTERN;
  const timeoutMs = Number.isFinite(resolvedOptions.timeoutMs)
    ? Math.max(1, Number(resolvedOptions.timeoutMs))
    : 15_000;
  const redirectGraceMs = Number.isFinite(resolvedOptions.redirectGraceMs)
    ? Math.max(0, Number(resolvedOptions.redirectGraceMs))
    : Math.min(2_500, timeoutMs);
  const overlay = page.locator(overlaySelector);
  if (redirectPattern.test(page.url())) {
    await waitForHeaderAuthReady(page);
    return true;
  }
  let redirected = false;
  try {
    await Promise.any([
      page.waitForURL(redirectPattern, { timeout: timeoutMs }).then(() => {
        redirected = true;
      }),
      overlay.waitFor({ state: 'visible', timeout: timeoutMs })
    ]);
  } catch (error) {
    if (!redirectPattern.test(page.url())) {
      throw error;
    }
    redirected = true;
  }
  if (!redirected && redirectGraceMs > 0) {
    await page.waitForURL(redirectPattern, { timeout: redirectGraceMs }).then(() => {
      redirected = true;
    }).catch(() => {});
  }
  if (redirected || redirectPattern.test(page.url())) {
    await waitForHeaderAuthReady(page);
    return true;
  }
  return redirected || redirectPattern.test(page.url());
}
