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
import { buildSubscriptionConfirmationToken } from "../helpers/subscriptionToken.js";

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);
const adminCookie = buildSessionCookie(config, adminUser);

let clientIPCounter = 1;
function nextClientIP() {
  const suffix = clientIPCounter % 250;
  clientIPCounter += 1;
  return `10.0.0.${suffix || 1}`;
}

async function createPublicSite(label) {
  return createTestSite(config, adminCookie, {
    name: buildUniqueName(label),
    allowedOrigin: buildUniqueOrigin(label),
    ownerEmail: config.adminEmail
  });
}

async function postFeedbackRequest(siteId, contact, message, originOverride, clientIP) {
  return apiRequest({
    baseURL: config.baseURL,
    path: "/api/feedback",
    method: "POST",
    origin: originOverride,
    clientIP: clientIP || nextClientIP(),
    body: {
      site_id: siteId,
      contact,
      message
    }
  });
}

async function postSubscriptionRequest(siteId, email, originOverride, clientIP) {
  return apiRequest({
    baseURL: config.baseURL,
    path: "/api/subscriptions",
    method: "POST",
    origin: originOverride,
    clientIP: clientIP || nextClientIP(),
    body: {
      site_id: siteId,
      email,
      name: "",
      source_url: ""
    }
  });
}

async function postSubscriptionMutation(path, siteId, email, originOverride, clientIP) {
  return apiRequest({
    baseURL: config.baseURL,
    path,
    method: "POST",
    origin: originOverride,
    clientIP: clientIP || nextClientIP(),
    body: {
      site_id: siteId,
      email
    }
  });
}

test.describe("public feedback api", () => {
  let site;

  test.beforeAll(async () => {
    site = await createPublicSite("Feedback API");
  });

  test("rejects missing site id", async () => {
    const { response, payload } = await postFeedbackRequest("", "person@example.com", "Hello", site.allowed_origin);
    expect(response.status).toBe(400);
    expect(payload.error).toBe("missing_fields");
  });

  test("rejects missing contact", async () => {
    const { response, payload } = await postFeedbackRequest(site.id, "", "Hello", site.allowed_origin);
    expect(response.status).toBe(400);
    expect(payload.error).toBe("missing_fields");
  });

  test("rejects missing message", async () => {
    const { response, payload } = await postFeedbackRequest(site.id, "person@example.com", "", site.allowed_origin);
    expect(response.status).toBe(400);
    expect(payload.error).toBe("missing_fields");
  });

  test("rejects invalid json", async () => {
    const { response, payload } = await apiRequest({
      baseURL: config.baseURL,
      path: "/api/feedback",
      method: "POST",
      origin: site.allowed_origin,
      clientIP: nextClientIP(),
      rawBody: "{",
      contentType: "application/json"
    });
    expect(response.status).toBe(400);
  });

  test("rejects unknown site", async () => {
    const { response, payload } = await postFeedbackRequest("missing-site", "person@example.com", "Hello", site.allowed_origin);
    expect(response.status).toBe(404);
    expect(payload.error).toBe("unknown_site");
  });

  test("rejects forbidden origin", async () => {
    const forbiddenOrigin = buildUniqueOrigin("feedback-forbidden");
    const { response, payload } = await postFeedbackRequest(site.id, "person@example.com", "Hello", forbiddenOrigin);
    expect(response.status).toBe(403);
    expect(payload.error).toBe("origin_forbidden");
  });

  test("accepts valid feedback", async () => {
    const { response, payload } = await postFeedbackRequest(site.id, "person@example.com", "Hello", site.allowed_origin);
    expect(response.status).toBe(200);
    expect(payload.status).toBe("ok");
  });

  test("rate limits repeated feedback requests", async () => {
    const clientIP = nextClientIP();
    let lastResult;
    for (let attemptIndex = 0; attemptIndex < 7; attemptIndex += 1) {
      lastResult = await postFeedbackRequest(site.id, "person@example.com", "", site.allowed_origin, clientIP);
    }
    expect(lastResult.response.status).toBe(429);
    expect(lastResult.payload.error).toBe("rate_limited");
  });
});

test.describe("public subscription api", () => {
  let site;

  test.beforeAll(async () => {
    site = await createPublicSite("Subscription API");
  });

  test("rejects missing site id", async () => {
    const { response, payload } = await postSubscriptionRequest("", "user@example.com", site.allowed_origin);
    expect(response.status).toBe(400);
    expect(payload.error).toBe("missing_fields");
  });

  test("rejects missing email", async () => {
    const { response, payload } = await postSubscriptionRequest(site.id, "", site.allowed_origin);
    expect(response.status).toBe(400);
    expect(payload.error).toBe("missing_fields");
  });

  test("rejects invalid email", async () => {
    const { response, payload } = await postSubscriptionRequest(site.id, "not-an-email", site.allowed_origin);
    expect(response.status).toBe(400);
    expect(payload.error).toBe("invalid_email");
  });

  test("rejects unknown site", async () => {
    const { response, payload } = await postSubscriptionRequest("missing-site", "user@example.com", site.allowed_origin);
    expect(response.status).toBe(404);
    expect(payload.error).toBe("unknown_site");
  });

  test("rejects forbidden origin", async () => {
    const forbiddenOrigin = buildUniqueOrigin("subscribe-forbidden");
    const { response, payload } = await postSubscriptionRequest(site.id, "user@example.com", forbiddenOrigin);
    expect(response.status).toBe(403);
    expect(payload.error).toBe("origin_forbidden");
  });

  test("accepts valid subscription", async () => {
    const email = buildUniqueEmail("subscriber");
    const { response, payload } = await postSubscriptionRequest(site.id, email, site.allowed_origin);
    expect(response.status).toBe(200);
    expect(payload.status).toBe("ok");
    expect(payload.subscriber_id).toBeTruthy();
  });

  test("rejects duplicate subscription", async () => {
    const email = buildUniqueEmail("duplicate");
    await postSubscriptionRequest(site.id, email, site.allowed_origin);
    const { response, payload } = await postSubscriptionRequest(site.id, email, site.allowed_origin);
    expect(response.status).toBe(409);
    expect(payload.error).toBe("duplicate_subscription");
  });

  test("rate limits repeated subscription requests", async () => {
    const clientIP = nextClientIP();
    let lastResult;
    for (let attemptIndex = 0; attemptIndex < 7; attemptIndex += 1) {
      lastResult = await postSubscriptionRequest(site.id, "", site.allowed_origin, clientIP);
    }
    expect(lastResult.response.status).toBe(429);
    expect(lastResult.payload.error).toBe("rate_limited");
  });
});

test.describe("subscription confirmation flows", () => {
  let site;

  test.beforeAll(async () => {
    site = await createPublicSite("Confirm API");
  });

  test("confirm rejects missing fields", async () => {
    const { response, payload } = await postSubscriptionMutation("/api/subscriptions/confirm", "", "", site.allowed_origin);
    expect(response.status).toBe(400);
    expect(payload.error).toBe("missing_fields");
  });

  test("confirm rejects unknown site", async () => {
    const { response, payload } = await postSubscriptionMutation("/api/subscriptions/confirm", "missing-site", "user@example.com", site.allowed_origin);
    expect(response.status).toBe(404);
    expect(payload.error).toBe("unknown_site");
  });

  test("confirm rejects forbidden origin", async () => {
    const email = buildUniqueEmail("forbidden-confirm");
    const forbiddenOrigin = buildUniqueOrigin("confirm-forbidden");
    const { response, payload } = await postSubscriptionMutation("/api/subscriptions/confirm", site.id, email, forbiddenOrigin);
    expect(response.status).toBe(403);
    expect(payload.error).toBe("origin_forbidden");
  });

  test("confirm rejects unknown subscription", async () => {
    const { response, payload } = await postSubscriptionMutation("/api/subscriptions/confirm", site.id, buildUniqueEmail("missing"), site.allowed_origin);
    expect(response.status).toBe(404);
    expect(payload.error).toBe("unknown_subscription");
  });

  test("confirm succeeds for pending subscriber", async () => {
    const email = buildUniqueEmail("confirm");
    await postSubscriptionRequest(site.id, email, site.allowed_origin);
    const { response, payload } = await postSubscriptionMutation("/api/subscriptions/confirm", site.id, email, site.allowed_origin);
    expect(response.status).toBe(200);
    expect(payload.status).toBe("ok");
  });

  test("confirm rejects unsubscribed subscriber", async () => {
    const email = buildUniqueEmail("unsubscribed-confirm");
    await postSubscriptionRequest(site.id, email, site.allowed_origin);
    await postSubscriptionMutation("/api/subscriptions/unsubscribe", site.id, email, site.allowed_origin);
    const { response, payload } = await postSubscriptionMutation("/api/subscriptions/confirm", site.id, email, site.allowed_origin);
    expect(response.status).toBe(409);
    expect(payload.error).toBe("unsubscribed");
  });

  test("unsubscribe succeeds", async () => {
    const email = buildUniqueEmail("unsubscribe");
    await postSubscriptionRequest(site.id, email, site.allowed_origin);
    const { response, payload } = await postSubscriptionMutation("/api/subscriptions/unsubscribe", site.id, email, site.allowed_origin);
    expect(response.status).toBe(200);
    expect(payload.status).toBe("ok");
  });

  test("confirm returns ok for already confirmed", async () => {
    const email = buildUniqueEmail("already-confirmed");
    await postSubscriptionRequest(site.id, email, site.allowed_origin);
    await postSubscriptionMutation("/api/subscriptions/confirm", site.id, email, site.allowed_origin);
    const { response, payload } = await postSubscriptionMutation("/api/subscriptions/confirm", site.id, email, site.allowed_origin);
    expect(response.status).toBe(200);
    expect(payload.status).toBe("ok");
  });

  test("rate limits confirmation requests", async () => {
    const clientIP = nextClientIP();
    const email = buildUniqueEmail("rate-confirm");
    let lastResult;
    for (let attemptIndex = 0; attemptIndex < 7; attemptIndex += 1) {
      lastResult = await postSubscriptionMutation("/api/subscriptions/confirm", site.id, email, site.allowed_origin, clientIP);
    }
    expect(lastResult.response.status).toBe(429);
    expect(lastResult.payload.error).toBe("rate_limited");
  });

  test("rate limits unsubscribe requests", async () => {
    const clientIP = nextClientIP();
    const email = buildUniqueEmail("rate-unsubscribe");
    let lastResult;
    for (let attemptIndex = 0; attemptIndex < 7; attemptIndex += 1) {
      lastResult = await postSubscriptionMutation("/api/subscriptions/unsubscribe", site.id, email, site.allowed_origin, clientIP);
    }
    expect(lastResult.response.status).toBe(429);
    expect(lastResult.payload.error).toBe("rate_limited");
  });
});

test.describe("subscription link endpoints", () => {
  let site;

  test.beforeAll(async () => {
    site = await createPublicSite("Link API");
  });

  test("confirm link rejects missing token", async () => {
    const { response, payload } = await apiRequest({
      baseURL: config.baseURL,
      path: "/api/subscriptions/confirm-link",
      method: "GET"
    });
    expect(response.status).toBe(400);
    expect(payload.message).toContain("Missing confirmation token");
  });

  test("confirm link rejects invalid token", async () => {
    const { response, payload } = await apiRequest({
      baseURL: config.baseURL,
      path: "/api/subscriptions/confirm-link?token=invalid",
      method: "GET"
    });
    expect(response.status).toBe(400);
    expect(payload.message).toContain("Invalid or expired token");
  });

  test("confirm link returns confirmation payload", async () => {
    const email = buildUniqueEmail("confirm-link");
    const { payload: createPayload } = await postSubscriptionRequest(site.id, email, site.allowed_origin);
    const token = buildSubscriptionConfirmationToken(config.subscriptionSecret, createPayload.subscriber_id, site.id, email, 60);
    const { response, payload } = await apiRequest({
      baseURL: config.baseURL,
      path: `/api/subscriptions/confirm-link?token=${encodeURIComponent(token)}`,
      method: "GET"
    });
    expect(response.status).toBe(200);
    expect(payload.heading).toContain("Subscription");
    expect(payload.open_url).toContain(site.allowed_origin);
  });

  test("confirm link reports already unsubscribed", async () => {
    const email = buildUniqueEmail("confirm-link-unsubscribed");
    const { payload: createPayload } = await postSubscriptionRequest(site.id, email, site.allowed_origin);
    await postSubscriptionMutation("/api/subscriptions/unsubscribe", site.id, email, site.allowed_origin);
    const token = buildSubscriptionConfirmationToken(config.subscriptionSecret, createPayload.subscriber_id, site.id, email, 60);
    const { response, payload } = await apiRequest({
      baseURL: config.baseURL,
      path: `/api/subscriptions/confirm-link?token=${encodeURIComponent(token)}`,
      method: "GET"
    });
    expect(response.status).toBe(409);
    expect(payload.message).toContain("unsubscribed");
  });

  test("unsubscribe link rejects missing token", async () => {
    const { response, payload } = await apiRequest({
      baseURL: config.baseURL,
      path: "/api/subscriptions/unsubscribe-link",
      method: "GET"
    });
    expect(response.status).toBe(400);
    expect(payload.message).toContain("Missing unsubscribe token");
  });

  test("unsubscribe link rejects invalid token", async () => {
    const { response, payload } = await apiRequest({
      baseURL: config.baseURL,
      path: "/api/subscriptions/unsubscribe-link?token=invalid",
      method: "GET"
    });
    expect(response.status).toBe(400);
    expect(payload.message).toContain("Invalid or expired token");
  });

  test("unsubscribe link updates subscriber", async () => {
    const email = buildUniqueEmail("unsubscribe-link");
    const { payload: createPayload } = await postSubscriptionRequest(site.id, email, site.allowed_origin);
    const token = buildSubscriptionConfirmationToken(config.subscriptionSecret, createPayload.subscriber_id, site.id, email, 60);
    const { response, payload } = await apiRequest({
      baseURL: config.baseURL,
      path: `/api/subscriptions/unsubscribe-link?token=${encodeURIComponent(token)}`,
      method: "GET"
    });
    expect(response.status).toBe(200);
    expect(payload.message).toContain("unsubscribed");
  });

  test("unsubscribe link confirms already unsubscribed", async () => {
    const email = buildUniqueEmail("unsubscribe-link-already");
    const { payload: createPayload } = await postSubscriptionRequest(site.id, email, site.allowed_origin);
    await postSubscriptionMutation("/api/subscriptions/unsubscribe", site.id, email, site.allowed_origin);
    const token = buildSubscriptionConfirmationToken(config.subscriptionSecret, createPayload.subscriber_id, site.id, email, 60);
    const { response, payload } = await apiRequest({
      baseURL: config.baseURL,
      path: `/api/subscriptions/unsubscribe-link?token=${encodeURIComponent(token)}`,
      method: "GET"
    });
    expect(response.status).toBe(200);
    expect(payload.message).toContain("already unsubscribed");
  });
});

test.describe("widget config endpoint", () => {
  let site;

  test.beforeAll(async () => {
    site = await createPublicSite("Widget Config");
  });

  test("rejects missing site id", async () => {
    const { response, payload } = await apiRequest({
      baseURL: config.baseURL,
      path: "/api/widget-config",
      method: "GET",
      origin: site.allowed_origin
    });
    expect(response.status).toBe(400);
    expect(payload.error).toBe("missing_site_id");
  });

  test("rejects unknown site", async () => {
    const { response, payload } = await apiRequest({
      baseURL: config.baseURL,
      path: "/api/widget-config?site_id=missing",
      method: "GET",
      origin: site.allowed_origin
    });
    expect(response.status).toBe(404);
    expect(payload.error).toBe("unknown_site");
  });

  test("rejects forbidden origin", async () => {
    const forbiddenOrigin = buildUniqueOrigin("widget-forbidden");
    const { response, payload } = await apiRequest({
      baseURL: config.baseURL,
      path: `/api/widget-config?site_id=${encodeURIComponent(site.id)}`,
      method: "GET",
      origin: forbiddenOrigin
    });
    expect(response.status).toBe(403);
    expect(payload.error).toBe("origin_forbidden");
  });

  test("returns widget placement defaults", async () => {
    const { response, payload } = await apiRequest({
      baseURL: config.baseURL,
      path: `/api/widget-config?site_id=${encodeURIComponent(site.id)}`,
      method: "GET",
      origin: site.allowed_origin
    });
    expect(response.status).toBe(200);
    expect(payload.site_id).toBe(site.id);
    expect(payload.widget_bubble_side).toBeTruthy();
  });

  test("returns demo widget config", async () => {
    const { response, payload } = await apiRequest({
      baseURL: config.baseURL,
      path: "/api/widget-config?site_id=__loopaware_widget_demo__",
      method: "GET",
      origin: site.allowed_origin
    });
    expect(response.status).toBe(200);
    expect(payload.site_id).toBe("__loopaware_widget_demo__");
  });
});

test.describe("visit collection endpoint", () => {
  let site;

  test.beforeAll(async () => {
    site = await createTestSite(config, adminCookie, {
      name: buildUniqueName("Visits"),
      allowedOrigin: buildUniqueOrigin("visits"),
      ownerEmail: config.adminEmail
    });
  });

  test("rejects missing site id", async () => {
    const { response } = await apiRequest({
      baseURL: config.baseURL,
      path: "/api/visits",
      method: "GET",
      origin: site.allowed_origin
    });
    expect(response.status).toBe(400);
  });

  test("rejects unknown site", async () => {
    const { response } = await apiRequest({
      baseURL: config.baseURL,
      path: "/api/visits?site_id=missing",
      method: "GET",
      origin: site.allowed_origin
    });
    expect(response.status).toBe(404);
  });

  test("rejects forbidden origin", async () => {
    const forbiddenOrigin = buildUniqueOrigin("visit-forbidden");
    const forbiddenURL = `${forbiddenOrigin}/visit`;
    const { response } = await apiRequest({
      baseURL: config.baseURL,
      path: `/api/visits?site_id=${encodeURIComponent(site.id)}&url=${encodeURIComponent(forbiddenURL)}`,
      method: "GET",
      origin: forbiddenOrigin
    });
    expect(response.status).toBe(403);
  });

  test("rejects invalid visitor id", async () => {
    const { response, payload } = await apiRequest({
      baseURL: config.baseURL,
      path: `/api/visits?site_id=${encodeURIComponent(site.id)}&url=${encodeURIComponent(`${site.allowed_origin}/visit`)}&visitor_id=bad`,
      method: "GET",
      origin: site.allowed_origin
    });
    expect(response.status).toBe(400);
    expect(String(payload)).toContain("invalid_visitor");
  });

  test("rejects invalid url", async () => {
    const { response, payload } = await apiRequest({
      baseURL: config.baseURL,
      path: `/api/visits?site_id=${encodeURIComponent(site.id)}&url=//bad-url`,
      method: "GET",
      origin: site.allowed_origin
    });
    expect(response.status).toBe(400);
    expect(String(payload)).toContain("invalid_url");
  });

  test("records visit and returns pixel", async () => {
    const { response } = await apiRequest({
      baseURL: config.baseURL,
      path: `/api/visits?site_id=${encodeURIComponent(site.id)}&url=${encodeURIComponent(`${site.allowed_origin}/visit`)}`,
      method: "GET",
      origin: site.allowed_origin
    });
    expect(response.status).toBe(200);
    expect(response.headers.get("content-type") || "").toContain("image/gif");
  });

  test("accepts referer when url is missing", async () => {
    const refererURL = `${site.allowed_origin}/visit-referer`;
    const { response } = await apiRequest({
      baseURL: config.baseURL,
      path: `/api/visits?site_id=${encodeURIComponent(site.id)}`,
      method: "GET",
      headers: {
        Origin: site.allowed_origin,
        Referer: refererURL
      }
    });
    expect(response.status).toBe(200);
    expect(response.headers.get("content-type") || "").toContain("image/gif");
  });
});
