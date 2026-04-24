// @ts-check
import { test, expect } from "@playwright/test";
import { execFile } from "node:child_process";
import { promisify } from "node:util";
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
  captureBrowserSentryError,
  captureSentryError,
  getSentryIssueDetail,
  listSentryIssues,
  rotateSentryToken,
  updateSentryIssueStatus
} from "../helpers/api.js";

const config = resolveTestConfig();
const execFileAsync = promisify(execFile);
const adminUser = buildAdminUser(config);
const nonAdminUser = buildAdminUser(config, {
  email: buildUniqueEmail("la-sentry-user"),
  displayName: "LA Sentry User"
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

async function runPythonClientCapture(site, tokenPayload) {
  const script = `
import json
import os
from la_sentry import Client, LASentryConfig

client = Client(LASentryConfig(
    endpoint=os.environ["LOOPAWARE_LA_SENTRY_ENDPOINT"],
    site_id=os.environ["LOOPAWARE_LA_SENTRY_SITE_ID"],
    ingest_token=os.environ["LOOPAWARE_LA_SENTRY_TOKEN"],
    environment="integration-python",
    release="2026.04.24-python",
    default_tags={"service": "python-client"},
))

try:
    raise RuntimeError("python client capture failed")
except RuntimeError as error:
    response = client.capture_error(error, {
        "event_id": os.environ["LOOPAWARE_LA_SENTRY_EVENT_ID"],
        "tags": {"language": "python"},
        "request": {"method": "GET", "url": "https://python-client.example/sync"},
        "extra": {"worker": "alpha"},
    })
    print(json.dumps(response, sort_keys=True))
`;
  const { stdout } = await execFileAsync("python3", ["-c", script], {
    cwd: config.repositoryRoot,
    env: {
      ...process.env,
      PYTHONPATH: `${config.repositoryRoot}/clients/python`,
      LOOPAWARE_LA_SENTRY_ENDPOINT: tokenPayload.ingest_endpoint,
      LOOPAWARE_LA_SENTRY_SITE_ID: site.id,
      LOOPAWARE_LA_SENTRY_TOKEN: tokenPayload.ingest_token,
      LOOPAWARE_LA_SENTRY_EVENT_ID: `python-${Date.now()}`
    }
  });
  return JSON.parse(stdout.trim());
}

test.describe("LA Sentry developer monitoring api", () => {
  test("requires a configured ingest token", async () => {
    const site = await createSentrySite("LA Sentry Token Missing");
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
    const site = await createSentrySite("LA Sentry Token Invalid");
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
    const site = await createSentrySite("LA Sentry Capture");
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
    const site = await createSentrySite("LA Sentry Unauthorized");
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

  test("captures browser errors from allowed origins without an ingest token", async () => {
    const site = await createSentrySite("LA Sentry Browser API");
    const event = sentryEvent(site, {
      eventId: "browser-origin-event",
      message: "browser route failed",
      exceptionType: "TypeError"
    });
    event.environment = "browser";
    event.release = "browser-release";
    event.request = {
      method: "GET",
      url: `${site.allowed_origin}/checkout?token=hidden`,
      referrer: `${site.allowed_origin}/pricing?secret=hidden`,
      user_agent: "Playwright Browser"
    };
    event.tags = { service: "browser-harness" };

    const capture = await captureBrowserSentryError(config, event, site.allowed_origin);
    expect(capture.status).toBe("ok");

    const issuesPayload = await listSentryIssues(config, adminCookie(), site.id);
    expect(issuesPayload.issues).toHaveLength(1);
    expect(issuesPayload.issues[0].platform).toBe("javascript");
    expect(issuesPayload.issues[0].environment).toBe("browser");

    const detail = await getSentryIssueDetail(config, adminCookie(), site.id, issuesPayload.issues[0].id);
    expect(detail.latest_occurrence.request.url).toBe(`${site.allowed_origin}/checkout`);
    expect(detail.latest_occurrence.request.referrer).toBe(`${site.allowed_origin}/pricing`);
    expect(detail.latest_occurrence.request.user_agent).toBe("Playwright Browser");
  });

  test("rejects browser errors from unknown origins", async () => {
    const site = await createSentrySite("LA Sentry Browser Forbidden");
    const { response, payload } = await apiRequest({
      baseURL: config.baseURL,
      path: "/sentry/browser-errors",
      method: "POST",
      origin: buildUniqueOrigin("sentry-browser-forbidden"),
      body: sentryEvent(site, { eventId: "browser-forbidden-event" })
    });

    expect(response.status).toBe(403);
    expect(payload.error).toBe("origin_forbidden");
  });

  test("python client captures errors through protected ingest", async () => {
    const site = await createSentrySite("LA Sentry Python Client");
    const tokenPayload = await rotateSentryToken(config, adminCookie(), site.id);
    const capture = await runPythonClientCapture(site, tokenPayload);

    expect(capture.status).toBe("ok");
    const issuesPayload = await listSentryIssues(config, adminCookie(), site.id);
    expect(issuesPayload.issues).toHaveLength(1);
    expect(issuesPayload.issues[0].platform).toBe("python");
    expect(issuesPayload.issues[0].environment).toBe("integration-python");

    const detail = await getSentryIssueDetail(config, adminCookie(), site.id, issuesPayload.issues[0].id);
    expect(detail.latest_occurrence.message).toBe("python client capture failed");
    expect(detail.latest_occurrence.tags.language).toBe("python");
    expect(detail.latest_occurrence.tags.service).toBe("python-client");
    expect(detail.latest_occurrence.request.url).toBe("https://python-client.example/sync");
  });
});
