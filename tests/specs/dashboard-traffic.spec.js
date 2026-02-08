// @ts-check
import { test, expect } from '@playwright/test';
import * as crypto from 'node:crypto';
import { resolveTestConfig } from '../helpers/config.js';
import { buildSessionCookie } from '../helpers/auth.js';
import { buildAdminUser, buildUniqueName, buildUniqueOrigin, createTestSite, openDashboard, selectSite } from '../helpers/fixtures.js';
import { collectVisit, fetchVisitStats } from '../helpers/api.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);
const cookie = buildSessionCookie(config, adminUser);

function buildVisitorId() {
  return crypto.randomUUID();
}

async function createTrafficSite() {
  return createTestSite(config, cookie, {
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
  const stats = await fetchVisitStats(config, cookie, site.id);
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#visit-count')).toHaveText(`${stats.visit_count} visits`);
  await expect(page.locator('#unique-visitor-count')).toHaveText(`${stats.unique_visitor_count} unique`);
});
