// @ts-check
(function() {
  if (document && document.documentElement) {
    document.documentElement.setAttribute('data-loopaware-auth-script', 'true');
  }

  var AUTH_STATE_VALUES = Object.freeze({
    syncing: 'syncing',
    authenticated: 'authenticated',
    unauthenticated: 'unauthenticated'
  });
  var AUTH_STYLE_ELEMENT_ID = 'loopaware-header-auth-style';
  var AUTH_STATE_CHANGE_EVENT = 'loopaware:auth-state-change';
  var GOOGLE_SIGNIN_GATE_MAX_ATTEMPTS = 40;
  var GOOGLE_SIGNIN_GATE_POLL_INTERVAL_MS = 100;
  var GOOGLE_SIGNIN_GATE_SLOW_POLL_INTERVAL_MS = 1000;
  var HELPER_SYNC_MAX_ATTEMPTS = 60;
  var HELPER_SYNC_INTERVAL_MS = 100;
  var APP_PATHNAME = '/app';
  var LOGIN_PATHNAME = '/login';
  var store = window.__loopawareHeaderAuthStore;

  if (!store || typeof store !== 'object') {
    store = {
      snapshot: null,
      helperSyncStarted: false,
      redirectTarget: ''
    };
    window.__loopawareHeaderAuthStore = store;
  }

  function showOverlay() {
    if (typeof window.showLogoutOverlay === 'function') {
      window.showLogoutOverlay();
    }
  }

  function hideOverlay() {
    if (typeof window.hideLogoutOverlay === 'function') {
      window.hideLogoutOverlay();
    }
  }

  function injectAuthStyles() {
    if (!document || typeof document.getElementById !== 'function') {
      return;
    }
    if (document.getElementById(AUTH_STYLE_ELEMENT_ID)) {
      return;
    }
    var styleElement = document.createElement('style');
    styleElement.id = AUTH_STYLE_ELEMENT_ID;
    styleElement.textContent =
      'mpr-header[data-loopaware-auth-state="syncing"] [data-mpr-header="google-signin"],' +
      'mpr-header[data-loopaware-auth-state="syncing"] mpr-user[data-loopaware-user-menu="true"]{' +
      'display:none !important;' +
      '}' +
      'mpr-header[data-loopaware-auth-state="authenticated"] [data-mpr-header="google-signin"]{' +
      'display:none !important;' +
      '}' +
      'mpr-header[data-loopaware-auth-state="authenticated"] mpr-user[data-loopaware-user-menu="true"]{' +
      'display:inline-flex !important;' +
      '}' +
      'mpr-header[data-loopaware-auth-state="unauthenticated"] mpr-user[data-loopaware-user-menu="true"]{' +
      'display:none !important;' +
      '}' +
      'mpr-header[data-loopaware-auth-state="unauthenticated"] [data-mpr-header="google-signin"]{' +
      'display:flex !important;' +
      '}';
    var head = document.head || document.documentElement;
    if (!head || typeof head.appendChild !== 'function') {
      return;
    }
    head.appendChild(styleElement);
  }

  function pruneHeaderUserMenus(headerHost) {
    if (!headerHost || typeof headerHost.querySelectorAll !== 'function') {
      return;
    }
    var userMenus = headerHost.querySelectorAll('mpr-user');
    if (!userMenus || userMenus.length <= 1) {
      return;
    }
    var preferred = headerHost.querySelector('mpr-user[data-loopaware-user-menu="true"]');
    var keep = preferred || userMenus[0];
    for (var index = 0; index < userMenus.length; index += 1) {
      var candidate = userMenus[index];
      if (!candidate || candidate === keep) {
        continue;
      }
      if (candidate.parentNode) {
        candidate.parentNode.removeChild(candidate);
      }
    }
  }

  function normalizeBaseURL(value) {
    if (!value) {
      return '';
    }
    var trimmed = String(value).trim();
    if (!trimmed) {
      return '';
    }
    return trimmed.replace(/\/+$/, '');
  }

  function normalizePath(value, fallback) {
    if (!value) {
      return fallback;
    }
    var trimmed = String(value).trim();
    if (!trimmed) {
      return fallback;
    }
    return trimmed;
  }

  function resolveLogoutURL(headerHost) {
    var logoutPath = '/auth/logout';
    var baseUrl = '';
    if (typeof window.getAuthEndpoints === 'function') {
      try {
        var endpoints = window.getAuthEndpoints();
        if (endpoints && typeof endpoints.logoutUrl === 'string' && endpoints.logoutUrl.trim()) {
          return endpoints.logoutUrl;
        }
      } catch (error) {}
    }
    if (headerHost && typeof headerHost.getAttribute === 'function') {
      baseUrl = normalizeBaseURL(headerHost.getAttribute('tauth-url'));
      logoutPath = normalizePath(headerHost.getAttribute('tauth-logout-path'), logoutPath);
    }
    if (!baseUrl && window.location && typeof window.location.origin === 'string') {
      baseUrl = normalizeBaseURL(window.location.origin);
    }
    if (logoutPath.indexOf('http://') === 0 || logoutPath.indexOf('https://') === 0) {
      return logoutPath;
    }
    if (!baseUrl) {
      return logoutPath;
    }
    if (logoutPath.indexOf('/') === 0) {
      return baseUrl + logoutPath;
    }
    return baseUrl + '/' + logoutPath;
  }

  function resolveLogoutHeaders(headerHost) {
    var headers = { 'X-Requested-With': 'XMLHttpRequest' };
    if (headerHost && typeof headerHost.getAttribute === 'function') {
      var tenantId = headerHost.getAttribute('tauth-tenant-id');
      if (tenantId) {
        headers['X-TAuth-Tenant'] = tenantId;
      }
    }
    return headers;
  }

  function submitLogoutForm(logoutURL) {
    return new Promise(function(resolve) {
      if (!document || !logoutURL) {
        resolve();
        return;
      }
      var root = document.body || document.documentElement;
      if (!root || typeof document.createElement !== 'function') {
        resolve();
        return;
      }
      var iframeName = 'loopaware-logout-target';
      var iframe = document.createElement('iframe');
      iframe.name = iframeName;
      iframe.setAttribute('data-loopaware-logout-target', 'true');
      iframe.style.display = 'none';
      var form = document.createElement('form');
      form.method = 'POST';
      form.action = logoutURL;
      form.target = iframeName;
      form.style.display = 'none';
      form.setAttribute('data-loopaware-logout-form', 'true');
      root.appendChild(iframe);
      root.appendChild(form);
      try {
        form.submit();
      } catch (error) {}
      window.setTimeout(function() {
        if (form.parentNode) {
          form.parentNode.removeChild(form);
        }
        if (iframe.parentNode) {
          iframe.parentNode.removeChild(iframe);
        }
        resolve();
      }, 1500);
    });
  }

  function performLogoutRequest(headerHost, logoutDelegate) {
    var logoutURL = resolveLogoutURL(headerHost);
    var logoutRequest = function() {
      return window.fetch(logoutURL, {
        method: 'POST',
        credentials: 'include',
        headers: resolveLogoutHeaders(headerHost)
      });
    };
    function assertLogoutResponseOk(response) {
      if (!response || typeof response.ok !== 'boolean') {
        return response;
      }
      if (response.ok) {
        return response;
      }
      var error = new Error('loopaware.logout_failed');
      error.code = 'loopaware.logout_failed';
      error.status = response.status;
      throw error;
    }
    function invokeLogoutDelegate() {
      if (typeof logoutDelegate !== 'function') {
        return Promise.resolve();
      }
      try {
        return Promise.resolve(logoutDelegate()).catch(function() {});
      } catch (error) {
        return Promise.resolve();
      }
    }
    var logoutWithFetchFallback = function() {
      return logoutRequest()
        .then(assertLogoutResponseOk)
        .catch(function(error) {
          if (error && typeof error.status === 'number' && error.status) {
            throw error;
          }
          return submitLogoutForm(logoutURL);
        });
    };
    return logoutWithFetchFallback().then(function(result) {
      return invokeLogoutDelegate().then(function() {
        return result;
      });
    }).catch(function(error) {
      hideOverlay();
      throw error;
    });
  }

  function ensureLogoutFallback(headerHost) {
    if (!window || !headerHost) {
      return;
    }
    window.__loopawareLogoutHeaderHost = headerHost;
    var existingWrapper = window.__loopawareLogoutWrapper;
    if (typeof existingWrapper !== 'function' || existingWrapper.__loopawareLogoutWrapper !== true) {
      existingWrapper = function() {
        showOverlay();
        var resolvedHost = window.__loopawareLogoutHeaderHost || headerHost;
        return performLogoutRequest(resolvedHost, window.__loopawareLogoutDelegate);
      };
      existingWrapper.__loopawareLogoutWrapper = true;
      window.__loopawareLogoutWrapper = existingWrapper;
    }
    if (typeof window.__loopawareLogoutDelegate !== 'function' && typeof window.logout === 'function' && window.logout.__loopawareLogoutWrapper !== true) {
      window.__loopawareLogoutDelegate = window.logout;
    }
    if (window.__loopawareLogoutTracking === true) {
      return;
    }
    window.__loopawareLogoutTracking = true;
    try {
      Object.defineProperty(window, 'logout', {
        configurable: true,
        enumerable: true,
        get: function() {
          return window.__loopawareLogoutWrapper || existingWrapper;
        },
        set: function(value) {
          if (typeof value === 'function' && value.__loopawareLogoutWrapper === true) {
            window.__loopawareLogoutWrapper = value;
            return;
          }
          if (typeof value === 'function') {
            window.__loopawareLogoutDelegate = value;
            return;
          }
          window.__loopawareLogoutDelegate = undefined;
        }
      });
    } catch (error) {
      try {
        window.logout = existingWrapper;
      } catch (innerError) {}
      return;
    }
    try {
      window.logout = existingWrapper;
    } catch (error) {}
  }

  function disableGoogleAutoSelect() {
    if (!window || !window.google || !window.google.accounts || !window.google.accounts.id) {
      return;
    }
    var identityAPI = window.google.accounts.id;
    if (typeof identityAPI.cancel === 'function') {
      try {
        identityAPI.cancel();
      } catch (error) {}
    }
    if (typeof identityAPI.disableAutoSelect === 'function') {
      try {
        identityAPI.disableAutoSelect();
      } catch (error) {}
    }
  }

  function resolveGoogleSigninTarget(headerHost) {
    if (!headerHost || typeof headerHost.querySelector !== 'function') {
      return null;
    }
    var container = headerHost.querySelector('[data-mpr-header="google-signin"]');
    if (!container) {
      return null;
    }
    var wrapper = container.querySelector('[data-mpr-google-wrapper="true"]');
    if (wrapper) {
      return wrapper;
    }
    return container;
  }

  function hasGooglePromptNonce() {
    return !!(window && ((window.__googleInitConfig && window.__googleInitConfig.nonce) || window.__loopawareGooglePromptNonce));
  }

  function normalizeNonceValue(value) {
    if (!value) {
      return '';
    }
    if (typeof value === 'string') {
      return value.trim();
    }
    if (typeof value === 'object') {
      if (typeof value.nonce === 'string') {
        return value.nonce.trim();
      }
      if (value.nonce && typeof value.nonce.nonce === 'string') {
        return value.nonce.nonce.trim();
      }
    }
    return '';
  }

  function storeGooglePromptNonce(value) {
    if (!window) {
      return;
    }
    var nonce = normalizeNonceValue(value);
    if (!nonce) {
      return;
    }
    window.__loopawareGooglePromptNonce = nonce;
    if (!window.__googleInitConfig) {
      window.__googleInitConfig = {};
    }
    window.__googleInitConfig.nonce = nonce;
  }

  function wrapRequestNonce(requestNonce) {
    if (typeof requestNonce !== 'function') {
      return requestNonce;
    }
    if (requestNonce.__loopawareNonceWrapper === true) {
      return requestNonce;
    }
    var wrapper = function() {
      var result;
      try {
        result = requestNonce.apply(this, arguments);
      } finally {
        try {
          if (result && typeof result.then === 'function') {
            result.then(storeGooglePromptNonce).catch(function() {});
          } else {
            storeGooglePromptNonce(result);
          }
        } catch (error) {}
      }
      return result;
    };
    wrapper.__loopawareNonceWrapper = true;
    return wrapper;
  }

  function ensureRequestNonceTracking() {
    if (!window) {
      return;
    }
    if (typeof window.requestNonce === 'function') {
      window.requestNonce = wrapRequestNonce(window.requestNonce);
      return;
    }
    if (window.__loopawareRequestNonceTracking === true) {
      return;
    }
    window.__loopawareRequestNonceTracking = true;
    try {
      Object.defineProperty(window, 'requestNonce', {
        configurable: true,
        enumerable: true,
        get: function() {
          return undefined;
        },
        set: function(value) {
          var wrapped = wrapRequestNonce(value);
          try {
            Object.defineProperty(window, 'requestNonce', {
              configurable: true,
              enumerable: true,
              writable: true,
              value: wrapped
            });
          } catch (error) {
            window.requestNonce = wrapped;
          }
        }
      });
    } catch (error) {}
  }

  function setGoogleSigninDisabled(target, disabled) {
    if (!target || typeof target.setAttribute !== 'function') {
      return;
    }
    if (disabled) {
      target.setAttribute('data-loopaware-signin-disabled', 'true');
      target.setAttribute('aria-disabled', 'true');
      if (target.tagName === 'BUTTON') {
        target.disabled = true;
      }
      if (target.style) {
        target.style.pointerEvents = 'none';
      }
      return;
    }
    target.removeAttribute('data-loopaware-signin-disabled');
    target.removeAttribute('aria-disabled');
    if (target.tagName === 'BUTTON') {
      target.disabled = false;
    }
    if (target.style && target.style.pointerEvents === 'none') {
      target.style.pointerEvents = '';
    }
  }

  function clearGoogleSigninGate(target) {
    if (target) {
      target.removeAttribute('data-loopaware-signin-gate');
      setGoogleSigninDisabled(target, false);
    }
    var headerHost = document.querySelector('mpr-header');
    if (!headerHost || typeof headerHost.querySelectorAll !== 'function') {
      return;
    }
    var disabledNodes = headerHost.querySelectorAll('[data-loopaware-signin-disabled="true"]');
    if (!disabledNodes) {
      return;
    }
    for (var index = 0; index < disabledNodes.length; index += 1) {
      var node = disabledNodes[index];
      if (!node || typeof node.removeAttribute !== 'function') {
        continue;
      }
      node.removeAttribute('data-loopaware-signin-gate');
      setGoogleSigninDisabled(node, false);
    }
  }

  function gateGoogleSigninUntilNonce(headerHost) {
    if (!headerHost) {
      return;
    }
    ensureRequestNonceTracking();
    var remainingAttempts = GOOGLE_SIGNIN_GATE_MAX_ATTEMPTS;
    function scheduleNextAttempt() {
      var interval = remainingAttempts > 0 ? GOOGLE_SIGNIN_GATE_POLL_INTERVAL_MS : GOOGLE_SIGNIN_GATE_SLOW_POLL_INTERVAL_MS;
      window.setTimeout(attemptGate, interval);
    }
    function attemptGate() {
      var target = resolveGoogleSigninTarget(headerHost);
      if (!target) {
        if (remainingAttempts > 0) {
          remainingAttempts -= 1;
        }
        scheduleNextAttempt();
        return;
      }
      if (hasGooglePromptNonce()) {
        clearGoogleSigninGate(target);
        return;
      }
      if (target.getAttribute('data-loopaware-signin-gate') !== 'true') {
        target.setAttribute('data-loopaware-signin-gate', 'true');
      }
      setGoogleSigninDisabled(target, true);
      if (remainingAttempts > 0) {
        remainingAttempts -= 1;
      }
      scheduleNextAttempt();
    }
    attemptGate();
  }

  function resolveAuthHost(event) {
    if (event && event.target && event.target.nodeType === 1 && typeof event.target.matches === 'function') {
      if (event.target.matches('mpr-header')) {
        return event.target;
      }
    }
    return document.querySelector('mpr-header');
  }

  function resolveHeaderRootElement(headerHost) {
    if (!headerHost || typeof headerHost.querySelector !== 'function') {
      return null;
    }
    return headerHost.querySelector('header.mpr-header');
  }

  function resolveLoopawareUserMenu(headerHost) {
    if (!headerHost || typeof headerHost.querySelector !== 'function') {
      return null;
    }
    return headerHost.querySelector('mpr-user[data-loopaware-user-menu="true"]');
  }

  function normalizeTextValue(value) {
    if (typeof value !== 'string') {
      return '';
    }
    return value.trim();
  }

  function resolveAvatarURL(profile) {
    if (!profile || typeof profile !== 'object') {
      return '';
    }
    if (typeof profile.avatar_url === 'string') {
      return profile.avatar_url.trim();
    }
    if (typeof profile.avatarURL === 'string') {
      return profile.avatarURL.trim();
    }
    if (typeof profile.picture === 'string') {
      return profile.picture.trim();
    }
    if (profile.avatar && typeof profile.avatar === 'object' && typeof profile.avatar.url === 'string') {
      return profile.avatar.url.trim();
    }
    if (typeof profile.url === 'string') {
      return profile.url.trim();
    }
    return '';
  }

  function normalizeProfile(profile) {
    if (!profile || typeof profile !== 'object') {
      return null;
    }
    var email = normalizeTextValue(profile.user_email || profile.email || '');
    var display = normalizeTextValue(profile.display || profile.user_display_name || profile.name || email);
    var avatarURL = resolveAvatarURL(profile);
    if (!email && !display && !avatarURL) {
      return null;
    }
    if (!display) {
      display = email;
    }
    return {
      email: email,
      display: display,
      avatarURL: avatarURL
    };
  }

  function clearHeaderProfileAttributes(headerHost) {
    if (!headerHost || typeof headerHost.removeAttribute !== 'function') {
      return;
    }
    if (headerHost.hasAttribute('data-user-display')) {
      headerHost.removeAttribute('data-user-display');
    }
    if (headerHost.hasAttribute('data-user-email')) {
      headerHost.removeAttribute('data-user-email');
    }
    if (headerHost.hasAttribute('data-user-avatar-url')) {
      headerHost.removeAttribute('data-user-avatar-url');
    }
  }

  function applyHeaderProfileAttributes(headerHost, profile) {
    if (!headerHost || typeof headerHost.setAttribute !== 'function') {
      return;
    }
    if (!profile) {
      clearHeaderProfileAttributes(headerHost);
      return;
    }
    if (profile.display) {
      if (headerHost.getAttribute('data-user-display') !== profile.display) {
        headerHost.setAttribute('data-user-display', profile.display);
      }
    } else if (headerHost.hasAttribute('data-user-display')) {
      headerHost.removeAttribute('data-user-display');
    }
    if (profile.email) {
      if (headerHost.getAttribute('data-user-email') !== profile.email) {
        headerHost.setAttribute('data-user-email', profile.email);
      }
    } else if (headerHost.hasAttribute('data-user-email')) {
      headerHost.removeAttribute('data-user-email');
    }
    if (profile.avatarURL) {
      if (headerHost.getAttribute('data-user-avatar-url') !== profile.avatarURL) {
        headerHost.setAttribute('data-user-avatar-url', profile.avatarURL);
      }
    } else if (headerHost.hasAttribute('data-user-avatar-url')) {
      headerHost.removeAttribute('data-user-avatar-url');
    }
  }

  function setHeaderAuthStateAttribute(headerHost, stateValue) {
    if (!headerHost || typeof headerHost.setAttribute !== 'function') {
      return;
    }
    if (headerHost.getAttribute('data-loopaware-auth-state') !== stateValue) {
      headerHost.setAttribute('data-loopaware-auth-state', stateValue);
    }
  }

  function applyHeaderAuthenticatedClass(headerHost, authenticated) {
    var headerRoot = resolveHeaderRootElement(headerHost);
    if (!headerRoot || !headerRoot.classList) {
      return;
    }
    headerRoot.classList.toggle('mpr-header--authenticated', authenticated);
  }

  function applySnapshot(headerHost, snapshot) {
    if (!headerHost) {
      return snapshot;
    }
    applyHeaderAuthenticatedClass(headerHost, snapshot.status === AUTH_STATE_VALUES.authenticated);
    setHeaderAuthStateAttribute(headerHost, snapshot.status);
    if (snapshot.status === AUTH_STATE_VALUES.authenticated) {
      applyHeaderProfileAttributes(headerHost, snapshot.profile);
      clearGoogleSigninGate(resolveGoogleSigninTarget(headerHost));
      hideOverlay();
      if (shouldRedirectToApp(headerHost, snapshot)) {
        redirectTo(APP_PATHNAME);
      }
      return snapshot;
    }
    clearHeaderProfileAttributes(headerHost);
    if (snapshot.status === AUTH_STATE_VALUES.unauthenticated) {
      disableGoogleAutoSelect();
      gateGoogleSigninUntilNonce(headerHost);
      if (shouldRedirectToLogin(headerHost)) {
        showOverlay();
        redirectTo(LOGIN_PATHNAME);
        return snapshot;
      }
      hideOverlay();
      return snapshot;
    }
    gateGoogleSigninUntilNonce(headerHost);
    hideOverlay();
    return snapshot;
  }

  function commitSnapshot(headerHost, snapshot) {
    var normalizedProfile = snapshot && snapshot.profile ? normalizeProfile(snapshot.profile) : null;
    var normalizedSnapshot = createSnapshot(snapshot.status, normalizedProfile, snapshot.source);
    if (snapshotsEqual(store.snapshot, normalizedSnapshot)) {
      return store.snapshot || normalizedSnapshot;
    }
    store.snapshot = normalizedSnapshot;
    applySnapshot(headerHost, normalizedSnapshot);
    dispatchAuthStateChange(headerHost, normalizedSnapshot);
    return normalizedSnapshot;
  }

  function syncFromObservedState(headerHost) {
    if (!headerHost) {
      return createSnapshot(AUTH_STATE_VALUES.syncing, null, 'dom');
    }
    return commitSnapshot(headerHost, resolveObservedSnapshot(headerHost));
  }

  function syncFromAuthenticatedProfile(headerHost, profile, source) {
    var normalizedProfile = normalizeProfile(profile);
    if (!normalizedProfile) {
      return syncFromObservedState(headerHost);
    }
    return commitSnapshot(headerHost, createSnapshot(AUTH_STATE_VALUES.authenticated, normalizedProfile, source));
  }

  function syncFromUnauthenticatedState(headerHost, source) {
    return commitSnapshot(headerHost, createSnapshot(AUTH_STATE_VALUES.unauthenticated, null, source));
  }

  function readHeaderProfileAttributes(headerHost) {
    if (!headerHost || typeof headerHost.getAttribute !== 'function') {
      return null;
    }
    return normalizeProfile({
      email: headerHost.getAttribute('data-user-email'),
      display: headerHost.getAttribute('data-user-display'),
      avatar_url: headerHost.getAttribute('data-user-avatar-url')
    });
  }

  function readUserMenuProfile(userMenu) {
    if (!userMenu || typeof userMenu.getAttribute !== 'function') {
      return null;
    }
    return normalizeProfile({
      email: userMenu.getAttribute('data-user-email'),
      display: userMenu.getAttribute('data-user-display'),
      avatar_url: userMenu.getAttribute('data-user-avatar-url')
    });
  }

  function createSnapshot(status, profile, source) {
    return {
      status: status,
      profile: profile || null,
      source: source || ''
    };
  }

  function snapshotsEqual(left, right) {
    if (left === right) {
      return true;
    }
    if (!left || !right) {
      return false;
    }
    var leftProfile = left.profile || null;
    var rightProfile = right.profile || null;
    return left.status === right.status &&
      ((leftProfile === null && rightProfile === null) ||
        (leftProfile && rightProfile &&
          leftProfile.email === rightProfile.email &&
          leftProfile.display === rightProfile.display &&
          leftProfile.avatarURL === rightProfile.avatarURL));
  }

  function resolveObservedSnapshot(headerHost) {
    if (!headerHost) {
      return createSnapshot(AUTH_STATE_VALUES.syncing, null, 'dom');
    }
    var userMenu = resolveLoopawareUserMenu(headerHost);
    var userMenuStatus = userMenu && typeof userMenu.getAttribute === 'function'
      ? normalizeTextValue(userMenu.getAttribute('data-mpr-user-status'))
      : '';
    var profile = readHeaderProfileAttributes(headerHost) || readUserMenuProfile(userMenu);
    if (profile) {
      return createSnapshot(AUTH_STATE_VALUES.authenticated, profile, 'dom');
    }
    if (userMenuStatus === AUTH_STATE_VALUES.unauthenticated) {
      return createSnapshot(AUTH_STATE_VALUES.unauthenticated, null, 'dom');
    }
    return createSnapshot(AUTH_STATE_VALUES.syncing, null, 'dom');
  }

  function shouldRedirectToApp(headerHost, snapshot) {
    if (!headerHost || !snapshot || snapshot.status !== AUTH_STATE_VALUES.authenticated) {
      return false;
    }
    if (headerHost.getAttribute('data-loopaware-auth-redirect') !== 'true') {
      return false;
    }
    if (!window.location || typeof window.location.pathname !== 'string') {
      return true;
    }
    return window.location.pathname !== APP_PATHNAME && window.location.pathname !== APP_PATHNAME + '/';
  }

  function shouldRedirectToLogin(headerHost) {
    return !!(headerHost && typeof headerHost.getAttribute === 'function' && headerHost.getAttribute('data-loopaware-auth-redirect-on-logout') === 'true');
  }

  function redirectTo(pathname) {
    if (!window.location || typeof window.location.assign !== 'function') {
      return;
    }
    if (store.redirectTarget === pathname) {
      return;
    }
    store.redirectTarget = pathname;
    window.location.assign(pathname);
  }

  function dispatchAuthStateChange(headerHost, snapshot) {
    if (!headerHost || typeof headerHost.dispatchEvent !== 'function' || typeof CustomEvent !== 'function') {
      return;
    }
    var detail = {
      status: snapshot.status,
      profile: snapshot.profile,
      source: snapshot.source
    };
    try {
      headerHost.dispatchEvent(new CustomEvent(AUTH_STATE_CHANGE_EVENT, {
        detail: detail,
        bubbles: true
      }));
    } catch (error) {}
  }


  function scheduleHelperSync(headerHost) {
    if (!headerHost || store.helperSyncStarted === true) {
      return;
    }
    store.helperSyncStarted = true;
    var remainingAttempts = HELPER_SYNC_MAX_ATTEMPTS;

    function finalizeFallback() {
      if (!store.snapshot || store.snapshot.status === AUTH_STATE_VALUES.syncing) {
        syncFromUnauthenticatedState(headerHost, 'helper-timeout');
      }
    }

    function syncAttempt() {
      if (!headerHost || !document.contains(headerHost)) {
        return;
      }
      if (typeof window.getCurrentUser !== 'function') {
        if (remainingAttempts > 0) {
          remainingAttempts -= 1;
          window.setTimeout(syncAttempt, HELPER_SYNC_INTERVAL_MS);
          return;
        }
        finalizeFallback();
        return;
      }
      var helperResult;
      try {
        helperResult = window.getCurrentUser();
      } catch (error) {
        syncFromUnauthenticatedState(headerHost, 'helper-error');
        return;
      }
      Promise.resolve(helperResult).then(function(profile) {
        var normalizedProfile = normalizeProfile(profile);
        if (normalizedProfile) {
          syncFromAuthenticatedProfile(headerHost, normalizedProfile, 'helper');
          return;
        }
        if (!store.snapshot || store.snapshot.status !== AUTH_STATE_VALUES.authenticated) {
          syncFromUnauthenticatedState(headerHost, 'helper');
        }
      }).catch(function() {
        if (!store.snapshot || store.snapshot.status !== AUTH_STATE_VALUES.authenticated) {
          syncFromUnauthenticatedState(headerHost, 'helper-error');
        }
      });
    }

    syncAttempt();
  }

  function observeHeaderState(headerHost) {
    if (!headerHost || headerHost.__loopawareAuthObserver || typeof MutationObserver !== 'function') {
      return;
    }
    var observer = new MutationObserver(function() {
      syncFromObservedState(headerHost);
    });
    observer.observe(headerHost, {
      subtree: true,
      childList: true,
      attributes: true,
      attributeFilter: ['data-mpr-user-status', 'data-user-display', 'data-user-email', 'data-user-avatar-url']
    });
    headerHost.__loopawareAuthObserver = observer;
  }

  function openAccountSettingsModal() {
    var modalElement = document.getElementById('settings-modal');
    if (!modalElement) {
      return false;
    }
    if (window.bootstrap && window.bootstrap.Modal && typeof window.bootstrap.Modal.getOrCreateInstance === 'function') {
      var modalInstance = window.bootstrap.Modal.getOrCreateInstance(modalElement);
      if (modalInstance && typeof modalInstance.show === 'function') {
        modalInstance.show();
        return true;
      }
    }
    return false;
  }

  function handleUserMenuItem(event) {
    if (!event || !event.detail || event.detail.action !== 'account-settings') {
      return;
    }
    if (typeof event.preventDefault === 'function') {
      event.preventDefault();
    }
    openAccountSettingsModal();
  }

  function handleHeaderSettingsClick(event) {
    if (event && typeof event.preventDefault === 'function') {
      event.preventDefault();
    }
    openAccountSettingsModal();
  }

  function handleUserMenuLogout() {
    showOverlay();
    disableGoogleAutoSelect();
  }

  function handleAuthenticatedEvent(event) {
    var headerHost = resolveAuthHost(event);
    if (!headerHost) {
      return;
    }
    pruneHeaderUserMenus(headerHost);
    var profile = event && event.detail ? event.detail.profile : null;
    syncFromAuthenticatedProfile(headerHost, profile, 'event');
  }

  function handleUnauthenticatedEvent(event) {
    var headerHost = resolveAuthHost(event);
    if (!headerHost) {
      hideOverlay();
      return;
    }
    syncFromUnauthenticatedState(headerHost, 'event');
  }

  var authListenersAttached = false;
  var userMenuListenersAttached = false;

  function attachUserMenuListeners() {
    if (userMenuListenersAttached || !document || typeof document.addEventListener !== 'function') {
      return;
    }
    document.addEventListener('mpr-user:menu-item', handleUserMenuItem);
    document.addEventListener('mpr-ui:header:settings-click', handleHeaderSettingsClick);
    document.addEventListener('mpr-user:logout', handleUserMenuLogout);
    userMenuListenersAttached = true;
  }

  function attachHeaderAuth(headerHost) {
    attachUserMenuListeners();
    injectAuthStyles();
    if (!authListenersAttached && document && typeof document.addEventListener === 'function') {
      document.addEventListener('mpr-ui:auth:authenticated', handleAuthenticatedEvent);
      document.addEventListener('mpr-ui:auth:unauthenticated', handleUnauthenticatedEvent);
      authListenersAttached = true;
    }
    if (!headerHost) {
      return;
    }
    pruneHeaderUserMenus(headerHost);
    ensureLogoutFallback(headerHost);
    if (typeof headerHost.addEventListener === 'function' && headerHost.getAttribute('data-loopaware-auth-listeners') !== 'true') {
      headerHost.setAttribute('data-loopaware-auth-listeners', 'true');
      headerHost.addEventListener('mpr-ui:auth:authenticated', handleAuthenticatedEvent);
      headerHost.addEventListener('mpr-ui:auth:unauthenticated', handleUnauthenticatedEvent);
    }
    headerHost.setAttribute('data-loopaware-auth-bound', 'true');
    headerHost.setAttribute('data-loopaware-auth-state', AUTH_STATE_VALUES.syncing);
    observeHeaderState(headerHost);
    syncFromObservedState(headerHost);
    scheduleHelperSync(headerHost);
  }

  var bindingInProgress = false;

  function bindHeaderAuth() {
    if (bindingInProgress) {
      return;
    }
    bindingInProgress = true;
    var remainingAttempts = 120;

    function attemptBind() {
      var headerHost = document.querySelector('mpr-header');
      if (headerHost && headerHost.getAttribute('data-loopaware-auth-bound') !== 'true') {
        attachHeaderAuth(headerHost);
      }
      remainingAttempts -= 1;
      if (remainingAttempts > 0) {
        window.setTimeout(attemptBind, 100);
        return;
      }
      bindingInProgress = false;
    }

    attemptBind();
  }

  bindHeaderAuth();
  document.addEventListener('DOMContentLoaded', bindHeaderAuth);
})();
