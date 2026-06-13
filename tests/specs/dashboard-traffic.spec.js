// @ts-check
import { test, expect } from '@playwright/test';
import * as crypto from 'node:crypto';
import { resolveTestConfig } from '../helpers/config.js';
import { buildSessionCookie } from '../helpers/auth.js';
import { buildAdminUser, buildUniqueEmail, buildUniqueName, buildUniqueOrigin, createTestSite, openDashboard, selectSite, waitForDashboardReady } from '../helpers/fixtures.js';
import { apiRequest, collectVisit, fetchPortfolioTrafficReport, fetchPortfolioTrafficReports, fetchPortfolioTrafficReportSchedule, fetchTrafficReportSchedule, fetchVisitStats, savePortfolioTrafficReportSchedule, saveTrafficReportSchedule } from '../helpers/api.js';

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

async function locationBubbleRadius(page, source, signal) {
  const radius = await page.locator(`#locations-map circle[data-location-source="${source}"][data-location-signal="${signal}"]`).getAttribute('r');
  if (!radius) {
    throw new Error(`missing_location_bubble:${source}:${signal}`);
  }
  return Number(radius);
}

async function locationBubbleCount(page, source, signal) {
  const count = await page.locator(`#locations-map .traffic-map__bubble-count[data-location-source="${source}"][data-location-signal="${signal}"]`).textContent();
  if (!count) {
    throw new Error(`missing_location_bubble_count:${source}:${signal}`);
  }
  return count.trim();
}

async function locationBubbleCoordinates(page, source, signal) {
  const circle = page.locator(`#locations-map circle[data-location-source="${source}"][data-location-signal="${signal}"]`);
  const [x, y, latitude, longitude] = await Promise.all([
    circle.getAttribute('cx'),
    circle.getAttribute('cy'),
    circle.getAttribute('data-latitude'),
    circle.getAttribute('data-longitude')
  ]);
  if (!x || !y || !latitude || !longitude) {
    throw new Error(`missing_location_coordinates:${source}:${signal}`);
  }
  return {
    x: Number(x),
    y: Number(y),
    latitude: Number(latitude),
    longitude: Number(longitude)
  };
}

async function locationLabelCenterX(page, source, signal) {
  const labelBox = page.locator(`#locations-map .traffic-map__marker[data-location-source="${source}"][data-location-signal="${signal}"] .traffic-map__label-box`);
  const [x, width] = await Promise.all([
    labelBox.getAttribute('x'),
    labelBox.getAttribute('width')
  ]);
  if (!x || !width) {
    throw new Error(`missing_location_label_box:${source}:${signal}`);
  }
  return Number(x) + (Number(width) / 2);
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
  await expectTrendAxisLabels(page, '#traffic-trend-chart', 1);
  const topPageRow = page.locator('#top-pages-chart .path-row[aria-label="/alpha 2 visits"]');
  await expect(topPageRow).toBeVisible();
  await expect(topPageRow.locator('.path-row__icon svg')).toBeVisible();
  await expect(topPageRow.locator('.path-row__count')).toHaveText('2');
  await expect(page.locator('#traffic-attribution-chart')).toContainText('google');
  await expect(page.locator('#traffic-engagement-summary')).toContainText('Returning rate');
  await expect(page.locator('#device-types-chart .device-row[aria-label="mobile 1 visits"]')).toBeVisible();
  await expect(page.locator('#device-types-chart .device-row[aria-label="mobile 1 visits"] .device-row__icon svg')).toBeVisible();
  await expect(page.locator('#locations-map svg')).toBeVisible();
  await expect(page.locator('#locations-map circle[data-location-source="timezone"][data-location-signal="America/New_York"]')).toBeVisible();
  await expect(page.locator('#locations-map .traffic-map__bubble-label[data-location-source="timezone"][data-location-signal="America/New_York"]')).toHaveText('New York');
  expect(await locationBubbleCount(page, 'timezone', 'America/New_York')).toBe('2');
});

test('traffic report schedule can be configured from dashboard', async ({ page }) => {
  const site = await createTrafficSite();
  const cookie = buildAdminCookie();
  const teamEmail = buildUniqueEmail('traffic-report-member');
  const teamMember = await apiRequest({
    baseURL: config.baseURL,
    cookie,
    path: `/api/sites/${site.id}/team`,
    method: 'POST',
    body: { email: teamEmail }
  });
  expect(teamMember.response.status, JSON.stringify(teamMember.payload)).toBe(200);
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await page.locator('#dashboard-section-tab-traffic').click();
  await expect(page.locator('[data-dashboard-card="traffic-report"]')).toBeVisible();
  await expect(page.locator('[data-dashboard-card="traffic-report"] .card-header #traffic-report-enabled')).toBeVisible();
  await expect(page.locator('#traffic-report-save-button')).toHaveCount(0);
  await expect(page.locator('#traffic-report-recipient')).toHaveValue(config.adminEmail);
  await expect(page.locator('#traffic-report-recipient')).toHaveAttribute('readonly', '');
  await expect(page.locator('#traffic-report-recipient-mode-manager')).toBeChecked();
  await expect(page.locator('label[for="traffic-report-recipient-mode-team"]')).toContainText('Whole team');
  await expect(page.locator('label[for="traffic-report-recipient-mode-selected"]')).toContainText('Selected members');
  await expect(page.locator('#traffic-report-timezone')).toHaveAttribute('readonly', '');
  const detectedTimezone = await page.evaluate(() => Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC');
  await expect(page.locator('#traffic-report-timezone')).toHaveValue(detectedTimezone);

  await page.locator('#traffic-report-enabled').check();
  await page.locator('label[for="traffic-report-frequency-weekly"]').click();
  await expect(page.locator('#traffic-report-weekday-container')).toBeVisible();
  await page.locator('#traffic-report-send-time').fill('14:30');
  await page.locator('#traffic-report-weekday').selectOption('5');
  await page.locator('label[for="traffic-report-recipient-mode-selected"]').click();
  await expect(page.locator('#traffic-report-selected-members-container')).toBeVisible();
  const selectedMemberCheckbox = page.locator(`#traffic-report-selected-members-list input[data-recipient-email="${teamEmail.toLowerCase()}"]`);
  await expect(selectedMemberCheckbox).toBeVisible();
  await selectedMemberCheckbox.check();

  await expect.poll(async () => {
    const schedule = await fetchTrafficReportSchedule(config, cookie, site.id);
    return {
      enabled: schedule.enabled,
      frequency: schedule.frequency,
      recipient_email: schedule.recipient_email,
      recipient_mode: schedule.recipient_mode,
      recipient_emails: schedule.recipient_emails,
      timezone: schedule.timezone,
      send_hour: schedule.send_hour,
      send_minute: schedule.send_minute,
      weekday: schedule.weekday
    };
  }, { timeout: 10000 }).toEqual({
    enabled: true,
    frequency: 'weekly',
    recipient_email: config.adminEmail,
    recipient_mode: 'selected',
    recipient_emails: [teamEmail.toLowerCase()],
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

test('path, device, and location fetch failures show traffic error state', async ({ page }) => {
  const site = await createTrafficSite();
  await page.route(`**/api/sites/${site.id}/visits/stats*`, async (route) => {
    await route.fulfill({
      status: 500,
      contentType: 'application/json',
      body: JSON.stringify({ error: 'query_failed' })
    });
  });
  await page.route(`**/api/sites/${site.id}/visits/devices*`, async (route) => {
    await route.fulfill({
      status: 500,
      contentType: 'application/json',
      body: JSON.stringify({ error: 'query_failed' })
    });
  });
  await page.route(`**/api/sites/${site.id}/visits/locations*`, async (route) => {
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
  await expect(page.locator('#locations-map')).toContainText('Failed to load data.');
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

test('traffic interval selector refreshes selected-site stats', async ({ page }) => {
  const site = await createTrafficSite();
  await collectVisit(config, site, { url: `${site.allowed_origin}/alpha`, visitorId: buildVisitorId() });
  await collectVisit(config, site, { url: `${site.allowed_origin}/beta`, visitorId: buildVisitorId() });
  /** @type {string[]} */
  const requestedIntervals = [];
  await page.route(`**/api/sites/${site.id}/visits/stats*`, async (route) => {
    const url = new URL(route.request().url());
    requestedIntervals.push(url.searchParams.get('interval') || '');
    if (url.searchParams.get('interval') === '1day') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          site_id: site.id,
          interval: '1day',
          visit_count: 1,
          unique_visitor_count: 1,
          top_pages: [{ path: '/alpha', visit_count: 1 }],
          recent_visits: []
        })
      });
      return;
    }
    await route.continue();
  });

  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await page.locator('#dashboard-section-tab-traffic').click();
  await expect(page.locator('#traffic-interval-all')).toBeChecked();
  await expect(page.locator('#visit-count')).toHaveText('2 visits');
  await expect.poll(() => requestedIntervals).toContain('all');

  await page.locator('label[for="traffic-interval-1day"]').click();
  await expect(page.locator('#traffic-interval-1day')).toBeChecked();
  await expect(page.locator('#visit-count')).toHaveText('1 visits');
  await expect(page.locator('#unique-visitor-count')).toHaveText('1 unique');
  await expect(page.locator('#top-pages-chart .path-row__label').first()).toHaveText('/alpha');
  await expect.poll(() => requestedIntervals).toContain('1day');
});

test('traffic CSV button opens the selected interval export', async ({ page }) => {
  const site = await createTrafficSite();
  await collectVisit(config, site, { url: `${site.allowed_origin}/alpha`, visitorId: buildVisitorId() });
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await page.locator('#dashboard-section-tab-traffic').click();
  await page.locator('label[for="traffic-interval-30days"]').click();
  await expect(page.locator('#traffic-interval-30days')).toBeChecked();
  await page.evaluate(() => {
    window.sessionStorage.setItem('loopaware-test-traffic-export-url', '');
    window.open = function(url) {
      window.sessionStorage.setItem('loopaware-test-traffic-export-url', String(url || ''));
      return null;
    };
  });

  await page.locator('#traffic-export-button').click();
  const exportURL = await page.evaluate(() => window.sessionStorage.getItem('loopaware-test-traffic-export-url') || '');
  const parsedURL = new URL(exportURL, config.baseURL);
  expect(parsedURL.pathname).toBe(`/api/sites/${site.id}/visits/export`);
  expect(parsedURL.searchParams.get('interval')).toBe('30days');
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
    url: `${site.allowed_origin}/tablet`,
    visitorId: buildVisitorId(),
    viewport: '800x600',
    screenResolution: '1024x768'
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
  const tabletRow = page.locator('#device-types-chart .device-row[aria-label="tablet 1 visits"]');
  const desktopRow = page.locator('#device-types-chart .device-row[aria-label="desktop 1 visits"]');
  await expect(mobileRow).toBeVisible();
  await expect(mobileRow.locator('.device-row__icon svg')).toBeVisible();
  await expect(mobileRow.locator('.device-row__icon svg rect').first()).toHaveAttribute('width', '8');
  await expect(mobileRow.locator('.device-row__icon svg rect').first()).toHaveAttribute('height', '18');
  await expect(mobileRow.locator('.device-row__count')).toHaveText('1');
  await expect(tabletRow).toBeVisible();
  await expect(tabletRow.locator('.device-row__icon svg')).toBeVisible();
  await expect(tabletRow.locator('.device-row__icon svg rect').first()).toHaveAttribute('width', '18');
  await expect(tabletRow.locator('.device-row__icon svg rect').first()).toHaveAttribute('height', '14');
  await expect(tabletRow.locator('.device-row__count')).toHaveText('1');
  await expect(desktopRow).toBeVisible();
  await expect(desktopRow.locator('.device-row__icon svg')).toBeVisible();
  await expect(desktopRow.locator('.device-row__icon svg rect').first()).toHaveAttribute('width', '18');
  await expect(desktopRow.locator('.device-row__icon svg rect').first()).toHaveAttribute('height', '12');
  await expect(desktopRow.locator('.device-row__count')).toHaveText('1');
});

test('device row graph shows placeholder for new sites', async ({ page }) => {
  const site = await createTrafficSite();
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#device-types-chart')).toContainText('No device data yet');
});

test('location map shows visit distribution with proportional bubbles', async ({ page }) => {
  const site = await createTrafficSite();
  await collectVisit(config, site, {
    url: `${site.allowed_origin}/page0`,
    visitorId: buildVisitorId(),
    timezone: 'America/Los_Angeles'
  });
  await collectVisit(config, site, {
    url: `${site.allowed_origin}/page0-repeat`,
    visitorId: buildVisitorId(),
    timezone: 'America/Los_Angeles'
  });
  await collectVisit(config, site, {
    url: `${site.allowed_origin}/page0-third`,
    visitorId: buildVisitorId(),
    timezone: 'America/Los_Angeles'
  });
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
  await expect(page.locator('#locations-map svg')).toBeVisible();
  await expect(page.locator('#locations-map svg')).toHaveAttribute('data-map-source', 'natural-earth-110m');
  await expect(page.locator('#locations-map svg')).toHaveAttribute('data-projection', 'equirectangular');
  await expect(page.locator('#locations-map .traffic-map__land')).toHaveCount(1);
  await expect(page.locator('#locations-map .traffic-map__land')).toHaveAttribute('data-map-source', 'natural-earth-110m');
  await expect(page.locator('#locations-map .traffic-map__label-box')).toHaveCount(3);
  await expect(page.locator('#locations-map circle[data-location-source="timezone"][data-location-signal="America/Los_Angeles"]')).toBeVisible();
  await expect(page.locator('#locations-map circle[data-location-source="timezone"][data-location-signal="America/New_York"]')).toBeVisible();
  await expect(page.locator('#locations-map circle[data-location-source="timezone"][data-location-signal="Europe/London"]')).toBeVisible();
  await expect(page.locator('#locations-map .traffic-map__bubble-label[data-location-source="timezone"][data-location-signal="America/Los_Angeles"]')).toHaveText('Los Angeles');
  await expect(page.locator('#locations-map .traffic-map__bubble-label[data-location-source="timezone"][data-location-signal="America/New_York"]')).toHaveText('New York');
  await expect(page.locator('#locations-map .traffic-map__bubble-label[data-location-source="timezone"][data-location-signal="Europe/London"]')).toHaveText('London');
  await expect(page.locator('#locations-map .traffic-map__bubble-count[data-location-source="timezone"][data-location-signal="America/Los_Angeles"]')).toBeVisible();
  expect(await locationBubbleCount(page, 'timezone', 'America/Los_Angeles')).toBe('3');
  expect(await locationBubbleCount(page, 'timezone', 'America/New_York')).toBe('2');
  expect(await locationBubbleCount(page, 'timezone', 'Europe/London')).toBe('1');
  const losAngelesRadius = await locationBubbleRadius(page, 'timezone', 'America/Los_Angeles');
  const newYorkRadius = await locationBubbleRadius(page, 'timezone', 'America/New_York');
  const londonRadius = await locationBubbleRadius(page, 'timezone', 'Europe/London');
  const landPath = await page.locator('#locations-map .traffic-map__land').getAttribute('d');
  const losAngelesCoordinates = await locationBubbleCoordinates(page, 'timezone', 'America/Los_Angeles');
  const newYorkCoordinates = await locationBubbleCoordinates(page, 'timezone', 'America/New_York');
  const londonCoordinates = await locationBubbleCoordinates(page, 'timezone', 'Europe/London');
  const losAngelesLabelCenterX = await locationLabelCenterX(page, 'timezone', 'America/Los_Angeles');
  const newYorkLabelCenterX = await locationLabelCenterX(page, 'timezone', 'America/New_York');
  const londonLabelCenterX = await locationLabelCenterX(page, 'timezone', 'Europe/London');
  expect(landPath.length).toBeGreaterThan(1000);
  expect(losAngelesCoordinates.latitude).toBeCloseTo(34.0522, 4);
  expect(losAngelesCoordinates.longitude).toBeCloseTo(-118.2437, 4);
  expect(losAngelesCoordinates.x).toBeCloseTo(162.93, 1);
  expect(losAngelesCoordinates.y).toBeCloseTo(93.25, 1);
  expect(newYorkCoordinates.latitude).toBeCloseTo(40.7128, 4);
  expect(newYorkCoordinates.longitude).toBeCloseTo(-74.006, 3);
  expect(newYorkCoordinates.x).toBeCloseTo(236.66, 1);
  expect(newYorkCoordinates.y).toBeCloseTo(82.15, 1);
  expect(londonCoordinates.latitude).toBeCloseTo(51.5072, 4);
  expect(londonCoordinates.longitude).toBeCloseTo(-0.1276, 4);
  expect(londonCoordinates.x).toBeCloseTo(359.79, 1);
  expect(londonCoordinates.y).toBeCloseTo(64.15, 1);
  expect(losAngelesLabelCenterX).toBeCloseTo(losAngelesCoordinates.x, 1);
  expect(newYorkLabelCenterX).toBeCloseTo(newYorkCoordinates.x, 1);
  expect(londonLabelCenterX).toBeCloseTo(londonCoordinates.x, 1);
  expect(losAngelesRadius).toBeGreaterThan(newYorkRadius);
  expect(newYorkRadius).toBeGreaterThan(londonRadius);
  expect(losAngelesRadius).toBeLessThan(24);
});

test('location map uses locale signal when timezone is missing', async ({ page }) => {
  const site = await createTrafficSite();
  await collectVisit(config, site, {
    url: `${site.allowed_origin}/locale-only`,
    visitorId: buildVisitorId(),
    locale: 'en-US'
  });
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await page.locator('#dashboard-section-tab-traffic').click();
  await expect(page.locator('#locations-map circle[data-location-source="locale"][data-location-signal="US"]')).toBeVisible();
  await expect(page.locator('#locations-map .traffic-map__bubble-label[data-location-source="locale"][data-location-signal="US"]')).toHaveText('United States');
  expect(await locationBubbleCount(page, 'locale', 'US')).toBe('1');
});

test('location map prefers edge geo signal with confidence metadata', async ({ page }) => {
  const site = await createTrafficSite();
  await collectVisit(config, site, {
    url: `${site.allowed_origin}/edge-geo`,
    visitorId: buildVisitorId(),
    timezone: 'Europe/London',
    locale: 'en-GB',
    headers: {
      'CF-IPCountry': 'US',
      'CF-Region-Code': 'CA',
      'CF-IPCity': 'San Francisco',
      'CF-IPLatitude': '37.7749',
      'CF-IPLongitude': '-122.4194'
    }
  });
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await page.locator('#dashboard-section-tab-traffic').click();

  const signal = 'cloudflare:US:CA:San Francisco';
  const edgeBubble = page.locator(`#locations-map circle[data-location-source="edge_geo"][data-location-signal="${signal}"]`);
  await expect(edgeBubble).toBeVisible();
  await expect(edgeBubble).toHaveAttribute('data-location-country', 'US');
  await expect(edgeBubble).toHaveAttribute('data-location-region', 'CA');
  await expect(edgeBubble).toHaveAttribute('data-location-city', 'San Francisco');
  await expect(edgeBubble).toHaveAttribute('data-location-confidence', '95');
  await expect(page.locator(`#locations-map .traffic-map__bubble-label[data-location-source="edge_geo"][data-location-signal="${signal}"]`)).toHaveText('San Francisco, CA');
  expect(await locationBubbleCount(page, 'edge_geo', signal)).toBe('1');
});

test('location map shows placeholder for new sites', async ({ page }) => {
  const site = await createTrafficSite();
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#locations-map')).toContainText('No location data yet');
});
