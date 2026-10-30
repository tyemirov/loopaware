// @ts-check
/// <reference types="node" />

import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import ts from "@typescript/typescript6";

const mobileRoot = path.resolve(import.meta.dirname, "..");
const apiSourcePath = path.join(mobileRoot, "src/api.ts");
const apiSource = fs.readFileSync(apiSourcePath, "utf8");
const transpiled = ts.transpileModule(apiSource, {
  compilerOptions: {
    module: ts.ModuleKind.ES2022,
    target: ts.ScriptTarget.ES2022,
  },
  fileName: apiSourcePath,
});
const moduleUrl = `data:text/javascript;base64,${Buffer.from(transpiled.outputText).toString("base64")}`;
const { LoopAwareApiClient, LoopAwareApiError } = await import(moduleUrl);

const site = {
  id: "site-1",
  traffic_profile: "detailed",
  name: "Boundary Site",
  allowed_origin: "https://example.com",
  subscribe_allowed_origins: "",
  widget_allowed_origins: "",
  traffic_allowed_origins: "",
  owner_email: "owner@example.com",
  favicon_url: "",
  widget: "",
  created_at: 0,
  feedback_count: 0,
  subscriber_count: 0,
  visit_count: 0,
  unique_visitor_count: 0,
  sentry_token_configured: false,
  widget_bubble_side: "right",
  widget_bubble_bottom_offset: 16,
  widget_accent_color: "#0d6efd",
  widget_show_message_input: true,
  widget_show_sentiment_buttons: true,
  access_role: "admin",
};

const basePayloads = {
  "/api/sites/site-1/messages": { site_id: site.id, messages: null },
  "/api/sites/site-1/subscribers": { site_id: site.id, subscribers: null },
  "/api/sites/site-1/visits/stats": {
    site_id: site.id,
    interval: "30days",
    visit_count: 0,
    unique_visitor_count: 0,
    top_pages: null,
    recent_visits: null,
  },
  "/api/sites/site-1/visits/trend": { site_id: site.id, interval: "30days", days: 30, trend: null },
  "/api/sites/site-1/visits/attribution": {
    site_id: site.id,
    interval: "30days",
    limit: 10,
    sources: null,
    mediums: null,
    campaigns: null,
  },
  "/api/sites/site-1/visits/engagement": {
    site_id: site.id,
    interval: "30days",
    days: 30,
    tracked_visitor_count: 0,
    returning_visitor_count: 0,
    returning_visitor_rate: 0,
    average_pages_per_visitor: 0,
    depth_distribution: {
      single_page: 0,
      two_to_three_pages: 0,
      four_to_seven_pages: 0,
      eight_or_more_pages: 0,
    },
    observed_time_distribution: {
      under_30_seconds: 0,
      between_30_and_119_seconds: 0,
      between_120_and_599_seconds: 0,
      at_least_600_seconds: 0,
    },
  },
  "/api/sites/site-1/visits/devices": {
    site_id: site.id,
    interval: "30days",
    limit: 10,
    device_types: null,
    top_resolutions: null,
    top_viewports: null,
  },
  "/api/sites/site-1/visits/locations": { site_id: site.id, interval: "30days", limit: 10, locations: null },
  "/api/sites/site-1/sentry/issues": { site_id: site.id, issues: null },
  "/api/sites/site-1/mobile-apps": { site_id: site.id, mobile_apps: null },
  "/api/sites/site-1/team": { site_id: site.id, team_members: null },
  "/api/sites/site-1/traffic-report-schedule": null,
  "/api/sites/site-1/health-monitor": {
    site_id: site.id,
    enabled: false,
    target_url: "https://example.com",
    interval_seconds: 300,
    timeout_seconds: 10,
    failure_threshold: 2,
    recipient_email: "owner@example.com",
    recipient_mode: "manager",
    recipient_emails: [],
    status: "unknown",
    consecutive_failures: 0,
    next_check_at: 0,
    last_checked_at: 0,
    last_success_at: 0,
    last_failure_at: 0,
    last_status_code: 0,
    last_error_code: "",
    last_error_message: "",
    last_duration_ms: 0,
    last_alerted_at: 0,
    email_enabled: true,
    persisted: false,
  },
};

const client = new LoopAwareApiClient(runtimeConfig(), createFetcher(basePayloads));
const dashboard = await client.siteDashboard(site, "30days");
assert.deepEqual(dashboard.messages, []);
assert.deepEqual(dashboard.subscribers, []);
assert.deepEqual(dashboard.stats.top_pages, []);
assert.deepEqual(dashboard.stats.recent_visits, []);
assert.deepEqual(dashboard.trend.trend, []);
assert.deepEqual(dashboard.attribution.sources, []);
assert.deepEqual(dashboard.attribution.mediums, []);
assert.deepEqual(dashboard.attribution.campaigns, []);
assert.deepEqual(dashboard.devices.device_types, []);
assert.deepEqual(dashboard.devices.top_resolutions, []);
assert.deepEqual(dashboard.devices.top_viewports, []);
assert.deepEqual(dashboard.locations.locations, []);
assert.deepEqual(dashboard.sentryIssues, []);
assert.deepEqual(dashboard.mobileApps, []);
assert.deepEqual(dashboard.teamMembers, []);
assert.equal(dashboard.healthMonitor?.status, "unknown");

/** @type {Record<string, unknown>} */
const aggregatePayloads = Object.fromEntries(Object.entries(basePayloads).filter(([requestPath]) => !/\/(attribution|engagement|devices|locations)$/.test(requestPath)));
aggregatePayloads["/api/sites/site-1/visits/stats"] = { ...basePayloads["/api/sites/site-1/visits/stats"], traffic_profile: "aggregate", unique_visitor_count: null };
const aggregateClient = new LoopAwareApiClient(runtimeConfig(), createFetcher(aggregatePayloads));
const aggregateDashboard = await aggregateClient.siteDashboard({ ...site, traffic_profile: "aggregate" }, "30days");
assert.equal(aggregateDashboard.stats.unique_visitor_count, null);
assert.equal(aggregateDashboard.attribution, null);
assert.equal(aggregateDashboard.engagement, null);
assert.equal(aggregateDashboard.devices, null);
assert.equal(aggregateDashboard.locations, null);

const invalidClient = new LoopAwareApiClient(runtimeConfig(), createFetcher({
  ...basePayloads,
  "/api/sites/site-1/subscribers": { site_id: site.id, subscribers: "invalid" },
}));
await assert.rejects(() => invalidClient.siteDashboard(site, "30days"), (error) => {
  assert(error instanceof LoopAwareApiError);
  const apiError = /** @type {{ code: string; message: string }} */ (error);
  assert.equal(apiError.code, "mobile_api_invalid_collection");
  assert.match(apiError.message, /subscribers/);
  return true;
});

const invalidTrendClient = new LoopAwareApiClient(runtimeConfig(), createFetcher({
  ...basePayloads,
  "/api/sites/site-1/visits/trend": {
    site_id: site.id, interval: "30days", days: 1,
    trend: [{ date: "2026-10-27", page_views: 12, unique_visitors: null }],
  },
}));
await assert.rejects(() => invalidTrendClient.siteDashboard(site, "30days"), (error) => {
  assert(error instanceof LoopAwareApiError);
  const apiError = /** @type {{ code: string; message: string }} */ (error);
  assert.equal(apiError.code, "mobile_api_invalid_trend");
  assert.match(apiError.message, /site-1.*2026-10-27.*unique visitor count/);
  return true;
});

const recoveringFetcher = createRecoveringFetcher(basePayloads);
const recoveringClient = new LoopAwareApiClient(runtimeConfig(), recoveringFetcher.fetcher);
const recoveredDashboard = await recoveringClient.siteDashboard(site, "30days");
assert.deepEqual(recoveredDashboard.messages, []);
assert.equal(recoveringFetcher.refreshCount(), 1);
assert.equal(recoveringFetcher.refreshTenantId(), "loopaware");
for (const requestPath of Object.keys(basePayloads)) {
  assert.equal(recoveringFetcher.apiRequestCount(requestPath), 2, `expected ${requestPath} to retry once after refresh`);
}

for (const [url, expected] of [
  ["/api/me/avatar?v=123", "https://loopaware-api.example/api/me/avatar?v=123"],
  ["https://images.example/photo.png", "https://images.example/photo.png"],
  ["data:image/svg+xml;base64,PHN2Zy8+", "data:image/svg+xml;base64,PHN2Zy8+"],
]) {
  const accountClient = new LoopAwareApiClient(runtimeConfig(), createFetcher({
    "/api/me": { email: "owner@example.com", name: "Owner", role: "admin", avatar: { url } },
  }));
  assert.equal((await accountClient.me()).avatar?.url, expected);
}

for (const avatar of [null, "invalid", {url: 42}, {url: "file:///private/photo.png"}, {url: "https://[invalid"}]) {
  const invalidAccountClient = new LoopAwareApiClient(runtimeConfig(), createFetcher({
    "/api/me": { email: "owner@example.com", name: "Owner", role: "admin", avatar },
  }));
  await assert.rejects(() => invalidAccountClient.me(), (error) => {
    assert(error instanceof LoopAwareApiError);
    const apiError = /** @type {{ code: string; message: string }} */ (error);
    assert.equal(apiError.code, "mobile_api_invalid_avatar");
    assert.match(apiError.message, /Load account avatar/);
    return true;
  });
}

const imageAccount = { email: "owner@example.com", name: "Owner", role: "admin", avatar: { url: "https://loopaware-api.example/api/me/avatar?v=123" } };
for (const cancellation of ["before-recovery", "during-recovery"]) {
  const controller = new AbortController();
  let avatarRequests = 0, refreshRequests = 0;
  /** @type {(response: Response) => void} */ let completeResponse = () => {};
  let notifyStarted = () => {};
  const started = new Promise(resolve => { notifyStarted = () => resolve(undefined); });
  const delayed = new Promise(resolve => { completeResponse = resolve; });
  const cancelClient = new LoopAwareApiClient(runtimeConfig(), /** @type {typeof fetch} */ (async input => {
    if (String(input).endsWith("/auth/refresh")) {
      refreshRequests++; notifyStarted(); return delayed;
    }
    avatarRequests++;
    if (cancellation === "before-recovery") { notifyStarted(); return delayed; }
    return new Response('{"error":"unauthorized"}', { status: 401 });
  }));
  const pending = cancelClient.accountAvatar(imageAccount, controller.signal);
  await started; controller.abort();
  completeResponse(new Response(cancellation === "before-recovery" ? '{"error":"unauthorized"}' : "{}", {
    status: cancellation === "before-recovery" ? 401 : 200,
  }));
  await assert.rejects(pending);
  assert.equal(avatarRequests, 1, "cancelled photos cannot retry after account changes");
  assert.equal(refreshRequests, cancellation === "before-recovery" ? 0 : 1);
}

let notifyReadStarted = () => {};
const readStarted = new Promise(resolve => { notifyReadStarted = () => resolve(undefined); });
let abortedReads = 0, closedBlobs = 0;
class PendingNativeFileReader {
  readAsDataURL() { notifyReadStarted(); }
  abort() { abortedReads++; }
}
const originalFileReader = Reflect.get(globalThis, "FileReader");
Reflect.set(globalThis, "FileReader", PendingNativeFileReader);
try {
  const conversionClient = new LoopAwareApiClient(runtimeConfig(), /** @type {typeof fetch} */ (async () => {
    const response = new Response(new Uint8Array([1, 2, 3]), { headers: { "Content-Type": "image/png" } });
    const readBlob = response.blob.bind(response);
    response.blob = async () => Object.assign(await readBlob(), { close() { closedBlobs++; } });
    return response;
  }));
  const controller = new AbortController();
  const pending = conversionClient.accountAvatar(imageAccount, controller.signal);
  await readStarted; controller.abort();
  await assert.rejects(pending);
  assert.equal(abortedReads, 1);
  assert.equal(closedBlobs, 1);
} finally {
  if (originalFileReader === undefined) Reflect.deleteProperty(globalThis, "FileReader");
  else Reflect.set(globalThis, "FileReader", originalFileReader);
}

for (const faviconURL of ["https://foreign.example/icon.png", "file:///private/icon.png", "https://[invalid"]) {
  const client = new LoopAwareApiClient(runtimeConfig(), /** @type {typeof fetch} */ (async () => { throw new Error("Unexpected image transfer"); }));
  await assert.rejects(client.siteFavicon({ ...site, favicon_url: faviconURL }, new AbortController().signal), error => error instanceof LoopAwareApiError && error.code === "mobile_api_invalid_favicon");
}

for (const [mime, size] of [["text/html", 4], ["image/png", 0], ["image/png", 128 * 1024 + 1]]) {
  let closed = 0;
  const client = new LoopAwareApiClient(runtimeConfig(), /** @type {typeof fetch} */ (async () => {
    const response = new Response(new Uint8Array(Number(size)), { headers: { "Content-Type": String(mime) } });
    const blob = response.blob.bind(response);
    response.blob = async () => Object.assign(await blob(), { close() { closed++; } });
    return response;
  }));
  await assert.rejects(client.siteFavicon({ ...site, favicon_url: "/api/sites/site-1/favicon?ts=123" }, new AbortController().signal), error => error instanceof LoopAwareApiError && error.code === "mobile_api_invalid_favicon");
  assert.equal(closed, 1);
}

// Exercise the public image endpoint adapter while four transfers occupy its queue.
{
  const transfers = [];
  /** @type {Array<(response: Response) => void>} */ const finish = [];
  const client = new LoopAwareApiClient(runtimeConfig(), /** @type {typeof fetch} */ (async input => {
    transfers.push(String(input));
    return new Promise(resolve => finish.push(resolve));
  }));
  const controllers = Array.from({length: 6}, () => new AbortController());
  const pending = controllers.map((controller, index) => client.siteFavicon({ ...site, favicon_url: `/api/sites/site-1/favicon?ts=${index}` }, controller.signal).catch(error => error));
  await new Promise(resolve => setImmediate(resolve));
  assert.equal(transfers.length, 4);
  controllers[4].abort();
  finish[0](new Response(null, {status: 503}));
  await pending[0];
  await new Promise(resolve => setImmediate(resolve));
  assert.equal(transfers.length, 5);
  assert.match(transfers[4], /ts=5$/);
  assert(!transfers.some(uri => uri.endsWith("ts=4")));
  finish.slice(1).forEach(resolve => resolve(new Response(null, {status:503})));
  await Promise.all(pending);
}

console.log("mobile api boundary checks passed");

function runtimeConfig() {
  return {
    apiBaseUrl: "https://loopaware-api.example",
    tauthBaseUrl: "https://tauth-api.example",
    tauthTenantId: "loopaware",
  };
}

/**
 * @param {Record<string, unknown>} payloadsByPath
 * @returns {typeof fetch}
 */
function createFetcher(payloadsByPath) {
  return (async (input) => {
    const requestUrl = new URL(input instanceof Request ? input.url : String(input));
    const payload = payloadsByPath[requestUrl.pathname];
    if (!(requestUrl.pathname in payloadsByPath)) {
      throw new Error(`unexpected_request: ${requestUrl.pathname}`);
    }
    return new Response(JSON.stringify(payload), {
      headers: { "Content-Type": "application/json" },
      status: 200,
    });
  });
}

/**
 * @param {Record<string, unknown>} payloadsByPath
 * @returns {{
 *   fetcher: typeof fetch,
 *   refreshCount: () => number,
 *   refreshTenantId: () => string,
 *   apiRequestCount: (requestPath: string) => number
 * }}
 */
function createRecoveringFetcher(payloadsByPath) {
  const apiRequestCounts = new Map();
  let refreshRequestCount = 0;
  let refreshTenantId = "";
  const fetcher = /** @type {typeof fetch} */ (async (input, init) => {
    const requestUrl = new URL(input instanceof Request ? input.url : String(input));
    if (requestUrl.pathname === "/auth/refresh") {
      refreshRequestCount += 1;
      refreshTenantId = readHeader(init?.headers, "X-TAuth-Tenant");
      assert.equal(init?.credentials, "include");
      await delay(5);
      return new Response(null, { status: 204 });
    }
    if (!(requestUrl.pathname in payloadsByPath)) {
      throw new Error(`unexpected_request: ${requestUrl.pathname}`);
    }
    assert.equal(init?.credentials, "include");
    const requestCount = (apiRequestCounts.get(requestUrl.pathname) || 0) + 1;
    apiRequestCounts.set(requestUrl.pathname, requestCount);
    if (requestCount === 1) {
      return new Response(JSON.stringify({ error: "expired_session" }), {
        headers: { "Content-Type": "application/json" },
        status: 401,
      });
    }
    return new Response(JSON.stringify(payloadsByPath[requestUrl.pathname]), {
      headers: { "Content-Type": "application/json" },
      status: 200,
    });
  });
  return {
    fetcher,
    refreshCount: () => refreshRequestCount,
    refreshTenantId: () => refreshTenantId,
    apiRequestCount: (requestPath) => apiRequestCounts.get(requestPath) || 0,
  };
}

/**
 * @param {HeadersInit | undefined} headers
 * @param {string} name
 * @returns {string}
 */
function readHeader(headers, name) {
  if (!headers) {
    return "";
  }
  if (headers instanceof Headers) {
    return headers.get(name) || "";
  }
  if (Array.isArray(headers)) {
    const entry = headers.find(([headerName]) => headerName.toLowerCase() === name.toLowerCase());
    return entry ? entry[1] : "";
  }
  const record = /** @type {Record<string, string>} */ (headers);
  return record[name] || "";
}

/**
 * @param {number} milliseconds
 * @returns {Promise<void>}
 */
function delay(milliseconds) {
  return new Promise((resolve) => {
    setTimeout(resolve, milliseconds);
  });
}
