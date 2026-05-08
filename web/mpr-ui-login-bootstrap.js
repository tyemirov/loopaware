// @ts-check
(function () {
  var TRIGGER_SELECTOR = '[data-loopaware-login-config="true"][data-config-url]';
  var BUNDLE_MARKER_SELECTOR = 'script[data-mpr-ui-bundle-src]';
  var bundleLoadPromise = null;

  function whenDocumentReady(callback) {
    if (!document || typeof document.addEventListener !== 'function') {
      callback();
      return;
    }
    if (document.readyState && document.readyState !== 'loading') {
      callback();
      return;
    }
    document.addEventListener('DOMContentLoaded', callback, { once: true });
  }

  function loadScript(scriptUrl) {
    if (bundleLoadPromise) {
      return bundleLoadPromise;
    }
    bundleLoadPromise = new Promise(function (resolve, reject) {
      var scriptElement = document.createElement('script');
      scriptElement.async = true;
      scriptElement.defer = true;
      scriptElement.src = scriptUrl;
      scriptElement.onload = function () {
        resolve();
      };
      scriptElement.onerror = function () {
        reject(new Error('mpr_ui_login_bundle_load_failed'));
      };
      document.head.appendChild(scriptElement);
    });
    return bundleLoadPromise;
  }

  function readBundleSource() {
    var marker = document.querySelector(BUNDLE_MARKER_SELECTOR);
    if (!marker || typeof marker.getAttribute !== 'function') {
      throw new Error('mpr_ui_login_bundle_marker_missing');
    }
    var source = String(marker.getAttribute('data-mpr-ui-bundle-src') || '').trim();
    if (!source) {
      throw new Error('mpr_ui_login_bundle_source_missing');
    }
    return source;
  }

  function bootstrapLoginControls() {
    var trigger = document.querySelector(TRIGGER_SELECTOR);
    if (!trigger || typeof trigger.getAttribute !== 'function') {
      return;
    }
    var configUrl = String(trigger.getAttribute('data-config-url') || '').trim();
    if (!configUrl) {
      return;
    }
    if (!window.MPRUI || typeof window.MPRUI.applyYamlConfig !== 'function') {
      throw new Error('mpr_ui_config_loader_missing');
    }
    window.MPRUI
      .applyYamlConfig({
        configUrl: configUrl,
        headerSelector: 'mpr-header[data-loopaware-login-auth-header="true"]'
      })
      .then(function () {
        return loadScript(readBundleSource());
      })
      .catch(function (error) {
        console.error('[loopaware] Login auth bootstrap failed:', error);
      });
  }

  whenDocumentReady(bootstrapLoginControls);
})();
