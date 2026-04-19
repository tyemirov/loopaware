// @ts-check
(function () {
  if (typeof document !== 'undefined' && document.documentElement) {
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
  var PUBLIC_AUTH_RECOVERY_POLL_INTERVAL_MS = 500;
  var PUBLIC_AUTH_RECOVERY_MAX_ATTEMPTS = 120;
  var PUBLIC_AUTH_SCREEN_ELEMENT_ID = 'loopaware-public-auth-screen';
  var DEFAULT_PUBLIC_AUTH_TITLE = 'Opening LoopAware';
  var DEFAULT_PUBLIC_AUTH_MESSAGE = 'Preparing your authenticated workspace.';
  var LOGOUT_REQUEST_TIMEOUT_MS = 2000;
  var INITIAL_APP_AUTH_SETTLE_TIMEOUT_MS = 3000;
  var APP_PATHNAME = '/app';
  var LOGIN_PATHNAME = '/login';
  var EXPLICIT_LOGOUT_STORAGE_KEY = 'loopaware_explicit_logout';
  var store = window.__loopawareHeaderAuthStore;
  var authListenersAttached = false;
  var userMenuListenersAttached = false;
  var publicAuthRecoveryListenersAttached = false;
  var publicSigninListenersAttached = false;
  var bindingInProgress = false;

  if (!store || typeof store !== 'object') {
    store = {
      logoutPending: false,
      snapshot: null,
      redirectTarget: '',
      loginRedirectPending: false,
      runtimeRefreshPending: false,
      publicAuthRecoveryTimerId: 0,
      publicAuthRecoveryRemainingAttempts: 0
    };
    window.__loopawareHeaderAuthStore = store;
  }

  function resolveExplicitLogoutStorage() {
    if (!window) {
      return null;
    }
    try {
      if (window.sessionStorage) {
        return window.sessionStorage;
      }
    } catch (error) {}
    try {
      if (window.localStorage) {
        return window.localStorage;
      }
    } catch (error) {}
    return null;
  }

  function hasExplicitLogoutState() {
    var storage = resolveExplicitLogoutStorage();
    if (!storage || typeof storage.getItem !== 'function') {
      return false;
    }
    try {
      return storage.getItem(EXPLICIT_LOGOUT_STORAGE_KEY) === 'true';
    } catch (error) {
      return false;
    }
  }

  function syncExplicitLogoutState(headerHost) {
    if (!headerHost || typeof headerHost.toggleAttribute !== 'function') {
      return;
    }
    headerHost.toggleAttribute('data-loopaware-explicit-logout', hasExplicitLogoutState());
  }

  function markExplicitLogoutState() {
    var storage = resolveExplicitLogoutStorage();
    if (storage && typeof storage.setItem === 'function') {
      try {
        storage.setItem(EXPLICIT_LOGOUT_STORAGE_KEY, 'true');
      } catch (error) {}
    }
    syncExplicitLogoutState(document.querySelector('mpr-header'));
  }

  function clearExplicitLogoutState() {
    var storage = resolveExplicitLogoutStorage();
    if (storage && typeof storage.removeItem === 'function') {
      try {
        storage.removeItem(EXPLICIT_LOGOUT_STORAGE_KEY);
      } catch (error) {}
    }
    syncExplicitLogoutState(document.querySelector('mpr-header'));
  }

  function createSnapshot(status, source) {
    return {
      status: status,
      source: source || ''
    };
  }

  function resolveAuthSettleState(headerHost, shouldCreate) {
    if (!headerHost) {
      return null;
    }
    var settleState = headerHost.__loopawareAuthSettleState;
    if (!settleState && shouldCreate) {
      settleState = {
        settled: false,
        timeoutId: 0
      };
      headerHost.__loopawareAuthSettleState = settleState;
    }
    return settleState || null;
  }

  function resolveLogoutOverlay() {
    if (!document || typeof document.getElementById !== 'function') {
      return null;
    }
    return document.getElementById('logout-overlay');
  }

  function setBodyLogoutState(isActive) {
    if (!document || !document.body || !document.body.classList) {
      return;
    }
    document.body.classList.toggle('logging-out', isActive === true);
  }

  function showOverlay() {
    if (typeof window.showLogoutOverlay === 'function') {
      window.showLogoutOverlay();
      return;
    }
    var overlay = resolveLogoutOverlay();
    if (overlay) {
      overlay.classList.remove('d-none');
      overlay.classList.add('d-flex');
      overlay.style.display = 'flex';
    }
    setBodyLogoutState(true);
  }

  function hideOverlay() {
    if (typeof window.hideLogoutOverlay === 'function') {
      window.hideLogoutOverlay();
      return;
    }
    var overlay = resolveLogoutOverlay();
    if (overlay) {
      overlay.classList.add('d-none');
      overlay.classList.remove('d-flex');
      overlay.style.display = '';
    }
    setBodyLogoutState(false);
  }

  function resolvePublicAuthMain() {
    if (!document || typeof document.querySelector !== 'function') {
      return null;
    }
    return document.querySelector('main');
  }

  function resolvePublicAuthScreen() {
    if (!document || typeof document.getElementById !== 'function') {
      return null;
    }
    return document.getElementById(PUBLIC_AUTH_SCREEN_ELEMENT_ID);
  }

  function parseHeaderTransitionConfig(headerHost) {
    if (!headerHost || typeof headerHost.getAttribute !== 'function') {
      return {};
    }
    var rawValue = headerHost.getAttribute('auth-transition');
    if (!rawValue) {
      return {};
    }
    try {
      var parsedValue = JSON.parse(rawValue);
      if (parsedValue && typeof parsedValue === 'object') {
        return parsedValue;
      }
    } catch (error) {}
    return {};
  }

  function resolvePublicAuthScreenCopy(headerHost) {
    var transitionConfig = parseHeaderTransitionConfig(headerHost);
    var title = normalizeTextValue(transitionConfig.title) || DEFAULT_PUBLIC_AUTH_TITLE;
    var message = normalizeTextValue(transitionConfig.message) || DEFAULT_PUBLIC_AUTH_MESSAGE;
    return {
      title: title,
      message: message
    };
  }

  function ensurePublicAuthScreen(headerHost) {
    var mainElement = resolvePublicAuthMain();
    if (!mainElement || typeof document.createElement !== 'function') {
      return null;
    }
    var screenElement = resolvePublicAuthScreen();
    if (!screenElement) {
      screenElement = document.createElement('section');
      screenElement.id = PUBLIC_AUTH_SCREEN_ELEMENT_ID;
      screenElement.className = 'loopaware-public-auth-screen';
      screenElement.hidden = true;
      screenElement.setAttribute('aria-hidden', 'true');
      screenElement.setAttribute('aria-live', 'polite');
      screenElement.setAttribute('data-loopaware-visible', 'false');

      var contentElement = document.createElement('div');
      contentElement.className = 'loopaware-public-auth-screen__content';

      var spinnerElement = document.createElement('div');
      spinnerElement.className = 'spinner-border text-primary loopaware-public-auth-screen__spinner';
      spinnerElement.setAttribute('role', 'status');

      var spinnerLabel = document.createElement('span');
      spinnerLabel.className = 'visually-hidden';
      spinnerLabel.textContent = 'Signing in...';
      spinnerElement.appendChild(spinnerLabel);

      var titleElement = document.createElement('h1');
      titleElement.className = 'h3 fw-semibold mb-3';
      titleElement.setAttribute('data-loopaware-public-auth-screen-copy', 'title');

      var messageElement = document.createElement('p');
      messageElement.className = 'lead mb-0 loopaware-public-auth-screen__message';
      messageElement.setAttribute('data-loopaware-public-auth-screen-copy', 'message');

      contentElement.appendChild(spinnerElement);
      contentElement.appendChild(titleElement);
      contentElement.appendChild(messageElement);
      screenElement.appendChild(contentElement);
      mainElement.appendChild(screenElement);
    }

    var copy = resolvePublicAuthScreenCopy(headerHost);
    var titleTarget = screenElement.querySelector('[data-loopaware-public-auth-screen-copy="title"]');
    var messageTarget = screenElement.querySelector('[data-loopaware-public-auth-screen-copy="message"]');
    if (titleTarget) {
      titleTarget.textContent = copy.title;
    }
    if (messageTarget) {
      messageTarget.textContent = copy.message;
    }
    return screenElement;
  }

  function showPublicAuthScreen(headerHost) {
    if (!document || !document.body || typeof document.body.setAttribute !== 'function') {
      return;
    }
    var mainElement = resolvePublicAuthMain();
    var screenElement = ensurePublicAuthScreen(headerHost);
    if (!mainElement || !screenElement) {
      return;
    }
    document.body.setAttribute('data-loopaware-public-auth-screen', 'true');
    mainElement.setAttribute('aria-busy', 'true');
    screenElement.hidden = false;
    screenElement.setAttribute('aria-hidden', 'false');
    screenElement.setAttribute('data-loopaware-visible', 'true');
  }

  function hidePublicAuthScreen() {
    if (!document || !document.body || typeof document.body.removeAttribute !== 'function') {
      return;
    }
    var mainElement = resolvePublicAuthMain();
    var screenElement = resolvePublicAuthScreen();
    document.body.removeAttribute('data-loopaware-public-auth-screen');
    if (mainElement) {
      mainElement.removeAttribute('aria-busy');
    }
    if (!screenElement) {
      return;
    }
    screenElement.hidden = true;
    screenElement.setAttribute('aria-hidden', 'true');
    screenElement.setAttribute('data-loopaware-visible', 'false');
  }

  function markLogoutPending() {
    store.logoutPending = true;
  }

  function clearLogoutPending() {
    store.logoutPending = false;
  }

  function isLogoutPending() {
    return store.logoutPending === true;
  }

  function startLogoutTransition() {
    markExplicitLogoutState();
    markLogoutPending();
    showOverlay();
  }

  function cancelLogoutTransition() {
    clearLogoutPending();
    hideOverlay();
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
      '}' +
      'mpr-header[data-loopaware-explicit-logout] [data-mpr-header="auth-transition"]{' +
      'display:none !important;' +
      '}' +
      '#' + PUBLIC_AUTH_SCREEN_ELEMENT_ID + '{' +
      'display:none;' +
      'width:100%;' +
      'min-height:clamp(18rem,42vh,28rem);' +
      'padding:3rem 1.5rem;' +
      '}' +
      '#' + PUBLIC_AUTH_SCREEN_ELEMENT_ID + '[data-loopaware-visible="true"]{' +
      'display:flex;' +
      'align-items:center;' +
      'justify-content:center;' +
      '}' +
      '#' + PUBLIC_AUTH_SCREEN_ELEMENT_ID + ' .loopaware-public-auth-screen__content{' +
      'width:min(100%,32rem);' +
      'margin:0 auto;' +
      'text-align:center;' +
      '}' +
      '#' + PUBLIC_AUTH_SCREEN_ELEMENT_ID + ' .loopaware-public-auth-screen__spinner{' +
      'margin:0 auto 1rem auto;' +
      '}' +
      '#' + PUBLIC_AUTH_SCREEN_ELEMENT_ID + ' .loopaware-public-auth-screen__message{' +
      'max-width:28rem;' +
      'margin:0 auto;' +
      'color:var(--bs-secondary-color);' +
      '}' +
      'body[data-loopaware-public-auth-screen="true"] main{' +
      'display:flex !important;' +
      'align-items:center;' +
      '}' +
      'body[data-loopaware-public-auth-screen="true"] main > :not(#' + PUBLIC_AUTH_SCREEN_ELEMENT_ID + '){' +
      'display:none !important;' +
      '}' +
      'mpr-header[data-loopaware-public-auth-screen-mode] [data-mpr-header="auth-transition"]{' +
      'display:none !important;' +
      '}';
    var head = document.head || document.documentElement;
    if (!head || typeof head.appendChild !== 'function') {
      return;
    }
    head.appendChild(styleElement);
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
    return new Promise(function (resolve) {
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
      window.setTimeout(function () {
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

  function createLogoutTimeoutError() {
    var error = new Error('loopaware.logout_timeout');
    error.code = 'loopaware.logout_timeout';
    return error;
  }

  function withTimeout(task, timeoutMs) {
    return new Promise(function (resolve, reject) {
      var settled = false;
      var timeoutId = window.setTimeout(function () {
        if (settled) {
          return;
        }
        settled = true;
        reject(createLogoutTimeoutError());
      }, timeoutMs);
      Promise.resolve()
        .then(task)
        .then(function (result) {
          if (settled) {
            return;
          }
          settled = true;
          window.clearTimeout(timeoutId);
          resolve(result);
        })
        .catch(function (error) {
          if (settled) {
            return;
          }
          settled = true;
          window.clearTimeout(timeoutId);
          reject(error);
        });
    });
  }

  function performLogoutRequest(headerHost, logoutDelegate) {
    var logoutURL = resolveLogoutURL(headerHost);
    var logoutRequest = function () {
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
        return Promise.resolve(logoutDelegate()).catch(function () {});
      } catch (error) {
        return Promise.resolve();
      }
    }
    var logoutWithFetchFallback = function () {
      return withTimeout(logoutRequest, LOGOUT_REQUEST_TIMEOUT_MS)
        .then(assertLogoutResponseOk)
        .catch(function (error) {
          if (error && typeof error.status === 'number' && error.status) {
            throw error;
          }
          return submitLogoutForm(logoutURL);
        });
    };
    return logoutWithFetchFallback()
      .then(function (result) {
        return invokeLogoutDelegate().then(function () {
          return result;
        });
      })
      .catch(function (error) {
        cancelLogoutTransition();
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
      existingWrapper = function () {
        startLogoutTransition();
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
        get: function () {
          return window.__loopawareLogoutWrapper || existingWrapper;
        },
        set: function (value) {
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
    var wrapper = function () {
      var result;
      try {
        result = requestNonce.apply(this, arguments);
      } finally {
        try {
          if (result && typeof result.then === 'function') {
            result.then(storeGooglePromptNonce).catch(function () {});
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
        get: function () {
          return undefined;
        },
        set: function (value) {
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

  function resolveAuthController(headerHost) {
    if (!headerHost || typeof headerHost !== 'object') {
      return null;
    }
    var authController = headerHost.__authController;
    if (!authController || typeof authController.prepareGoogleNonce !== 'function') {
      return null;
    }
    return authController;
  }

  function forceHeaderAuthRefresh(headerHost) {
    if (!headerHost || typeof headerHost.getAttribute !== 'function' || typeof headerHost.setAttribute !== 'function') {
      return;
    }
    var signInLabel = headerHost.getAttribute('sign-in-label');
    if (typeof signInLabel !== 'string') {
      return;
    }
    headerHost.setAttribute('sign-in-label', signInLabel + ' ');
    headerHost.setAttribute('sign-in-label', signInLabel);
  }

  function primeGoogleSigninNonce(headerHost) {
    var authController = resolveAuthController(headerHost);
    if (authController) {
      try {
        Promise.resolve(authController.prepareGoogleNonce()).catch(function () {});
      } catch (error) {}
      return;
    }
    forceHeaderAuthRefresh(headerHost);
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
    var target = event && event.target && event.target.nodeType === 1 ? event.target : null;
    if (target && typeof target.matches === 'function' && target.matches('mpr-header')) {
      return target;
    }
    if (target && typeof target.closest === 'function') {
      var scopedHost = target.closest('mpr-header');
      if (scopedHost) {
        return scopedHost;
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

  function resolveObservedSnapshot(headerHost) {
    if (!headerHost) {
      return createSnapshot(AUTH_STATE_VALUES.syncing, 'dom');
    }
    var headerRoot = resolveHeaderRootElement(headerHost);
    var userMenu = resolveLoopawareUserMenu(headerHost);
    var userMenuStatus = userMenu && typeof userMenu.getAttribute === 'function'
      ? normalizeTextValue(userMenu.getAttribute('data-mpr-user-status'))
      : '';
    var headerAuthenticated = !!(headerRoot && headerRoot.classList && headerRoot.classList.contains('mpr-header--authenticated'));

    if (headerAuthenticated || userMenuStatus === AUTH_STATE_VALUES.authenticated) {
      return createSnapshot(AUTH_STATE_VALUES.authenticated, 'dom');
    }
    if (userMenuStatus === AUTH_STATE_VALUES.unauthenticated || userMenuStatus === 'error') {
      return createSnapshot(AUTH_STATE_VALUES.unauthenticated, 'dom');
    }
    return createSnapshot(AUTH_STATE_VALUES.syncing, 'dom');
  }

  function setHeaderAuthStateAttribute(headerHost, stateValue) {
    if (!headerHost || typeof headerHost.setAttribute !== 'function') {
      return;
    }
    if (headerHost.getAttribute('data-loopaware-auth-state') !== stateValue) {
      headerHost.setAttribute('data-loopaware-auth-state', stateValue);
    }
  }

  function shouldRedirectToApp(headerHost, snapshot) {
    if (!headerHost || !snapshot || snapshot.status !== AUTH_STATE_VALUES.authenticated) {
      return false;
    }
    if (hasExplicitLogoutState() && headerHost.getAttribute('data-loopaware-auth-redirect') === 'true') {
      return false;
    }
    if (store.loginRedirectPending !== true && headerHost.getAttribute('data-loopaware-auth-redirect') !== 'true') {
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

  function isPublicAuthScreenMode(headerHost) {
    return !!(headerHost && !shouldRedirectToLogin(headerHost));
  }

  function syncPublicAuthScreenMode(headerHost) {
    if (!headerHost || typeof headerHost.toggleAttribute !== 'function') {
      return;
    }
    headerHost.toggleAttribute('data-loopaware-public-auth-screen-mode', isPublicAuthScreenMode(headerHost));
  }

  function supportsPublicSigninRecovery(headerHost) {
    if (!headerHost || typeof headerHost.getAttribute !== 'function') {
      return false;
    }
    if (shouldRedirectToLogin(headerHost)) {
      return false;
    }
    return !!(
      normalizeTextValue(headerHost.getAttribute('google-site-id')) &&
      normalizeTextValue(headerHost.getAttribute('tauth-login-path'))
    );
  }

  function shouldRefreshPublicAuthSession(headerHost) {
    if (!headerHost || typeof headerHost.getAttribute !== 'function') {
      return false;
    }
    if (headerHost.getAttribute('data-loopaware-auth-redirect') === 'true') {
      return true;
    }
    return supportsPublicSigninRecovery(headerHost);
  }

  function clearPublicAuthRecoveryPolling() {
    if (store.publicAuthRecoveryTimerId) {
      window.clearTimeout(store.publicAuthRecoveryTimerId);
      store.publicAuthRecoveryTimerId = 0;
    }
    store.publicAuthRecoveryRemainingAttempts = 0;
  }

  function resetPublicAuthRecovery() {
    clearPublicAuthRecoveryPolling();
    store.loginRedirectPending = false;
  }

  function markAppAuthSettled(headerHost) {
    var settleState = resolveAuthSettleState(headerHost, false);
    if (!settleState) {
      return;
    }
    settleState.settled = true;
    if (settleState.timeoutId) {
      window.clearTimeout(settleState.timeoutId);
      settleState.timeoutId = 0;
    }
  }

  function isAppAuthSettled(headerHost) {
    if (!shouldRedirectToLogin(headerHost)) {
      return true;
    }
    var settleState = resolveAuthSettleState(headerHost, false);
    return !!(settleState && settleState.settled === true);
  }

  function ensureAppAuthSettling(headerHost) {
    if (!shouldRedirectToLogin(headerHost)) {
      return null;
    }
    var settleState = resolveAuthSettleState(headerHost, true);
    if (settleState.settled === true || settleState.timeoutId) {
      return settleState;
    }
    settleState.timeoutId = window.setTimeout(function () {
      settleState.timeoutId = 0;
      settleState.settled = true;
      syncFromObservedState(headerHost);
    }, INITIAL_APP_AUTH_SETTLE_TIMEOUT_MS);
    return settleState;
  }

  function redirectTo(pathname) {
    if (!window.location || typeof window.location.assign !== 'function') {
      return;
    }
    if (store.redirectTarget === pathname) {
      return;
    }
    store.redirectTarget = pathname;
    if (pathname === APP_PATHNAME) {
      clearExplicitLogoutState();
      resetPublicAuthRecovery();
    }
    window.location.assign(pathname);
  }

  function normalizeSnapshotForExplicitLogout(headerHost, snapshot) {
    syncExplicitLogoutState(headerHost);
    if (!snapshot || snapshot.status !== AUTH_STATE_VALUES.authenticated) {
      return snapshot;
    }
    if (!headerHost || typeof headerHost.getAttribute !== 'function') {
      return snapshot;
    }
    if (headerHost.getAttribute('data-loopaware-auth-redirect') !== 'true' || !hasExplicitLogoutState()) {
      return snapshot;
    }
    return createSnapshot(AUTH_STATE_VALUES.unauthenticated, snapshot.source || 'explicit-logout');
  }

  function dispatchAuthStateChange(headerHost, snapshot) {
    if (!headerHost || typeof headerHost.dispatchEvent !== 'function' || typeof CustomEvent !== 'function') {
      return;
    }
    try {
      headerHost.dispatchEvent(new CustomEvent(AUTH_STATE_CHANGE_EVENT, {
        detail: {
          status: snapshot.status,
          source: snapshot.source
        },
        bubbles: true
      }));
    } catch (error) {}
  }

  function applySnapshot(headerHost, snapshot) {
    if (!headerHost) {
      return;
    }
    if (isPublicAuthScreenMode(headerHost) && (
      store.loginRedirectPending === true ||
      (snapshot && snapshot.status === AUTH_STATE_VALUES.authenticated && shouldRedirectToApp(headerHost, snapshot))
    )) {
      showPublicAuthScreen(headerHost);
    } else {
      hidePublicAuthScreen();
    }
    syncExplicitLogoutState(headerHost);
    setHeaderAuthStateAttribute(headerHost, snapshot.status);
    if (snapshot.status === AUTH_STATE_VALUES.authenticated) {
      clearGoogleSigninGate(resolveGoogleSigninTarget(headerHost));
      if (!isLogoutPending()) {
        hideOverlay();
      }
      if (shouldRedirectToApp(headerHost, snapshot)) {
        redirectTo(APP_PATHNAME);
      }
      return;
    }
    if (snapshot.status === AUTH_STATE_VALUES.unauthenticated) {
      clearLogoutPending();
      disableGoogleAutoSelect();
      gateGoogleSigninUntilNonce(headerHost);
      if (shouldRedirectToLogin(headerHost)) {
        markExplicitLogoutState();
        showOverlay();
        redirectTo(LOGIN_PATHNAME);
        return;
      }
      hideOverlay();
      return;
    }
    gateGoogleSigninUntilNonce(headerHost);
    if (!isLogoutPending()) {
      hideOverlay();
    }
  }

  function commitSnapshot(headerHost, snapshot) {
    snapshot = normalizeSnapshotForExplicitLogout(headerHost, snapshot);
    var previousSnapshot = store.snapshot;
    var shouldPrimeGoogleNonce = !!(
      previousSnapshot &&
      previousSnapshot.status === AUTH_STATE_VALUES.authenticated &&
      snapshot &&
      snapshot.status === AUTH_STATE_VALUES.unauthenticated
    );
    store.snapshot = snapshot;
    applySnapshot(headerHost, snapshot);
    if (shouldPrimeGoogleNonce) {
      primeGoogleSigninNonce(headerHost);
    }
    if (!previousSnapshot || previousSnapshot.status !== snapshot.status) {
      dispatchAuthStateChange(headerHost, snapshot);
    }
    return snapshot;
  }

  function syncFromObservedState(headerHost) {
    var observedSnapshot = resolveObservedSnapshot(headerHost);
    var previousSnapshot = store.snapshot;
    if (
      observedSnapshot.status === AUTH_STATE_VALUES.unauthenticated &&
      shouldRedirectToLogin(headerHost) &&
      !isAppAuthSettled(headerHost) &&
      (!previousSnapshot || previousSnapshot.status === AUTH_STATE_VALUES.syncing)
    ) {
      ensureAppAuthSettling(headerHost);
      return commitSnapshot(headerHost, createSnapshot(AUTH_STATE_VALUES.syncing, 'dom-pending'));
    }
    if (observedSnapshot.status === AUTH_STATE_VALUES.authenticated) {
      markAppAuthSettled(headerHost);
    }
    if (
      observedSnapshot.status === AUTH_STATE_VALUES.syncing &&
      previousSnapshot &&
      previousSnapshot.status !== AUTH_STATE_VALUES.syncing
    ) {
      return commitSnapshot(headerHost, createSnapshot(previousSnapshot.status, 'dom-stable'));
    }
    return commitSnapshot(headerHost, observedSnapshot);
  }

  function syncFromAuthenticatedState(headerHost, source) {
    clearPublicAuthRecoveryPolling();
    markAppAuthSettled(headerHost);
    return commitSnapshot(headerHost, createSnapshot(AUTH_STATE_VALUES.authenticated, source));
  }

  function syncFromUnauthenticatedState(headerHost, source) {
    markAppAuthSettled(headerHost);
    return commitSnapshot(headerHost, createSnapshot(AUTH_STATE_VALUES.unauthenticated, source));
  }

  function refreshPublicAuthSession(headerHost, source) {
    if (!shouldRefreshPublicAuthSession(headerHost) || typeof window.getCurrentUser !== 'function') {
      return;
    }
    if (store.runtimeRefreshPending === true) {
      return;
    }
    store.runtimeRefreshPending = true;
    var profilePromise;
    try {
      profilePromise = window.getCurrentUser();
    } catch (error) {
      store.runtimeRefreshPending = false;
      return;
    }
    Promise.resolve(profilePromise)
      .then(function (profile) {
        store.runtimeRefreshPending = false;
        if (profile && typeof profile === 'object') {
          syncFromAuthenticatedState(headerHost, source || 'runtime');
          return;
        }
        syncFromObservedState(headerHost);
      })
      .catch(function () {
        store.runtimeRefreshPending = false;
      });
  }

  function handlePublicAuthRecoveryEvent(event) {
    if (event && event.type === 'visibilitychange' && document && document.visibilityState === 'hidden') {
      return;
    }
    var headerHost = document.querySelector('mpr-header');
    if (!headerHost) {
      return;
    }
    refreshPublicAuthSession(headerHost, event && event.type ? 'runtime-' + event.type : 'runtime');
  }

  function schedulePublicAuthRecoveryPolling(headerHost, source) {
    if (!shouldRefreshPublicAuthSession(headerHost)) {
      resetPublicAuthRecovery();
      hidePublicAuthScreen();
      return;
    }
    if (store.publicAuthRecoveryRemainingAttempts <= 0) {
      resetPublicAuthRecovery();
      hidePublicAuthScreen();
      return;
    }
    if (store.publicAuthRecoveryTimerId) {
      return;
    }
    store.publicAuthRecoveryTimerId = window.setTimeout(function () {
      store.publicAuthRecoveryTimerId = 0;
      store.publicAuthRecoveryRemainingAttempts -= 1;
      refreshPublicAuthSession(headerHost, source || 'runtime-poll');
      if (store.snapshot && store.snapshot.status === AUTH_STATE_VALUES.authenticated) {
        clearPublicAuthRecoveryPolling();
        return;
      }
      schedulePublicAuthRecoveryPolling(headerHost, source || 'runtime-poll');
    }, PUBLIC_AUTH_RECOVERY_POLL_INTERVAL_MS);
  }

  function startPublicAuthRecoveryPolling(headerHost, source) {
    if (!shouldRefreshPublicAuthSession(headerHost)) {
      return;
    }
    store.loginRedirectPending = true;
    store.publicAuthRecoveryRemainingAttempts = PUBLIC_AUTH_RECOVERY_MAX_ATTEMPTS;
    showPublicAuthScreen(headerHost);
    refreshPublicAuthSession(headerHost, source || 'runtime-start');
    schedulePublicAuthRecoveryPolling(headerHost, source || 'runtime-poll');
  }

  function attachPublicAuthRecoveryListeners(headerHost) {
    if (publicAuthRecoveryListenersAttached || !shouldRefreshPublicAuthSession(headerHost)) {
      return;
    }
    if (window && typeof window.addEventListener === 'function') {
      window.addEventListener('focus', handlePublicAuthRecoveryEvent);
      window.addEventListener('pageshow', handlePublicAuthRecoveryEvent);
    }
    if (document && typeof document.addEventListener === 'function') {
      document.addEventListener('visibilitychange', handlePublicAuthRecoveryEvent);
    }
    publicAuthRecoveryListenersAttached = true;
  }

  function handlePublicSigninIntent(event) {
    var target = event && event.target && event.target.nodeType === 1 ? event.target : null;
    if (!target || typeof target.closest !== 'function') {
      return;
    }
    if (!target.closest('[data-mpr-header="google-signin"]')) {
      return;
    }
    var headerHost = resolveAuthHost(event);
    if (!headerHost) {
      return;
    }
    clearExplicitLogoutState();
    startPublicAuthRecoveryPolling(headerHost, 'runtime-signin');
  }

  function attachPublicSigninListeners(headerHost) {
    if (publicSigninListenersAttached || !headerHost || !shouldRefreshPublicAuthSession(headerHost) || typeof headerHost.addEventListener !== 'function') {
      return;
    }
    headerHost.addEventListener('click', handlePublicSigninIntent, true);
    publicSigninListenersAttached = true;
  }

  function observeHeaderState(headerHost) {
    if (!headerHost || headerHost.__loopawareAuthObserver || typeof MutationObserver !== 'function') {
      return;
    }
    var observer = new MutationObserver(function () {
      syncFromObservedState(headerHost);
    });
    observer.observe(headerHost, {
      subtree: true,
      childList: true,
      attributes: true,
      attributeFilter: ['class', 'data-mpr-user-status']
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
    startLogoutTransition();
    disableGoogleAutoSelect();
  }

  function handleAuthenticatedEvent(event) {
    var headerHost = resolveAuthHost(event);
    if (!headerHost) {
      return;
    }
    syncFromAuthenticatedState(headerHost, 'event');
  }

  function handleUnauthenticatedEvent(event) {
    var headerHost = resolveAuthHost(event);
    if (!headerHost) {
      hideOverlay();
      hidePublicAuthScreen();
      return;
    }
    resetPublicAuthRecovery();
    syncFromUnauthenticatedState(headerHost, 'event');
  }

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
    syncExplicitLogoutState(headerHost);
    syncPublicAuthScreenMode(headerHost);
    ensureLogoutFallback(headerHost);
    attachPublicAuthRecoveryListeners(headerHost);
    attachPublicSigninListeners(headerHost);
    if (typeof headerHost.addEventListener === 'function' && headerHost.getAttribute('data-loopaware-auth-listeners') !== 'true') {
      headerHost.setAttribute('data-loopaware-auth-listeners', 'true');
      headerHost.addEventListener('mpr-ui:auth:authenticated', handleAuthenticatedEvent);
      headerHost.addEventListener('mpr-ui:auth:unauthenticated', handleUnauthenticatedEvent);
    }
    if (headerHost.getAttribute('data-loopaware-auth-bound') !== 'true') {
      headerHost.setAttribute('data-loopaware-auth-bound', 'true');
    }
    ensureAppAuthSettling(headerHost);
    observeHeaderState(headerHost);
    syncFromObservedState(headerHost);
    refreshPublicAuthSession(headerHost, 'runtime-bind');
  }

  function bindHeaderAuth() {
    if (bindingInProgress) {
      return;
    }
    bindingInProgress = true;
    var remainingAttempts = 120;

    function attemptBind() {
      var headerHost = document.querySelector('mpr-header');
      if (headerHost) {
        attachHeaderAuth(headerHost);
        bindingInProgress = false;
        return;
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
