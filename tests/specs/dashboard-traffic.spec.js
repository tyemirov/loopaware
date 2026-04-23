// @ts-check
import { test, expect } from '@playwright/test';
import * as crypto from 'node:crypto';
import { resolveTestConfig } from '../helpers/config.js';
import { buildSessionCookie } from '../helpers/auth.js';
import { buildAdminUser, buildUniqueName, buildUniqueOrigin, createTestSite, openDashboard, selectSite } from '../helpers/fixtures.js';
import { collectVisit, fetchTrafficReportSchedule, fetchVisitStats } from '../helpers/api.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);

function buildAdminCookie() {
  return buildSessionCookie(config, adminUser);
}

function buildVisitorId() {
  return crypto.randomUUID();
}

async function createTrafficSite() {
  return createTestSite(config, buildAdminCookie(), {
    name: buildUniqueName('Traffic Site'),
    allowedOrigin: buildUniqueOrigin('traffic'),
    ownerEmail: config.adminEmail
  });
}

test('traffic counts show zero for new sites', async ({ page }) => {
  const site = await createTrafficSite();
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#visit-count')).toHaveClass(/d-none/);
  await expect(page.locator('#unique-visitor-count')).toHaveClass(/d-none/);
  await expect(page.locator('#top-pages-table-body')).toContainText('No visits yet');
});

test('traffic counts update for distinct visitors', async ({ page }) => {
  const site = await createTrafficSite();
  await collectVisit(config, site, { url: `${site.allowed_origin}/alpha`, visitorId: buildVisitorId() });
  await collectVisit(config, site, { url: `${site.allowed_origin}/beta`, visitorId: buildVisitorId() });
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#visit-count')).toHaveText('2 visits');
  await expect(page.locator('#unique-visitor-count')).toHaveText('2 unique');
});

test('unique visitor count does not double count repeat visitor', async ({ page }) => {
  const site = await createTrafficSite();
  const visitorId = buildVisitorId();
  await collectVisit(config, site, { url: `${site.allowed_origin}/alpha`, visitorId });
  await collectVisit(config, site, { url: `${site.allowed_origin}/alpha`, visitorId });
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#visit-count')).toHaveText('2 visits');
  await expect(page.locator('#unique-visitor-count')).toHaveText('1 unique');
});

test('top pages list includes visited paths', async ({ page }) => {
  const site = await createTrafficSite();
  await collectVisit(config, site, { url: `${site.allowed_origin}/alpha`, visitorId: buildVisitorId() });
  await collectVisit(config, site, { url: `${site.allowed_origin}/beta`, visitorId: buildVisitorId() });
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#top-pages-table-body')).toContainText('/alpha');
  await expect(page.locator('#top-pages-table-body')).toContainText('/beta');
});

test('top pages are sorted by count', async ({ page }) => {
  const site = await createTrafficSite();
  await collectVisit(config, site, { url: `${site.allowed_origin}/alpha`, visitorId: buildVisitorId() });
  await collectVisit(config, site, { url: `${site.allowed_origin}/alpha`, visitorId: buildVisitorId() });
  await collectVisit(config, site, { url: `${site.allowed_origin}/beta`, visitorId: buildVisitorId() });
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  const firstRowPath = page.locator('#top-pages-table-body tr').first().locator('td').first();
  await expect(firstRowPath).toHaveText('/alpha');
});

test('traffic status stays hidden on success', async ({ page }) => {
  const site = await createTrafficSite();
  await collectVisit(config, site, { url: `${site.allowed_origin}/alpha`, visitorId: buildVisitorId() });
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#traffic-status')).toHaveClass(/d-none/);
});

test('traffic report schedule can be configured from dashboard', async ({ page }) => {
  const site = await createTrafficSite();
  const cookie = buildAdminCookie();
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await page.locator('#dashboard-section-tab-traffic').click();
  await expect(page.locator('[data-dashboard-card="traffic-report"]')).toBeVisible();
  await expect(page.locator('#traffic-report-recipient')).toHaveValue(config.adminEmail);

  await page.locator('#traffic-report-enabled').check();
  await page.locator('label[for="traffic-report-frequency-weekly"]').click();
  await expect(page.locator('#traffic-report-weekday-container')).toBeVisible();
  await page.locator('#traffic-report-send-time').fill('14:30');
  await page.locator('#traffic-report-timezone').fill('America/New_York');
  await page.locator('#traffic-report-weekday').selectOption('5');
  await page.locator('#traffic-report-recipient').fill('reports@example.com');
  await page.locator('#traffic-report-save-button').click();

  await expect(page.locator('#traffic-report-status')).toHaveText('Traffic report schedule saved.');
  await expect(page.locator('#traffic-report-next-send')).not.toHaveText('Not scheduled.');

  const schedule = await fetchTrafficReportSchedule(config, cookie, site.id);
  expect(schedule.enabled).toBe(true);
  expect(schedule.frequency).toBe('weekly');
  expect(schedule.recipient_email).toBe('reports@example.com');
  expect(schedule.timezone).toBe('America/New_York');
  expect(schedule.send_hour).toBe(14);
  expect(schedule.send_minute).toBe(30);
  expect(schedule.weekday).toBe(5);
  expect(schedule.next_send_at).toBeGreaterThan(0);
});

test('device and timezone fetch failures show traffic error state', async ({ page }) => {
  const site = await createTrafficSite();
  await page.route(`**/api/sites/${site.id}/visits/devices`, async (route) => {
    await route.fulfill({
      status: 500,
      contentType: 'application/json',
      body: JSON.stringify({ error: 'query_failed' })
    });
  });
  await page.route(`**/api/sites/${site.id}/visits/timezones`, async (route) => {
    await route.fulfill({
      status: 500,
      contentType: 'application/json',
      body: JSON.stringify({ error: 'query_failed' })
    });
  });
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#traffic-status')).toHaveText('Failed to load data.');
  await expect(page.locator('#device-types-table-body')).toContainText('Failed to load data.');
  await expect(page.locator('#timezones-table-body')).toContainText('Failed to load data.');
});

test('traffic stats refresh after reload', async ({ page }) => {
  const site = await createTrafficSite();
  await collectVisit(config, site, { url: `${site.allowed_origin}/alpha`, visitorId: buildVisitorId() });
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#visit-count')).toHaveText('1 visits');
  await collectVisit(config, site, { url: `${site.allowed_origin}/beta`, visitorId: buildVisitorId() });
  await page.reload({ waitUntil: 'domcontentloaded' });
  await page.locator('#user-name').waitFor();
  await selectSite(page, site.id);
  await expect(page.locator('#visit-count')).toHaveText('2 visits');
});

test('dashboard counts match visit stats API', async ({ page }) => {
  const site = await createTrafficSite();
  await collectVisit(config, site, { url: `${site.allowed_origin}/alpha`, visitorId: buildVisitorId() });
  await collectVisit(config, site, { url: `${site.allowed_origin}/alpha`, visitorId: buildVisitorId() });
  await collectVisit(config, site, { url: `${site.allowed_origin}/beta`, visitorId: buildVisitorId() });
  const stats = await fetchVisitStats(config, buildAdminCookie(), site.id);
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#visit-count')).toHaveText(`${stats.visit_count} visits`);
  await expect(page.locator('#unique-visitor-count')).toHaveText(`${stats.unique_visitor_count} unique`);
});

test('device types table shows breakdown by viewport width', async ({ page }) => {
  const site = await createTrafficSite();
  await collectVisit(config, site, {
    url: `${site.allowed_origin}/mobile`,
    visitorId: buildVisitorId(),
    viewport: '375x667',
    screenResolution: '750x1334'
  });
  await collectVisit(config, site, {
    url: `${site.allowed_origin}/desktop`,
    visitorId: buildVisitorId(),
    viewport: '1440x900',
    screenResolution: '1920x1080'
  });
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#device-types-table-body')).toContainText('mobile');
  await expect(page.locator('#device-types-table-body')).toContainText('desktop');
});

test('device types table shows placeholder for new sites', async ({ page }) => {
  const site = await createTrafficSite();
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#device-types-table-body')).toContainText('No device data yet');
});

test('timezones table shows distribution', async ({ page }) => {
  const site = await createTrafficSite();
  await collectVisit(config, site, {
    url: `${site.allowed_origin}/page1`,
    visitorId: buildVisitorId(),
    timezone: 'America/New_York'
  });
  await collectVisit(config, site, {
    url: `${site.allowed_origin}/page2`,
    visitorId: buildVisitorId(),
    timezone: 'Europe/London'
  });
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#timezones-table-body')).toContainText('America/New_York');
  await expect(page.locator('#timezones-table-body')).toContainText('Europe/London');
});

test('timezones table shows placeholder for new sites', async ({ page }) => {
  const site = await createTrafficSite();
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#timezones-table-body')).toContainText('No timezone data yet');
});
