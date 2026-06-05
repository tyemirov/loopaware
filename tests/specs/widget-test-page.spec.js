// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildSessionCookie } from '../helpers/auth.js';
import { buildAdminUser, buildUniqueName, buildUniqueOrigin, createTestSite } from '../helpers/fixtures.js';
import { listSites, updateSite } from '../helpers/api.js';
import { applySessionCookie, parseRgb } from '../helpers/browser.js';
import { installExternalAssetStubs, waitForExternalAssetStubsToSettle } from '../helpers/externalAssets.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);

let site;

function buildAdminCookie() {
  return buildSessionCookie(config, adminUser);
}

test.beforeAll(async () => {
  site = await createTestSite(config, buildAdminCookie(), {
    name: buildUniqueName('Widget Test Page'),
    allowedOrigin: buildUniqueOrigin('widget-test-page'),
    ownerEmail: config.adminEmail
  });
});

test('widget test page saves and previews feedback bubble accent color', async ({ page }) => {
  await updateSite(config, buildAdminCookie(), site.id, {
    widget_accent_color: '#0d6efd'
  });
  await installExternalAssetStubs(page, config);
  await applySessionCookie(page.context(), config, adminUser);
  await page.goto(`/app/widget-test?site_id=${encodeURIComponent(site.id)}`, { waitUntil: 'domcontentloaded' });
  await waitForExternalAssetStubsToSettle(page);
  await page.locator('#mp-feedback-bubble').waitFor();
  await expect(page.locator('#widget-test-accent-color')).toHaveValue('#0d6efd');

  await page.locator('#widget-test-accent-color').fill('#ff5500');
  await expect(page.locator('#widget-test-summary-accent')).toHaveText('#ff5500');
  const bubbleColor = await page.locator('#mp-feedback-bubble').evaluate((element) => getComputedStyle(element).backgroundColor);
  expect(parseRgb(bubbleColor)).toEqual({ red: 255, green: 85, blue: 0 });

  const saveRequest = page.waitForRequest((request) => request.url().includes(`/api/sites/${site.id}`) && request.method() === 'PATCH');
  await page.locator('#widget-test-save').click();
  const request = await saveRequest;
  expect(JSON.parse(request.postData() || '{}')).toMatchObject({
    widget_accent_color: '#ff5500'
  });
  await expect.poll(async () => {
    const payload = await listSites(config, buildAdminCookie());
    const refreshedSite = Array.isArray(payload.sites) ? payload.sites.find((entry) => entry.id === site.id) : null;
    return refreshedSite ? refreshedSite.widget_accent_color : '';
  }).toBe('#ff5500');
});
