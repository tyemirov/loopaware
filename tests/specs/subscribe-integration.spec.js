// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildSessionCookie } from '../helpers/auth.js';
import { buildAdminUser, ensureSiteForOrigin } from '../helpers/fixtures.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);
const cookie = buildSessionCookie(config, adminUser);
const QUERY_TEXT_CASES = Object.freeze([
  { name: 'cta label', params: { cta: 'Save = Ship' }, assert: async (page) => {
    await expect(page.locator('#mp-subscribe-submit')).toContainText('Save = Ship');
  } },
  { name: 'success message', params: { success: 'A=B' }, assert: async (page) => {
    await page.route('**/public/subscriptions', async (route) => {
      await route.fulfill({
        status: 201,
        contentType: 'application/json; charset=utf-8',
        body: JSON.stringify({ status: 'created' })
      });
    });
    await page.locator('#mp-subscribe-email').fill(`user-${Date.now()}@example.com`);
    await page.locator('#mp-subscribe-name').fill('Example User');
    await page.locator('#mp-subscribe-submit').click();
    await expect(page.locator('#mp-subscribe-status')).toContainText('A=B');
  } },
  { name: 'error message', params: { error: 'Try=A=B' }, assert: async (page) => {
    await page.route('**/public/subscriptions', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json; charset=utf-8',
        body: JSON.stringify({ error: 'server_error' })
      });
    });
    await page.locator('#mp-subscribe-email').fill(`user-${Date.now()}@example.com`);
    await page.locator('#mp-subscribe-name').fill('Example User');
    await page.locator('#mp-subscribe-submit').click();
    await expect(page.locator('#mp-subscribe-status')).toContainText('Try=A=B');
  } }
]);

let site;

async function openSubscribePage(page, params) {
  const search = new URLSearchParams({ site_id: site.id });
  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null) {
        search.set(key, String(value));
      }
    });
  }
  await page.goto(`/subscribe-demo/?${search.toString()}`, { waitUntil: 'domcontentloaded' });
  await expect(page.locator('#subscribe-demo-status')).toContainText('Loaded');
  await page.locator('#mp-subscribe-form').waitFor();
}

test.beforeAll(async () => {
  site = await ensureSiteForOrigin(config, cookie, {
    allowedOrigin: config.baseOrigin,
    ownerEmail: config.adminEmail
  });
});

test('inline subscribe form renders email input', async ({ page }) => {
  await openSubscribePage(page);
  await expect(page.locator('#mp-subscribe-email')).toBeVisible();
});

test('name input hides when name_field is false', async ({ page }) => {
  await openSubscribePage(page, { name_field: 'false' });
  await expect(page.locator('#mp-subscribe-name')).toHaveCount(0);
});

for (const queryTextCase of QUERY_TEXT_CASES) {
  test(`query-string ${queryTextCase.name} preserves equals signs`, async ({ page }) => {
    await openSubscribePage(page, queryTextCase.params);
    await queryTextCase.assert(page);
  });
}

test('successful subscribe shows default success message', async ({ page }) => {
  await openSubscribePage(page);
  await page.locator('#mp-subscribe-email').fill(`user-${Date.now()}@example.com`);
  await page.locator('#mp-subscribe-name').fill('Example User');
  await page.locator('#mp-subscribe-submit').click();
  await expect(page.locator('#mp-subscribe-status')).toContainText("You're on the list!");
});

test('already subscribed flow shows message', async ({ page }) => {
  await openSubscribePage(page);
  const email = `repeat-${Date.now()}@example.com`;
  await page.locator('#mp-subscribe-email').fill(email);
  await page.locator('#mp-subscribe-submit').click();
  await expect(page.locator('#mp-subscribe-status')).toContainText("You're on the list!");
  await page.locator('#mp-subscribe-email').fill(email);
  await page.locator('#mp-subscribe-submit').click();
  await expect(page.locator('#mp-subscribe-status')).toContainText("You're already subscribed!");
});

test('invalid email shows validation message', async ({ page }) => {
  await openSubscribePage(page);
  await page.locator('#mp-subscribe-email').fill('not-an-email');
  await page.locator('#mp-subscribe-submit').click();
  await expect(page.locator('#mp-subscribe-status')).toContainText('Please enter a valid email.');
});

test('target parameter renders inside container', async ({ page }) => {
  await openSubscribePage(page, { target: 'subscribe-demo' });
  await expect(page.locator('#subscribe-demo #mp-subscribe-form')).toHaveCount(1);
});
