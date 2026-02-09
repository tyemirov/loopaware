// @ts-check
(function () {
  function normalizeOrigin(value) {
    var trimmed = String(value || "").trim();
    if (!trimmed) {
      return "";
    }
    if (trimmed.indexOf("http://") !== 0 && trimmed.indexOf("https://") !== 0) {
      return "";
    }
    return trimmed.replace(/\/+$/, "");
  }

  function resolveRuntimeEnv() {
    var hostname = "";
    var pageOrigin = "";
    try {
      hostname = window.location && window.location.hostname ? String(window.location.hostname) : "";
      pageOrigin = window.location && window.location.origin ? String(window.location.origin) : "";
    } catch (error) {}

    // Defaults are keyed off the frontend hostname because GitHub Pages is static.
    // Use query params for per-request overrides.
    /** @type {Array<{ name: string, hostnames: string[], services: { apiOrigin: string, tauthOrigin: string, pinguinOrigin: string } }>} */
    var environments = [
      {
        name: "production",
        hostnames: ["loopaware.mprlab.com"],
        services: {
          apiOrigin: "https://loopaware-api.mprlab.com",
          tauthOrigin: "https://tauth-api.mprlab.com",
          pinguinOrigin: "https://pinguin-api.mprlab.com",
        },
      },
      {
        name: "development",
        hostnames: ["computercat.tyemirov.net"],
        // dev is expected to run behind a reverse proxy (ghttp/Caddy) on the same origin.
        services: {
          apiOrigin: "",
          tauthOrigin: "",
          pinguinOrigin: "",
        },
      },
    ];

    var envName = "default";
    /** @type {{ apiOrigin: string, tauthOrigin: string, pinguinOrigin: string }} */
    var defaults = { apiOrigin: "", tauthOrigin: "", pinguinOrigin: "" };
    for (var i = 0; i < environments.length; i += 1) {
      var env = environments[i];
      if (env && Array.isArray(env.hostnames) && env.hostnames.indexOf(hostname) >= 0) {
        envName = env.name;
        defaults = env.services;
        break;
      }
    }

    var params = new URLSearchParams(window.location.search || "");
    var apiOrigin = normalizeOrigin(params.get("api_origin") || "") || normalizeOrigin(defaults.apiOrigin);
    var tauthOrigin =
      normalizeOrigin(params.get("tauth_origin") || "") ||
      normalizeOrigin(defaults.tauthOrigin) ||
      apiOrigin;
    var pinguinOrigin =
      normalizeOrigin(params.get("pinguin_origin") || "") ||
      normalizeOrigin(defaults.pinguinOrigin);

    // Treat same-origin as the default "single-origin" mode, so the app doesn't
    // generate unnecessary api_origin params in snippets.
    if (pageOrigin) {
      if (apiOrigin && apiOrigin === pageOrigin) {
        apiOrigin = "";
      }
      if (tauthOrigin && tauthOrigin === pageOrigin) {
        tauthOrigin = "";
      }
      if (pinguinOrigin && pinguinOrigin === pageOrigin) {
        pinguinOrigin = "";
      }
    }

    return {
      envName: envName,
      apiOrigin: apiOrigin,
      tauthOrigin: tauthOrigin,
      pinguinOrigin: pinguinOrigin,
    };
  }

  var resolved = resolveRuntimeEnv();
  window.__LOOPAWARE_RUNTIME_ENV__ = resolved.envName;
  window.__LOOPAWARE_API_ORIGIN__ = resolved.apiOrigin;
  window.__LOOPAWARE_TAUTH_ORIGIN__ = resolved.tauthOrigin;
  // Reserved for future use (the frontend currently does not call Pinguin).
  window.__LOOPAWARE_PINGUIN_ORIGIN__ = resolved.pinguinOrigin;

  var script = document.getElementById("tauth-script");
  if (!script) {
    return;
  }
  script.src = resolved.tauthOrigin ? resolved.tauthOrigin + "/tauth.js" : "/tauth.js";
})();
