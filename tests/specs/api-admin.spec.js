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
const adminCookie = buildSessionCookie(config, adminUser);
const baseOrigin = config.baseOrigin || new URL(config.baseURL).origin;

const nonAdminUser = buildAdminUser(config, {
  email: buildUniqueEmail("user"),
  displayName: "Regular User"
});
const nonAdminCookie = buildSessionCookie(config, nonAdminUser);

async function adminRequest(options) {
  return apiRequest({
    baseURL: config.baseURL,
    cookie: adminCookie,
    ...options
  });
}

async function nonAdminRequest(options) {
  return apiRequest({
    baseURL: config.baseURL,
    cookie: nonAdminCookie,
    ...options
  });
}

async function createAdminSite(label, overrides) {
  return createTestSite(config, adminCookie, {
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
      path: "/api/feedback",
      method: "POST",
      origin: site.allowed_origin,
      clientIP: "10.1.1.1",
      body: {
        site_id: site.id,
        contact: "contact@example.com",
        message: "Feedback message"
      }
    });
    const { response, payload } = await adminRequest({
      path: `/api/sites/${site.id}/messages`,
      method: "GET"
    });
    expect(response.status).toBe(200);
    const messages = Array.isArray(payload.messages) ? payload.messages : [];
    expect(messages.some((message) => message.message === "Feedback message")).toBe(true);
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
      path: "/api/subscriptions",
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
      path: "/api/subscriptions",
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
      path: "/api/subscriptions",
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
      path: "/api/subscriptions",
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
      path: `/api/visits?site_id=${encodeURIComponent(site.id)}&url=${encodeURIComponent(`${baseOrigin}/visit`)}`,
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
});
