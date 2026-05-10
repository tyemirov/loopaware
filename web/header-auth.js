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
  var AUTH_STATE_CHANGE_EVENT = 'loopaware:auth-state-change';
  var AUTH_HOME_LINK_SELECTOR = '[data-loopaware-auth-home-link="true"]';
  var MPR_AUTH_STATUS_AUTHENTICATING = 'authenticating';
  var MPR_HEADER_SIGNIN_CLICK_EVENT = 'mpr-ui:header:signin-click';
  var APP_PATHNAME = '/app';
  var LOGIN_PATHNAME = '/login';
  var INITIAL_APP_AUTH_SETTLE_TIMEOUT_MS = 3000;
  var EXPLICIT_LOGOUT_STORAGE_KEY = 'loopaware_explicit_logout';
  var LOGOUT_MAIN_DISPLAY_BACKUP_ATTR = 'data-loopaware-logout-main-display';
  var LOGOUT_MAIN_HIDDEN_ATTR = 'data-loopaware-logout-main-hidden';
  var store = window.__loopawareHeaderAuthStore;
  var authListenersAttached = false;
  var userMenuListenersAttached = false;
  var bindingInProgress = false;

  if (!store || typeof store !== 'object') {
    store = {
      loginRedirectPending: false,
      logoutPending: false,
      snapshot: null,
      redirectTarget: ''
    };
    window.__loopawareHeaderAuthStore = store;
  }

  function resolveExplicitLogoutStorage() {
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

  function syncExplicitLogoutState(headerHost) {
    if (!headerHost || typeof headerHost.toggleAttribute !== 'function') {
      return;
    }
    headerHost.toggleAttribute('data-loopaware-explicit-logout', hasExplicitLogoutState());
  }

  function resolveLogoutOverlay() {
    return document.getElementById('logout-overlay');
  }

  function setMainLogoutHiddenState(isHidden) {
    var mainElement = document.querySelector('main');
    if (!mainElement || !mainElement.style) {
      return;
    }
    if (isHidden) {
      if (mainElement.getAttribute(LOGOUT_MAIN_HIDDEN_ATTR) === 'true') {
        return;
      }
      var currentDisplay = mainElement.style.display;
      if (mainElement.getAttribute(LOGOUT_MAIN_DISPLAY_BACKUP_ATTR) === null) {
        if (typeof currentDisplay === 'string') {
          mainElement.setAttribute(LOGOUT_MAIN_DISPLAY_BACKUP_ATTR, currentDisplay);
        } else {
          mainElement.setAttribute(LOGOUT_MAIN_DISPLAY_BACKUP_ATTR, '');
        }
      }
      mainElement.style.display = 'none';
      mainElement.setAttribute(LOGOUT_MAIN_HIDDEN_ATTR, 'true');
      return;
    }
    if (mainElement.getAttribute(LOGOUT_MAIN_HIDDEN_ATTR) !== 'true') {
      return;
    }
    var restoredDisplay = mainElement.getAttribute(LOGOUT_MAIN_DISPLAY_BACKUP_ATTR);
    if (restoredDisplay === null || restoredDisplay === '') {
      mainElement.style.display = '';
    } else {
      mainElement.style.display = restoredDisplay;
    }
    mainElement.removeAttribute(LOGOUT_MAIN_HIDDEN_ATTR);
    mainElement.removeAttribute(LOGOUT_MAIN_DISPLAY_BACKUP_ATTR);
  }

  function setBodyLogoutState(isActive) {
    if (!document.body || !document.body.classList) {
      return;
    }
    document.body.classList.toggle('logging-out', isActive === true);
  }

  function showOverlay() {
    if (typeof window.showLogoutOverlay === 'function') {
      window.showLogoutOverlay();
    } else {
      var overlay = resolveLogoutOverlay();
      if (overlay) {
        overlay.classList.remove('d-none');
        overlay.classList.add('d-flex');
        overlay.style.display = 'flex';
      }
      setBodyLogoutState(true);
    }
    setMainLogoutHiddenState(true);
  }

  function hideOverlay() {
    if (typeof window.hideLogoutOverlay === 'function') {
      window.hideLogoutOverlay();
    } else {
      var overlay = resolveLogoutOverlay();
      if (overlay) {
        overlay.classList.add('d-none');
        overlay.classList.remove('d-flex');
        overlay.style.display = '';
      }
      setBodyLogoutState(false);
    }
    setMainLogoutHiddenState(false);
  }

  function createSnapshot(status, source) {
    return {
      status: status,
      source: source || ''
    };
  }

  function normalizeTextValue(value) {
    return typeof value === 'string' ? value.trim() : '';
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

  function shouldRedirectToLogin(headerHost) {
    return !!(
      headerHost &&
      typeof headerHost.getAttribute === 'function' &&
      headerHost.getAttribute('data-loopaware-auth-redirect-on-logout') === 'true'
    );
  }

  function ensureAppAuthSettling(headerHost) {
    if (!shouldRedirectToLogin(headerHost) || headerHost.__loopawareAuthSettleStarted === true) {
      return;
    }
    headerHost.__loopawareAuthSettleStarted = true;
    window.setTimeout(function () {
      headerHost.__loopawareAuthSettled = true;
      syncFromObservedState(headerHost, 'settle-timeout');
    }, INITIAL_APP_AUTH_SETTLE_TIMEOUT_MS);
  }

  function markAppAuthSettled(headerHost) {
    if (headerHost) {
      headerHost.__loopawareAuthSettled = true;
    }
  }

  function resolveObservedSnapshot(headerHost, source) {
    if (!headerHost) {
      return createSnapshot(AUTH_STATE_VALUES.syncing, source || 'missing-header');
    }
    var mprStatus = normalizeTextValue(headerHost.getAttribute('data-mpr-auth-status'));
    if (mprStatus === 'authenticated') {
      return createSnapshot(AUTH_STATE_VALUES.authenticated, source || 'mpr-status');
    }
    if (mprStatus === 'bootstrapping' || mprStatus === 'authenticating') {
      return createSnapshot(AUTH_STATE_VALUES.syncing, source || 'mpr-status');
    }
    if (mprStatus === 'unauthenticated') {
      return createSnapshot(AUTH_STATE_VALUES.unauthenticated, source || 'mpr-status');
    }
    var headerRoot = typeof headerHost.querySelector === 'function'
      ? headerHost.querySelector('header.mpr-header')
      : null;
    if (headerRoot && headerRoot.classList && headerRoot.classList.contains('mpr-header--authenticated')) {
      return createSnapshot(AUTH_STATE_VALUES.authenticated, source || 'dom');
    }
    var userMenu = typeof headerHost.querySelector === 'function'
      ? headerHost.querySelector('mpr-user[data-loopaware-user-menu="true"], mpr-user[data-mpr-header="user-menu"]')
      : null;
    var userMenuStatus = userMenu && typeof userMenu.getAttribute === 'function'
      ? normalizeTextValue(userMenu.getAttribute('data-mpr-user-status'))
      : '';
    if (userMenuStatus === 'authenticated') {
      return createSnapshot(AUTH_STATE_VALUES.authenticated, source || 'dom');
    }
    if (userMenuStatus === 'unauthenticated' || userMenuStatus === 'error') {
      return createSnapshot(AUTH_STATE_VALUES.unauthenticated, source || 'dom');
    }
    return createSnapshot(AUTH_STATE_VALUES.syncing, source || 'dom');
  }

  function normalizeSnapshotForProtectedBoot(headerHost, snapshot) {
    if (!shouldRedirectToLogin(headerHost) || snapshot.status !== AUTH_STATE_VALUES.unauthenticated) {
      return snapshot;
    }
    if (headerHost.__loopawareAuthSettled === true || snapshot.source === 'event' || snapshot.source === 'settle-timeout') {
      return snapshot;
    }
    ensureAppAuthSettling(headerHost);
    return createSnapshot(AUTH_STATE_VALUES.syncing, 'protected-boot');
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

  function setHeaderAuthStateAttribute(headerHost, stateValue) {
    if (!headerHost || typeof headerHost.setAttribute !== 'function') {
      return;
    }
    if (headerHost.getAttribute('data-loopaware-auth-state') !== stateValue) {
      headerHost.setAttribute('data-loopaware-auth-state', stateValue);
    }
  }

  function resolveAuthHomePath(snapshot) {
    return snapshot && snapshot.status === AUTH_STATE_VALUES.authenticated ? APP_PATHNAME : LOGIN_PATHNAME;
  }

  function syncAuthHomeLinks(headerHost, snapshot) {
    if (!headerHost || typeof headerHost.querySelectorAll !== 'function') {
      return;
    }
    var homePath = resolveAuthHomePath(snapshot);
    var homeLinks = headerHost.querySelectorAll(AUTH_HOME_LINK_SELECTOR);
    for (var linkIndex = 0; linkIndex < homeLinks.length; linkIndex += 1) {
      var homeLink = homeLinks[linkIndex];
      if (homeLink && typeof homeLink.setAttribute === 'function' && homeLink.getAttribute('href') !== homePath) {
        homeLink.setAttribute('href', homePath);
      }
    }
  }

  function shouldRedirectToApp(headerHost, snapshot) {
    if (!headerHost || !snapshot || snapshot.status !== AUTH_STATE_VALUES.authenticated) {
      return false;
    }
    if (hasExplicitLogoutState() && headerHost.getAttribute('data-loopaware-auth-redirect') === 'true') {
      return false;
    }
    if (headerHost.getAttribute('data-loopaware-auth-redirect') !== 'true') {
      return false;
    }
    if (!window.location || typeof window.location.pathname !== 'string') {
      return true;
    }
    if (window.location.pathname === LOGIN_PATHNAME || window.location.pathname === LOGIN_PATHNAME + '/') {
      return true;
    }
    return (
      store.loginRedirectPending === true &&
      window.location.pathname !== APP_PATHNAME &&
      window.location.pathname !== APP_PATHNAME + '/'
    );
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
      store.loginRedirectPending = false;
      clearExplicitLogoutState();
    }
    window.location.assign(pathname);
  }

  function dispatchAuthStateChange(headerHost, snapshot) {
    if (!headerHost || typeof headerHost.dispatchEvent !== 'function' || typeof CustomEvent !== 'function') {
      return;
    }
    headerHost.dispatchEvent(new CustomEvent(AUTH_STATE_CHANGE_EVENT, {
      detail: {
        status: snapshot.status,
        source: snapshot.source
      },
      bubbles: true
    }));
  }

  function applySnapshot(headerHost, snapshot) {
    if (!headerHost) {
      return;
    }
    setHeaderAuthStateAttribute(headerHost, snapshot.status);
    syncAuthHomeLinks(headerHost, snapshot);
    if (snapshot.status === AUTH_STATE_VALUES.authenticated) {
      markAppAuthSettled(headerHost);
      store.logoutPending = false;
      hideOverlay();
      if (shouldRedirectToApp(headerHost, snapshot)) {
        redirectTo(APP_PATHNAME);
      }
      return;
    }
    if (snapshot.status === AUTH_STATE_VALUES.unauthenticated) {
      store.logoutPending = false;
      if (shouldRedirectToLogin(headerHost)) {
        markExplicitLogoutState();
        showOverlay();
        redirectTo(LOGIN_PATHNAME);
        return;
      }
      hideOverlay();
      return;
    }
    if (store.logoutPending !== true) {
      hideOverlay();
    }
  }

  function commitSnapshot(headerHost, snapshot) {
    snapshot = normalizeSnapshotForProtectedBoot(headerHost, snapshot);
    snapshot = normalizeSnapshotForExplicitLogout(headerHost, snapshot);
    var previousSnapshot = store.snapshot;
    store.snapshot = snapshot;
    applySnapshot(headerHost, snapshot);
    if (!previousSnapshot || previousSnapshot.status !== snapshot.status) {
      dispatchAuthStateChange(headerHost, snapshot);
    }
    return snapshot;
  }

  function syncFromObservedState(headerHost, source) {
    return commitSnapshot(headerHost, resolveObservedSnapshot(headerHost, source || 'dom'));
  }

  function syncFromAuthenticatedState(headerHost, source) {
    markAppAuthSettled(headerHost);
    return commitSnapshot(headerHost, createSnapshot(AUTH_STATE_VALUES.authenticated, source || 'event'));
  }

  function syncFromUnauthenticatedState(headerHost, source) {
    markAppAuthSettled(headerHost);
    return commitSnapshot(headerHost, createSnapshot(AUTH_STATE_VALUES.unauthenticated, source || 'event'));
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
    markExplicitLogoutState();
    store.loginRedirectPending = false;
    store.logoutPending = true;
    showOverlay();
  }

  function markSigninIntent() {
    clearExplicitLogoutState();
    store.loginRedirectPending = true;
  }

  function handleHeaderSigninClick() {
    markSigninIntent();
  }

  function handleAuthStatusChange(event) {
    var headerHost = resolveAuthHost(event);
    if (event && event.detail && event.detail.status === MPR_AUTH_STATUS_AUTHENTICATING) {
      markSigninIntent();
    }
    if (headerHost) {
      syncFromObservedState(headerHost, 'status-change');
    }
  }

  function handleAuthenticatedEvent(event) {
    var headerHost = resolveAuthHost(event);
    if (headerHost) {
      syncFromAuthenticatedState(headerHost, 'event');
    }
  }

  function handleUnauthenticatedEvent(event) {
    var headerHost = resolveAuthHost(event);
    if (headerHost) {
      syncFromUnauthenticatedState(headerHost, 'event');
      return;
    }
    hideOverlay();
  }

  function observeHeaderState(headerHost) {
    if (!headerHost || headerHost.__loopawareAuthObserver || typeof MutationObserver !== 'function') {
      return;
    }
    var observer = new MutationObserver(function () {
      syncFromObservedState(headerHost, 'mutation');
    });
    observer.observe(headerHost, {
      subtree: true,
      childList: true,
      attributes: true,
      attributeFilter: ['class', 'data-mpr-auth-status', 'data-mpr-user-status']
    });
    headerHost.__loopawareAuthObserver = observer;
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

  function attachAuthListeners() {
    if (authListenersAttached || !document || typeof document.addEventListener !== 'function') {
      return;
    }
    document.addEventListener('mpr-ui:auth:authenticated', handleAuthenticatedEvent);
    document.addEventListener('mpr-ui:auth:unauthenticated', handleUnauthenticatedEvent);
    document.addEventListener('mpr-ui:auth:status-change', handleAuthStatusChange);
    document.addEventListener(MPR_HEADER_SIGNIN_CLICK_EVENT, handleHeaderSigninClick);
    document.addEventListener('mpr-ui:orchestration:ready', function () {
      bindHeaderAuth();
    });
    authListenersAttached = true;
  }

  function waitForMprUiAutoOrchestrationReady() {
    if (window.MPRUI && typeof window.MPRUI.whenAutoOrchestrationReady === 'function') {
      return window.MPRUI.whenAutoOrchestrationReady();
    }
    return Promise.resolve();
  }

  function attachHeaderAuth(headerHost) {
    attachUserMenuListeners();
    attachAuthListeners();
    if (!headerHost) {
      return;
    }
    if (typeof headerHost.addEventListener === 'function' && headerHost.getAttribute('data-loopaware-auth-listeners') !== 'true') {
      headerHost.setAttribute('data-loopaware-auth-listeners', 'true');
      headerHost.addEventListener('mpr-ui:auth:authenticated', handleAuthenticatedEvent);
      headerHost.addEventListener('mpr-ui:auth:unauthenticated', handleUnauthenticatedEvent);
      headerHost.addEventListener('mpr-ui:auth:status-change', handleAuthStatusChange);
      headerHost.addEventListener(MPR_HEADER_SIGNIN_CLICK_EVENT, handleHeaderSigninClick);
    }
    syncExplicitLogoutState(headerHost);
    if (headerHost.getAttribute('data-loopaware-auth-bound') !== 'true') {
      headerHost.setAttribute('data-loopaware-auth-bound', 'true');
    }
    ensureAppAuthSettling(headerHost);
    observeHeaderState(headerHost);
    waitForMprUiAutoOrchestrationReady()
      .catch(function () {})
      .then(function () {
        syncFromObservedState(headerHost, 'orchestration-ready');
      });
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
