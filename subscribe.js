// @ts-check
(function(){
  var formContainerId = "mp-subscribe-form";
  var emailInputId = "mp-subscribe-email";
  var nameInputId = "mp-subscribe-name";
  var submitButtonId = "mp-subscribe-submit";
  var statusElementId = "mp-subscribe-status";
  var bubbleId = "mp-subscribe-bubble";
  var panelId = "mp-subscribe-panel";

  var modeBubble = "bubble";
  var modeInline = "inline";

  /**
   * @typedef {Object} SubscribeConfig
   * @property {string} siteId
   * @property {string} apiOrigin
   * @property {string} mode
   * @property {string} accent
   * @property {string} cta
   * @property {string} success
   * @property {string} error
   * @property {string} alreadySubscribed
   * @property {string} invalidEmail
   * @property {boolean} hideName
   * @property {string} targetId
   * @property {string} onSuccess
   * @property {string} onError
   */

  var subscribeDefaults = {
    mode: modeInline,
    accent: "#0d6efd",
    cta: "Subscribe",
    success: "You're on the list!",
    error: "Please try again.",
    alreadySubscribed: "You're already subscribed!",
    invalidEmail: "Please enter a valid email.",
    emailPlaceholder: "you@example.com",
    namePlaceholder: "Your name (optional)"
  };
  // Public embeds should only expose site_id; LoopAware-owned script hosts map to the API internally.
  var loopAwarePublicAPIOriginsByScriptOrigin = {
    "https://loopaware.mprlab.com": "https://loopaware-api.mprlab.com",
    "https://tyemirov.github.io": "https://loopaware-api.mprlab.com"
  };

  function selectScriptTag() {
    var current = document.currentScript;
    if (current) {
      return current;
    }
    var candidates = document.querySelectorAll('script[src*="subscribe.js"]');
    if (candidates.length > 0) {
      return candidates[candidates.length - 1];
    }
    return null;
  }

  function getQueryParam(search, name) {
    if (!search || !name) {
      return null;
    }
    var query = search.indexOf("?") === 0 ? search.substring(1) : search;
    if (!query) {
      return null;
    }
    var pairs = query.split("&");
    for (var i = 0; i < pairs.length; i++) {
      var rawPair = pairs[i];
      var separatorIndex = rawPair.indexOf("=");
      var rawKey = separatorIndex === -1 ? rawPair : rawPair.slice(0, separatorIndex);
      var rawValue = separatorIndex === -1 ? "" : rawPair.slice(separatorIndex + 1);
      var decodedKey = null;
      try {
        decodedKey = decodeURIComponent(rawKey.replace(/\+/g, " "));
      } catch(decodeError) {
        continue;
      }
      if (decodedKey === name) {
        try {
          return decodeURIComponent(rawValue.replace(/\+/g, " "));
        } catch(decodeError) {
          return rawValue.replace(/\+/g, " ");
        }
      }
    }
    return null;
  }

  function normalizeAPIOrigin(rawValue) {
    if (typeof rawValue !== "string") {
      return null;
    }
    var trimmed = rawValue.trim();
    if (!trimmed) {
      return null;
    }
    if (trimmed.indexOf("http://") !== 0 && trimmed.indexOf("https://") !== 0) {
      return null;
    }
    try {
      var parsed = new URL(trimmed);
      var origin = parsed && typeof parsed.origin === "string" ? parsed.origin : "";
      if (!origin || origin === "null") {
        return null;
      }
      var pathname = parsed.pathname || "";
      if (pathname && pathname !== "/") {
        return null;
      }
      if (parsed.search || parsed.hash) {
        return null;
      }
      if (parsed.username || parsed.password) {
        return null;
      }
      return origin.replace(/\/+$/, "");
    } catch(parseError) {
      return null;
    }
  }

  function resolveAPIOriginCandidate(scriptTag) {
    if (!scriptTag) {
      throw new Error("subscribe.js: resolve_origin.failed: missing script tag");
    }
    var candidate = scriptTag.getAttribute("data-api-origin");
    if (candidate) {
      return candidate;
    }
    if (scriptTag.src) {
      var link = document.createElement("a");
      link.href = scriptTag.src;
      var queryOrigin = getQueryParam(link.search || "", "api_origin");
      if (queryOrigin) {
        return queryOrigin;
      }
    }
    return null;
  }

  function resolveInternalAPIOrigin(scriptTag) {
    if (!scriptTag || !scriptTag.src) {
      return null;
    }
    try {
      var scriptLink = document.createElement("a");
      scriptLink.href = scriptTag.src;
      if (!scriptLink.protocol || !scriptLink.host) {
        return null;
      }
      var scriptOrigin = scriptLink.protocol + "//" + scriptLink.host;
      return loopAwarePublicAPIOriginsByScriptOrigin[scriptOrigin] || null;
    } catch(parseError) {
      return null;
    }
  }

  function resolveAPIOrigin(scriptTag) {
    var candidate = resolveAPIOriginCandidate(scriptTag);
    if (candidate) {
      var normalizedCandidate = normalizeAPIOrigin(candidate);
      if (!normalizedCandidate) {
        throw new Error("subscribe.js: resolve_origin.failed: invalid api_origin format");
      }
      return normalizedCandidate;
    }
    var internalOrigin = resolveInternalAPIOrigin(scriptTag);
    if (internalOrigin) {
      return internalOrigin;
    }
    // Fallback to script origin is allowed as a core behavioral contract
    if (scriptTag.src) {
      try {
        var scriptLink = document.createElement("a");
        scriptLink.href = scriptTag.src;
        if (scriptLink.protocol && scriptLink.host) {
          return scriptLink.protocol + "//" + scriptLink.host;
        }
      } catch(originError){}
    }
    // If script host is not available, fallback to current host
    if (window.location && window.location.protocol && window.location.host) {
      return window.location.protocol + "//" + window.location.host;
    }
    throw new Error("subscribe.js: resolve_origin.failed: api_origin not provided and cannot be resolved");
  }

  /**
   * @param {HTMLOrSVGScriptElement | null} scriptTag
   * @returns {SubscribeConfig}
   */
  function parseConfig(scriptTag) {
    if (!scriptTag) {
      throw new Error("subscribe.js: parse_config.failed: missing script tag");
    }
    var search = "";
    try {
      if (scriptTag instanceof HTMLScriptElement && scriptTag.src) {
        var link = document.createElement("a");
        link.href = scriptTag.src;
        search = link.search || "";
      }
    } catch(parseError){
      // Non-fatal, search will be empty
    }

    var apiOrigin = resolveAPIOrigin(scriptTag);

    var siteId = getQueryParam(search, "site_id") || scriptTag.getAttribute("data-site-id");
    if (!siteId) {
      throw new Error("subscribe.js: parse_config.failed: site_id not provided in data-site-id or query string");
    }

    var mode = (getQueryParam(search, "mode") || scriptTag.getAttribute("data-mode") || subscribeDefaults.mode).toLowerCase();
    if (mode !== modeBubble) {
      mode = modeInline;
    }
    var accent = getQueryParam(search, "accent") || scriptTag.getAttribute("data-accent") || subscribeDefaults.accent;
    var cta = getQueryParam(search, "cta") || scriptTag.getAttribute("data-cta") || subscribeDefaults.cta;
    var success = getQueryParam(search, "success") || scriptTag.getAttribute("data-success") || subscribeDefaults.success;
    var error = getQueryParam(search, "error") || scriptTag.getAttribute("data-error") || subscribeDefaults.error;
    var hideName = getQueryParam(search, "name_field") === "false" || scriptTag.getAttribute("data-name-field") === "false";
    var targetId = getQueryParam(search, "target") || scriptTag.getAttribute("data-target") || "";
    
    var alreadySubscribed = getQueryParam(search, "already_subscribed") || scriptTag.getAttribute("data-already-subscribed") || subscribeDefaults.alreadySubscribed;
    var invalidEmail = getQueryParam(search, "invalid_email") || scriptTag.getAttribute("data-invalid-email") || subscribeDefaults.invalidEmail;
    var onSuccess = getQueryParam(search, "onSuccess") || scriptTag.getAttribute("data-on-success") || "";
    var onError = getQueryParam(search, "onError") || scriptTag.getAttribute("data-on-error") || "";

    return {
      siteId: String(siteId).trim(),
      apiOrigin: apiOrigin,
      mode: mode,
      accent: accent,
      cta: cta,
      success: success,
      error: error,
      alreadySubscribed: alreadySubscribed,
      invalidEmail: invalidEmail,
      hideName: hideName,
      targetId: String(targetId).trim(),
      onSuccess: onSuccess,
      onError: onError
    };
  }

  function buildEndpoint(config) {
    return config.apiOrigin + "/public/subscriptions";
  }

  function createInlineContainer() {
    var container = document.createElement("div");
    container.id = formContainerId;
    container.style.maxWidth = "420px";
    container.style.padding = "12px";
    container.style.border = "1px solid rgba(0,0,0,0.08)";
    container.style.borderRadius = "10px";
    container.style.boxShadow = "0 8px 24px rgba(0,0,0,0.12)";
    container.style.fontFamily = "system-ui, -apple-system, Segoe UI, Roboto, Ubuntu, Cantarell, Noto Sans, sans-serif";
    container.style.background = "#fff";
    container.style.color = "#1f2937";
    container.style.boxSizing = "border-box";
    return container;
  }

  /**
   * @param {SubscribeConfig} config
   */
  function createBubbleContainer(config) {
    var bubble = document.createElement("div");
    bubble.id = bubbleId;
    bubble.style.position = "fixed";
    bubble.style.right = "16px";
    bubble.style.bottom = "16px";
    bubble.style.width = "56px";
    bubble.style.height = "56px";
    bubble.style.borderRadius = "50%";
    bubble.style.background = config.accent;
    bubble.style.color = "#fff";
    bubble.style.display = "flex";
    bubble.style.alignItems = "center";
    bubble.style.justifyContent = "center";
    bubble.style.cursor = "pointer";
    bubble.style.fontSize = "24px";
    bubble.style.boxShadow = "0 10px 30px rgba(0,0,0,0.25)";
    bubble.style.zIndex = "2147483647";
    bubble.setAttribute("aria-label", "Open subscribe form");
    bubble.innerText = "✉️";
    return bubble;
  }

  function createPanelContainer() {
    var panel = document.createElement("div");
    panel.id = panelId;
    panel.style.position = "fixed";
    panel.style.right = "16px";
    panel.style.bottom = "84px";
    panel.style.width = "360px";
    panel.style.maxWidth = "92vw";
    panel.style.padding = "12px";
    panel.style.borderRadius = "12px";
    panel.style.border = "1px solid rgba(0,0,0,0.08)";
    panel.style.boxShadow = "0 16px 36px rgba(0,0,0,0.2)";
    panel.style.background = "#fff";
    panel.style.color = "#1f2937";
    panel.style.fontFamily = "system-ui, -apple-system, Segoe UI, Roboto, Ubuntu, Cantarell, Noto Sans, sans-serif";
    panel.style.display = "none";
    panel.style.zIndex = "2147483647";
    panel.style.boxSizing = "border-box";
    return panel;
  }

  /**
   * @param {SubscribeConfig} config
   */
  function createFormElements(config) {
    var email = document.createElement("input");
    email.id = emailInputId;
    email.type = "email";
    email.placeholder = subscribeDefaults.emailPlaceholder;
    email.required = true;
    email.style.width = "100%";
    email.style.padding = "10px 12px";
    email.style.border = "1px solid #d1d5db";
    email.style.borderRadius = "8px";
    email.style.fontSize = "14px";
    email.style.boxSizing = "border-box";
    email.autocomplete = "email";

    var name = null;
    if (!config.hideName) {
      name = document.createElement("input");
      name.id = nameInputId;
      name.type = "text";
      name.placeholder = subscribeDefaults.namePlaceholder;
      name.style.width = "100%";
      name.style.padding = "10px 12px";
      name.style.border = "1px solid #d1d5db";
      name.style.borderRadius = "8px";
      name.style.fontSize = "14px";
      name.style.boxSizing = "border-box";
      name.autocomplete = "name";
    }

    var submit = document.createElement("button");
    submit.id = submitButtonId;
    submit.type = "button";
    submit.innerText = config.cta;
    submit.style.width = "100%";
    submit.style.padding = "10px 12px";
    submit.style.border = "0";
    submit.style.borderRadius = "8px";
    submit.style.fontWeight = "600";
    submit.style.cursor = "pointer";
    submit.style.background = config.accent;
    submit.style.color = "#fff";
    submit.style.boxSizing = "border-box";

    var status = document.createElement("div");
    status.id = statusElementId;
    status.style.minHeight = "16px";
    status.style.marginTop = "8px";
    status.style.fontSize = "13px";
    status.style.color = "#374151";

    return { email: email, name: name, submit: submit, status: status };
  }

  /**
   * @param {HTMLElement} container
   * @param {any} formElements
   * @param {HTMLElement | null} targetElement
   */
  function renderInline(container, formElements, targetElement) {
    var heading = document.createElement("div");
    heading.style.fontWeight = "600";
    heading.style.marginBottom = "8px";
    heading.innerText = "Get updates";
    container.appendChild(heading);

    container.appendChild(formElements.email);
    if (formElements.name) {
      container.appendChild(formElements.name);
    }
    var spacer = document.createElement("div");
    spacer.style.height = "8px";
    container.appendChild(spacer);
    container.appendChild(formElements.submit);
    container.appendChild(formElements.status);
    var host = targetElement || document.body;
    if (host) {
      host.appendChild(container);
    }
  }

  /**
   * @param {HTMLElement} bubble
   * @param {HTMLElement} panel
   * @param {any} formElements
   */
  function renderBubble(bubble, panel, formElements) {
    panel.appendChild(formElements.email);
    if (formElements.name) {
      panel.appendChild(formElements.name);
    }
    var spacer = document.createElement("div");
    spacer.style.height = "8px";
    panel.appendChild(spacer);
    panel.appendChild(formElements.submit);
    panel.appendChild(formElements.status);
    document.body.appendChild(bubble);
    document.body.appendChild(panel);
  }

  function validateEmail(value) {
    var trimmed = (value || "").trim();
    return trimmed.length > 3 && trimmed.indexOf("@") > 0;
  }

  function showStatus(statusElement, message, color) {
    statusElement.innerText = message;
    statusElement.style.color = color;
  }

  function fireCallback(callbackName, detail) {
    if (callbackName && typeof window[callbackName] === "function") {
      try {
        window[callbackName](detail);
      } catch (callbackError) {
        console.error("subscribe.js callback error:", callbackError);
      }
    }
  }

  function dispatchSubscribeEvent(targetId, eventName, detail) {
    var target = targetId ? document.getElementById(targetId) : document.body;
    if (target) {
      var event = new CustomEvent(eventName, { detail: detail, bubbles: true });
      target.dispatchEvent(event);
    }
  }

  /**
   * @param {SubscribeConfig} config
   * @param {string} endpoint
   * @param {any} formElements
   * @param {((forceHide?: boolean) => void) | null} togglePanel
   */
  function attachBehavior(config, endpoint, formElements, togglePanel) {
    var sending = false;
    formElements.submit.addEventListener("click", function(){
      if (sending) { return; }
      var emailValue = (formElements.email.value || "").trim();
      var nameValue = "";
      if (formElements.name) {
        nameValue = (formElements.name.value || "").trim();
      }
      if (!validateEmail(emailValue)) {
        showStatus(formElements.status, config.invalidEmail, "#dc2626");
        formElements.email.focus();
        return;
      }
      sending = true;
      formElements.submit.disabled = true;
      showStatus(formElements.status, "Sending...", "#2563eb");

      var payload = {
        site_id: config.siteId,
        email: emailValue,
        source_url: window.location ? window.location.href : ""
      };
      if (formElements.name) {
        payload.name = nameValue;
      }

      var fetchOptions = {
        method: "POST",
        headers: {"Content-Type": "application/json"},
        body: JSON.stringify(payload),
        keepalive: true
      };

      fetch(endpoint, fetchOptions).then(function(resp){
        if (resp.ok) {
          return resp.json().then(function(data) {
            return { ok: true, status: resp.status, data: data };
          });
        }
        return { ok: false, status: resp.status };
      }).then(function(result){
        formElements.submit.disabled = false;
        sending = false;

        if (result.ok) {
          formElements.email.value = "";
          if (formElements.name) {
            formElements.name.value = "";
          }
          showStatus(formElements.status, config.success, "#15803d");
          if (typeof togglePanel === "function") {
            togglePanel(true);
          }
          var successDetail = { email: emailValue, status: result.status };
          fireCallback(config.onSuccess, successDetail);
          dispatchSubscribeEvent(config.targetId, "loopaware:subscribe:success", successDetail);
        } else if (result.status === 409) {
          showStatus(formElements.status, config.alreadySubscribed, "#2563eb");
          var alreadyDetail = { email: emailValue, status: 409, reason: "already_subscribed" };
          fireCallback(config.onSuccess, alreadyDetail);
          dispatchSubscribeEvent(config.targetId, "loopaware:subscribe:success", alreadyDetail);
        } else if (result.status === 400) {
          showStatus(formElements.status, config.invalidEmail, "#dc2626");
          var invalidDetail = { email: emailValue, status: 400, reason: "invalid_email" };
          fireCallback(config.onError, invalidDetail);
          dispatchSubscribeEvent(config.targetId, "loopaware:subscribe:error", invalidDetail);
        } else {
          showStatus(formElements.status, config.error, "#dc2626");
          var errorDetail = { email: emailValue, status: result.status, reason: "unknown" };
          fireCallback(config.onError, errorDetail);
          dispatchSubscribeEvent(config.targetId, "loopaware:subscribe:error", errorDetail);
        }
      }).catch(function(err){
        console.error(err);
        showStatus(formElements.status, config.error, "#dc2626");
        formElements.submit.disabled = false;
        sending = false;
        var errorDetail = { email: emailValue, status: 0, reason: "network", error: err.message };
        fireCallback(config.onError, errorDetail);
        dispatchSubscribeEvent(config.targetId, "loopaware:subscribe:error", errorDetail);
      });
    });
  }

  function main() {
    var scriptTag = selectScriptTag();
    var config = parseConfig(scriptTag);
    
    var targetElement = null;
    if (config.targetId) {
      targetElement = document.getElementById(config.targetId);
    }
    var endpoint = buildEndpoint(config);
    var formElements = createFormElements(config);
    /** @type {((forceHide?: boolean) => void) | null} */
    var togglePanel = null;

    if (config.mode === modeBubble) {
      var bubble = createBubbleContainer(config);
      var panel = createPanelContainer();
      renderBubble(bubble, panel, formElements);
      togglePanel = function(forceHide){
        var hidden = panel.style.display === "none";
        if (forceHide === true) {
          panel.style.display = "none";
          return;
        }
        panel.style.display = hidden ? "block" : "none";
        if (!hidden) {
          return;
        }
        formElements.email.focus();
      };
      bubble.addEventListener("click", function(){
        if (togglePanel) togglePanel(false);
      });
    } else {
      renderInline(createInlineContainer(), formElements, targetElement);
    }

    attachBehavior(config, endpoint, formElements, togglePanel);
  }

  try {
    if (!document.body) {
      window.addEventListener("DOMContentLoaded", main);
    } else {
      main();
    }
  } catch(renderError) {
    console.error(renderError);
    // No silent fallback - if main fails, it fails.
  }
})();
