// @ts-check
import * as crypto from 'node:crypto';
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildSessionCookie } from '../helpers/auth.js';
import { collectVisit } from '../helpers/api.js';
import { applySessionCookie } from '../helpers/browser.js';
import { installExternalAssetStubs, waitForExternalAssetStubsToSettle } from '../helpers/externalAssets.js';
import { buildAdminUser, buildUniqueName, buildUniqueOrigin, createTestSite } from '../helpers/fixtures.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);
const poisonedPath = '/<img src=x onerror=document.documentElement.dataset.auditXss=1>';

let site;

function buildAdminCookie() {
  return buildSessionCookie(config, adminUser);
}

test.beforeAll(async () => {
  site = await createTestSite(config, buildAdminCookie(), {
    name: buildUniqueName('Traffic Test Page'),
    allowedOrigin: buildUniqueOrigin('traffic-test-page'),
    ownerEmail: config.adminEmail
  });
  await collectVisit(config, site, {
    url: `${site.allowed_origin}${poisonedPath}`,
    visitorId: crypto.randomUUID()
  });
});

test('traffic test page renders stored paths as inert text', async ({ page }) => {
  await installExternalAssetStubs(page, config);
  await applySessionCookie(page.context(), config, adminUser);
  await page.goto(`/app/traffic-test?site_id=${encodeURIComponent(site.id)}`, { waitUntil: 'domcontentloaded' });
  await waitForExternalAssetStubsToSettle(page);

  const topPages = page.locator('#traffic-test-top-pages');
  await expect(topPages.locator('td.text-break')).toHaveText(poisonedPath);
  await expect(topPages.locator('img')).toHaveCount(0);
  expect(await page.evaluate(() => document.documentElement.dataset.auditXss || '')).toBe('');
});
