// @ts-check
(function(){
  var globalClientName = "LASentry";
  var browserEndpointPath = "/sentry/browser-errors";
  var defaultEnvironment = "browser";
  var defaultLevel = "error";
  var maxStackFrames = 30;
  var maxStringLength = 500;
  var maxTagValueLength = 200;
  var maxMessageLength = 2000;
  var captureSourceWindowError = "window_error";
  var captureSourceUnhandledRejection = "unhandled_rejection";
  var captureSourceManual = "manual";
  var autoCaptureDisabledValue = "false";
  var loopAwarePublicAPIOriginsByScriptOrigin = {
    "https://loopaware.mprlab.com": "https://loopaware-api.mprlab.com",
    "https://tyemirov.github.io": "https://loopaware-api.mprlab.com"
  };

  /**
   * @typedef {{filename: string, function: string, module: string, line: number, column: number, in_app: boolean}} SentryStackFrame
   */

  /**
   * @typedef {{siteId: string, apiOrigin: string, environment: string, release: string, defaultTags: Record<string, string>, autoCapture: boolean}} BrowserLASentryConfig
   */

  /**
   * @typedef {{eventId?: string, level?: string, message?: string, exceptionType?: string, userHash?: string, tags?: Record<string, unknown>, extra?: unknown, source?: string}} CaptureAttributes
   */

  /**
   * @typedef {{message: string, exceptionType: string, stack: string}} NormalizedError
   */

  function selectScriptTag() {
    var currentScript = document.currentScript;
    if (currentScript) {
      return currentScript;
    }
    var candidates = document.querySelectorAll('script[src*="la-sentry.js"]');
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
    for (var pairIndex = 0; pairIndex < pairs.length; pairIndex += 1) {
      var rawPair = pairs[pairIndex];
      var separatorIndex = rawPair.indexOf("=");
      var rawKey = separatorIndex === -1 ? rawPair : rawPair.slice(0, separatorIndex);
      var rawValue = separatorIndex === -1 ? "" : rawPair.slice(separatorIndex + 1);
      var decodedKey = "";
      try {
        decodedKey = decodeURIComponent(rawKey.replace(/\+/g, " "));
      } catch(decodeKeyError) {
        decodedKey = "";
      }
      if (decodedKey !== name) {
        continue;
      }
      try {
        return decodeURIComponent(rawValue.replace(/\+/g, " "));
      } catch(decodeValueError) {
        return rawValue.replace(/\+/g, " ");
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
      if (parsed.search || parsed.hash || parsed.username || parsed.password) {
        return null;
      }
      return origin.replace(/\/+$/, "");
    } catch(parseError) {
      return null;
    }
  }

  function resolveAPIOriginCandidate(scriptTag) {
    if (!scriptTag) {
      throw new Error("la-sentry.js: resolve_origin.failed: missing script tag");
    }
    var candidate = scriptTag.getAttribute("data-api-origin");
    if (candidate) {
      return candidate;
    }
    if (scriptTag instanceof HTMLScriptElement && scriptTag.src) {
      var link = document.createElement("a");
      link.href = scriptTag.src;
      return getQueryParam(link.search || "", "api_origin");
    }
    return null;
  }

  function resolveInternalAPIOrigin(scriptTag) {
    if (!scriptTag || !(scriptTag instanceof HTMLScriptElement) || !scriptTag.src) {
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

  function resolveAPIOrigin(scriptTag) {
    var candidate = resolveAPIOriginCandidate(scriptTag);
    if (candidate) {
      var normalizedCandidate = normalizeAPIOrigin(candidate);
      if (!normalizedCandidate) {
        throw new Error("la-sentry.js: resolve_origin.failed: invalid api_origin format");
      }
      return normalizedCandidate;
    }
    var internalOrigin = resolveInternalAPIOrigin(scriptTag);
    if (internalOrigin) {
      return internalOrigin;
    }
    if (scriptTag instanceof HTMLScriptElement && scriptTag.src) {
      try {
        var link = document.createElement("a");
        link.href = scriptTag.src;
        if (link.protocol && link.host) {
          return link.protocol + "//" + link.host;
        }
      } catch(originError) {
        void originError;
      }
    }
    if (window.location && window.location.protocol && window.location.host) {
      return window.location.protocol + "//" + window.location.host;
    }
    throw new Error("la-sentry.js: resolve_origin.failed: api_origin not provided and cannot be resolved");
  }

  function readScriptSearch(scriptTag) {
    if (!(scriptTag instanceof HTMLScriptElement) || !scriptTag.src) {
      return "";
    }
    try {
      var link = document.createElement("a");
      link.href = scriptTag.src;
      return link.search || "";
    } catch(parseError) {
      return "";
    }
  }

  function readConfigString(scriptTag, search, queryName, attributeName, fallbackValue) {
    var queryValue = getQueryParam(search, queryName);
    if (queryValue) {
      return String(queryValue).trim();
    }
    if (scriptTag) {
      var attributeValue = scriptTag.getAttribute(attributeName);
      if (attributeValue) {
        return String(attributeValue).trim();
      }
    }
    return fallbackValue;
  }

  function readDefaultTags(scriptTag, search) {
    var tags = {};
    var serviceName = readConfigString(scriptTag, search, "service", "data-service", "");
    if (serviceName) {
      tags.service = truncateString(serviceName, maxTagValueLength);
    }
    return tags;
  }

  function parseConfig(scriptTag) {
    if (!scriptTag) {
      throw new Error("la-sentry.js: parse_config.failed: missing script tag");
    }
    var search = readScriptSearch(scriptTag);
    var siteId = readConfigString(scriptTag, search, "site_id", "data-site-id", "");
    if (!siteId) {
      throw new Error("la-sentry.js: parse_config.failed: site_id not provided");
    }
    var environment = readConfigString(scriptTag, search, "environment", "data-environment", defaultEnvironment) || defaultEnvironment;
    var release = readConfigString(scriptTag, search, "release", "data-release", "");
    var autoCaptureValue = readConfigString(scriptTag, search, "auto_capture", "data-auto-capture", "true");
    return {
      siteId: siteId,
      apiOrigin: resolveAPIOrigin(scriptTag),
      environment: truncateString(environment, maxStringLength),
      release: truncateString(release, maxStringLength),
      defaultTags: readDefaultTags(scriptTag, search),
      autoCapture: autoCaptureValue.toLowerCase() !== autoCaptureDisabledValue
    };
  }

  function truncateString(value, maxLength) {
    var normalized = String(value || "").trim();
    if (normalized.length <= maxLength) {
      return normalized;
    }
    return normalized.slice(0, maxLength);
  }

  function buildEventID() {
    try {
      if (window.crypto && typeof window.crypto.randomUUID === "function") {
        return window.crypto.randomUUID();
      }
    } catch(cryptoError) {
      return "browser-" + Date.now().toString(16) + "-" + Math.random().toString(16).slice(2);
    }
    return "browser-" + Date.now().toString(16) + "-" + Math.random().toString(16).slice(2);
  }

  function sanitizeURL(rawValue) {
    var rawText = String(rawValue || "").trim();
    if (!rawText) {
      return "";
    }
    try {
      var parsed = new URL(rawText, window.location.href);
      if (!parsed.protocol || !parsed.host) {
        return "";
      }
      return parsed.protocol + "//" + parsed.host + parsed.pathname;
    } catch(parseError) {
      return "";
    }
  }

  function currentRequestMetadata() {
    return {
      url: sanitizeURL(window.location && window.location.href ? window.location.href : ""),
      referrer: sanitizeURL(document.referrer || ""),
      user_agent: truncateString(navigator.userAgent || "", maxStringLength)
    };
  }

  function normalizeErrorInput(errorInput) {
    if (errorInput instanceof Error) {
      return {
        message: truncateString(errorInput.message || errorInput.name || "Browser error", maxMessageLength),
        exceptionType: truncateString(errorInput.name || "Error", maxStringLength),
        stack: typeof errorInput.stack === "string" ? errorInput.stack : ""
      };
    }
    var message = "";
    try {
      message = typeof errorInput === "string" ? errorInput : JSON.stringify(errorInput);
    } catch(stringifyError) {
      message = String(errorInput);
    }
    return {
      message: truncateString(message || "Browser error", maxMessageLength),
      exceptionType: "Error",
      stack: ""
    };
  }

  function parseStackFrameLine(rawLine) {
    var line = String(rawLine || "").trim();
    if (!line) {
      return null;
    }
    var chromeMatch = line.match(/^at\s+(.*?)\s+\((.*):(\d+):(\d+)\)$/);
    var anonymousChromeMatch = line.match(/^at\s+(.*):(\d+):(\d+)$/);
    var firefoxMatch = line.match(/^(.*?)@(.*):(\d+):(\d+)$/);
    var functionName = "";
    var filename = "";
    var lineNumber = 0;
    var columnNumber = 0;

    if (chromeMatch) {
      functionName = chromeMatch[1] || "";
      filename = chromeMatch[2] || "";
      lineNumber = Number(chromeMatch[3] || 0);
      columnNumber = Number(chromeMatch[4] || 0);
    } else if (anonymousChromeMatch) {
      filename = anonymousChromeMatch[1] || "";
      lineNumber = Number(anonymousChromeMatch[2] || 0);
      columnNumber = Number(anonymousChromeMatch[3] || 0);
    } else if (firefoxMatch) {
      functionName = firefoxMatch[1] || "";
      filename = firefoxMatch[2] || "";
      lineNumber = Number(firefoxMatch[3] || 0);
      columnNumber = Number(firefoxMatch[4] || 0);
    } else {
      return null;
    }

    var sanitizedFilename = sanitizeURL(filename) || truncateString(filename, maxStringLength);
    if (!sanitizedFilename && !functionName) {
      return null;
    }
    return {
      filename: sanitizedFilename,
      function: truncateString(functionName, maxStringLength),
      module: "",
      line: Number.isFinite(lineNumber) ? Math.max(0, Math.round(lineNumber)) : 0,
      column: Number.isFinite(columnNumber) ? Math.max(0, Math.round(columnNumber)) : 0,
      in_app: isInAppFrame(sanitizedFilename)
    };
  }

  function isInAppFrame(filename) {
    if (!filename || !window.location || !window.location.origin) {
      return false;
    }
    try {
      return new URL(filename, window.location.href).origin === window.location.origin;
    } catch(parseError) {
      return false;
    }
  }

  function parseStackTrace(stack) {
    var stackText = String(stack || "");
    if (!stackText) {
      return [];
    }
    var frames = [];
    var lines = stackText.split(/\r?\n/);
    for (var lineIndex = 0; lineIndex < lines.length && frames.length < maxStackFrames; lineIndex += 1) {
      var frame = parseStackFrameLine(lines[lineIndex]);
      if (frame) {
        frames.push(frame);
      }
    }
    return frames;
  }

  function normalizeTags(config, attributes) {
    var tags = {};
    Object.keys(config.defaultTags).forEach(function(tagKey) {
      tags[tagKey] = config.defaultTags[tagKey];
    });
    if (!attributes || !attributes.tags || typeof attributes.tags !== "object") {
      return tags;
    }
    Object.keys(attributes.tags).forEach(function(tagKey) {
      var normalizedKey = truncateString(tagKey, maxTagValueLength);
      if (!normalizedKey) {
        return;
      }
      tags[normalizedKey] = truncateString(String(attributes.tags[tagKey] || ""), maxTagValueLength);
    });
    return tags;
  }

  function cloneJSONValue(value) {
    if (value === undefined) {
      return {};
    }
    try {
      return JSON.parse(JSON.stringify(value));
    } catch(serializationError) {
      return { serialization_error: "extra_not_serializable" };
    }
  }

  function buildPayload(config, errorInput, attributes) {
    var resolvedAttributes = attributes || {};
    var normalizedError = normalizeErrorInput(errorInput);
    var message = resolvedAttributes.message ? String(resolvedAttributes.message) : normalizedError.message;
    var exceptionType = resolvedAttributes.exceptionType ? String(resolvedAttributes.exceptionType) : normalizedError.exceptionType;
    return {
      site_id: config.siteId,
      event_id: truncateString(resolvedAttributes.eventId || buildEventID(), maxStringLength),
      timestamp: new Date().toISOString(),
      platform: "javascript",
      environment: config.environment,
      release: config.release,
      level: truncateString(resolvedAttributes.level || defaultLevel, maxStringLength),
      message: truncateString(message, maxMessageLength),
      exception_type: truncateString(exceptionType, maxStringLength),
      stacktrace: parseStackTrace(normalizedError.stack),
      request: currentRequestMetadata(),
      user_hash: truncateString(resolvedAttributes.userHash || "", maxStringLength),
      tags: normalizeTags(config, resolvedAttributes),
      extra: cloneJSONValue(resolvedAttributes.extra || {})
    };
  }

  function submitPayload(config, payload) {
    return fetch(config.apiOrigin + browserEndpointPath, {
      method: "POST",
      headers: { "Accept": "application/json", "Content-Type": "application/json" },
      body: JSON.stringify(payload),
      credentials: "omit",
      keepalive: true,
      referrer: window.location.href,
      referrerPolicy: "strict-origin-when-cross-origin"
    }).then(function(response) {
      if (!response || !response.ok) {
        var statusCode = response ? response.status : 0;
        throw new Error("la-sentry.js: capture_failed: HTTP " + statusCode);
      }
      return response.json();
    });
  }

  function install(config) {
    function captureError(errorInput, attributes) {
      return submitPayload(config, buildPayload(config, errorInput, attributes || {}));
    }

    function captureMessage(message, attributes) {
      return captureError(new Error(String(message || "Browser message")), attributes || {});
    }

    function reportCaptureFailure(captureErrorValue) {
      console.error("la-sentry.js: capture_failed", captureErrorValue);
    }

    if (config.autoCapture) {
      window.addEventListener("error", function(event) {
        var errorValue = event.error instanceof Error ? event.error : new Error(event.message || "Unhandled browser error");
        captureError(errorValue, {
          source: captureSourceWindowError,
          extra: {
            source: captureSourceWindowError,
            filename: sanitizeURL(event.filename || ""),
            line: event.lineno || 0,
            column: event.colno || 0
          }
        }).catch(reportCaptureFailure);
      });
      window.addEventListener("unhandledrejection", function(event) {
        var reason = event.reason;
        var errorValue = reason instanceof Error ? reason : new Error(String(reason || "Unhandled promise rejection"));
        captureError(errorValue, {
          exceptionType: "UnhandledRejection",
          source: captureSourceUnhandledRejection,
          extra: {
            source: captureSourceUnhandledRejection
          }
        }).catch(reportCaptureFailure);
      });
    }

    return Object.freeze({
      captureError: function(errorInput, attributes) {
        return captureError(errorInput, attributes || { source: captureSourceManual });
      },
      captureMessage: captureMessage
    });
  }

  try {
    var config = parseConfig(selectScriptTag());
    window[globalClientName] = install(config);
  } catch(initError) {
    console.error("la-sentry.js: init_failed", initError);
    throw initError;
  }
})();
