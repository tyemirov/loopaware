// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildSessionCookie } from '../helpers/auth.js';
import { buildAdminUser, buildBaseOrigin, buildUniqueName, buildUniqueOrigin, createTestSite, openDashboard, selectSite } from '../helpers/fixtures.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config, { displayName: 'Admin Example' });
const baseOrigin = buildBaseOrigin(config);

let cookie;
let primarySite;
let searchSite;

test.beforeAll(async () => {
  cookie = buildSessionCookie(config, adminUser);
  const primaryOrigin = buildUniqueOrigin('primary');
  const searchOrigin = buildUniqueOrigin('search');
  const primaryName = buildUniqueName('Primary Site');
  const searchName = buildUniqueName('Searchable Site');
  primarySite = await createTestSite(config, cookie, {
    name: primaryName,
    allowedOrigin: primaryOrigin,
    ownerEmail: config.adminEmail
  });
  searchSite = await createTestSite(config, cookie, {
    name: searchName,
    allowedOrigin: searchOrigin,
    ownerEmail: config.adminEmail
  });
});

test('lists existing sites in sidebar', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await expect(page.locator(`#sites-list [data-site-id="${primarySite.id}"]`)).toContainText(primarySite.name);
  await expect(page.locator(`#sites-list [data-site-id="${searchSite.id}"]`)).toContainText(searchSite.name);
});

test('selecting a site populates form fields', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await selectSite(page, primarySite.id);
  await expect(page.locator('#edit-site-name')).toHaveValue(primarySite.name);
  await expect(page.locator('#edit-site-origin')).toHaveValue(primarySite.allowed_origin);
  await expect(page.locator('#edit-site-owner')).toHaveValue(primarySite.owner_email);
});

test('site created timestamp renders when selected', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await selectSite(page, primarySite.id);
  await expect(page.locator('#site-created-at')).not.toHaveText('');
});

test('delete site button enables after selection', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await page.locator('#new-site-button').click();
  await expect(page.locator('#delete-site-button')).toBeDisabled();
  await selectSite(page, primarySite.id);
  await expect(page.locator('#delete-site-button')).toBeEnabled();
});

test('site search filters matching entries', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await page.locator('#site-search-toggle-button').click();
  await page.locator('#site-search-input').fill('Searchable');
  await expect(page.locator(`#sites-list [data-site-id="${searchSite.id}"]`)).toBeVisible();
});

test('site search shows empty state on no match', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await page.locator('#site-search-toggle-button').click();
  await page.locator('#site-search-input').fill('NoMatch');
  await expect(page.locator('#empty-sites-message')).toContainText('No sites match your search');
});

test('create new site via form', async ({ page }) => {
  const uniqueOrigin = buildUniqueOrigin('new-site');
  const uniqueName = buildUniqueName('New Site');
  await openDashboard(page, config, adminUser);
  await page.locator('#new-site-button').click();
  await page.locator('#edit-site-name').fill(uniqueName);
  await page.locator('#edit-site-origin').fill(uniqueOrigin);
  await page.locator('#edit-site-owner').fill(config.adminEmail);
  await expect(page.locator('#sites-list')).toContainText(uniqueName);
  await expect(page.locator('#edit-site-name')).toHaveValue(uniqueName);
});

test('validation requires site name', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await page.locator('#new-site-button').click();
  await page.locator('#edit-site-origin').fill(baseOrigin);
  await page.locator('#edit-site-owner').fill(config.adminEmail);
  await expect(page.locator('#site-status')).toContainText('Site name is required');
});

test('validation requires valid origin', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await page.locator('#new-site-button').click();
  await page.locator('#edit-site-name').fill('Origin Missing Protocol');
  await page.locator('#edit-site-origin').fill('example.com');
  await page.locator('#edit-site-owner').fill(config.adminEmail);
  await expect(page.locator('#site-status')).toContainText('Allowed origins must include protocol');
});

test('validation requires valid owner email', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await page.locator('#new-site-button').click();
  await page.locator('#edit-site-name').fill('Invalid Owner');
  await page.locator('#edit-site-origin').fill(baseOrigin);
  await page.locator('#edit-site-owner').fill('not-an-email');
  await expect(page.locator('#site-status')).toContainText('Provide a valid owner email');
});

test('validation rejects invalid widget offset', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await page.locator('#new-site-button').click();
  await page.locator('#edit-site-name').fill('Invalid Offset');
  await page.locator('#edit-site-origin').fill(baseOrigin);
  await page.locator('#edit-site-owner').fill(config.adminEmail);
  await page.locator('#widget-placement-bottom-offset').fill('999');
  await expect(page.locator('#site-status')).toContainText('Provide a whole number between 0 and 240');
});
