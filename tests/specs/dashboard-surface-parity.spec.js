// @ts-check
import * as crypto from 'node:crypto';
import { readFileSync } from 'node:fs';
import { stripTypeScriptTypes } from 'node:module';
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildAdminUser, buildUniqueEmail, buildUniqueName, buildUniqueOrigin, createTestSite, openDashboard, selectSite } from '../helpers/fixtures.js';
import { buildSessionCookie } from '../helpers/auth.js';
import { collectVisit, fetchPortfolioTrafficReportSchedule, savePortfolioTrafficReportSchedule } from '../helpers/api.js';

const config = resolveTestConfig();
const admin = buildAdminUser(config);

test('mobile world map image decodes at phone width with escaped Unicode labels', async ({ page }, testInfo) => {
  const compile = name => stripTypeScriptTypes(readFileSync(new URL(`../../mobile/src/${name}.ts`, import.meta.url), 'utf8'));
  const moduleURL = source => `data:text/javascript;base64,${Buffer.from(source).toString('base64')}`;
  const worldURL = moduleURL(compile('location-world'));
  const { locationMapImage } = await import(moduleURL(compile('location-map').replace('"./location-world"', JSON.stringify(worldURL))));
  const location = {label: '東京 & <West> 🌍', latitude: 35.68, longitude: 139.65, unique_visitor_count: 8, visit_count: 12};
  const colors = {surface:'#fff',surfaceAlt:'#eef3f1',muted:'#65726e',border:'#d8e0dd',teal:'#157f78',tealDark:'#0e5e59',ink:'#17211f'};
  const uri = locationMapImage([location, {...location, label:"12345678901234🌍long"}], colors);
  expect(uri).toMatch(/^data:image\/svg\+xml;base64,/);
  expect(Buffer.from(uri.split(',')[1], 'base64').toString('utf8')).toContain('東京 &amp; &lt;West&gt; 🌍');
  await page.setViewportSize({width:375,height:600});
  await page.setContent(`<img alt="Mobile world map" style="width:100%;aspect-ratio:2.4" src="${uri}">`);
  await expect.poll(() => page.getByAltText('Mobile world map').evaluate(image => /** @type {HTMLImageElement} */ (image).naturalWidth)).toBe(720);
  await expect(page.getByAltText('Mobile world map')).toBeVisible();
  await testInfo.attach('mobile-world-map', {body: await page.screenshot(), contentType: 'image/png'});
});

test('world map retains geographic placement and readable locations on web', async ({ page }) => {
  const site = await createTestSite(config, buildSessionCookie(config, admin), {
    name: buildUniqueName('Map parity'), allowedOrigin: buildUniqueOrigin('map-parity'), ownerEmail: config.adminEmail,
  });
  await collectVisit(config, site, { url: `${site.allowed_origin}/map`, visitorId: crypto.randomUUID(), timezone: 'America/Los_Angeles' });
  await openDashboard(page, config, admin);
  await selectSite(page, site.id);
  await page.locator('#dashboard-section-tab-traffic').click();
  const map = page.locator('#locations-map svg');
  await expect(map).toHaveAttribute('data-map-source', 'natural-earth-110m');
  await expect(map).toHaveAttribute('data-projection', 'equirectangular');
  await expect(map.locator('.traffic-map__bubble')).toHaveAttribute('cx', '162.93');
  await expect(map.locator('.traffic-map__bubble-label')).toHaveText('Los Angeles');
  for (const width of [375, 1280]) {
    await page.setViewportSize({ width, height: 900 });
    await expect(map).toBeVisible();
    expect(await map.evaluate(element => element.getBoundingClientRect().right)).toBeLessThanOrEqual(width);
  }
});

test('header statistics sits right of the avatar and preserves dashboard navigation', async ({ page }) => {
  const site = await createTestSite(config, buildSessionCookie(config, admin), {
    name: buildUniqueName('Header reports'), allowedOrigin: buildUniqueOrigin('header-reports'), ownerEmail: config.adminEmail,
  });
  await openDashboard(page, config, admin);
  await selectSite(page, site.id);
  await page.locator('#dashboard-section-tab-traffic').click();
  const statistics = page.getByRole('button', { name: 'Statistics', exact: true });
  await expect(statistics).toBeVisible();
  const avatar = await page.locator('mpr-user[data-loopaware-user-menu="true"]').boundingBox();
  const icon = await statistics.boundingBox();
  expect(avatar).not.toBeNull(); expect(icon).not.toBeNull();
  expect(icon.x).toBeGreaterThanOrEqual(avatar.x + avatar.width);
  await statistics.click();
  await expect(page.locator('#all-sites-traffic-view')).toBeVisible();
  await expect(page.locator('#global-traffic-report-library')).toBeVisible();
  const dashboard = page.getByRole('button', { name: 'Back to dashboard', exact: true });
  await expect(dashboard.locator('i')).toHaveClass('bi bi-speedometer2');
  await dashboard.click();
  await expect(page.locator('#site-dashboard-view')).toBeVisible();
  await expect(statistics.locator('i')).toHaveClass('bi bi-bar-chart-line');
  await expect(page.locator(`#sites-list [data-site-id="${site.id}"]`)).toHaveClass(/active/);
  await expect(page.locator('#dashboard-section-tab-traffic')).toHaveAttribute('aria-selected', 'true');
});

for (const writeState of ['pending', 'in-flight', 'failed']) {
  test(`header dashboard navigation retains ${writeState} report schedule edits`, async ({ page }) => {
    const user = buildAdminUser(config, { email: buildUniqueEmail('statistics-schedule') });
    const cookie = buildSessionCookie(config, user);
    await createTestSite(config, cookie, {
      name: buildUniqueName('Statistics schedule'),
      allowedOrigin: buildUniqueOrigin('statistics-schedule'),
      ownerEmail: user.email,
    });
    await savePortfolioTrafficReportSchedule(config, cookie, {
      enabled: true, frequency: 'daily', recipient_email: user.email,
      timezone: 'Pacific/Honolulu', send_hour: 7, send_minute: 15, weekday: 1, month_day: 1,
    });
    await openDashboard(page, config, user);
    const statistics = page.getByRole('button', { name: 'Statistics', exact: true });
    await statistics.click();
    const sendTime = page.locator('#all-sites-traffic-report-send-time');
    await expect(sendTime).toHaveValue('07:15');

    await page.clock.pauseAt(new Date());
    await sendTime.fill('13:25');
    const dashboard = page.getByRole('button', { name: 'Back to dashboard', exact: true });
    let expectedTime = '13:25';
    let expectedHour = 13;
    let expectedMinute = 25;
    /** @type {() => void} */
    let releaseWrite = () => {};
    if (writeState === 'in-flight') {
      const gate = new Promise(resolve => { releaseWrite = () => resolve(undefined); });
      await page.route('**/api/reports/traffic/portfolio/schedule*', async route => {
        if (route.request().method() === 'PUT') {
          await gate;
        }
        await route.continue();
      });
      const started = page.waitForRequest(request => request.method() === 'PUT' && request.url().includes('/portfolio/schedule'));
      await page.clock.runFor(1000);
      await started;
      await sendTime.fill('14:35');
      expectedTime = '14:35';
      expectedHour = 14;
      expectedMinute = 35;
    } else if (writeState === 'failed') {
      await page.route('**/api/reports/traffic/portfolio/schedule*', route => route.fulfill({
        status: 503, contentType: 'application/json', body: JSON.stringify({ error: 'save_failed' }),
      }), { times: 1 });
    }
    await dashboard.click();
    if (writeState === 'in-flight') {
      await expect(page.locator('#all-sites-traffic-view')).toBeVisible();
      releaseWrite();
    } else if (writeState === 'failed') {
      await expect(page.locator('#all-sites-traffic-report-status')).toHaveText('Failed to save site.');
      await expect(page.locator('#all-sites-traffic-view')).toBeVisible();
      await expect(sendTime).toHaveValue(expectedTime);
      await dashboard.click();
    }
    await expect(page.locator('#site-dashboard-view')).toBeVisible();
    await expect(page.locator('#dashboard-section-tab-feedback')).toHaveAttribute('aria-selected', 'true');
    await expect.poll(async () => {
      const schedule = await fetchPortfolioTrafficReportSchedule(config, cookie);
      return { send_hour: schedule.send_hour, send_minute: schedule.send_minute, timezone: schedule.timezone };
    }).toEqual({ send_hour: expectedHour, send_minute: expectedMinute, timezone: 'Pacific/Honolulu' });

    await statistics.click();
    await expect(sendTime).toHaveValue(expectedTime);
  });
}

for (const trafficProfile of ['detailed', 'aggregate']) {
  test(`seven days controls ${trafficProfile} traffic and export across navigation`, async ({ page }) => {
    const cookie = buildSessionCookie(config, admin);
    const site = await createTestSite(config, cookie, {
      name: buildUniqueName('Seven days'), allowedOrigin: buildUniqueOrigin('seven-days'), ownerEmail: config.adminEmail,
      trafficProfile,
    });
    const other = await createTestSite(config, cookie, {
      name: buildUniqueName('Other window'), allowedOrigin: buildUniqueOrigin('other-window'), ownerEmail: config.adminEmail,
      trafficProfile,
    });
    await openDashboard(page, config, admin);
    await selectSite(page, site.id);
    await page.locator('#dashboard-section-tab-traffic').click();
    const requests = [];
    page.on('request', request => { if (request.url().includes(`/api/sites/${site.id}/visits/`)) requests.push(new URL(request.url())); });
    await expect(page.getByText('7 days', { exact: true })).toBeVisible();
    await page.getByText('7 days', { exact: true }).click();
    await expect(page.locator('#traffic-interval-7days')).toBeChecked();
    const suffixes = trafficProfile === 'detailed' ? ['stats', 'trend', 'engagement', 'attribution', 'devices', 'locations'] : ['stats', 'trend'];
    for (const suffix of suffixes) {
      await expect.poll(() => requests.some(url => url.pathname.endsWith(`/visits/${suffix}`) && url.searchParams.get('interval') === '7days')).toBe(true);
    }
    if (trafficProfile === 'detailed') {
      const exported = page.context().waitForEvent('request', request => request.url().includes(`/api/sites/${site.id}/visits/export?interval=7days`));
      await page.locator('#traffic-export-button').click();
      await exported;
    } else {
      expect(requests.some(url => /\/(locations|devices|attribution|engagement)$/.test(url.pathname))).toBe(false);
      await expect(page.locator('#locations-map svg')).toHaveCount(0);
      await expect(page.locator('#locations-map')).toContainText('Unavailable for daily counts');
    }
    await page.getByRole('button', { name: 'Statistics', exact: true }).click();
    await page.locator('#all-sites-traffic-back-button').click();
    await expect(page.locator('#traffic-interval-7days')).toBeChecked();
    const nextStats = page.waitForRequest(request => request.url().includes(`/api/sites/${other.id}/visits/stats?interval=7days`));
    await selectSite(page, other.id);
    await nextStats;
    await expect(page.locator('#traffic-interval-7days')).toBeChecked();
  });
}
