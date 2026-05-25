// @ts-check
import { test, expect } from "@playwright/test";
import { resolveTestConfig } from "../helpers/config.js";
import { buildSessionCookie } from "../helpers/auth.js";
import {
  buildAdminUser,
  buildUniqueEmail,
  buildUniqueName,
  buildUniqueOrigin,
  createTestSite
} from "../helpers/fixtures.js";
import { apiRequest } from "../helpers/api.js";

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);
const baseOrigin = config.baseOrigin || new URL(config.baseURL).origin;

const nonAdminUser = buildAdminUser(config, {
  email: buildUniqueEmail("user"),
  displayName: "Regular User"
});

function buildAdminCookie() {
  return buildSessionCookie(config, adminUser);
}

function buildNonAdminCookie() {
  return buildSessionCookie(config, nonAdminUser);
}

async function adminRequest(options) {
  return apiRequest({
    baseURL: config.baseURL,
    cookie: buildAdminCookie(),
    ...options
  });
}

async function nonAdminRequest(options) {
  return apiRequest({
    baseURL: config.baseURL,
    cookie: buildNonAdminCookie(),
    ...options
  });
}

async function createAdminSite(label, overrides) {
  return createTestSite(config, buildAdminCookie(), {
    name: buildUniqueName(label),
    allowedOrigin: buildUniqueOrigin(label),
    ownerEmail: config.adminEmail,
    ...overrides
  });
}

test.describe("admin api authentication", () => {
  test("rejects unauthenticated current user", async () => {
    const { response, payload } = await apiRequest({
      baseURL: config.baseURL,
      path: "/api/me",
      method: "GET"
    });
    expect(response.status).toBe(401);
    expect(payload.error).toBe("unauthorized");
  });

  test("rejects unauthenticated site list", async () => {
    const { response, payload } = await apiRequest({
      baseURL: config.baseURL,
      path: "/api/sites",
      method: "GET"
    });
    expect(response.status).toBe(401);
    expect(payload.error).toBe("unauthorized");
  });

  test("rejects unauthenticated site creation", async () => {
    const { response, payload } = await apiRequest({
      baseURL: config.baseURL,
      path: "/api/sites",
      method: "POST",
      body: { name: "Site", allowed_origin: baseOrigin }
    });
    expect(response.status).toBe(401);
    expect(payload.error).toBe("unauthorized");
  });
});

test.describe("admin api sites", () => {
  test("returns current user payload", async () => {
    const { response, payload } = await adminRequest({ path: "/api/me", method: "GET" });
    expect(response.status).toBe(200);
    expect(payload.email).toBe(adminUser.email);
    expect(payload.role).toBe("admin");
  });

  test("lists sites", async () => {
    const site = await createAdminSite("List Sites");
    const { response, payload } = await adminRequest({ path: "/api/sites", method: "GET" });
    expect(response.status).toBe(200);
    const siteIds = Array.isArray(payload.sites) ? payload.sites.map((entry) => entry.id) : [];
    expect(siteIds).toContain(site.id);
  });

  test("create site rejects invalid json", async () => {
    const { response, payload } = await adminRequest({
      path: "/api/sites",
      method: "POST",
      rawBody: "{",
      contentType: "application/json"
    });
    expect(response.status).toBe(400);
  });

  test("create site rejects missing fields", async () => {
    const { response, payload } = await adminRequest({
      path: "/api/sites",
      method: "POST",
      body: { name: "", allowed_origin: "" }
    });
    expect(response.status).toBe(400);
    expect(payload.error).toBe("missing_fields");
  });

  test("create site defaults owner email when blank", async () => {
    const origin = buildUniqueOrigin("owner-default");
    const { response, payload } = await adminRequest({
      path: "/api/sites",
      method: "POST",
      body: { name: "Owner Default", allowed_origin: origin, owner_email: "" }
    });
    expect(response.status).toBe(200);
    expect(payload.allowed_origin).toBe(origin);
    expect(String(payload.owner_email).toLowerCase()).toBe(config.adminEmail.toLowerCase());
  });

  test("create site rejects invalid widget side", async () => {
    const { response, payload } = await adminRequest({
      path: "/api/sites",
      method: "POST",
      body: {
        name: "Widget Side",
        allowed_origin: buildUniqueOrigin("widget-side"),
        widget_bubble_side: "top"
      }
    });
    expect(response.status).toBe(400);
    expect(payload.error).toBe("invalid_widget_side");
  });

  test("create site rejects invalid widget offset", async () => {
    const { response, payload } = await adminRequest({
      path: "/api/sites",
      method: "POST",
      body: {
        name: "Widget Offset",
        allowed_origin: buildUniqueOrigin("widget-offset"),
        widget_bubble_bottom_offset: 9999
      }
    });
    expect(response.status).toBe(400);
    expect(payload.error).toBe("invalid_widget_offset");
  });

  test("create site rejects disabling both widget feedback inputs", async () => {
    const { response, payload } = await adminRequest({
      path: "/api/sites",
      method: "POST",
      body: {
        name: "Widget Visibility Invalid",
        allowed_origin: buildUniqueOrigin("widget-visibility-invalid"),
        widget_show_message_input: false,
        widget_show_sentiment_buttons: false
      }
    });
    expect(response.status).toBe(400);
    expect(payload.error).toBe("invalid_widget_feedback_visibility");
  });

  test("create site accepts disabling widget message input when sentiment stays enabled", async () => {
    const { response, payload } = await adminRequest({
      path: "/api/sites",
      method: "POST",
      body: {
        name: "Widget Visibility Valid",
        allowed_origin: buildUniqueOrigin("widget-visibility-valid"),
        widget_show_message_input: false,
        widget_show_sentiment_buttons: true
      }
    });
    expect(response.status).toBe(200);
    expect(payload.widget_show_message_input).toBe(false);
    expect(payload.widget_show_sentiment_buttons).toBe(true);
  });

  test("create site rejects duplicate origin", async () => {
    const duplicateOrigin = buildUniqueOrigin("duplicate");
    await adminRequest({
      path: "/api/sites",
      method: "POST",
      body: { name: "Duplicate A", allowed_origin: duplicateOrigin }
    });
    const { response, payload } = await adminRequest({
      path: "/api/sites",
      method: "POST",
      body: { name: "Duplicate B", allowed_origin: duplicateOrigin }
    });
    expect(response.status).toBe(409);
    expect(payload.error).toBe("site_exists");
  });

  test("create site succeeds", async () => {
    const origin = buildUniqueOrigin("create-success");
    const { response, payload } = await adminRequest({
      path: "/api/sites",
      method: "POST",
      body: { name: "Created Site", allowed_origin: origin }
    });
    expect(response.status).toBe(200);
    expect(payload.allowed_origin).toBe(origin);
    expect(payload.name).toBe("Created Site");
  });

  test("update site rejects invalid json", async () => {
    const site = await createAdminSite("Update Invalid");
    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}`,
      method: "PATCH",
      rawBody: "{",
      contentType: "application/json"
    });
    expect(response.status).toBe(400);
  });

  test("update site rejects no changes", async () => {
    const site = await createAdminSite("Update Empty");
    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}`,
      method: "PATCH",
      body: {}
    });
    expect(response.status).toBe(400);
    expect(payload.error).toBe("nothing_to_update");
  });

  test("update site rejects blank name", async () => {
    const site = await createAdminSite("Update Blank");
    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}`,
      method: "PATCH",
      body: { name: "" }
    });
    expect(response.status).toBe(400);
    expect(payload.error).toBe("missing_fields");
  });

  test("update site rejects invalid owner", async () => {
    const site = await createAdminSite("Update Owner");
    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}`,
      method: "PATCH",
      body: { owner_email: "" }
    });
    expect(response.status).toBe(400);
    expect(payload.error).toBe("invalid_owner");
  });

  test("update site rejects invalid widget side", async () => {
    const site = await createAdminSite("Update Widget Side");
    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}`,
      method: "PATCH",
      body: { widget_bubble_side: "top" }
    });
    expect(response.status).toBe(400);
    expect(payload.error).toBe("invalid_widget_side");
  });

  test("update site rejects invalid widget offset", async () => {
    const site = await createAdminSite("Update Widget Offset");
    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}`,
      method: "PATCH",
      body: { widget_bubble_bottom_offset: -5 }
    });
    expect(response.status).toBe(400);
    expect(payload.error).toBe("invalid_widget_offset");
  });

  test("update site rejects disabling both widget feedback inputs", async () => {
    const site = await createAdminSite("Update Widget Visibility Invalid");
    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}`,
      method: "PATCH",
      body: {
        widget_show_message_input: false,
        widget_show_sentiment_buttons: false
      }
    });
    expect(response.status).toBe(400);
    expect(payload.error).toBe("invalid_widget_feedback_visibility");
  });

  test("update site accepts toggling widget feedback visibility", async () => {
    const site = await createAdminSite("Update Widget Visibility");
    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}`,
      method: "PATCH",
      body: {
        widget_show_message_input: true,
        widget_show_sentiment_buttons: false
      }
    });
    expect(response.status).toBe(200);
    expect(payload.widget_show_message_input).toBe(true);
    expect(payload.widget_show_sentiment_buttons).toBe(false);
  });

  test("update site rejects conflicting origin", async () => {
    const firstSite = await createAdminSite("Update Conflict A");
    const secondSite = await createAdminSite("Update Conflict B");
    const { response, payload } = await adminRequest({
      path: `/api/sites/${firstSite.id}`,
      method: "PATCH",
      body: { allowed_origin: secondSite.allowed_origin }
    });
    expect(response.status).toBe(409);
    expect(payload.error).toBe("site_exists");
  });

  test("update site succeeds", async () => {
    const site = await createAdminSite("Update Success");
    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}`,
      method: "PATCH",
      body: { name: "Updated Name" }
    });
    expect(response.status).toBe(200);
    expect(payload.name).toBe("Updated Name");
  });

  test("delete site rejects unauthorized user", async () => {
    const site = await createAdminSite("Delete Unauthorized");
    const { response, payload } = await nonAdminRequest({
      path: `/api/sites/${site.id}`,
      method: "DELETE"
    });
    expect(response.status).toBe(403);
    expect(payload.error).toBe("not_authorized");
  });

  test("delete site succeeds", async () => {
    const site = await createAdminSite("Delete Success");
    const { response } = await adminRequest({
      path: `/api/sites/${site.id}`,
      method: "DELETE"
    });
    expect(response.status).toBe(204);
  });
});

test.describe("admin api messages and subscribers", () => {
  test("lists feedback messages", async () => {
    const site = await createAdminSite("Messages");
    await apiRequest({
      baseURL: config.baseURL,
      path: "/public/feedback",
      method: "POST",
      origin: site.allowed_origin,
      clientIP: "10.1.1.1",
      body: {
        site_id: site.id,
        contact: "contact@example.com",
        message: "Feedback message",
        sentiment: "happy"
      }
    });
    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}/messages`,
      method: "GET"
    });
    expect(response.status).toBe(200);
    const messages = Array.isArray(payload.messages) ? payload.messages : [];
    expect(messages.some((message) => message.message === "Feedback message")).toBe(true);
    expect(messages.some((message) => message.sentiment === "happy")).toBe(true);
  });

  test("rejects messages for unauthorized user", async () => {
    const site = await createAdminSite("Messages Unauthorized");
    const { response, payload } = await nonAdminRequest({
      path: `/api/sites/${site.id}/messages`,
      method: "GET"
    });
    expect(response.status).toBe(403);
    expect(payload.error).toBe("not_authorized");
  });

  test("lists subscribers", async () => {
    const site = await createAdminSite("Subscribers List");
    const email = buildUniqueEmail("subscriber-list");
    await apiRequest({
      baseURL: config.baseURL,
      path: "/public/subscriptions",
      method: "POST",
      origin: site.allowed_origin,
      clientIP: "10.2.2.1",
      body: {
        site_id: site.id,
        email,
        name: "Subscriber",
        source_url: ""
      }
    });
    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}/subscribers`,
      method: "GET"
    });
    expect(response.status).toBe(200);
    const emails = Array.isArray(payload.subscribers) ? payload.subscribers.map((entry) => entry.email) : [];
    expect(emails).toContain(email);
  });

  test("updates subscriber status", async () => {
    const site = await createAdminSite("Subscribers Update");
    const email = buildUniqueEmail("subscriber-update");
    const { payload: created } = await apiRequest({
      baseURL: config.baseURL,
      path: "/public/subscriptions",
      method: "POST",
      origin: site.allowed_origin,
      clientIP: "10.2.2.2",
      body: {
        site_id: site.id,
        email,
        name: "Subscriber",
        source_url: ""
      }
    });
    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}/subscribers/${created.subscriber_id}`,
      method: "PATCH",
      body: { status: "unsubscribed" }
    });
    expect(response.status).toBe(200);
    expect(payload.status).toBe("ok");
  });

  test("deletes subscriber", async () => {
    const site = await createAdminSite("Subscribers Delete");
    const email = buildUniqueEmail("subscriber-delete");
    const { payload: created } = await apiRequest({
      baseURL: config.baseURL,
      path: "/public/subscriptions",
      method: "POST",
      origin: site.allowed_origin,
      clientIP: "10.2.2.3",
      body: {
        site_id: site.id,
        email,
        name: "Subscriber",
        source_url: ""
      }
    });
    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}/subscribers/${created.subscriber_id}`,
      method: "DELETE"
    });
    expect(response.status).toBe(200);
    expect(payload.status).toBe("ok");
  });

  test("exports subscribers as csv", async () => {
    const site = await createAdminSite("Subscribers Export");
    const email = buildUniqueEmail("subscriber-export");
    await apiRequest({
      baseURL: config.baseURL,
      path: "/public/subscriptions",
      method: "POST",
      origin: site.allowed_origin,
      clientIP: "10.2.2.4",
      body: {
        site_id: site.id,
        email,
        name: "Subscriber",
        source_url: ""
      }
    });
    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}/subscribers/export`,
      method: "GET"
    });
    expect(response.status).toBe(200);
    expect(response.headers.get("content-type") || "").toContain("text/csv");
    expect(String(payload)).toContain(email);
  });

  test("rejects subscribers for unauthorized user", async () => {
    const site = await createAdminSite("Subscribers Unauthorized");
    const { response, payload } = await nonAdminRequest({
      path: `/api/sites/${site.id}/subscribers`,
      method: "GET"
    });
    expect(response.status).toBe(403);
    expect(payload.error).toBe("not_authorized");
  });
});

test.describe("admin api visit stats", () => {
  test("returns visit stats", async () => {
    const site = await createAdminSite("Visit Stats", { allowedOrigin: baseOrigin });
    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(site.id)}&url=${encodeURIComponent(`${baseOrigin}/visit`)}`,
      method: "GET",
      origin: baseOrigin
    });
    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}/visits/stats`,
      method: "GET"
    });
    expect(response.status).toBe(200);
    expect(payload.visit_count).toBeGreaterThanOrEqual(1);
  });

  test("rejects visit stats for unauthorized user", async () => {
    const site = await createAdminSite("Visit Stats Unauthorized");
    const { response, payload } = await nonAdminRequest({
      path: `/api/sites/${site.id}/visits/stats`,
      method: "GET"
    });
    expect(response.status).toBe(403);
    expect(payload.error).toBe("not_authorized");
  });

  test("returns visit trend", async () => {
    const site = await createAdminSite("Visit Trend");
    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(site.id)}&url=${encodeURIComponent(`${site.allowed_origin}/visit-a`)}&visitor_id=11111111-1111-1111-1111-111111111111`,
      method: "GET",
      headers: { Origin: site.allowed_origin },
    });
    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(site.id)}&url=${encodeURIComponent(`${site.allowed_origin}/visit-b`)}&visitor_id=22222222-2222-2222-2222-222222222222`,
      method: "GET",
      headers: { Origin: site.allowed_origin },
    });

    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}/visits/trend`,
      method: "GET"
    });
    expect(response.status).toBe(200);
    expect(payload.days).toBe(7);
    expect(Array.isArray(payload.trend)).toBe(true);
    expect(payload.trend).toHaveLength(7);

    const todayUTC = new Date().toISOString().slice(0, 10);
    const todayPoint = payload.trend.find((entry) => entry.date === todayUTC);
    expect(todayPoint).toBeTruthy();
    expect(todayPoint.page_views).toBeGreaterThanOrEqual(2);
    expect(todayPoint.unique_visitors).toBeGreaterThanOrEqual(2);
  });

  test("returns visit attribution breakdown", async () => {
    const site = await createAdminSite("Visit Attribution");

    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(site.id)}&url=${encodeURIComponent(`${site.allowed_origin}/pricing?utm_source=google&utm_medium=cpc&utm_campaign=spring`)}`,
      method: "GET",
      headers: { Origin: site.allowed_origin },
    });
    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(site.id)}&url=${encodeURIComponent(`${site.allowed_origin}/signup?utm_source=google&utm_medium=cpc&utm_campaign=spring`)}`,
      method: "GET",
      headers: { Origin: site.allowed_origin },
    });
    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(site.id)}&url=${encodeURIComponent(`${site.allowed_origin}/blog`)}&referrer=${encodeURIComponent("https://news.ycombinator.com/item?id=1")}`,
      method: "GET",
      headers: { Origin: site.allowed_origin },
    });
    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(site.id)}&url=${encodeURIComponent(`${site.allowed_origin}/crawl?utm_source=bot&utm_medium=automation&utm_campaign=spider`)}`,
      method: "GET",
      headers: {
        Origin: site.allowed_origin,
        "User-Agent": "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"
      },
    });

    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}/visits/attribution`,
      method: "GET"
    });
    expect(response.status).toBe(200);
    expect(payload.limit).toBe(10);

    const sourceCounts = Object.fromEntries((payload.sources || []).map((entry) => [entry.value, entry.visit_count]));
    expect(sourceCounts.google).toBe(2);
    expect(sourceCounts["news.ycombinator.com"]).toBe(1);
    expect(sourceCounts.bot).toBeUndefined();

    const mediumCounts = Object.fromEntries((payload.mediums || []).map((entry) => [entry.value, entry.visit_count]));
    expect(mediumCounts.cpc).toBe(2);
    expect(mediumCounts.referral).toBe(1);
    expect(mediumCounts.automation).toBeUndefined();

    const campaignCounts = Object.fromEntries((payload.campaigns || []).map((entry) => [entry.value, entry.visit_count]));
    expect(campaignCounts.spring).toBe(2);
    expect(campaignCounts.none).toBe(1);
    expect(campaignCounts.spider).toBeUndefined();
  });

  test("returns visit engagement metrics", async () => {
    const site = await createAdminSite("Visit Engagement");

    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(site.id)}&url=${encodeURIComponent(`${site.allowed_origin}/first`)}&visitor_id=11111111-1111-1111-1111-111111111111`,
      method: "GET",
      headers: { Origin: site.allowed_origin },
    });
    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(site.id)}&url=${encodeURIComponent(`${site.allowed_origin}/second`)}&visitor_id=22222222-2222-2222-2222-222222222222`,
      method: "GET",
      headers: { Origin: site.allowed_origin },
    });
    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(site.id)}&url=${encodeURIComponent(`${site.allowed_origin}/third`)}&visitor_id=22222222-2222-2222-2222-222222222222`,
      method: "GET",
      headers: { Origin: site.allowed_origin },
    });
    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(site.id)}&url=${encodeURIComponent(`${site.allowed_origin}/bot`)}&visitor_id=33333333-3333-3333-3333-333333333333`,
      method: "GET",
      headers: {
        Origin: site.allowed_origin,
        "User-Agent": "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"
      },
    });

    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}/visits/engagement`,
      method: "GET"
    });
    expect(response.status).toBe(200);
    expect(payload.days).toBe(30);
    expect(payload.tracked_visitor_count).toBe(2);
    expect(payload.returning_visitor_count).toBe(1);
    expect(payload.returning_visitor_rate).toBe(0.5);
    expect(payload.average_pages_per_visitor).toBe(1.5);
    expect(payload.depth_distribution.single_page).toBe(1);
    expect(payload.depth_distribution.two_to_three_pages).toBe(1);
    expect(payload.depth_distribution.four_to_seven_pages).toBe(0);
    expect(payload.depth_distribution.eight_or_more_pages).toBe(0);
  });

  test("rejects invalid engagement days", async () => {
    const site = await createAdminSite("Visit Engagement Invalid");
    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}/visits/engagement?days=0`,
      method: "GET"
    });
    expect(response.status).toBe(400);
    expect(payload.error).toBe("invalid_days");
  });

  test("rejects invalid attribution limit", async () => {
    const site = await createAdminSite("Visit Attribution Invalid");
    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}/visits/attribution?limit=0`,
      method: "GET"
    });
    expect(response.status).toBe(400);
    expect(payload.error).toBe("invalid_limit");
  });

  test("returns device breakdown from viewport and resolution", async () => {
    const site = await createAdminSite("Device Breakdown");

    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(site.id)}&url=${encodeURIComponent(`${site.allowed_origin}/mobile`)}&viewport=375x667&screen_resolution=750x1334`,
      method: "GET",
      headers: { Origin: site.allowed_origin },
    });
    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(site.id)}&url=${encodeURIComponent(`${site.allowed_origin}/tablet`)}&viewport=800x600&screen_resolution=1024x768`,
      method: "GET",
      headers: { Origin: site.allowed_origin },
    });
    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(site.id)}&url=${encodeURIComponent(`${site.allowed_origin}/desktop`)}&viewport=1440x900&screen_resolution=1920x1080`,
      method: "GET",
      headers: { Origin: site.allowed_origin },
    });
    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(site.id)}&url=${encodeURIComponent(`${site.allowed_origin}/bot`)}&viewport=1920x1080&screen_resolution=1920x1080`,
      method: "GET",
      headers: {
        Origin: site.allowed_origin,
        "User-Agent": "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"
      },
    });

    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}/visits/devices`,
      method: "GET"
    });
    expect(response.status).toBe(200);
    expect(payload.limit).toBe(10);

    const deviceCounts = Object.fromEntries((payload.device_types || []).map((entry) => [entry.device_type, entry.visit_count]));
    expect(deviceCounts.mobile).toBe(1);
    expect(deviceCounts.tablet).toBe(1);
    expect(deviceCounts.desktop).toBe(1);

    expect(Array.isArray(payload.top_resolutions)).toBe(true);
    expect(Array.isArray(payload.top_viewports)).toBe(true);
  });

  test("returns timezone distribution", async () => {
    const site = await createAdminSite("Timezone Distribution");

    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(site.id)}&url=${encodeURIComponent(`${site.allowed_origin}/page1`)}&timezone=${encodeURIComponent("America/New_York")}`,
      method: "GET",
      headers: { Origin: site.allowed_origin },
    });
    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(site.id)}&url=${encodeURIComponent(`${site.allowed_origin}/page2`)}&timezone=${encodeURIComponent("America/New_York")}`,
      method: "GET",
      headers: { Origin: site.allowed_origin },
    });
    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(site.id)}&url=${encodeURIComponent(`${site.allowed_origin}/page3`)}&timezone=${encodeURIComponent("Europe/London")}`,
      method: "GET",
      headers: { Origin: site.allowed_origin },
    });
    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(site.id)}&url=${encodeURIComponent(`${site.allowed_origin}/bot`)}&timezone=${encodeURIComponent("Asia/Tokyo")}`,
      method: "GET",
      headers: {
        Origin: site.allowed_origin,
        "User-Agent": "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"
      },
    });

    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}/visits/timezones`,
      method: "GET"
    });
    expect(response.status).toBe(200);
    expect(payload.limit).toBe(10);

    const tzCounts = Object.fromEntries((payload.timezones || []).map((entry) => [entry.timezone, entry.visit_count]));
    expect(tzCounts["America/New_York"]).toBe(2);
    expect(tzCounts["Europe/London"]).toBe(1);
    expect(tzCounts["Asia/Tokyo"]).toBeUndefined();
  });

  test("rejects invalid device breakdown limit", async () => {
    const site = await createAdminSite("Device Breakdown Invalid");
    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}/visits/devices?limit=0`,
      method: "GET"
    });
    expect(response.status).toBe(400);
    expect(payload.error).toBe("invalid_limit");
  });

  test("rejects invalid timezone distribution limit", async () => {
    const site = await createAdminSite("Timezone Distribution Invalid");
    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}/visits/timezones?limit=0`,
      method: "GET"
    });
    expect(response.status).toBe(400);
    expect(payload.error).toBe("invalid_limit");
  });

  test("returns portfolio traffic report for sites owned by the current user", async () => {
    const nonAdminCookie = buildNonAdminCookie();
    const ownedSite = await createTestSite(config, nonAdminCookie, {
      name: buildUniqueName("Portfolio Owned"),
      allowedOrigin: buildUniqueOrigin("portfolio-owned"),
      ownerEmail: nonAdminUser.email
    });
    const secondOwnedSite = await createTestSite(config, nonAdminCookie, {
      name: buildUniqueName("Portfolio Second"),
      allowedOrigin: buildUniqueOrigin("portfolio-second"),
      ownerEmail: nonAdminUser.email
    });
    const foreignSite = await createAdminSite("Portfolio Foreign");

    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(ownedSite.id)}&url=${encodeURIComponent(`${ownedSite.allowed_origin}/owned-a`)}&visitor_id=11111111-1111-1111-1111-111111111111`,
      method: "GET",
      headers: { Origin: ownedSite.allowed_origin }
    });
    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(secondOwnedSite.id)}&url=${encodeURIComponent(`${secondOwnedSite.allowed_origin}/owned-b`)}&visitor_id=22222222-2222-2222-2222-222222222222`,
      method: "GET",
      headers: { Origin: secondOwnedSite.allowed_origin }
    });
    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(foreignSite.id)}&url=${encodeURIComponent(`${foreignSite.allowed_origin}/foreign`)}&visitor_id=33333333-3333-3333-3333-333333333333`,
      method: "GET",
      headers: { Origin: foreignSite.allowed_origin }
    });

    const { response, payload } = await nonAdminRequest({
      path: "/api/reports/traffic/portfolio",
      method: "GET"
    });
    expect(response.status).toBe(200);
    expect(payload.scope).toBe("owned");
    expect(payload.site_count).toBe(2);
    expect(payload.visit_count).toBe(2);
    expect(payload.unique_visitor_count).toBe(2);
    expect(payload.trend).toHaveLength(30);
    const siteNames = (payload.sites || []).map((entry) => entry.site_name);
    expect(siteNames).toContain(ownedSite.name);
    expect(siteNames).toContain(secondOwnedSite.name);
    expect(siteNames).not.toContain(foreignSite.name);
    const topPagePaths = (payload.top_pages || []).map((entry) => entry.path);
    expect(topPagePaths).toContain("/owned-a");
    expect(topPagePaths).toContain("/owned-b");
    expect(topPagePaths).not.toContain("/foreign");
  });

  test("portfolio traffic report schedule can be configured", async () => {
    const detectedTimezone = "UTC";
    const { response: defaultResponse, payload: defaultPayload } = await nonAdminRequest({
      path: "/api/reports/traffic/portfolio/schedule",
      method: "GET"
    });
    expect(defaultResponse.status).toBe(200);
    expect(defaultPayload.frequency).toBe("weekly");
    expect(defaultPayload.recipient_email).toBe(nonAdminUser.email);
    expect(defaultPayload.persisted).toBe(false);

    const { response, payload } = await nonAdminRequest({
      path: "/api/reports/traffic/portfolio/schedule",
      method: "PUT",
      body: {
        enabled: true,
        frequency: "monthly",
        recipient_email: "ignored@example.com",
        timezone: detectedTimezone,
        send_hour: 13,
        send_minute: 15,
        weekday: 1,
        month_day: 14
      }
    });
    expect(response.status).toBe(200);
    expect(payload.enabled).toBe(true);
    expect(payload.frequency).toBe("monthly");
    expect(payload.recipient_email).toBe(nonAdminUser.email);
    expect(payload.timezone).toBe(detectedTimezone);
    expect(payload.send_hour).toBe(13);
    expect(payload.send_minute).toBe(15);
    expect(payload.month_day).toBe(14);
    expect(payload.next_send_at).toBeGreaterThan(0);
    expect(payload.persisted).toBe(true);
  });

  test("portfolio traffic report definitions scope data and schedules", async () => {
    const nonAdminCookie = buildNonAdminCookie();
    const ownedSite = await createTestSite(config, nonAdminCookie, {
      name: buildUniqueName("Scoped Portfolio One"),
      allowedOrigin: buildUniqueOrigin("scoped-portfolio-one"),
      ownerEmail: nonAdminUser.email
    });
    const secondOwnedSite = await createTestSite(config, nonAdminCookie, {
      name: buildUniqueName("Scoped Portfolio Two"),
      allowedOrigin: buildUniqueOrigin("scoped-portfolio-two"),
      ownerEmail: nonAdminUser.email
    });
    const foreignSite = await createAdminSite("Scoped Portfolio Foreign");

    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(ownedSite.id)}&url=${encodeURIComponent(`${ownedSite.allowed_origin}/scoped-one`)}&visitor_id=44444444-4444-4444-4444-444444444444`,
      method: "GET",
      headers: { Origin: ownedSite.allowed_origin }
    });
    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(secondOwnedSite.id)}&url=${encodeURIComponent(`${secondOwnedSite.allowed_origin}/scoped-two`)}&visitor_id=55555555-5555-5555-5555-555555555555`,
      method: "GET",
      headers: { Origin: secondOwnedSite.allowed_origin }
    });
    await apiRequest({
      baseURL: config.baseURL,
      path: `/public/visits?site_id=${encodeURIComponent(foreignSite.id)}&url=${encodeURIComponent(`${foreignSite.allowed_origin}/scoped-foreign`)}&visitor_id=66666666-6666-6666-6666-666666666666`,
      method: "GET",
      headers: { Origin: foreignSite.allowed_origin }
    });

    const { response: listResponse, payload: listPayload } = await nonAdminRequest({
      path: "/api/reports/traffic/portfolio/reports",
      method: "GET"
    });
    expect(listResponse.status).toBe(200);
    expect((listPayload.reports || [])[0]).toMatchObject({
      id: "all-sites-traffic",
      is_default: true
    });
    const availableSiteIds = (listPayload.available_sites || []).map((entry) => entry.site_id);
    expect(availableSiteIds).toContain(ownedSite.id);
    expect(availableSiteIds).toContain(secondOwnedSite.id);
    expect(availableSiteIds).not.toContain(foreignSite.id);

    const { response: createResponse, payload: createPayload } = await nonAdminRequest({
      path: "/api/reports/traffic/portfolio/reports",
      method: "POST",
      body: {
        name: "Executive traffic",
        site_ids: [ownedSite.id]
      }
    });
    expect(createResponse.status).toBe(201);
    expect(createPayload.name).toBe("Executive traffic");
    expect(createPayload.site_ids).toEqual([ownedSite.id]);

    const customReportPath = `/api/reports/traffic/portfolio?report_id=${encodeURIComponent(createPayload.id)}`;
    const { response: scopedResponse, payload: scopedPayload } = await nonAdminRequest({
      path: customReportPath,
      method: "GET"
    });
    expect(scopedResponse.status).toBe(200);
    expect(scopedPayload.report_id).toBe(createPayload.id);
    expect(scopedPayload.report_name).toBe("Executive traffic");
    expect(scopedPayload.site_count).toBe(1);
    expect((scopedPayload.sites || []).map((entry) => entry.site_name)).toEqual([ownedSite.name]);
    expect((scopedPayload.top_pages || []).map((entry) => entry.path)).toContain("/scoped-one");
    expect((scopedPayload.top_pages || []).map((entry) => entry.path)).not.toContain("/scoped-two");
    expect((scopedPayload.top_pages || []).map((entry) => entry.path)).not.toContain("/scoped-foreign");

    const { response: invalidUpdateResponse, payload: invalidUpdatePayload } = await nonAdminRequest({
      path: `/api/reports/traffic/portfolio/reports/${encodeURIComponent(createPayload.id)}`,
      method: "PUT",
      body: {
        name: "Executive traffic",
        site_ids: [ownedSite.id, foreignSite.id]
      }
    });
    expect(invalidUpdateResponse.status).toBe(400);
    expect(invalidUpdatePayload.error).toBe("invalid_portfolio_traffic_report");

    const { response: updateResponse, payload: updatePayload } = await nonAdminRequest({
      path: `/api/reports/traffic/portfolio/reports/${encodeURIComponent(createPayload.id)}`,
      method: "PUT",
      body: {
        name: "Executive and product traffic",
        site_ids: [ownedSite.id, secondOwnedSite.id]
      }
    });
    expect(updateResponse.status).toBe(200);
    expect(updatePayload.name).toBe("Executive and product traffic");
    expect(updatePayload.site_ids).toEqual([ownedSite.id, secondOwnedSite.id]);

    const { response: updatedScopedResponse, payload: updatedScopedPayload } = await nonAdminRequest({
      path: customReportPath,
      method: "GET"
    });
    expect(updatedScopedResponse.status).toBe(200);
    expect(updatedScopedPayload.site_count).toBe(2);
    expect((updatedScopedPayload.top_pages || []).map((entry) => entry.path)).toContain("/scoped-one");
    expect((updatedScopedPayload.top_pages || []).map((entry) => entry.path)).toContain("/scoped-two");

    const defaultScheduleBody = {
      enabled: true,
      frequency: "monthly",
      recipient_email: "ignored@example.com",
      timezone: "UTC",
      send_hour: 8,
      send_minute: 30,
      weekday: 1,
      month_day: 7
    };
    const customScheduleBody = {
      enabled: true,
      frequency: "weekly",
      recipient_email: "ignored@example.com",
      timezone: "UTC",
      send_hour: 10,
      send_minute: 45,
      weekday: 3,
      month_day: 1
    };
    const { response: defaultScheduleResponse } = await nonAdminRequest({
      path: "/api/reports/traffic/portfolio/schedule",
      method: "PUT",
      body: defaultScheduleBody
    });
    expect(defaultScheduleResponse.status).toBe(200);
    const { response: customScheduleResponse } = await nonAdminRequest({
      path: `/api/reports/traffic/portfolio/schedule?report_id=${encodeURIComponent(createPayload.id)}`,
      method: "PUT",
      body: customScheduleBody
    });
    expect(customScheduleResponse.status).toBe(200);

    const { payload: loadedDefaultSchedule } = await nonAdminRequest({
      path: "/api/reports/traffic/portfolio/schedule",
      method: "GET"
    });
    const { payload: loadedCustomSchedule } = await nonAdminRequest({
      path: `/api/reports/traffic/portfolio/schedule?report_id=${encodeURIComponent(createPayload.id)}`,
      method: "GET"
    });
    expect(loadedDefaultSchedule.report_id).toBe("all-sites-traffic");
    expect(loadedDefaultSchedule.frequency).toBe("monthly");
    expect(loadedDefaultSchedule.send_hour).toBe(8);
    expect(loadedCustomSchedule.report_id).toBe(createPayload.id);
    expect(loadedCustomSchedule.frequency).toBe("weekly");
    expect(loadedCustomSchedule.send_hour).toBe(10);
  });
});
