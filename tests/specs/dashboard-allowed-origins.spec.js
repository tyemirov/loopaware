// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildSessionCookie } from '../helpers/auth.js';
import { buildAdminUser, buildUniqueName, buildUniqueOrigin, createTestSite, openDashboard, selectSite } from '../helpers/fixtures.js';
import { listSites, updateSite } from '../helpers/api.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);
const cookie = buildSessionCookie(config, adminUser);

const originTypes = [
  {
    name: 'widget',
    dataAttribute: 'data-widget-origin',
    placeholderAttribute: 'data-widget-origin-placeholder',
    addButtonAttribute: 'data-widget-origin-add',
    removeButtonAttribute: 'data-widget-origin-remove',
    apiField: 'widget_allowed_origins'
  },
  {
    name: 'subscribe',
    dataAttribute: 'data-subscribe-origin',
    placeholderAttribute: 'data-subscribe-origin-placeholder',
    addButtonAttribute: 'data-subscribe-origin-add',
    removeButtonAttribute: 'data-subscribe-origin-remove',
    apiField: 'subscribe_allowed_origins'
  },
  {
    name: 'traffic',
    dataAttribute: 'data-traffic-origin',
    placeholderAttribute: 'data-traffic-origin-placeholder',
    addButtonAttribute: 'data-traffic-origin-add',
    removeButtonAttribute: 'data-traffic-origin-remove',
    apiField: 'traffic_allowed_origins'
  }
];

let site;

async function resetAllowedOrigins() {
  await updateSite(config, cookie, site.id, {
    widget_allowed_origins: '',
    subscribe_allowed_origins: '',
    traffic_allowed_origins: ''
  });
}

async function openSite(page, section) {
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  if (section === 'subscribe') {
    await page.locator('#dashboard-section-tab-subscriptions').click();
    await expect(page.locator('#subscribe-allowed-origins-list')).toBeVisible();
  }
  if (section === 'traffic') {
    await page.locator('#dashboard-section-tab-traffic').click();
    await expect(page.locator('#traffic-allowed-origins-list')).toBeVisible();
  }
}

async function loadSite() {
  const payload = await listSites(config, cookie);
  const sites = Array.isArray(payload?.sites) ? payload.sites : [];
  const match = sites.find((entry) => entry.id === site.id);
  if (!match) {
    throw new Error('missing_site');
  }
  return match;
}

test.beforeAll(async () => {
  site = await createTestSite(config, cookie, {
    name: buildUniqueName('Allowed Origins Site'),
    allowedOrigin: buildUniqueOrigin('allowed-origins'),
    ownerEmail: config.adminEmail
  });
});

test.beforeEach(async () => {
  await resetAllowedOrigins();
});

for (const originType of originTypes) {
  test(`${originType.name} allowed origins add persists`, async ({ page }) => {
    const originValue = `http://${originType.name}-${Date.now()}.example.com`;
    await openSite(page, originType.name);
    await page.locator(`input[${originType.placeholderAttribute}]`).fill(originValue);
    await page.locator(`button[${originType.addButtonAttribute}]`).click();
    const existingInputs = page.locator(`input[${originType.dataAttribute}]:not([${originType.placeholderAttribute}])`);
    await expect(existingInputs).toHaveCount(1);
    await expect(existingInputs.first()).toHaveValue(originValue);
    await expect(page.locator('#site-status')).toContainText('Site updated');
    const updated = await loadSite();
    expect(String(updated[originType.apiField] || '')).toBe(originValue);
  });

  test(`${originType.name} allowed origins rehydrate`, async ({ page }) => {
    const firstOrigin = `http://${originType.name}-first-${Date.now()}.example.com`;
    const secondOrigin = `http://${originType.name}-second-${Date.now()}.example.com`;
    await updateSite(config, cookie, site.id, {
      [originType.apiField]: `${firstOrigin} ${secondOrigin}`
    });
    await openSite(page, originType.name);
    const existingInputs = page.locator(`input[${originType.dataAttribute}]:not([${originType.placeholderAttribute}])`);
    await expect(existingInputs).toHaveCount(2);
    await expect(existingInputs.nth(0)).toHaveValue(firstOrigin);
    await expect(existingInputs.nth(1)).toHaveValue(secondOrigin);
  });

  test(`${originType.name} allowed origins remove persists`, async ({ page }) => {
    const firstOrigin = `http://${originType.name}-remove-a-${Date.now()}.example.com`;
    const secondOrigin = `http://${originType.name}-remove-b-${Date.now()}.example.com`;
    await updateSite(config, cookie, site.id, {
      [originType.apiField]: `${firstOrigin} ${secondOrigin}`
    });
    await openSite(page, originType.name);
    const removeButtons = page.locator(`button[${originType.removeButtonAttribute}]`);
    await expect(removeButtons).toHaveCount(2);
    await removeButtons.first().click();
    await expect(page.locator('#site-status')).toContainText('Site updated');
    const updated = await loadSite();
    expect(String(updated[originType.apiField] || '')).toBe(secondOrigin);
  });

  test(`${originType.name} allowed origins invalid blocks save`, async ({ page }) => {
    await openSite(page, originType.name);
    await page.locator(`input[${originType.placeholderAttribute}]`).fill('invalid.example.com');
    await page.locator(`button[${originType.addButtonAttribute}]`).click();
    await expect(page.locator('#site-status')).toContainText('Allowed origins must include protocol and hostname');
    const updated = await loadSite();
    expect(String(updated[originType.apiField] || '')).toBe('');
  });
}
