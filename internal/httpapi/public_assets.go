package httpapi

import (
	"bytes"
	"fmt"
	"html/template"
	texttemplate "text/template"
)

const (
	publicBrandName              = "LoopAware"
	publicThemeStorageKey        = "loopaware_public_theme"
	publicLandingThemeStorageKey = "loopaware_landing_theme"
	publicLegacyThemeStorageKey  = "landing_theme"
	publicLandingPath            = LandingPagePath
	publicDashboardPath          = "/app"
	publicSignInLabel            = "Login"
	publicSignOutLabel           = "Logout"
	publicHeroScrollTarget       = "#top"
	publicHeroAttributeName      = "data-public-hero"
	publicHeroAttributeValue     = "true"
	publicHeroScrollAttribute    = "data-scroll-to-top"
	publicHeroScrollValue        = "true"
	publicSharedStylesCSS        = `.landing-body {
        transition: background-color 0.3s ease, color 0.3s ease;
      }
      .landing-header {
        position: sticky;
        top: 0;
        z-index: 1030;
        padding: 0;
        transition: background-color 0.3s ease;
      }
      .landing-brand {
        font-size: 1.25rem;
        font-weight: 600;
      }
      .landing-logo {
        display: inline-flex;
        align-items: center;
        justify-content: center;
      }
      .landing-logo-image {
        width: 48px;
        height: 48px;
      }
      .landing-card {
        transition: transform 0.2s ease, box-shadow 0.2s ease;
        cursor: default;
      }
      .landing-card:hover,
      .landing-card:focus-visible {
        transform: translateY(-4px);
        box-shadow: 0 1.25rem 1.5rem -1rem rgba(15, 23, 42, 0.35);
      }
      .landing-card:focus-visible {
        outline: 0;
      }
      .landing-header .mpr-header__actions {
        margin-left: auto;
      }
      .landing-header .mpr-header__nav {
        margin-left: 0;
      }
      .landing-header mpr-user[data-mpr-header="user-menu"]:not([data-loopaware-user-menu="true"]) {
        display: none !important;
      }
      body[data-bs-theme="dark"] .landing-header {
        --mpr-color-surface-primary: #0f172a;
        --mpr-color-text-primary: #f8fafc;
        --mpr-color-border: rgba(148, 163, 184, 0.25);
        --mpr-chip-bg: rgba(148, 163, 184, 0.18);
      }
      body[data-bs-theme="dark"] .landing-card {
        background-color: rgba(15, 23, 42, 0.8);
        color: #e2e8f0;
      }
      body[data-bs-theme="dark"] .landing-card p {
        color: #cbd5f5;
      }
      body[data-bs-theme="dark"] .landing-footer {
        background-color: #0f172a;
        color: #94a3b8;
      }
      body[data-bs-theme="light"] .landing-header {
        --mpr-color-surface-primary: #ffffff;
        --mpr-color-text-primary: #0f172a;
        --mpr-color-border: rgba(148, 163, 184, 0.2);
        --mpr-chip-bg: rgba(148, 163, 184, 0.18);
      }
      body[data-bs-theme="light"] .landing-card {
        background-color: #f8fafc;
        color: #0f172a;
      }
      body[data-bs-theme="light"] .landing-card p {
        color: #475569;
      }
      body[data-bs-theme="light"] .landing-footer {
        background-color: #ffffff;
        color: #475569;
      }`
	privacyPageStylesCSS = `body{font:16px/1.5 system-ui,Segoe UI,Roboto,Helvetica,Arial,sans-serif;margin:0}
      .privacy-container{max-width:800px;margin:40px auto}
      .privacy-heading{font-size:1.6rem;margin-bottom:.2rem}`
)

var (
	publicHeaderTemplate = template.Must(template.New("public_header").Parse(`<mpr-header class="landing-header" google-site-id="{{.GoogleClientID}}"{{if .TauthBaseURL}} tauth-url="{{.TauthBaseURL}}"{{end}} tauth-tenant-id="{{.TauthTenantID}}" tauth-login-path="{{.TauthLoginPath}}" tauth-logout-path="{{.TauthLogoutPath}}" tauth-nonce-path="{{.TauthNoncePath}}" sign-in-label="{{.SignInLabel}}" sign-out-label="{{.SignOutLabel}}"{{if .AuthRedirectAttr}} {{.AuthRedirectAttr}}{{end}}>
  <a slot="brand" class="landing-brand d-inline-flex align-items-center gap-3 text-decoration-none" href="{{.HeroTarget}}" {{.HeroDataAttribute}}{{if .HeroScrollAttribute}} {{.HeroScrollAttribute}}{{end}}>
    <span class="landing-logo">
      <img src="{{.LogoDataURI}}" alt="LoopAware logo" class="landing-logo-image" />
    </span>
    <span>{{.BrandName}}</span>
  </a>
  <mpr-user
    slot="aux"
    data-loopaware-user-menu="true"
    display-mode="avatar"
    logout-url="/login"
    menu-items='[{"label":"Account settings","href":"/app"}]'
  ></mpr-user>
</mpr-header>`))
	publicThemeScriptTemplate = template.Must(template.New("public_theme_script").Parse(`(function() {
  var publicThemeStorageKey = '{{.PublicThemeStorageKey}}';
  var landingThemeStorageKey = '{{.LandingThemeStorageKey}}';
  var legacyThemeStorageKey = '{{.LegacyThemeStorageKey}}';
  var rootElement = document.body;
  var documentRoot = document.documentElement;
  var footerElement = document.querySelector('mpr-footer');
  function applyPublicTheme(theme) {
    var normalizedTheme = theme === 'light' ? 'light' : 'dark';
    rootElement.setAttribute('data-bs-theme', normalizedTheme);
    rootElement.setAttribute('data-mpr-theme', normalizedTheme);
    if (documentRoot) {
      documentRoot.setAttribute('data-bs-theme', normalizedTheme);
      documentRoot.setAttribute('data-mpr-theme', normalizedTheme);
    }
    rootElement.classList.toggle('bg-body', true);
    rootElement.classList.toggle('text-body', true);
  }
  function parseThemeConfig(rawValue) {
    if (!rawValue) {
      return {};
    }
    try {
      var parsed = JSON.parse(rawValue);
      if (parsed && typeof parsed === 'object') {
        return parsed;
      }
    } catch (error) {
      console.error(error);
    }
    return {};
  }
  function updateFooterThemeConfig(theme) {
    if (!footerElement) {
      return;
    }
    var normalizedTheme = theme === 'light' ? 'light' : 'dark';
    var config = parseThemeConfig(footerElement.getAttribute('theme-config'));
    if (!config.attribute) {
      config.attribute = 'data-bs-theme';
    }
    config.initialMode = normalizedTheme;
    footerElement.setAttribute('theme-config', JSON.stringify(config));
  }
  function loadPublicTheme() {
    var storedTheme = localStorage.getItem(publicThemeStorageKey);
    if (storedTheme === null) {
      var landingStoredTheme = localStorage.getItem(landingThemeStorageKey);
      if (landingStoredTheme === null) {
        var legacyStoredTheme = localStorage.getItem(legacyThemeStorageKey);
        if (legacyStoredTheme === 'light' || legacyStoredTheme === 'dark') {
          landingStoredTheme = legacyStoredTheme;
          localStorage.setItem(landingThemeStorageKey, landingStoredTheme);
        }
      }
      if (landingStoredTheme === 'light' || landingStoredTheme === 'dark') {
        storedTheme = landingStoredTheme;
        localStorage.setItem(publicThemeStorageKey, storedTheme);
      }
    }
    return storedTheme;
  }
  function persistPublicTheme(theme) {
    localStorage.setItem(publicThemeStorageKey, theme);
    localStorage.setItem(landingThemeStorageKey, theme);
  }
  function initializePublicTheme() {
    var storedTheme = loadPublicTheme();
    var normalizedTheme = storedTheme === 'light' || storedTheme === 'dark' ? storedTheme : 'dark';
    applyPublicTheme(normalizedTheme);
    updateFooterThemeConfig(normalizedTheme);
    if (storedTheme !== 'light' && storedTheme !== 'dark') {
      persistPublicTheme(normalizedTheme);
    }
  }
  if (footerElement) {
    footerElement.addEventListener('mpr-footer:theme-change', function(event) {
      var nextTheme = event && event.detail && event.detail.theme === 'dark' ? 'dark' : 'light';
      applyPublicTheme(nextTheme);
      persistPublicTheme(nextTheme);
    });
  }
  initializePublicTheme();
  var heroElement = document.querySelector('[{{.HeroAttributeName}}]');
  if (heroElement) {
    var shouldScrollToTop = heroElement.getAttribute('{{.HeroScrollAttributeName}}') === '{{.HeroScrollAttributeValue}}';
    if (shouldScrollToTop) {
      heroElement.addEventListener('click', function(event) {
        event.preventDefault();
        window.scrollTo({ top: 0, behavior: 'smooth' });
      });
    }
  }
})();`))
	publicAuthScriptTemplate = texttemplate.Must(texttemplate.New("public_auth_script").Parse(`(function() {
  if (document && document.documentElement) {
    document.documentElement.setAttribute('data-loopaware-auth-script', 'true');
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

  function submitLogoutForm(logoutUrl) {
    return new Promise(function(resolve) {
      if (!document || !logoutUrl) {
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
      form.action = logoutUrl;
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
    var logoutUrl = resolveLogoutURL(headerHost);
    var logoutRequest = function() {
      return window.fetch(logoutUrl, {
        method: 'POST',
        credentials: 'include',
        headers: resolveLogoutHeaders(headerHost)
      });
    };
    var logoutWithFetchFallback = function() {
      return logoutRequest().catch(function() {
        return submitLogoutForm(logoutUrl);
      });
    };
    if (typeof logoutDelegate === 'function') {
      try {
        return Promise.resolve(logoutDelegate())
          .catch(function() {
            return logoutWithFetchFallback();
          });
      } catch (error) {
        return logoutWithFetchFallback();
      }
    }
    return logoutWithFetchFallback();
  }

  function ensureLogoutFallback(headerHost) {
    if (!window || !headerHost) {
      return;
    }
    if (typeof window.logout === 'function' && window.logout.__loopawareLogoutWrapper === true) {
      return;
    }
    if (typeof window.__loopawareLogoutDelegate !== 'function' && typeof window.logout === 'function') {
      window.__loopawareLogoutDelegate = window.logout;
    }
    var wrapper = function() {
      return performLogoutRequest(headerHost, window.__loopawareLogoutDelegate);
    };
    wrapper.__loopawareLogoutWrapper = true;
    window.logout = wrapper;
  }

  function disableGoogleAutoSelect() {
    if (!window || !window.google || !window.google.accounts || !window.google.accounts.id) {
      return;
    }
    var identityApi = window.google.accounts.id;
    if (typeof identityApi.cancel === 'function') {
      try {
        identityApi.cancel();
      } catch (error) {}
    }
    if (typeof identityApi.disableAutoSelect === 'function') {
      try {
        identityApi.disableAutoSelect();
      } catch (error) {}
    }
  }

  var googleSigninGateMaxAttempts = 40;
  var googleSigninGatePollIntervalMs = 100;
  var googleSigninGateSlowPollIntervalMs = 1000;

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
      } catch (error) {
        throw error;
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

  function ensureRequestNonceNonceTracking() {
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
    ensureRequestNonceNonceTracking();
    var remainingAttempts = googleSigninGateMaxAttempts;
    function scheduleNextGateAttempt() {
      var interval = remainingAttempts > 0 ? googleSigninGatePollIntervalMs : googleSigninGateSlowPollIntervalMs;
      window.setTimeout(attemptGate, interval);
    }
    function attemptGate() {
      var target = resolveGoogleSigninTarget(headerHost);
      if (!target) {
        if (remainingAttempts > 0) {
          remainingAttempts -= 1;
        }
        scheduleNextGateAttempt();
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
      scheduleNextGateAttempt();
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

  function handleAuthenticatedEvent(event) {
    var headerHost = resolveAuthHost(event);
    if (!headerHost) {
      return;
    }
    pruneHeaderUserMenus(headerHost);
    clearGoogleSigninGate(resolveGoogleSigninTarget(headerHost));
    if (headerHost.getAttribute('data-loopaware-auth-redirect') === 'true') {
      window.location.assign('{{.DashboardPath}}');
    }
  }

  function handleUnauthenticatedEvent(event) {
    var headerHost = resolveAuthHost(event);
    disableGoogleAutoSelect();
    if (!headerHost) {
      return;
    }
    gateGoogleSigninUntilNonce(headerHost);
    if (headerHost.getAttribute('data-loopaware-auth-redirect-on-logout') === 'true') {
      window.location.assign('{{.LandingPath}}');
    }
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
    if (!event || !event.detail) {
      return;
    }
    if (event.detail.action !== 'account-settings') {
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
    disableGoogleAutoSelect();
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
    gateGoogleSigninUntilNonce(headerHost);
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
})();`))
)

type publicHeaderTemplateData struct {
	LogoDataURI         template.URL
	BrandName           string
	HeroTarget          string
	HeroDataAttribute   template.HTMLAttr
	HeroScrollAttribute template.HTMLAttr
	GoogleClientID      string
	TauthBaseURL        string
	TauthTenantID       string
	TauthLoginPath      string
	TauthLogoutPath     string
	TauthNoncePath      string
	SignInLabel         string
	SignOutLabel        string
	AuthRedirectAttr    template.HTMLAttr
}

type publicThemeScriptTemplateData struct {
	PublicThemeStorageKey    string
	LandingThemeStorageKey   string
	LegacyThemeStorageKey    string
	HeroAttributeName        string
	HeroScrollAttributeName  string
	HeroScrollAttributeValue string
}

type publicAuthScriptTemplateData struct {
	DashboardPath string
	LandingPath   string
}

type publicPageType string

const (
	publicPageLanding publicPageType = "landing"
	publicPagePrivacy publicPageType = "privacy"
)

type publicHeroBehavior struct {
	Target       string
	ShouldScroll bool
}

func renderPublicHeader(logoDataURI template.URL, isAuthenticated bool, pageType publicPageType, authConfig AuthClientConfig, enableAuthRedirect bool) (template.HTML, error) {
	heroBehavior := resolvePublicHeroBehavior(isAuthenticated, pageType)
	data := publicHeaderTemplateData{
		LogoDataURI:       logoDataURI,
		BrandName:         publicBrandName,
		HeroTarget:        heroBehavior.Target,
		HeroDataAttribute: template.HTMLAttr(fmt.Sprintf(`%s="%s"`, publicHeroAttributeName, publicHeroAttributeValue)),
		GoogleClientID:    authConfig.GoogleClientID,
		TauthBaseURL:      authConfig.TauthBaseURL,
		TauthTenantID:     authConfig.TauthTenantID,
		TauthLoginPath:    TauthLoginPath,
		TauthLogoutPath:   TauthLogoutPath,
		TauthNoncePath:    TauthNoncePath,
		SignInLabel:       publicSignInLabel,
		SignOutLabel:      publicSignOutLabel,
	}
	if heroBehavior.ShouldScroll {
		data.HeroScrollAttribute = template.HTMLAttr(fmt.Sprintf(`%s="%s"`, publicHeroScrollAttribute, publicHeroScrollValue))
	}
	if enableAuthRedirect {
		data.AuthRedirectAttr = template.HTMLAttr(`data-loopaware-auth-redirect="true"`)
	}
	var buffer bytes.Buffer
	if err := publicHeaderTemplate.Execute(&buffer, data); err != nil {
		return "", err
	}
	return template.HTML(buffer.String()), nil
}

func renderPublicThemeScript() (template.JS, error) {
	data := publicThemeScriptTemplateData{
		PublicThemeStorageKey:    publicThemeStorageKey,
		LandingThemeStorageKey:   publicLandingThemeStorageKey,
		LegacyThemeStorageKey:    publicLegacyThemeStorageKey,
		HeroAttributeName:        publicHeroAttributeName,
		HeroScrollAttributeName:  publicHeroScrollAttribute,
		HeroScrollAttributeValue: publicHeroScrollValue,
	}
	var buffer bytes.Buffer
	if err := publicThemeScriptTemplate.Execute(&buffer, data); err != nil {
		return "", err
	}
	return template.JS(buffer.String()), nil
}

func renderPublicAuthScript() (template.JS, error) {
	data := publicAuthScriptTemplateData{
		DashboardPath: publicDashboardPath,
		LandingPath:   publicLandingPath,
	}
	var buffer bytes.Buffer
	if err := publicAuthScriptTemplate.Execute(&buffer, data); err != nil {
		return "", err
	}
	return template.JS(buffer.String()), nil
}

func resolvePublicHeroBehavior(isAuthenticated bool, pageType publicPageType) publicHeroBehavior {
	if isAuthenticated {
		return publicHeroBehavior{Target: publicDashboardPath}
	}
	if pageType == publicPageLanding {
		return publicHeroBehavior{Target: publicHeroScrollTarget, ShouldScroll: true}
	}
	return publicHeroBehavior{Target: publicLandingPath}
}

func sharedPublicStyles() template.CSS {
	return template.CSS(publicSharedStylesCSS)
}

func privacyPageStyles() template.CSS {
	return template.CSS(privacyPageStylesCSS)
}
