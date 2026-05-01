// @ts-check

/**
 * @param {string} sessionCookieName
 * @param {{ silentBootstrap?: boolean, bootstrapDelayMs?: number, currentUserDelayMs?: number, exchangeDelayMs?: number, sessionCookieValue?: string }} [options]
 * @returns {string}
 */
export function renderTauthStub(sessionCookieName, options) {
  const resolvedCookieName = sessionCookieName || 'app_session';
  const resolvedOptions = options || {};
  const silentBootstrap = resolvedOptions.silentBootstrap === true;
  const bootstrapDelayMs = Number.isFinite(resolvedOptions.bootstrapDelayMs)
    ? Math.max(0, Number(resolvedOptions.bootstrapDelayMs))
    : 0;
  const currentUserDelayMs = Number.isFinite(resolvedOptions.currentUserDelayMs)
    ? Math.max(0, Number(resolvedOptions.currentUserDelayMs))
    : 0;
  const exchangeDelayMs = Number.isFinite(resolvedOptions.exchangeDelayMs)
    ? Math.max(0, Number(resolvedOptions.exchangeDelayMs))
    : 0;
  const sessionCookieValue = typeof resolvedOptions.sessionCookieValue === 'string'
    ? resolvedOptions.sessionCookieValue.trim()
    : '';
  return `(() => {
  if (typeof window === 'undefined') {
    return;
  }

  var runtimeKey = '__loopawareTestTauthRuntime';
  var sessionCookieName = ${JSON.stringify(resolvedCookieName)};
  var exchangeSessionCookieValue = ${JSON.stringify(sessionCookieValue)};
  var silentBootstrap = ${silentBootstrap ? 'true' : 'false'};
  var bootstrapDelayMs = ${bootstrapDelayMs};
  var currentUserDelayMs = ${currentUserDelayMs};
  var exchangeDelayMs = ${exchangeDelayMs};

  var runtime = window[runtimeKey];
  if (!runtime || typeof runtime !== 'object') {
    runtime = { tenantId: '', profile: null, options: null, exchangeProfile: null, nonceCounter: 0 };
    window[runtimeKey] = runtime;
  }
  if (!Object.prototype.hasOwnProperty.call(runtime, 'exchangeProfile')) {
    runtime.exchangeProfile = null;
  }
  if (!Number.isFinite(runtime.nonceCounter)) {
    runtime.nonceCounter = 0;
  }

  function readCookieValue(name) {
    if (typeof document === 'undefined' || typeof document.cookie !== 'string') {
      return '';
    }
    var prefix = String(name || '').trim() + '=';
    if (prefix === '=') {
      return '';
    }
    var parts = document.cookie.split(';');
    for (var index = 0; index < parts.length; index += 1) {
      var entry = parts[index];
      if (!entry) {
        continue;
      }
      var trimmed = entry.trim();
      if (trimmed.indexOf(prefix) !== 0) {
        continue;
      }
      return trimmed.slice(prefix.length);
    }
    return '';
  }

  function decodeBase64Url(input) {
    if (!input || typeof input !== 'string' || typeof window.atob !== 'function') {
      return '';
    }
    var normalized = input.replace(/-/g, '+').replace(/_/g, '/');
    var padding = normalized.length % 4;
    if (padding === 2) {
      normalized += '==';
    } else if (padding === 3) {
      normalized += '=';
    } else if (padding !== 0) {
      return '';
    }
    try {
      return window.atob(normalized);
    } catch (error) {
      return '';
    }
  }

  function parseSessionClaims() {
    var token = readCookieValue(sessionCookieName);
    if (!token) {
      return null;
    }
    var parts = token.split('.');
    if (!parts || parts.length < 2) {
      return null;
    }
    var payload = decodeBase64Url(parts[1]);
    if (!payload) {
      return null;
    }
    try {
      return JSON.parse(payload);
    } catch (error) {
      return null;
    }
  }

  function resolveProfileFromClaims(claims) {
    if (!claims || typeof claims !== 'object') {
      return null;
    }
    var email = typeof claims.user_email === 'string' ? claims.user_email.trim() : '';
    var display = typeof claims.user_display_name === 'string' ? claims.user_display_name.trim() : '';
    var avatarUrl = typeof claims.user_avatar_url === 'string' ? claims.user_avatar_url.trim() : '';
    var userId = typeof claims.user_id === 'string' ? claims.user_id.trim() : '';
    var roles = Array.isArray(claims.user_roles) ? claims.user_roles.slice() : [];
    if (!email && !display && !avatarUrl && !userId) {
      return null;
    }
    if (!display) {
      display = email;
    }
    return {
      user_id: userId,
      user_email: email,
      email: email,
      display: display,
      avatar_url: avatarUrl,
      roles: roles
    };
  }

  function hydrateProfile() {
    var claims = parseSessionClaims();
    runtime.profile = resolveProfileFromClaims(claims);
    return runtime.profile;
  }

  function normalizeRuntimeProfile(profile) {
    if (!profile || typeof profile !== 'object') {
      return null;
    }
    var email = typeof profile.user_email === 'string'
      ? profile.user_email.trim()
      : (typeof profile.email === 'string' ? profile.email.trim() : '');
    var display = typeof profile.display === 'string'
      ? profile.display.trim()
      : (typeof profile.user_display_name === 'string' ? profile.user_display_name.trim() : '');
    var avatarUrl = typeof profile.avatar_url === 'string'
      ? profile.avatar_url.trim()
      : (typeof profile.user_avatar_url === 'string' ? profile.user_avatar_url.trim() : '');
    var userId = typeof profile.user_id === 'string' ? profile.user_id.trim() : '';
    var roles = Array.isArray(profile.roles)
      ? profile.roles.slice()
      : (Array.isArray(profile.user_roles) ? profile.user_roles.slice() : []);
    if (!display) {
      display = email;
    }
    return {
      user_id: userId,
      user_email: email,
      email: email,
      display: display,
      avatar_url: avatarUrl,
      roles: roles
    };
  }

  function resolveExchangeProfile() {
    return normalizeRuntimeProfile(runtime.exchangeProfile) ||
      normalizeRuntimeProfile(runtime.profile) ||
      resolveProfileFromClaims(parseSessionClaims()) || {
        user_id: 'test-user',
        user_email: 'user@example.com',
        email: 'user@example.com',
        display: 'Test User',
        avatar_url: '',
        roles: []
      };
  }

  function persistExchangeSessionCookie() {
    if (!exchangeSessionCookieValue || typeof document === 'undefined') {
      return;
    }
    var secureDirective = window && window.location && window.location.protocol === 'https:' ? '; Secure' : '';
    document.cookie = sessionCookieName + '=' + exchangeSessionCookieValue + '; Path=/; SameSite=Lax' + secureDirective;
  }

  function readCredentialNonce(credential) {
    if (typeof credential !== 'string') {
      return '';
    }
    var prefix = 'stub-google-credential::';
    if (credential.indexOf(prefix) !== 0) {
      return '';
    }
    return credential.slice(prefix.length);
  }

  function setAuthTenantId(tenantId) {
    runtime.tenantId = String(tenantId || '');
  }

  function getCurrentUser() {
    if (currentUserDelayMs > 0) {
      return new Promise(function(resolve) {
        window.setTimeout(function() {
          resolve(runtime.profile);
        }, currentUserDelayMs);
      });
    }
    return runtime.profile;
  }

  function initAuthClient(options) {
    runtime.options = options || null;
    var profile = hydrateProfile();
    return new Promise(function (resolve) {
      var finalize = function () {
        if (silentBootstrap) {
          resolve();
          return;
        }
        try {
          if (profile && options && typeof options.onAuthenticated === 'function') {
            options.onAuthenticated(profile);
          }
          if (!profile && options && typeof options.onUnauthenticated === 'function') {
            options.onUnauthenticated();
          }
        } catch (error) {}
        resolve();
      };
      if (bootstrapDelayMs > 0) {
        window.setTimeout(finalize, bootstrapDelayMs);
        return;
      }
      finalize();
    });
  }

  function apiFetch(url, initOptions) {
    var merged = Object.assign({}, initOptions || {});
    merged.credentials = 'include';
    return window.fetch(url, merged);
  }

  function getAuthEndpoints() {
    return {
      baseUrl: '',
      meUrl: '/api/me',
      nonceUrl: '/auth/nonce',
      googleUrl: '/auth/google',
      refreshUrl: '/auth/refresh',
      logoutUrl: '/auth/logout'
    };
  }

  function requestNonce() {
    runtime.nonceCounter += 1;
    return Promise.resolve('test-nonce-' + String(runtime.nonceCounter));
  }

  function exchangeGoogleCredential(input) {
    var normalizedInput = input && typeof input === 'object' ? input : {};
    var credential = typeof normalizedInput.credential === 'string' ? normalizedInput.credential : '';
    var nonceToken = typeof normalizedInput.nonceToken === 'string' ? normalizedInput.nonceToken : '';
    var credentialNonce = readCredentialNonce(credential);
    if (!credentialNonce || !nonceToken || credentialNonce !== nonceToken) {
      return Promise.reject(new Error('tauth.exchange_failed'));
    }
    var exchangeProfile = resolveExchangeProfile();
    if (exchangeDelayMs > 0) {
      return new Promise(function(resolve) {
        window.setTimeout(function() {
          runtime.profile = exchangeProfile;
          persistExchangeSessionCookie();
          resolve(runtime.profile);
        }, exchangeDelayMs);
      });
    }
    runtime.profile = exchangeProfile;
    persistExchangeSessionCookie();
    return Promise.resolve(runtime.profile);
  }

  function clearSessionCookie() {
    if (typeof document === 'undefined') {
      return;
    }
    var expireDirective = 'Max-Age=0; path=/';
    var hostName = window && window.location && typeof window.location.hostname === 'string'
      ? window.location.hostname
      : '';
    document.cookie = sessionCookieName + '=; ' + expireDirective;
    if (hostName) {
      document.cookie = sessionCookieName + '=; ' + expireDirective + '; domain=' + hostName;
    }
  }

  function logout() {
    runtime.profile = null;
    clearSessionCookie();
    return Promise.resolve();
  }

  hydrateProfile();

  if (typeof window.setAuthTenantId !== 'function') {
    window.setAuthTenantId = setAuthTenantId;
  }
  if (typeof window.getCurrentUser !== 'function') {
    window.getCurrentUser = getCurrentUser;
  }
  if (typeof window.initAuthClient !== 'function') {
    window.initAuthClient = initAuthClient;
  }
  if (typeof window.apiFetch !== 'function') {
    window.apiFetch = apiFetch;
  }
  if (typeof window.getAuthEndpoints !== 'function') {
    window.getAuthEndpoints = getAuthEndpoints;
  }
  if (typeof window.requestNonce !== 'function') {
    window.requestNonce = requestNonce;
  }
  if (typeof window.exchangeGoogleCredential !== 'function') {
    window.exchangeGoogleCredential = exchangeGoogleCredential;
  }
  if (typeof window.logout !== 'function') {
    window.logout = logout;
  }
})();`;
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {{ sessionCookieName?: string }} config
 * @param {{ silentBootstrap?: boolean, delayMs?: number, bootstrapDelayMs?: number, currentUserDelayMs?: number, exchangeDelayMs?: number, sessionCookieValue?: string }} [options]
 * @returns {Promise<void>}
 */
export async function installTauthStub(page, config, options) {
  const scriptBody = renderTauthStub(config.sessionCookieName, options);
  const browserPage = /** @type {any} */ (page);
  if (browserPage.__loopawareTauthStubScriptBody === scriptBody) {
    return;
  }
  if (typeof browserPage.__loopawareTauthStubScriptBody === 'string') {
    throw new Error('tauth_stub_already_installed_with_different_options');
  }
  browserPage.__loopawareTauthStubScriptBody = scriptBody;
  const delayMs = Number.isFinite(options?.delayMs) ? Math.max(0, Number(options.delayMs)) : 0;
  await page.route('**/tauth.js', async (route) => {
    if (delayMs > 0) {
      await new Promise((resolve) => {
        setTimeout(resolve, delayMs);
      });
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/javascript; charset=utf-8',
      body: scriptBody
    });
  });
}
