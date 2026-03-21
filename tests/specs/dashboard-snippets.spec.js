// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildSessionCookie } from '../helpers/auth.js';
import { buildAdminUser, buildBaseOrigin, buildUniqueName, buildUniqueOrigin, createTestSite, installClipboardStub, openDashboard, selectSite } from '../helpers/fixtures.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);
const baseOrigin = buildBaseOrigin(config);
const escapedBaseOrigin = baseOrigin.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

let site;

function buildAdminCookie() {
  return buildSessionCookie(config, adminUser);
}

test.beforeAll(async () => {
  site = await createTestSite(config, buildAdminCookie(), {
    name: buildUniqueName('Snippet Site'),
    allowedOrigin: buildUniqueOrigin('snippet'),
    ownerEmail: config.adminEmail
  });
});

test('widget snippet placeholder when no site selected', async ({ page }) => {
  await openDashboard(page, config, adminUser, { allowEmptySites: true });
  await page.locator('#new-site-button').click();
  await expect(page.locator('#widget-snippet')).toHaveValue('Save the site to generate a widget snippet.');
});

test('widget snippet uses base origin', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#widget-snippet')).toHaveValue(new RegExp(`${escapedBaseOrigin}/widget\\.js`));
});

test('subscribe snippet uses base origin', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#subscribe-widget-snippet')).toHaveValue(new RegExp(`${escapedBaseOrigin}/subscribe\\.js`));
});

test('traffic snippet uses base origin', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#traffic-widget-snippet')).toHaveValue(new RegExp(`${escapedBaseOrigin}/pixel\\.js`));
});

test('widget snippet includes site id', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#widget-snippet')).toHaveValue(new RegExp(`site_id=${site.id}`));
});

test('subscribe snippet includes site id', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#subscribe-widget-snippet')).toHaveValue(new RegExp(`site_id=${site.id}`));
});

test('traffic snippet includes site id', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#traffic-widget-snippet')).toHaveValue(new RegExp(`site_id=${site.id}`));
});

test('customer snippets do not expose api_origin', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#widget-snippet')).not.toHaveValue(/api_origin=/);
  await expect(page.locator('#subscribe-widget-snippet')).not.toHaveValue(/api_origin=/);
  await expect(page.locator('#traffic-widget-snippet')).not.toHaveValue(/api_origin=/);
});

test('copy widget snippet updates button label', async ({ page }) => {
  await openDashboard(page, config, adminUser, { clipboard: true });
  await selectSite(page, site.id);
  await expect(page.locator('#copy-widget-snippet')).toBeEnabled();
  await page.locator('#copy-widget-snippet').click();
  await expect(page.locator('#copy-widget-snippet')).toContainText('Snippet copied');
});

test('copy subscribe snippet updates button label', async ({ page }) => {
  await openDashboard(page, config, adminUser, { clipboard: true });
  await selectSite(page, site.id);
  await page.locator('#dashboard-section-tab-subscriptions').click();
  await expect(page.locator('[data-widget-card="subscribe"]')).toBeVisible();
  await expect(page.locator('#copy-subscribe-widget-snippet')).toBeEnabled();
  await page.locator('#copy-subscribe-widget-snippet').click();
  await expect(page.locator('#copy-subscribe-widget-snippet')).toContainText('Snippet copied');
});

test('copy traffic snippet updates button label', async ({ page }) => {
  await openDashboard(page, config, adminUser, { clipboard: true });
  await selectSite(page, site.id);
  await page.locator('#dashboard-section-tab-traffic').click();
  await expect(page.locator('[data-widget-card="traffic"]')).toBeVisible();
  await expect(page.locator('#copy-traffic-widget-snippet')).toBeEnabled();
  await page.locator('#copy-traffic-widget-snippet').click();
  await expect(page.locator('#copy-traffic-widget-snippet')).toContainText('Snippet copied');
});
