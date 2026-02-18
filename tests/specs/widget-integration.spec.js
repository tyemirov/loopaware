// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildSessionCookie } from '../helpers/auth.js';
import { buildAdminUser, ensureSiteForOrigin } from '../helpers/fixtures.js';
import { updateSite } from '../helpers/api.js';
import { parseRgb } from '../helpers/browser.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);
const cookie = buildSessionCookie(config, adminUser);

let site;

async function openWidgetPage(page, siteId, options) {
  const resolvedOptions = options || {};
  const basePath = resolvedOptions.dark ? '/widget-integration-dark/' : '/widget-integration/';
  const params = new URLSearchParams({ site_id: siteId });
  if (resolvedOptions.apiOrigin) {
    params.set('api_origin', resolvedOptions.apiOrigin);
  }
  await page.goto(`${basePath}?${params.toString()}`, { waitUntil: 'domcontentloaded' });
  await page.locator('#mp-feedback-bubble').waitFor();
}

test.beforeAll(async () => {
  site = await ensureSiteForOrigin(config, cookie, {
    allowedOrigin: config.baseOrigin,
    ownerEmail: config.adminEmail
  });
});

test.beforeEach(async () => {
  await updateSite(config, cookie, site.id, {
    widget_bubble_side: 'right',
    widget_bubble_bottom_offset: 16
  });
});

test('widget bubble renders on integration page', async ({ page }) => {
  await openWidgetPage(page, site.id);
  await expect(page.locator('#mp-feedback-bubble')).toBeVisible();
});

test('widget panel opens on bubble click', async ({ page }) => {
  await openWidgetPage(page, site.id);
  await page.locator('#mp-feedback-bubble').click();
  await expect(page.locator('#mp-feedback-panel')).toBeVisible();
});

test('widget close button hides panel', async ({ page }) => {
  await openWidgetPage(page, site.id);
  await page.locator('#mp-feedback-bubble').click();
  await page.locator('button[aria-label="Close feedback panel"]').click();
  await expect(page.locator('#mp-feedback-panel')).toBeHidden();
});

test('widget submission shows success message', async ({ page }) => {
  await openWidgetPage(page, site.id);
  await page.locator('#mp-feedback-bubble').click();
  const contactInput = page.locator('#mp-feedback-contact');
  const messageInput = page.locator('#mp-feedback-message');
  await contactInput.fill('widget@example.com');
  await messageInput.fill('Widget feedback');
  await expect(contactInput).toHaveValue('widget@example.com');
  await expect(messageInput).toHaveValue('Widget feedback');
  const feedbackResponse = page.waitForResponse((response) => response.url().includes('/api/feedback') && response.status() === 200);
  await page.locator('#mp-feedback-panel button:has-text("Send")').click();
  await feedbackResponse;
  await expect(page.locator('#mp-feedback-panel')).toContainText('Thanks! Sent.');
});

test('widget keeps api_origin path prefixes for config and feedback', async ({ page }) => {
  const apiOriginWithPath = `${config.baseOrigin}/app`;
  let widgetConfigRequestCaptured = false;
  let feedbackRequestCaptured = false;

  await page.route('**/app/api/widget-config**', async (route) => {
    const requestURL = new URL(route.request().url());
    if (requestURL.searchParams.get('site_id') === site.id) {
      widgetConfigRequestCaptured = true;
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        site_id: site.id,
        widget_bubble_side: 'right',
        widget_bubble_bottom_offset: 16
      })
    });
  });

  await page.route('**/app/api/feedback', async (route) => {
    if (route.request().method() === 'POST') {
      const payload = route.request().postDataJSON();
      if (payload && payload.site_id === site.id) {
        feedbackRequestCaptured = true;
      }
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'ok' })
    });
  });

  await openWidgetPage(page, site.id, { apiOrigin: apiOriginWithPath });
  await page.locator('#mp-feedback-bubble').click();
  await page.locator('#mp-feedback-contact').fill('widget@example.com');
  await page.locator('#mp-feedback-message').fill('Widget feedback');
  await page.locator('#mp-feedback-panel button:has-text("Send")').click();

  await expect.poll(() => widgetConfigRequestCaptured).toBe(true);
  await expect.poll(() => feedbackRequestCaptured).toBe(true);
  await expect(page.locator('#mp-feedback-panel')).toContainText('Thanks! Sent.');
});

test('widget branding link uses expected label and href', async ({ page }) => {
  await openWidgetPage(page, site.id);
  await page.locator('#mp-feedback-bubble').click();
  const brandingLink = page.locator('#mp-feedback-branding a');
  await expect(brandingLink).toHaveText('Marco Polo Research Lab');
  await expect(brandingLink).toHaveAttribute('href', 'https://mprlab.com');
});

test('widget uses light theme bubble color', async ({ page }) => {
  await openWidgetPage(page, site.id);
  const bubbleColor = await page.locator('#mp-feedback-bubble').evaluate((element) => getComputedStyle(element).backgroundColor);
  expect(parseRgb(bubbleColor)).toEqual({ red: 13, green: 110, blue: 253 });
});

test('widget uses dark theme bubble color', async ({ page }) => {
  await openWidgetPage(page, site.id, { dark: true });
  const bubbleColor = await page.locator('#mp-feedback-bubble').evaluate((element) => getComputedStyle(element).backgroundColor);
  expect(parseRgb(bubbleColor)).toEqual({ red: 77, green: 171, blue: 247 });
});

test('widget placement honors left side', async ({ page }) => {
  await updateSite(config, cookie, site.id, {
    widget_bubble_side: 'left',
    widget_bubble_bottom_offset: 48
  });
  await openWidgetPage(page, site.id);
  const style = await page.locator('#mp-feedback-bubble').evaluate((element) => ({ left: element.style.left, right: element.style.right }));
  expect(style.left).toBe('16px');
  expect(style.right).toBe('');
});

test('widget placement applies custom bottom offset', async ({ page }) => {
  await updateSite(config, cookie, site.id, {
    widget_bubble_side: 'left',
    widget_bubble_bottom_offset: 48
  });
  await openWidgetPage(page, site.id);
  const bottom = await page.locator('#mp-feedback-bubble').evaluate((element) => element.style.bottom);
  expect(bottom).toBe('48px');
});

test('widget panel offset tracks bubble offset', async ({ page }) => {
  await updateSite(config, cookie, site.id, {
    widget_bubble_side: 'left',
    widget_bubble_bottom_offset: 48
  });
  await openWidgetPage(page, site.id);
  const panelBottom = await page.locator('#mp-feedback-panel').evaluate((element) => element.style.bottom);
  expect(panelBottom).toBe('112px');
});
