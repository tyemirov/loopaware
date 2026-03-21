// @ts-check
(function(){
  var endpoint = "/public/visits";
  var storageKey = "loopaware_visitor_id";
  // Public embeds should only expose site_id; LoopAware-owned script hosts map to the API internally.
  var loopAwarePublicAPIOriginsByScriptOrigin = {
    "https://loopaware.mprlab.com": "https://loopaware-api.mprlab.com",
    "https://tyemirov.github.io": "https://loopaware-api.mprlab.com"
  };

  function resolveScriptTag() {
    var script = document.currentScript;
    if (script) {
      return script;
    }
    var scripts = document.querySelectorAll('script[src*="pixel.js"]');
    if (scripts.length > 0) {
      return scripts[scripts.length - 1];
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
      var pair = pairs[i].split("=");
      if (decodeURIComponent(pair[0].replace(/\+/g, " ")) === name) {
        return decodeURIComponent((pair[1] || "").replace(/\+/g, " "));
      }
    }
    return null;
  }

  function resolveSiteId(script) {
    if (!script) {
      throw new Error("pixel.js: resolve_site_id.failed: missing script tag");
    }
    var search = "";
    try {
      if (script instanceof HTMLScriptElement && script.src) {
        var link = document.createElement("a");
        link.href = script.src;
        search = link.search || "";
      }
    } catch(e){}
    var siteId = getQueryParam(search, "site_id") || script.getAttribute("data-site-id");
    if (!siteId) {
      throw new Error("pixel.js: resolve_site_id.failed: site_id not provided in data-site-id or query string");
    }
    return String(siteId).trim();
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
      throw new Error("pixel.js: resolve_origin.failed: missing script tag");
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
        throw new Error("pixel.js: resolve_origin.failed: invalid api_origin format");
      }
      return normalizedCandidate;
    }
    var internalOrigin = resolveInternalAPIOrigin(scriptTag);
    if (internalOrigin) {
      return internalOrigin;
    }
    // Fallback to script origin is allowed as a core behavioral contract
    if (scriptTag && scriptTag.src) {
      try {
        var scriptLink = document.createElement("a");
        scriptLink.href = scriptTag.src;
        if (scriptLink.protocol && scriptLink.host) {
          return scriptLink.protocol + "//" + scriptLink.host;
        }
      } catch(originError){}
    }
    // Fallback to current host is allowed as a core behavioral contract for the pixel
    if (window.location && window.location.protocol && window.location.host) {
      return window.location.protocol + "//" + window.location.host;
    }
    throw new Error("pixel.js: resolve_origin.failed: api_origin not provided and cannot be resolved");
  }

  function resolveEndpoint(script) {
    var origin = resolveAPIOrigin(script);
    return origin + endpoint;
  }

  function getVisitorId() {
    try {
      var existing = window.localStorage.getItem(storageKey);
      if (existing) {
        return existing;
      }
      var generated = crypto.randomUUID ? crypto.randomUUID() : (Date.now().toString(16) + Math.random().toString(16).slice(2, 10));
      window.localStorage.setItem(storageKey, generated);
      return generated;
    } catch(e){
      // LocalStorage access might be blocked, non-fatal for visitor tracking
      return "";
    }
  }

  function shouldUseBeacon(requestURL) {
    if (typeof navigator === "undefined" || !navigator.sendBeacon) {
      return false;
    }
    try {
      if (!window.location || !window.location.origin) {
        return true;
      }
      var parsedRequestURL = new URL(requestURL, window.location.href);
      return parsedRequestURL.origin === window.location.origin;
    } catch(parseError) {}
    return false;
  }

  function collect() {
    var script = resolveScriptTag();
    if (!script) {
      throw new Error("pixel.js: collect.failed: script tag not found");
    }
    var siteId = resolveSiteId(script);
    var url = window.location ? window.location.href : "";
    var referrer = document.referrer || "";
    var target = resolveEndpoint(script);

    var queryString = "site_id=" + encodeURIComponent(siteId);
    if (url) queryString += "&url=" + encodeURIComponent(url);
    if (referrer) queryString += "&referrer=" + encodeURIComponent(referrer);
    var visitorId = getVisitorId();
    if (visitorId) queryString += "&visitor_id=" + encodeURIComponent(visitorId);

    var requestURL = target + "?" + queryString;

    if (shouldUseBeacon(requestURL)) {
      var blob = new Blob([], { type: "application/octet-stream" });
      navigator.sendBeacon(requestURL, blob);
      return;
    }
    var img = new Image(1, 1);
    img.src = requestURL;
  }

  try {
    if (document.readyState === "complete" || document.readyState === "interactive") {
      collect();
    } else {
      document.addEventListener("DOMContentLoaded", collect);
    }
  } catch(e){
    console.error(e);
    throw e;
  }
})();
