// @ts-check

const JS_YAML_URL = 'https://cdn.jsdelivr.net/npm/js-yaml@4.1.0/dist/js-yaml.min.js';
const GOOGLE_IDENTITY_URL = 'https://accounts.google.com/gsi/client';
const GOOGLE_IDENTITY_STYLE_URL = 'https://accounts.google.com/gsi/style';
const GOOGLE_IDENTITY_BUTTON_URL_PATTERN = /^https:\/\/accounts\.google\.com\/gsi\/button(?:\?.*)?$/;
const BOOTSTRAP_CSS_URL = 'https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css';
const BOOTSTRAP_ICONS_CSS_URL = 'https://cdn.jsdelivr.net/npm/bootstrap-icons@1.11.3/font/bootstrap-icons.min.css';
const JS_YAML_STUB = `window.jsyaml = {
  load: function(source) {
    var text = String(source || '');
    if (text.indexOf('origins:') !== -1 && text.indexOf('googleClientId:') !== -1) {
      return {
        environments: [
          {
            description: 'Production',
            origins: ['https://loopaware.mprlab.com', 'https://tyemirov.github.io'],
            auth: {
              tauthUrl: 'https://tauth-api.mprlab.com',
              googleClientId: '281540686395-b0ndglao5r6u6qih2etrtqudf6t0qbt0.apps.googleusercontent.com',
              tenantId: 'loopaware',
              loginPath: '/auth/google',
              logoutPath: '/auth/logout',
              noncePath: '/auth/nonce'
            },
            authButton: {
              text: 'signin_with',
              size: 'large',
              theme: 'outline'
            }
          },
          {
            description: 'development',
            origins: ['http://localhost:8080', 'http://127.0.0.1:8080', 'http://localhost:8090', 'http://127.0.0.1:8090', 'https://computercat.tyemirov.net:4443'],
            auth: {
              tauthUrl: '',
              googleClientId: '281540686395-b0ndglao5r6u6qih2etrtqudf6t0qbt0.apps.googleusercontent.com',
              tenantId: 'loopaware',
              loginPath: '/auth/google',
              logoutPath: '/auth/logout',
              noncePath: '/auth/nonce'
            },
            authButton: {
              text: 'signin_with',
              size: 'large',
              theme: 'outline'
            }
          }
        ]
      };
    }
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
const GOOGLE_IDENTITY_STUB = `(() => {
  window.google = window.google || {};
  window.google.accounts = window.google.accounts || {};

  var state = window.__loopawareGoogleIdentityState;
  if (!state || typeof state !== 'object') {
    state = {
      autoCredentialOnClick: false,
      initializeCalls: [],
      lastInitializeConfig: null,
      renderCount: 0
    };
    window.__loopawareGoogleIdentityState = state;
  }

  function emitCredential(options) {
    var config = state.lastInitializeConfig;
    if (!config || typeof config.callback !== 'function') {
      throw new Error('loopaware.google_stub_missing_callback');
    }
    var override = options && typeof options === 'object' ? options : {};
    var nonce = typeof override.nonce === 'string' ? override.nonce : String(config.nonce || '');
    var credential = typeof override.credential === 'string'
      ? override.credential
      : 'stub-google-credential::' + nonce;
    config.callback({ credential: credential });
  }

  state.emitCredential = emitCredential;

  window.google.accounts.id = {
    __mprUiTesting: {
      isInitialized: function() {
        var config = state.lastInitializeConfig;
        return !!(config && String(config.nonce || '').length > 0);
      },
      getInitializedNonce: function() {
        var config = state.lastInitializeConfig;
        return config ? String(config.nonce || '') : '';
      },
      getInitializeCallCount: function() {
        return state.initializeCalls.length;
      },
      enableAutoCredentialOnClick: function() {
        var config = state.lastInitializeConfig;
        if (!config || !String(config.nonce || '')) {
          throw new Error('loopaware.google_stub_not_initialized');
        }
        state.autoCredentialOnClick = true;
      }
    },
    initialize: function(config) {
      state.lastInitializeConfig = config || null;
      state.initializeCalls.push({
        nonce: String(config && config.nonce ? config.nonce : ''),
        clientId: String(config && (config.client_id || config.clientId) ? (config.client_id || config.clientId) : '')
      });
    },
    renderButton: function(target) {
      state.renderCount += 1;
      if (target && typeof target.setAttribute === 'function') {
        target.setAttribute('data-google-stubbed', 'true');
      }
      if (target && target.ownerDocument && typeof target.appendChild === 'function') {
        target.innerHTML = '';
        var button = target.ownerDocument.createElement('button');
        button.type = 'button';
        button.setAttribute('data-test', 'google-signin');
        button.textContent = 'Sign in with Google';
        target.appendChild(button);
        button.addEventListener('click', function() {
          if (state.autoCredentialOnClick === true) {
            try {
              emitCredential();
            } catch (error) {}
          }
        });
      }
    },
    prompt: function() {},
    cancel: function() {},
    disableAutoSelect: function() {},
    revoke: function() {}
  };
})();`;
const GOOGLE_IDENTITY_BUTTON_STUB = '<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"></head><body></body></html>';
const EMPTY_CSS_STUB = '';
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
 * @returns {Promise<void>}
 */
export async function waitForGoogleIdentityStubInitialized(page) {
  await page.waitForFunction(() => {
    const win = /** @type {any} */ (window);
    const testingApi = win.MPRUI && win.MPRUI.testing && win.MPRUI.testing.googleIdentity;
    return !!(
      testingApi &&
      typeof testingApi.isInitialized === 'function' &&
      testingApi.isInitialized() === true
    );
  });
}

/**
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<void>}
 */
export async function enableAutoGoogleCredentialOnClick(page) {
  await waitForGoogleIdentityStubInitialized(page);
  await page.evaluate(() => {
    const win = /** @type {any} */ (window);
    const testingApi = win.MPRUI && win.MPRUI.testing && win.MPRUI.testing.googleIdentity;
    if (!testingApi || typeof testingApi.enableAutoCredentialOnClick !== 'function') {
      throw new Error('loopaware.mpr_ui_google_identity_testing_missing');
    }
    testingApi.enableAutoCredentialOnClick();
  });
}

/**
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<string>}
 */
export async function getGoogleIdentityInitializedNonce(page) {
  await waitForGoogleIdentityStubInitialized(page);
  return page.evaluate(() => {
    const win = /** @type {any} */ (window);
    const testingApi = win.MPRUI && win.MPRUI.testing && win.MPRUI.testing.googleIdentity;
    if (!testingApi || typeof testingApi.getInitializedNonce !== 'function') {
      throw new Error('loopaware.mpr_ui_google_identity_testing_missing');
    }
    return String(testingApi.getInitializedNonce() || '');
  });
}

/**
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<number>}
 */
export async function getGoogleIdentityInitializeCallCount(page) {
  await waitForGoogleIdentityStubInitialized(page);
  return page.evaluate(() => {
    const win = /** @type {any} */ (window);
    const testingApi = win.MPRUI && win.MPRUI.testing && win.MPRUI.testing.googleIdentity;
    if (!testingApi || typeof testingApi.getInitializeCallCount !== 'function') {
      throw new Error('loopaware.mpr_ui_google_identity_testing_missing');
    }
    return Number(testingApi.getInitializeCallCount());
  });
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
 * the configured CDN URLs. These stubs prevent third-party CSS from blocking
 * parser progress when the network is slow.
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
}
