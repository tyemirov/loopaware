// @ts-check
import { createSite, listSites } from './api.js';
import { applySessionCookie, setLocalStorage } from './browser.js';
import { installTauthStub } from './tauthStub.js';

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
    avatarUrl: resolvedOverrides.avatarUrl || '',
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

export async function openDashboard(page, config, user, options) {
  const resolvedOptions = options || {};
  await installTauthStub(page, config);
  if (resolvedOptions.clipboard === true) {
    await installClipboardStub(page);
  }
  if (resolvedOptions.localStorage && typeof resolvedOptions.localStorage === 'object') {
    await setLocalStorage(page, resolvedOptions.localStorage);
  }
  await applySessionCookie(page.context(), config, user);
  await page.goto('/app', { waitUntil: 'domcontentloaded' });
  await page.locator('#user-name').waitFor();
  if (resolvedOptions.waitForSites !== false) {
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
  }
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
