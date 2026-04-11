// @ts-check
import * as fs from 'node:fs';
import * as path from 'node:path';
import { fileURLToPath } from 'node:url';

const JS_YAML_URL = 'https://cdn.jsdelivr.net/npm/js-yaml@4.1.0/dist/js-yaml.min.js';
const GOOGLE_IDENTITY_URL = 'https://accounts.google.com/gsi/client';
const GOOGLE_IDENTITY_STYLE_URL = 'https://accounts.google.com/gsi/style';
const GOOGLE_IDENTITY_BUTTON_URL_PATTERN = /^https:\/\/accounts\.google\.com\/gsi\/button(?:\?.*)?$/;
const BOOTSTRAP_CSS_URL = 'https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css';
const BOOTSTRAP_ICONS_CSS_URL = 'https://cdn.jsdelivr.net/npm/bootstrap-icons@1.11.3/font/bootstrap-icons.min.css';
const BOOTSTRAP_BUNDLE_URL = 'https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/js/bootstrap.bundle.min.js';
const helperDirectory = path.dirname(fileURLToPath(import.meta.url));
const BOOTSTRAP_BUNDLE_PATH = path.resolve(helperDirectory, '../../../MediaOps/node_modules/bootstrap/dist/js/bootstrap.bundle.min.js');
const BOOTSTRAP_BUNDLE_BODY = fs.existsSync(BOOTSTRAP_BUNDLE_PATH) ? fs.readFileSync(BOOTSTRAP_BUNDLE_PATH, 'utf-8') : '';
const JS_YAML_STUB = `window.jsyaml = {
  load: function() {
    return {
      environments: [
        {
          name: 'production',
          hostnames: ['loopaware.mprlab.com', 'tyemirov.github.io'],
          services: {
            apiOrigin: 'https://loopaware-api.mprlab.com',
            tauthOrigin: 'https://tauth-api.mprlab.com',
            siteWidgetSiteId: 'a3222433-92ec-473a-9255-0797226c2273'
          }
        },
        {
          name: 'development',
          hostnames: ['computercat.tyemirov.net', 'localhost', '127.0.0.1'],
          services: {
            apiOrigin: '',
            tauthOrigin: '',
            siteWidgetSiteId: ''
          }
        }
      ]
    };
  }
};`;
const GOOGLE_IDENTITY_STUB = `window.google = window.google || {};
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
};`;
const GOOGLE_IDENTITY_BUTTON_STUB = '<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"></head><body></body></html>';
const EMPTY_CSS_STUB = '';
const EMPTY_SCRIPT_STUB = '';
const DEFAULT_ASSET_SETTLE_TIMEOUT_MS = 5_000;
const DEFAULT_ASSET_SETTLE_QUIET_MS = 150;

function stripIntegrityAttributes(body) {
  return String(body || '')
    .replace(/\s+integrity="[^"]*"/g, '')
    .replace(/\s+crossorigin="anonymous"/g, '');
}

/**
 * @param {any} browserPage
 * @returns {{ pending: number, lastActivityAt: number }}
 */
function ensureAssetRouteTracker(browserPage) {
  if (!browserPage.__loopawareExternalAssetRouteTracker || typeof browserPage.__loopawareExternalAssetRouteTracker !== 'object') {
    browserPage.__loopawareExternalAssetRouteTracker = {
      pending: 0,
      lastActivityAt: Date.now()
    };
  }
  return browserPage.__loopawareExternalAssetRouteTracker;
}

/**
 * @param {any} browserPage
 * @param {() => Promise<void>} callback
 * @returns {Promise<void>}
 */
async function runTrackedRoute(browserPage, callback) {
  const tracker = ensureAssetRouteTracker(browserPage);
  tracker.pending += 1;
  tracker.lastActivityAt = Date.now();
  try {
    await callback();
  } finally {
    tracker.pending = Math.max(0, tracker.pending - 1);
    tracker.lastActivityAt = Date.now();
  }
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {{ timeoutMs?: number, quietMs?: number }} [options]
 * @returns {Promise<void>}
 */
export async function waitForExternalAssetStubsToSettle(page, options) {
  const browserPage = /** @type {any} */ (page);
  const tracker = ensureAssetRouteTracker(browserPage);
  const timeoutMs = Number.isFinite(options?.timeoutMs)
    ? Math.max(1, Number(options?.timeoutMs))
    : DEFAULT_ASSET_SETTLE_TIMEOUT_MS;
  const quietMs = Number.isFinite(options?.quietMs)
    ? Math.max(0, Number(options?.quietMs))
    : DEFAULT_ASSET_SETTLE_QUIET_MS;
  const deadline = Date.now() + timeoutMs;
  while (Date.now() <= deadline) {
    if (tracker.pending === 0 && Date.now() - tracker.lastActivityAt >= quietMs) {
      return;
    }
    await new Promise((resolve) => {
      setTimeout(resolve, 25);
    });
  }
  throw new Error('loopaware_external_asset_stubs_not_settled');
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {{ baseOrigin?: string, baseURL?: string, sessionCookieName?: string }} [config]
 * @returns {Promise<void>}
 */
export async function installExternalAssetStubs(page, config) {
  const browserPage = /** @type {any} */ (page);
  if (browserPage.__loopawareExternalAssetStubsInstalled === true) {
    return;
  }
  browserPage.__loopawareExternalAssetStubsInstalled = true;
  ensureAssetRouteTracker(browserPage);
  const appOrigin = config?.baseOrigin || (config?.baseURL ? new URL(config.baseURL).origin : '');

  await page.route(JS_YAML_URL, async (route) => {
    await runTrackedRoute(browserPage, async () => {
      await route.fulfill({
        status: 200,
        contentType: 'application/javascript; charset=utf-8',
        body: JS_YAML_STUB
      });
    });
  });
  await page.route(GOOGLE_IDENTITY_URL, async (route) => {
    await runTrackedRoute(browserPage, async () => {
      await route.fulfill({
        status: 200,
        contentType: 'application/javascript; charset=utf-8',
        body: GOOGLE_IDENTITY_STUB
      });
    });
  });
  await page.route(GOOGLE_IDENTITY_STYLE_URL, async (route) => {
    await runTrackedRoute(browserPage, async () => {
      await route.fulfill({
        status: 200,
        contentType: 'text/css; charset=utf-8',
        body: EMPTY_CSS_STUB
      });
    });
  });
  await page.route(GOOGLE_IDENTITY_BUTTON_URL_PATTERN, async (route) => {
    await runTrackedRoute(browserPage, async () => {
      await route.fulfill({
        status: 200,
        contentType: 'text/html; charset=utf-8',
        body: GOOGLE_IDENTITY_BUTTON_STUB
      });
    });
  });
  await page.route(BOOTSTRAP_ICONS_CSS_URL, async (route) => {
    await runTrackedRoute(browserPage, async () => {
      await route.fulfill({
        status: 200,
        contentType: 'text/css; charset=utf-8',
        body: EMPTY_CSS_STUB
      });
    });
  });
  await page.route(BOOTSTRAP_BUNDLE_URL, async (route) => {
    await runTrackedRoute(browserPage, async () => {
      await route.fulfill({
        status: 200,
        contentType: 'application/javascript; charset=utf-8',
        body: BOOTSTRAP_BUNDLE_BODY
      });
    });
  });
  await page.route('**/*', async (route) => {
    await runTrackedRoute(browserPage, async () => {
      const request = route.request();
      if (request.resourceType() !== 'document') {
        await route.fallback();
        return;
      }
      if (appOrigin) {
        const requestOrigin = new URL(request.url()).origin;
        if (requestOrigin !== appOrigin) {
          await route.fallback();
          return;
        }
      }
      const response = await route.fetch();
      const headers = response.headers();
      const contentType = String(headers['content-type'] || headers['Content-Type'] || '');
      if (!contentType.includes('text/html')) {
        await route.fallback();
        return;
      }
      const body = await response.text();
      await route.fulfill({
        response,
        body: stripIntegrityAttributes(body)
      });
    });
  });
}

/**
 * Asset-inspection tests only need the page to finish parsing so they can verify
 * the configured CDN URLs. These stubs prevent third-party CSS and Bootstrap JS
 * from blocking parser progress when the network is slow.
 *
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<void>}
 */
export async function installAssetInspectionStubs(page) {
  const browserPage = /** @type {any} */ (page);
  if (browserPage.__loopawareAssetInspectionStubsInstalled === true) {
    return;
  }
  browserPage.__loopawareAssetInspectionStubsInstalled = true;

  await page.route(BOOTSTRAP_CSS_URL, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'text/css; charset=utf-8',
      body: EMPTY_CSS_STUB
    });
  });
  await page.route(BOOTSTRAP_ICONS_CSS_URL, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'text/css; charset=utf-8',
      body: EMPTY_CSS_STUB
    });
  });
  await page.route(BOOTSTRAP_BUNDLE_URL, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/javascript; charset=utf-8',
      body: EMPTY_SCRIPT_STUB
    });
  });
}
