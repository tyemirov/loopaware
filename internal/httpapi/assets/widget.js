(function(){
  var widgetInitialized = false;
  var themeNameLight = "light";
  var themeNameDark = "dark";
  var themeAttributeName = "data-theme";
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
  var widgetPlacementSideValue = "{{ .WidgetBubbleSide }}";
  var widgetPlacementBottomOffsetValue = {{ .WidgetBottomOffset }};
  var widgetPlacementDefaultBottomOffsetValue = 16;
  var widgetPlacementHorizontalOffsetValue = "16px";
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
  var widgetHeaderContainerDisplayValue = "block";
  var widgetHeaderContainerPositionValue = "relative";
  var widgetHeaderContainerMarginBottomValue = "12px";
  var widgetHeaderPaddingRightValue = "48px";
  var widgetHeaderTitleMarginBottomValue = "8px";
  var widgetCloseButtonText = "×";
  var widgetCloseButtonFontSizeValue = "24px";
  var widgetCloseButtonLineHeightValue = "1";
  var widgetCloseButtonCursorValue = "pointer";
  var widgetCloseButtonPaddingValue = "0";
  var widgetCloseButtonBorderValue = "none";
  var widgetCloseButtonBackgroundValue = "transparent";
  var widgetCloseButtonPositionValue = "absolute";
  var widgetCloseButtonTopValue = "0px";
  var widgetCloseButtonRightValue = "0px";
  var widgetCloseButtonWidthValue = "28px";
  var widgetCloseButtonHeightValue = "28px";
  var widgetCloseButtonOpacityValue = "0.6";
  var widgetCloseButtonHoverOpacityValue = "1";
  var widgetCloseButtonAriaLabel = "Close feedback panel";
  var widgetDemoModeFlagName = "LOOPAWARE_WIDGET_DEMO_MODE";
  var widgetTestModeFlagName = "LOOPAWARE_WIDGET_TEST_MODE";
  var widgetTestEndpointFlagName = "LOOPAWARE_WIDGET_TEST_ENDPOINT";
  var widgetDemoModeEnabled = false;
  var widgetTestModeEnabled = false;
  var widgetTestEndpointOverride = "";
  try {
    if (typeof window === "object" && window) {
      widgetDemoModeEnabled = Boolean(window[widgetDemoModeFlagName]);
      widgetTestModeEnabled = Boolean(window[widgetTestModeFlagName]);
      var testEndpointCandidate = window[widgetTestEndpointFlagName];
      if (typeof testEndpointCandidate === "string" && testEndpointCandidate.trim().length > 0) {
        widgetTestEndpointOverride = testEndpointCandidate;
      }
    }
  } catch(testModeReadError){}
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

      var resolvedBubbleSide = (widgetPlacementSideValue || "").toLowerCase() === "left" ? "left" : "right";
      var resolvedBottomOffset = Number(widgetPlacementBottomOffsetValue);
      if (!isFinite(resolvedBottomOffset) || resolvedBottomOffset < 0) {
        resolvedBottomOffset = widgetPlacementDefaultBottomOffsetValue;
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
      bubble.style.boxShadow = themePalette.bubbleShadow;
      bubble.style.background = themePalette.bubbleBackground;
      bubble.style.cursor = "pointer";
      bubble.style.display = "flex";
      bubble.style.alignItems = "center";
      bubble.style.justifyContent = "center";
      bubble.style.zIndex = "2147483647";
      bubble.style.color = themePalette.bubbleTextColor;
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
      panel.style.background = themePalette.panelBackground;
      panel.style.border = themePalette.panelBorder;
      panel.style.boxShadow = themePalette.panelShadow;
      panel.style.borderRadius = "12px";
      panel.style.padding = "12px";
      panel.style.fontFamily = "system-ui, -apple-system, Segoe UI, Roboto, Ubuntu, Cantarell, Noto Sans, sans-serif";
      panel.style.display = panelDisplayNoneValue;
      panel.style.zIndex = "2147483647";
      panel.style.color = themePalette.panelTextColor;

      var panelContainer = document.createElement("div");
      panelContainer.style.position = "relative";
      panel.appendChild(panelContainer);

      var headerContainer = document.createElement("div");
      headerContainer.style.display = widgetHeaderContainerDisplayValue;
      headerContainer.style.position = widgetHeaderContainerPositionValue;
      headerContainer.style.marginBottom = widgetHeaderContainerMarginBottomValue;
      headerContainer.style.paddingRight = widgetHeaderPaddingRightValue;
      panelContainer.appendChild(headerContainer);

      var headline = document.createElement("div");
      headline.style.fontWeight = "600";
      headline.style.marginBottom = widgetHeaderTitleMarginBottomValue;
      headline.innerText = widgetDemoModeEnabled ? "Example widget" : "Send feedback";
      headerContainer.appendChild(headline);

      var closeButton = document.createElement("button");
      closeButton.type = "button";
      closeButton.innerText = widgetCloseButtonText;
      closeButton.style.position = widgetCloseButtonPositionValue;
      closeButton.style.top = widgetCloseButtonTopValue;
      closeButton.style.right = widgetCloseButtonRightValue;
      closeButton.style.width = widgetCloseButtonWidthValue;
      closeButton.style.height = widgetCloseButtonHeightValue;
      closeButton.style.padding = widgetCloseButtonPaddingValue;
      closeButton.style.border = widgetCloseButtonBorderValue;
      closeButton.style.background = widgetCloseButtonBackgroundValue;
      closeButton.style.color = themePalette.closeButtonColor;
      closeButton.style.fontSize = widgetCloseButtonFontSizeValue;
      closeButton.style.lineHeight = widgetCloseButtonLineHeightValue;
      closeButton.style.cursor = widgetCloseButtonCursorValue;
      closeButton.style.opacity = widgetCloseButtonOpacityValue;
      closeButton.style.boxSizing = boxSizingBorderBoxValue;
      closeButton.setAttribute("aria-label", widgetCloseButtonAriaLabel);
      headerContainer.appendChild(closeButton);

      var contact = document.createElement("input");
      contact.type = "text";
      contact.placeholder = "Email or phone";
      contact.autocomplete = "email";
      contact.style.width = "100%";
      contact.style.margin = "6px 0";
      contact.style.padding = "10px";
      contact.style.border = themePalette.inputBorder;
      contact.style.borderRadius = "8px";
      contact.style.background = themePalette.inputBackground;
      contact.style.color = themePalette.inputTextColor;
      contact.style.boxSizing = boxSizingBorderBoxValue;
      panelContainer.appendChild(contact);

      var message = document.createElement("textarea");
      message.placeholder = "Your message";
      message.rows = 4;
      message.style.width = "100%";
      message.style.margin = "6px 0 8px";
      message.style.padding = "10px";
      message.style.border = themePalette.inputBorder;
      message.style.borderRadius = "8px";
      message.style.background = themePalette.inputBackground;
      message.style.color = themePalette.inputTextColor;
      message.style.boxSizing = boxSizingBorderBoxValue;
      panelContainer.appendChild(message);

      var send = document.createElement("button");
      send.type = "button";
      send.innerText = "Send";
      send.style.width = "100%";
      send.style.padding = "10px 12px";
      send.style.border = "0";
      send.style.borderRadius = "8px";
      send.style.background = themePalette.buttonBackground;
      send.style.color = themePalette.buttonTextColor;
      send.style.fontWeight = "600";
      send.style.cursor = "pointer";
      send.style.boxSizing = boxSizingBorderBoxValue;
      panelContainer.appendChild(send);

      var status = document.createElement("div");
      status.style.marginTop = "6px";
      status.style.fontSize = "12px";
      status.style.minHeight = "16px";
      status.style.color = themePalette.statusPendingColor;
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
        panel.style.display = (panel.style.display === panelDisplayNoneValue ? panelDisplayBlockValue : panelDisplayNoneValue);
      });

      function show(messageText, statusState) {
        status.innerText = messageText;
        var statusColor = themePalette.statusNegativeColor;
        if (statusState === statusStateSuccess) {
          statusColor = themePalette.statusPositiveColor;
        } else if (statusState === statusStatePending) {
          statusColor = themePalette.statusPendingColor;
        }
        status.style.color = statusColor;
      }

      function validate() {
        var contactValue = (contact.value || "").trim();
        var messageValue = (message.value || "").trim();
        if (contactValue.length < 3) { show("Please enter a valid email or phone.", statusStateError); return null; }
        if (messageValue.length === 0) { show("Please write a message.", statusStateError); return null; }
        return {contact: contactValue, message: messageValue};
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
          site_id: "{{ .SiteID }}",
          contact: valid.contact,
          message: valid.message
        });

        var endpoint = (location.protocol + "//" + location.host + "/api/feedback");
        try {
          var scriptTag = document.currentScript || (function(){
            var candidates = document.querySelectorAll('script[src*="widget.js"]');
            return candidates[candidates.length - 1];
          })();
          if (scriptTag && scriptTag.src) {
            var link = document.createElement("a");
            link.href = scriptTag.src;
            endpoint = link.protocol + "//" + link.host + "/api/feedback";
          }
        } catch(fetchError){}

        var targetEndpoint = widgetTestEndpointOverride || endpoint;

        fetch(targetEndpoint, {
          method: "POST",
          headers: {"Content-Type": "application/json"},
          body: payload,
          keepalive: true
        }).then(function(resp){
          if (!resp.ok) { throw new Error("HTTP " + resp.status); }
          return resp.json();
        }).then(function(){
          show("Thanks! Sent.", statusStateSuccess);
          contact.value = "";
          message.value = "";
          send.disabled = false;
          schedulePanelAutoHide();
        }).catch(function(err){
          show("Failed to send. Please try again.", statusStateError);
          send.disabled = false;
          console.error(err);
        });
      });
    } catch(widgetError) {
      widgetInitialized = false;
      console.error(widgetError);
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
      var rootElement = document.documentElement;
      if (!rootElement) {
        return null;
      }
      var attributeValue = (rootElement.getAttribute(themeAttributeName) || "").toLowerCase();
      if (attributeValue === themeNameDark) {
        return themeNameDark;
      }
      if (attributeValue === themeNameLight) {
        return themeNameLight;
      }
      if (rootElement.classList) {
        if (rootElement.classList.contains(themeClassNameDark)) {
          return themeNameDark;
        }
        if (rootElement.classList.contains(themeClassNameLight)) {
          return themeNameLight;
        }
      }
    } catch(explicitThemeError){}
    return null;
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

  if (document.readyState === "loading") {
    var domContentLoadedListener = function(){
      document.removeEventListener("DOMContentLoaded", domContentLoadedListener);
      scheduleWhenBodyReady();
    };
    document.addEventListener("DOMContentLoaded", domContentLoadedListener);
  } else {
    scheduleWhenBodyReady();
  }
})();
