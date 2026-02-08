// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildSessionCookie } from '../helpers/auth.js';
import { buildAdminUser, buildUniqueName, buildUniqueOrigin, createTestSite, openDashboard, selectSite } from '../helpers/fixtures.js';
import { createFeedback } from '../helpers/api.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);
const cookie = buildSessionCookie(config, adminUser);

async function createFeedbackSite() {
  return createTestSite(config, cookie, {
    name: buildUniqueName('Feedback Site'),
    allowedOrigin: buildUniqueOrigin('feedback'),
    ownerEmail: config.adminEmail
  });
}

test('feedback table prompts for site selection', async ({ page }) => {
  await openDashboard(page, config, adminUser, { allowEmptySites: true });
  await page.locator('#new-site-button').click();
  await expect(page.locator('#feedback-table-body')).toContainText('Select a site to see details');
});

test('feedback table shows empty state after selection', async ({ page }) => {
  const site = await createFeedbackSite();
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#edit-site-name')).toHaveValue(site.name);
  await expect(page.locator('#feedback-table-body')).toContainText('No feedback yet');
});

test('feedback table lists messages', async ({ page }) => {
  const site = await createFeedbackSite();
  await createFeedback(config, site, { contact: 'alpha@example.com', message: 'Alpha feedback' });
  await createFeedback(config, site, { contact: 'beta@example.com', message: 'Beta feedback' });
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#edit-site-name')).toHaveValue(site.name);
  await expect(page.locator('#feedback-table-body')).toContainText('Alpha feedback');
  await expect(page.locator('#feedback-table-body')).toContainText('Beta feedback');
});

test('feedback count badge reflects totals', async ({ page }) => {
  const site = await createFeedbackSite();
  await createFeedback(config, site, { contact: 'one@example.com', message: 'One' });
  await createFeedback(config, site, { contact: 'two@example.com', message: 'Two' });
  await createFeedback(config, site, { contact: 'three@example.com', message: 'Three' });
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#edit-site-name')).toHaveValue(site.name);
  await expect(page.locator('#feedback-count')).toContainText('3');
});

test('feedback search filters messages', async ({ page }) => {
  const site = await createFeedbackSite();
  await createFeedback(config, site, { contact: 'alpha@example.com', message: 'Alpha search' });
  await createFeedback(config, site, { contact: 'beta@example.com', message: 'Beta search' });
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#edit-site-name')).toHaveValue(site.name);
  await expect(page.locator('#feedback-table-body')).toContainText('Alpha search');
  await expect(page.locator('#feedback-table-body')).toContainText('Beta search');
  await page.locator('#messages-search-toggle-button').click();
  await page.locator('#messages-search-input').fill('Alpha');
  await expect(page.locator('#feedback-table-body')).toContainText('Alpha search');
  await expect(page.locator('#feedback-table-body')).not.toContainText('Beta search');
});

test('feedback search shows empty state', async ({ page }) => {
  const site = await createFeedbackSite();
  await createFeedback(config, site, { contact: 'alpha@example.com', message: 'Alpha search' });
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#edit-site-name')).toHaveValue(site.name);
  await expect(page.locator('#feedback-table-body')).toContainText('Alpha search');
  await page.locator('#messages-search-toggle-button').click();
  await page.locator('#messages-search-input').fill('NoMatch');
  await expect(page.locator('#feedback-table-body')).toContainText('No feedback matches your search');
});

test('refresh button reports success', async ({ page }) => {
  const site = await createFeedbackSite();
  await createFeedback(config, site, { contact: 'refresh@example.com', message: 'Refresh me' });
  await openDashboard(page, config, adminUser);
  await selectSite(page, site.id);
  await expect(page.locator('#edit-site-name')).toHaveValue(site.name);
  await expect(page.locator('#feedback-table-body')).toContainText('Refresh me');
  await page.locator('#refresh-messages-button').click();
  await expect(page.locator('#refresh-messages-button')).toContainText('Feedback refreshed');
});

test('feedback search toggle reveals input', async ({ page }) => {
  await openDashboard(page, config, adminUser);
  await page.locator('#messages-search-toggle-button').click();
  await expect(page.locator('#messages-search-toggle-button')).toHaveAttribute('aria-expanded', 'true');
  await expect(page.locator('#messages-search-container')).toBeVisible();
});
