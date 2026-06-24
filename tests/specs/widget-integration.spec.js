// @ts-check
import { test, expect } from '@playwright/test';
import { resolveTestConfig } from '../helpers/config.js';
import { buildSessionCookie } from '../helpers/auth.js';
import { buildAdminUser, ensureSiteForOrigin } from '../helpers/fixtures.js';
import { updateSite } from '../helpers/api.js';
import { parseRgb } from '../helpers/browser.js';

const config = resolveTestConfig();
const adminUser = buildAdminUser(config);

let site;

function buildAdminCookie() {
  return buildSessionCookie(config, adminUser);
}

async function openWidgetPage(page, siteId, options) {
  const resolvedOptions = options || {};
  const basePath = resolvedOptions.dark ? '/widget-integration-dark/' : '/widget-integration/';
  const params = new URLSearchParams({ site_id: siteId });
  if (resolvedOptions.apiOrigin) {
    params.set('api_origin', resolvedOptions.apiOrigin);
  }
  await page.goto(`${basePath}?${params.toString()}`, { waitUntil: 'domcontentloaded' });
  await expect(page.locator('#widget-integration-status')).toContainText('Loaded');
  await page.locator('#mp-feedback-bubble').waitFor();
}

test.beforeAll(async () => {
  site = await ensureSiteForOrigin(config, buildAdminCookie(), {
    allowedOrigin: config.baseOrigin,
    ownerEmail: config.adminEmail
  });
});

test.beforeEach(async () => {
  await updateSite(config, buildAdminCookie(), site.id, {
    widget_bubble_side: 'right',
    widget_bubble_bottom_offset: 16,
    widget_accent_color: '#0d6efd',
    widget_show_message_input: true,
    widget_show_sentiment_buttons: true
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

test('widget sentiment buttons render as circular icon controls', async ({ page }) => {
  await openWidgetPage(page, site.id);
  await page.locator('#mp-feedback-bubble').click();
  const sentimentStyle = await page.locator('#mp-feedback-sentiment-happy').evaluate((element) => {
    const computedStyle = getComputedStyle(element);
    return {
      width: computedStyle.width,
      height: computedStyle.height,
      borderRadius: computedStyle.borderRadius,
      fontSize: computedStyle.fontSize,
      borderTopWidth: computedStyle.borderTopWidth,
      backgroundColor: computedStyle.backgroundColor
    };
  });
  expect(sentimentStyle).toEqual({
    width: '64px',
    height: '64px',
    borderRadius: '999px',
    fontSize: '48px',
    borderTopWidth: '0px',
    backgroundColor: 'rgba(0, 0, 0, 0)'
  });
});

test('widget hides message input when site disables text feedback', async ({ page }) => {
  await updateSite(config, buildAdminCookie(), site.id, {
    widget_show_message_input: false,
    widget_show_sentiment_buttons: true
  });
  await openWidgetPage(page, site.id);
  await page.locator('#mp-feedback-bubble').click();
  await expect(page.locator('#mp-feedback-message')).toHaveCount(0);
  await expect(page.locator('#mp-feedback-sentiment')).toBeVisible();
});

test('widget hides sentiment buttons when site disables sentiment feedback', async ({ page }) => {
  await updateSite(config, buildAdminCookie(), site.id, {
    widget_show_message_input: true,
    widget_show_sentiment_buttons: false
  });
  await openWidgetPage(page, site.id);
  await page.locator('#mp-feedback-bubble').click();
  await expect(page.locator('#mp-feedback-sentiment')).toHaveCount(0);
  await expect(page.locator('#mp-feedback-message')).toBeVisible();
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
  
  await contactInput.clear();
  await contactInput.fill('widget@example.com');
  await messageInput.clear();
  await messageInput.fill('Widget feedback');
  const sourceURL = page.url();
  
  await expect(contactInput).toHaveValue('widget@example.com');
  await expect(messageInput).toHaveValue('Widget feedback');
  const feedbackRequest = page.waitForRequest((request) => request.url().includes('/public/feedback') && request.method() === 'POST');
  const feedbackResponse = page.waitForResponse((response) => response.url().includes('/public/feedback') && response.status() === 200);
  await page.locator('#mp-feedback-panel button:has-text("Send")').click();
  const request = await feedbackRequest;
  await feedbackResponse;
  expect(JSON.parse(request.postData() || '{}')).toMatchObject({ contact: 'widget@example.com', message: 'Widget feedback', sentiment: '', source_url: sourceURL });
  await expect(page.locator('#mp-feedback-sentiment-happy')).toHaveAttribute('aria-pressed', 'false');
  await expect(page.locator('#mp-feedback-panel')).toContainText('Thanks! Sent.');
});

test('widget submission accepts sentiment without message', async ({ page }) => {
  await openWidgetPage(page, site.id);
  await page.locator('#mp-feedback-bubble').click();
  await page.locator('#mp-feedback-contact').fill('widget@example.com');
  await page.locator('#mp-feedback-sentiment-happy').click();
  await expect(page.locator('#mp-feedback-sentiment-happy')).toHaveAttribute('aria-pressed', 'true');
  const feedbackRequest = page.waitForRequest((request) => request.url().includes('/public/feedback') && request.method() === 'POST');
  const feedbackResponse = page.waitForResponse((response) => response.url().includes('/public/feedback') && response.status() === 200);
  await page.locator('#mp-feedback-panel button:has-text("Send")').click();
  const request = await feedbackRequest;
  await feedbackResponse;
  expect(JSON.parse(request.postData() || '{}')).toMatchObject({ contact: 'widget@example.com', message: '', sentiment: 'happy' });
  await expect(page.locator('#mp-feedback-panel')).toContainText('Thanks! Sent.');
});

test('widget submission accepts sentiment with message', async ({ page }) => {
  await openWidgetPage(page, site.id);
  await page.locator('#mp-feedback-bubble').click();
  await page.locator('#mp-feedback-contact').fill('+1 (415) 555-1212');
  await page.locator('#mp-feedback-sentiment-sad').click();
  await page.locator('#mp-feedback-message').fill('Needs work');
  const feedbackRequest = page.waitForRequest((request) => request.url().includes('/public/feedback') && request.method() === 'POST');
  const feedbackResponse = page.waitForResponse((response) => response.url().includes('/public/feedback') && response.status() === 200);
  await page.locator('#mp-feedback-panel button:has-text("Send")').click();
  const request = await feedbackRequest;
  await feedbackResponse;
  expect(JSON.parse(request.postData() || '{}')).toMatchObject({ contact: '+14155551212', message: 'Needs work', sentiment: 'sad' });
  await expect(page.locator('#mp-feedback-panel')).toContainText('Thanks! Sent.');
});

test('widget rejects api_origin values with path prefixes', async ({ page }) => {
  const apiOriginWithPath = `${config.baseOrigin}/app`;
  /** @type {string[]} */
  const consoleErrors = [];
  page.on('console', (message) => {
    if (message.type() === 'error') {
      consoleErrors.push(message.text());
    }
  });

  const params = new URLSearchParams({ site_id: site.id, api_origin: apiOriginWithPath });
  await page.goto(`/widget-integration/?${params.toString()}`, { waitUntil: 'domcontentloaded' });

  await expect.poll(() => consoleErrors.some((entry) => entry.includes('invalid api_origin'))).toBe(true);
  await expect(page.locator('#mp-feedback-bubble')).toHaveCount(0);
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

test('widget uses configured accent color on dark pages', async ({ page }) => {
  await openWidgetPage(page, site.id, { dark: true });
  const bubbleColor = await page.locator('#mp-feedback-bubble').evaluate((element) => getComputedStyle(element).backgroundColor);
  expect(parseRgb(bubbleColor)).toEqual({ red: 13, green: 110, blue: 253 });
});

test('widget applies custom accent color to bubble and send button', async ({ page }) => {
  await updateSite(config, buildAdminCookie(), site.id, {
    widget_accent_color: '#ff5500'
  });
  await openWidgetPage(page, site.id);
  const bubbleColor = await page.locator('#mp-feedback-bubble').evaluate((element) => getComputedStyle(element).backgroundColor);
  expect(parseRgb(bubbleColor)).toEqual({ red: 255, green: 85, blue: 0 });
  await page.locator('#mp-feedback-bubble').click();
  const sendButtonColor = await page.locator('#mp-feedback-panel button:has-text("Send")').evaluate((element) => getComputedStyle(element).backgroundColor);
  expect(parseRgb(sendButtonColor)).toEqual({ red: 255, green: 85, blue: 0 });
});

test('widget placement honors left side', async ({ page }) => {
  await updateSite(config, buildAdminCookie(), site.id, {
    widget_bubble_side: 'left',
    widget_bubble_bottom_offset: 48
  });
  await openWidgetPage(page, site.id);
  const style = await page.locator('#mp-feedback-bubble').evaluate((element) => ({ left: element.style.left, right: element.style.right }));
  expect(style.left).toBe('16px');
  expect(style.right).toBe('');
});

test('widget placement applies custom bottom offset', async ({ page }) => {
  await updateSite(config, buildAdminCookie(), site.id, {
    widget_bubble_side: 'left',
    widget_bubble_bottom_offset: 48
  });
  await openWidgetPage(page, site.id);
  const bottom = await page.locator('#mp-feedback-bubble').evaluate((element) => element.style.bottom);
  expect(bottom).toBe('48px');
});

test('widget panel offset tracks bubble offset', async ({ page }) => {
  await updateSite(config, buildAdminCookie(), site.id, {
    widget_bubble_side: 'left',
    widget_bubble_bottom_offset: 48
  });
  await openWidgetPage(page, site.id);
  const panelBottom = await page.locator('#mp-feedback-panel').evaluate((element) => element.style.bottom);
  expect(panelBottom).toBe('112px');
});
