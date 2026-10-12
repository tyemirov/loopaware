// @ts-check
/// <reference types="node" />

import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createServer } from "node:http";
import { createRequire } from "node:module";
import { resolve, dirname } from "node:path";
import { Script } from "node:vm";
import React from "react";
import { act, create } from "react-test-renderer";
import ts from "@typescript/typescript6";

globalThis.IS_REACT_ACT_ENVIRONMENT = true;
const require = createRequire(import.meta.url);
const mobileRoot = resolve(import.meta.dirname, "..");
const account = { email: "owner@example.com", name: "Operator", role: "admin" };
const site = {
  id: "aggregate-site", name: "Aggregate site", traffic_profile: "aggregate", access_role: "admin",
  allowed_origin: "https://example.com", unique_visitor_count: null, visit_count: 12,
};

await scenario("fresh login survives dashboard failure", { initialSession: false, dashboardFailure: true }, async (app) => {
  await app.signIn();
  app.expectSignedIn();
  assert.match(app.text(), /Request failed with 409/);
  assert.match(app.text(), /Aggregate site/);
  assert.match(app.text(), /Site unavailable/);
});
await scenario("restored session survives dashboard failure", { initialSession: true, dashboardFailure: true }, async (app) => {
  app.expectSignedIn();
  assert.match(app.text(), /Aggregate site/);
  assert.match(app.text(), /Request failed with 409/);
});
await scenario("aggregate dashboard skips unavailable metrics", { initialSession: false }, async (app) => {
  await app.signIn();
  app.expectSignedIn();
  assert.match(app.text(), /Aggregate site/);
  assert.doesNotMatch(app.text(), /Request failed|Site unavailable/);
  assert.equal(app.requests.some((path) => /\/(attribution|engagement|devices|locations)$/.test(path)), false);
});
await scenario("authentication rejection stays signed out", { initialSession: false, rejectLogin: true }, async (app) => {
  await app.signIn();
  assert.match(app.text(), /Sign in with Google/);
  assert.doesNotMatch(app.text(), /Sign out/);
  assert.match(app.text(), /Request failed with 401/);
  assert.equal(app.requests.includes("/api/sites"), false);
});
await scenario("site discovery failure stays authenticated", { initialSession: true, sitesFailure: true }, async (app) => {
  app.expectSignedIn();
  assert.match(app.text(), /Sites unavailable/);
  assert.doesNotMatch(app.text(), /No sites/);
});
await scenario("late dashboard data cannot replace another selected site", { initialSession: true, deferDashboard: true }, async (app) => {
  app.expectSignedIn();
  await app.press("Other site");
  assert.match(app.text(), /Visits 23/);
  await app.press("Reports");
  assert.match(app.text(), /Visits 35/);
  assert.doesNotMatch(app.text(), /All-sites report data unavailable/);
  await app.release();
  assert.match(app.text(), /Visits 35/);
  await app.press("Overview");
  assert.match(app.text(), /Visits 23/);
  assert.doesNotMatch(app.text(), /Visits 12/);
});
await scenario("late site discovery cannot replace another selected site", { initialSession: true, deferSites: true }, async (app) => {
  app.expectSignedIn();
  await app.refresh();
  await app.press("Other site");
  assert.match(app.text(), /Visits 23/);
  await app.press("Reports");
  assert.match(app.text(), /Visits 58/);
  await app.release();
  assert.match(app.text(), /Visits 58/);
  await app.press("Overview");
  assert.match(app.text(), /Visits 23/);
  assert.doesNotMatch(app.text(), /Visits 12/);
});
await scenario("portfolio failure preserves the selected-site dashboard", { initialSession: true, portfolioFailure: true }, async (app) => {
  app.expectSignedIn();
  assert.match(app.text(), /Visits 12/);
  assert.match(app.text(), /Request failed with 500/);
  await app.press("Reports");
  assert.match(app.text(), /All-sites report data unavailable/);
});
await scenario("late portfolio data cannot replace a new session's reports", { initialSession: true, deferPortfolio: true }, async (app) => {
  await app.press("Other site");
  assert.match(app.text(), /Visits 23/);
  await app.press("Sign out");
  assert.match(app.text(), /Sign in with Google/);
  await app.signIn();
  await app.press("Reports");
  assert.match(app.text(), /Visits 58/);
  await app.release();
  assert.match(app.text(), /Visits 58/);
  assert.doesNotMatch(app.text(), /Visits 35/);
});

console.log("mobile app flow checks passed");

/**
 * @param {string} name
 * @param {{ initialSession: boolean, dashboardFailure?: boolean, rejectLogin?: boolean, sitesFailure?: boolean, deferDashboard?: boolean, deferSites?: boolean, portfolioFailure?: boolean, deferPortfolio?: boolean }} options
 * @param {(app: { signIn: () => Promise<void>, press: (label: string) => Promise<void>, refresh: () => Promise<void>, release: () => Promise<void>, expectSignedIn: () => void, text: () => string, requests: string[] }) => Promise<void>} check
 */
async function scenario(name, options, check) {
  let authenticated = options.initialSession;
  /** @type {string[]} */
  const requests = [];
  let releaseResponse = () => {};
  let deferred = options.deferDashboard || options.deferPortfolio;
  let responseHeld = false;
  let responseConsumed = () => {};
  const consumed = new Promise(resolve => { responseConsumed = () => resolve(undefined); });
  /** @type {typeof fetch} */
  const fetcher = async (input, init) => {
    const path = new URL(String(input)).pathname;
    const deferredRequest = (options.deferSites && path === "/api/sites" && requests.filter(value => value === path).length === 1)
      || (options.deferDashboard && path === `/api/sites/${site.id}/visits/stats`)
      || (options.deferPortfolio && path === "/api/reports/traffic/portfolio" && !requests.includes(path));
    const response = await fetch(input, init);
    if (deferredRequest) {
      const readJSON = response.json.bind(response);
      response.json = async () => {
        const data = await readJSON();
        responseConsumed();
        return data;
      };
    }
    return response;
  };
  const server = createServer((request, response) => {
    const path = new URL(request.url || "/", "http://localhost").pathname;
    requests.push(path);
    let status = 200;
    /** @type {unknown} */
    let payload;
    switch (path) {
      case "/api/me": status = authenticated ? 200 : 401; payload = authenticated ? account : { error: "unauthorized" }; break;
      case "/auth/refresh": status = 401; payload = { error: "missing_session" }; break;
      case "/auth/nonce": payload = { nonce: "issued-nonce" }; break;
      case "/auth/google/native/config":
        payload = { clients: [{ platform: "ios", client_id: "ios-client", redirect_uris: ["loopaware:/oauth"] }], authorization_endpoint: "https://accounts.example/auth", token_endpoint: "https://accounts.example/token", scopes: ["openid"] }; break;
      case "/auth/google/native":
        assert.equal(request.headers["x-tauth-tenant"], "loopaware");
        authenticated = !options.rejectLogin;
        status = authenticated ? 200 : 401; payload = authenticated ? account : { error: "invalid_google_token" }; break;
      case "/auth/logout": authenticated = false; payload = {}; break;
      case "/api/sites": status = options.sitesFailure ? 500 : 200; payload = options.sitesFailure ? { error: "query_failed" } : { sites: options.deferDashboard || options.deferSites || options.deferPortfolio ? [site, { ...site, id: "other-site", name: "Other site" }] : [site] }; break;
      case "/api/reports/traffic/portfolio/reports": payload = { reports: [], available_sites: [] }; break;
      case "/api/reports/traffic/portfolio":
        status = options.portfolioFailure ? 500 : 200;
        payload = options.portfolioFailure ? { error: "query_failed" } : { site_count: 2, visit_count: requests.filter(value => value === path).length === 1 ? 35 : 58, unique_visitor_count: null, trend: [], sites: [] }; break;
      case "/api/reports/traffic/portfolio/schedule": payload = null; break;
      default:
        if (/\/visits\/(attribution|engagement|devices|locations)$/.test(path) || (options.dashboardFailure && path.endsWith("/visits/stats"))) {
          status = 409; payload = { error: "metric_unavailable", traffic_profile: "aggregate" };
        } else if (path.endsWith("/visits/stats")) {
          payload = { site_id: path.split("/")[3], interval: "30days", traffic_profile: "aggregate", visit_count: path.includes("other-site") ? 23 : 12, unique_visitor_count: null, top_pages: null, recent_visits: null };
        } else if (path.endsWith("/visits/trend")) {
          payload = { site_id: path.split("/")[3], interval: "30days", trend: [] };
        } else {
          const field = { messages: "messages", subscribers: "subscribers", issues: "issues", "mobile-apps": "mobile_apps", team: "team_members" }[path.split("/").at(-1) || ""];
          payload = field ? { [field]: [] } : null;
        }
    }
    const send = () => {
      response.writeHead(status, { "Content-Type": "application/json" });
      response.end(JSON.stringify(payload));
    };
    if ((options.deferSites && path === "/api/sites" && requests.filter(value => value === path).length === 2)
      || (options.deferDashboard && path === `/api/sites/${site.id}/visits/stats`)
      || (options.deferPortfolio && path === "/api/reports/traffic/portfolio" && requests.filter(value => value === path).length === 1)) {
      responseHeld = true;
      releaseResponse = send;
    } else send();
  });
  await new Promise(resolve => server.listen(0, "127.0.0.1", () => resolve(undefined)));
  const address = server.address();
  assert(address && typeof address === "object");
  const origin = `http://127.0.0.1:${address.port}`;
  const { App, assetLoads } = loadApp(origin, fetcher);
  /** @type {import("react-test-renderer").ReactTestRenderer | undefined} */
  let renderer;
  function tree() { assert(renderer); return renderer; }
  try {
    await act(async () => { renderer = create(React.createElement(App)); });
    await settle();
    const text = () => textContent(tree().toJSON());
    /** @param {string} label */
    async function press(label) {
      deferred = false;
      const button = tree().root.findAll(node => String(node.type) === "Pressable" && textContent(node.props.children).includes(label))[0];
      assert(button, `${name}: ${label}`);
      assert.notEqual(button.props.disabled, true, `${name}: ${label} is disabled`);
      // Site selection events start an asynchronous load without returning its promise.
      await act(async () => { button.props.onPress(); });
      await settle();
    }
    await check({
      text, requests,
      press,
      async refresh() {
        deferred = true;
        const control = tree().root.findAll(node => String(node.type) === "ScrollView" && node.props.refreshControl)[0]?.props.refreshControl;
        assert(control);
        await act(async () => { control.props.onRefresh(); });
        await settle();
      },
      async release() {
        deferred = false;
        await act(async () => { releaseResponse(); await consumed; });
        await settle();
      },
      expectSignedIn() { assert.match(text(), /Sign out/, name); assert.doesNotMatch(text(), /Sign in with Google/, name); },
      async signIn() { await press("Sign in with Google"); },
    });
    assert(assetLoads() > 0, `${name}: real screen components rendered`);
  } finally {
    if (renderer) await act(async () => renderer?.unmount());
    server.closeAllConnections();
    await new Promise(resolve => server.close(resolve));
  }

  async function settle() {
    // Flush real HTTP responses and the React effects they trigger.
    for (let attempt = 0; attempt < 50; attempt++) {
      await act(async () => { await new Promise(resolve => setTimeout(resolve, 10)); });
      if (deferred && responseHeld) return;
      if (!tree().root.findAll(node => String(node.type) === "ActivityIndicator").length) return;
    }
    throw new Error(`${name}: app did not finish loading`);
  }
}

/** @param {unknown} node @returns {string} */
function textContent(node) {
  if (node === null || node === undefined || typeof node === "boolean") return "";
  if (typeof node === "string" || typeof node === "number") return String(node);
  if (Array.isArray(node)) return node.map(textContent).join(" ");
  const element = /** @type {{ children?: unknown, props?: { children?: unknown } }} */ (node);
  return textContent(element.children ?? element.props?.children);
}

/** @param {string} origin @param {typeof fetch} fetcher */
function loadApp(origin, fetcher) {
  let assetLoadCount = 0;
  const nativeComponents = ["ActivityIndicator", "Image", "Pressable", "RefreshControl", "ScrollView", "Text", "View"];
  /** @type {Record<string, unknown>} */
  const mocks = {
    "./assets/icon.png": 1,
    "react-native": { ...Object.fromEntries(nativeComponents.map(name => [name, name])), Platform: { OS: "ios" }, StyleSheet: { create: (/** @type {unknown} */ value) => value, hairlineWidth: 1 } },
    "react-native-safe-area-context": { SafeAreaProvider: "SafeAreaProvider", SafeAreaView: "SafeAreaView" },
    "expo-status-bar": { StatusBar: "StatusBar" },
    "expo-secure-store": { getItemAsync: async () => null, setItemAsync: async () => undefined },
    "expo-constants": { __esModule: true, default: { expoConfig: { extra: { loopAware: { apiBaseUrl: origin, tauthBaseUrl: origin, tauthTenantId: "loopaware" } } } } },
    "expo-crypto": { randomUUID: () => "nonce" },
    "expo-web-browser": { maybeCompleteAuthSession() {} },
    "expo-auth-session": {
      ResponseType: { Code: "code" },
      AuthRequest: class {
        codeVerifier = "pkce-verifier";
        async promptAsync() { return { type: "success", params: { code: "authorization-code" } }; }
      },
      exchangeCodeAsync: async () => ({ idToken: "google-id-token" }),
    },
  };
  /** @type {Map<string, { exports: any }>} */
  const modules = new Map();
  /** @param {string} filename @returns {any} */
  function load(filename) {
    const cached = modules.get(filename);
    if (cached) return cached.exports;
    const module = { exports: {} };
    modules.set(filename, module);
    const compiled = ts.transpileModule(readFileSync(filename, "utf8"), { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022, jsx: ts.JsxEmit.ReactJSX, esModuleInterop: true }, fileName: filename });
    const execute = new Script(`(function(require, module, exports, fetch) { ${compiled.outputText}\n})`, { filename }).runInThisContext();
    execute((/** @type {string} */ name) => {
      if (name === "./assets/icon.png") assetLoadCount += 1;
      if (name in mocks) return mocks[name];
      if (name.startsWith(".")) return load(resolve(dirname(filename), `${name}.ts`));
      return require(name);
    }, module, module.exports, fetcher);
    return module.exports;
  }
  return { App: load(resolve(mobileRoot, "App.tsx")).default, assetLoads: () => assetLoadCount };
}
