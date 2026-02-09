// @ts-check
(function () {
  function renderFatalError(message) {
    var doc = document;
    if (!doc) return;
    var host = doc.body || doc.documentElement;
    if (!host) return;

    var banner = doc.createElement("pre");
    banner.setAttribute("data-loopaware-fatal", "true");
    banner.style.position = "fixed";
    banner.style.left = "0";
    banner.style.right = "0";
    banner.style.top = "0";
    banner.style.zIndex = "2147483647";
    banner.style.padding = "12px 16px";
    banner.style.margin = "0";
    banner.style.background = "#fff";
    banner.style.color = "#b00020";
    banner.style.borderBottom = "2px solid #b00020";
    banner.style.whiteSpace = "pre-wrap";
    banner.style.fontFamily = "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, \"Liberation Mono\", \"Courier New\", monospace";
    banner.textContent = message;
    host.appendChild(banner);
  }

  function readTextSync(url) {
    var xhr = new XMLHttpRequest();
    try {
      xhr.open("GET", url, false);
      xhr.setRequestHeader("Accept", "text/plain");
      xhr.send(null);
    } catch (error) {
      throw new Error("runtime_env.config_fetch_failed: GET " + url + " failed: " + String(error));
    }
    if (xhr.status !== 200) {
      throw new Error(
        "runtime_env.config_fetch_failed: GET " +
          url +
          " returned " +
          String(xhr.status) +
          (xhr.statusText ? " " + String(xhr.statusText) : "")
      );
    }
    return String(xhr.responseText || "");
  }

  function stripJsonYamlComments(text) {
    return String(text || "")
      .replace(/^\uFEFF/, "")
      .split(/\r?\n/)
      .filter(function (line) {
        return !/^\s*#/.test(line);
      })
      .join("\n");
  }

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

  /**
   * @param {any} value
   * @param {string} label
   */
  function requireString(value, label) {
    if (typeof value !== "string") {
      throw new Error("runtime_env.config_invalid: expected " + label + " to be a string");
    }
    return value;
  }

  /**
   * @param {any} value
   * @param {string} label
   */
  function requireStringArray(value, label) {
    if (!Array.isArray(value)) {
      throw new Error("runtime_env.config_invalid: expected " + label + " to be a list of strings");
    }
    for (var i = 0; i < value.length; i += 1) {
      if (typeof value[i] !== "string") {
        throw new Error("runtime_env.config_invalid: expected " + label + " to contain only strings");
      }
    }
    return value;
  }

  /**
   * @param {any} value
   * @param {string} label
   */
  function requireArray(value, label) {
    if (!Array.isArray(value)) {
      throw new Error("runtime_env.config_invalid: expected " + label + " to be a list");
    }
    return value;
  }

  /**
   * @param {any} value
   * @param {string} label
   */
  function requireOrigin(value, label) {
    var raw = requireString(value, label).trim();
    if (!raw) {
      return "";
    }
    var normalized = normalizeOrigin(raw);
    if (!normalized) {
      throw new Error("runtime_env.config_invalid: " + label + " must be an absolute http(s) origin or empty");
    }
    return normalized;
  }

  /**
   * `config.yml` uses JSON syntax (valid YAML 1.2) so we can parse it without
   * a YAML parser and still keep the config editable over HTTP.
   *
   * @param {string} text
   * @returns {{ environments: Array<{ name: string, hostnames: string[], services: { apiOrigin: string, tauthOrigin: string } }> }}
   */
  function parseConfig(text) {
    var cleaned = stripJsonYamlComments(text).trim();
    if (!cleaned) {
      throw new Error("runtime_env.config_invalid: /config.yml is empty");
    }

    /** @type {any} */
    var parsed = null;
    try {
      parsed = JSON.parse(cleaned);
    } catch (error) {
      throw new Error(
        "runtime_env.config_invalid: /config.yml must be JSON syntax (YAML 1.2 compatible). Parse error: " + String(error)
      );
    }
    if (!parsed || typeof parsed !== "object") {
      throw new Error("runtime_env.config_invalid: /config.yml must contain an object");
    }

    var environments = requireArray(parsed.environments, "/config.yml environments");

    /** @type {Array<{ name: string, hostnames: string[], services: { apiOrigin: string, tauthOrigin: string } }>} */
    var normalized = [];
    for (var i = 0; i < environments.length; i += 1) {
      var env = environments[i];
      if (!env || typeof env !== "object") {
        throw new Error("runtime_env.config_invalid: environments[" + String(i) + "] must be an object");
      }
      var name = requireString(env.name, "environments[" + String(i) + "].name").trim();
      if (!name) {
        throw new Error("runtime_env.config_invalid: environments[" + String(i) + "].name must be non-empty");
      }

      var hostnames = requireStringArray(env.hostnames, "environments[" + String(i) + "].hostnames")
        .map(function (hostname) {
          return String(hostname).trim().toLowerCase();
        })
        .filter(function (hostname) {
          return hostname !== "";
        });
      if (hostnames.length === 0) {
        throw new Error("runtime_env.config_invalid: environments[" + String(i) + "].hostnames must be non-empty");
      }

      if (!env.services || typeof env.services !== "object") {
        throw new Error("runtime_env.config_invalid: environments[" + String(i) + "].services must be an object");
      }
      var apiOrigin = requireOrigin(env.services.apiOrigin, "environments[" + String(i) + "].services.apiOrigin");
      var tauthOrigin = requireOrigin(env.services.tauthOrigin, "environments[" + String(i) + "].services.tauthOrigin");

      normalized.push({
        name: name,
        hostnames: hostnames,
        services: {
          apiOrigin: apiOrigin,
          tauthOrigin: tauthOrigin,
        },
      });
    }

    return { environments: normalized };
  }

  function resolveRuntimeEnv(config) {
    var hostname = String(window.location && window.location.hostname ? window.location.hostname : "").toLowerCase();
    var pageOrigin = String(window.location && window.location.origin ? window.location.origin : "");

    var envName = "default";
    /** @type {{ apiOrigin: string, tauthOrigin: string }} */
    var defaults = { apiOrigin: "", tauthOrigin: "" };

    for (var i = 0; i < config.environments.length; i += 1) {
      var env = config.environments[i];
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

    // Treat same-origin as the default "single-origin" mode, so the app doesn't
    // generate unnecessary api_origin params in snippets.
    if (pageOrigin) {
      if (apiOrigin && apiOrigin === pageOrigin) {
        apiOrigin = "";
      }
      if (tauthOrigin && tauthOrigin === pageOrigin) {
        tauthOrigin = "";
      }
    }

    if (envName === "default" && !apiOrigin && !tauthOrigin) {
      throw new Error(
        "runtime_env.env_unmapped_hostname: hostname '" +
          hostname +
          "' is not configured in /config.yml. Add it under environments[].hostnames or use ?api_origin=...&tauth_origin=..."
      );
    }

    return {
      envName: envName,
      apiOrigin: apiOrigin,
      tauthOrigin: tauthOrigin,
    };
  }

  try {
    var configText = readTextSync("/config.yml");
    var config = parseConfig(configText);
    var resolved = resolveRuntimeEnv(config);

    window.__LOOPAWARE_RUNTIME_ENV__ = resolved.envName;
    window.__LOOPAWARE_API_ORIGIN__ = resolved.apiOrigin;
    window.__LOOPAWARE_TAUTH_ORIGIN__ = resolved.tauthOrigin;

    var script = document.getElementById("tauth-script");
    if (!script) {
      return;
    }
    script.src = resolved.tauthOrigin ? resolved.tauthOrigin + "/tauth.js" : "/tauth.js";
  } catch (error) {
    var err = error instanceof Error ? error : new Error(String(error));
    var message = "LoopAware frontend bootstrap failed.\n\n" + String(err && err.message ? err.message : err);
    try {
      renderFatalError(message);
    } catch (renderError) {
      err.message = String(err.message || err) + "\n\n(runtime_env.render_failed: " + String(renderError) + ")";
    }
    throw err;
  }
})();
