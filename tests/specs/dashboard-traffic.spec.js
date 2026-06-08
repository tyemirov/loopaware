// @ts-check
import { test, expect } from '@playwright/test';
import * as crypto from 'node:crypto';
import { resolveTestConfig } from '../helpers/config.js';
import { buildSessionCookie } from '../helpers/auth.js';
import { buildAdminUser, buildUniqueEmail, buildUniqueName, buildUniqueOrigin, createTestSite, openDashboard, selectSite, waitForDashboardReady } from '../helpers/fixtures.js';
import { collectVisit, fetchPortfolioTrafficReport, fetchPortfolioTrafficReports, fetchPortfolioTrafficReportSchedule, fetchTrafficReportSchedule, fetchVisitStats, savePortfolioTrafficReportSchedule, saveTrafficReportSchedule } from '../helpers/api.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);

function buildAdminCookie() {
  return buildSessionCookie(config, adminUser);
}

function buildVisitorId() {
  return crypto.randomUUID();
}

/**
 * @param {number} daysBeforeToday
 * @returns {string}
 */
function buildTrendDateLabel(daysBeforeToday) {
  const date = new Date();
  date.setUTCHours(0, 0, 0, 0);
  date.setUTCDate(date.getUTCDate() - daysBeforeToday);
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    timeZone: 'UTC'
  });
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} chartSelector
 * @param {number} trendWindowDays
 * @returns {Promise<void>}
 */
async function expectTrendAxisLabels(page, chartSelector, trendWindowDays) {
  const scaleLabels = page.locator(`${chartSelector} svg text`);
  const lastPointDayOffset = trendWindowDays - 1;
  const middlePointIndex = Math.round(lastPointDayOffset / 2);
  const middlePointDayOffset = lastPointDayOffset - middlePointIndex;
  const dateLabelOffsets = Array.from(new Set([lastPointDayOffset, middlePointDayOffset, 0]));
  await expect(scaleLabels.filter({ hasText: 'Visits / visitors' }).first()).toBeVisible();
  await expect(scaleLabels.filter({ hasText: /^2$/ }).first()).toBeVisible();
  await expect(scaleLabels.filter({ hasText: /^0$/ }).first()).toBeVisible();
  for (const dayOffset of dateLabelOffsets) {
    await expect(scaleLabels.filter({ hasText: buildTrendDateLabel(dayOffset) }).first()).toBeVisible();
  }
}

async function timezoneBubbleRadius(page, timezone) {
  const radius = await page.locator(`#timezones-map circle[data-timezone="${timezone}"]`).getAttribute('r');
  if (!radius) {
    throw new Error(`missing_timezone_bubble:${timezone}`);
  }
  return Number(radius);
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
  await expect(page.locator('#top-pages-chart')).toContainText('No visits yet');
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

test('top pages row graph includes visited paths', async ({ page }) => {
  const site = await createTrafficSite();
  await collectVisit(config, site, { url: `${site.allowed_origin}/alpha`, visitorId: buildVisitorId() });
  await collectVisit(config, site, { url: `${site.allowed_origin}/beta`, visitorId: buildVisitorId() });
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await page.locator('#dashboard-section-tab-traffic').click();
  const alphaRow = page.locator('#top-pages-chart .path-row[aria-label="/alpha 1 visits"]');
  const betaRow = page.locator('#top-pages-chart .path-row[aria-label="/beta 1 visits"]');
  await expect(alphaRow).toBeVisible();
  await expect(alphaRow.locator('.path-row__icon svg')).toBeVisible();
  await expect(alphaRow.locator('.path-row__count')).toHaveText('1');
  await expect(betaRow).toBeVisible();
  await expect(betaRow.locator('.path-row__count')).toHaveText('1');
});

test('top pages are sorted by count', async ({ page }) => {
  const site = await createTrafficSite();
  await collectVisit(config, site, { url: `${site.allowed_origin}/alpha`, visitorId: buildVisitorId() });
  await collectVisit(config, site, { url: `${site.allowed_origin}/alpha`, visitorId: buildVisitorId() });
  await collectVisit(config, site, { url: `${site.allowed_origin}/beta`, visitorId: buildVisitorId() });
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await page.locator('#dashboard-section-tab-traffic').click();
  const firstRow = page.locator('#top-pages-chart .path-row').first();
  await expect(firstRow.locator('.path-row__rank')).toHaveText('1');
  await expect(firstRow.locator('.path-row__label')).toHaveText('/alpha');
  await expect(firstRow.locator('.path-row__count')).toHaveText('2');
});

test('traffic status stays hidden on success', async ({ page }) => {
  const site = await createTrafficSite();
  await collectVisit(config, site, { url: `${site.allowed_origin}/alpha`, visitorId: buildVisitorId() });
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#traffic-status')).toHaveClass(/d-none/);
});

test('traffic graphics render selected-site trends and breakdowns', async ({ page }) => {
  const site = await createTrafficSite();
  const visitorId = buildVisitorId();
  await collectVisit(config, site, {
    url: `${site.allowed_origin}/alpha?utm_source=google&utm_medium=cpc&utm_campaign=graphics`,
    visitorId,
    viewport: '375x667',
    screenResolution: '750x1334',
    timezone: 'America/New_York'
  });
  await collectVisit(config, site, {
    url: `${site.allowed_origin}/alpha?utm_source=google&utm_medium=cpc&utm_campaign=graphics`,
    visitorId,
    viewport: '1440x900',
    screenResolution: '1920x1080',
    timezone: 'America/New_York'
  });

  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await page.locator('#dashboard-section-tab-traffic').click();

  await expect(page.locator('#traffic-trend-chart svg')).toBeVisible();
  await expectTrendAxisLabels(page, '#traffic-trend-chart', 7);
  const topPageRow = page.locator('#top-pages-chart .path-row[aria-label="/alpha 2 visits"]');
  await expect(topPageRow).toBeVisible();
  await expect(topPageRow.locator('.path-row__icon svg')).toBeVisible();
  await expect(topPageRow.locator('.path-row__count')).toHaveText('2');
  await expect(page.locator('#traffic-attribution-chart')).toContainText('google');
  await expect(page.locator('#traffic-engagement-summary')).toContainText('Returning rate');
  await expect(page.locator('#device-types-chart .device-row[aria-label="mobile 1 visits"]')).toBeVisible();
  await expect(page.locator('#device-types-chart .device-row[aria-label="mobile 1 visits"] .device-row__icon svg')).toBeVisible();
  await expect(page.locator('#timezones-map svg')).toBeVisible();
  await expect(page.locator('#timezones-map circle[data-timezone="America/New_York"]')).toBeVisible();
  await expect(page.locator('#timezones-map')).toContainText('New York 2');
});

test('traffic report schedule can be configured from dashboard', async ({ page }) => {
  const site = await createTrafficSite();
  const cookie = buildAdminCookie();
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await page.locator('#dashboard-section-tab-traffic').click();
  await expect(page.locator('[data-dashboard-card="traffic-report"]')).toBeVisible();
  await expect(page.locator('[data-dashboard-card="traffic-report"] .card-header #traffic-report-enabled')).toBeVisible();
  await expect(page.locator('#traffic-report-save-button')).toHaveCount(0);
  await expect(page.locator('#traffic-report-recipient')).toHaveValue(config.adminEmail);
  await expect(page.locator('#traffic-report-recipient')).toHaveAttribute('readonly', '');
  await expect(page.locator('#traffic-report-timezone')).toHaveAttribute('readonly', '');
  const detectedTimezone = await page.evaluate(() => Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC');
  await expect(page.locator('#traffic-report-timezone')).toHaveValue(detectedTimezone);

  await page.locator('#traffic-report-enabled').check();
  await page.locator('label[for="traffic-report-frequency-weekly"]').click();
  await expect(page.locator('#traffic-report-weekday-container')).toBeVisible();
  await page.locator('#traffic-report-send-time').fill('14:30');
  await page.locator('#traffic-report-weekday').selectOption('5');

  await expect.poll(async () => {
    const schedule = await fetchTrafficReportSchedule(config, cookie, site.id);
    return {
      enabled: schedule.enabled,
      frequency: schedule.frequency,
      recipient_email: schedule.recipient_email,
      timezone: schedule.timezone,
      send_hour: schedule.send_hour,
      send_minute: schedule.send_minute,
      weekday: schedule.weekday
    };
  }, { timeout: 10000 }).toEqual({
    enabled: true,
    frequency: 'weekly',
    recipient_email: config.adminEmail,
    timezone: detectedTimezone,
    send_hour: 14,
    send_minute: 30,
    weekday: 5
  });
  await expect(page.locator('#traffic-report-status')).toHaveText('Traffic report schedule saved.');
  await expect(page.locator('#traffic-report-next-send')).not.toHaveText('Not scheduled.');

  const schedule = await fetchTrafficReportSchedule(config, cookie, site.id);
  expect(schedule.next_send_at).toBeGreaterThan(0);
});

test('traffic report schedule preserves saved timezone from dashboard edits', async ({ page }) => {
  const site = await createTrafficSite();
  const cookie = buildAdminCookie();
  const savedTimezone = 'Pacific/Honolulu';
  await saveTrafficReportSchedule(config, cookie, site.id, {
    enabled: true,
    frequency: 'daily',
    recipient_email: config.adminEmail,
    timezone: savedTimezone,
    send_hour: 9,
    send_minute: 0,
    weekday: 1,
    month_day: 1
  });

  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await page.locator('#dashboard-section-tab-traffic').click();
  await expect(page.locator('#traffic-report-timezone')).toHaveValue(savedTimezone);
  await page.locator('#traffic-report-send-time').fill('12:05');

  await expect.poll(async () => {
    const schedule = await fetchTrafficReportSchedule(config, cookie, site.id);
    return {
      timezone: schedule.timezone,
      send_hour: schedule.send_hour,
      send_minute: schedule.send_minute
    };
  }, { timeout: 10000 }).toEqual({
    timezone: savedTimezone,
    send_hour: 12,
    send_minute: 5
  });
});

test('settings entry opens all-sites traffic reporting', async ({ page }) => {
  const portfolioUser = buildAdminUser(config, {
    email: buildUniqueEmail('portfolio-dashboard'),
    displayName: 'Portfolio Dashboard'
  });
  const cookie = buildSessionCookie(config, portfolioUser);
  const firstSite = await createTestSite(config, cookie, {
    name: buildUniqueName('Portfolio Alpha Site'),
    allowedOrigin: buildUniqueOrigin('portfolio-alpha'),
    ownerEmail: portfolioUser.email
  });
  const secondSite = await createTestSite(config, cookie, {
    name: buildUniqueName('Portfolio Beta Site'),
    allowedOrigin: buildUniqueOrigin('portfolio-beta'),
    ownerEmail: portfolioUser.email
  });
  await collectVisit(config, firstSite, { url: `${firstSite.allowed_origin}/portfolio-alpha`, visitorId: buildVisitorId() });
  await collectVisit(config, secondSite, { url: `${secondSite.allowed_origin}/portfolio-beta`, visitorId: buildVisitorId() });

  await openDashboard(page, config, portfolioUser);
  await selectSite(page, firstSite.id);
  await page.evaluate(() => {
    document.dispatchEvent(new CustomEvent('mpr-user:menu-item', { detail: { action: 'account-settings' } }));
  });
  await expect(page.locator('#settings-modal')).toBeVisible();
  await page.locator('#settings-open-all-sites-traffic').click();

  await expect(page.locator('#settings-modal')).toBeHidden();
  await expect(page.locator('#site-dashboard-view')).toBeHidden();
  await expect(page.locator('#all-sites-traffic-view')).toBeVisible();
  await expect(page.locator('#sites-list')).toBeHidden();
  await expect(page.locator('#site-form')).toBeHidden();
  await expect(page.locator('#all-sites-traffic-view #site-form')).toHaveCount(0);
  await expect(page.locator('#all-sites-traffic-view h1')).toHaveText('Reports');
  await expect(page.locator('#global-traffic-reports-list')).toContainText('All sites traffic');
  await expect(page.locator('#global-traffic-report-name')).toBeHidden();
  await expect(page.locator('#global-traffic-report-readonly-name')).toBeVisible();
  await expect(page.locator('#global-traffic-report-readonly-name')).toContainText('All sites traffic');
  await expect(page.locator('#global-traffic-report-readonly-name')).toContainText('Default report');
  await expect(page.locator('#all-sites-traffic-sites-table-body')).toContainText(firstSite.name);
  await expect(page.locator('#all-sites-traffic-sites-table-body')).toContainText(secondSite.name);
  await expect(page.locator('#all-sites-traffic-view')).not.toContainText('Top pages');
  await expect(page.locator('#all-sites-traffic-trend-chart svg')).toBeVisible();
  await expectTrendAxisLabels(page, '#all-sites-traffic-trend-chart', 30);
  await expect(page.locator('#all-sites-site-count')).toHaveText('2 sites');
  await expect(page.locator('#global-traffic-report-sites-chip')).toHaveText('2 sites');
  await expect(page.locator('#global-traffic-report-sites-summary')).toHaveText('2 of 2 sites');
  await expect(page.locator(`#global-traffic-report-sites-list input[data-global-traffic-site-id="${firstSite.id}"]`)).toBeDisabled();

  await expect(page.locator('#all-sites-traffic-report-recipient')).toHaveValue(portfolioUser.email);
  await expect(page.locator('#all-sites-traffic-report-recipient')).toHaveAttribute('readonly', '');
  await page.locator('#all-sites-traffic-report-enabled').check();
  await page.locator('label[for="all-sites-traffic-report-frequency-monthly"]').click();
  await page.locator('#all-sites-traffic-report-send-time').fill('08:45');
  await page.locator('#all-sites-traffic-report-month-day').selectOption('14');

  await expect.poll(async () => {
    const schedule = await fetchPortfolioTrafficReportSchedule(config, cookie);
    return {
      enabled: schedule.enabled,
      frequency: schedule.frequency,
      recipient_email: schedule.recipient_email,
      send_hour: schedule.send_hour,
      send_minute: schedule.send_minute,
      month_day: schedule.month_day
    };
  }, { timeout: 10000 }).toEqual({
    enabled: true,
    frequency: 'monthly',
    recipient_email: portfolioUser.email,
    send_hour: 8,
    send_minute: 45,
    month_day: 14
  });

  await page.locator('#global-traffic-report-create-button').click();
  await expect(page.locator('#global-traffic-reports-list')).toContainText('Traffic report 2');
  await expect(page.locator('#global-traffic-report-readonly-name')).toBeHidden();
  await expect(page.locator('#global-traffic-report-name')).toBeEnabled();
  await expect(page.locator('#global-traffic-report-name')).toHaveValue('Traffic report 2');
  const customReportId = await page.locator('#global-traffic-reports-list .active').getAttribute('data-global-traffic-report-id');
  expect(customReportId).toBeTruthy();
  await expect(page.locator('#global-traffic-report-schedule-chip')).toHaveText('Schedule off');
  await expect(page.locator('#all-sites-traffic-report-enabled')).toBeEnabled();
  await expect(page.locator('#all-sites-traffic-report-enabled')).not.toBeChecked();
  await page.locator('#global-traffic-report-name').fill('Executive traffic');
  await page.locator('#global-traffic-report-title').click();
  await expect(page.locator('#global-traffic-report-title')).toHaveText('Executive traffic');
  await expect(page.locator('#global-traffic-reports-list')).toContainText('Executive traffic');
  await expect.poll(async () => {
    const reportDefinitions = await fetchPortfolioTrafficReports(config, cookie);
    const reports = Array.isArray(reportDefinitions.reports) ? reportDefinitions.reports : [];
    const savedReport = reports.find((report) => report.id === customReportId);
    return savedReport ? savedReport.name : '';
  }, { timeout: 10000 }).toBe('Executive traffic');
  await page.locator('#all-sites-traffic-report-enabled').check();
  await page.locator('label[for="all-sites-traffic-report-frequency-weekly"]').click();
  await page.locator('#all-sites-traffic-report-send-time').fill('10:15');
  await page.locator('#all-sites-traffic-report-weekday').selectOption('3');
  await expect.poll(async () => {
    const schedule = await fetchPortfolioTrafficReportSchedule(config, cookie, customReportId || '');
    return {
      enabled: schedule.enabled,
      frequency: schedule.frequency,
      recipient_email: schedule.recipient_email,
      send_hour: schedule.send_hour,
      send_minute: schedule.send_minute,
      weekday: schedule.weekday
    };
  }, { timeout: 10000 }).toEqual({
    enabled: true,
    frequency: 'weekly',
    recipient_email: portfolioUser.email,
    send_hour: 10,
    send_minute: 15,
    weekday: 3
  });
  await page.locator(`#global-traffic-report-sites-list input[data-global-traffic-site-id="${secondSite.id}"]`).uncheck();
  await expect(page.locator('#global-traffic-report-sites-summary')).toHaveText('1 of 2 sites');
  await expect(page.locator('#global-traffic-report-sites-chip')).toHaveText('1 site');
  await expect(page.locator('#all-sites-site-count')).toHaveText('1 site');
  await expect(page.locator('#all-sites-traffic-sites-table-body')).toContainText(firstSite.name);
  await expect(page.locator('#all-sites-traffic-sites-table-body')).not.toContainText(secondSite.name);
  await expect.poll(async () => {
    const report = await fetchPortfolioTrafficReport(config, cookie, customReportId || '');
    return report.site_count;
  }, { timeout: 10000 }).toBe(1);
  await page.locator('#global-traffic-report-clear-sites').click();
  await expect(page.locator('#global-traffic-report-sites-summary')).toHaveText('0 of 2 sites');
  await expect(page.locator('#all-sites-traffic-sites-table-body')).toContainText('No sites yet.');
  await page.locator('#global-traffic-report-select-all-sites').click();
  await expect(page.locator('#global-traffic-report-sites-summary')).toHaveText('2 of 2 sites');
  await expect(page.locator('#all-sites-traffic-sites-table-body')).toContainText(secondSite.name);
  await expect.poll(async () => {
    const report = await fetchPortfolioTrafficReport(config, cookie, customReportId || '');
    return report.site_count;
  }, { timeout: 10000 }).toBe(2);

  await openDashboard(page, config, portfolioUser);
  await selectSite(page, firstSite.id);
  await page.evaluate(() => {
    document.dispatchEvent(new CustomEvent('mpr-user:menu-item', { detail: { action: 'account-settings' } }));
  });
  await page.locator('#settings-open-all-sites-traffic').click();
  await expect(page.locator('#global-traffic-reports-list')).toContainText('Executive traffic');
  await page.locator('#global-traffic-reports-list button').filter({ hasText: 'Executive traffic' }).click();
  await expect(page.locator('#global-traffic-report-name')).toHaveValue('Executive traffic');
  await expect(page.locator('#global-traffic-report-sites-summary')).toHaveText('2 of 2 sites');
  await expect(page.locator('#global-traffic-report-schedule-chip')).toHaveText('Schedule on');

  await page.locator('#all-sites-traffic-back-button').click();
  await expect(page.locator('#site-dashboard-view')).toBeVisible();
  await expect(page.locator('#all-sites-traffic-view')).toBeHidden();
  await expect(page.locator('#dashboard-section-tab-traffic')).toHaveClass(/active/);
  await expect(page.locator('#traffic-report-title')).toContainText('Traffic reports');
  await expect(page.locator('[data-widget-card="traffic"]')).toBeVisible();
  await expect(page.locator('#traffic-title')).toContainText('Traffic');
});

test('all-sites report definition load failure stays visible', async ({ page }) => {
  const portfolioUser = buildAdminUser(config, {
    email: buildUniqueEmail('portfolio-load-failure'),
    displayName: 'Portfolio Load Failure'
  });
  const cookie = buildSessionCookie(config, portfolioUser);
  const site = await createTestSite(config, cookie, {
    name: buildUniqueName('Portfolio Failure Site'),
    allowedOrigin: buildUniqueOrigin('portfolio-load-failure'),
    ownerEmail: portfolioUser.email
  });
  await page.route('**/api/reports/traffic/portfolio/reports', async (route) => {
    await route.fulfill({
      status: 500,
      contentType: 'application/json',
      body: JSON.stringify({ error: 'load_failed' })
    });
  });

  await openDashboard(page, config, portfolioUser);
  await selectSite(page, site.id);
  await page.evaluate(() => {
    document.dispatchEvent(new CustomEvent('mpr-user:menu-item', { detail: { action: 'account-settings' } }));
  });
  await page.locator('#settings-open-all-sites-traffic').click();
  await expect(page.locator('#all-sites-traffic-status')).toHaveText('Failed to load data.');
  await expect(page.locator('#all-sites-traffic-status')).toBeVisible();
});

test('all-sites traffic report schedule preserves saved timezone from dashboard edits', async ({ page }) => {
  const portfolioUser = buildAdminUser(config, {
    email: buildUniqueEmail('portfolio-timezone'),
    displayName: 'Portfolio Timezone'
  });
  const cookie = buildSessionCookie(config, portfolioUser);
  const site = await createTestSite(config, cookie, {
    name: buildUniqueName('Portfolio Timezone Site'),
    allowedOrigin: buildUniqueOrigin('portfolio-timezone'),
    ownerEmail: portfolioUser.email
  });
  const savedTimezone = 'Pacific/Honolulu';
  await savePortfolioTrafficReportSchedule(config, cookie, {
    enabled: true,
    frequency: 'daily',
    recipient_email: portfolioUser.email,
    timezone: savedTimezone,
    send_hour: 7,
    send_minute: 15,
    weekday: 1,
    month_day: 1
  });

  await openDashboard(page, config, portfolioUser);
  await selectSite(page, site.id);
  await page.evaluate(() => {
    document.dispatchEvent(new CustomEvent('mpr-user:menu-item', { detail: { action: 'account-settings' } }));
  });
  await page.locator('#settings-open-all-sites-traffic').click();
  await expect(page.locator('#all-sites-traffic-report-timezone')).toHaveValue(savedTimezone);
  await page.locator('#all-sites-traffic-report-send-time').fill('13:25');

  await expect.poll(async () => {
    const schedule = await fetchPortfolioTrafficReportSchedule(config, cookie);
    return {
      timezone: schedule.timezone,
      send_hour: schedule.send_hour,
      send_minute: schedule.send_minute
    };
  }, { timeout: 10000 }).toEqual({
    timezone: savedTimezone,
    send_hour: 13,
    send_minute: 25
  });
});

test('path, device, and timezone fetch failures show traffic error state', async ({ page }) => {
  const site = await createTrafficSite();
  await page.route(`**/api/sites/${site.id}/visits/stats`, async (route) => {
    await route.fulfill({
      status: 500,
      contentType: 'application/json',
      body: JSON.stringify({ error: 'query_failed' })
    });
  });
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
  await expect(page.locator('#top-pages-chart')).toContainText('Failed to load data.');
  await expect(page.locator('#device-types-chart')).toContainText('Failed to load data.');
  await expect(page.locator('#timezones-map')).toContainText('Failed to load data.');
});

test('traffic stats refresh after reload', async ({ page }) => {
  const site = await createTrafficSite();
  await collectVisit(config, site, { url: `${site.allowed_origin}/alpha`, visitorId: buildVisitorId() });
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#visit-count')).toHaveText('1 visits');
  await collectVisit(config, site, { url: `${site.allowed_origin}/beta`, visitorId: buildVisitorId() });
  await page.reload({ waitUntil: 'domcontentloaded' });
  await waitForDashboardReady(page);
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

test('device row graph shows breakdown by viewport width', async ({ page }) => {
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
  await page.locator('#dashboard-section-tab-traffic').click();
  const mobileRow = page.locator('#device-types-chart .device-row[aria-label="mobile 1 visits"]');
  const desktopRow = page.locator('#device-types-chart .device-row[aria-label="desktop 1 visits"]');
  await expect(mobileRow).toBeVisible();
  await expect(mobileRow.locator('.device-row__icon svg')).toBeVisible();
  await expect(mobileRow.locator('.device-row__count')).toHaveText('1');
  await expect(desktopRow).toBeVisible();
  await expect(desktopRow.locator('.device-row__icon svg')).toBeVisible();
  await expect(desktopRow.locator('.device-row__count')).toHaveText('1');
});

test('device row graph shows placeholder for new sites', async ({ page }) => {
  const site = await createTrafficSite();
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#device-types-chart')).toContainText('No device data yet');
});

test('timezone map shows visit distribution with proportional bubbles', async ({ page }) => {
  const site = await createTrafficSite();
  await collectVisit(config, site, {
    url: `${site.allowed_origin}/page1`,
    visitorId: buildVisitorId(),
    timezone: 'America/New_York'
  });
  await collectVisit(config, site, {
    url: `${site.allowed_origin}/page2`,
    visitorId: buildVisitorId(),
    timezone: 'America/New_York'
  });
  await collectVisit(config, site, {
    url: `${site.allowed_origin}/page3`,
    visitorId: buildVisitorId(),
    timezone: 'Europe/London'
  });
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await page.locator('#dashboard-section-tab-traffic').click();
  await expect(page.locator('#timezones-map svg')).toBeVisible();
  await expect(page.locator('#timezones-map circle[data-timezone="America/New_York"]')).toBeVisible();
  await expect(page.locator('#timezones-map circle[data-timezone="Europe/London"]')).toBeVisible();
  await expect(page.locator('#timezones-map')).toContainText('New York 2');
  await expect(page.locator('#timezones-map')).toContainText('London 1');
  const newYorkRadius = await timezoneBubbleRadius(page, 'America/New_York');
  const londonRadius = await timezoneBubbleRadius(page, 'Europe/London');
  expect(newYorkRadius).toBeGreaterThan(londonRadius);
});

test('timezone map shows placeholder for new sites', async ({ page }) => {
  const site = await createTrafficSite();
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#timezones-map')).toContainText('No timezone data yet');
});
