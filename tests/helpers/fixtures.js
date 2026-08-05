// @ts-check
import { randomBytes } from 'node:crypto';
import { createSite, listSites } from './api.js';
import { applySessionCookie, setLocalStorage } from './browser.js';
import { installExternalAssetStubs, waitForExternalAssetStubsToSettle } from './externalAssets.js';
import { installTauthStub } from './tauthStub.js';

const DEFAULT_AVATAR_DATA_URL = 'data:image/gif;base64,R0lGODlhAQABAIABAP///wAAACwAAAAAAQABAAACAkQBADs=';
const DEFAULT_LOGOUT_REDIRECT_PATTERN = /\/login(?:\/)?(?:[?#].*)?$/;
const APP_PATHNAME = '/app';
const LANDING_PATHNAME = '/';
const LOGIN_PATHNAME = '/login';
const MPR_UI_TESTING_FIXTURE_PATH = '/privacy';

function randomSuffix() {
  return randomBytes(12).toString('hex');
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
 * @param {{ email: string, displayName: string, avatarUrl: string, userId: string, issuer?: string }} user
 * @returns {{ user_id: string, user_email: string, email: string, display: string, avatar_url: string, roles: string[] }}
 */
function mprUiTestingProfileFromUser(user) {
  return {
    user_id: user.userId,
    user_email: user.email,
    email: user.email,
    display: user.displayName,
    avatar_url: user.avatarUrl,
    roles: []
  };
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {{ email: string, displayName: string, avatarUrl: string, userId: string, issuer?: string }} user
 * @returns {Promise<void>}
 */
export async function authenticateMprUiTestingSession(page, user) {
  const profile = mprUiTestingProfileFromUser(user);
  await page.waitForFunction((currentProfile) => {
    const testingApi = window.MPRUI && window.MPRUI.testing;
    const headerHost = document.querySelector('mpr-header');
    if (!headerHost || headerHost.getAttribute('data-mpr-auth-status') === null) {
      return false;
    }
    if (!testingApi || typeof testingApi.authenticate !== 'function') {
      return false;
    }
    testingApi.authenticate(headerHost, currentProfile);
    return headerHost.getAttribute('data-mpr-auth-status') === 'authenticated';
  }, profile);
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {{ baseOrigin?: string, baseURL: string, sessionCookieName?: string }} config
 * @param {{ email: string, displayName: string, avatarUrl: string, userId: string, issuer?: string }} user
 * @param {{
 *   localStorage?: Record<string, string | number | boolean | null | undefined>,
 *   tauth?: {
 *     silentBootstrap?: boolean,
 *     delayMs?: number,
 *     bootstrapDelayMs?: number,
 *     currentUserDelayMs?: number,
 *     exchangeDelayMs?: number,
 *     sessionCookieValue?: string
 *   }
 * }} options
 * @returns {Promise<void>}
 */
async function authenticateMprUiTestingContext(page, config, user, options) {
  const fixturePage = await page.context().newPage();
  try {
    await prepareLoopAwarePage(fixturePage, config, options);
    await fixturePage.goto(MPR_UI_TESTING_FIXTURE_PATH, { waitUntil: 'commit' });
    await authenticateMprUiTestingSession(fixturePage, user);
  } finally {
    await fixturePage.close();
  }
}

/**
 * @param {string} path
 * @returns {string}
 */
function restoredSessionPath(path) {
  const requestedPathname = new URL(path, 'http://loopaware.test').pathname.replace(/\/+$/g, '') || '/';
  return requestedPathname === LANDING_PATHNAME || requestedPathname === LOGIN_PATHNAME
    ? APP_PATHNAME
    : path;
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} path
 * @returns {Promise<void>}
 */
async function waitForMprUiSessionRestored(page, path) {
  const expectedPath = restoredSessionPath(path);
  await page.waitForFunction((targetPath) => {
    const normalizePathname = (value) => {
      const pathname = new URL(value, window.location.origin).pathname.replace(/\/+$/g, '');
      return pathname || '/';
    };
    const headerHost = document.querySelector('mpr-header');
    return (
      normalizePathname(window.location.pathname) === normalizePathname(targetPath) &&
      headerHost &&
      headerHost.getAttribute('data-mpr-auth-status') === 'authenticated'
    );
  }, expectedPath);
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} path
 * @returns {Promise<void>}
 */
async function waitForLoopAwareSessionRestored(page, path) {
  const expectedPath = restoredSessionPath(path);
  await page.waitForFunction((targetPath) => {
    const normalizePathname = (value) => {
      const pathname = new URL(value, window.location.origin).pathname.replace(/\/+$/g, '');
      return pathname || '/';
    };
    const headerHost = document.querySelector('mpr-header');
    return (
      normalizePathname(window.location.pathname) === normalizePathname(targetPath) &&
      headerHost &&
      headerHost.getAttribute('data-loopaware-auth-bound') === 'true' &&
      headerHost.getAttribute('data-loopaware-auth-state') === 'authenticated'
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
 *   waitUntil?: 'commit' | 'domcontentloaded' | 'load' | 'networkidle'
 * }} [options]
 * @returns {Promise<void>}
 */
export async function openMprUiAuthenticatedPage(page, config, user, path, options) {
  const resolvedOptions = options || {};
  const waitUntil = resolvedOptions.waitUntil || 'commit';
  await applySessionCookie(page.context(), config, user);
  await prepareLoopAwarePage(page, config, resolvedOptions);
  await authenticateMprUiTestingContext(page, config, user, resolvedOptions);
  await page.goto(path, { waitUntil });
  await waitForMprUiSessionRestored(page, path);
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
 *   waitForHeaderAuth?: boolean
 * }} [options]
 * @returns {Promise<void>}
 */
export async function openAuthenticatedPage(page, config, user, path, options) {
  const resolvedOptions = options || {};
  await openMprUiAuthenticatedPage(page, config, user, path, resolvedOptions);
  await waitForLoopAwareSessionRestored(page, path);
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
