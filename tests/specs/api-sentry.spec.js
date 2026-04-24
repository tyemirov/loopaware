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
import {
  apiRequest,
  captureSentryError,
  getSentryIssueDetail,
  listSentryIssues,
  rotateSentryToken,
  updateSentryIssueStatus
} from "../helpers/api.js";

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);
const nonAdminUser = buildAdminUser(config, {
  email: buildUniqueEmail("sentry-user"),
  displayName: "Sentry User"
});

function adminCookie() {
  return buildSessionCookie(config, adminUser);
}

function nonAdminCookie() {
  return buildSessionCookie(config, nonAdminUser);
}

async function createSentrySite(label) {
  return createTestSite(config, adminCookie(), {
    name: buildUniqueName(label),
    allowedOrigin: buildUniqueOrigin(label),
    ownerEmail: config.adminEmail
  });
}

function sentryEvent(site, overrides) {
  const resolvedOverrides = overrides || {};
  return {
    site_id: site.id,
    event_id: resolvedOverrides.eventId || `event-${Date.now()}-${Math.random().toString(16).slice(2)}`,
    timestamp: new Date().toISOString(),
    platform: "go",
    environment: "integration",
    release: "2026.04.24",
    level: resolvedOverrides.level || "error",
    message: resolvedOverrides.message || "database connection refused",
    exception_type: resolvedOverrides.exceptionType || "DatabaseError",
    stacktrace: [
      {
        filename: "/app/internal/server.go",
        function: "github.com/MarkoPoloResearchLab/loopaware/internal/server.handle",
        module: "server",
        line: 42,
        column: 7,
        in_app: true
      }
    ],
    request: {
      method: "GET",
      url: "https://poodlescanner.example/sync"
    },
    user_hash: "user-hash-1",
    tags: {
      service: "poodlescanner"
    },
    extra: {
      shard: "alpha"
    }
  };
}

test.describe("sentry developer monitoring api", () => {
  test("requires a configured ingest token", async () => {
    const site = await createSentrySite("Sentry Token Missing");
    const { response, payload } = await apiRequest({
      baseURL: config.baseURL,
      path: "/sentry/errors",
      method: "POST",
      body: sentryEvent(site)
    });

    expect(response.status).toBe(403);
    expect(payload.error).toBe("sentry_token_not_configured");
  });

  test("rejects invalid ingest credentials", async () => {
    const site = await createSentrySite("Sentry Token Invalid");
    await rotateSentryToken(config, adminCookie(), site.id);

    const { response, payload } = await apiRequest({
      baseURL: config.baseURL,
      path: "/sentry/errors",
      method: "POST",
      headers: {
        Authorization: "Bearer not-the-token"
      },
      body: sentryEvent(site)
    });

    expect(response.status).toBe(403);
    expect(payload.error).toBe("invalid_sentry_token");
  });

  test("captures, groups, deduplicates, and updates issue status", async () => {
    const site = await createSentrySite("Sentry Capture");
    const tokenPayload = await rotateSentryToken(config, adminCookie(), site.id);

    const event = sentryEvent(site, { eventId: "event-grouped-primary" });
    const capture = await captureSentryError(config, tokenPayload.ingest_token, event);
    expect(capture.status).toBe("ok");
    expect(capture.duplicate).toBe(false);

    const duplicateCapture = await captureSentryError(config, tokenPayload.ingest_token, event);
    expect(duplicateCapture.duplicate).toBe(true);

    const issuesPayload = await listSentryIssues(config, adminCookie(), site.id);
    expect(issuesPayload.site_id).toBe(site.id);
    expect(issuesPayload.issues).toHaveLength(1);
    const issue = issuesPayload.issues[0];
    expect(issue.title).toContain("DatabaseError");
    expect(issue.status).toBe("unresolved");
    expect(issue.occurrence_count).toBe(1);

    const detail = await getSentryIssueDetail(config, adminCookie(), site.id, issue.id);
    expect(detail.latest_occurrence.event_id).toBe(event.event_id);
    expect(detail.latest_occurrence.stacktrace[0].filename).toBe("/app/internal/server.go");
    expect(detail.latest_occurrence.tags.service).toBe("poodlescanner");

    const resolved = await updateSentryIssueStatus(config, adminCookie(), site.id, issue.id, "resolved");
    expect(resolved.status).toBe("resolved");

    await captureSentryError(config, tokenPayload.ingest_token, sentryEvent(site, { eventId: "event-grouped-regression" }));
    const regressedPayload = await listSentryIssues(config, adminCookie(), site.id);
    expect(regressedPayload.issues[0].status).toBe("unresolved");
    expect(regressedPayload.issues[0].occurrence_count).toBe(2);
  });

  test("keeps issue access scoped to site managers", async () => {
    const site = await createSentrySite("Sentry Unauthorized");
    const tokenPayload = await rotateSentryToken(config, adminCookie(), site.id);
    await captureSentryError(config, tokenPayload.ingest_token, sentryEvent(site));

    const { response, payload } = await apiRequest({
      baseURL: config.baseURL,
      path: `/api/sites/${site.id}/sentry/issues`,
      method: "GET",
      cookie: nonAdminCookie()
    });

    expect(response.status).toBe(403);
    expect(payload.error).toBe("not_authorized");
  });
});
