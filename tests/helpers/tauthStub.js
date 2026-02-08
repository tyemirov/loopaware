// @ts-check

export function renderTauthStub(sessionCookieName) {
  const resolvedCookieName = sessionCookieName || 'app_session';
  return `(() => {
  if (typeof window === 'undefined') {
    return;
  }

  var runtimeKey = '__loopawareTestTauthRuntime';
  var sessionCookieName = '${resolvedCookieName}';

  var runtime = window[runtimeKey];
  if (!runtime || typeof runtime !== 'object') {
    runtime = { tenantId: '', profile: null, options: null };
    window[runtimeKey] = runtime;
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

  function setAuthTenantId(tenantId) {
    runtime.tenantId = String(tenantId || '');
  }

  function getCurrentUser() {
    return runtime.profile;
  }

  function initAuthClient(options) {
    runtime.options = options || null;
    var profile = hydrateProfile();
    try {
      if (profile && options && typeof options.onAuthenticated === 'function') {
        options.onAuthenticated(profile);
      }
      if (!profile && options && typeof options.onUnauthenticated === 'function') {
        options.onUnauthenticated();
      }
    } catch (error) {}
    return Promise.resolve();
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
    return Promise.resolve('test-nonce');
  }

  function exchangeGoogleCredential() {
    return Promise.reject(new Error('tauth.exchange_not_supported'));
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

export async function installTauthStub(page, config) {
  const scriptBody = renderTauthStub(config.sessionCookieName);
  await page.route('**/tauth.js', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/javascript; charset=utf-8',
      body: scriptBody
    });
  });
}
