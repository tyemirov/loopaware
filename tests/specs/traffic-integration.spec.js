// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildSessionCookie } from '../helpers/auth.js';
import { buildAdminUser, ensureSiteForOrigin } from '../helpers/fixtures.js';
import { fetchVisitStats } from '../helpers/api.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);
const cookie = buildSessionCookie(config, adminUser);
let site;

function buildCrossOriginAPIOrigin(baseURL) {
  const parsedBaseURL = new URL(baseURL);
  const alternateHostname = parsedBaseURL.hostname === 'localhost' ? '127.0.0.1' : 'localhost';
  return `${parsedBaseURL.protocol}//${alternateHostname}${parsedBaseURL.port ? `:${parsedBaseURL.port}` : ''}`;
}

async function openTrafficPage(page, siteId, options) {
  const resolvedOptions = options || {};
  const params = new URLSearchParams({ site_id: siteId });
  if (resolvedOptions.apiOrigin) {
    params.set('api_origin', resolvedOptions.apiOrigin);
  }
  await page.goto(`/traffic-integration/?${params.toString()}`, { waitUntil: 'domcontentloaded' });
}

test.beforeAll(async () => {
  site = await ensureSiteForOrigin(config, cookie, {
    allowedOrigin: config.baseOrigin,
    ownerEmail: config.adminEmail
  });
});

test('traffic integration requires site_id', async ({ page }) => {
  await page.goto('/traffic-integration/', { waitUntil: 'domcontentloaded' });
  await expect(page.locator('#traffic-integration-status')).toContainText('Missing site_id query parameter');
});

test('traffic integration stores visitor id', async ({ page }) => {
  await openTrafficPage(page, site.id);
  await page.waitForFunction(() => localStorage.getItem('loopaware_visitor_id'));
  const visitorId = await page.evaluate(() => localStorage.getItem('loopaware_visitor_id'));
  expect(visitorId).toBeTruthy();
});

test('traffic integration records a visit on load', async ({ page }) => {
  await openTrafficPage(page, site.id);
  await expect.poll(async () => {
    const stats = await fetchVisitStats(config, cookie, site.id);
    return stats.visit_count;
  }).toBeGreaterThan(0);
});

test('traffic integration increments visit count on reload', async ({ page }) => {
  await openTrafficPage(page, site.id);
  await expect.poll(async () => {
    const stats = await fetchVisitStats(config, cookie, site.id);
    return stats.visit_count;
  }).toBeGreaterThan(0);
  const initialStats = await fetchVisitStats(config, cookie, site.id);
  const initialCount = initialStats.visit_count;
  await page.reload({ waitUntil: 'domcontentloaded' });
  await expect.poll(async () => {
    const stats = await fetchVisitStats(config, cookie, site.id);
    return stats.visit_count;
  }).toBeGreaterThan(initialCount);
});

test('traffic integration keeps unique visitor count stable on reload', async ({ page }) => {
  const baselineStats = await fetchVisitStats(config, cookie, site.id);
  const baselineVisitCount = baselineStats.visit_count;
  await openTrafficPage(page, site.id);
  await expect.poll(async () => {
    const stats = await fetchVisitStats(config, cookie, site.id);
    return stats.visit_count;
  }).toBeGreaterThan(baselineVisitCount);
  const initialStats = await fetchVisitStats(config, cookie, site.id);
  const initialUnique = initialStats.unique_visitor_count;
  const initialVisitCount = initialStats.visit_count;
  await page.reload({ waitUntil: 'domcontentloaded' });
  await expect.poll(async () => {
    const stats = await fetchVisitStats(config, cookie, site.id);
    return stats.visit_count;
  }).toBeGreaterThan(initialVisitCount);
  const reloadStats = await fetchVisitStats(config, cookie, site.id);
  expect(reloadStats.unique_visitor_count).toBe(initialUnique);
});

test('traffic integration reports top pages', async ({ page }) => {
  await openTrafficPage(page, site.id);
  await expect.poll(async () => {
    const stats = await fetchVisitStats(config, cookie, site.id);
    return Array.isArray(stats.top_pages) ? stats.top_pages.length : 0;
  }).toBeGreaterThan(0);
  const stats = await fetchVisitStats(config, cookie, site.id);
  const topPages = Array.isArray(stats.top_pages) ? stats.top_pages : [];
  const paths = topPages.map((entry) => entry.path);
  expect(paths).toContain('/traffic-integration');
});

test('traffic integration keeps visitor id across reloads', async ({ page }) => {
  await openTrafficPage(page, site.id);
  await page.waitForFunction(() => localStorage.getItem('loopaware_visitor_id'));
  const firstId = await page.evaluate(() => localStorage.getItem('loopaware_visitor_id'));
  await page.reload({ waitUntil: 'domcontentloaded' });
  await page.waitForFunction(() => localStorage.getItem('loopaware_visitor_id'));
  const secondId = await page.evaluate(() => localStorage.getItem('loopaware_visitor_id'));
  expect(firstId).toBe(secondId);
});

test('traffic integration uses GET pixel request for cross-origin api_origin', async ({ page }) => {
  const crossOriginAPIOrigin = buildCrossOriginAPIOrigin(config.baseURL);
  const visitRequestPromise = page.waitForRequest((request) => request.url().includes('/public/visits'));
  await openTrafficPage(page, site.id, { apiOrigin: crossOriginAPIOrigin });
  const visitRequest = await visitRequestPromise;
  expect(visitRequest.method()).toBe('GET');
});
