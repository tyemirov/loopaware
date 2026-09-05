// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildSessionCookie } from '../helpers/auth.js';
import { buildAdminUser, buildBaseOrigin, buildUniqueEmail, buildUniqueName, buildUniqueOrigin, createTestSite, openDashboard, selectSite } from '../helpers/fixtures.js';
import { apiRequest, listSites, updateSite } from '../helpers/api.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config, { displayName: 'Admin Example' });
const baseOrigin = buildBaseOrigin(config);

let primarySite;
let searchSite;

function buildAdminCookie() {
  return buildSessionCookie(config, adminUser);
}

async function createScrollableSiteOwner() {
  const owner = buildAdminUser(config, { email: buildUniqueEmail('scroll-owner') });
  for (let index = 0; index < 12; index += 1) {
    await createTestSite(config, buildAdminCookie(), {
      name: buildUniqueName(`Scroll Site ${index}`),
      allowedOrigin: buildUniqueOrigin('scroll-site'),
      ownerEmail: owner.email
    });
  }
  return owner;
}

test.beforeAll(async () => {
  const primaryOrigin = buildUniqueOrigin('primary');
  const searchOrigin = buildUniqueOrigin('search');
  const primaryName = buildUniqueName('Primary Site');
  const searchName = buildUniqueName('Searchable Site');
  primarySite = await createTestSite(config, buildAdminCookie(), {
    name: primaryName,
    allowedOrigin: primaryOrigin,
    ownerEmail: config.adminEmail
  });
  searchSite = await createTestSite(config, buildAdminCookie(), {
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

test('site selection and automatic save preserve the scrolled list position', async ({ page }) => {
  const owner = await createScrollableSiteOwner();
  await openDashboard(page, config, owner);
  const list = page.locator('#sites-list');
  const rows = list.locator('[data-site-id]');
  await expect(rows).toHaveCount(12);
  await list.evaluate((element) => { element.scrollTop = element.scrollHeight; });
  const scrollTop = await list.evaluate((element) => element.scrollTop);
  expect(scrollTop).toBeGreaterThan(0);

  for (const index of [11, 10]) {
    const row = rows.nth(index);
    const siteName = await row.locator('span').innerText();
    await row.click();
    await expect(page.locator('#edit-site-name')).toHaveValue(siteName);
    await expect(row).toHaveClass(/\bactive\b/);
    await expect.poll(() => list.evaluate((element) => element.scrollTop)).toBe(scrollTop);
    await expect(row).toBeInViewport({ ratio: 1 });
  }

  const updatedName = buildUniqueName('Updated Scroll Site');
  await page.locator('#edit-site-name').fill(updatedName);
  await expect(rows.nth(10)).toContainText(updatedName);
  await expect(rows.nth(10)).toHaveClass(/\bactive\b/);
  await expect.poll(() => list.evaluate((element) => element.scrollTop)).toBe(scrollTop);
  await expect(rows.nth(10)).toBeInViewport({ ratio: 1 });
});

test('creating a site reveals its selected row from a scrolled list', async ({ page }) => {
  const owner = await createScrollableSiteOwner();
  await openDashboard(page, config, owner);
  const list = page.locator('#sites-list');
  await expect(list.locator('[data-site-id]')).toHaveCount(12);
  await list.evaluate((element) => { element.scrollTop = element.scrollHeight; });
  expect(await list.evaluate((element) => element.scrollTop)).toBeGreaterThan(0);

  await page.locator('#new-site-button').click();
  const name = buildUniqueName('Created From Scrolled List');
  await page.locator('#edit-site-name').fill(name);
  await page.locator('#edit-site-origin').fill(buildUniqueOrigin('created-scroll-site'));
  await page.locator('#edit-site-owner').fill(owner.email);
  const selectedRow = list.locator('[data-site-id].active');
  await expect(selectedRow).toContainText(name);
  await expect(page.locator('#edit-site-name')).toHaveValue(name);
  await expect(selectedRow).toBeInViewport({ ratio: 1 });
});

test('filtering a scrolled list reveals the replacement selection', async ({ page }) => {
  const owner = await createScrollableSiteOwner();
  await openDashboard(page, config, owner);
  const list = page.locator('#sites-list');
  const rows = list.locator('[data-site-id]');
  await expect(rows).toHaveCount(12);
  await rows.last().click();
  const excludedName = buildUniqueName('Excluded Site');
  await page.locator('#edit-site-name').fill(excludedName);
  await expect(rows.last()).toContainText(excludedName);
  expect(await list.evaluate((element) => element.scrollTop)).toBeGreaterThan(0);

  await page.locator('#site-search-toggle-button').click();
  await page.locator('#site-search-input').fill('Scroll Site');
  await expect(rows).toHaveCount(11);
  await expect(rows.first()).toHaveClass(/\bactive\b/);
  await expect(page.locator('#edit-site-name')).toHaveValue(await rows.first().locator('span').innerText());
  await expect(rows.first()).toBeInViewport({ ratio: 1 });
});

test('clearing or hiding search reveals the selection in the full list', async ({ page }) => {
  const owner = await createScrollableSiteOwner();
  await openDashboard(page, config, owner);
  const list = page.locator('#sites-list');
  const rows = list.locator('[data-site-id]');
  await expect(rows).toHaveCount(12);
  await rows.last().click();
  const selectedName = await page.locator('#edit-site-name').inputValue();
  await page.locator('#site-search-toggle-button').click();
  const search = page.locator('#site-search-input');

  for (const action of ['clear', 'hide']) {
    await search.fill(selectedName);
    await expect(rows).toHaveCount(1);
    if (action === 'clear') {
      await search.fill('');
    } else {
      await search.press('Escape');
    }
    await expect(rows).toHaveCount(12);
    await expect(page.locator('#edit-site-name')).toHaveValue(selectedName);
    await expect(rows.last()).toHaveClass(/\bactive\b/);
    await expect(rows.last()).toBeInViewport({ ratio: 1 });
  }
});

test('deleting a site reveals the replacement selection from a scrolled list', async ({ page }) => {
  const owner = await createScrollableSiteOwner();
  await openDashboard(page, config, owner);
  const list = page.locator('#sites-list');
  const rows = list.locator('[data-site-id]');
  await expect(rows).toHaveCount(12);
  await rows.last().click();
  const siteName = await page.locator('#edit-site-name').inputValue();
  expect(await list.evaluate((element) => element.scrollTop)).toBeGreaterThan(0);

  await page.locator('#delete-site-button').click();
  await page.locator('#delete-site-confirm-name').fill(siteName);
  await page.locator('#delete-site-confirm-button').click();
  await expect(page.locator('#delete-site-modal')).toBeHidden();
  await expect(rows).toHaveCount(11);
  await expect(rows.first()).toHaveClass(/\bactive\b/);
  await expect(page.locator('#edit-site-name')).toHaveValue(await rows.first().locator('span').innerText());
  await expect(rows.first()).toBeInViewport({ ratio: 1 });
});

test('delete site button enables after selection', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await page.locator('#new-site-button').click();
  await expect(page.locator('#delete-site-button')).toBeDisabled();
  await selectSite(page, primarySite.id);
  await expect(page.locator('#delete-site-button')).toBeEnabled();
});

test('site team member can be added and removed from dashboard', async ({ page }) => {
  const teamEmail = buildUniqueEmail('dashboard-team');
  await openDashboard(page, config, adminUser);
  await selectSite(page, primarySite.id);
  await expect(page.locator('#dashboard-section-tab-admin')).toBeVisible();
  await page.locator('#dashboard-section-tab-admin').click();
  await expect(page.locator('#site-team-card')).toBeVisible();
  await page.locator('#team-member-email').fill(teamEmail);
  await page.locator('#add-team-member-button').click();
  await expect(page.locator('#team-members-table-body')).toContainText(teamEmail.toLowerCase());
  await page.locator(`#team-members-table-body button[data-team-member-email="${teamEmail.toLowerCase()}"]`).click();
  await expect(page.locator('#team-members-table-body')).not.toContainText(teamEmail.toLowerCase());
});

test('assigned team member sees site details as read-only', async ({ page }) => {
  const site = await createTestSite(config, buildAdminCookie(), {
    name: buildUniqueName('Team Read Only'),
    allowedOrigin: buildUniqueOrigin('team-read-only'),
    ownerEmail: config.adminEmail
  });
  const teamEmail = buildUniqueEmail('team-readonly');
  const teamUser = buildAdminUser(config, {
    email: teamEmail,
    displayName: 'Team Read Only'
  });
  const { response, payload } = await apiRequest({
    baseURL: config.baseURL,
    cookie: buildAdminCookie(),
    path: `/api/sites/${site.id}/team`,
    method: 'POST',
    body: { email: teamEmail }
  });
  expect(response.status, JSON.stringify(payload)).toBe(200);

  await openDashboard(page, config, teamUser);
  await selectSite(page, site.id);
  await expect(page.locator(`#sites-list [data-site-id="${site.id}"]`)).toContainText(site.name);
  await expect(page.locator('#edit-site-name')).toBeDisabled();
  await expect(page.locator('#edit-site-origin')).toBeDisabled();
  await expect(page.locator('#edit-site-owner')).toBeDisabled();
  await expect(page.locator('#delete-site-button')).toBeDisabled();
  await expect(page.locator('#dashboard-section-tab-admin')).toBeHidden();
  await expect(page.locator('#site-team-card')).toBeHidden();
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

test('validation rejects hiding both widget feedback inputs', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await selectSite(page, primarySite.id);
  await page.locator('#widget-show-message-input').uncheck();
  await page.locator('#widget-show-sentiment-buttons').uncheck();
  await expect(page.locator('#site-status')).toContainText('Enable the message input, sentiment buttons, or both');
});

test('feedback widget accent color persists', async ({ page }) => {
  await updateSite(config, buildAdminCookie(), primarySite.id, { widget_accent_color: '#0d6efd' });
  await openDashboard(page, config, adminUser);
  await selectSite(page, primarySite.id);
  await page.locator('#widget-accent-color').fill('#22aa77');
  await expect.poll(async () => {
    const payload = await listSites(config, buildAdminCookie());
    const refreshedSite = Array.isArray(payload.sites) ? payload.sites.find((entry) => entry.id === primarySite.id) : null;
    return refreshedSite ? refreshedSite.widget_accent_color : '';
  }).toBe('#22aa77');

  await openDashboard(page, config, adminUser);
  await selectSite(page, primarySite.id);
  await expect(page.locator('#widget-accent-color')).toHaveValue('#22aa77');
});
