// @ts-check
import { test, expect } from '@playwright/test';
import { randomUUID } from 'node:crypto';
import { resolveTestConfig } from '../helpers/config.js';
import { buildSessionCookie } from '../helpers/auth.js';
import { apiRequest, collectVisit, fetchPortfolioTrafficReport } from '../helpers/api.js';
import { buildAdminUser, buildUniqueEmail, buildUniqueOrigin, createTestSite, openDashboard } from '../helpers/fixtures.js';
import { captureLocatorScreenshot, ensureScreenshotDirectory } from '../helpers/screenshot.js';

const config = resolveTestConfig();
const reportChartSelector = '#all-sites-traffic-trend-chart';

async function createDailyCountOwner() {
  const owner = buildAdminUser(config, { email: buildUniqueEmail('trend-owner') });
  const cookie = buildSessionCookie(config, owner);
  const site = await createTestSite(config, cookie, {
    name: 'Daily counts', allowedOrigin: buildUniqueOrigin('trend-counts'), ownerEmail: owner.email, trafficProfile: 'aggregate'
  });
  return { owner, cookie, site };
}

async function countRequest(site) {
  const { response } = await apiRequest({
    baseURL: config.baseURL, path: `/public/sites/${site.id}/visit-counts`, method: 'POST', origin: site.allowed_origin, body: {}
  });
  expect(response.status).toBe(204);
}

for (const profile of ['aggregate', 'mixed']) {
  test(`${profile} report graphs the complete request trend with labeled axes`, async ({ page }) => {
    const { owner, cookie, site } = await createDailyCountOwner();
    await countRequest(site);
    if (profile === 'mixed') {
      const detailed = await createTestSite(config, cookie, {
        name: 'Detailed traffic', allowedOrigin: buildUniqueOrigin('trend-detailed'), ownerEmail: owner.email
      });
      await collectVisit(config, detailed, { url: `${detailed.allowed_origin}/`, visitorId: randomUUID() });
    }
    const report = await fetchPortfolioTrafficReport(config, cookie, undefined);
    await openDashboard(page, config, owner, { localStorage: { loopaware_public_theme: 'dark' } });
    await page.locator('#header-statistics-button').click();
    const chart = page.locator(reportChartSelector);
    const graph = chart.getByRole('img');
    await expect(graph).toBeVisible();
    await expect(graph.locator('text').filter({ hasText: /^Recorded requests$/ })).toBeVisible();
    await expect(graph.locator('text').filter({ hasText: /^Date \(UTC\)$/ })).toBeVisible();
    await expect(graph.locator('circle')).toHaveCount(report.trend.length);
    await expect(graph.locator('circle').last()).toHaveAttribute('aria-label', `${report.trend.at(-1).date}: ${report.visit_count} recorded requests`);
    await expect(chart).not.toContainText('Unique');
    const aggregateRow = page.locator('#all-sites-traffic-sites-table-body tr').filter({ hasText: 'Daily counts' });
    await expect(aggregateRow).toContainText('Not collected (daily counts)');
    if (profile === 'mixed') {
      await expect(page.locator('#all-sites-traffic-sites-table-body tr').filter({ hasText: 'Detailed traffic' }).locator('td').last()).toHaveText('1');
    }
    await captureLocatorScreenshot(chart, ensureScreenshotDirectory('dashboard-I064'), `${profile}-report`);
    await page.setViewportSize({ width: 390, height: 844 });
    await expect(graph).toBeVisible();
    expect((await graph.boundingBox()).width).toBeLessThanOrEqual(390);
  });
}

test('single-day aggregate traffic has a visible point and UTC date axis', async ({ page }) => {
  const { owner, site } = await createDailyCountOwner();
  await countRequest(site);
  await openDashboard(page, config, owner);
  await page.locator('#dashboard-section-tab-traffic').click();
  const graph = page.locator('#traffic-trend-chart').getByRole('img');
  await expect(graph).toBeVisible();
  await expect(graph.locator('circle')).toHaveCount(1);
  await expect(graph.locator('circle')).toHaveAttribute('aria-label', /: 1 requests$/);
  await expect(graph).toContainText('Date (UTC)');
  await expect(graph).toContainText('Requests');
});

test('zero-request report retains labeled axes and zero data points', async ({ page }) => {
  const { owner } = await createDailyCountOwner();
  await openDashboard(page, config, owner);
  await page.locator('#header-statistics-button').click();
  const graph = page.locator(reportChartSelector).getByRole('img');
  await expect(graph).toBeVisible();
  await expect(graph.locator('circle')).toHaveCount(30);
  await expect(graph.locator('circle').first()).toHaveAttribute('aria-label', /: 0 recorded requests$/);
  await expect(graph.locator('text').filter({ hasText: /^0$/ })).toBeVisible();
  await expect(graph).toContainText('Date (UTC)');
});

test('request trend positions sparse dates by elapsed UTC time', async ({ page }) => {
  const { owner, cookie } = await createDailyCountOwner();
  const report = await fetchPortfolioTrafficReport(config, cookie, undefined);
  const trend = ['2026-11-01', '2026-11-02', '2026-11-10'].map((date, index) => ({
    date, page_views: index + 1, unique_visitors: null
  }));
  await page.route(/\/api\/reports\/traffic\/portfolio(?:\?|$)/, route => route.fulfill({
    json: { ...report, visit_count: 6, trend, sites: report.sites.map(site => ({ ...site, visit_count: 6 })) }
  }));
  await openDashboard(page, config, owner);
  await page.locator('#header-statistics-button').click();
  const graph = page.locator(reportChartSelector).getByRole('img');
  await expect(graph.locator('circle')).toHaveCount(3);
  const positions = await graph.locator('circle').evaluateAll(markers => markers.map(marker => Number(marker.getAttribute('cx'))));
  expect((positions[1] - positions[0]) / (positions[2] - positions[0])).toBeCloseTo(1 / 9, 2);
  await expect(graph).toHaveAttribute('aria-label', /Date \(UTC\) 2026-11-01 to 2026-11-10/);
});
