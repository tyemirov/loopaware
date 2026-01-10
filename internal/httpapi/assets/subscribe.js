// @ts-check
(function(){
  var formContainerId = "mp-subscribe-form";
  var emailInputId = "mp-subscribe-email";
  var nameInputId = "mp-subscribe-name";
  var submitButtonId = "mp-subscribe-submit";
  var statusElementId = "mp-subscribe-status";
  var bubbleId = "mp-subscribe-bubble";
  var panelId = "mp-subscribe-panel";
  var defaultAccentColor = "#0d6efd";
  var defaultCTA = "Subscribe";
  var defaultSuccessText = "You're on the list!";
  var defaultErrorText = "Please try again.";
  var defaultEmailPlaceholder = "you@example.com";
  var defaultNamePlaceholder = "Your name (optional)";
  var modeBubble = "bubble";
  var modeInline = "inline";

  function selectScriptTag() {
    var current = document.currentScript;
    if (current) {
      return current;
    }
    var candidates = document.querySelectorAll('script[src*="subscribe.js"]');
    return candidates[candidates.length - 1];
  }

  function parseConfig(scriptTag) {
    var search = "";
    try {
      var link = document.createElement("a");
      link.href = scriptTag.src || "";
      search = link.search || "";
    } catch(parseError){}
    var params = new URLSearchParams(search);
    var mode = (params.get("mode") || modeInline).toLowerCase();
    if (mode !== modeBubble) {
      mode = modeInline;
    }
    var accent = params.get("accent") || defaultAccentColor;
    var cta = params.get("cta") || defaultCTA;
    var success = params.get("success") || defaultSuccessText;
    var error = params.get("error") || defaultErrorText;
    var hideName = params.get("name_field") === "false";
    var targetId = params.get("target") || scriptTag.getAttribute("data-target") || "";
    if (targetId) {
      targetId = String(targetId).trim();
    }
    var siteId = params.get("site_id") || scriptTag.getAttribute("data-site-id") || "";
    if (!siteId) {
      siteId = "{{ .SiteID }}";
    }
    return {
      siteId: siteId,
      accent: accent,
      mode: mode,
      cta: cta,
      success: success,
      error: error,
      hideName: hideName,
      targetId: targetId
    };
  }

  function buildEndpoint(scriptTag) {
    var endpoint = (location.protocol + "//" + location.host + "/api/subscriptions");
    try {
      if (scriptTag && scriptTag.src) {
        var link = document.createElement("a");
        link.href = scriptTag.src;
        endpoint = link.protocol + "//" + link.host + "/api/subscriptions";
      }
    } catch(endpointError){}
    return endpoint;
  }

  function createInlineContainer(config) {
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

  function createBubbleContainer(config) {
    var bubble = document.createElement("div");
    bubble.id = bubbleId;
    bubble.style.position = "fixed";
    bubble.style.right = "16px";
    bubble.style.bottom = "16px";
    bubble.style.width = "56px";
    bubble.style.height = "56px";
    bubble.style.borderRadius = "50%";
    bubble.style.background = config.accent || defaultAccentColor;
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

  function createFormElements(config) {
    var email = document.createElement("input");
    email.id = emailInputId;
    email.type = "email";
    email.placeholder = defaultEmailPlaceholder;
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
      name.placeholder = defaultNamePlaceholder;
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
    submit.innerText = config.cta || defaultCTA;
    submit.style.width = "100%";
    submit.style.padding = "10px 12px";
    submit.style.border = "0";
    submit.style.borderRadius = "8px";
    submit.style.fontWeight = "600";
    submit.style.cursor = "pointer";
    submit.style.background = config.accent || defaultAccentColor;
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
        showStatus(formElements.status, "Enter a valid email.", "#dc2626");
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
        if (!resp.ok) { throw new Error("HTTP " + resp.status); }
        return resp.json();
      }).then(function(){
        formElements.email.value = "";
        if (formElements.name) {
          formElements.name.value = "";
        }
        showStatus(formElements.status, config.success || defaultSuccessText, "#15803d");
        formElements.submit.disabled = false;
        sending = false;
        if (typeof togglePanel === "function") {
          togglePanel(true);
        }
      }).catch(function(err){
        console.error(err);
        showStatus(formElements.status, config.error || defaultErrorText, "#dc2626");
        formElements.submit.disabled = false;
        sending = false;
      });
    });
  }

  function main() {
    var scriptTag = selectScriptTag();
    var config = parseConfig(scriptTag);
    if (!config.siteId) {
      console.error("subscribe.js: missing site_id");
      return;
    }
    var targetElement = null;
    if (config.targetId) {
      targetElement = document.getElementById(config.targetId);
    }
    var endpoint = buildEndpoint(scriptTag);
    var formElements = createFormElements(config);
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
        togglePanel(false);
      });
    } else {
      renderInline(createInlineContainer(config), formElements, targetElement);
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
  }
})();
