// @ts-check
(function(){
  var siteId = "{{ .SiteID }}";
  var endpoint = "/api/visits";
  var storageKey = "loopaware_visitor_id";

  function resolveEndpoint() {
    try {
      var script = document.currentScript;
      if (!script) {
        var scripts = document.querySelectorAll('script[src*="pixel.js"]');
        script = scripts[scripts.length - 1];
      }
      if (script && script.src) {
        var link = document.createElement("a");
        link.href = script.src;
        return link.protocol + "//" + link.host + endpoint;
      }
    } catch(e){}
    return endpoint;
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
    if (!siteId) {
      return;
    }
    var url = window.location ? window.location.href : "";
    var referrer = document.referrer || "";
    var target = resolveEndpoint();

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
