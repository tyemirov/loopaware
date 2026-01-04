package httpapi

import (
	"bytes"
	"fmt"
	"html/template"
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
      .landing-header .mpr-header__chip {
        flex-direction: row;
        align-items: center;
        gap: 0.5rem;
      }
      .landing-header .loopaware-header-profile-body {
        display: flex;
        flex-direction: column;
        gap: 0.25rem;
      }
      .landing-header .loopaware-header-avatar {
        width: 32px;
        height: 32px;
        border-radius: 999px;
        object-fit: cover;
        box-shadow: 0 0 0 2px rgba(148, 163, 184, 0.25);
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
	publicAuthScriptTemplate = template.Must(template.New("public_auth_script").Parse(`(function() {
  if (document && document.documentElement) {
    document.documentElement.setAttribute('data-loopaware-auth-script', 'true');
  }
  function ensureProfileBody(profileContainer, nameNode, signOutNode) {
    var body = profileContainer.querySelector('[data-loopaware-profile-body]');
    if (!body) {
      body = document.createElement('div');
      body.className = 'loopaware-header-profile-body';
      body.setAttribute('data-loopaware-profile-body', 'true');
      profileContainer.insertBefore(body, nameNode);
      body.appendChild(nameNode);
      body.appendChild(signOutNode);
    }
    return body;
  }

  function resolveCustomProfileElements(headerHost) {
    if (!headerHost) {
      return null;
    }
    var profileMenu = headerHost.querySelector('[data-loopaware-profile-menu="true"]');
    if (!profileMenu) {
      return null;
    }
    var toggleButton = profileMenu.querySelector('[data-loopaware-profile-toggle="true"]');
    var menuItems = profileMenu.querySelector('[data-loopaware-profile-menu-items="true"]');
    var profileName = profileMenu.querySelector('[data-loopaware-profile-name="true"]');
    if (!toggleButton || !menuItems || !profileName) {
      return null;
    }
    return {
      profileMenu: profileMenu,
      toggleButton: toggleButton,
      menuItems: menuItems,
      profileName: profileName,
      avatar: profileMenu.querySelector('[data-loopaware-avatar]'),
      settingsButton: profileMenu.querySelector('[data-loopaware-settings="true"]'),
      logoutButton: profileMenu.querySelector('[data-loopaware-logout="true"]')
    };
  }

  function resolveProfileAttribute(headerRoot, headerHost, attributeName) {
    if (headerRoot && typeof headerRoot.getAttribute === 'function') {
      var value = headerRoot.getAttribute(attributeName);
      if (value) {
        return value;
      }
    }
    if (headerHost && typeof headerHost.getAttribute === 'function') {
      return headerHost.getAttribute(attributeName) || '';
    }
    return '';
  }

  function resolveProfileDisplay(profile, headerRoot, headerHost) {
    if (profile && profile.display) {
      return profile.display;
    }
    if (profile && profile.name) {
      return profile.name;
    }
    if (profile && profile.email) {
      return profile.email;
    }
    return resolveProfileAttribute(headerRoot, headerHost, 'data-user-display') ||
      resolveProfileAttribute(headerRoot, headerHost, 'data-user-display-name') ||
      resolveProfileAttribute(headerRoot, headerHost, 'data-user-name') ||
      resolveProfileAttribute(headerRoot, headerHost, 'data-user-email') ||
      '';
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

  function performLogoutRequest(headerHost) {
    var logoutUrl = resolveLogoutURL(headerHost);
    var logoutRequest = function() {
      return window.fetch(logoutUrl, {
        method: 'POST',
        credentials: 'include',
        headers: resolveLogoutHeaders(headerHost)
      });
    };
    if (typeof window.logout === 'function') {
      try {
        return Promise.resolve(window.logout())
          .catch(function() {
            return null;
          })
          .then(function() {
            return logoutRequest();
          });
      } catch (error) {
        return logoutRequest();
      }
    }
    return logoutRequest();
  }

  function handleLogout(headerHost) {
    var redirectToLanding = function() {
      if (headerHost && headerHost.getAttribute('data-loopaware-auth-redirect-on-logout') === 'true') {
        window.location.assign('{{.LandingPath}}');
      }
    };
    performLogoutRequest(headerHost)
      .then(redirectToLanding)
      .catch(redirectToLanding);
  }

  function ensureCustomProfileMenu(headerHost, profileElements) {
    if (!profileElements || !profileElements.profileMenu || !profileElements.toggleButton || !profileElements.menuItems) {
      return;
    }
    var profileMenu = profileElements.profileMenu;
    var toggleButton = profileElements.toggleButton;
    var menuItems = profileElements.menuItems;
    var dropdownInstance = null;
    var setMenuOpen = function() {};
    if (window.bootstrap && window.bootstrap.Dropdown && typeof window.bootstrap.Dropdown.getOrCreateInstance === 'function') {
      dropdownInstance = window.bootstrap.Dropdown.getOrCreateInstance(toggleButton);
    }
    if (!profileMenu.getAttribute('data-loopaware-dropdown-bound')) {
      profileMenu.setAttribute('data-loopaware-dropdown-bound', 'true');
      setMenuOpen = function(shouldOpen) {
        if (dropdownInstance && typeof dropdownInstance.show === 'function' && typeof dropdownInstance.hide === 'function') {
          if (shouldOpen) {
            dropdownInstance.show();
          } else {
            dropdownInstance.hide();
          }
          return;
        }
        if (shouldOpen) {
          menuItems.classList.add('show');
          toggleButton.setAttribute('aria-expanded', 'true');
        } else {
          menuItems.classList.remove('show');
          toggleButton.setAttribute('aria-expanded', 'false');
        }
      };
      var clickInsideMenu = function(event) {
        if (!event) {
          return false;
        }
        if (profileMenu.contains(event.target)) {
          return true;
        }
        if (typeof event.composedPath === 'function') {
          var path = event.composedPath();
          var pathIndex = 0;
          while (pathIndex !== path.length) {
            if (path[pathIndex] === profileMenu) {
              return true;
            }
            pathIndex += 1;
          }
        }
        return false;
      };
      toggleButton.addEventListener('click', function(event) {
        event.preventDefault();
        event.stopPropagation();
        var isOpen = menuItems.classList.contains('show');
        setMenuOpen(!isOpen);
      });
      document.addEventListener('click', function(event) {
        if (!clickInsideMenu(event)) {
          setMenuOpen(false);
        }
      });
    }
    if (profileElements.settingsButton && !profileElements.settingsButton.getAttribute('data-loopaware-settings-bound')) {
      profileElements.settingsButton.setAttribute('data-loopaware-settings-bound', 'true');
      profileElements.settingsButton.addEventListener('click', function(event) {
        if (event) {
          event.preventDefault();
          event.stopPropagation();
        }
        setMenuOpen(false);
        var targetSelector = profileElements.settingsButton.getAttribute('data-bs-target');
        if (!targetSelector) {
          return;
        }
        var modalElement = document.querySelector(targetSelector);
        if (!modalElement) {
          return;
        }
        if (window.bootstrap && window.bootstrap.Modal && typeof window.bootstrap.Modal.getOrCreateInstance === 'function') {
          var modalInstance = window.bootstrap.Modal.getOrCreateInstance(modalElement);
          if (modalInstance && typeof modalInstance.show === 'function') {
            modalInstance.show();
          }
        }
      });
    }
    if (profileElements.logoutButton && !profileElements.logoutButton.getAttribute('data-loopaware-logout-bound')) {
      profileElements.logoutButton.setAttribute('data-loopaware-logout-bound', 'true');
      profileElements.logoutButton.addEventListener('click', function(event) {
        event.preventDefault();
        handleLogout(headerHost);
      });
    }
  }

  function resolveAvatarURL(headerRoot, headerHost, profile) {
    if (profile && profile.avatar_url) {
      return profile.avatar_url;
    }
    return resolveProfileAttribute(headerRoot, headerHost, 'data-user-avatar-url');
  }

  function updateCustomProfile(profileElements, profile, headerRoot, headerHost) {
    if (!profileElements) {
      return;
    }
    var displayName = resolveProfileDisplay(profile, headerRoot, headerHost);
    if (profileElements.profileName) {
      profileElements.profileName.textContent = displayName;
    }
    var avatarUrl = resolveAvatarURL(headerRoot, headerHost, profile);
    var avatar = profileElements.avatar;
    if (!avatar && avatarUrl && profileElements.toggleButton) {
      avatar = document.createElement('img');
      avatar.className = 'loopaware-header-avatar';
      avatar.setAttribute('data-loopaware-avatar', 'true');
      avatar.alt = 'User avatar';
      profileElements.toggleButton.insertBefore(avatar, profileElements.toggleButton.firstChild);
      profileElements.avatar = avatar;
    }
    if (avatar) {
      if (!avatarUrl) {
        avatar.removeAttribute('src');
        avatar.classList.add('d-none');
      } else {
        avatar.classList.remove('d-none');
        avatar.src = avatarUrl;
        if (profile && profile.display) {
          avatar.alt = profile.display;
        }
      }
    }
  }

  function resolveHeaderRoot(headerHost) {
    if (!headerHost) {
      return null;
    }
    var root = headerHost.querySelector('header.mpr-header');
    if (root) {
      return root;
    }
    if (headerHost.shadowRoot && typeof headerHost.shadowRoot.querySelector === 'function') {
      return headerHost.shadowRoot.querySelector('header.mpr-header');
    }
    return null;
  }

  function updateHeaderAvatar(headerHost, profile) {
    if (!headerHost) {
      return;
    }
    var headerRoot = resolveHeaderRoot(headerHost);
    var customProfile = resolveCustomProfileElements(headerHost);
    if (customProfile) {
      ensureCustomProfileMenu(headerHost, customProfile);
      updateCustomProfile(customProfile, profile, headerRoot, headerHost);
      return;
    }
    var profileContainer = headerHost.querySelector('[data-mpr-header="profile"]');
    var profileName = headerHost.querySelector('[data-mpr-header="profile-name"]');
    var signOutButton = headerHost.querySelector('[data-mpr-header="sign-out-button"]');
    if (!profileContainer || !profileName || !signOutButton) {
      return;
    }
    var avatarUrl = resolveAvatarURL(headerRoot, headerHost, profile);
    var body = ensureProfileBody(profileContainer, profileName, signOutButton);
    var avatar = profileContainer.querySelector('[data-loopaware-avatar]');
    if (!avatarUrl) {
      if (avatar) {
        avatar.remove();
      }
      return;
    }
    if (!avatar) {
      avatar = document.createElement('img');
      avatar.className = 'loopaware-header-avatar';
      avatar.setAttribute('data-loopaware-avatar', 'true');
      avatar.alt = 'User avatar';
      profileContainer.insertBefore(avatar, body);
    }
    if (profile && profile.display) {
      avatar.alt = profile.display;
    }
    avatar.src = avatarUrl;
  }

  function ensureHeaderProfileReady(headerHost) {
    if (!headerHost) {
      return;
    }
    var customProfile = resolveCustomProfileElements(headerHost);
    if (customProfile) {
      updateHeaderAvatar(headerHost, null);
      return;
    }
    var remainingAttempts = 20;
    function attemptSetup() {
      var profileContainer = headerHost.querySelector('[data-mpr-header="profile"]');
      var profileName = headerHost.querySelector('[data-mpr-header="profile-name"]');
      var signOutButton = headerHost.querySelector('[data-mpr-header="sign-out-button"]');
      if (profileContainer && profileName && signOutButton) {
        updateHeaderAvatar(headerHost, null);
        return;
      }
      remainingAttempts -= 1;
      if (remainingAttempts > 0) {
        window.setTimeout(attemptSetup, 50);
      }
    }
    attemptSetup();
  }

  function hasAuthenticatedProfile(headerHost) {
    if (!headerHost) {
      return false;
    }
    var headerRoot = resolveHeaderRoot(headerHost);
    if (headerRoot && headerRoot.classList.contains('mpr-header--authenticated')) {
      return true;
    }
    return Boolean(headerHost.getAttribute('data-user-id') || headerHost.getAttribute('data-user-email') || headerHost.getAttribute('data-user-display'));
  }

  var authBootstrapAttempts = 0;
  var authBootstrapInFlight = false;
  var authBootstrapLimit = 3;
  function attemptAuthBootstrap(headerHost) {
    if (!headerHost || authBootstrapInFlight || authBootstrapAttempts >= authBootstrapLimit) {
      return;
    }
    if (typeof window.getCurrentUser !== 'function') {
      return;
    }
    authBootstrapInFlight = true;
    authBootstrapAttempts += 1;
    Promise.resolve()
      .then(function() {
        return window.getCurrentUser();
      })
      .then(function(profile) {
        authBootstrapInFlight = false;
        if (!profile) {
          return;
        }
        updateHeaderAvatar(headerHost, profile);
        if (headerHost.getAttribute('data-loopaware-auth-redirect') === 'true') {
          window.location.assign('{{.DashboardPath}}');
        }
      })
      .catch(function() {
        authBootstrapInFlight = false;
      });
  }

  function observeHeaderAuthState(headerHost) {
    if (!headerHost || typeof MutationObserver !== 'function') {
      return;
    }
    if (headerHost.getAttribute('data-loopaware-auth-observer') === 'true') {
      return;
    }
    headerHost.setAttribute('data-loopaware-auth-observer', 'true');
    var wasAuthenticated = hasAuthenticatedProfile(headerHost);
    if (wasAuthenticated && headerHost.getAttribute('data-loopaware-auth-redirect') === 'true') {
      window.location.assign('{{.DashboardPath}}');
      return;
    }
    var observer = new MutationObserver(function() {
      var isAuthenticated = hasAuthenticatedProfile(headerHost);
      if (isAuthenticated) {
        updateHeaderAvatar(headerHost, null);
        if (headerHost.getAttribute('data-loopaware-auth-redirect') === 'true') {
          window.location.assign('{{.DashboardPath}}');
          return;
        }
      } else if (wasAuthenticated) {
        updateHeaderAvatar(headerHost, null);
        if (headerHost.getAttribute('data-loopaware-auth-redirect-on-logout') === 'true') {
          window.location.assign('{{.LandingPath}}');
          return;
        }
      }
      if (isAuthenticated !== wasAuthenticated) {
        wasAuthenticated = isAuthenticated;
      }
    });
    observer.observe(headerHost, {
      attributes: true,
      attributeFilter: ['data-user-id', 'data-user-email', 'data-user-display', 'data-user-avatar-url']
    });
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
    var profile = event && event.detail && event.detail.profile ? event.detail.profile : null;
    updateHeaderAvatar(headerHost, profile);
    if (headerHost.getAttribute('data-loopaware-auth-redirect') === 'true') {
      window.location.assign('{{.DashboardPath}}');
    }
  }

  function handleUnauthenticatedEvent(event) {
    var headerHost = resolveAuthHost(event);
    if (!headerHost) {
      return;
    }
    updateHeaderAvatar(headerHost, null);
    if (headerHost.getAttribute('data-loopaware-auth-redirect-on-logout') === 'true') {
      window.location.assign('{{.LandingPath}}');
    }
  }

  var authListenersAttached = false;
  function attachHeaderAuth(headerHost) {
    if (!authListenersAttached && document && typeof document.addEventListener === 'function') {
      document.addEventListener('mpr-ui:auth:authenticated', handleAuthenticatedEvent);
      document.addEventListener('mpr-ui:auth:unauthenticated', handleUnauthenticatedEvent);
      authListenersAttached = true;
    }
    if (!headerHost) {
      return;
    }
    if (typeof headerHost.addEventListener === 'function' && headerHost.getAttribute('data-loopaware-auth-listeners') !== 'true') {
      headerHost.setAttribute('data-loopaware-auth-listeners', 'true');
      headerHost.addEventListener('mpr-ui:auth:authenticated', handleAuthenticatedEvent);
      headerHost.addEventListener('mpr-ui:auth:unauthenticated', handleUnauthenticatedEvent);
    }
    headerHost.setAttribute('data-loopaware-auth-bound', 'true');
    ensureHeaderProfileReady(headerHost);
    observeHeaderAuthState(headerHost);
    (function redirectIfAlreadyAuthenticated() {
      var remainingAttempts = 120;
      function attemptRedirect() {
        var currentHeader = document.querySelector('mpr-header');
        var shouldRedirect = currentHeader && currentHeader.getAttribute('data-loopaware-auth-redirect') === 'true';
        if (shouldRedirect && hasAuthenticatedProfile(currentHeader)) {
          window.location.assign('{{.DashboardPath}}');
          return;
        }
        if (shouldRedirect) {
          attemptAuthBootstrap(currentHeader);
        }
        remainingAttempts -= 1;
        if (remainingAttempts > 0) {
          window.setTimeout(attemptRedirect, 100);
        }
      }
      attemptRedirect();
    }());
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
