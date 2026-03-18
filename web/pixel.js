// @ts-check
(function(){
  var endpoint = "/public/visits";
  var storageKey = "loopaware_visitor_id";

  function resolveScriptTag() {
    var script = document.currentScript;
    if (script) {
      return script;
    }
    var scripts = document.querySelectorAll('script[src*="pixel.js"]');
    return scripts[scripts.length - 1];
  }

  function getQueryParam(search, name) {
    if (!search || !name) {
      return "";
    }
    var query = search.indexOf("?") === 0 ? search.substring(1) : search;
    var pairs = query.split("&");
    for (var i = 0; i < pairs.length; i++) {
      var pair = pairs[i].split("=");
      if (decodeURIComponent(pair[0]) === name) {
        return decodeURIComponent(pair[1] || "");
      }
    }
    return "";
  }

  function resolveSiteId(script) {
    if (!script || !script.src) {
      return "";
    }
    try {
      var link = document.createElement("a");
      link.href = script.src;
      var siteId = getQueryParam(link.search || "", "site_id") || script.getAttribute("data-site-id") || "";
      return String(siteId || "").trim();
    } catch(e){}
    return "";
  }

  function resolveEndpoint(script) {
    var apiOriginOverride = resolveAPIOriginOverride(script);
    if (apiOriginOverride) {
      return apiOriginOverride + endpoint;
    }
    try {
      if (script && script.src) {
        var link = document.createElement("a");
        link.href = script.src;
        return link.protocol + "//" + link.host + endpoint;
      }
    } catch(e){}
    return endpoint;
  }

  function normalizeAPIOriginOverride(rawValue) {
    if (typeof rawValue !== "string") {
      return "";
    }
    var trimmed = rawValue.trim();
    if (!trimmed) {
      return "";
    }
    if (trimmed.indexOf("http://") !== 0 && trimmed.indexOf("https://") !== 0) {
      return "";
    }
    try {
      var parsed = new URL(trimmed);
      var origin = parsed && typeof parsed.origin === "string" ? parsed.origin : "";
      if (!origin || origin === "null") {
        return "";
      }
      return origin.replace(/\/+$/, "");
    } catch(parseError) {}
    return "";
  }

  function resolveAPIOriginOverride(scriptTag) {
    if (!scriptTag) {
      return "";
    }
    var candidate = "";
    try {
      if (typeof scriptTag.getAttribute === "function") {
        candidate = scriptTag.getAttribute("data-api-origin") || "";
      }
    } catch(attributeError){}
    try {
      if (scriptTag.src) {
        var link = document.createElement("a");
        link.href = scriptTag.src;
        var queryOrigin = getQueryParam(link.search || "", "api_origin");
        if (queryOrigin) {
          candidate = queryOrigin;
        }
      }
    } catch(parseError){}
    return normalizeAPIOriginOverride(candidate);
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
      return "";
    }
  }

  function shouldUseBeacon(requestURL) {
    if (!navigator.sendBeacon) {
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
    var siteId = resolveSiteId(script);
    if (!siteId) {
      return;
    }
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
  } catch(e){}
})();
