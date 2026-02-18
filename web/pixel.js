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

  function resolveSiteId(script) {
    if (!script || !script.src) {
      return "";
    }
    try {
      var link = document.createElement("a");
      link.href = script.src;
      var params = new URLSearchParams(link.search || "");
      var siteId = params.get("site_id") || script.getAttribute("data-site-id") || "";
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
        var params = new URLSearchParams(link.search || "");
        var queryOrigin = params.get("api_origin") || "";
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

  function collect() {
    var script = resolveScriptTag();
    var siteId = resolveSiteId(script);
    if (!siteId) {
      return;
    }
    var url = window.location ? window.location.href : "";
    var referrer = document.referrer || "";
    var target = resolveEndpoint(script);

    var params = new URLSearchParams();
    params.set("site_id", siteId);
    if (url) params.set("url", url);
    if (referrer) params.set("referrer", referrer);
    var visitorId = getVisitorId();
    if (visitorId) params.set("visitor_id", visitorId);

    if (navigator.sendBeacon) {
      var blob = new Blob([], { type: "application/octet-stream" });
      navigator.sendBeacon(target + "?" + params.toString(), blob);
      return;
    }
    var img = new Image(1, 1);
    img.src = target + "?" + params.toString();
  }

  try {
    if (document.readyState === "complete" || document.readyState === "interactive") {
      collect();
    } else {
      document.addEventListener("DOMContentLoaded", collect);
    }
  } catch(e){}
})();
