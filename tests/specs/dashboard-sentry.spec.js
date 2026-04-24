// @ts-check
import { test, expect } from "@playwright/test";
import { resolveTestConfig } from "../helpers/config.js";
import { buildSessionCookie } from "../helpers/auth.js";
import {
  buildAdminUser,
  buildUniqueName,
  buildUniqueOrigin,
  createTestSite,
  openDashboard,
  selectSite
} from "../helpers/fixtures.js";
import { captureSentryError, rotateSentryToken } from "../helpers/api.js";

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);

function adminCookie() {
  return buildSessionCookie(config, adminUser);
}

async function createDashboardSentrySite() {
  return createTestSite(config, adminCookie(), {
    name: buildUniqueName("Dashboard Sentry"),
    allowedOrigin: buildUniqueOrigin("dashboard-sentry"),
    ownerEmail: config.adminEmail
  });
}

function dashboardSentryEvent(site) {
  return {
    site_id: site.id,
    event_id: `dashboard-sentry-${Date.now()}-${Math.random().toString(16).slice(2)}`,
    timestamp: new Date().toISOString(),
    platform: "go",
    environment: "production",
    release: "dashboard-test",
    level: "error",
    message: "checkout worker crashed",
    exception_type: "CheckoutWorkerError",
    stacktrace: [
      {
        filename: "/srv/poodlescanner/checkout.go",
        function: "checkout.run",
        module: "checkout",
        line: 77,
        column: 3,
        in_app: true
      }
    ],
    request: {
      method: "POST",
      url: "https://poodlescanner.example/checkout"
    },
    user_hash: "user-hash-dashboard",
    tags: {
      queue: "checkout"
    },
    extra: {
      attempt: 2
    }
  };
}

test("dashboard sentry tab shows client config, issues, details, and status actions", async ({ page }) => {
  const site = await createDashboardSentrySite();
  const tokenPayload = await rotateSentryToken(config, adminCookie(), site.id);
  await captureSentryError(config, tokenPayload.ingest_token, dashboardSentryEvent(site));

  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await page.locator("#dashboard-section-tab-sentry").click();

  await expect(page.locator('[data-widget-card="sentry"]')).toBeVisible();
  await expect(page.locator("#sentry-ingest-endpoint")).toHaveValue(/\/sentry\/errors$/);
  await expect(page.locator("#sentry-ingest-token")).toHaveValue(/Token configured/);
  await expect(page.locator("#rotate-sentry-token-button")).toBeEnabled();

  const issueRow = page.locator('#sentry-issues-table-body tr:has-text("CheckoutWorkerError")').first();
  await expect(issueRow).toBeVisible();
  await expect(issueRow).toContainText("production");
  await expect(issueRow).toContainText("unresolved");

  await issueRow.click();
  await expect(page.locator("#sentry-issue-detail")).toBeVisible();
  await expect(page.locator("#sentry-issue-detail-title")).toContainText("CheckoutWorkerError");
  await expect(page.locator("#sentry-issue-stacktrace")).toContainText("/srv/poodlescanner/checkout.go:77");
  await expect(page.locator("#sentry-issue-tags")).toContainText("checkout");

  await page.locator("#sentry-issue-resolve-button").click();
  await expect(issueRow).toContainText("resolved");

  await page.locator("#sentry-issue-reopen-button").click();
  await expect(issueRow).toContainText("unresolved");
});

test("dashboard can rotate and reveal a new sentry token", async ({ page }) => {
  const site = await createDashboardSentrySite();

  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await page.locator("#dashboard-section-tab-sentry").click();

  await expect(page.locator("#sentry-ingest-token")).toHaveValue(/No token configured/);
  await page.locator("#rotate-sentry-token-button").click();
  await expect(page.locator("#sentry-config-status")).toContainText("Sentry token rotated.");
  await expect(page.locator("#sentry-ingest-token")).toHaveValue(/^las_/);
});
