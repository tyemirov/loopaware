// @ts-check
(function(){
  var widgetInitialized = false;
  var themeNameLight = "light";
  var themeNameDark = "dark";
  var themeAttributeName = "data-theme";
  var bootstrapThemeAttributeName = "data-bs-theme";
  var themeClassNameLight = "light";
  var themeClassNameDark = "dark";
  var themeMediaQueryDark = "(prefers-color-scheme: dark)";
  var luminanceDarkThreshold = 0.4;
  var luminanceLightThreshold = 0.6;
  var statusStateSuccess = "success";
  var statusStateError = "error";
  var statusStatePending = "pending";
  var boxSizingBorderBoxValue = "border-box";
  var panelDisplayBlockValue = "block";
  var panelDisplayNoneValue = "none";
  var panelAutoHideDelayMilliseconds = 2000;
  var panelVerticalSpacingPixels = 64;
  var widgetBrandingElementID = "mp-feedback-branding";
  var widgetBrandingLinkURL = "https://mprlab.com";
  var widgetBrandingLinkText = "Marco Polo Research Lab";
  var widgetBrandingFontSizeValue = "10px";
  var widgetBrandingMarginTopValue = "2px";
  var widgetBrandingTextAlignValue = "center";
  var widgetBrandingOpacityValue = "0.7";
  var widgetBrandingLinkTargetValue = "_blank";
  var widgetBrandingLinkRelValue = "noopener noreferrer";
  var widgetBrandingTextDecorationValue = "none";
  var widgetBrandingLineHeightValue = "1.2";
  var widgetBrandingLinkColorValue = "#b8860b";
  var widgetBrandingStaticText = "Built by ";
  var widgetHeaderContainerDisplayValue = "flex";
  var widgetHeaderContainerAlignItemsValue = "center";
  var widgetHeaderContainerJustifyContentValue = "space-between";
  var widgetHeaderContainerGapValue = "12px";
  var widgetHeaderContainerMarginBottomValue = "8px";
  var widgetHeadlineElementID = "mp-feedback-headline";
  var widgetHeadlineFontWeightValue = "600";
  var widgetHeadlineFlexGrowValue = "1";
  var widgetCloseButtonText = "×";
  var widgetCloseButtonFontSizeValue = "24px";
  var widgetCloseButtonLineHeightValue = "1";
  var widgetCloseButtonCursorValue = "pointer";
  var widgetCloseButtonPaddingValue = "0";
  var widgetCloseButtonBorderValue = "none";
  var widgetCloseButtonBackgroundValue = "transparent";
  var widgetCloseButtonWidthValue = "28px";
  var widgetCloseButtonHeightValue = "28px";
  var widgetCloseButtonMarginLeftValue = "12px";
  var widgetCloseButtonFlexShrinkValue = "0";
  var widgetCloseButtonOpacityValue = "0.6";
  var widgetCloseButtonHoverOpacityValue = "1";
  var widgetCloseButtonAriaLabel = "Close feedback panel";
  var widgetSentimentPromptText = "How was your experience?";
  var widgetSentimentSadValue = "sad";
  var widgetSentimentNeutralValue = "neutral";
  var widgetSentimentHappyValue = "happy";
  var widgetSentimentButtonSizeValue = "64px";
  var widgetSentimentButtonFontSizeValue = "48px";
  var widgetSentimentButtonRowGapValue = "10px";
  var widgetSentimentButtonTransitionValue = "transform 120ms ease, opacity 120ms ease, filter 120ms ease";
  var widgetSentimentOptions = [
    { value: widgetSentimentSadValue, label: "Sad", emoji: "🙁" },
    { value: widgetSentimentNeutralValue, label: "Neutral", emoji: "😐" },
    { value: widgetSentimentHappyValue, label: "Happy", emoji: "🙂" }
  ];
  var feedbackPhoneMinimumDigits = 10;
  var feedbackPhoneMaximumDigits = 15;
  var feedbackPhoneAllowedCharactersPattern = /^[0-9+().\-\s]+$/;
  var widgetDemoModeFlagName = "LOOPAWARE_WIDGET_DEMO_MODE";
  var widgetTestModeFlagName = "LOOPAWARE_WIDGET_TEST_MODE";
  var widgetTestEndpointFlagName = "LOOPAWARE_WIDGET_TEST_ENDPOINT";
  var widgetDemoModeEnabled = false;
  var widgetTestModeEnabled = false;
  var widgetTestEndpointOverride = "";
  var widgetSiteId = "";
  var widgetApiOrigin = "";
  // Public embeds should only expose site_id; LoopAware-owned script hosts map to the API internally.
  var loopAwarePublicAPIOriginsByScriptOrigin = {
    "https://loopaware.mprlab.com": "https://loopaware-api.mprlab.com",
    "https://tyemirov.github.io": "https://loopaware-api.mprlab.com"
  };

  var widgetDefaults = {
    placementSide: "right",
    placementBottomOffset: 16,
    horizontalOffset: "16px"
  };

  var widgetPlacementSideValue = widgetDefaults.placementSide;
  var widgetPlacementBottomOffsetValue = widgetDefaults.placementBottomOffset;
  var widgetPlacementHorizontalOffsetValue = widgetDefaults.horizontalOffset;
  var widgetShowMessageInputValue = true;
  var widgetShowSentimentButtonsValue = true;

  try {
    if (typeof window === "object" && window) {
      widgetDemoModeEnabled = Boolean(window[widgetDemoModeFlagName]);
      widgetTestModeEnabled = Boolean(window[widgetTestModeFlagName]);
      var testEndpointCandidate = window[widgetTestEndpointFlagName];
      if (typeof testEndpointCandidate === "string" && testEndpointCandidate.trim().length > 0) {
        widgetTestEndpointOverride = testEndpointCandidate;
      }
    }
  } catch(testModeReadError){
    console.error("widget.js: test_mode_read_failed", testModeReadError);
  }

  var widgetThemePalettes = {
    light: {
      bubbleBackground: "#0d6efd",
      bubbleTextColor: "#ffffff",
      bubbleShadow: "0 4px 16px rgba(0,0,0,0.2)",
      panelBackground: "#ffffff",
      panelBorder: "1px solid rgba(0,0,0,0.1)",
      panelShadow: "0 8px 24px rgba(0,0,0,0.2)",
      panelTextColor: "#212529",
      inputBackground: "#ffffff",
      inputTextColor: "#212529",
      inputBorder: "1px solid #ced4da",
      buttonBackground: "#0d6efd",
      buttonTextColor: "#ffffff",
      statusPositiveColor: "#157347",
      statusNegativeColor: "#dc3545",
      statusPendingColor: "#0d6efd",
      closeButtonColor: "#6c757d"
    },
    dark: {
      bubbleBackground: "#4dabf7",
      bubbleTextColor: "#0b1526",
      bubbleShadow: "0 8px 24px rgba(0,0,0,0.6)",
      panelBackground: "#1f2937",
      panelBorder: "1px solid rgba(148,163,184,0.35)",
      panelShadow: "0 16px 32px rgba(2,6,23,0.85)",
      panelTextColor: "#f1f5f9",
      inputBackground: "#111827",
      inputTextColor: "#f8fafc",
      inputBorder: "1px solid rgba(148,163,184,0.5)",
      buttonBackground: "#2563eb",
      buttonTextColor: "#f8fafc",
      statusPositiveColor: "#34d399",
      statusNegativeColor: "#f87171",
      statusPendingColor: "#60a5fa",
      closeButtonColor: "#94a3b8"
    }
  };

  function resolveWidgetScriptTag() {
    var current = document.currentScript;
    if (current) {
      return current;
    }
    var candidates = document.querySelectorAll('script[src*="widget.js"]');
    if (candidates.length > 0) {
      return candidates[candidates.length - 1];
    }
    return null;
  }

  function normalizeWidgetAPIOrigin(rawValue) {
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

  function resolveWidgetAPIOriginCandidate(scriptTag) {
    if (!scriptTag) {
      throw new Error("widget.js: resolve_origin.failed: missing script tag");
    }
    var candidate = scriptTag.getAttribute("data-api-origin");
    if (candidate) {
      return candidate;
    }
    if (scriptTag instanceof HTMLScriptElement && scriptTag.src) {
      var link = document.createElement("a");
      link.href = scriptTag.src;
      var queryOrigin = getQueryParam(link.search || "", "api_origin");
      if (queryOrigin) {
        return queryOrigin;
      }
    }
    return null;
  }

  function resolveInternalWidgetAPIOrigin(scriptTag) {
    if (!scriptTag || !scriptTag.src) {
      return null;
    }
    try {
      var link = document.createElement("a");
      link.href = scriptTag.src;
      if (!link.protocol || !link.host) {
        return null;
      }
      var scriptOrigin = link.protocol + "//" + link.host;
      return loopAwarePublicAPIOriginsByScriptOrigin[scriptOrigin] || null;
    } catch(parseError) {
      return null;
    }
  }

  function resolveWidgetAPIOrigin(scriptTag) {
    var candidate = resolveWidgetAPIOriginCandidate(scriptTag);
    if (candidate) {
      var normalizedCandidate = normalizeWidgetAPIOrigin(candidate);
      if (!normalizedCandidate) {
        throw new Error("widget.js: resolve_origin.failed: invalid api_origin format");
      }
      return normalizedCandidate;
    }
    var internalOrigin = resolveInternalWidgetAPIOrigin(scriptTag);
    if (internalOrigin) {
      return internalOrigin;
    }
    // Fallback to script origin is allowed as a core behavioral contract for the widget
    if (scriptTag && scriptTag.src) {
      try {
        var link = document.createElement("a");
        link.href = scriptTag.src;
        if (link.protocol && link.host) {
          return link.protocol + "//" + link.host;
        }
      } catch(originError){}
    }
    // If script host is not available, fallback to current host
    if (window.location && window.location.protocol && window.location.host) {
      return window.location.protocol + "//" + window.location.host;
    }
    throw new Error("widget.js: resolve_origin.failed: api_origin not provided and cannot be resolved");
  }

  function resolveWidgetSiteId(scriptTag) {
    if (!scriptTag) {
      throw new Error("widget.js: resolve_site_id.failed: missing script tag");
    }
    var candidate = scriptTag.getAttribute("data-site-id");
    if (candidate) {
      return candidate;
    }
    if (scriptTag && scriptTag.src) {
      try {
        var link = document.createElement("a");
        link.href = scriptTag.src;
        var querySiteId = getQueryParam(link.search || "", "site_id");
        if (querySiteId) {
          return querySiteId;
        }
      } catch(e){}
    }
    throw new Error("widget.js: resolve_site_id.failed: site_id not provided in data-site-id or query string");
  }

  function normalizeWidgetPlacementSide(rawValue) {
    var normalized =
      typeof rawValue === "string" ? rawValue.trim().toLowerCase() : "";
    if (normalized === "left" || normalized === "right") {
      return normalized;
    }
    return null;
  }

  function normalizeWidgetPlacementOffset(rawValue) {
    var num = Number(rawValue);
    if (!Number.isFinite(num)) {
      return null;
    }
    return Math.round(num);
  }

  function normalizeWidgetVisibilityValue(rawValue) {
    if (typeof rawValue !== "boolean") {
      return null;
    }
    return rawValue;
  }

  function applyWidgetPlacementConfig(config) {
    if (!config) {
      return;
    }
    var side = normalizeWidgetPlacementSide(config.side);
    var offset = normalizeWidgetPlacementOffset(config.bottomOffset);
    if (side) {
      widgetPlacementSideValue = side;
    }
    if (typeof offset === "number") {
      widgetPlacementBottomOffsetValue = offset;
    }
    var showMessageInput = normalizeWidgetVisibilityValue(config.showMessageInput);
    var showSentimentButtons = normalizeWidgetVisibilityValue(config.showSentimentButtons);
    if (typeof showMessageInput === "boolean") {
      widgetShowMessageInputValue = showMessageInput;
    }
    if (typeof showSentimentButtons === "boolean") {
      widgetShowSentimentButtonsValue = showSentimentButtons;
    }
  }

  function fetchWidgetPlacementFromPublicAPI() {
    if (!widgetApiOrigin || !widgetSiteId) {
      return Promise.resolve(null);
    }
    var requestURL =
      widgetApiOrigin +
      "/public/widget-config?site_id=" +
      encodeURIComponent(widgetSiteId);
    return fetch(requestURL, {
      method: "GET",
      headers: { "Accept": "application/json" },
      credentials: "omit",
      referrer: window.location.href,
      referrerPolicy: "strict-origin-when-cross-origin",
    })
      .then(function(response) {
        if (!response) {
          var missingResponseError = new Error("widget_config_failed");
          // @ts-ignore
          missingResponseError.status = 0;
          throw missingResponseError;
        }
        if (response.status === 403 || response.status === 404) {
          var forbiddenError = new Error("widget_config_forbidden");
          // @ts-ignore
          forbiddenError.status = response.status;
          throw forbiddenError;
        }
        if (!response.ok) {
          var requestError = new Error("widget_config_failed");
          // @ts-ignore
          requestError.status = response.status;
          throw requestError;
        }
        return response.json();
      })
      .then(function(payload) {
        if (!payload || typeof payload !== "object") {
          return null;
        }
        return {
          side: payload.widget_bubble_side,
          bottomOffset: payload.widget_bubble_bottom_offset,
          showMessageInput: payload.widget_show_message_input,
          showSentimentButtons: payload.widget_show_sentiment_buttons,
        };
      });
  }

  function fetchWidgetPlacementFromDashboard() {
    if (!widgetApiOrigin || !widgetSiteId) {
      return Promise.resolve(null);
    }
    var requestURL = widgetApiOrigin + "/api/sites";
    return fetch(requestURL, {
      method: "GET",
      headers: { "Accept": "application/json" },
      credentials: "include",
      referrer: window.location.href,
      referrerPolicy: "strict-origin-when-cross-origin",
    })
      .then(function(response) {
        if (!response || !response.ok) {
          return null;
        }
        return response.json().catch(function() {
          return null;
        });
      })
      .then(function(payload) {
        if (
          !payload ||
          typeof payload !== "object" ||
          !Array.isArray(payload.sites)
        ) {
          return null;
        }
        for (var index = 0; index < payload.sites.length; index += 1) {
          var site = payload.sites[index];
          if (!site || typeof site !== "object") {
            continue;
          }
          if (String(site.id || "").trim() !== widgetSiteId) {
            continue;
          }
          return {
            side: site.widget_bubble_side,
            bottomOffset: site.widget_bubble_bottom_offset,
            showMessageInput: site.widget_show_message_input,
            showSentimentButtons: site.widget_show_sentiment_buttons,
          };
        }
        return null;
      });
  }

  function fetchWidgetPlacementConfig() {
    if (widgetTestModeEnabled) {
      return fetchWidgetPlacementFromDashboard();
    }
    return fetchWidgetPlacementFromPublicAPI();
  }

  function scheduleWhenBodyReady() {
    if (widgetInitialized) {
      return;
    }
    if (!document.body) {
      window.setTimeout(scheduleWhenBodyReady, 30);
      return;
    }
    renderWidget();
  }

  function renderWidget() {
    if (widgetInitialized) {
      return;
    }
    widgetInitialized = true;
    try {
      var existingBubble = document.getElementById("mp-feedback-bubble");
      if (existingBubble) {
        return;
      }
      var panelAutoHideTimer = null;

      var bodyElement = document.body;
      var themePalette = selectThemePalette(bodyElement);
      var currentStatusState = statusStatePending;
      var selectedSentimentValue = "";

      var resolvedBubbleSide = widgetPlacementSideValue === "left" ? "left" : "right";
      var resolvedBottomOffset = Number(widgetPlacementBottomOffsetValue);
      var resolvedShowMessageInput = widgetShowMessageInputValue !== false;
      var resolvedShowSentimentButtons = widgetShowSentimentButtonsValue !== false;
      if (!resolvedShowMessageInput && !resolvedShowSentimentButtons) {
        resolvedShowMessageInput = true;
        resolvedShowSentimentButtons = true;
      }
      if (!isFinite(resolvedBottomOffset) || resolvedBottomOffset < 0) {
        resolvedBottomOffset = widgetDefaults.placementBottomOffset;
      }
      var panelBottomOffset = resolvedBottomOffset + panelVerticalSpacingPixels;

      var bubble = document.createElement("div");
      bubble.id = "mp-feedback-bubble";
      bubble.style.position = "fixed";
      bubble.style.left = "";
      bubble.style.right = "";
      if (resolvedBubbleSide === "left") {
        bubble.style.left = widgetPlacementHorizontalOffsetValue;
      } else {
        bubble.style.right = widgetPlacementHorizontalOffsetValue;
      }
      bubble.style.bottom = resolvedBottomOffset + "px";
      bubble.style.width = "56px";
      bubble.style.height = "56px";
      bubble.style.borderRadius = "28px";
      bubble.style.cursor = "pointer";
      bubble.style.display = "flex";
      bubble.style.alignItems = "center";
      bubble.style.justifyContent = "center";
      bubble.style.zIndex = "2147483647";
      bubble.style.fontSize = "28px";
      bubble.style.userSelect = "none";
      bubble.setAttribute("aria-label","Send feedback");
      bubble.innerText = "💬";
      bodyElement.appendChild(bubble);

      var panel = document.createElement("div");
      panel.id = "mp-feedback-panel";
      panel.style.position = "fixed";
      panel.style.left = "";
      panel.style.right = "";
      if (resolvedBubbleSide === "left") {
        panel.style.left = widgetPlacementHorizontalOffsetValue;
      } else {
        panel.style.right = widgetPlacementHorizontalOffsetValue;
      }
      panel.style.bottom = panelBottomOffset + "px";
      panel.style.width = "320px";
      panel.style.maxWidth = "92vw";
      panel.style.borderRadius = "12px";
      panel.style.padding = "12px";
      panel.style.fontFamily = "system-ui, -apple-system, Segoe UI, Roboto, Ubuntu, Cantarell, Noto Sans, sans-serif";
      panel.style.display = panelDisplayNoneValue;
      panel.style.zIndex = "2147483647";

      var panelContainer = document.createElement("div");
      panelContainer.style.position = "relative";
      panel.appendChild(panelContainer);

      var headerContainer = document.createElement("div");
      headerContainer.style.display = widgetHeaderContainerDisplayValue;
      headerContainer.style.alignItems = widgetHeaderContainerAlignItemsValue;
      headerContainer.style.justifyContent = widgetHeaderContainerJustifyContentValue;
      headerContainer.style.gap = widgetHeaderContainerGapValue;
      headerContainer.style.marginBottom = widgetHeaderContainerMarginBottomValue;
      panelContainer.appendChild(headerContainer);

      var headline = document.createElement("div");
      headline.id = widgetHeadlineElementID;
      headline.style.fontWeight = widgetHeadlineFontWeightValue;
      headline.style.flexGrow = widgetHeadlineFlexGrowValue;
      headline.innerText = widgetDemoModeEnabled ? "Example widget" : "Send feedback";
      headerContainer.appendChild(headline);

      var closeButton = document.createElement("button");
      closeButton.type = "button";
      closeButton.innerText = widgetCloseButtonText;
      closeButton.style.width = widgetCloseButtonWidthValue;
      closeButton.style.height = widgetCloseButtonHeightValue;
      closeButton.style.padding = widgetCloseButtonPaddingValue;
      closeButton.style.border = widgetCloseButtonBorderValue;
      closeButton.style.background = widgetCloseButtonBackgroundValue;
      closeButton.style.fontSize = widgetCloseButtonFontSizeValue;
      closeButton.style.lineHeight = widgetCloseButtonLineHeightValue;
      closeButton.style.cursor = widgetCloseButtonCursorValue;
      closeButton.style.opacity = widgetCloseButtonOpacityValue;
      closeButton.style.boxSizing = boxSizingBorderBoxValue;
      closeButton.style.marginLeft = widgetCloseButtonMarginLeftValue;
      closeButton.style.flexShrink = widgetCloseButtonFlexShrinkValue;
      closeButton.setAttribute("aria-label", widgetCloseButtonAriaLabel);
      headerContainer.appendChild(closeButton);

      var contact = document.createElement("input");
      contact.id = "mp-feedback-contact";
      contact.type = "text";
      contact.placeholder = "Email or phone";
      contact.autocomplete = "off";
      contact.style.width = "100%";
      contact.style.margin = "6px 0";
      contact.style.padding = "10px";
      contact.style.borderRadius = "8px";
      contact.style.boxSizing = boxSizingBorderBoxValue;
      panelContainer.appendChild(contact);

      var sentimentSection = null;
      var sentimentButtonRow = null;
      var sentimentButtons = [];

      if (resolvedShowSentimentButtons) {
        sentimentSection = document.createElement("div");
        sentimentSection.id = "mp-feedback-sentiment";
        sentimentSection.style.margin = "6px 0 8px";
        panelContainer.appendChild(sentimentSection);

        var sentimentLabel = document.createElement("div");
        sentimentLabel.textContent = widgetSentimentPromptText;
        sentimentLabel.style.fontSize = "12px";
        sentimentLabel.style.fontWeight = "600";
        sentimentLabel.style.marginBottom = "6px";
        sentimentSection.appendChild(sentimentLabel);

        sentimentButtonRow = document.createElement("div");
        sentimentButtonRow.style.display = "flex";
        sentimentButtonRow.style.justifyContent = "center";
        sentimentButtonRow.style.alignItems = "center";
        sentimentButtonRow.style.gap = widgetSentimentButtonRowGapValue;
        sentimentSection.appendChild(sentimentButtonRow);
      }

      function updateSentimentSelectionStyles() {
        for (var sentimentButtonIndex = 0; sentimentButtonIndex < sentimentButtons.length; sentimentButtonIndex++) {
          var sentimentButton = sentimentButtons[sentimentButtonIndex];
          var sentimentValue = sentimentButton.getAttribute("data-sentiment-value") || "";
          var isSelected = sentimentValue === selectedSentimentValue;
          sentimentButton.setAttribute("aria-pressed", isSelected ? "true" : "false");
          sentimentButton.style.border = "0";
          sentimentButton.style.background = "transparent";
          sentimentButton.style.color = themePalette.panelTextColor;
          sentimentButton.style.fontWeight = isSelected ? "700" : "600";
          sentimentButton.style.opacity = isSelected ? "1" : "0.72";
          sentimentButton.style.transform = isSelected ? "scale(1.08)" : "scale(1)";
          sentimentButton.style.filter = isSelected ? "drop-shadow(0 6px 12px rgba(0,0,0,0.16))" : "none";
        }
      }

      function setSelectedSentiment(value) {
        if (value === selectedSentimentValue) {
          selectedSentimentValue = "";
        } else {
          selectedSentimentValue = value || "";
        }
        updateSentimentSelectionStyles();
      }

      if (resolvedShowSentimentButtons && sentimentButtonRow) {
        for (var sentimentOptionIndex = 0; sentimentOptionIndex < widgetSentimentOptions.length; sentimentOptionIndex++) {
          var sentimentOption = widgetSentimentOptions[sentimentOptionIndex];
          var sentimentButton = document.createElement("button");
          sentimentButton.type = "button";
          sentimentButton.id = "mp-feedback-sentiment-" + sentimentOption.value;
          sentimentButton.setAttribute("data-sentiment-value", sentimentOption.value);
          sentimentButton.setAttribute("aria-label", sentimentOption.label);
          sentimentButton.setAttribute("aria-pressed", "false");
          sentimentButton.title = sentimentOption.label;
          sentimentButton.textContent = sentimentOption.emoji;
          sentimentButton.style.width = widgetSentimentButtonSizeValue;
          sentimentButton.style.height = widgetSentimentButtonSizeValue;
          sentimentButton.style.padding = "0";
          sentimentButton.style.borderRadius = "999px";
          sentimentButton.style.border = "0";
          sentimentButton.style.background = "transparent";
          sentimentButton.style.boxSizing = boxSizingBorderBoxValue;
          sentimentButton.style.cursor = "pointer";
          sentimentButton.style.display = "flex";
          sentimentButton.style.alignItems = "center";
          sentimentButton.style.justifyContent = "center";
          sentimentButton.style.flex = "0 0 auto";
          sentimentButton.style.fontSize = widgetSentimentButtonFontSizeValue;
          sentimentButton.style.lineHeight = "1";
          sentimentButton.style.transition = widgetSentimentButtonTransitionValue;
          sentimentButton.addEventListener("click", function(event){
            var button = event.currentTarget;
            var value = button && typeof button.getAttribute === "function" ? (button.getAttribute("data-sentiment-value") || "") : "";
            setSelectedSentiment(value);
          });
          sentimentButtonRow.appendChild(sentimentButton);
          sentimentButtons.push(sentimentButton);
        }
      }

      var message = null;
      if (resolvedShowMessageInput) {
        message = document.createElement("textarea");
        message.id = "mp-feedback-message";
        message.placeholder = "Your message";
        message.rows = 4;
        message.style.width = "100%";
        message.style.margin = "6px 0 8px";
        message.style.padding = "10px";
        message.style.borderRadius = "8px";
        message.style.boxSizing = boxSizingBorderBoxValue;
        panelContainer.appendChild(message);
      }

      var send = null;
      function getPanelFocusableElements() {
        var orderedElements = [contact];
        for (var focusableSentimentIndex = 0; focusableSentimentIndex < sentimentButtons.length; focusableSentimentIndex++) {
          orderedElements.push(sentimentButtons[focusableSentimentIndex]);
        }
        if (message) {
          orderedElements.push(message);
        }
        orderedElements.push(send);
        var focusableElements = [];
        for (var elementIndex = 0; elementIndex < orderedElements.length; elementIndex++) {
          var candidateElement = orderedElements[elementIndex];
          if (candidateElement) {
            focusableElements.push(candidateElement);
          }
        }
        return focusableElements;
      }

      /** @param {KeyboardEvent} event */
      function handleInputTabNavigation(event) {
        if (event.key !== "Tab") {
          return;
        }
        var focusableElements = getPanelFocusableElements();
        var focusedElement = event.target;
        var currentIndex = focusableElements.indexOf(focusedElement);
        if (currentIndex === -1 || focusableElements.length === 0) {
          return;
        }
        var direction = event.shiftKey === true ? -1 : 1;
        var nextIndex = (currentIndex + direction + focusableElements.length) % focusableElements.length;
        event.preventDefault();
        focusInputElement(focusableElements[nextIndex]);
      }

      contact.addEventListener("keydown", handleInputTabNavigation);
      if (message) {
        message.addEventListener("keydown", handleInputTabNavigation);
      }
      for (var sentimentKeydownIndex = 0; sentimentKeydownIndex < sentimentButtons.length; sentimentKeydownIndex++) {
        sentimentButtons[sentimentKeydownIndex].addEventListener("keydown", handleInputTabNavigation);
      }

      send = document.createElement("button");
      send.type = "button";
      send.innerText = "Send";
      send.style.width = "100%";
      send.style.padding = "10px 12px";
      send.style.border = "0";
      send.style.borderRadius = "8px";
      send.style.fontWeight = "600";
      send.style.cursor = "pointer";
      send.style.boxSizing = boxSizingBorderBoxValue;
      panelContainer.appendChild(send);
      send.addEventListener("keydown", handleInputTabNavigation);

      var status = document.createElement("div");
      status.style.marginTop = "6px";
      status.style.fontSize = "12px";
      status.style.minHeight = "16px";
      panelContainer.appendChild(status);

      var brandingContainer = document.createElement("div");
      brandingContainer.id = widgetBrandingElementID;
      brandingContainer.style.marginTop = widgetBrandingMarginTopValue;
      brandingContainer.style.fontSize = widgetBrandingFontSizeValue;
      brandingContainer.style.textAlign = widgetBrandingTextAlignValue;
      brandingContainer.style.opacity = widgetBrandingOpacityValue;
      brandingContainer.style.lineHeight = widgetBrandingLineHeightValue;

      var brandingStaticTextNode = document.createElement("span");
      brandingStaticTextNode.innerText = widgetBrandingStaticText;
      var brandingLink = document.createElement("a");
      brandingLink.href = widgetBrandingLinkURL;
      brandingLink.innerText = widgetBrandingLinkText;
      brandingLink.target = widgetBrandingLinkTargetValue;
      brandingLink.rel = widgetBrandingLinkRelValue;
      brandingLink.style.color = widgetBrandingLinkColorValue;
      brandingLink.style.textDecoration = widgetBrandingTextDecorationValue;

      brandingContainer.appendChild(brandingStaticTextNode);
      brandingContainer.appendChild(brandingLink);
      panelContainer.appendChild(brandingContainer);

      bodyElement.appendChild(panel);

      /** @param {HTMLElement} targetElement */
      function focusInputElement(targetElement) {
        if (!targetElement || typeof targetElement.focus !== "function") {
          return;
        }
        try {
          targetElement.focus();
        } catch(focusImmediateError){}
        if (typeof window === "undefined") {
          return;
        }
        if (typeof window.requestAnimationFrame === "function") {
          window.requestAnimationFrame(function(){
            try {
              targetElement.focus();
            } catch(focusAnimationFrameError){}
          });
          return;
        }
        window.setTimeout(function(){
          try {
            targetElement.focus();
          } catch(focusTimeoutError){}
        }, 0);
      }

      function selectStatusColor(palette, statusState) {
        if (statusState === statusStateSuccess) {
          return palette.statusPositiveColor;
        }
        if (statusState === statusStatePending) {
          return palette.statusPendingColor;
        }
        return palette.statusNegativeColor;
      }

      function applyThemePaletteToElements(palette) {
        bubble.style.boxShadow = palette.bubbleShadow;
        bubble.style.background = palette.bubbleBackground;
        bubble.style.color = palette.bubbleTextColor;
        panel.style.background = palette.panelBackground;
        panel.style.border = palette.panelBorder;
        panel.style.boxShadow = palette.panelShadow;
        panel.style.color = palette.panelTextColor;
        contact.style.border = palette.inputBorder;
        contact.style.background = palette.inputBackground;
        contact.style.color = palette.inputTextColor;
        if (message) {
          message.style.border = palette.inputBorder;
          message.style.background = palette.inputBackground;
          message.style.color = palette.inputTextColor;
        }
        send.style.background = palette.buttonBackground;
        send.style.color = palette.buttonTextColor;
        closeButton.style.color = palette.closeButtonColor;
        status.style.color = selectStatusColor(palette, currentStatusState);
        updateSentimentSelectionStyles();
      }

      function refreshThemePalette() {
        var updatedPalette = selectThemePalette(bodyElement);
        if (updatedPalette === themePalette) {
          return;
        }
        themePalette = updatedPalette;
        applyThemePaletteToElements(themePalette);
      }

      function monitorThemeChanges() {
        if (typeof MutationObserver === "function") {
          try {
            var themeObserver = new MutationObserver(function(){
              refreshThemePalette();
            });
            themeObserver.observe(document.documentElement, {attributes: true, attributeFilter: ["class", "data-theme", "data-bs-theme", "style"]});
            if (document.body) {
              themeObserver.observe(document.body, {attributes: true, attributeFilter: ["class", "data-bs-theme", "style"]});
            }
          } catch(themeObserverError){}
        }
        if (typeof window !== "undefined" && typeof window.matchMedia === "function") {
          try {
            var themeMediaQueryList = window.matchMedia(themeMediaQueryDark);
            var handleThemeMediaChange = function(){
              refreshThemePalette();
            };
            if (typeof themeMediaQueryList.addEventListener === "function") {
              themeMediaQueryList.addEventListener("change", handleThemeMediaChange);
            } else if (typeof themeMediaQueryList.addListener === "function") {
              themeMediaQueryList.addListener(handleThemeMediaChange);
            }
          } catch(mediaQueryObserverError){}
        }
      }

      applyThemePaletteToElements(themePalette);
      monitorThemeChanges();

      function focusContactInput() {
        focusInputElement(contact);
      }

      function cancelPanelAutoHide() {
        if (panelAutoHideTimer) {
          window.clearTimeout(panelAutoHideTimer);
          panelAutoHideTimer = null;
        }
      }

      function schedulePanelAutoHide() {
        cancelPanelAutoHide();
        panelAutoHideTimer = window.setTimeout(function(){
          panel.style.display = panelDisplayNoneValue;
          cancelPanelAutoHide();
        }, panelAutoHideDelayMilliseconds);
      }

      closeButton.addEventListener("mouseenter", function(){
        closeButton.style.opacity = widgetCloseButtonHoverOpacityValue;
      });

      closeButton.addEventListener("mouseleave", function(){
        closeButton.style.opacity = widgetCloseButtonOpacityValue;
      });

      closeButton.addEventListener("click", function(){
        cancelPanelAutoHide();
        panel.style.display = panelDisplayNoneValue;
      });

      bubble.addEventListener("click", function(){
        cancelPanelAutoHide();
        var panelShouldShow = panel.style.display === panelDisplayNoneValue;
        panel.style.display = (panelShouldShow ? panelDisplayBlockValue : panelDisplayNoneValue);
        if (panelShouldShow) {
          focusContactInput();
        }
      });

      function show(messageText, statusState) {
        var resolvedStatusState = statusState;
        if (!resolvedStatusState) {
          resolvedStatusState = statusStateError;
        }
        currentStatusState = resolvedStatusState;
        status.innerText = messageText;
        status.style.color = selectStatusColor(themePalette, currentStatusState);
      }

      function normalizeContactValue(rawValue) {
        var trimmed = String(rawValue || "").trim();
        if (!trimmed) {
          return null;
        }
        if (trimmed.indexOf("@") !== -1) {
          var normalizedEmail = trimmed.toLowerCase();
          if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(normalizedEmail)) {
            return null;
          }
          return normalizedEmail;
        }
        if (!feedbackPhoneAllowedCharactersPattern.test(trimmed)) {
          return null;
        }
        var plusMatches = trimmed.match(/\+/g);
        if (plusMatches && plusMatches.length > 1) {
          return null;
        }
        if (trimmed.indexOf("+") > 0) {
          return null;
        }
        var digitsOnlyValue = trimmed.replace(/[^0-9]/g, "");
        if (digitsOnlyValue.length < feedbackPhoneMinimumDigits || digitsOnlyValue.length > feedbackPhoneMaximumDigits) {
          return null;
        }
        if (trimmed.charAt(0) === "+") {
          return "+" + digitsOnlyValue;
        }
        return digitsOnlyValue;
      }

      function validate() {
        var contactValue = normalizeContactValue(contact.value || "");
        var messageValue = message ? (message.value || "").trim() : "";
        if (!contactValue) { show("Please enter a valid email or phone.", statusStateError); return null; }
        if (messageValue.length === 0 && !selectedSentimentValue) {
          if (!resolvedShowMessageInput) {
            show("Please choose a face.", statusStateError);
            return null;
          }
          if (!resolvedShowSentimentButtons) {
            show("Please write a message.", statusStateError);
            return null;
          }
          show("Please write a message, choose a face, or both.", statusStateError);
          return null;
        }
        return {contact: contactValue, message: messageValue, sentiment: selectedSentimentValue};
      }

      send.addEventListener("click", function(){
        cancelPanelAutoHide();
        var valid = validate();
        if (!valid) { return; }
        send.disabled = true;
        show("Sending...", statusStatePending);

        if (widgetDemoModeEnabled) {
          window.setTimeout(function(){
            show("Demo mode: feedback not sent.", statusStateSuccess);
            send.disabled = false;
            schedulePanelAutoHide();
          }, 200);
          return;
        }

        var payload = JSON.stringify({
          site_id: widgetSiteId,
          contact: valid.contact,
          message: valid.message,
          sentiment: valid.sentiment
        });

        var endpoint = widgetApiOrigin
          ? (widgetApiOrigin + "/public/feedback")
          : (location.protocol + "//" + location.host + "/public/feedback");

        var targetEndpoint = endpoint;
        if (widgetTestEndpointOverride) {
          if (widgetTestEndpointOverride.indexOf("http://") === 0 || widgetTestEndpointOverride.indexOf("https://") === 0) {
            targetEndpoint = widgetTestEndpointOverride;
          } else if (widgetApiOrigin && widgetTestEndpointOverride.indexOf("/") === 0) {
            targetEndpoint = widgetApiOrigin + widgetTestEndpointOverride;
          } else {
            targetEndpoint = widgetTestEndpointOverride;
          }
        }

        var fetchOptions = {
          method: "POST",
          headers: {"Content-Type": "application/json"},
          body: payload,
          credentials: widgetTestModeEnabled ? "include" : "same-origin",
          referrer: window.location.href,
          referrerPolicy: "strict-origin-when-cross-origin"
        };
        fetch(targetEndpoint, fetchOptions).then(function(resp){
          if (!resp.ok) { throw new Error("HTTP " + resp.status); }
          return resp.json();
        }).then(function(){
          show("Thanks! Sent.", statusStateSuccess);
          contact.value = "";
          if (message) {
            message.value = "";
          }
          selectedSentimentValue = "";
          updateSentimentSelectionStyles();
          send.disabled = false;
          schedulePanelAutoHide();
        }).catch(function(err){
          show("Failed to send. Please try again.", statusStateError);
          send.disabled = false;
          console.error("widget.js: send_feedback_failed", err);
        });
      });
    } catch(widgetError) {
      widgetInitialized = false;
      console.error("widget.js: render_failed", widgetError);
      throw widgetError;
    }
  }

  function selectThemePalette(bodyElement) {
    var detectedTheme = detectPageTheme(bodyElement);
    var palette = widgetThemePalettes[detectedTheme];
    if (!palette) {
      return widgetThemePalettes[themeNameLight];
    }
    return palette;
  }

  function detectPageTheme(bodyElement) {
    var explicitTheme = detectExplicitTheme();
    if (explicitTheme) {
      return explicitTheme;
    }
    var backgroundTheme = detectThemeFromBackground(bodyElement);
    if (backgroundTheme) {
      return backgroundTheme;
    }
    if (typeof window !== "undefined" && typeof window.matchMedia === "function") {
      try {
        if (window.matchMedia(themeMediaQueryDark).matches) {
          return themeNameDark;
        }
      } catch(matchMediaError){}
    }
    return themeNameLight;
  }

  function detectExplicitTheme() {
    try {
      var rootTheme = resolveThemeFromElement(document.documentElement);
      if (rootTheme) {
        return rootTheme;
      }
      var bodyTheme = resolveThemeFromElement(document.body);
      if (bodyTheme) {
        return bodyTheme;
      }
    } catch(explicitThemeError){}
    return null;
  }

  function resolveThemeFromElement(element) {
    if (!element) {
      return null;
    }
    var attributeValue = readThemeAttribute(element);
    if (attributeValue === themeNameDark) {
      return themeNameDark;
    }
    if (attributeValue === themeNameLight) {
      return themeNameLight;
    }
    if (element.classList) {
      if (element.classList.contains(themeClassNameDark)) {
        return themeNameDark;
      }
      if (element.classList.contains(themeClassNameLight)) {
        return themeNameLight;
      }
    }
    return null;
  }

  function readThemeAttribute(element) {
    if (!element) {
      return "";
    }
    var attributeNames = [themeAttributeName, bootstrapThemeAttributeName];
    for (var index = 0; index < attributeNames.length; index++) {
      var candidateValue = element.getAttribute(attributeNames[index]);
      if (candidateValue && candidateValue.length > 0) {
        return candidateValue.toLowerCase();
      }
    }
    return "";
  }

  function detectThemeFromBackground(bodyElement) {
    if (!bodyElement || typeof window === "undefined" || typeof window.getComputedStyle !== "function") {
      return null;
    }
    try {
      var computedStyle = window.getComputedStyle(bodyElement);
      var backgroundColor = computedStyle ? computedStyle.backgroundColor : "";
      var parsedColor = parseRGBColor(backgroundColor);
      if (!parsedColor) {
        var rootElement = document.documentElement;
        if (rootElement) {
          var rootStyle = window.getComputedStyle(rootElement);
          parsedColor = parseRGBColor(rootStyle ? rootStyle.backgroundColor : "");
        }
      }
      if (!parsedColor) {
        return null;
      }
      var luminance = computeRelativeLuminance(parsedColor);
      if (luminance <= luminanceDarkThreshold) {
        return themeNameDark;
      }
      if (luminance >= luminanceLightThreshold) {
        return themeNameLight;
      }
    } catch(backgroundError){}
    return null;
  }

  function parseRGBColor(colorValue) {
    if (!colorValue) {
      return null;
    }
    var normalizedValue = colorValue.trim().toLowerCase();
    if (normalizedValue.length === 0) {
      return null;
    }
    if (normalizedValue.charAt(0) === "#") {
      if (normalizedValue.length === 4) {
        var redDigit = normalizedValue.charAt(1);
        var greenDigit = normalizedValue.charAt(2);
        var blueDigit = normalizedValue.charAt(3);
        return {
          red: parseInt(redDigit + redDigit, 16),
          green: parseInt(greenDigit + greenDigit, 16),
          blue: parseInt(blueDigit + blueDigit, 16)
        };
      }
      if (normalizedValue.length === 7) {
        return {
          red: parseInt(normalizedValue.slice(1, 3), 16),
          green: parseInt(normalizedValue.slice(3, 5), 16),
          blue: parseInt(normalizedValue.slice(5, 7), 16)
        };
      }
      return null;
    }
    if (normalizedValue.indexOf("rgb") === 0) {
      var startIndex = normalizedValue.indexOf("(");
      var endIndex = normalizedValue.lastIndexOf(")");
      if (startIndex === -1 || endIndex === -1) {
        return null;
      }
      var componentValues = normalizedValue.slice(startIndex + 1, endIndex).split(",");
      if (componentValues.length < 3) {
        return null;
      }
      var redComponent = parseColorComponent(componentValues[0]);
      var greenComponent = parseColorComponent(componentValues[1]);
      var blueComponent = parseColorComponent(componentValues[2]);
      if (redComponent === null || greenComponent === null || blueComponent === null) {
        return null;
      }
      return {
        red: redComponent,
        green: greenComponent,
        blue: blueComponent
      };
    }
    return null;
  }

  function parseColorComponent(componentText) {
    if (typeof componentText !== "string") {
      return null;
    }
    var trimmed = componentText.trim();
    if (trimmed.length === 0) {
      return null;
    }
    if (trimmed.indexOf("%") !== -1) {
      var percentageValue = parseFloat(trimmed.replace("%", ""));
      if (isNaN(percentageValue)) {
        return null;
      }
      return clampColorComponent(Math.round((percentageValue / 100) * 255));
    }
    var numericValue = parseFloat(trimmed);
    if (isNaN(numericValue)) {
      return null;
    }
    return clampColorComponent(Math.round(numericValue));
  }

  function computeRelativeLuminance(color) {
    var linearRed = normalizeChannelValue(color.red);
    var linearGreen = normalizeChannelValue(color.green);
    var linearBlue = normalizeChannelValue(color.blue);
    return (0.2126 * linearRed) + (0.7152 * linearGreen) + (0.0722 * linearBlue);
  }

  function normalizeChannelValue(channelValue) {
    var normalized = channelValue / 255;
    if (normalized <= 0.03928) {
      return normalized / 12.92;
    }
    return Math.pow((normalized + 0.055) / 1.055, 2.4);
  }

  function clampColorComponent(componentValue) {
    if (componentValue < 0) {
      return 0;
    }
    if (componentValue > 255) {
      return 255;
    }
    return componentValue;
  }

  function initializeWidget() {
    try {
      var scriptTag = resolveWidgetScriptTag();
      widgetApiOrigin = resolveWidgetAPIOrigin(scriptTag);
      widgetSiteId = resolveWidgetSiteId(scriptTag);

      if (widgetDemoModeEnabled) {
        scheduleWhenBodyReady();
        return;
      }

      fetchWidgetPlacementConfig()
        .then(function(config) {
          applyWidgetPlacementConfig(config);
          scheduleWhenBodyReady();
        })
        .catch(function(error) {
          var status = error && typeof error.status === "number" ? error.status : 0;
          if (status === 403 || status === 404) {
            console.error("widget.js: initialize_failed: forbidden or not found", error);
            return;
          }
          console.error("widget.js: initialize_failed: config fetch error", error);
          scheduleWhenBodyReady();
        });
    } catch(initError) {
      console.error("widget.js: initialize_failed", initError);
      throw initError;
    }
  }

  if (document.readyState === "loading") {
    var domContentLoadedListener = function(){
      document.removeEventListener("DOMContentLoaded", domContentLoadedListener);
      initializeWidget();
    };
    document.addEventListener("DOMContentLoaded", domContentLoadedListener);
  } else {
    initializeWidget();
  }
})();
