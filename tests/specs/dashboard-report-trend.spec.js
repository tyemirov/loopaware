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

async function openInteractiveReport(page) {
  const { owner, cookie, site } = await createDailyCountOwner();
  const detailed = await createTestSite(config, cookie, {
    name: 'Detailed hover traffic', allowedOrigin: buildUniqueOrigin('hover-detailed'), ownerEmail: owner.email
  });
  const report = await fetchPortfolioTrafficReport(config, cookie, undefined);
  const trend = [
    { date: '2026-11-01', page_views: 8, unique_visitors: 2 },
    { date: '2026-11-02', page_views: 21, unique_visitors: 3 },
    { date: '2026-11-10', page_views: 15, unique_visitors: 1 }
  ];
  await page.route(/\/api\/reports\/traffic\/portfolio(?:\?|$)/, route => route.fulfill({
    json: {
      ...report, visit_count: 44, unique_visitor_count: 3, trend,
      sites: report.sites.map(row => ({ ...row, visit_count: row.site_id === site.id ? 10 : 34, unique_visitor_count: row.site_id === detailed.id ? 3 : 0 }))
    }
  }));
  await openDashboard(page, config, owner, { localStorage: { loopaware_public_theme: 'dark' } });
  await page.locator('#header-statistics-button').click();
  const chart = page.locator(reportChartSelector);
  await expect(chart.locator('svg circle')).toHaveCount(6);
  return chart;
}

async function hoverTrendDate(page, chart, pointIndex) {
  const graph = chart.getByRole('img');
  await graph.evaluate(element => element.scrollIntoView({ block: 'center', inline: 'nearest', behavior: 'instant' }));
  const markerBox = await graph.locator('circle').nth(pointIndex).boundingBox();
  const graphBox = await graph.boundingBox();
  const position = { x: markerBox.x + markerBox.width / 2, y: graphBox.y + graphBox.height / 2 };
  expect(await graph.evaluate((element, point) => element.contains(document.elementFromPoint(point.x, point.y)), position)).toBe(true);
  await page.mouse.move(position.x, position.y);
}

test('report plot shows date and exact Visits and Unique values on hover and keyboard navigation', async ({ page }) => {
  const chart = await openInteractiveReport(page);
  const graph = chart.getByRole('img');
  const tooltip = chart.getByRole('tooltip');
  await hoverTrendDate(page, chart, 1);
  await expect(tooltip).toBeVisible();
  await expect(tooltip).toContainText('2026-11-02');
  await expect(tooltip).toContainText('Visits: 21');
  await expect(tooltip).toContainText('Unique: 3');
  await expect(chart.locator(':scope > .d-flex')).toHaveText(/Visits.*Unique/);
  await captureLocatorScreenshot(chart, ensureScreenshotDirectory('dashboard-I065'), 'hover-values');
  await hoverTrendDate(page, chart, 0);
  await expect(tooltip).toContainText('Visits: 8');
  await expect(tooltip).toContainText('Unique: 2');
  const graphBox = await graph.boundingBox();
  await page.mouse.move(graphBox.x - 4, graphBox.y - 4);
  await expect(tooltip).toBeHidden();
  await graph.focus();
  await expect(tooltip).toContainText('2026-11-10');
  await graph.press('ArrowLeft');
  await expect(tooltip).toContainText('Visits: 21');
  await graph.press('Home');
  await expect(tooltip).toContainText('Visits: 8');
  await graph.press('End');
  await expect(tooltip).toContainText('Visits: 15');
  await graph.press('Escape');
  await expect(tooltip).toBeHidden();
  await page.setViewportSize({ width: 390, height: 844 });
  await hoverTrendDate(page, chart, 2);
  await expect(tooltip).toContainText('Visits: 15');
  const tooltipBox = await tooltip.boundingBox();
  const chartBox = await chart.boundingBox();
  expect(tooltipBox.x).toBeGreaterThanOrEqual(chartBox.x);
  expect(tooltipBox.x + tooltipBox.width).toBeLessThanOrEqual(chartBox.x + chartBox.width);
  const dateLineCount = await tooltip.evaluate(element => {
    const range = document.createRange();
    range.setStart(element.firstChild, 0);
    range.setEnd(element.firstChild, '2026-11-10 (UTC)'.length);
    return range.getClientRects().length;
  });
  expect(dateLineCount).toBe(1);
  await captureLocatorScreenshot(chart, ensureScreenshotDirectory('dashboard-I065'), 'hover-values-narrow');
  await page.getByRole('button', { name: 'Dashboard', exact: true }).click();
  await page.getByRole('button', { name: 'Statistics', exact: true }).click();
  await expect(chart.locator('svg circle')).toHaveCount(6);
  await expect(tooltip).toBeHidden();
  await hoverTrendDate(page, chart, 0);
  await expect(tooltip).toContainText('Visits: 8');
});

test.describe('touch report graph', () => {
  test.use({ hasTouch: true });
  test('tapping the plot shows the nearest daily values', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    const chart = await openInteractiveReport(page);
    const graph = chart.getByRole('img');
    await graph.evaluate(element => element.scrollIntoView({ block: 'center', inline: 'nearest', behavior: 'instant' }));
    const graphBox = await graph.boundingBox();
    const firstBox = await graph.locator('circle').first().boundingBox();
    await graph.tap({ position: { x: firstBox.x + firstBox.width / 2 - graphBox.x, y: graphBox.height / 2 } });
    await expect(chart.getByRole('tooltip')).toContainText('Visits: 8');
    await expect(chart.getByRole('tooltip')).toContainText('Unique: 2');
    const markerBox = await graph.locator('circle').nth(2).boundingBox();
    await graph.tap({ position: { x: markerBox.x + markerBox.width / 2 - graphBox.x, y: graphBox.height / 2 } });
    await expect(chart.getByRole('tooltip')).toContainText('Visits: 15');
    await expect(chart.getByRole('tooltip')).toContainText('Unique: 1');
  });
});

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
  test(`${profile} report graphs recorded Visits and Unique with labeled axes`, async ({ page }) => {
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
    await expect(graph.locator('text').filter({ hasText: /^Visits \/ visitors$/ })).toBeVisible();
    await expect(graph.locator('text').filter({ hasText: /^Date \(UTC\)$/ })).toBeVisible();
    await expect(graph.locator('circle')).toHaveCount(report.trend.length * 2);
    await expect(graph.locator('circle').nth(report.trend.length - 1)).toHaveAttribute('aria-label', `${report.trend.at(-1).date}: ${report.visit_count} visits`);
    await expect(graph.locator('circle').last()).toHaveAttribute('aria-label', `${report.trend.at(-1).date}: ${report.trend.at(-1).unique_visitors} unique visitors`);
    await expect(chart.locator(':scope > .d-flex')).toHaveText(/Visits.*Unique/);
    await expect(page.locator('#all-sites-unique-visitor-count')).toHaveText(`${profile === 'mixed' ? 1 : 0} unique`);
    const aggregateRow = page.locator('#all-sites-traffic-sites-table-body tr').filter({ hasText: 'Daily counts' });
    await expect(aggregateRow.locator('td').last()).toHaveText('0');
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
  await expect(graph.locator('circle')).toHaveCount(60);
  await expect(graph.locator('circle').first()).toHaveAttribute('aria-label', /: 0 visits$/);
  await expect(graph.locator('text').filter({ hasText: /^0$/ })).toBeVisible();
  await expect(graph).toContainText('Date (UTC)');
});

test('request trend positions sparse dates by elapsed UTC time', async ({ page }) => {
  const { owner, cookie } = await createDailyCountOwner();
  const report = await fetchPortfolioTrafficReport(config, cookie, undefined);
  const trend = ['2026-11-01', '2026-11-02', '2026-11-10'].map((date, index) => ({
    date, page_views: index + 1, unique_visitors: 0
  }));
  await page.route(/\/api\/reports\/traffic\/portfolio(?:\?|$)/, route => route.fulfill({
    json: { ...report, visit_count: 6, unique_visitor_count: 0, trend, sites: report.sites.map(site => ({ ...site, visit_count: 6, unique_visitor_count: 0 })) }
  }));
  await openDashboard(page, config, owner);
  await page.locator('#header-statistics-button').click();
  const graph = page.locator(reportChartSelector).getByRole('img');
  await expect(graph.locator('circle')).toHaveCount(6);
  const positions = await graph.locator('circle').evaluateAll(markers => markers.slice(0, 3).map(marker => Number(marker.getAttribute('cx'))));
  expect((positions[1] - positions[0]) / (positions[2] - positions[0])).toBeCloseTo(1 / 9, 2);
  await expect(graph).toHaveAttribute('aria-label', /Date \(UTC\) 2026-11-01 to 2026-11-10/);
});
