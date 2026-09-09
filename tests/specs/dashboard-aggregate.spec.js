// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildSessionCookie } from '../helpers/auth.js';
import { buildAdminUser, buildUniqueName, buildUniqueOrigin, createTestSite, openDashboard, selectSite } from '../helpers/fixtures.js';
import { apiRequest } from '../helpers/api.js';

test('aggregate traffic displays request counts and unavailable visitor metrics', async ({ page }) => {
  const config = resolveTestConfig();
  const admin = buildAdminUser(config);
  const site = await createTestSite(config, buildSessionCookie(config, admin), {
    name: buildUniqueName('Aggregate'), allowedOrigin: buildUniqueOrigin('aggregate'), ownerEmail: config.adminEmail, trafficProfile: "aggregate"
  });
  const counted = await apiRequest({ baseURL: config.baseURL, path: `/public/sites/${site.id}/visit-counts`, method: 'POST', origin: site.allowed_origin, body: {} });
  expect(counted.response.status).toBe(204);
  await openDashboard(page, config, admin);
  await selectSite(page, site.id);
  await expect(page.locator('#visit-count')).toHaveText('1 request');
  await expect(page.locator('#unique-visitor-count')).toHaveText('Unique visitors unavailable');
  await expect(page.locator('#top-pages-chart')).toContainText('Unavailable for daily counts');
  await expect(page.locator('#device-types-chart')).toContainText('Unavailable for daily counts');
});

test('count client sends only one empty request without cookies or referrer', async ({ page }) => {
  const config = resolveTestConfig();
  const admin = buildAdminUser(config);
  const site = await createTestSite(config, buildSessionCookie(config, admin), {
    name: buildUniqueName('Count client'), allowedOrigin: buildUniqueOrigin('count-client').replace('.example.com', '.localhost'), ownerEmail: config.adminEmail, trafficProfile: "aggregate"
  });
  await page.context().grantPermissions(["local-network-access"], { origin: site.allowed_origin });
  const calls = [];
  page.on('request', (request) => { if (request.url().includes('/visit-counts')) calls.push(request); });
  await page.context().addCookies([{ name: 'private-session', value: 'private-marker', url: config.baseURL }]);
  await page.route(site.allowed_origin + '/', (route) => route.fulfill({ contentType: 'text/html', body: `<script src="${config.baseURL}/count.js?site_id=${site.id}"></script>` }));
  await page.goto(site.allowed_origin);
  await expect.poll(() => calls.length).toBe(1);
  expect(calls[0].method()).toBe('POST');
  expect(calls[0].postData()).toBe('{}');
  const headers = await calls[0].allHeaders();
  expect(headers.cookie).toBeUndefined();
  expect(headers.referer).toBeUndefined();
  await expect(page.locator('script')).toHaveAttribute('data-count-state', 'ready');
});

test('site creator selects a fixed aggregate profile', async ({ page }) => {
  const config = resolveTestConfig();
  const admin = buildAdminUser(config);
  await openDashboard(page, config, admin);
  await page.locator('#new-site-button').click();
  await page.getByLabel('Traffic collection').selectOption('aggregate');
  const name = buildUniqueName('Native counts');
  await page.locator('#edit-site-name').fill(name);
  await page.locator('#edit-site-origin').fill(buildUniqueOrigin('native-counts'));
  await page.locator('#edit-site-owner').fill(admin.email);
  await expect(page.locator('#sites-list [data-site-id].active')).toContainText(name);
  await expect(page.getByLabel('Traffic collection')).toHaveValue('aggregate');
  await expect(page.getByLabel('Traffic collection')).toBeDisabled();
  await expect(page.locator('#traffic-widget-snippet')).toHaveValue(/count\.js/);
});
