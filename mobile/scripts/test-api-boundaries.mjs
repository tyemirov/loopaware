// @ts-check
/// <reference types="node" />

import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import ts from "typescript";

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
