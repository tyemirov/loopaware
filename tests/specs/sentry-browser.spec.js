// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildSessionCookie } from '../helpers/auth.js';
import { buildAdminUser, ensureSiteForOrigin } from '../helpers/fixtures.js';
import { getSentryIssueDetail, listSentryIssues } from '../helpers/api.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);

let site;

function buildAdminCookie() {
  return buildSessionCookie(config, adminUser);
}

test.beforeAll(async () => {
  site = await ensureSiteForOrigin(config, buildAdminCookie(), {
    allowedOrigin: config.baseOrigin,
    ownerEmail: config.adminEmail
  });
});

test('browser sentry harness captures explicit errors', async ({ page }) => {
  const eventMessage = `browser harness capture ${Date.now()}`;
  const search = new URLSearchParams({
    site_id: site.id,
    environment: 'browser-e2e',
    release: 'browser-e2e-release'
  });

  await page.goto(`/sentry-integration/?${search.toString()}`, { waitUntil: 'domcontentloaded' });
  await expect(page.locator('#sentry-integration-status')).toContainText('Loaded');

  const capture = await page.evaluate(async (message) => {
    const sentryClient = /** @type {any} */ (window).LoopAwareSentry;
    if (!sentryClient || typeof sentryClient.captureError !== 'function') {
      throw new Error('missing sentry browser client');
    }
    return sentryClient.captureError(new Error(message), {
      tags: { scenario: 'browser-e2e' },
      extra: { component: 'checkout' }
    });
  }, eventMessage);
  expect(capture.status).toBe('ok');

  const issuesPayload = await listSentryIssues(config, buildAdminCookie(), site.id);
  const issue = issuesPayload.issues.find((candidate) => candidate.title.includes(eventMessage));
  if (!issue) {
    throw new Error('browser sentry issue was not created');
  }
  expect(issue.platform).toBe('javascript');
  expect(issue.environment).toBe('browser-e2e');

  const detail = await getSentryIssueDetail(config, buildAdminCookie(), site.id, issue.id);
  expect(detail.latest_occurrence.message).toBe(eventMessage);
  expect(detail.latest_occurrence.tags.scenario).toBe('browser-e2e');
  expect(detail.latest_occurrence.request.url).toBe(`${config.baseOrigin}/sentry-integration/`);
});
